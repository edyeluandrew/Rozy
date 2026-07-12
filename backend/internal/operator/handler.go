package operator

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/rozy/backend/internal/auth"
	"github.com/rozy/backend/internal/dispatch"
	"github.com/rozy/backend/internal/platform/httpx"
	"github.com/rozy/backend/internal/realtime"
	"github.com/rozy/backend/internal/trip"
)

var validate = validator.New()

type Handler struct {
	svc      *Service
	dispatch *dispatch.Service
	trips    *trip.Service
	events   realtime.Publisher
	tripRepo *trip.Repository
}

func NewHandler(svc *Service, dispatchSvc *dispatch.Service, tripSvc *trip.Service, tripRepo *trip.Repository, events realtime.Publisher) *Handler {
	if events == nil {
		events = realtime.NopPublisher{}
	}
	return &Handler{svc: svc, dispatch: dispatchSvc, trips: tripSvc, tripRepo: tripRepo, events: events}
}

type registerBody struct {
	RideType string `json:"ride_type" validate:"required,oneof=boda car_basic car_xl"`
}

type locationBody struct {
	Lat float64 `json:"lat" validate:"required"`
	Lng float64 `json:"lng" validate:"required"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body registerBody
	if err := decodeJSON(r, &body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := validate.Struct(body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "ride_type must be boda, car_basic, or car_xl")
		return
	}

	userID, err := uuid.Parse(claims.UserID.String())
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid user")
		return
	}

	profile, err := h.svc.Register(r.Context(), userID, body.RideType)
	if err != nil {
		writeOperatorErr(w, err)
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]any{
		"operator": profile,
		"message":  "Registration started. Complete verification to go online.",
	})
}

func (h *Handler) Profile(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	userID, err := uuid.Parse(claims.UserID.String())
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid user")
		return
	}

	profile, err := h.svc.GetProfile(r.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.JSON(w, http.StatusOK, map[string]any{"registered": false})
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "could not load profile")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"registered": true,
		"operator":   profile,
	})
}

func (h *Handler) GoOnline(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromContext(r.Context())
	userID, _ := uuid.Parse(claims.UserID.String())

	var body locationBody
	_ = json.NewDecoder(r.Body).Decode(&body)

	profile, err := h.svc.GoOnline(r.Context(), userID, body.Lat, body.Lng)
	if err != nil {
		writeOperatorErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"operator": profile, "online": true})
}

func (h *Handler) GoOffline(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromContext(r.Context())
	userID, _ := uuid.Parse(claims.UserID.String())

	profile, err := h.svc.GoOffline(r.Context(), userID)
	if err != nil {
		writeOperatorErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"operator": profile, "online": false})
}

func (h *Handler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromContext(r.Context())
	userID, _ := uuid.Parse(claims.UserID.String())

	var body locationBody
	if err := decodeJSON(r, &body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}

	if err := h.svc.UpdateLocation(r.Context(), userID, body.Lat, body.Lng); err != nil {
		writeOperatorErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) IncomingTrip(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromContext(r.Context())
	userID, _ := uuid.Parse(claims.UserID.String())

	trip, err := h.svc.IncomingTrip(r.Context(), userID)
	if errors.Is(err, ErrNoIncomingTrip) {
		httpx.JSON(w, http.StatusOK, map[string]any{"trip": nil})
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load trip")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"trip": trip})
}

func (h *Handler) AcceptTrip(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromContext(r.Context())
	userID, _ := uuid.Parse(claims.UserID.String())
	tripID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid trip id")
		return
	}

	opID, err := h.svc.OperatorIDForUser(r.Context(), userID)
	if err != nil {
		writeOperatorErr(w, err)
		return
	}

	if err := h.dispatch.AcceptTrip(r.Context(), tripID, opID); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	trip.PublishStatus(r.Context(), h.tripRepo, h.events, tripID, "driver_arriving")
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "driver_arriving"})
}

func (h *Handler) RejectTrip(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromContext(r.Context())
	userID, _ := uuid.Parse(claims.UserID.String())
	tripID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid trip id")
		return
	}

	opID, err := h.svc.OperatorIDForUser(r.Context(), userID)
	if err != nil {
		writeOperatorErr(w, err)
		return
	}

	if err := h.dispatch.RejectAndRematch(r.Context(), tripID, opID); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "searching"})
}

func (h *Handler) ActiveTrip(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromContext(r.Context())
	userID, _ := uuid.Parse(claims.UserID.String())

	t, err := h.trips.ActiveForOperator(r.Context(), userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load trip")
		return
	}
	if t == nil {
		httpx.JSON(w, http.StatusOK, map[string]any{"active": false})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"active": true, "trip": t})
}

func (h *Handler) ArrivedTrip(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromContext(r.Context())
	userID, _ := uuid.Parse(claims.UserID.String())
	tripID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid trip id")
		return
	}
	if err := h.trips.ArrivedForOperator(r.Context(), userID, tripID); err != nil {
		writeTripLifecycleErr(w, err)
		return
	}
	trip.PublishStatus(r.Context(), h.tripRepo, h.events, tripID, "driver_arriving")
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "driver_arriving", "arrived": "true"})
}

type startTripBody struct {
	PIN string `json:"pin"`
}

func (h *Handler) StartTrip(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromContext(r.Context())
	userID, _ := uuid.Parse(claims.UserID.String())
	tripID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid trip id")
		return
	}
	var body startTripBody
	if err := decodeJSON(r, &body); err != nil || body.PIN == "" {
		httpx.Error(w, http.StatusBadRequest, "pin required")
		return
	}
	if err := h.trips.StartForOperator(r.Context(), userID, tripID, body.PIN); err != nil {
		writeTripLifecycleErr(w, err)
		return
	}
	trip.PublishStatus(r.Context(), h.tripRepo, h.events, tripID, "in_progress")
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "in_progress"})
}

func (h *Handler) CompleteTrip(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromContext(r.Context())
	userID, _ := uuid.Parse(claims.UserID.String())
	tripID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid trip id")
		return
	}
	result, err := h.trips.CompleteForOperator(r.Context(), userID, tripID)
	if err != nil {
		writeTripLifecycleErr(w, err)
		return
	}
	trip.PublishStatus(r.Context(), h.tripRepo, h.events, tripID, "completed")
	h.events.PublishToUser(r.Context(), userID, "wallet:updated", map[string]any{
		"balance": result.WalletBalance,
		"trip_id": tripID.String(),
	})
	httpx.JSON(w, http.StatusOK, result)
}

func writeTripLifecycleErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, trip.ErrNotOperatorTrip), errors.Is(err, trip.ErrTripNotFound):
		httpx.Error(w, http.StatusNotFound, err.Error())
	case errors.Is(err, trip.ErrInvalidPIN):
		httpx.Error(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, trip.ErrArrivedRequired):
		httpx.Error(w, http.StatusConflict, err.Error())
	case errors.Is(err, trip.ErrInvalidTransition):
		httpx.Error(w, http.StatusConflict, err.Error())
	default:
		httpx.Error(w, http.StatusBadRequest, err.Error())
	}
}

func writeOperatorErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotDriver):
		httpx.Error(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrAlreadyRegistered):
		httpx.Error(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrInvalidRideType):
		httpx.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrNotVerified):
		httpx.Error(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrWalletBlocked):
		httpx.Error(w, http.StatusPaymentRequired, err.Error())
	case errors.Is(err, ErrCannotGoOnline):
		httpx.Error(w, http.StatusConflict, err.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, "operator request failed")
	}
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
