package marketplace

import (
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func TestQualityViolation_String(t *testing.T) {
	t.Parallel()
	v := QualityViolation{Metric: "sharpe_ratio", Actual: "0.5", Threshold: "1.0"}
	s := v.String()
	if s == "" {
		t.Fatal("expected non-empty string")
	}
}

func TestNew(t *testing.T) {
	t.Parallel()
	s := New(nil, nil, zap.NewNop())
	if s == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestService_Setters(t *testing.T) {
	t.Parallel()
	s := &Service{}
	s.SetLivePerfCollector(nil)
	s.SetWalletRepo(nil)
	s.SetNotificationSender(nil)
	s.SetOptimizer(nil)
}

func TestIsUniqueViolation_True(t *testing.T) {
	t.Parallel()
	err := &pgconn.PgError{Code: "23505"}
	if !isUniqueViolation(err) {
		t.Fatal("expected true for 23505")
	}
}

func TestIsUniqueViolation_False(t *testing.T) {
	t.Parallel()
	if isUniqueViolation(errors.New("some error")) {
		t.Fatal("expected false for generic error")
	}
}

func TestIsUniqueViolation_Wrapped(t *testing.T) {
	t.Parallel()
	pgErr := &pgconn.PgError{Code: "23505"}
	wrapped := errors.Join(pgErr, errors.New("context"))
	if !isUniqueViolation(wrapped) {
		t.Fatal("expected true for wrapped pg error")
	}
}

func TestPublishedCache_Key(t *testing.T) {
	t.Parallel()
	c := newPublishedCache()
	k := c.key("user-1", "forex", "EURUSD", "sharpe", "all", 10, 0)
	if k == "" {
		t.Fatal("expected non-empty key")
	}
}

func TestPublishedCache_SetGet(t *testing.T) {
	t.Parallel()
	c := newPublishedCache()
	data := []PublishedStrategy{{StrategyID: "strat-1"}}
	c.set("key-1", data)
	got, ok := c.get("key-1")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if len(got) != 1 || got[0].StrategyID != "strat-1" {
		t.Fatalf("expected strat-1, got %v", got)
	}
}

func TestPublishedCache_GetMiss(t *testing.T) {
	t.Parallel()
	c := newPublishedCache()
	_, ok := c.get("nonexistent")
	if ok {
		t.Fatal("expected cache miss")
	}
}

func TestPublishedCache_GetExpired(t *testing.T) {
	t.Parallel()
	c := newPublishedCache()
	// Manually insert expired entry
	c.m["expired"] = publishedCacheEntry{
		data:      []PublishedStrategy{{StrategyID: "old"}},
		expiresAt: time.Now().Add(-time.Hour),
	}
	_, ok := c.get("expired")
	if ok {
		t.Fatal("expected cache miss for expired entry")
	}
}

func TestPublishedCache_Clear(t *testing.T) {
	t.Parallel()
	c := newPublishedCache()
	c.set("key-1", []PublishedStrategy{{StrategyID: "s1"}})
	c.set("key-2", []PublishedStrategy{{StrategyID: "s2"}})
	c.clear()
	_, ok1 := c.get("key-1")
	_, ok2 := c.get("key-2")
	if ok1 || ok2 {
		t.Fatal("expected all entries cleared")
	}
}

func TestPublishedCache_EvictionOnOverflow(t *testing.T) {
	t.Parallel()
	c := newPublishedCache()
	// Fill beyond 256 to trigger eviction
	for i := 0; i < 260; i++ {
		c.set(string(rune(i)), []PublishedStrategy{{StrategyID: "s"}})
	}
	// Should not panic, and should have evicted some expired entries
}

func TestComputeCouponDiscount_ZeroFinalAmount(t *testing.T) {
	t.Parallel()
	row := CouponRow{
		Enabled:       true,
		DiscountType:  "fixed",
		DiscountValue: dec(100),
	}
	result, _ := computeCouponDiscount(row, dec(100))
	if !result.FinalAmount.IsZero() {
		t.Fatalf("expected zero final amount, got %s", result.FinalAmount.String())
	}
}

func TestComputeCouponDiscount_NegativeFinalClampedToZero(t *testing.T) {
	t.Parallel()
	row := CouponRow{
		Enabled:       true,
		DiscountType:  "fixed",
		DiscountValue: dec(150),
	}
	result, _ := computeCouponDiscount(row, dec(100))
	if result.FinalAmount.LessThan(decimal.Zero) {
		t.Fatalf("expected non-negative final amount, got %s", result.FinalAmount.String())
	}
}
