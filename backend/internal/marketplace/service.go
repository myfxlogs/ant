package marketplace

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/google/uuid"
)

// Service implements the C2C marketplace (strategy publish + subscribe).
type Service struct {
	pg *pgxpool.Pool
}

// New creates a marketplace service.
func New(pg *pgxpool.Pool) *Service {
	return &Service{pg: pg}
}

// PublishedStrategy represents a strategy published to the marketplace.
type PublishedStrategy struct {
	PublishID       string
	StrategyID      string
	StrategyName    string
	PublisherUserID string
	PublishedAt     time.Time
}

// SubscriptionItem represents a user's subscription.
type SubscriptionItem struct {
	SubscriptionID string
	TargetUserID   string
	StrategyID     string
	Kind           string
	Active         bool
	CreatedAt      time.Time
}

// Publish adds a strategy to the marketplace.
func (s *Service) Publish(ctx context.Context, userID, strategyID string) (string, error) {
	id := uuid.New().String()
	_, err := s.pg.Exec(ctx, `
		INSERT INTO user_strategy_publishes (id, user_id, platform_strategy_id, published_at)
		VALUES ($1, $2, $3, now())
	`, id, userID, strategyID)
	if err != nil {
		return "", fmt.Errorf("marketplace: publish: %w", err)
	}
	return id, nil
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
	return err
}

// ListPublished returns strategies published by a user.
func (s *Service) ListPublished(ctx context.Context, userID string, limit int) ([]PublishedStrategy, error) {
	if limit <= 0 { limit = 50 }
	rows, err := s.pg.Query(ctx, `
		SELECT usp.id, usp.platform_strategy_id, COALESCE(ps.name, usp.platform_strategy_id::text),
			usp.user_id, usp.published_at
		FROM user_strategy_publishes usp
		LEFT JOIN platform_strategies ps ON ps.id::text = usp.platform_strategy_id::text
		WHERE usp.user_id::text = $1
		ORDER BY usp.published_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []PublishedStrategy
	for rows.Next() {
		var p PublishedStrategy
		if err := rows.Scan(&p.PublishID, &p.StrategyID, &p.StrategyName, &p.PublisherUserID, &p.PublishedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListSubscriptions returns active subscriptions for a user.
func (s *Service) ListSubscriptions(ctx context.Context, userID string) ([]SubscriptionItem, error) {
	rows, err := s.pg.Query(ctx, `
		SELECT id, target_user_id, target_strategy_id, kind, active, created_at
		FROM user_subscriptions
		WHERE subscriber_user_id::text = $1 AND active = true
		ORDER BY created_at DESC
	`, userID)
	if err != nil { return nil, err }
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
