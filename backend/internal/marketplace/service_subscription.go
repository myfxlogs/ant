// Package marketplace — subscription operations (Subscribe/Unsubscribe/ListSubscriptions).
package marketplace

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
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
func (s *Service) RenewSubscriptions(ctx context.Context) (renewed, failed int, err error) {
	rows, qErr := s.pg.Query(ctx, `
		SELECT us.id, us.subscriber_user_id::text, us.target_user_id::text,
		       us.target_strategy_id::text, ms.price_amount::text, ms.title,
		       ms.platform_fee_rate::text, COALESCE(ms.refund_window_days, 7)
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
		subID, userID, publisherID, strategyID, title string
		priceAmount, platformFeeRate                  string
		refundWindowDays                             int
	}
	var renewals []renewalItem
	for rows.Next() {
		var r renewalItem
		if err := rows.Scan(&r.subID, &r.userID, &r.publisherID, &r.strategyID, &r.priceAmount, &r.title, &r.platformFeeRate, &r.refundWindowDays); err != nil {
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

		// Parse amounts as decimal for precise arithmetic.
		priceDec, decErr := decimal.NewFromString(p.priceAmount)
		if decErr != nil {
			_ = tx.Rollback(ctx)
			failed++
			continue
		}
		feeRateDec, err := decimal.NewFromString(p.platformFeeRate)
		if err != nil {
			s.log.Warn("renewal: invalid platform_fee_rate", zap.String("subID", p.subID), zap.String("feeRate", p.platformFeeRate), zap.Error(err))
			_ = tx.Rollback(ctx)
			failed++
			continue
		}
		feeDec := priceDec.Mul(feeRateDec)
		pubDec := priceDec.Sub(feeDec)

		amountStr := priceDec.StringFixed(2)
		negAmountStr := "-" + amountStr

		// 1. Lock buyer wallet and deduct.
		uid, err := uuid.Parse(p.userID)
		if err != nil {
			s.log.Warn("renewal: invalid user_id", zap.String("subID", p.subID), zap.String("userID", p.userID), zap.Error(err))
			_ = tx.Rollback(ctx)
			failed++
			continue
		}
		var buyerWalletID uuid.UUID
		if err := tx.QueryRow(ctx,
			`SELECT id FROM user_wallets WHERE user_id = $1 FOR UPDATE`,
			uid,
		).Scan(&buyerWalletID); err != nil {
			_ = tx.Rollback(ctx)
			failed++
			continue
		}

		// 2. Charge buyer via AdjustBalanceTx (hash chain + idempotency).
		buyerDesc := fmt.Sprintf("Subscription renewal: %s", p.title)
		_, chargeErr := s.walletRepo.AdjustBalanceTx(ctx, tx, buyerWalletID, uid,
			negAmountStr, TxTypePurchase, buyerDesc, nil, IdemKeyRenewBuy+p.subID)

		if chargeErr != nil {
			// Insufficient balance — deactivate and commit (not rollback,
			// otherwise the deactivation is lost and we retry forever).
			if _, dErr := tx.Exec(ctx, `UPDATE user_subscriptions SET active = false WHERE id = $1`, p.subID); dErr != nil {
				s.log.Warn("renewal: deactivate failed", zap.String("subID", p.subID), zap.Error(dErr))
				_ = tx.Rollback(ctx)
				failed++
				continue
			}
			if cErr := tx.Commit(ctx); cErr != nil {
				s.log.Warn("renewal: deactivate commit failed", zap.String("subID", p.subID), zap.Error(cErr))
			}
			failed++
			continue
		}

		// 3. Create frozen settlement for renewal (Phase 5.4).
		pubID, err := uuid.Parse(p.publisherID)
		if err != nil {
			s.log.Warn("renewal: invalid publisher_id", zap.String("subID", p.subID), zap.String("publisherID", p.publisherID), zap.Error(err))
			_ = tx.Rollback(ctx)
			failed++
			continue
		}
		pubAmountStr := pubDec.StringFixed(2)
		feeStr := feeDec.StringFixed(2)
		subUUID, err := uuid.Parse(p.subID)
		if err != nil {
			s.log.Warn("renewal: invalid sub_id", zap.String("subID", p.subID), zap.Error(err))
			_ = tx.Rollback(ctx)
			failed++
			continue
		}
		if err := s.createFrozenSettlementTx(ctx, tx, subUUID, uid, pubID, amountStr, feeStr, pubAmountStr, p.refundWindowDays); err != nil {
			s.log.Warn("renewal: create settlement failed", zap.String("subID", p.subID), zap.Error(err))
			_ = tx.Rollback(ctx)
			failed++
			continue
		}

		// 4. Extend subscription by 30 days.
		if _, eErr := tx.Exec(ctx,
			`UPDATE user_subscriptions SET expires_at = now() + INTERVAL '30 days' WHERE id = $1`,
			p.subID,
		); eErr != nil {
			s.log.Warn("renewal: extend failed", zap.String("subID", p.subID), zap.Error(eErr))
			_ = tx.Rollback(ctx)
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
		// D5: Align first tick to next midnight UTC + jitter (0-5min) to avoid
		// thundering herd on startup and make renewal timing deterministic.
		now := time.Now().UTC()
		nextMidnight := now.Truncate(24 * time.Hour).Add(24 * time.Hour)
		initialDelay := time.Until(nextMidnight) + time.Duration(time.Now().UnixNano()%int64(5*time.Minute))
		if initialDelay < 0 {
			initialDelay = 0
		}

		// Run once at startup to catch overdue renewals.
		renewed, failed, _ := s.RenewSubscriptions(ctx)
		if renewed+failed > 0 {
			log.Info("subscription renewal startup run", zap.Int("renewed", renewed), zap.Int("failed", failed))
		}

		// Wait until first aligned tick, then switch to 24h ticker.
		timer := time.NewTimer(initialDelay)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				runCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
				renewed, failed, err := s.RenewSubscriptions(runCtx)
				cancel()
				if err != nil {
					log.Error("subscription renewal failed", zap.Error(err))
				} else if renewed+failed > 0 {
					log.Info("subscription renewal complete", zap.Int("renewed", renewed), zap.Int("failed", failed))
				}
				timer.Reset(24 * time.Hour)
			}
		}
	}()
}

// CanAccessCode returns true if the user can view the full strategy code.
// Access is granted if the user is the template owner, has an active
// subscription/purchase to the published strategy, or has an active free trial.
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
	if exists {
		return true, nil
	}

	// Check if user has an active free trial.
	hasTrial, _ := s.HasActiveTrial(ctx, userID, strategyID)
	return hasTrial, nil
}
