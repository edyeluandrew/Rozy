package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/rozy/backend/internal/auth"
	"github.com/rozy/backend/internal/operator"
	"github.com/rozy/backend/internal/platform/httpx"
	"github.com/rozy/backend/internal/platform/storage"
)

type Handler struct {
	svc      *Service
	files    *storage.Local
	operator *operator.Service
}

func NewHandler(svc *Service, files *storage.Local, operatorSvc *operator.Service) *Handler {
	return &Handler{svc: svc, files: files, operator: operatorSvc}
}

func (h *Handler) Queue(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	items, err := h.svc.Queue(r.Context(), status)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load queue")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	detail, err := h.svc.Detail(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrSubmissionNotFound) {
			httpx.Error(w, http.StatusNotFound, err.Error())
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "could not load submission")
		return
	}
	httpx.JSON(w, http.StatusOK, detail)
}

func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.svc.Approve(r.Context(), id, claims.UserID); err != nil {
		writeAdminErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Reason == "" {
		body.Reason = "Documents did not pass review"
	}
	if err := h.svc.Reject(r.Context(), id, claims.UserID, body.Reason); err != nil {
		writeAdminErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.Stats(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load stats")
		return
	}
	httpx.JSON(w, http.StatusOK, stats)
}

func (h *Handler) ActiveTrips(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ActiveTrips(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load trips")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"trips": items})
}

func (h *Handler) Operators(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	items, err := h.svc.Operators(r.Context(), status)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load operators")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"operators": items})
}

func (h *Handler) CreditWallet(w http.ResponseWriter, r *http.Request) {
	operatorID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid operator id")
		return
	}
	var body struct {
		Amount    int64  `json:"amount"`
		Reference string `json:"reference"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Amount <= 0 {
		httpx.Error(w, http.StatusBadRequest, "amount must be positive")
		return
	}
	if body.Reference == "" {
		body.Reference = "admin-credit"
	}
	if err := h.operator.CreditWallet(r.Context(), operatorID, body.Amount, body.Reference); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "credited"})
}

func (h *Handler) ServeFile(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(chi.URLParam(r, "*"), "/")
	if key == "" {
		httpx.Error(w, http.StatusBadRequest, "missing key")
		return
	}
	path := h.files.AbsPath(key)
	if _, err := os.Stat(path); err != nil {
		httpx.Error(w, http.StatusNotFound, "file not found")
		return
	}
	http.ServeFile(w, r, filepath.Clean(path))
}

func writeAdminErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrSubmissionNotFound):
		httpx.Error(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrNotPending):
		httpx.Error(w, http.StatusConflict, err.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, err.Error())
	}
}
