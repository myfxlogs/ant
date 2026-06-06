// Package marketplace — subscription operations (Subscribe/Unsubscribe/ListSubscriptions).
package marketplace

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SubscriptionItem represents a user's subscription.
type SubscriptionItem struct {
	SubscriptionID string
	TargetUserID   string
	StrategyID     string
	Kind           string
	Active         bool
	CreatedAt      time.Time
}

// Subscribe subscribes a user to a published strategy.
func (s *Service) Subscribe(ctx context.Context, userID, publisherUserID, strategyID, kind string) (string, error) {
	id := uuid.New().String()
	_, err := s.pg.Exec(ctx, `
		INSERT INTO user_subscriptions (id, subscriber_user_id, target_user_id, target_strategy_id, kind, active)
		VALUES ($1, $2, $3, $4, $5, true)
	`, id, userID, publisherUserID, strategyID, kind)
	if err != nil {
		return "", fmt.Errorf("marketplace: subscribe: %w", err)
	}
	return id, nil
}

// Unsubscribe deactivates a subscription.
func (s *Service) Unsubscribe(ctx context.Context, userID, subscriptionID string) error {
	_, err := s.pg.Exec(ctx, `
		UPDATE user_subscriptions SET active = false
		WHERE id = $1 AND subscriber_user_id = $2
	`, subscriptionID, userID)
	if err != nil {
		return fmt.Errorf("marketplace: unsubscribe: %w", err)
	}
	return nil
}

// ListSubscriptions returns active subscriptions for a user.
func (s *Service) ListSubscriptions(ctx context.Context, userID string) ([]SubscriptionItem, error) {
	rows, err := s.pg.Query(ctx, `
		SELECT id, target_user_id, target_strategy_id, kind, active, created_at
		FROM user_subscriptions
		WHERE subscriber_user_id::text = $1 AND active = true
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SubscriptionItem
	for rows.Next() {
		var sub SubscriptionItem
		if err := rows.Scan(&sub.SubscriptionID, &sub.TargetUserID, &sub.StrategyID, &sub.Kind, &sub.Active, &sub.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, sub)
	}
	return out, rows.Err()
}
