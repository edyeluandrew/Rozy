package wallet

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/rozy/backend/internal/realtime"
)

type Service struct {
	repo   *Repository
	events realtime.Publisher
	cfg    WebhookConfig
}

type WebhookConfig struct {
	MTNSecret    string
	AirtelSecret string
	Env          string
}

func NewService(repo *Repository, events realtime.Publisher, cfg WebhookConfig) *Service {
	if events == nil {
		events = realtime.NopPublisher{}
	}
	return &Service{repo: repo, events: events, cfg: cfg}
}

func (s *Service) GetWallet(ctx context.Context, userID uuid.UUID) (*WalletView, []Transaction, error) {
	opID, err := s.repo.OperatorIDByUser(ctx, userID)
	if err != nil {
		return nil, nil, ErrNotOperator
	}
	w, err := s.repo.GetWallet(ctx, opID)
	if err != nil {
		return nil, nil, err
	}
	txs, err := s.repo.ListTransactions(ctx, opID, 20)
	if err != nil {
		return nil, nil, err
	}
	if txs == nil {
		txs = []Transaction{}
	}
	return w, txs, nil
}

type InitiateInput struct {
	Amount   int64  `json:"amount"`
	Provider string `json:"provider"`
}

type InitiateResult struct {
	Recharge     *Recharge `json:"recharge"`
	Instructions string    `json:"instructions"`
	WebhookHint  string    `json:"webhook_hint,omitempty"`
}

func (s *Service) InitiateRecharge(ctx context.Context, userID uuid.UUID, in InitiateInput) (*InitiateResult, error) {
	if in.Amount < 1000 {
		return nil, ErrInvalidAmount
	}
	provider := strings.ToLower(strings.TrimSpace(in.Provider))
	if provider != "mtn" && provider != "airtel" {
		return nil, ErrInvalidProvider
	}

	opID, err := s.repo.OperatorIDByUser(ctx, userID)
	if err != nil {
		return nil, ErrNotOperator
	}

	rc, err := s.repo.CreateRecharge(ctx, opID, in.Amount, provider)
	if err != nil {
		return nil, err
	}

	instructions := "Approve the MTN MoMo prompt on your phone. Your Rozy wallet updates automatically when payment succeeds."
	if provider == "airtel" {
		instructions = "Approve the Airtel Money prompt on your phone. Your Rozy wallet updates automatically when payment succeeds."
	}

	out := &InitiateResult{
		Recharge:     rc,
		Instructions: instructions,
	}
	if s.cfg.Env == "development" {
		out.WebhookHint = fmt.Sprintf(
			`Dev: POST /v1/webhooks/%s {"reference":"%s","transaction_id":"sim-%s","amount":%d,"status":"successful"}`,
			provider, rc.IdempotencyKey, rc.ID, in.Amount,
		)
	}
	return out, nil
}

type WebhookInput struct {
	TransactionID          string `json:"transaction_id"`
	Reference              string `json:"reference"`
	Amount                 int64  `json:"amount"`
	Status                 string `json:"status"`
	Phone                  string `json:"phone"`
	FinancialTransactionID string `json:"financialTransactionId"`
	ExternalID             string `json:"externalId"`
}

func (w *WebhookInput) normalize() {
	if w.TransactionID == "" {
		w.TransactionID = w.FinancialTransactionID
	}
	if w.Reference == "" {
		w.Reference = w.ExternalID
	}
}

func (w WebhookInput) isSuccess() bool {
	s := strings.ToLower(strings.TrimSpace(w.Status))
	return s == "successful" || s == "success" || s == "completed"
}

func (s *Service) HandleWebhook(ctx context.Context, provider string, body []byte, signature string) (*CompleteResult, error) {
	secret := s.cfg.MTNSecret
	if provider == "airtel" {
		secret = s.cfg.AirtelSecret
	}
	if secret != "" && !verifySignature(body, signature, secret) {
		return nil, errors.New("invalid webhook signature")
	}

	var in WebhookInput
	if err := json.Unmarshal(body, &in); err != nil {
		return nil, err
	}
	in.normalize()

	if in.Reference == "" && in.TransactionID == "" {
		return nil, errors.New("reference or transaction_id required")
	}

	success := in.isSuccess()
	result, err := s.repo.CompleteRecharge(ctx, in.Reference, in.TransactionID, in.Amount, success)
	if err != nil {
		if !success {
			log.Printf("[wallet] %s webhook failed payment ref=%s", provider, in.Reference)
		}
		return nil, err
	}

	if result != nil && !result.AlreadyDone {
		s.events.PublishToUser(ctx, result.UserID, "wallet:updated", map[string]any{
			"balance":     result.BalanceAfter,
			"amount":      result.Amount,
			"recharge_id": result.RechargeID,
			"provider":    provider,
		})
	}

	return result, nil
}

func verifySignature(body []byte, signature, secret string) bool {
	if signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(strings.ToLower(signature)), []byte(strings.ToLower(expected)))
}

func ReadBody(r io.Reader, limit int64) ([]byte, error) {
	if limit <= 0 {
		limit = 1 << 20
	}
	return io.ReadAll(io.LimitReader(r, limit))
}

func ParseAmount(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case string:
		i, _ := strconv.ParseInt(n, 10, 64)
		return i
	case int64:
		return n
	default:
		return 0
	}
}
