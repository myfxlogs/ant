package service

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/repository"
)

// ManagedSettingProvider reads admin-managed settings at runtime.
// Implemented by *agent.SettingsStore.
type ManagedSettingProvider interface {
	GetManagedSetting(ctx context.Context, key string) (string, error)
}

var (
	platformCostBreakerTripped = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_platform_cost_breaker_tripped_total",
			Help: "Number of times the platform daily cost circuit breaker has tripped.",
		},
		nil,
	)
	platformCostBreakerActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "ai_platform_cost_breaker_active",
			Help: "1 if the platform daily cost circuit breaker is currently tripped, 0 otherwise.",
		},
	)
	dailyQuotaRejected = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_daily_quota_rejected_total",
			Help: "Number of AI calls rejected due to daily quota limits.",
		},
		[]string{"reason"},
	)
)

const (
	defaultMaxSessionsPerDay = 5
	defaultMaxTokensPerDay   = 200_000

	managedKeyDailyMaxSessions = "ai_daily_max_sessions"
	managedKeyDailyMaxTokens   = "ai_daily_max_tokens"
	managedKeyDailyCostLimit   = "ai_daily_cost_limit_usd"
)

// DailyQuotaConfig holds per-user daily AI usage limits.
// Defaults are used when admin-managed settings are not configured.
type DailyQuotaConfig struct {
	MaxSessionsPerDay int // default: 5
	MaxTokensPerDay   int // default: 200_000
}

// DailyQuotaChecker enforces per-user daily AI call quotas.
// Config is re-read from agent_managed_settings on each check (with env var fallback).
type DailyQuotaChecker struct {
	defaults DailyQuotaConfig
	repo     *repository.AITokenUsageRepository
	provider ManagedSettingProvider // optional: nil = use env defaults
	log      *zap.Logger
}

func NewDailyQuotaChecker(repo *repository.AITokenUsageRepository, cfg DailyQuotaConfig, log *zap.Logger) *DailyQuotaChecker {
	if cfg.MaxSessionsPerDay <= 0 {
		cfg.MaxSessionsPerDay = defaultMaxSessionsPerDay
	}
	if cfg.MaxTokensPerDay <= 0 {
		cfg.MaxTokensPerDay = defaultMaxTokensPerDay
	}
	return &DailyQuotaChecker{defaults: cfg, repo: repo, log: log}
}

// SetManagedSettingProvider enables runtime config from agent_managed_settings.
func (q *DailyQuotaChecker) SetManagedSettingProvider(p ManagedSettingProvider) {
	q.provider = p
}

func (q *DailyQuotaChecker) resolveConfig(ctx context.Context) DailyQuotaConfig {
	cfg := q.defaults
	if q.provider == nil {
		return cfg
	}
	if v, _ := q.provider.GetManagedSetting(ctx, managedKeyDailyMaxSessions); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxSessionsPerDay = n
		}
	}
	if v, _ := q.provider.GetManagedSetting(ctx, managedKeyDailyMaxTokens); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxTokensPerDay = n
		}
	}
	return cfg
}

// CheckQuota returns nil if the user is within daily limits, or an error explaining the limit hit.
func (q *DailyQuotaChecker) CheckQuota(ctx context.Context, userID uuid.UUID) error {
	cfg := q.resolveConfig(ctx)
	tokens, err := q.repo.DailyTokenUsage(ctx, userID)
	if err != nil {
		q.log.Warn("daily quota check: failed to query token usage", zap.Error(err))
		return nil
	}
	if tokens >= cfg.MaxTokensPerDay {
		dailyQuotaRejected.WithLabelValues("tokens").Inc()
		return fmt.Errorf("daily token quota exceeded (%d/%d tokens used today)", tokens, cfg.MaxTokensPerDay)
	}
	sessions, err := q.repo.DailySessionCount(ctx, userID)
	if err != nil {
		q.log.Warn("daily quota check: failed to query session count", zap.Error(err))
		return nil
	}
	if sessions >= cfg.MaxSessionsPerDay {
		dailyQuotaRejected.WithLabelValues("sessions").Inc()
		return fmt.Errorf("daily session quota exceeded (%d/%d sessions used today)", sessions, cfg.MaxSessionsPerDay)
	}
	return nil
}

func (q *DailyQuotaChecker) Config() DailyQuotaConfig {
	return q.defaults
}

// PlatformCostBreaker is a global circuit breaker that trips when the
// platform-wide daily AI cost exceeds a configurable threshold (default $50/day).
// When tripped, all system-paid AI calls are rejected until the next UTC midnight.
type PlatformCostBreaker struct {
	mu          sync.Mutex
	threshold   decimal.Decimal
	repo        *repository.AITokenUsageRepository
	provider    ManagedSettingProvider // optional: nil = use env threshold
	log         *zap.Logger
	tripped     bool
	trippedAt   time.Time
	lastChecked time.Time
}

func NewPlatformCostBreaker(repo *repository.AITokenUsageRepository, threshold decimal.Decimal, log *zap.Logger) *PlatformCostBreaker {
	if threshold.LessThanOrEqual(decimal.Zero) {
		threshold = decimal.NewFromInt(50) // default $50/day
	}
	return &PlatformCostBreaker{threshold: threshold, repo: repo, log: log}
}

func (b *PlatformCostBreaker) SetManagedSettingProvider(p ManagedSettingProvider) {
	b.provider = p
}

func (b *PlatformCostBreaker) resolveThreshold(ctx context.Context) decimal.Decimal {
	if b.provider == nil {
		return b.threshold
	}
	if v, _ := b.provider.GetManagedSetting(ctx, managedKeyDailyCostLimit); v != "" {
		if d, err := decimal.NewFromString(v); err == nil && d.IsPositive() {
			return d
		}
	}
	return b.threshold
}

// IsTripped checks whether the platform daily cost has exceeded the threshold.
// Caches the result for 30 seconds to avoid hammering the DB on every AI call.
func (b *PlatformCostBreaker) IsTripped(ctx context.Context) bool {
	if b.repo == nil {
		return false // fail-open when no DB
	}
	b.mu.Lock()
	if b.tripped && time.Since(b.trippedAt) < 30*time.Second {
		b.mu.Unlock()
		return true
	}
	if !b.tripped && time.Since(b.lastChecked) < 30*time.Second {
		b.mu.Unlock()
		return false
	}
	b.mu.Unlock()

	costStr, err := b.repo.DailyPlatformCost(ctx)
	if err != nil {
		b.log.Warn("platform cost breaker: failed to query daily cost", zap.Error(err))
		return false
	}
	cost, err := decimal.NewFromString(costStr)
	if err != nil {
		b.log.Warn("platform cost breaker: failed to parse cost", zap.String("cost", costStr), zap.Error(err))
		return false
	}

	threshold := b.resolveThreshold(ctx)
	exceeded := cost.GreaterThanOrEqual(threshold)

	b.mu.Lock()
	b.lastChecked = time.Now()
	if exceeded {
		if !b.tripped {
			b.tripped = true
			b.trippedAt = time.Now()
			platformCostBreakerTripped.WithLabelValues().Inc()
			platformCostBreakerActive.Set(1)
			b.log.Error("platform cost breaker TRIPPED — system-paid AI blocked, BYO-key users unaffected",
				zap.String("daily_cost", costStr),
				zap.String("threshold", threshold.String()),
			)
		}
	} else {
		if b.tripped {
			platformCostBreakerActive.Set(0)
		}
		b.tripped = false
	}
	b.mu.Unlock()

	return exceeded
}

func (b *PlatformCostBreaker) Threshold() decimal.Decimal {
	return b.threshold
}
