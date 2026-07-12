package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rozy/backend/internal/admin"
	"github.com/rozy/backend/internal/auth"
	"github.com/rozy/backend/internal/dispatch"
	"github.com/rozy/backend/internal/operator"
	"github.com/rozy/backend/internal/places"
	"github.com/rozy/backend/internal/platform/config"
	"github.com/rozy/backend/internal/platform/db"
	"github.com/rozy/backend/internal/platform/dev"
	"github.com/rozy/backend/internal/platform/httpx"
	redisclient "github.com/rozy/backend/internal/platform/redis"
	"github.com/rozy/backend/internal/platform/storage"
	"github.com/rozy/backend/internal/realtime"
	"github.com/rozy/backend/internal/trip"
	"github.com/rozy/backend/internal/verification"
	"github.com/rozy/backend/internal/wallet"
)

type Server struct {
	cfg  *config.Config
	pool *pgxpool.Pool
	mux  chi.Router
}

func New(cfg *config.Config, pool *pgxpool.Pool, redis *redisclient.Client) *Server {
	s := &Server{cfg: cfg, pool: pool}
	s.routes(redis)
	return s
}

func (s *Server) routes(redis *redisclient.Client) {
	authRepo := auth.NewRepository(s.pool)
	tokens := auth.NewTokenService(s.cfg.JWTSecret, s.cfg.JWTExpiry)
	sms := auth.ConsoleSMS{}
	authSvc := auth.NewService(authRepo, tokens, sms, s.cfg.OTPExpiry)
	authHandler := auth.NewHandler(authSvc)

	hub := realtime.NewHub()
	events := hub
	wsHandler := realtime.NewHandler(hub, tokens)

	dispatchSvc := dispatch.NewService(s.pool, redis, events)

	opRepo := operator.NewRepository(s.pool)
	var geoStore operator.GeoStore
	if redis != nil {
		geoStore = redis
	}
	opSvc := operator.NewService(opRepo, geoStore, events)

	tripRepo := trip.NewRepository(s.pool)
	tripSvc := trip.NewService(tripRepo, dispatchSvc, events)
	tripHandler := trip.NewHandler(tripSvc)

	opHandler := operator.NewHandler(opSvc, dispatchSvc, tripSvc, tripRepo, events)

	walletRepo := wallet.NewRepository(s.pool)
	walletSvc := wallet.NewService(walletRepo, events, wallet.WebhookConfig{
		MTNSecret:    s.cfg.MTNWebhookSecret,
		AirtelSecret: s.cfg.AirtelWebhookSecret,
		Env:          s.cfg.Env,
	})
	walletHandler := wallet.NewHandler(walletSvc)

	fileStore, _ := storage.NewLocal(s.cfg.UploadDir)
	verRepo := verification.NewRepository(s.pool)
	verSvc := verification.NewService(verRepo)
	verHandler := verification.NewHandler(verSvc, fileStore)

	placesRepo := places.NewRepository(s.pool)
	placesHandler := places.NewHandler(placesRepo)

	adminRepo := admin.NewRepository(s.pool)
	adminSvc := admin.NewService(adminRepo)
	adminHandler := admin.NewHandler(adminSvc, fileStore, opSvc)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Webhook-Signature", "X-Rozy-Signature"},
		AllowCredentials: true,
	}))

	if redis == nil {
		log.Println("[server] redis unavailable — dispatch uses postgres fallback only")
	}

	r.Get("/health", s.handleHealth)
	r.Get("/dev", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(dev.TestPageHTML)
	})

	r.Route("/v1", func(v1 chi.Router) {
		v1.Get("/", func(w http.ResponseWriter, r *http.Request) {
			httpx.JSON(w, http.StatusOK, map[string]string{
				"service": "rozy-api",
				"version": "0.1.0-mvp",
			})
		})

		v1.Get("/ws", wsHandler.ServeWS)

		v1.Post("/webhooks/mtn", walletHandler.WebhookMTN)
		v1.Post("/webhooks/airtel", walletHandler.WebhookAirtel)

		v1.Route("/auth", func(authRouter chi.Router) {
			authRouter.Post("/otp/request", authHandler.RequestOTP)
			authRouter.Post("/otp/verify", authHandler.VerifyOTP)
			authRouter.With(auth.Middleware(tokens)).Get("/me", authHandler.Me)
		})

		v1.Route("/operator", func(opRouter chi.Router) {
			opRouter.Use(auth.Middleware(tokens))

			opRouter.Post("/register", opHandler.Register)
			opRouter.Get("/profile", opHandler.Profile)
			opRouter.Get("/wallet", walletHandler.GetWallet)
			opRouter.Post("/wallet/recharge", walletHandler.InitiateRecharge)
			opRouter.Post("/online", opHandler.GoOnline)
			opRouter.Post("/offline", opHandler.GoOffline)
			opRouter.Post("/location", opHandler.UpdateLocation)
			opRouter.Get("/trips/incoming", opHandler.IncomingTrip)
			opRouter.Get("/trips/active", opHandler.ActiveTrip)
			opRouter.Post("/trips/{id}/accept", opHandler.AcceptTrip)
			opRouter.Post("/trips/{id}/reject", opHandler.RejectTrip)
			opRouter.Post("/trips/{id}/arrived", opHandler.ArrivedTrip)
			opRouter.Post("/trips/{id}/start", opHandler.StartTrip)
			opRouter.Post("/trips/{id}/complete", opHandler.CompleteTrip)
			opRouter.Post("/verification/upload", verHandler.Upload)
			opRouter.Post("/verification/submit", verHandler.Submit)
			opRouter.Get("/verification/status", verHandler.Status)
		})

		v1.Get("/places/search", placesHandler.Search)
		v1.Post("/fare/estimate", tripHandler.Estimate)

		v1.Route("/trips", func(tripRouter chi.Router) {
			tripRouter.Use(auth.Middleware(tokens))
			tripRouter.Post("/", tripHandler.Create)
			tripRouter.Get("/active", tripHandler.Active)
			tripRouter.Get("/{id}", tripHandler.Get)
			tripRouter.Post("/{id}/cancel", tripHandler.Cancel)
		})

		v1.Route("/admin", func(ar chi.Router) {
			ar.Use(auth.Middleware(tokens))
			ar.Use(auth.RequireRole("admin"))
			ar.Get("/stats", adminHandler.Stats)
			ar.Get("/trips/active", adminHandler.ActiveTrips)
			ar.Get("/operators", adminHandler.Operators)
			ar.Get("/verification/queue", adminHandler.Queue)
			ar.Get("/verification/{id}", adminHandler.Detail)
			ar.Post("/verification/{id}/approve", adminHandler.Approve)
			ar.Post("/verification/{id}/reject", adminHandler.Reject)
			ar.Post("/operators/{id}/wallet/credit", adminHandler.CreditWallet)
		})

		v1.With(auth.FlexibleMiddleware(tokens)).With(auth.RequireRole("admin")).Get("/admin/files/*", adminHandler.ServeFile)
	})

	s.mux = r
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	code := http.StatusOK
	if err := db.Ping(r.Context(), s.pool); err != nil {
		status = "degraded"
		code = http.StatusServiceUnavailable
	}
	writeJSON(w, code, map[string]string{
		"status": status,
		"db":     dbStatus(r.Context(), s.pool),
	})
}

func dbStatus(ctx context.Context, pool *pgxpool.Pool) string {
	if err := db.Ping(ctx, pool); err != nil {
		return "down"
	}
	return "up"
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
