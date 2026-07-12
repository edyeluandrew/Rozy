package operator

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rozy/backend/internal/realtime"
	redisclient "github.com/rozy/backend/internal/platform/redis"
)

type GeoStore interface {
	SetOperatorLocation(ctx context.Context, rideType, operatorID string, lng, lat float64) error
	RemoveOperator(ctx context.Context, rideType, operatorID string) error
}

type Service struct {
	repo   *Repository
	redis  GeoStore
	events realtime.Publisher
}

func NewService(repo *Repository, redis GeoStore, events realtime.Publisher) *Service {
	if events == nil {
		events = realtime.NopPublisher{}
	}
	return &Service{repo: repo, redis: redis, events: events}
}

func (s *Service) Register(ctx context.Context, userID uuid.UUID, rideTypeRaw string) (*Profile, error) {
	role, err := s.repo.GetUserRole(ctx, userID)
	if err != nil {
		return nil, err
	}
	if role != "driver" {
		return nil, ErrNotDriver
	}

	if _, err := s.repo.GetByUserID(ctx, userID); err == nil {
		return nil, ErrAlreadyRegistered
	} else if !IsNotFound(err) {
		return nil, err
	}

	opType, rideType, err := ParseRideType(rideTypeRaw)
	if err != nil {
		return nil, err
	}

	cityID, err := s.repo.GetMbararaCityID(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("mbarara city not configured")
		}
		return nil, err
	}

	profile, err := s.repo.Create(ctx, userID, opType, rideType, cityID)
	if err != nil {
		if IsUniqueViolation(err) {
			return nil, ErrAlreadyRegistered
		}
		return nil, err
	}
	return profile, nil
}

func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	profile, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return profile, nil
}

func (s *Service) OperatorIDForUser(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	_, opID, err := s.operatorForUser(ctx, userID)
	return opID, err
}

func (s *Service) operatorForUser(ctx context.Context, userID uuid.UUID) (*Profile, uuid.UUID, error) {
	profile, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, uuid.Nil, err
	}
	opID, err := uuid.Parse(profile.ID)
	if err != nil {
		return nil, uuid.Nil, err
	}
	return profile, opID, nil
}

func (s *Service) GoOnline(ctx context.Context, userID uuid.UUID, lat, lng float64) (*Profile, error) {
	profile, opID, err := s.operatorForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	verified, err := s.repo.IsVerified(ctx, opID)
	if err != nil {
		return nil, err
	}
	if !verified {
		return nil, ErrNotVerified
	}

	if err := s.repo.EnforceWalletGate(ctx, opID); err != nil {
		return nil, err
	}

	switch profile.Status {
	case "offline", "available":
	default:
		return nil, fmt.Errorf("%w: %s", ErrCannotGoOnline, profile.Status)
	}

	if lat == 0 && lng == 0 {
		lat, lng = -0.6072, 30.6586
	}

	if err := s.repo.SetOnline(ctx, opID, lat, lng); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCannotGoOnline
		}
		return nil, err
	}

	if s.redis != nil {
		_ = s.redis.SetOperatorLocation(ctx, string(profile.RideType), profile.ID, lng, lat)
	}

	return s.repo.GetByUserID(ctx, userID)
}

func (s *Service) GoOffline(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	profile, opID, err := s.operatorForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	if err := s.repo.SetOffline(ctx, opID); err != nil {
		return nil, err
	}

	if s.redis != nil {
		_ = s.redis.RemoveOperator(ctx, string(profile.RideType), profile.ID)
	}

	return s.repo.GetByUserID(ctx, userID)
}

func (s *Service) UpdateLocation(ctx context.Context, userID uuid.UUID, lat, lng float64) error {
	profile, opID, err := s.operatorForUser(ctx, userID)
	if err != nil {
		return err
	}

	if profile.Status != "available" && profile.Status != "busy" {
		return ErrCannotGoOnline
	}

	if err := s.repo.UpdateLocation(ctx, opID, lat, lng); err != nil {
		return err
	}

	if s.redis != nil && profile.Status == "available" {
		return s.redis.SetOperatorLocation(ctx, string(profile.RideType), profile.ID, lng, lat)
	}

	s.publishDriverLocation(ctx, opID, lat, lng)
	return s.repo.TouchLocation(ctx, opID, lat, lng)
}

func (s *Service) publishDriverLocation(ctx context.Context, operatorID uuid.UUID, lat, lng float64) {
	passengerID, tripID, err := s.repo.ActiveTripPassenger(ctx, operatorID)
	if err != nil {
		return
	}

	s.events.PublishToUser(ctx, passengerID, "trip:driver_location", map[string]any{
		"trip_id": tripID.String(),
		"lat":     lat,
		"lng":     lng,
	})
}

func (s *Service) IncomingTrip(ctx context.Context, userID uuid.UUID) (*IncomingTrip, error) {
	_, opID, err := s.operatorForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	trip, err := s.repo.GetIncomingTrip(ctx, opID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoIncomingTrip
	}
	return trip, err
}

func (s *Service) CreditWallet(ctx context.Context, operatorID uuid.UUID, amount int64, ref string) error {
	return s.repo.CreditWallet(ctx, operatorID, amount, ref)
}

var _ GeoStore = (*redisclient.Client)(nil)
