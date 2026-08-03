package service

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// mockSettingProvider is a test ManagedSettingProvider.
type mockSettingProvider struct {
	values map[string]string
}

func (m *mockSettingProvider) GetManagedSetting(_ context.Context, key string) (string, error) {
	return m.values[key], nil
}

func TestDailyQuotaConfig_Defaults(t *testing.T) {
	dq := NewDailyQuotaChecker(nil, DailyQuotaConfig{}, zap.NewNop())
	cfg := dq.Config()
	if cfg.MaxSessionsPerDay != 5 {
		t.Errorf("default MaxSessionsPerDay = %d, want 5", cfg.MaxSessionsPerDay)
	}
	if cfg.MaxTokensPerDay != 200_000 {
		t.Errorf("default MaxTokensPerDay = %d, want 200000", cfg.MaxTokensPerDay)
	}
}

func TestDailyQuotaConfig_Custom(t *testing.T) {
	dq := NewDailyQuotaChecker(nil, DailyQuotaConfig{
		MaxSessionsPerDay: 10,
		MaxTokensPerDay:   500_000,
	}, zap.NewNop())
	cfg := dq.Config()
	if cfg.MaxSessionsPerDay != 10 {
		t.Errorf("MaxSessionsPerDay = %d, want 10", cfg.MaxSessionsPerDay)
	}
	if cfg.MaxTokensPerDay != 500_000 {
		t.Errorf("MaxTokensPerDay = %d, want 500000", cfg.MaxTokensPerDay)
	}
}

func TestDailyQuotaChecker_ManagedSettingOverride(t *testing.T) {
	dq := NewDailyQuotaChecker(nil, DailyQuotaConfig{
		MaxSessionsPerDay: 5,
		MaxTokensPerDay:   200_000,
	}, zap.NewNop())
	dq.SetManagedSettingProvider(&mockSettingProvider{
		values: map[string]string{
			"ai_daily_max_sessions": "20",
			"ai_daily_max_tokens":   "999999",
		},
	})
	cfg := dq.resolveConfig(context.Background())
	if cfg.MaxSessionsPerDay != 20 {
		t.Errorf("managed override MaxSessionsPerDay = %d, want 20", cfg.MaxSessionsPerDay)
	}
	if cfg.MaxTokensPerDay != 999999 {
		t.Errorf("managed override MaxTokensPerDay = %d, want 999999", cfg.MaxTokensPerDay)
	}
}

func TestDailyQuotaChecker_ManagedSettingFallback(t *testing.T) {
	dq := NewDailyQuotaChecker(nil, DailyQuotaConfig{
		MaxSessionsPerDay: 5,
		MaxTokensPerDay:   200_000,
	}, zap.NewNop())
	// Provider returns empty values — should fall back to env defaults.
	dq.SetManagedSettingProvider(&mockSettingProvider{values: map[string]string{}})
	cfg := dq.resolveConfig(context.Background())
	if cfg.MaxSessionsPerDay != 5 {
		t.Errorf("fallback MaxSessionsPerDay = %d, want 5", cfg.MaxSessionsPerDay)
	}
	if cfg.MaxTokensPerDay != 200_000 {
		t.Errorf("fallback MaxTokensPerDay = %d, want 200000", cfg.MaxTokensPerDay)
	}
}

func TestPlatformCostBreaker_DefaultThreshold(t *testing.T) {
	b := NewPlatformCostBreaker(nil, decimal.Zero, zap.NewNop())
	if !b.Threshold().Equal(decimal.NewFromInt(50)) {
		t.Errorf("default threshold = %s, want 50", b.Threshold())
	}
}

func TestPlatformCostBreaker_CustomThreshold(t *testing.T) {
	b := NewPlatformCostBreaker(nil, decimal.NewFromInt(100), zap.NewNop())
	if !b.Threshold().Equal(decimal.NewFromInt(100)) {
		t.Errorf("threshold = %s, want 100", b.Threshold())
	}
}

func TestPlatformCostBreaker_NilRepoFailsOpen(t *testing.T) {
	b := NewPlatformCostBreaker(nil, decimal.NewFromInt(50), zap.NewNop())
	if b.IsTripped(nil) {
		t.Error("IsTripped with nil repo should return false (fail-open)")
	}
}

func TestPlatformCostBreaker_ManagedSettingOverride(t *testing.T) {
	b := NewPlatformCostBreaker(nil, decimal.NewFromInt(50), zap.NewNop())
	b.SetManagedSettingProvider(&mockSettingProvider{
		values: map[string]string{
			"ai_daily_cost_limit_usd": "100",
		},
	})
	threshold := b.resolveThreshold(context.Background())
	if !threshold.Equal(decimal.NewFromInt(100)) {
		t.Errorf("managed override threshold = %s, want 100", threshold)
	}
}

func TestPlatformCostBreaker_ManagedSettingFallback(t *testing.T) {
	b := NewPlatformCostBreaker(nil, decimal.NewFromInt(50), zap.NewNop())
	b.SetManagedSettingProvider(&mockSettingProvider{values: map[string]string{}})
	threshold := b.resolveThreshold(context.Background())
	if !threshold.Equal(decimal.NewFromInt(50)) {
		t.Errorf("fallback threshold = %s, want 50", threshold)
	}
}
