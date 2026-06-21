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

// Subscribe subscribes a user to a published strategy. Only free strategies
// can be obtained via Subscribe; paid strategies must use PurchaseStrategy.
func (s *Service) Subscribe(ctx context.Context, userID, publisherUserID, strategyID, kind string) (string, error) {
	// Guard: reject paid strategies — they must go through PurchaseStrategy.
	var priceModel string
	var priceAmount float64
	err := s.pg.QueryRow(ctx,
		`SELECT price_model, COALESCE(price_amount, 0) FROM marketplace_strategies WHERE strategy_id = $1`,
		strategyID,
	).Scan(&priceModel, &priceAmount)
	if err == nil && priceModel == "once" && priceAmount > 0 {
		return "", fmt.Errorf("marketplace: paid strategies require purchase, not subscribe")
	}
	// If the strategy is not in marketplace_strategies (e.g. internal subscriptions), allow.

	id := uuid.New().String()
	_, err = s.pg.Exec(ctx, `
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

// CanAccessCode returns true if the user can view the full strategy code.
// Access is granted if the user is the template owner OR has an active
// subscription/purchase to the published strategy.
func (s *Service) CanAccessCode(ctx context.Context, userID, strategyID string) (bool, error) {
	// Check if user is the template owner.
	var ownerID string
	err := s.pg.QueryRow(ctx,
		`SELECT user_id::text FROM strategy_templates WHERE id::text = $1`,
		strategyID,
	).Scan(&ownerID)
	if err != nil {
		return false, nil // template not found → deny
	}
	if ownerID == userID {
		return true, nil // owner always has access
	}

	// Check if user has an active subscription.
	var exists bool
	err = s.pg.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM user_subscriptions WHERE subscriber_user_id::text = $1 AND target_strategy_id = $2 AND active = true)`,
		userID, strategyID,
	).Scan(&exists)
	if err != nil {
		return false, nil
	}
	return exists, nil
}
