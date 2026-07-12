package admin

import (
	"context"

	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Queue(ctx context.Context, status string) ([]QueueItem, error) {
	items, err := s.repo.ListQueue(ctx, status)
	if items == nil {
		items = []QueueItem{}
	}
	return items, err
}

func (s *Service) Detail(ctx context.Context, id uuid.UUID) (*SubmissionDetail, error) {
	return s.repo.GetSubmission(ctx, id)
}

func (s *Service) Approve(ctx context.Context, submissionID, adminID uuid.UUID) error {
	return s.repo.Approve(ctx, submissionID, adminID)
}

func (s *Service) Reject(ctx context.Context, submissionID, adminID uuid.UUID, reason string) error {
	return s.repo.Reject(ctx, submissionID, adminID, reason)
}

func (s *Service) Stats(ctx context.Context) (map[string]int, error) {
	pending, err := s.repo.PendingCount(ctx)
	if err != nil {
		return nil, err
	}
	active, err := s.repo.ActiveTripsCount(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]int{
		"pending_verifications": pending,
		"active_trips":          active,
	}, nil
}

func (s *Service) ActiveTrips(ctx context.Context) ([]ActiveTripItem, error) {
	items, err := s.repo.ListActiveTrips(ctx)
	if items == nil {
		items = []ActiveTripItem{}
	}
	return items, err
}

func (s *Service) Operators(ctx context.Context, status string) ([]OperatorListItem, error) {
	items, err := s.repo.ListOperators(ctx, status)
	if items == nil {
		items = []OperatorListItem{}
	}
	return items, err
}
