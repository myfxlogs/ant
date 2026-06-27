// Package marketplace — subscription operations (Subscribe/Unsubscribe/ListSubscriptions).
package marketplace

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
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
//
// publisherUserID is used as a fallback when the strategy is not found in
// marketplace_strategies (e.g. internal/copy-trade subscriptions). When the
// strategy is published, the actual publisher is read from the DB and the
// parameter is overwritten.
func (s *Service) Subscribe(ctx context.Context, userID, publisherUserID, strategyID, kind string) (string, error) {
	// Look up strategy metadata from marketplace_strategies.
	// Use the DB publisher_id as the source of truth; fall back to the
	// client-supplied value only when the strategy is not in the marketplace
	// (e.g. internal subscriptions).
	var priceModel string
	var priceAmount float64
	var dbPublisherID string
	err := s.pg.QueryRow(ctx,
		`SELECT price_model, COALESCE(price_amount, 0), publisher_id::text FROM marketplace_strategies WHERE strategy_id = $1 AND status = 'published'`,
		strategyID,
	).Scan(&priceModel, &priceAmount, &dbPublisherID)
	if err == nil {
		// Strategy is published — use DB publisher as source of truth.
		publisherUserID = dbPublisherID

		// Guard: any priced strategy (once, subscription, etc.) must go through PurchaseStrategy.
		if priceModel != PriceModelFree && priceAmount > 0 {
			return "", fmt.Errorf("marketplace: paid strategies require purchase, not subscribe")
		}
	}
	// If the strategy is not in marketplace_strategies (e.g. internal subscriptions),
	// fall back to the provided publisherUserID.

	// Guard: cannot subscribe to your own strategy (self-subscribe is nonsensical).
	if userID == publisherUserID {
		return "", fmt.Errorf("marketplace: cannot subscribe to your own strategy")
	}

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

// ListActiveSubscribers returns active subscriptions for a specific strategy.
// Used by CopyTradeEngine to efficiently find subscribers for signal replication.
// NOTE: TargetUserID is populated with subscriber_user_id (the account to copy TO).
func (s *Service) ListActiveSubscribers(ctx context.Context, strategyID string) ([]SubscriptionItem, error) {
	rows, err := s.pg.Query(ctx, `
		SELECT id, subscriber_user_id, target_strategy_id, kind, active, created_at
		FROM user_subscriptions
		WHERE target_strategy_id = $1 AND active = true
		  AND (expires_at IS NULL OR expires_at > now())
		ORDER BY created_at DESC
	`, strategyID)
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

// Unsubscribe deactivates a subscription and decrements the subscriber counter.
func (s *Service) Unsubscribe(ctx context.Context, userID, subscriptionID string) error {
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return fmt.Errorf("marketplace: unsubscribe begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

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
	publishedCacheClear()
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

// RenewSubscriptions finds active subscription-model subscriptions that have
// expired and attempts to charge for another period. Successful renewals get
// a 30-day extension; failed ones (insufficient balance) are deactivated.
// Returns the number of renewed and failed subscriptions.
func (s *Service) RenewSubscriptions(ctx context.Context) (renewed, failed int, err error) {
	rows, qErr := s.pg.Query(ctx, `
		SELECT us.id, us.subscriber_user_id::text, us.target_strategy_id::text,
		       ms.price_amount, ms.title, ms.platform_fee_rate
		FROM user_subscriptions us
		JOIN marketplace_strategies ms ON ms.strategy_id = us.target_strategy_id
		WHERE us.kind = $1 AND us.active = true
		  AND us.expires_at IS NOT NULL AND us.expires_at <= now()
		LIMIT 100`, SubKindSubscription)
	if qErr != nil {
		return 0, 0, qErr
	}
	defer rows.Close()

	type renewalItem struct {
		subID, userID, strategyID, title string
		priceAmount, platformFeeRate     float64
	}
	var renewals []renewalItem
	for rows.Next() {
		var r renewalItem
		if err := rows.Scan(&r.subID, &r.userID, &r.strategyID, &r.priceAmount, &r.title, &r.platformFeeRate); err != nil {
			continue
		}
		renewals = append(renewals, r)
	}
	rows.Close()

	for _, p := range renewals {
		tx, txErr := s.pg.Begin(ctx)
		if txErr != nil {
			failed++
			continue
		}

		amountStr := fmt.Sprintf("%.2f", p.priceAmount)
		var balAfter string
		chargeErr := tx.QueryRow(ctx,
			`UPDATE user_wallets SET balance = balance - $1::numeric, updated_at = now()
			 WHERE user_id = $2 AND balance >= $1::numeric
			 RETURNING balance::text`,
			amountStr, p.userID,
		).Scan(&balAfter)

		if chargeErr != nil {
			// Insufficient balance — deactivate.
			if _, dErr := tx.Exec(ctx, `UPDATE user_subscriptions SET active = false WHERE id = $1`, p.subID); dErr != nil {
				s.log.Warn("renewal: deactivate failed", zap.String("subID", p.subID), zap.Error(dErr))
			}
			tx.Rollback(ctx)
			failed++
			continue
		}

		// Extend subscription by 30 days.
		if _, eErr := tx.Exec(ctx,
			`UPDATE user_subscriptions SET expires_at = now() + INTERVAL '30 days' WHERE id = $1`,
			p.subID,
		); eErr != nil {
			s.log.Warn("renewal: extend failed", zap.String("subID", p.subID), zap.Error(eErr))
			tx.Rollback(ctx)
			failed++
			continue
		}

		// Record renewal transaction. balance_before = balAfter + amount (pre-deduction balance).
		uid, _ := uuid.Parse(p.userID)
		if _, iErr := tx.Exec(ctx,
			`INSERT INTO wallet_transactions (id, wallet_id, user_id, tx_type, amount, balance_before, balance_after, description)
			 VALUES ($1, (SELECT id FROM user_wallets WHERE user_id = $2), $2, $3, $4::numeric + $5::numeric, $6, $7)`,
			uuid.New(), uid, TxTypePurchase, amountStr, amountStr, balAfter,
			fmt.Sprintf("Subscription renewal: %s", p.title),
		); iErr != nil {
			s.log.Warn("renewal: insert tx failed", zap.String("subID", p.subID), zap.Error(iErr))
			tx.Rollback(ctx)
			failed++
			continue
		}

		if cErr := tx.Commit(ctx); cErr != nil {
			s.log.Warn("renewal: commit failed", zap.String("subID", p.subID), zap.Error(cErr))
			failed++
			continue
		}
		renewed++
	}

	return renewed, failed, nil
}

// StartRenewalLoop runs a daily subscription renewal ticker in a background
// goroutine. Call during server startup.
func (s *Service) StartRenewalLoop(ctx context.Context, log *zap.Logger) {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		// Run once at startup to catch overdue renewals.
		renewed, failed, _ := s.RenewSubscriptions(ctx)
		if renewed+failed > 0 {
			log.Info("subscription renewal startup run", zap.Int("renewed", renewed), zap.Int("failed", failed))
		}
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
				renewed, failed, err := s.RenewSubscriptions(runCtx)
				cancel()
				if err != nil {
					log.Error("subscription renewal failed", zap.Error(err))
				} else if renewed+failed > 0 {
					log.Info("subscription renewal complete", zap.Int("renewed", renewed), zap.Int("failed", failed))
				}
			}
		}
	}()
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

	// Check if user has an active, non-expired subscription.
	var exists bool
	err = s.pg.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM user_subscriptions
		 WHERE subscriber_user_id::text = $1 AND target_strategy_id = $2 AND active = true
		   AND (expires_at IS NULL OR expires_at > now()))`,
		userID, strategyID,
	).Scan(&exists)
	if err != nil {
		return false, nil
	}
	return exists, nil
}
