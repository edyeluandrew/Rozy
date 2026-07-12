package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/rozy/backend/internal/auth"
	"github.com/rozy/backend/internal/platform/db"
)

const testOTP = "123456"

var (
	driverPhone    = "+256700000081"
	passengerPhone = "+256700000082"
	adminPhone     = "+256700000000"
)

type runner struct {
	base   string
	client *http.Client
	steps  int
	failed int
}

type apiResponse struct {
	status int
	body   map[string]any
	raw    string
}

func main() {
	_ = godotenv.Load()

	apiBase := flag.String("api", envOr("API_BASE_URL", "http://localhost:8080"), "API base URL without /v1")
	flag.Parse()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		fatal("DATABASE_URL is required (set in backend/.env)")
	}

	ctx := context.Background()
	pool, err := db.NewPool(ctx, databaseURL)
	if err != nil {
		fatal("db connect: %v", err)
	}
	defer pool.Close()

	r := &runner{
		base:   strings.TrimRight(*apiBase, "/"),
		client: &http.Client{Timeout: 30 * time.Second},
	}

	fmt.Println("Rozy API integration tests")
	fmt.Printf("API: %s\n", r.base)
	fmt.Println(strings.Repeat("-", 50))

	authRepo := auth.NewRepository(pool)

	// --- Public ---
	r.check("GET /health", func() error {
		res, err := r.get("/health", "")
		if err != nil {
			return err
		}
		if res.status != 200 {
			return fmt.Errorf("status %d", res.status)
		}
		if res.body["status"] != "ok" {
			return fmt.Errorf("unexpected body: %s", res.raw)
		}
		return nil
	})

	r.check("GET /v1/", func() error {
		res, err := r.get("/v1/", "")
		if err != nil {
			return err
		}
		if res.status != 200 || res.body["service"] != "rozy-api" {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		return nil
	})

	r.check("GET /v1/places/search", func() error {
		res, err := r.get("/v1/places/search?q=hospital", "")
		if err != nil {
			return err
		}
		if res.status != 200 {
			return fmt.Errorf("status %d", res.status)
		}
		places, _ := res.body["places"].([]any)
		if len(places) == 0 {
			return fmt.Errorf("expected places")
		}
		return nil
	})

	r.check("POST /v1/fare/estimate", func() error {
		res, err := r.post("/v1/fare/estimate", "", map[string]any{
			"pickup":    map[string]float64{"lat": -0.6072, "lng": 30.6586},
			"dest":      map[string]float64{"lat": -0.6010, "lng": 30.6490},
			"ride_type": "boda",
		})
		if err != nil {
			return err
		}
		if res.status != 200 || res.body["estimated_fare"] == nil {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		return nil
	})

	r.check("preflight: dispatch routes available", func() error {
		res, err := r.post("/v1/operator/online", "invalid-token", map[string]float64{"lat": 0, "lng": 0})
		if err != nil {
			return err
		}
		if res.status == 404 {
			return fmt.Errorf("POST /v1/operator/online returned 404 — restart API with latest code (make run)")
		}
		if res.status != 401 {
			return fmt.Errorf("expected 401 without token, got %d", res.status)
		}
		return nil
	})

	// --- Auth tokens ---
	driverToken := r.login(ctx, authRepo, "driver", driverPhone)
	passengerToken := r.login(ctx, authRepo, "passenger", passengerPhone)
	adminToken := r.login(ctx, authRepo, "", adminPhone)

	r.check("POST /v1/auth/otp/request", func() error {
		if err := authRepo.SaveOTP(ctx, driverPhone, testOTP, time.Now().Add(10*time.Minute)); err != nil {
			return err
		}
		res, err := r.post("/v1/auth/otp/request", "", map[string]string{"phone": driverPhone})
		if err != nil {
			return err
		}
		if res.status != 200 {
			return fmt.Errorf("status %d", res.status)
		}
		return nil
	})

	r.check("GET /v1/auth/me (driver)", func() error {
		res, err := r.get("/v1/auth/me", driverToken)
		if err != nil {
			return err
		}
		if res.status != 200 || res.body["role"] != "driver" {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		return nil
	})

	r.check("GET /v1/auth/me (passenger)", func() error {
		res, err := r.get("/v1/auth/me", passengerToken)
		if err != nil {
			return err
		}
		if res.status != 200 || res.body["role"] != "passenger" {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		return nil
	})

	// --- Driver setup ---
	var operatorID string

	r.check("POST /v1/operator/register", func() error {
		res, err := r.post("/v1/operator/register", driverToken, map[string]string{"ride_type": "boda"})
		if err != nil {
			return err
		}
		if res.status != 201 && res.status != 409 {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		return nil
	})

	r.check("GET /v1/operator/profile", func() error {
		res, err := r.get("/v1/operator/profile", driverToken)
		if err != nil {
			return err
		}
		if res.status != 200 || res.body["registered"] != true {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		op, _ := res.body["operator"].(map[string]any)
		operatorID, _ = op["id"].(string)
		if operatorID == "" {
			return fmt.Errorf("missing operator id")
		}
		return nil
	})

	r.check("GET /v1/operator/verification/status", func() error {
		res, err := r.get("/v1/operator/verification/status", driverToken)
		if err != nil {
			return err
		}
		if res.status != 200 {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		return nil
	})

	submissionID := r.ensureVerification(driverToken)

	if submissionID != "" {
		r.check("GET /v1/admin/stats", func() error {
			res, err := r.get("/v1/admin/stats", adminToken)
			if err != nil {
				return err
			}
			if res.status != 200 {
				return fmt.Errorf("status %d", res.status)
			}
			return nil
		})

		r.check("GET /v1/admin/verification/queue", func() error {
			res, err := r.get("/v1/admin/verification/queue", adminToken)
			if err != nil {
				return err
			}
			if res.status != 200 {
				return fmt.Errorf("status %d", res.status)
			}
			return nil
		})

		r.check("GET /v1/admin/verification/{id}", func() error {
			res, err := r.get("/v1/admin/verification/"+submissionID, adminToken)
			if err != nil {
				return err
			}
			if res.status != 200 {
				return fmt.Errorf("status %d body %s", res.status, res.raw)
			}
			return nil
		})

		r.check("POST /v1/admin/verification/{id}/approve", func() error {
			res, err := r.post("/v1/admin/verification/"+submissionID+"/approve", adminToken, nil)
			if err != nil {
				return err
			}
			if res.status != 200 {
				return fmt.Errorf("status %d body %s", res.status, res.raw)
			}
			return nil
		})
	}

	r.check("POST /v1/admin/operators/{id}/wallet/credit", func() error {
		res, err := r.post("/v1/admin/operators/"+operatorID+"/wallet/credit", adminToken, map[string]any{
			"amount":    20000,
			"reference": fmt.Sprintf("test-api-credit-%d", time.Now().UnixNano()),
		})
		if err != nil {
			return err
		}
		if res.status != 200 {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		return nil
	})

	var rechargeKey string
	r.check("GET /v1/operator/wallet", func() error {
		res, err := r.get("/v1/operator/wallet", driverToken)
		if err != nil {
			return err
		}
		if res.status != 200 {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		wallet, _ := res.body["wallet"].(map[string]any)
		if wallet == nil {
			return fmt.Errorf("missing wallet: %s", res.raw)
		}
		return nil
	})

	r.check("POST /v1/operator/wallet/recharge", func() error {
		res, err := r.post("/v1/operator/wallet/recharge", driverToken, map[string]any{
			"amount":   5000,
			"provider": "mtn",
		})
		if err != nil {
			return err
		}
		if res.status != 201 {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		recharge, _ := res.body["recharge"].(map[string]any)
		if recharge == nil {
			return fmt.Errorf("missing recharge: %s", res.raw)
		}
		rechargeKey = asString(recharge["idempotency_key"])
		if rechargeKey == "" {
			return fmt.Errorf("missing idempotency_key: %s", res.raw)
		}
		return nil
	})

	r.check("POST /v1/webhooks/mtn (simulate success)", func() error {
		if rechargeKey == "" {
			return fmt.Errorf("recharge key not set")
		}
		res, err := r.post("/v1/webhooks/mtn", "", map[string]any{
			"reference":      rechargeKey,
			"transaction_id": fmt.Sprintf("sim-txn-%d", time.Now().UnixNano()),
			"amount":         5000,
			"status":         "successful",
		})
		if err != nil {
			return err
		}
		if res.status != 200 {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		return nil
	})

	r.check("POST /v1/webhooks/mtn (idempotent)", func() error {
		res, err := r.post("/v1/webhooks/mtn", "", map[string]any{
			"reference": rechargeKey,
			"status":    "successful",
		})
		if err != nil {
			return err
		}
		if res.status != 200 {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		if res.body["idempotent"] != true {
			return fmt.Errorf("expected idempotent replay: %s", res.raw)
		}
		return nil
	})

	var airtelRechargeKey string
	r.check("POST /v1/operator/wallet/recharge (airtel)", func() error {
		res, err := r.post("/v1/operator/wallet/recharge", driverToken, map[string]any{
			"amount":   3000,
			"provider": "airtel",
		})
		if err != nil {
			return err
		}
		if res.status != 201 {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		recharge, _ := res.body["recharge"].(map[string]any)
		airtelRechargeKey = asString(recharge["idempotency_key"])
		if airtelRechargeKey == "" {
			return fmt.Errorf("missing idempotency_key: %s", res.raw)
		}
		return nil
	})

	r.check("POST /v1/webhooks/airtel (simulate success)", func() error {
		res, err := r.post("/v1/webhooks/airtel", "", map[string]any{
			"reference":      airtelRechargeKey,
			"transaction_id": fmt.Sprintf("airtel-sim-%d", time.Now().UnixNano()),
			"amount":         3000,
			"status":         "successful",
		})
		if err != nil {
			return err
		}
		if res.status != 200 {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		return nil
	})

	r.check("GET /v1/admin/operators", func() error {
		res, err := r.get("/v1/admin/operators", adminToken)
		if err != nil {
			return err
		}
		if res.status != 200 {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		ops, _ := res.body["operators"].([]any)
		if ops == nil {
			return fmt.Errorf("missing operators: %s", res.raw)
		}
		return nil
	})

	// Cancel any leftover active passenger trip and reset driver before going online.
	r.check("GET /v1/trips/active (cleanup)", func() error {
		res, err := r.get("/v1/trips/active", passengerToken)
		if err != nil {
			return err
		}
		if res.status != 200 {
			return fmt.Errorf("status %d", res.status)
		}
		if res.body["active"] == true {
			trip, _ := res.body["trip"].(map[string]any)
			tripID, _ := trip["id"].(string)
			if tripID != "" {
				cancel, err := r.post("/v1/trips/"+tripID+"/cancel", passengerToken, nil)
				if err != nil {
					return err
				}
				if cancel.status != 200 {
					return fmt.Errorf("cancel status %d", cancel.status)
				}
			}
		}
		return nil
	})

	r.check("POST /v1/operator/offline (cleanup)", func() error {
		res, err := r.post("/v1/operator/offline", driverToken, nil)
		if err != nil {
			return err
		}
		if res.status != 200 {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		return nil
	})

	r.check("POST /v1/operator/online", func() error {
		res, err := r.post("/v1/operator/online", driverToken, map[string]float64{
			"lat": -0.6072,
			"lng": 30.6586,
		})
		if err != nil {
			return err
		}
		if res.status != 200 {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		op, _ := res.body["operator"].(map[string]any)
		if op["status"] != "available" {
			return fmt.Errorf("expected available, got %v", op["status"])
		}
		return nil
	})

	r.check("POST /v1/operator/location", func() error {
		res, err := r.post("/v1/operator/location", driverToken, map[string]float64{
			"lat": -0.6072,
			"lng": 30.6586,
		})
		if err != nil {
			return err
		}
		if res.status != 200 {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		return nil
	})

	var tripID string
	var tripPIN string

	r.check("POST /v1/trips", func() error {
		res, err := r.post("/v1/trips", passengerToken, map[string]any{
			"pickup":          map[string]float64{"lat": -0.6072, "lng": 30.6586},
			"dest":            map[string]float64{"lat": -0.6010, "lng": 30.6490},
			"ride_type":       "boda",
			"pickup_landmark": "Clock Tower",
			"dest_landmark":   "Hospital",
		})
		if err != nil {
			return err
		}
		if res.status != 201 {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		tripID, _ = res.body["id"].(string)
		tripPIN, _ = res.body["trip_pin"].(string)
		if tripID == "" {
			return fmt.Errorf("missing trip id")
		}
		status, _ := res.body["status"].(string)
		if status != "driver_assigned" && status != "searching" {
			return fmt.Errorf("unexpected status %s (dispatch may need online driver nearby)", status)
		}
		return nil
	})

	r.check("GET /v1/admin/trips/active", func() error {
		res, err := r.get("/v1/admin/trips/active", adminToken)
		if err != nil {
			return err
		}
		if res.status != 200 {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		trips, _ := res.body["trips"].([]any)
		if trips == nil {
			return fmt.Errorf("missing trips: %s", res.raw)
		}
		if len(trips) == 0 {
			return fmt.Errorf("expected at least one active trip")
		}
		return nil
	})

	r.check("GET /v1/trips/active", func() error {
		res, err := r.get("/v1/trips/active", passengerToken)
		if err != nil {
			return err
		}
		if res.status != 200 || res.body["active"] != true {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		return nil
	})

	r.check("GET /v1/trips/{id}", func() error {
		res, err := r.get("/v1/trips/"+tripID, passengerToken)
		if err != nil {
			return err
		}
		if res.status != 200 {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		return nil
	})

	r.check("GET /v1/operator/trips/incoming", func() error {
		res, err := r.get("/v1/operator/trips/incoming", driverToken)
		if err != nil {
			return err
		}
		if res.status != 200 {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		trip := res.body["trip"]
		if trip == nil {
			return fmt.Errorf("expected incoming trip")
		}
		return nil
	})

	r.check("POST /v1/operator/trips/{id}/reject (rematch)", func() error {
		res, err := r.post("/v1/operator/trips/"+tripID+"/reject", driverToken, nil)
		if err != nil {
			return err
		}
		if res.status != 200 {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		return nil
	})

	// Wait briefly for rematch dispatch.
	time.Sleep(500 * time.Millisecond)

	r.check("GET /v1/operator/trips/incoming (after rematch)", func() error {
		res, err := r.get("/v1/operator/trips/incoming", driverToken)
		if err != nil {
			return err
		}
		if res.status != 200 || res.body["trip"] == nil {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		return nil
	})

	r.check("POST /v1/operator/trips/{id}/accept", func() error {
		res, err := r.post("/v1/operator/trips/"+tripID+"/accept", driverToken, nil)
		if err != nil {
			return err
		}
		if res.status != 200 {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		return nil
	})

	r.check("POST /v1/operator/location (live tracking)", func() error {
		res, err := r.post("/v1/operator/location", driverToken, map[string]float64{
			"lat": -0.6065,
			"lng": 30.6578,
		})
		if err != nil {
			return err
		}
		if res.status != 200 {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		return nil
	})

	r.check("GET /v1/trips/{id} (driver on map)", func() error {
		res, err := r.get("/v1/trips/"+tripID, passengerToken)
		if err != nil {
			return err
		}
		if res.status != 200 {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		driver, _ := res.body["driver"].(map[string]any)
		if driver == nil {
			return fmt.Errorf("expected driver snapshot")
		}
		if res.body["driver_distance_km"] == nil {
			return fmt.Errorf("expected driver_distance_km")
		}
		if res.body["driver_eta_minutes"] == nil {
			return fmt.Errorf("expected driver_eta_minutes")
		}
		return nil
	})

	r.check("GET /v1/operator/trips/active", func() error {
		res, err := r.get("/v1/operator/trips/active", driverToken)
		if err != nil {
			return err
		}
		if res.status != 200 || res.body["active"] != true {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		return nil
	})

	r.check("POST /v1/operator/trips/{id}/arrived", func() error {
		res, err := r.post("/v1/operator/trips/"+tripID+"/arrived", driverToken, nil)
		if err != nil {
			return err
		}
		if res.status != 200 {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		return nil
	})

	r.check("POST /v1/operator/trips/{id}/start", func() error {
		if tripPIN == "" {
			return fmt.Errorf("missing trip_pin from create response")
		}
		res, err := r.post("/v1/operator/trips/"+tripID+"/start", driverToken, map[string]string{"pin": tripPIN})
		if err != nil {
			return err
		}
		if res.status != 200 {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		if res.body["status"] != "in_progress" {
			return fmt.Errorf("expected in_progress")
		}
		return nil
	})

	r.check("GET /v1/trips/{id} (in_progress)", func() error {
		res, err := r.get("/v1/trips/"+tripID, passengerToken)
		if err != nil {
			return err
		}
		if res.status != 200 || res.body["status"] != "in_progress" {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		return nil
	})

	r.check("POST /v1/operator/trips/{id}/complete", func() error {
		res, err := r.post("/v1/operator/trips/"+tripID+"/complete", driverToken, nil)
		if err != nil {
			return err
		}
		if res.status != 200 {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		if res.body["status"] != "completed" {
			return fmt.Errorf("expected completed")
		}
		if res.body["rozy_fee"] == nil || res.body["final_fare"] == nil {
			return fmt.Errorf("missing fare fields")
		}
		return nil
	})

	r.check("GET /v1/trips/{id} (completed)", func() error {
		res, err := r.get("/v1/trips/"+tripID, passengerToken)
		if err != nil {
			return err
		}
		if res.status != 200 || res.body["status"] != "completed" {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		return nil
	})

	r.check("GET /v1/trips/active (after complete)", func() error {
		res, err := r.get("/v1/trips/active", passengerToken)
		if err != nil {
			return err
		}
		if res.status != 200 || res.body["active"] == true {
			return fmt.Errorf("expected no active trip")
		}
		return nil
	})

	r.check("POST /v1/operator/offline", func() error {
		res, err := r.post("/v1/operator/offline", driverToken, nil)
		if err != nil {
			return err
		}
		if res.status != 200 {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		return nil
	})

	fmt.Println(strings.Repeat("-", 50))
	if r.failed > 0 {
		fmt.Printf("FAILED: %d of %d checks failed\n", r.failed, r.steps)
		os.Exit(1)
	}
	fmt.Printf("PASSED: all %d checks OK\n", r.steps)
}

func (r *runner) login(ctx context.Context, repo *auth.Repository, role, phone string) string {
	var token string
	label := role
	if label == "" {
		label = "existing"
	}
	r.check("POST /v1/auth/otp/verify ("+label+")", func() error {
		if err := repo.SaveOTP(ctx, phone, testOTP, time.Now().Add(10*time.Minute)); err != nil {
			return fmt.Errorf("seed otp: %w", err)
		}
		body := map[string]string{"phone": phone, "code": testOTP}
		if role != "" {
			body["role"] = role
		}
		verify, err := r.post("/v1/auth/otp/verify", "", body)
		if err != nil {
			return err
		}
		if verify.status != 200 {
			return fmt.Errorf("verify status %d body %s", verify.status, verify.raw)
		}
		token, _ = verify.body["token"].(string)
		if token == "" {
			return fmt.Errorf("missing token")
		}
		return nil
	})
	return token
}

func (r *runner) ensureVerification(driverToken string) string {
	var submissionID string

	statusRes, err := r.get("/v1/operator/verification/status", driverToken)
	if err == nil && statusRes.status == 200 {
		if statusRes.body["status"] == "approved" {
			fmt.Println("  · driver already verified — skipping submit/approve")
			return ""
		}
		if statusRes.body["status"] == "pending" {
			submissionID, _ = statusRes.body["submission_id"].(string)
			if submissionID != "" {
				fmt.Println("  · verification already pending — skipping submit")
				return submissionID
			}
		}
	}

	docs := make([]map[string]string, 0, 3)
	for _, docType := range []string{"nin_front", "selfie", "permit"} {
		docType := docType
		r.check("POST /v1/operator/verification/upload ("+docType+")", func() error {
			res, err := r.upload(driverToken, docType)
			if err != nil {
				return err
			}
			if res.status != 200 {
				return fmt.Errorf("status %d body %s", res.status, res.raw)
			}
			docs = append(docs, map[string]string{
				"doc_type":    docType,
				"storage_key": asString(res.body["storage_key"]),
				"sha256_hash": asString(res.body["sha256_hash"]),
				"mime_type":   asString(res.body["mime_type"]),
			})
			return nil
		})
	}

	suffix := fmt.Sprintf("%d", time.Now().Unix()%100000)
	r.check("POST /v1/operator/verification/submit", func() error {
		res, err := r.post("/v1/operator/verification/submit", driverToken, map[string]any{
			"legal_name":       "Test Driver " + suffix,
			"nin":              fmt.Sprintf("CM%s1234567", suffix),
			"permit_number":    "P-" + suffix,
			"permit_expiry":    "2027-01-01",
			"insurance_expiry": "2026-12-01",
			"plate":            fmt.Sprintf("UB%sA", suffix),
			"bike_make":        "Bajaj",
			"bike_color":       "Red",
			"documents":        docs,
		})
		if err != nil {
			return err
		}
		if res.status != 201 && res.status != 409 {
			return fmt.Errorf("status %d body %s", res.status, res.raw)
		}
		submissionID = asString(res.body["submission_id"])
		return nil
	})

	if submissionID == "" {
		// Re-fetch from status if submit returned conflict.
		statusRes, _ := r.get("/v1/operator/verification/status", driverToken)
		submissionID = asString(statusRes.body["submission_id"])
	}
	return submissionID
}

func (r *runner) check(name string, fn func() error) {
	r.steps++
	fmt.Printf("  %s ... ", name)
	if err := fn(); err != nil {
		r.failed++
		fmt.Printf("FAIL\n    %v\n", err)
		return
	}
	fmt.Println("OK")
}

func (r *runner) get(path, token string) (apiResponse, error) {
	req, err := http.NewRequest(http.MethodGet, r.base+path, nil)
	if err != nil {
		return apiResponse{}, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return r.do(req)
}

func (r *runner) post(path, token string, body any) (apiResponse, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return apiResponse{}, err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(http.MethodPost, r.base+path, reader)
	if err != nil {
		return apiResponse{}, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return r.do(req)
}

func (r *runner) upload(token, docType string) (apiResponse, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("doc_type", docType)
	fw, err := w.CreateFormFile("file", "test.jpg")
	if err != nil {
		return apiResponse{}, err
	}
	_, _ = fw.Write([]byte("fake-image-bytes-for-test"))
	_ = w.Close()

	req, err := http.NewRequest(http.MethodPost, r.base+"/v1/operator/verification/upload", &buf)
	if err != nil {
		return apiResponse{}, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	return r.do(req)
}

func (r *runner) do(req *http.Request) (apiResponse, error) {
	resp, err := r.client.Do(req)
	if err != nil {
		return apiResponse{}, err
	}
	defer resp.Body.Close()

	rawBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return apiResponse{}, err
	}
	raw := string(rawBytes)

	var body map[string]any
	_ = json.Unmarshal(rawBytes, &body)
	if body == nil {
		body = map[string]any{}
	}

	return apiResponse{status: resp.StatusCode, body: body, raw: raw}, nil
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
