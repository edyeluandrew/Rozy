package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/rozy/backend/internal/platform/httpx"
)

var validate = validator.New()

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type otpRequestBody struct {
	Phone string `json:"phone" validate:"required"`
}

type otpVerifyBody struct {
	Phone string `json:"phone" validate:"required"`
	Code  string `json:"code" validate:"required,len=6"`
	Role  string `json:"role" validate:"omitempty,oneof=passenger driver"`
}

type authResponse struct {
	Token     string    `json:"token"`
	ExpiresAt string    `json:"expires_at"`
	User      userDTO   `json:"user"`
}

type userDTO struct {
	ID    string `json:"id"`
	Phone string `json:"phone"`
	Role  string `json:"role"`
}

func (h *Handler) RequestOTP(w http.ResponseWriter, r *http.Request) {
	var body otpRequestBody
	if err := decodeJSON(r, &body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := validate.Struct(body); err != nil {
		httpx.ValidationError(w, validationDetails(err))
		return
	}

	if err := h.svc.RequestOTP(r.Context(), body.Phone); err != nil {
		if errors.Is(err, ErrInvalidPhone) {
			httpx.Error(w, http.StatusBadRequest, "invalid phone; use +256XXXXXXXXX")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "could not send otp")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]string{
		"message": "otp sent",
	})
}

func (h *Handler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var body otpVerifyBody
	if err := decodeJSON(r, &body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := validate.Struct(body); err != nil {
		httpx.ValidationError(w, validationDetails(err))
		return
	}

	user, token, expiresAt, err := h.svc.VerifyOTP(r.Context(), body.Phone, body.Code, body.Role)
	if err != nil {
		if errors.Is(err, ErrInvalidPhone) {
			httpx.Error(w, http.StatusBadRequest, "invalid phone")
			return
		}
		if errors.Is(err, ErrInvalidOTP) {
			httpx.Error(w, http.StatusUnauthorized, "invalid or expired otp")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "could not verify otp")
		return
	}

	httpx.JSON(w, http.StatusOK, authResponse{
		Token:     token,
		ExpiresAt: expiresAt.UTC().Format(time.RFC3339),
		User: userDTO{
			ID:    user.ID.String(),
			Phone: user.Phone,
			Role:  user.Role,
		},
	})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.svc.Me(r.Context(), claims.UserID.String())
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "user not found")
		return
	}

	httpx.JSON(w, http.StatusOK, userDTO{
		ID:    user.ID.String(),
		Phone: user.Phone,
		Role:  user.Role,
	})
}


func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func validationDetails(err error) map[string]string {
	out := map[string]string{}
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		for _, fe := range verrs {
			out[strings.ToLower(fe.Field())] = fe.Tag()
		}
	}
	return out
}
