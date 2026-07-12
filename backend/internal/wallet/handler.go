package wallet

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rozy/backend/internal/auth"
	"github.com/rozy/backend/internal/platform/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetWallet(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	view, txs, err := h.svc.GetWallet(r.Context(), claims.UserID)
	if err != nil {
		writeWalletErr(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"wallet":       view,
		"transactions": txs,
	})
}

func (h *Handler) InitiateRecharge(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body InitiateInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}

	result, err := h.svc.InitiateRecharge(r.Context(), claims.UserID, body)
	if err != nil {
		writeWalletErr(w, err)
		return
	}

	httpx.JSON(w, http.StatusCreated, result)
}

func (h *Handler) WebhookMTN(w http.ResponseWriter, r *http.Request) {
	h.handleWebhook(w, r, "mtn")
}

func (h *Handler) WebhookAirtel(w http.ResponseWriter, r *http.Request) {
	h.handleWebhook(w, r, "airtel")
}

func (h *Handler) handleWebhook(w http.ResponseWriter, r *http.Request, provider string) {
	body, err := ReadBody(r.Body, 1<<20)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid body")
		return
	}

	sig := r.Header.Get("X-Webhook-Signature")
	if sig == "" {
		sig = r.Header.Get("X-Rozy-Signature")
	}

	result, err := h.svc.HandleWebhook(r.Context(), provider, body, sig)
	if err != nil {
		if errors.Is(err, ErrRechargeNotFound) {
			httpx.Error(w, http.StatusNotFound, err.Error())
			return
		}
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	resp := map[string]any{"status": "ok"}
	if result != nil {
		resp["balance"] = result.BalanceAfter
		resp["recharge_id"] = result.RechargeID
		if result.AlreadyDone {
			resp["idempotent"] = true
		}
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func writeWalletErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotOperator):
		httpx.Error(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrInvalidAmount), errors.Is(err, ErrInvalidProvider):
		httpx.Error(w, http.StatusBadRequest, err.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, "wallet request failed")
	}
}
