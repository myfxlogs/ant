package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"alphaforge/internal/model"
)

// SubscriptionRepository provides database access for subscription plans and user subscriptions.
type SubscriptionRepository struct {
	pg *pgxpool.Pool
}

func NewSubscriptionRepository(pg *pgxpool.Pool) *SubscriptionRepository {
	return &SubscriptionRepository{pg: pg}
}

// ListPlans returns all active subscription plans ordered by sort_order.
func (r *SubscriptionRepository) ListPlans(ctx context.Context) ([]*model.SubscriptionPlan, error) {
	rows, err := r.pg.Query(ctx,
		`SELECT id, name, display_name, price_monthly::text, price_yearly::text,
		        max_ai_tokens_monthly, max_strategies, max_backtests_daily, max_live_strategies,
		        max_symbols_per_strategy, capability_tier, features::text, sort_order, is_active,
		        created_at, updated_at
		 FROM subscription_plans WHERE is_active = true ORDER BY sort_order`)
	if err != nil {
		return nil, fmt.Errorf("subscription repo: list plans: %w", err)
	}
	defer rows.Close()
	var out []*model.SubscriptionPlan
	for rows.Next() {
		var p model.SubscriptionPlan
		if err := rows.Scan(&p.ID, &p.Name, &p.DisplayName, &p.PriceMonthly, &p.PriceYearly,
			&p.MaxAITokensMonthly, &p.MaxStrategies, &p.MaxBacktestsDaily, &p.MaxLiveStrategies,
			&p.MaxSymbolsPerStrategy, &p.CapabilityTier, &p.Features, &p.SortOrder, &p.IsActive,
			&p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("subscription repo: scan plan: %w", err)
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

// GetPlanByName returns a single plan by name (e.g. "free", "pro", "enterprise").
func (r *SubscriptionRepository) GetPlanByName(ctx context.Context, name string) (*model.SubscriptionPlan, error) {
	var p model.SubscriptionPlan
	err := r.pg.QueryRow(ctx,
		`SELECT id, name, display_name, price_monthly::text, price_yearly::text,
		        max_ai_tokens_monthly, max_strategies, max_backtests_daily, max_live_strategies,
		        max_symbols_per_strategy, capability_tier, features::text, sort_order, is_active,
		        created_at, updated_at
		 FROM subscription_plans WHERE name = $1 AND is_active = true`, name).
		Scan(&p.ID, &p.Name, &p.DisplayName, &p.PriceMonthly, &p.PriceYearly,
			&p.MaxAITokensMonthly, &p.MaxStrategies, &p.MaxBacktestsDaily, &p.MaxLiveStrategies,
			&p.MaxSymbolsPerStrategy, &p.CapabilityTier, &p.Features, &p.SortOrder, &p.IsActive,
			&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("subscription repo: get plan by name: %w", err)
	}
	return &p, nil
}

// GetPlanByID returns a single plan by ID.
func (r *SubscriptionRepository) GetPlanByID(ctx context.Context, id uuid.UUID) (*model.SubscriptionPlan, error) {
	var p model.SubscriptionPlan
	err := r.pg.QueryRow(ctx,
		`SELECT id, name, display_name, price_monthly::text, price_yearly::text,
		        max_ai_tokens_monthly, max_strategies, max_backtests_daily, max_live_strategies,
		        max_symbols_per_strategy, capability_tier, features::text, sort_order, is_active,
		        created_at, updated_at
		 FROM subscription_plans WHERE id = $1`, id).
		Scan(&p.ID, &p.Name, &p.DisplayName, &p.PriceMonthly, &p.PriceYearly,
			&p.MaxAITokensMonthly, &p.MaxStrategies, &p.MaxBacktestsDaily, &p.MaxLiveStrategies,
			&p.MaxSymbolsPerStrategy, &p.CapabilityTier, &p.Features, &p.SortOrder, &p.IsActive,
			&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("subscription repo: get plan by id: %w", err)
	}
	return &p, nil
}

// GetActiveSubscription returns the user's active platform subscription, or nil if none.
// Unique index guarantees at most one active subscription per user.
func (r *SubscriptionRepository) GetActiveSubscription(ctx context.Context, userID uuid.UUID) (*model.UserPlatformSubscription, error) {
	var s model.UserPlatformSubscription
	err := r.pg.QueryRow(ctx,
		`SELECT id, user_id, plan_id, status, billing_cycle,
		        current_period_start, current_period_end, auto_renew, cancelled_at,
		        wallet_transaction_id, created_at, updated_at
		 FROM user_platform_subscriptions
		 WHERE user_id = $1 AND status = 'active'`, userID).
		Scan(&s.ID, &s.UserID, &s.PlanID, &s.Status, &s.BillingCycle,
			&s.CurrentPeriodStart, &s.CurrentPeriodEnd, &s.AutoRenew, &s.CancelledAt,
			&s.WalletTransactionID, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("subscription repo: get active: %w", err)
	}
	return &s, nil
}

// GetActiveSubscriptionTx returns the user's active subscription within a transaction (FOR UPDATE lock).
// Use this inside tx to prevent race conditions on concurrent subscribe/change-plan.
func (r *SubscriptionRepository) GetActiveSubscriptionTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (*model.UserPlatformSubscription, error) {
	var s model.UserPlatformSubscription
	err := tx.QueryRow(ctx,
		`SELECT id, user_id, plan_id, status, billing_cycle,
		        current_period_start, current_period_end, auto_renew, cancelled_at,
		        wallet_transaction_id, created_at, updated_at
		 FROM user_platform_subscriptions
		 WHERE user_id = $1 AND status = 'active'
		 FOR UPDATE`, userID).
		Scan(&s.ID, &s.UserID, &s.PlanID, &s.Status, &s.BillingCycle,
			&s.CurrentPeriodStart, &s.CurrentPeriodEnd, &s.AutoRenew, &s.CancelledAt,
			&s.WalletTransactionID, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("subscription repo: get active (tx): %w", err)
	}
	return &s, nil
}

// CreateSubscription inserts a new user platform subscription.
func (r *SubscriptionRepository) CreateSubscription(ctx context.Context, tx pgx.Tx, s *model.UserPlatformSubscription) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO user_platform_subscriptions (id, user_id, plan_id, status, billing_cycle,
		    current_period_start, current_period_end, auto_renew, wallet_transaction_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		s.ID, s.UserID, s.PlanID, s.Status, s.BillingCycle,
		s.CurrentPeriodStart, s.CurrentPeriodEnd, s.AutoRenew, s.WalletTransactionID)
	if err != nil {
		return fmt.Errorf("subscription repo: create: %w", err)
	}
	return nil
}

// DeactivateSubscription sets status to 'expired' for an active subscription.
func (r *SubscriptionRepository) DeactivateSubscription(ctx context.Context, tx pgx.Tx, subID uuid.UUID) error {
	_, err := tx.Exec(ctx,
		`UPDATE user_platform_subscriptions SET status = 'expired', updated_at = CURRENT_TIMESTAMP WHERE id = $1`, subID)
	if err != nil {
		return fmt.Errorf("subscription repo: deactivate: %w", err)
	}
	return nil
}

// CancelSubscription sets auto_renew=false and records cancelled_at.
func (r *SubscriptionRepository) CancelSubscription(ctx context.Context, subID uuid.UUID) error {
	_, err := r.pg.Exec(ctx,
		`UPDATE user_platform_subscriptions
		 SET auto_renew = false, cancelled_at = $2, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $1 AND status = 'active'`,
		subID, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("subscription repo: cancel: %w", err)
	}
	return nil
}

// UpdateSubscriptionPlan changes the plan_id and billing_cycle for an active subscription.
func (r *SubscriptionRepository) UpdateSubscriptionPlan(ctx context.Context, tx pgx.Tx, subID, planID uuid.UUID, billingCycle string, periodEnd time.Time) error {
	_, err := tx.Exec(ctx,
		`UPDATE user_platform_subscriptions
		 SET plan_id = $2, billing_cycle = $3, current_period_start = CURRENT_TIMESTAMP, current_period_end = $4, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $1`,
		subID, planID, billingCycle, periodEnd)
	if err != nil {
		return fmt.Errorf("subscription repo: update plan: %w", err)
	}
	return nil
}
