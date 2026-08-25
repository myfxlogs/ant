// Package marketplace — subscription operations (Subscribe/Unsubscribe/ListSubscriptions).
package marketplace

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// SubscriptionItem represents a user's subscription.
type SubscriptionItem struct {
	SubscriptionID string
	TargetUserID   string
	StrategyID     string
	Kind           string
	Active         bool
	CreatedAt      time.Time
	ExpiresAt      *time.Time
}

// Subscribe subscribes a user to a published strategy. Only free strategies
// can be obtained via Subscribe; paid strategies must use PurchaseStrategy.
//
// publisherUserID is used as a fallback when the strategy is not found in
// marketplace_strategies (e.g. internal subscriptions). When the
// strategy is published, the actual publisher is read from the DB and the
// parameter is overwritten.
func (s *Service) Subscribe(ctx context.Context, userID, publisherUserID, strategyID, kind string) (string, error) {
	// Look up strategy metadata from marketplace_strategies.
	// Use the DB publisher_id as the source of truth; fall back to the
	// client-supplied value only when the strategy is not in the marketplace
	// (e.g. internal subscriptions).
	var priceModel string
	var priceAmountStr string
	var dbPublisherID string
	err := s.pg.QueryRow(ctx,
		`SELECT price_model, COALESCE(price_amount::text, '0'), publisher_id::text FROM marketplace_strategies WHERE strategy_id = $1 AND status = 'published'`,
		strategyID,
	).Scan(&priceModel, &priceAmountStr, &dbPublisherID)
	if err == nil {
		// Strategy is published — use DB publisher as source of truth.
		publisherUserID = dbPublisherID

		// Guard: any priced strategy (once, subscription, etc.) must go through PurchaseStrategy.
		priceDec, _ := decimal.NewFromString(priceAmountStr)
		if priceModel != PriceModelFree && priceDec.IsPositive() {
			return "", fmt.Errorf("marketplace: paid strategies require purchase, not subscribe")
		}
	}
	// If the strategy is not in marketplace_strategies (e.g. internal subscriptions),
	// fall back to the provided publisherUserID.

	// Guard: cannot subscribe to your own strategy (self-subscribe is nonsensical).
	if userID == publisherUserID {
		return "", fmt.Errorf("marketplace: cannot subscribe to your own strategy")
	}

	// M2: Use ON CONFLICT to handle re-subscription after unsubscribe.
	// The UNIQUE(subscriber_user_id, target_strategy_id, kind) constraint
	// would block re-subscribe since the old row still exists with active=false.
	// M5: Wrap in transaction to atomically increment total_subscribers.
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("marketplace: subscribe: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	id := uuid.New().String()
	var subID string
	err = tx.QueryRow(ctx, `
		INSERT INTO user_subscriptions (id, subscriber_user_id, target_user_id, target_strategy_id, kind, active)
		VALUES ($1, $2, $3, $4, $5, true)
		ON CONFLICT (subscriber_user_id, target_strategy_id, kind)
		DO UPDATE SET active = true, created_at = now()
		RETURNING id::text
	`, id, userID, publisherUserID, strategyID, kind).Scan(&subID)
	if err != nil {
		return "", fmt.Errorf("marketplace: subscribe: %w", err)
	}

	// M5: Increment subscriber counter only for marketplace strategies.
	if dbPublisherID != "" {
		sid, perr := uuid.Parse(strategyID)
		if perr == nil {
			if _, err := tx.Exec(ctx,
				`UPDATE marketplace_strategies SET total_subscribers = total_subscribers + 1 WHERE strategy_id = $1`,
				sid,
			); err != nil {
				return "", fmt.Errorf("marketplace: subscribe: increment subscribers: %w", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("marketplace: subscribe: commit: %w", err)
	}
	s.pubCache.clear()
	return subID, nil
}

// Unsubscribe deactivates a subscription and decrements the subscriber counter.
func (s *Service) Unsubscribe(ctx context.Context, userID, subscriptionID string) error {
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return fmt.Errorf("marketplace: unsubscribe begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Deactivate the subscription and get the strategy ID.
	var strategyID string
	err = tx.QueryRow(ctx,
		`UPDATE user_subscriptions SET active = false
		 WHERE id = $1 AND subscriber_user_id = $2 AND active = true
		 RETURNING target_strategy_id::text`,
		subscriptionID, userID,
	).Scan(&strategyID)
	if err != nil {
		return fmt.Errorf("marketplace: unsubscribe: %w", err)
	}

	// Decrement subscriber counter (floor at 0).
	_, err = tx.Exec(ctx,
		`UPDATE marketplace_strategies SET total_subscribers = GREATEST(total_subscribers - 1, 0)
		 WHERE strategy_id = $1`,
		strategyID,
	)
	if err != nil {
		return fmt.Errorf("marketplace: decrement subscribers: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("marketplace: unsubscribe commit: %w", err)
	}
	s.pubCache.clear()
	return nil
}

// ListSubscriptions returns active subscriptions for a user.
func (s *Service) ListSubscriptions(ctx context.Context, userID string) ([]SubscriptionItem, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: invalid user_id: %w", err)
	}

	// Lazily notify subscriptions expiring within 3 days (push-first, no cron).
	go s.notifySubExpiring(context.WithoutCancel(ctx), uid)

	rows, err := s.pg.Query(ctx, `
		SELECT id, target_user_id, target_strategy_id, kind, active, created_at, expires_at
		FROM user_subscriptions
		WHERE subscriber_user_id = $1 AND active = true
		ORDER BY created_at DESC
	`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SubscriptionItem
	for rows.Next() {
		var sub SubscriptionItem
		if err := rows.Scan(&sub.SubscriptionID, &sub.TargetUserID, &sub.StrategyID, &sub.Kind, &sub.Active, &sub.CreatedAt, &sub.ExpiresAt); err != nil {
			return nil, err
		}
		out = append(out, sub)
	}
	return out, rows.Err()
}

// RenewSubscriptions finds active subscription-model subscriptions that have
// expired and attempts to charge for another period. Successful renewals get
// a 30-day extension; failed ones (insufficient balance) are deactivated.
// Revenue is split between publisher and platform, mirroring PurchaseStrategy.
// Returns the number of renewed and failed subscriptions.
type renewalItem struct {
	subID, userID, publisherID, strategyID, title string
	priceAmount, platformFeeRate                  string
	refundWindowDays                              int
}
