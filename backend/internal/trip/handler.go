package trip

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/rozy/backend/internal/auth"
	"github.com/rozy/backend/internal/platform/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Estimate(w http.ResponseWriter, r *http.Request) {
	var body EstimateInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	result, err := h.svc.Estimate(r.Context(), body)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "could not estimate fare")
		return
	}
	httpx.JSON(w, http.StatusOK, result)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body CreateInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}

	trip, err := h.svc.Create(r.Context(), claims.UserID, body)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotPassenger):
			httpx.Error(w, http.StatusForbidden, err.Error())
		case errors.Is(err, ErrActiveTripExists):
			httpx.Error(w, http.StatusConflict, err.Error())
		default:
			httpx.Error(w, http.StatusBadRequest, "could not create trip")
		}
		return
	}
	httpx.JSON(w, http.StatusCreated, trip)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid trip id")
		return
	}
	trip, err := h.svc.Get(r.Context(), claims.UserID, tripID)
	if err != nil {
		if errors.Is(err, ErrTripNotFound) {
			httpx.Error(w, http.StatusNotFound, err.Error())
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "could not load trip")
		return
	}
	httpx.JSON(w, http.StatusOK, trip)
}

func (h *Handler) Active(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	trip, err := h.svc.Active(r.Context(), claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load trip")
		return
	}
	if trip == nil {
		httpx.JSON(w, http.StatusOK, map[string]any{"active": false})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"active": true, "trip": trip})
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid trip id")
		return
	}
	if err := h.svc.Cancel(r.Context(), claims.UserID, tripID); err != nil {
		if errors.Is(err, ErrTripNotFound) {
			httpx.Error(w, http.StatusNotFound, err.Error())
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "could not cancel trip")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}
