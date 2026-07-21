package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// StartPlatformRenewalLoop runs a daily ticker that:
//  1. Expires active subscriptions whose current_period_end has passed and auto_renew=false.
//  2. Attempts to auto-renew subscriptions with auto_renew=true by charging the wallet.
//
// Call during server startup.
func (s *SubscriptionService) StartPlatformRenewalLoop(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		s.renewPlatformSubscriptions(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
				s.renewPlatformSubscriptions(runCtx)
				cancel()
			}
		}
	}()
}

func (s *SubscriptionService) renewPlatformSubscriptions(ctx context.Context) {
	now := time.Now().UTC()

	// 1. Expire subscriptions whose period has ended and auto_renew is false.
	_, err := s.pg.Exec(ctx,
		`UPDATE user_platform_subscriptions
		 SET status = 'expired', updated_at = CURRENT_TIMESTAMP
		 WHERE status = 'active' AND auto_renew = false AND current_period_end < $1`,
		now)
	if err != nil {
		s.log.Error("platform subscription expiry failed", zap.Error(err))
	}

	// 2. Auto-renew subscriptions whose period has ended and auto_renew is true.
	rows, err := s.pg.Query(ctx,
		`SELECT id, user_id, plan_id, billing_cycle
		 FROM user_platform_subscriptions
		 WHERE status = 'active' AND auto_renew = true AND current_period_end < $1`,
		now)
	if err != nil {
		s.log.Error("platform subscription renewal query failed", zap.Error(err))
		return
	}
	defer rows.Close()

	for rows.Next() {
		var subID, userID, planID uuid.UUID
		var billingCycle string
		if err := rows.Scan(&subID, &userID, &planID, &billingCycle); err != nil {
			s.log.Error("platform renewal scan failed", zap.Error(err))
			continue
		}
		if err := s.renewOnePlatformSubscription(ctx, subID, userID, planID, billingCycle); err != nil {
			s.log.Warn("platform subscription renewal failed, expiring",
				zap.String("subID", subID.String()), zap.Error(err))
			_, _ = s.pg.Exec(ctx,
				`UPDATE user_platform_subscriptions SET status = 'expired', updated_at = CURRENT_TIMESTAMP WHERE id = $1`,
				subID)
		}
	}
}

func (s *SubscriptionService) renewOnePlatformSubscription(ctx context.Context, subID, userID, planID uuid.UUID, billingCycle string) error {
	plan, err := s.repo.GetPlanByID(ctx, planID)
	if err != nil {
		return err
	}
	if plan == nil {
		return ErrPlanNotFound
	}

	priceStr := plan.PriceMonthly
	if billingCycle == "yearly" {
		priceStr = plan.PriceYearly
	}
	price, err := decimal.NewFromString(priceStr)
	if err != nil {
		return fmt.Errorf("subscription renewal: parse price: %w", err)
	}

	// Free plan: just extend the period.
	if price.LessThanOrEqual(decimal.Zero) {
		return s.extendPlatformSubscription(ctx, subID, billingCycle)
	}

	// Paid plan: charge wallet then extend.
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return fmt.Errorf("subscription renewal: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	walletRepo := s.walletSvc.Repo()
	wallet, err := walletRepo.GetByUserIDTx(ctx, tx, userID)
	if err != nil {
		return fmt.Errorf("subscription renewal: get wallet: %w", err)
	}
	if wallet == nil {
		return ErrWalletNotFound
	}

	balance, err := decimal.NewFromString(wallet.Balance)
	if err != nil {
		return fmt.Errorf("subscription renewal: parse balance: %w", err)
	}
	if balance.LessThan(price) {
		return fmt.Errorf("insufficient balance for renewal: have %s, need %s", wallet.Balance, priceStr)
	}

	_, err = walletRepo.AdjustBalanceTx(ctx, tx, wallet.ID, userID, price.Neg().String(), "purchase",
		fmt.Sprintf("Platform subscription: %s (%s) - renewal", plan.DisplayName, billingCycle), nil, "sub-renewal-"+subID.String())
	if err != nil {
		return fmt.Errorf("subscription renewal: charge wallet: %w", err)
	}

	now := time.Now().UTC()
	periodEnd := now.AddDate(0, 1, 0)
	if billingCycle == "yearly" {
		periodEnd = now.AddDate(1, 0, 0)
	}
	_, err = tx.Exec(ctx,
		`UPDATE user_platform_subscriptions
		 SET current_period_start = $1, current_period_end = $2, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $3`,
		now, periodEnd, subID)
	if err != nil {
		return fmt.Errorf("subscription renewal: extend period: %w", err)
	}

	return tx.Commit(ctx)
}

func (s *SubscriptionService) extendPlatformSubscription(ctx context.Context, subID uuid.UUID, billingCycle string) error {
	now := time.Now().UTC()
	periodEnd := now.AddDate(0, 1, 0)
	if billingCycle == "yearly" {
		periodEnd = now.AddDate(1, 0, 0)
	}
	_, err := s.pg.Exec(ctx,
		`UPDATE user_platform_subscriptions
		 SET current_period_start = $1, current_period_end = $2, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $3`,
		now, periodEnd, subID)
	return err
}
