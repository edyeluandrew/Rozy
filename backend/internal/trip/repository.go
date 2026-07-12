package trip

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrActiveTripExists = errors.New("passenger already has an active trip")
	ErrTripNotFound     = errors.New("trip not found")
	ErrNotPassenger     = errors.New("passenger account required")
)

type Coordinate struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type EstimateInput struct {
	Pickup   Coordinate `json:"pickup"`
	Dest     Coordinate `json:"dest"`
	RideType string     `json:"ride_type"`
}

type EstimateResult struct {
	DistanceKm    float64 `json:"distance_km"`
	EstimatedFare int64   `json:"estimated_fare"`
	Currency      string  `json:"currency"`
}

type CreateInput struct {
	Pickup         Coordinate `json:"pickup"`
	Dest           Coordinate `json:"dest"`
	RideType       string     `json:"ride_type"`
	PickupAddress  string     `json:"pickup_address,omitempty"`
	PickupLandmark string     `json:"pickup_landmark,omitempty"`
	DestAddress    string     `json:"dest_address,omitempty"`
	DestLandmark   string     `json:"dest_landmark,omitempty"`
}

type Trip struct {
	ID             string  `json:"id"`
	Status         string  `json:"status"`
	RideType       string  `json:"ride_type"`
	EstimatedFare  *int64  `json:"estimated_fare,omitempty"`
	FinalFare      *int64  `json:"final_fare,omitempty"`
	RozyFee        *int64  `json:"rozy_fee,omitempty"`
	EstimatedKm    float64 `json:"estimated_km,omitempty"`
	TripPIN        string  `json:"trip_pin,omitempty"`
	PickupLat      float64 `json:"pickup_lat"`
	PickupLng      float64 `json:"pickup_lng"`
	DestLat        float64 `json:"dest_lat"`
	DestLng        float64 `json:"dest_lng"`
	PickupLandmark string  `json:"pickup_landmark,omitempty"`
	DestLandmark   string  `json:"dest_landmark,omitempty"`
	ArrivedAt      string  `json:"arrived_at,omitempty"`
	Driver         *DriverSnapshot `json:"driver,omitempty"`
	DriverDistanceKm *float64        `json:"driver_distance_km,omitempty"`
	DriverEtaMinutes *int            `json:"driver_eta_minutes,omitempty"`
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) GetUserRole(ctx context.Context, userID uuid.UUID) (string, error) {
	var role string
	err := r.pool.QueryRow(ctx, `SELECT role::text FROM users WHERE id = $1`, userID).Scan(&role)
	return role, err
}

func (r *Repository) GetFareRule(ctx context.Context, rideType string) (FareRule, error) {
	var rule FareRule
	err := r.pool.QueryRow(ctx, `
		SELECT fr.base_fare, fr.per_km_rate, fr.min_fare, fr.rozy_fee_fixed,
		       fr.round_to, fr.road_factor_fallback::float8, fr.min_billable_km::float8
		FROM fare_rules fr
		JOIN cities c ON c.id = fr.city_id
		WHERE c.slug = 'mbarara' AND fr.ride_type = $1::ride_type AND fr.is_active = true
	`, rideType).Scan(
		&rule.BaseFare, &rule.PerKmRate, &rule.MinFare, &rule.RozyFeeFixed,
		&rule.RoundTo, &rule.RoadFactorFallback, &rule.MinBillableKm,
	)
	return rule, err
}

func (r *Repository) HasActiveTrip(ctx context.Context, passengerID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM trips
			WHERE passenger_id = $1
			  AND status IN ('requested','searching','driver_assigned','driver_arriving','in_progress')
		)
	`, passengerID).Scan(&exists)
	return exists, err
}

func (r *Repository) CreateTrip(ctx context.Context, passengerID uuid.UUID, in CreateInput, fare int64, km float64, pinHash string) (*Trip, error) {
	var cityID uuid.UUID
	if err := r.pool.QueryRow(ctx, `SELECT id FROM cities WHERE slug = 'mbarara'`).Scan(&cityID); err != nil {
		return nil, err
	}

	var t Trip
	err := r.pool.QueryRow(ctx, `
		INSERT INTO trips (
			passenger_id, city_id, ride_type, status,
			pickup, pickup_address, pickup_landmark,
			destination, dest_address, dest_landmark,
			estimated_fare, estimated_km, trip_pin_hash
		) VALUES (
			$1, $2, $3::ride_type, 'searching',
			ST_SetSRID(ST_MakePoint($4, $5), 4326)::geography,
			$6, $7,
			ST_SetSRID(ST_MakePoint($8, $9), 4326)::geography,
			$10, $11,
			$12, $13, $14
		)
		RETURNING id, status::text, ride_type::text, estimated_fare, estimated_km,
		          ST_Y(pickup::geometry), ST_X(pickup::geometry),
		          ST_Y(destination::geometry), ST_X(destination::geometry)
	`, passengerID, cityID, in.RideType,
		in.Pickup.Lng, in.Pickup.Lat, in.PickupAddress, in.PickupLandmark,
		in.Dest.Lng, in.Dest.Lat, in.DestAddress, in.DestLandmark,
		fare, km, pinHash,
	).Scan(&t.ID, &t.Status, &t.RideType, &t.EstimatedFare, &t.EstimatedKm,
		&t.PickupLat, &t.PickupLng, &t.DestLat, &t.DestLng)
	return &t, err
}

func (r *Repository) GetTrip(ctx context.Context, tripID, passengerID uuid.UUID) (*Trip, error) {
	var t Trip
	var arrivedAt *time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT id, status::text, ride_type::text, estimated_fare, final_fare, rozy_fee, estimated_km,
		       ST_Y(pickup::geometry), ST_X(pickup::geometry),
		       ST_Y(destination::geometry), ST_X(destination::geometry),
		       pickup_landmark, dest_landmark, arrived_at
		FROM trips WHERE id = $1 AND passenger_id = $2
	`, tripID, passengerID).Scan(
		&t.ID, &t.Status, &t.RideType, &t.EstimatedFare, &t.FinalFare, &t.RozyFee, &t.EstimatedKm,
		&t.PickupLat, &t.PickupLng, &t.DestLat, &t.DestLng,
		&t.PickupLandmark, &t.DestLandmark, &arrivedAt,
	)
	if err != nil {
		return nil, err
	}
	if arrivedAt != nil {
		t.ArrivedAt = arrivedAt.UTC().Format(time.RFC3339)
	}
	if err := r.enrichPassengerTrip(ctx, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repository) GetActiveTrip(ctx context.Context, passengerID uuid.UUID) (*Trip, error) {
	var t Trip
	var arrivedAt *time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT id, status::text, ride_type::text, estimated_fare, final_fare, rozy_fee, estimated_km,
		       ST_Y(pickup::geometry), ST_X(pickup::geometry),
		       ST_Y(destination::geometry), ST_X(destination::geometry),
		       pickup_landmark, dest_landmark, arrived_at
		FROM trips
		WHERE passenger_id = $1
		  AND status IN ('requested','searching','driver_assigned','driver_arriving','in_progress')
		ORDER BY created_at DESC LIMIT 1
	`, passengerID).Scan(
		&t.ID, &t.Status, &t.RideType, &t.EstimatedFare, &t.FinalFare, &t.RozyFee, &t.EstimatedKm,
		&t.PickupLat, &t.PickupLng, &t.DestLat, &t.DestLng,
		&t.PickupLandmark, &t.DestLandmark, &arrivedAt,
	)
	if err != nil {
		return nil, err
	}
	if arrivedAt != nil {
		t.ArrivedAt = arrivedAt.UTC().Format(time.RFC3339)
	}
	if err := r.enrichPassengerTrip(ctx, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repository) CancelTrip(ctx context.Context, tripID, passengerID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE trips
		SET status = 'cancelled', cancelled_at = now(), cancelled_by = $2, updated_at = now()
		WHERE id = $1 AND passenger_id = $2
		  AND status IN ('requested','searching','driver_assigned','driver_arriving','in_progress')
	`, tripID, passengerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	_, err = tx.Exec(ctx, `
		UPDATE operator_profiles
		SET status = 'available', updated_at = now()
		WHERE id = (
			SELECT operator_profile_id FROM trips WHERE id = $1
		) AND status = 'busy'
	`, tripID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func GeneratePIN() (plain, hash string, err error) {
	n, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return "", "", err
	}
	plain = fmt.Sprintf("%04d", n.Int64())
	sum := sha256.Sum256([]byte(plain))
	hash = hex.EncodeToString(sum[:])
	return plain, hash, nil
}
