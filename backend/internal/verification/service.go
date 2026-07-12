package verification

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Status(ctx context.Context, userID uuid.UUID) (*Status, error) {
	opID, _, _, err := s.repo.GetOperatorByUser(ctx, userID)
	if err != nil {
		return nil, ErrNotOperator
	}
	return s.repo.LatestSubmission(ctx, opID)
}

func (s *Service) Submit(ctx context.Context, userID uuid.UUID, in SubmitInput) (uuid.UUID, error) {
	opID, rideType, _, err := s.repo.GetOperatorByUser(ctx, userID)
	if err != nil {
		return uuid.Nil, ErrNotOperator
	}

	latest, err := s.repo.LatestSubmission(ctx, opID)
	if err != nil {
		return uuid.Nil, err
	}
	if latest.HasSubmission && latest.Status == "pending" {
		return uuid.Nil, ErrPendingExists
	}

	if err := validateSubmit(in, rideType); err != nil {
		return uuid.Nil, err
	}

	ninHash := HashNIN(in.NIN)
	taken, err := s.repo.NINExists(ctx, ninHash, opID)
	if err != nil {
		return uuid.Nil, err
	}
	if taken {
		return uuid.Nil, ErrNINTaken
	}

	id, err := s.repo.Submit(ctx, opID, rideType, in, ninHash)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if strings.Contains(pgErr.ConstraintName, "nin") {
				return uuid.Nil, ErrNINTaken
			}
			return uuid.Nil, fmt.Errorf("duplicate vehicle plate")
		}
		return uuid.Nil, err
	}
	return id, nil
}

func (s *Service) OperatorID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	id, _, _, err := s.repo.GetOperatorByUser(ctx, userID)
	return id, err
}

func validateSubmit(in SubmitInput, rideType string) error {
	if strings.TrimSpace(in.LegalName) == "" || strings.TrimSpace(in.NIN) == "" ||
		strings.TrimSpace(in.Plate) == "" || strings.TrimSpace(in.PermitNumber) == "" {
		return ErrInvalidSubmission
	}
	if len(in.Documents) < 3 {
		return fmt.Errorf("%w: upload required documents", ErrInvalidSubmission)
	}
	required := map[string]bool{"nin_front": true, "selfie": true, "permit": true}
	for _, d := range in.Documents {
		delete(required, d.DocType)
	}
	if len(required) > 0 {
		return fmt.Errorf("%w: missing required documents", ErrInvalidSubmission)
	}
	if rideType == "boda" && strings.TrimSpace(in.BikeColor) == "" {
		return ErrInvalidSubmission
	}
	if (rideType == "car_basic" || rideType == "car_xl") && strings.TrimSpace(in.CarMake) == "" {
		return ErrInvalidSubmission
	}
	return nil
}
