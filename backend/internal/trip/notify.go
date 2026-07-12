package trip

import (
	"context"

	"github.com/google/uuid"

	"github.com/rozy/backend/internal/realtime"
)

func (r *Repository) TripAudience(ctx context.Context, tripID uuid.UUID) (passengerID uuid.UUID, operatorUserID *uuid.UUID, err error) {
	var opUser *uuid.UUID
	err = r.pool.QueryRow(ctx, `
		SELECT t.passenger_id, op.user_id
		FROM trips t
		LEFT JOIN operator_profiles op ON op.id = t.operator_profile_id
		WHERE t.id = $1
	`, tripID).Scan(&passengerID, &opUser)
	return passengerID, opUser, err
}

func PublishStatus(ctx context.Context, repo *Repository, events realtime.Publisher, tripID uuid.UUID, status string) {
	if events == nil {
		return
	}
	passengerID, opUserID, err := repo.TripAudience(ctx, tripID)
	if err != nil {
		return
	}
	payload := map[string]any{"trip_id": tripID.String(), "status": status}
	events.PublishToUser(ctx, passengerID, "trip:status", payload)
	if opUserID != nil {
		events.PublishToUser(ctx, *opUserID, "trip:status", payload)
	}
}
