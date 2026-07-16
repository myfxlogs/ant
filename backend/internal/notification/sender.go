package notification

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"alphaforge/internal/repository"
)

// Sender creates notifications and broadcasts them to active SSE subscribers.
// Services inject a *Sender to emit notifications without needing both
// NotificationRepository and Subscriber separately.
type Sender struct {
	repo *repository.NotificationRepository
	sub  *Subscriber
	log  *zap.Logger
}

// NewSender creates a Sender ready for injection.
func NewSender(repo *repository.NotificationRepository, sub *Subscriber, log *zap.Logger) *Sender {
	return &Sender{repo: repo, sub: sub, log: log}
}

// Send persists a notification to PostgreSQL and broadcasts it to all active
// SSE subscribers for the given user. Returns the inserted row.
func (s *Sender) Send(
	ctx context.Context,
	userID uuid.UUID,
	typ, title, message string,
	data *structpb.Struct,
) (repository.NotificationRow, error) {
	row, err := s.repo.Insert(ctx, userID, typ, title, message, data)
	if err != nil {
		s.log.Error("notification sender: insert failed",
			zap.String("userID", userID.String()),
			zap.String("type", typ),
			zap.Error(err))
		return repository.NotificationRow{}, fmt.Errorf("notification insert: %w", err)
	}

	s.sub.Publish(userID, row)
	return row, nil
}
