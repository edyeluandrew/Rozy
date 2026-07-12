package dispatch

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rozy/backend/internal/realtime"
	redisclient "github.com/rozy/backend/internal/platform/redis"
)

var ErrNoDrivers = errors.New("no available drivers found")

type Service struct {
	pool   *pgxpool.Pool
	redis  *redisclient.Client
	radius []float64 // km search radii
	events realtime.Publisher
}

func NewService(pool *pgxpool.Pool, redis *redisclient.Client, events realtime.Publisher) *Service {
	if events == nil {
		events = realtime.NopPublisher{}
	}
	return &Service{
		pool:   pool,
		redis:  redis,
		radius: []float64{3, 5, 8},
		events: events,
	}
}

type tripPickup struct {
	ID       uuid.UUID
	RideType string
	Lat      float64
	Lng      float64
}

func (s *Service) MatchTrip(ctx context.Context, tripID uuid.UUID) error {
	tp, err := s.getSearchingTrip(ctx, tripID)
	if err != nil {
		return err
	}
	if tp == nil {
		return nil
	}

	candidates, err := s.findCandidates(ctx, tp)
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		log.Printf("[dispatch] no drivers for trip %s (%s)", tripID, tp.RideType)
		return ErrNoDrivers
	}

	for _, opID := range candidates {
		assigned, err := s.tryAssign(ctx, tripID, opID)
		if err != nil {
			log.Printf("[dispatch] assign trip %s to %s failed: %v", tripID, opID, err)
			continue
		}
		if assigned {
			log.Printf("[dispatch] trip %s assigned to operator %s", tripID, opID)
			s.notifyAssigned(ctx, tripID, opID)
			return nil
		}
	}
	return ErrNoDrivers
}

func (s *Service) findCandidates(ctx context.Context, tp *tripPickup) ([]uuid.UUID, error) {
	if s.redis != nil {
		for _, km := range s.radius {
			nearby, err := s.redis.SearchNearby(ctx, tp.RideType, tp.Lng, tp.Lat, km, 10)
			if err != nil {
				log.Printf("[dispatch] redis search: %v", err)
				break
			}
			if len(nearby) > 0 {
				return filterAvailable(ctx, s.pool, nearby)
			}
		}
	}
	return s.findCandidatesPostgres(ctx, tp)
}

func filterAvailable(ctx context.Context, pool *pgxpool.Pool, nearby []redisclient.NearbyOperator) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(nearby))
	for _, n := range nearby {
		id, err := uuid.Parse(n.OperatorID)
		if err != nil {
			continue
		}
		ok, err := isAssignable(ctx, pool, id)
		if err != nil || !ok {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *Service) findCandidatesPostgres(ctx context.Context, tp *tripPickup) ([]uuid.UUID, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT op.id
		FROM operator_profiles op
		WHERE op.status = 'available'
		  AND op.ride_type = $1::ride_type
		  AND op.wallet_balance >= op.wallet_min_balance
		  AND op.last_lat IS NOT NULL
		  AND EXISTS (
		    SELECT 1 FROM verified_operator_snapshots vs WHERE vs.operator_profile_id = op.id
		  )
		  AND (
		    6371 * acos(
		      cos(radians($2)) * cos(radians(op.last_lat)) * cos(radians(op.last_lng) - radians($3))
		      + sin(radians($2)) * sin(radians(op.last_lat))
		    )
		  ) <= $4
		ORDER BY (
		    6371 * acos(
		      cos(radians($2)) * cos(radians(op.last_lat)) * cos(radians(op.last_lng) - radians($3))
		      + sin(radians($2)) * sin(radians(op.last_lat))
		    )
		  )
		LIMIT 10
	`, tp.RideType, tp.Lat, tp.Lng, s.radius[len(s.radius)-1])
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func isAssignable(ctx context.Context, pool *pgxpool.Pool, operatorID uuid.UUID) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM operator_profiles op
			WHERE op.id = $1
			  AND op.status = 'available'
			  AND op.wallet_balance >= op.wallet_min_balance
			  AND EXISTS (
			    SELECT 1 FROM verified_operator_snapshots vs WHERE vs.operator_profile_id = op.id
			  )
		)
	`, operatorID).Scan(&ok)
	return ok, err
}

func (s *Service) tryAssign(ctx context.Context, tripID, operatorID uuid.UUID) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	var currentStatus string
	err = tx.QueryRow(ctx, `
		SELECT status::text FROM trips WHERE id = $1 FOR UPDATE
	`, tripID).Scan(&currentStatus)
	if err != nil {
		return false, err
	}
	if currentStatus != "searching" {
		return false, nil
	}

	var opStatus string
	err = tx.QueryRow(ctx, `
		SELECT status::text FROM operator_profiles WHERE id = $1 FOR UPDATE
	`, operatorID).Scan(&opStatus)
	if err != nil {
		return false, err
	}
	if opStatus != "available" {
		return false, nil
	}

	_, err = tx.Exec(ctx, `
		UPDATE operator_profiles SET status = 'busy', updated_at = now() WHERE id = $1
	`, operatorID)
	if err != nil {
		return false, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE trips
		SET operator_profile_id = $2, status = 'driver_assigned', assigned_at = now(), updated_at = now()
		WHERE id = $1 AND status = 'searching'
	`, tripID, operatorID)
	if err != nil {
		return false, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO trip_events (trip_id, from_status, to_status, metadata)
		VALUES ($1, 'searching', 'driver_assigned', jsonb_build_object('operator_id', $2::text))
	`, tripID, operatorID)
	if err != nil {
		return false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) ReleaseOperator(ctx context.Context, tripID, operatorID uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE operator_profiles SET status = 'available', updated_at = now()
		WHERE id = $1 AND status = 'busy'
	`, operatorID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE trips
		SET operator_profile_id = NULL, status = 'searching', assigned_at = NULL, updated_at = now()
		WHERE id = $1 AND operator_profile_id = $2 AND status = 'driver_assigned'
	`, tripID, operatorID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *Service) getSearchingTrip(ctx context.Context, tripID uuid.UUID) (*tripPickup, error) {
	var tp tripPickup
	err := s.pool.QueryRow(ctx, `
		SELECT id, ride_type::text, ST_Y(pickup::geometry), ST_X(pickup::geometry)
		FROM trips WHERE id = $1 AND status = 'searching'
	`, tripID).Scan(&tp.ID, &tp.RideType, &tp.Lat, &tp.Lng)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &tp, nil
}

// RejectAndRematch releases operator and tries next driver.
func (s *Service) RejectAndRematch(ctx context.Context, tripID, operatorID uuid.UUID) error {
	if err := s.ReleaseOperator(ctx, tripID, operatorID); err != nil {
		return err
	}
	return s.MatchTrip(ctx, tripID)
}

func (s *Service) notifyAssigned(ctx context.Context, tripID, operatorID uuid.UUID) {
	var passengerID, operatorUserID uuid.UUID
	var status, rideType string
	var fare *int64
	var pickupLat, pickupLng, destLat, destLng float64
	var pickupLandmark *string

	err := s.pool.QueryRow(ctx, `
		SELECT t.passenger_id, op.user_id, t.status::text, t.ride_type::text, t.estimated_fare,
		       ST_Y(t.pickup::geometry), ST_X(t.pickup::geometry),
		       ST_Y(t.destination::geometry), ST_X(t.destination::geometry),
		       t.pickup_landmark
		FROM trips t
		JOIN operator_profiles op ON op.id = $2
		WHERE t.id = $1
	`, tripID, operatorID).Scan(
		&passengerID, &operatorUserID, &status, &rideType, &fare,
		&pickupLat, &pickupLng, &destLat, &destLng, &pickupLandmark,
	)
	if err != nil {
		log.Printf("[dispatch] notify assign: %v", err)
		return
	}

	tripPayload := map[string]any{
		"trip_id":     tripID.String(),
		"status":      status,
		"ride_type":   rideType,
		"pickup_lat":  pickupLat,
		"pickup_lng":  pickupLng,
		"dest_lat":    destLat,
		"dest_lng":    destLng,
	}
	if fare != nil {
		tripPayload["estimated_fare"] = *fare
	}
	if pickupLandmark != nil {
		tripPayload["pickup_landmark"] = *pickupLandmark
	}

	s.events.PublishToUser(ctx, operatorUserID, "operator:ride_request", tripPayload)
	s.events.PublishToUser(ctx, operatorUserID, "trip:status", map[string]any{
		"trip_id": tripID.String(),
		"status":  status,
	})
	s.events.PublishToUser(ctx, passengerID, "trip:status", map[string]any{
		"trip_id": tripID.String(),
		"status":  status,
	})
	s.events.PublishToUser(ctx, passengerID, "trip:assigned", tripPayload)
}

func (s *Service) AcceptTrip(ctx context.Context, tripID, operatorID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE trips
		SET status = 'driver_arriving', updated_at = now()
		WHERE id = $1 AND operator_profile_id = $2 AND status = 'driver_assigned'
	`, tripID, operatorID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("trip not assigned to operator")
	}
	_, _ = s.pool.Exec(ctx, `
		INSERT INTO trip_events (trip_id, from_status, to_status, metadata)
		VALUES ($1, 'driver_assigned', 'driver_arriving', jsonb_build_object('operator_id', $2::text))
	`, tripID, operatorID)
	return nil
}
