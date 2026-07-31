package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"alphaforge/internal/model"
	"alphaforge/internal/pglisten"
	"alphaforge/internal/repository"
)

// QuotaChecker provides fast in-memory lookups of user subscription plan quotas.
// Cache is loaded at startup and refreshed via PG LISTEN on quota_change channel.
type QuotaChecker struct {
	mu          sync.RWMutex
	cache       map[uuid.UUID]*model.SubscriptionPlan
	defaultPlan *model.SubscriptionPlan
	pg          *pgxpool.Pool
	repo        *repository.SubscriptionRepository
	pgListen    *pglisten.Listener
	log         *zap.Logger
}

func NewQuotaChecker(repo *repository.SubscriptionRepository, pg *pgxpool.Pool, log *zap.Logger) *QuotaChecker {
	return &QuotaChecker{
		cache: make(map[uuid.UUID]*model.SubscriptionPlan),
		pg:    pg,
		repo:  repo,
		log:   log,
	}
}

// LoadAll loads all active user subscriptions and the free plan into the cache.
// Called at startup.
func (q *QuotaChecker) LoadAll(ctx context.Context) error {
	// Load the default free plan from DB (single source of truth).
	freePlan, err := q.repo.GetPlanByName(ctx, "free")
	if err != nil {
		return fmt.Errorf("quota checker: load free plan: %w", err)
	}
	if freePlan == nil {
		return fmt.Errorf("quota checker: free plan not found in database")
	}

	rows, err := q.pg.Query(ctx,
		`SELECT ups.user_id, sp.id, sp.name, sp.display_name, sp.price_monthly::text, sp.price_yearly::text,
		        sp.max_ai_tokens_monthly, sp.max_strategies, sp.max_backtests_daily, sp.max_live_strategies,
		        sp.max_symbols_per_strategy, sp.max_mt_accounts, sp.capability_tier, sp.features::text, sp.sort_order, sp.is_active,
		        sp.created_at, sp.updated_at
		 FROM user_platform_subscriptions ups
		 JOIN subscription_plans sp ON sp.id = ups.plan_id
		 WHERE ups.status = 'active'`)
	if err != nil {
		return err
	}
	defer rows.Close()

	q.mu.Lock()
	defer q.mu.Unlock()
	q.defaultPlan = freePlan
	q.cache = make(map[uuid.UUID]*model.SubscriptionPlan)
	for rows.Next() {
		var uid uuid.UUID
		var p model.SubscriptionPlan
		if err := rows.Scan(&uid, &p.ID, &p.Name, &p.DisplayName, &p.PriceMonthly, &p.PriceYearly,
			&p.MaxAITokensMonthly, &p.MaxStrategies, &p.MaxBacktestsDaily, &p.MaxLiveStrategies,
			&p.MaxSymbolsPerStrategy, &p.MaxMTAccounts, &p.CapabilityTier, &p.Features, &p.SortOrder, &p.IsActive,
			&p.CreatedAt, &p.UpdatedAt); err != nil {
			return err
		}
		q.cache[uid] = &p
	}
	q.log.Info("QuotaChecker: loaded subscriptions", zap.Int("count", len(q.cache)))
	return rows.Err()
}

// GetPlan returns the cached plan for a user, or the free plan as default.
func (q *QuotaChecker) GetPlan(userID uuid.UUID) *model.SubscriptionPlan {
	q.mu.RLock()
	defer q.mu.RUnlock()
	if p, ok := q.cache[userID]; ok {
		return p
	}
	return q.defaultPlan
}

// CheckAITokenQuota returns remaining AI tokens for the current month.
// Returns -1 if unlimited (0 in plan = unlimited).
func (q *QuotaChecker) CheckAITokenQuota(userID uuid.UUID, usedThisMonth int) int {
	plan := q.GetPlan(userID)
	if plan.MaxAITokensMonthly == 0 {
		return -1 // unlimited
	}
	remaining := plan.MaxAITokensMonthly - usedThisMonth
	if remaining < 0 {
		return 0
	}
	return remaining
}

// CheckStrategyLimit returns true if the user can create more strategies.
func (q *QuotaChecker) CheckStrategyLimit(userID uuid.UUID, currentCount int) bool {
	plan := q.GetPlan(userID)
	if plan.MaxStrategies == 0 {
		return true // unlimited
	}
	return currentCount < plan.MaxStrategies
}

// CheckBacktestDailyLimit returns true if the user can run more backtests today.
func (q *QuotaChecker) CheckBacktestDailyLimit(userID uuid.UUID, todayCount int) bool {
	plan := q.GetPlan(userID)
	if plan.MaxBacktestsDaily == 0 {
		return true // unlimited
	}
	return todayCount < plan.MaxBacktestsDaily
}

// CheckLiveStrategyLimit returns true if the user can run more live strategies.
func (q *QuotaChecker) CheckLiveStrategyLimit(userID uuid.UUID, currentLive int) bool {
	plan := q.GetPlan(userID)
	if plan.MaxLiveStrategies == 0 {
		return true // unlimited
	}
	return currentLive < plan.MaxLiveStrategies
}

// CheckSymbolLimit returns true if the user can add more symbols to a strategy.
func (q *QuotaChecker) CheckSymbolLimit(userID uuid.UUID, currentSymbols int) bool {
	plan := q.GetPlan(userID)
	if plan.MaxSymbolsPerStrategy == 0 {
		return true // unlimited
	}
	return currentSymbols < plan.MaxSymbolsPerStrategy
}

// CheckAccountLimit returns true if the user can bind more MT accounts.
func (q *QuotaChecker) CheckAccountLimit(userID uuid.UUID, currentCount int) bool {
	plan := q.GetPlan(userID)
	if plan.MaxMTAccounts == 0 {
		return true // unlimited
	}
	return currentCount < plan.MaxMTAccounts
}

// GetCapabilityTier returns the user's capability tier from their plan.
func (q *QuotaChecker) GetCapabilityTier(userID uuid.UUID) int {
	plan := q.GetPlan(userID)
	return plan.CapabilityTier
}

// SetPgListen injects the PG LISTEN listener for event-driven cache refresh.
func (q *QuotaChecker) SetPgListen(l *pglisten.Listener) {
	q.pgListen = l
}

// StartRefreshLoop starts a background goroutine that refreshes the cache
// when PG LISTEN notifications arrive on the quota_change channel.
// Falls back to no refresh if pgListen is not set.
func (q *QuotaChecker) StartRefreshLoop(ctx context.Context) {
	if q.pgListen == nil {
		q.log.Warn("QuotaChecker: pgListen not set, cache will not auto-refresh")
		return
	}

	go func() {
		notifCh, cancel, err := q.pgListen.Listen(ctx, "quota_change")
		if err != nil {
			q.log.Error("QuotaChecker: LISTEN quota_change failed", zap.Error(err))
			return
		}
		defer cancel()

		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-notifCh:
				if !ok {
					return
				}
				if err := q.LoadAll(ctx); err != nil {
					q.log.Warn("QuotaChecker: LISTEN-driven refresh failed", zap.Error(err))
				}
			}
		}
	}()
}
