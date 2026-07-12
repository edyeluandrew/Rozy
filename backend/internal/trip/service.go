package trip

import (
	"context"
	"errors"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rozy/backend/internal/realtime"
)

type Dispatcher interface {
	MatchTrip(ctx context.Context, tripID uuid.UUID) error
}

type Service struct {
	repo     *Repository
	dispatch Dispatcher
	events   realtime.Publisher
}

func NewService(repo *Repository, dispatch Dispatcher, events realtime.Publisher) *Service {
	if events == nil {
		events = realtime.NopPublisher{}
	}
	return &Service{repo: repo, dispatch: dispatch, events: events}
}

func (s *Service) Estimate(ctx context.Context, in EstimateInput) (*EstimateResult, error) {
	rule, err := s.repo.GetFareRule(ctx, in.RideType)
	if err != nil {
		return nil, err
	}
	h := HaversineKm(in.Pickup.Lat, in.Pickup.Lng, in.Dest.Lat, in.Dest.Lng)
	km := EstimateDistanceKm(0, h, rule.RoadFactorFallback)
	fare := CalculateFare(rule, km)
	return &EstimateResult{
		DistanceKm:    km,
		EstimatedFare: fare,
		Currency:      "UGX",
	}, nil
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, in CreateInput) (*Trip, error) {
	role, err := s.repo.GetUserRole(ctx, userID)
	if err != nil {
		return nil, err
	}
	if role != "passenger" {
		return nil, ErrNotPassenger
	}

	active, err := s.repo.HasActiveTrip(ctx, userID)
	if err != nil {
		return nil, err
	}
	if active {
		return nil, ErrActiveTripExists
	}

	est, err := s.Estimate(ctx, EstimateInput{
		Pickup: in.Pickup, Dest: in.Dest, RideType: in.RideType,
	})
	if err != nil {
		return nil, err
	}

	pin, pinHash, err := GeneratePIN()
	if err != nil {
		return nil, err
	}

	trip, err := s.repo.CreateTrip(ctx, userID, in, est.EstimatedFare, est.DistanceKm, pinHash)
	if err != nil {
		return nil, err
	}
	trip.TripPIN = pin

	if s.dispatch != nil {
		tripUUID, parseErr := uuid.Parse(trip.ID)
		if parseErr == nil {
			if err := s.dispatch.MatchTrip(ctx, tripUUID); err != nil {
				log.Printf("[trip] dispatch for %s: %v", trip.ID, err)
			}
			if refreshed, err := s.repo.GetTrip(ctx, tripUUID, userID); err == nil {
				refreshed.TripPIN = pin
				trip = refreshed
			}
		}
	}

	return trip, nil
}

func (s *Service) Get(ctx context.Context, userID, tripID uuid.UUID) (*Trip, error) {
	t, err := s.repo.GetTrip(ctx, tripID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTripNotFound
	}
	return t, err
}

func (s *Service) Active(ctx context.Context, userID uuid.UUID) (*Trip, error) {
	t, err := s.repo.GetActiveTrip(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return t, err
}

func (s *Service) Cancel(ctx context.Context, userID, tripID uuid.UUID) error {
	err := s.repo.CancelTrip(ctx, tripID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrTripNotFound
	}
	if err == nil {
		PublishStatus(ctx, s.repo, s.events, tripID, "cancelled")
	}
	return err
}

func (s *Service) ArrivedForOperator(ctx context.Context, userID, tripID uuid.UUID) error {
	opID, err := s.repo.GetOperatorIDByUser(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotOperatorTrip
		}
		return err
	}
	return s.repo.MarkArrived(ctx, tripID, opID)
}

func (s *Service) StartForOperator(ctx context.Context, userID, tripID uuid.UUID, pin string) error {
	opID, err := s.repo.GetOperatorIDByUser(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotOperatorTrip
		}
		return err
	}
	return s.repo.StartTrip(ctx, tripID, opID, pin)
}

func (s *Service) CompleteForOperator(ctx context.Context, userID, tripID uuid.UUID) (*CompleteResult, error) {
	opID, err := s.repo.GetOperatorIDByUser(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotOperatorTrip
		}
		return nil, err
	}
	return s.repo.CompleteTrip(ctx, tripID, opID)
}

func (s *Service) ActiveForOperator(ctx context.Context, userID uuid.UUID) (*Trip, error) {
	opID, err := s.repo.GetOperatorIDByUser(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	trip, err := s.repo.GetActiveTripForOperator(ctx, opID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return trip, err
}

func (s *Service) GetForOperator(ctx context.Context, userID, tripID uuid.UUID) (*Trip, error) {
	opID, err := s.repo.GetOperatorIDByUser(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotOperatorTrip
		}
		return nil, err
	}
	t, err := s.repo.GetTripForOperator(ctx, tripID, opID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTripNotFound
	}
	return t, err
}
