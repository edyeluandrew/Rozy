package realtime

import (
	"context"

	"github.com/google/uuid"
)

// Publisher pushes realtime events to connected clients.
type Publisher interface {
	PublishToUser(ctx context.Context, userID uuid.UUID, event string, payload any)
}

// NopPublisher discards events (tests / redis-less boot).
type NopPublisher struct{}

func (NopPublisher) PublishToUser(context.Context, uuid.UUID, string, any) {}
