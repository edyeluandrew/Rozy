package verification

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rozy/backend/internal/auth"
	"github.com/rozy/backend/internal/platform/httpx"
	"github.com/rozy/backend/internal/platform/storage"
)

type Handler struct {
	svc     *Service
	files   *storage.Local
}

func NewHandler(svc *Service, files *storage.Local) *Handler {
	return &Handler{svc: svc, files: files}
}

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	status, err := h.svc.Status(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, ErrNotOperator) {
			httpx.Error(w, http.StatusForbidden, "register as operator first")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "could not load status")
		return
	}
	httpx.JSON(w, http.StatusOK, status)
}

func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body SubmitInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}

	id, err := h.svc.Submit(r.Context(), claims.UserID, body)
	if err != nil {
		writeSubmitError(w, err)
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]any{
		"submission_id": id.String(),
		"status":        "pending",
		"message":       "Verification submitted. Admin will review shortly.",
	})
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	opID, err := h.svc.OperatorID(r.Context(), claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusForbidden, "register as operator first")
		return
	}

	if err := r.ParseMultipartForm(12 << 20); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	docType := r.FormValue("doc_type")
	if err := storage.ValidateDocType(docType); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid doc_type")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "file required")
		return
	}
	defer file.Close()

	key, hash, err := h.files.Save(opID.String(), docType, header.Filename, file)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "upload failed")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]string{
		"storage_key": key,
		"sha256_hash": hash,
		"mime_type":   header.Header.Get("Content-Type"),
	})
}

func writeSubmitError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotOperator):
		httpx.Error(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrPendingExists):
		httpx.Error(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrAlreadyApproved):
		httpx.Error(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrNINTaken):
		httpx.Error(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrInvalidSubmission):
		httpx.Error(w, http.StatusBadRequest, err.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, err.Error())
	}
}
