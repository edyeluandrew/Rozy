package wallet

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotOperator      = errors.New("operator profile required")
	ErrInvalidAmount    = errors.New("amount must be at least 1000 UGX")
	ErrInvalidProvider  = errors.New("provider must be mtn or airtel")
	ErrRechargeNotFound = errors.New("recharge not found")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

type WalletView struct {
	Balance    int64 `json:"balance"`
	MinBalance int64 `json:"min_balance"`
}

type Transaction struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Amount       int64  `json:"amount"`
	BalanceAfter int64  `json:"balance_after"`
	Reference    string `json:"reference"`
	CreatedAt    string `json:"created_at"`
}

type Recharge struct {
	ID             string `json:"id"`
	Amount         int64  `json:"amount"`
	Provider       string `json:"provider"`
	Status         string `json:"status"`
	IdempotencyKey string `json:"idempotency_key"`
	ExternalTxID   string `json:"external_tx_id,omitempty"`
	CreatedAt      string `json:"created_at"`
	CompletedAt    string `json:"completed_at,omitempty"`
}

func (r *Repository) OperatorIDByUser(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `SELECT id FROM operator_profiles WHERE user_id = $1`, userID).Scan(&id)
	return id, err
}

func (r *Repository) UserIDByOperator(ctx context.Context, operatorID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `SELECT user_id FROM operator_profiles WHERE id = $1`, operatorID).Scan(&id)
	return id, err
}

func (r *Repository) GetWallet(ctx context.Context, operatorID uuid.UUID) (*WalletView, error) {
	var w WalletView
	err := r.pool.QueryRow(ctx, `
		SELECT wallet_balance, wallet_min_balance FROM operator_profiles WHERE id = $1
	`, operatorID).Scan(&w.Balance, &w.MinBalance)
	return &w, err
}

func (r *Repository) ListTransactions(ctx context.Context, operatorID uuid.UUID, limit int) ([]Transaction, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, tx_type::text, amount, balance_after, reference, created_at
		FROM wallet_transactions
		WHERE operator_profile_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, operatorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Transaction
	for rows.Next() {
		var t Transaction
		var created time.Time
		if err := rows.Scan(&t.ID, &t.Type, &t.Amount, &t.BalanceAfter, &t.Reference, &created); err != nil {
			return nil, err
		}
		t.CreatedAt = created.UTC().Format(time.RFC3339)
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Repository) CreateRecharge(ctx context.Context, operatorID uuid.UUID, amount int64, provider string) (*Recharge, error) {
	idempotencyKey := fmt.Sprintf("rzy-%s", uuid.NewString())
	var rc Recharge
	var created time.Time
	err := r.pool.QueryRow(ctx, `
		INSERT INTO wallet_recharges (operator_profile_id, amount, provider, idempotency_key, expires_at)
		VALUES ($1, $2, $3::payment_provider, $4, now() + interval '30 minutes')
		RETURNING id, amount, provider::text, status::text, idempotency_key, created_at
	`, operatorID, amount, provider, idempotencyKey).Scan(
		&rc.ID, &rc.Amount, &rc.Provider, &rc.Status, &rc.IdempotencyKey, &created,
	)
	if err != nil {
		return nil, err
	}
	rc.CreatedAt = created.UTC().Format(time.RFC3339)
	return &rc, nil
}

type rechargeRow struct {
	ID             uuid.UUID
	OperatorID     uuid.UUID
	Amount         int64
	Status         string
	IdempotencyKey string
	ExternalTxID   *string
}

func (r *Repository) findRecharge(ctx context.Context, reference, externalTxID string) (*rechargeRow, error) {
	var row rechargeRow
	err := r.pool.QueryRow(ctx, `
		SELECT id, operator_profile_id, amount, status::text, idempotency_key, external_tx_id
		FROM wallet_recharges
		WHERE ($1 <> '' AND idempotency_key = $1)
		   OR ($2 <> '' AND external_tx_id = $2)
		   OR ($1 <> '' AND id::text = $1)
		ORDER BY created_at DESC
		LIMIT 1
	`, reference, externalTxID).Scan(
		&row.ID, &row.OperatorID, &row.Amount, &row.Status, &row.IdempotencyKey, &row.ExternalTxID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRechargeNotFound
	}
	return &row, err
}

type CompleteResult struct {
	RechargeID   string
	OperatorID   uuid.UUID
	UserID       uuid.UUID
	Amount       int64
	BalanceAfter int64
	AlreadyDone  bool
}

func (r *Repository) CompleteRecharge(ctx context.Context, reference, externalTxID string, amount int64, success bool) (*CompleteResult, error) {
	row, err := r.findRecharge(ctx, reference, externalTxID)
	if err != nil {
		return nil, err
	}

	if row.Status == "completed" {
		userID, _ := r.UserIDByOperator(ctx, row.OperatorID)
		bal, _ := r.GetWallet(ctx, row.OperatorID)
		balance := int64(0)
		if bal != nil {
			balance = bal.Balance
		}
		return &CompleteResult{
			RechargeID:   row.ID.String(),
			OperatorID:   row.OperatorID,
			UserID:       userID,
			Amount:       row.Amount,
			BalanceAfter: balance,
			AlreadyDone:  true,
		}, nil
	}

	if row.Status == "failed" || row.Status == "expired" {
		return nil, fmt.Errorf("recharge already %s", row.Status)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if !success {
		_, err = tx.Exec(ctx, `
			UPDATE wallet_recharges
			SET status = 'failed', external_tx_id = COALESCE(NULLIF($2, ''), external_tx_id), completed_at = now()
			WHERE id = $1 AND status = 'pending'
		`, row.ID, externalTxID)
		if err != nil {
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("payment failed")
	}

	creditAmount := row.Amount
	if amount > 0 {
		creditAmount = amount
	}

	var balance int64
	err = tx.QueryRow(ctx, `
		UPDATE operator_profiles
		SET wallet_balance = wallet_balance + $2,
		    status = CASE
		      WHEN status = 'wallet_blocked' AND wallet_balance + $2 >= wallet_min_balance THEN 'offline'
		      ELSE status
		    END,
		    updated_at = now()
		WHERE id = $1
		RETURNING wallet_balance
	`, row.OperatorID, creditAmount).Scan(&balance)
	if err != nil {
		return nil, err
	}

	ref := fmt.Sprintf("recharge-%s", row.ID)
	_, err = tx.Exec(ctx, `
		INSERT INTO wallet_transactions (operator_profile_id, tx_type, amount, balance_after, reference)
		VALUES ($1, 'recharge', $2, $3, $4)
	`, row.OperatorID, creditAmount, balance, ref)
	if err != nil {
		return nil, err
	}

	tag, err := tx.Exec(ctx, `
		UPDATE wallet_recharges
		SET status = 'completed',
		    external_tx_id = COALESCE(NULLIF($2, ''), external_tx_id),
		    amount = $3,
		    completed_at = now()
		WHERE id = $1 AND status = 'pending'
	`, row.ID, externalTxID, creditAmount)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, fmt.Errorf("recharge not pending")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	userID, err := r.UserIDByOperator(ctx, row.OperatorID)
	if err != nil {
		return nil, err
	}

	return &CompleteResult{
		RechargeID:   row.ID.String(),
		OperatorID:   row.OperatorID,
		UserID:       userID,
		Amount:       creditAmount,
		BalanceAfter: balance,
	}, nil
}
