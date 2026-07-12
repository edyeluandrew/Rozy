package trip

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrNotOperatorTrip   = errors.New("trip not assigned to this operator")
	ErrInvalidPIN        = errors.New("invalid trip pin")
	ErrInvalidTransition = errors.New("invalid trip status for this action")
	ErrArrivedRequired   = errors.New("mark arrived at pickup first")
)

type tripRow struct {
	ID            uuid.UUID
	Status        string
	OperatorID    *uuid.UUID
	PassengerID   uuid.UUID
	RideType      string
	EstimatedFare *int64
	EstimatedKm   *float64
	PINHash       *string
	ArrivedAt     *time.Time
}

func hashPIN(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

func (r *Repository) GetOperatorIDByUser(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `SELECT id FROM operator_profiles WHERE user_id = $1`, userID).Scan(&id)
	return id, err
}

func (r *Repository) loadTripForOperator(ctx context.Context, tripID, operatorID uuid.UUID) (*tripRow, error) {
	var row tripRow
	err := r.pool.QueryRow(ctx, `
		SELECT id, status::text, operator_profile_id, passenger_id, ride_type::text,
		       estimated_fare, estimated_km, trip_pin_hash, arrived_at
		FROM trips WHERE id = $1
	`, tripID).Scan(
		&row.ID, &row.Status, &row.OperatorID, &row.PassengerID, &row.RideType,
		&row.EstimatedFare, &row.EstimatedKm, &row.PINHash, &row.ArrivedAt,
	)
	if err != nil {
		return nil, err
	}
	if row.OperatorID == nil || *row.OperatorID != operatorID {
		return nil, ErrNotOperatorTrip
	}
	return &row, nil
}

func (r *Repository) GetActiveTripForOperator(ctx context.Context, operatorID uuid.UUID) (*Trip, error) {
	var t Trip
	var arrivedAt *time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT id, status::text, ride_type::text, estimated_fare, estimated_km,
		       ST_Y(pickup::geometry), ST_X(pickup::geometry),
		       ST_Y(destination::geometry), ST_X(destination::geometry),
		       arrived_at
		FROM trips
		WHERE operator_profile_id = $1
		  AND status IN ('driver_assigned','driver_arriving','in_progress')
		ORDER BY assigned_at DESC NULLS LAST, created_at DESC
		LIMIT 1
	`, operatorID).Scan(
		&t.ID, &t.Status, &t.RideType, &t.EstimatedFare, &t.EstimatedKm,
		&t.PickupLat, &t.PickupLng, &t.DestLat, &t.DestLng,
		&arrivedAt,
	)
	if err != nil {
		return nil, err
	}
	if arrivedAt != nil {
		t.ArrivedAt = arrivedAt.UTC().Format(time.RFC3339)
	}
	return &t, nil
}

func (r *Repository) MarkArrived(ctx context.Context, tripID, operatorID uuid.UUID) error {
	row, err := r.loadTripForOperator(ctx, tripID, operatorID)
	if err != nil {
		return err
	}
	if row.Status != string(StatusDriverArriving) {
		return fmt.Errorf("%w: %s", ErrInvalidTransition, row.Status)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE trips
		SET arrived_at = now(), updated_at = now()
		WHERE id = $1 AND operator_profile_id = $2 AND status = 'driver_arriving' AND arrived_at IS NULL
	`, tripID, operatorID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		// Idempotent if already arrived.
		var exists bool
		_ = tx.QueryRow(ctx, `SELECT arrived_at IS NOT NULL FROM trips WHERE id = $1`, tripID).Scan(&exists)
		if !exists {
			return ErrInvalidTransition
		}
	} else {
		_, _ = tx.Exec(ctx, `
			INSERT INTO trip_events (trip_id, from_status, to_status, metadata)
			VALUES ($1, 'driver_arriving', 'driver_arriving', jsonb_build_object('event', 'arrived'))
		`, tripID)
	}

	return tx.Commit(ctx)
}

func (r *Repository) StartTrip(ctx context.Context, tripID, operatorID uuid.UUID, pin string) error {
	row, err := r.loadTripForOperator(ctx, tripID, operatorID)
	if err != nil {
		return err
	}
	if err := MustTransition(Status(row.Status), StatusInProgress); err != nil {
		return ErrInvalidTransition
	}
	if row.ArrivedAt == nil {
		return ErrArrivedRequired
	}
	if row.PINHash == nil || hashPIN(pin) != *row.PINHash {
		return ErrInvalidPIN
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE trips
		SET status = 'in_progress', started_at = now(), updated_at = now()
		WHERE id = $1 AND operator_profile_id = $2 AND status = 'driver_arriving'
	`, tripID, operatorID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrInvalidTransition
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO trip_events (trip_id, from_status, to_status, metadata)
		VALUES ($1, 'driver_arriving', 'in_progress', jsonb_build_object('operator_id', $2::text))
	`, tripID, operatorID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

type CompleteResult struct {
	TripID        string `json:"trip_id"`
	Status        string `json:"status"`
	FinalFare     int64  `json:"final_fare"`
	RozyFee       int64  `json:"rozy_fee"`
	WalletBalance int64  `json:"wallet_balance"`
}

func (r *Repository) CompleteTrip(ctx context.Context, tripID, operatorID uuid.UUID) (*CompleteResult, error) {
	row, err := r.loadTripForOperator(ctx, tripID, operatorID)
	if err != nil {
		return nil, err
	}
	if row.Status != string(StatusInProgress) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidTransition, row.Status)
	}
	if row.EstimatedFare == nil {
		return nil, fmt.Errorf("trip missing estimated fare")
	}

	rule, err := r.GetFareRule(ctx, row.RideType)
	if err != nil {
		return nil, err
	}

	finalFare := *row.EstimatedFare
	rozyFee := rule.RozyFeeFixed
	ref := fmt.Sprintf("trip-%s-fee", tripID.String())

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE trips
		SET status = 'completed',
		    final_fare = $3,
		    rozy_fee = $4,
		    actual_km = estimated_km,
		    completed_at = now(),
		    updated_at = now()
		WHERE id = $1 AND operator_profile_id = $2 AND status = 'in_progress'
	`, tripID, operatorID, finalFare, rozyFee)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrInvalidTransition
	}

	var walletBalance int64
	err = tx.QueryRow(ctx, `
		UPDATE operator_profiles
		SET wallet_balance = wallet_balance - $2,
		    total_trips = total_trips + 1,
		    status = CASE
		      WHEN wallet_balance - $2 < wallet_min_balance THEN 'wallet_blocked'::operator_status
		      ELSE 'available'::operator_status
		    END,
		    updated_at = now()
		WHERE id = $1
		RETURNING wallet_balance
	`, operatorID, rozyFee).Scan(&walletBalance)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO wallet_transactions (operator_profile_id, tx_type, amount, balance_after, trip_id, reference)
		VALUES ($1, 'trip_fee', $2, $3, $4, $5)
	`, operatorID, -rozyFee, walletBalance, tripID, ref)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO trip_events (trip_id, from_status, to_status, metadata)
		VALUES ($1, 'in_progress', 'completed', jsonb_build_object(
			'operator_id', $2::text,
			'final_fare', to_jsonb($3::bigint),
			'rozy_fee', to_jsonb($4::bigint)
		))
	`, tripID, operatorID.String(), finalFare, rozyFee)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &CompleteResult{
		TripID:        tripID.String(),
		Status:        string(StatusCompleted),
		FinalFare:     finalFare,
		RozyFee:       rozyFee,
		WalletBalance: walletBalance,
	}, nil
}

func (r *Repository) GetTripForOperator(ctx context.Context, tripID, operatorID uuid.UUID) (*Trip, error) {
	var t Trip
	var arrivedAt *time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT id, status::text, ride_type::text, estimated_fare, estimated_km,
		       ST_Y(pickup::geometry), ST_X(pickup::geometry),
		       ST_Y(destination::geometry), ST_X(destination::geometry),
		       pickup_landmark, dest_landmark, arrived_at
		FROM trips
		WHERE id = $1 AND operator_profile_id = $2
	`, tripID, operatorID).Scan(
		&t.ID, &t.Status, &t.RideType, &t.EstimatedFare, &t.EstimatedKm,
		&t.PickupLat, &t.PickupLng, &t.DestLat, &t.DestLng,
		&t.PickupLandmark, &t.DestLandmark, &arrivedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTripNotFound
	}
	if err != nil {
		return nil, err
	}
	if arrivedAt != nil {
		t.ArrivedAt = arrivedAt.UTC().Format(time.RFC3339)
	}
	return &t, nil
}
