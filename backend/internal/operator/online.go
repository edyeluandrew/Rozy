package operator

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrNotVerified    = errors.New("operator not verified yet")
	ErrWalletBlocked  = errors.New("wallet balance below minimum")
	ErrCannotGoOnline = errors.New("cannot go online in current status")
	ErrNoIncomingTrip = errors.New("no incoming trip")
)

type IncomingTrip struct {
	ID            string  `json:"id"`
	Status        string  `json:"status"`
	RideType      string  `json:"ride_type"`
	EstimatedFare *int64  `json:"estimated_fare,omitempty"`
	PickupLat     float64 `json:"pickup_lat"`
	PickupLng     float64 `json:"pickup_lng"`
	DestLat       float64 `json:"dest_lat"`
	DestLng       float64 `json:"dest_lng"`
	PickupLandmark string `json:"pickup_landmark,omitempty"`
}

func (r *Repository) IsVerified(ctx context.Context, operatorID uuid.UUID) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM verified_operator_snapshots WHERE operator_profile_id = $1)
	`, operatorID).Scan(&ok)
	return ok, err
}

func (r *Repository) SetOnline(ctx context.Context, operatorID uuid.UUID, lat, lng float64) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE operator_profiles
		SET status = 'available', last_lat = $2, last_lng = $3, last_location_at = now(), updated_at = now()
		WHERE id = $1 AND status IN ('offline', 'available')
	`, operatorID, lat, lng)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) SetOffline(ctx context.Context, operatorID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE operator_profiles
		SET status = 'offline', updated_at = now()
		WHERE id = $1 AND status IN ('offline', 'available', 'busy')
	`, operatorID)
	return err
}

func (r *Repository) UpdateLocation(ctx context.Context, operatorID uuid.UUID, lat, lng float64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE operator_profiles
		SET last_lat = $2, last_lng = $3, last_location_at = now(), updated_at = now()
		WHERE id = $1 AND status IN ('available', 'busy')
	`, operatorID, lat, lng)
	return err
}

func (r *Repository) GetIncomingTrip(ctx context.Context, operatorID uuid.UUID) (*IncomingTrip, error) {
	var t IncomingTrip
	var fare *int64
	var landmark *string
	err := r.pool.QueryRow(ctx, `
		SELECT t.id, t.status::text, t.ride_type::text, t.estimated_fare,
		       ST_Y(t.pickup::geometry), ST_X(t.pickup::geometry),
		       ST_Y(t.destination::geometry), ST_X(t.destination::geometry),
		       t.pickup_landmark
		FROM trips t
		WHERE t.operator_profile_id = $1 AND t.status = 'driver_assigned'
		ORDER BY t.assigned_at DESC LIMIT 1
	`, operatorID).Scan(
		&t.ID, &t.Status, &t.RideType, &fare,
		&t.PickupLat, &t.PickupLng, &t.DestLat, &t.DestLng, &landmark,
	)
	if err != nil {
		return nil, err
	}
	t.EstimatedFare = fare
	if landmark != nil {
		t.PickupLandmark = *landmark
	}
	return &t, nil
}

func (r *Repository) ActiveTripPassenger(ctx context.Context, operatorID uuid.UUID) (passengerID, tripID uuid.UUID, err error) {
	err = r.pool.QueryRow(ctx, `
		SELECT t.passenger_id, t.id
		FROM trips t
		WHERE t.operator_profile_id = $1
		  AND t.status IN ('driver_assigned', 'driver_arriving', 'in_progress')
		ORDER BY t.assigned_at DESC NULLS LAST
		LIMIT 1
	`, operatorID).Scan(&passengerID, &tripID)
	return passengerID, tripID, err
}

func (r *Repository) CreditWallet(ctx context.Context, operatorID uuid.UUID, amount int64, reference string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var balance int64
	err = tx.QueryRow(ctx, `
		UPDATE operator_profiles
		SET wallet_balance = wallet_balance + $2,
		    status = CASE WHEN status = 'wallet_blocked' AND wallet_balance + $2 >= wallet_min_balance THEN 'offline' ELSE status END,
		    updated_at = now()
		WHERE id = $1
		RETURNING wallet_balance
	`, operatorID, amount).Scan(&balance)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO wallet_transactions (operator_profile_id, tx_type, amount, balance_after, reference)
		VALUES ($1, 'recharge', $2, $3, $4)
	`, operatorID, amount, balance, reference)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) OperatorIDByUser(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `SELECT id FROM operator_profiles WHERE user_id = $1`, userID).Scan(&id)
	return id, err
}

// WalletGate blocks operators with low balance.
func (r *Repository) EnforceWalletGate(ctx context.Context, operatorID uuid.UUID) error {
	var balance, minBal int64
	var status string
	err := r.pool.QueryRow(ctx, `
		SELECT wallet_balance, wallet_min_balance, status::text
		FROM operator_profiles WHERE id = $1
	`, operatorID).Scan(&balance, &minBal, &status)
	if err != nil {
		return err
	}
	if balance < minBal {
		if status == "available" || status == "offline" {
			_, _ = r.pool.Exec(ctx, `
				UPDATE operator_profiles SET status = 'wallet_blocked', updated_at = now() WHERE id = $1
			`, operatorID)
		}
		return ErrWalletBlocked
	}
	if status == "wallet_blocked" {
		_, _ = r.pool.Exec(ctx, `
			UPDATE operator_profiles SET status = 'offline', updated_at = now() WHERE id = $1
		`, operatorID)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, operatorID uuid.UUID) (*Profile, error) {
	var p Profile
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, operator_type::text, ride_type::text, status::text,
		       wallet_balance, wallet_min_balance
		FROM operator_profiles WHERE id = $1
	`, operatorID).Scan(
		&p.ID, &p.UserID, &p.OperatorType, &p.RideType, &p.Status,
		&p.WalletBalance, &p.WalletMin,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// TouchLocationTimestamp for busy operators without going offline.
func (r *Repository) TouchLocation(ctx context.Context, operatorID uuid.UUID, lat, lng float64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE operator_profiles SET last_lat=$2, last_lng=$3, last_location_at=$4, updated_at=now()
		WHERE id=$1
	`, operatorID, lat, lng, time.Now().UTC())
	return err
}
