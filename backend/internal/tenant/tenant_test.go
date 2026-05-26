package tenant

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestWithTenantID(t *testing.T) {
	ctx := WithTenantID(t.Context(), "t-001")
	if got := GetTenantID(ctx); got != "t-001" {
		t.Fatalf("GetTenantID: want t-001, got %q", got)
	}
}

func TestGetTenantID_Empty(t *testing.T) {
	if got := GetTenantID(t.Context()); got != "" {
		t.Fatalf("unset tenant_id: want empty, got %q", got)
	}
}

func TestLoggerWithTenant(t *testing.T) {
	log := zap.NewNop()
	ctx := WithTenantID(t.Context(), "t-001")
	augmented := LoggerWithTenant(ctx, log)
	if augmented == log {
		t.Fatal("LoggerWithTenant should wrap logger when tenant_id present")
	}
}

func TestLoggerWithTenant_Empty(t *testing.T) {
	log := zap.NewNop()
	augmented := LoggerWithTenant(t.Context(), log)
	if augmented != log {
		t.Fatal("LoggerWithTenant with no tenant should return same logger")
	}
}

func TestSpanWithTenant_NoSpan(t *testing.T) {
	// Should not panic.
	SpanWithTenant(t.Context(), "t-001")
}

func TestSpanWithTenant_EmptyID(t *testing.T) {
	SpanWithTenant(t.Context(), "")
}

func TestConfig_IsValid(t *testing.T) {
	if (Config{}).IsValid() {
		t.Fatal("empty config should be invalid")
	}
	if !DefaultConfig("t-001").IsValid() {
		t.Fatal("DefaultConfig should be valid")
	}
}

func TestLimiter_AllowSignal(t *testing.T) {
	l := NewLimiter()
	cfg := DefaultConfig("t-001")
	cfg.SignalMaxPerSec = 3

	for i := 0; i < 3; i++ {
		if !l.AllowSignal("t-001", cfg) {
			t.Fatalf("AllowSignal #%d: expected true", i+1)
		}
	}
	if l.AllowSignal("t-001", cfg) {
		t.Fatal("AllowSignal #4: expected false (rate limit)")
	}

	s, _, _ := l.BlockedCounts()
	if s < 1 {
		t.Fatalf("blocked signals: want >=1, got %d", s)
	}
}

func TestLimiter_AllowOrder(t *testing.T) {
	l := NewLimiter()
	cfg := DefaultConfig("t-001")
	cfg.OrderMaxPerSec = 2

	for i := 0; i < 2; i++ {
		if !l.AllowOrder("t-001", cfg) {
			t.Fatalf("AllowOrder #%d: expected true", i+1)
		}
	}
	if l.AllowOrder("t-001", cfg) {
		t.Fatal("AllowOrder #3: expected false (rate limit)")
	}
}

func TestLimiter_AllowCHWrite(t *testing.T) {
	l := NewLimiter()
	cfg := DefaultConfig("t-001")
	cfg.CHWriteBytesPerSec = 1000

	if !l.AllowCHWrite("t-001", cfg, 800) {
		t.Fatal("AllowCHWrite 800: expected true")
	}
	if l.AllowCHWrite("t-001", cfg, 300) {
		t.Fatal("AllowCHWrite +300 (total 1100): expected false")
	}
}

func TestLimiter_TenantIsolation(t *testing.T) {
	l := NewLimiter()
	cfg1 := DefaultConfig("t-001")
	cfg1.SignalMaxPerSec = 3
	cfg2 := DefaultConfig("t-002")
	cfg2.SignalMaxPerSec = 1

	// Exhaust t-002 signal limit.
	l.AllowSignal("t-002", cfg2)
	if l.AllowSignal("t-002", cfg2) {
		t.Fatal("t-002 2nd signal should be blocked")
	}

	// t-001 should still work (separate tenant).
	if !l.AllowSignal("t-001", cfg1) {
		t.Fatal("t-001 should still be allowed after t-002 exhaustion")
	}
}

func TestLimiter_UpsertConfig(t *testing.T) {
	l := NewLimiter()
	cfg := DefaultConfig("t-001")
	cfg.SignalMaxPerSec = 2

	l.UpsertConfig(cfg)
	if l.ActiveTenants() != 1 {
		t.Fatalf("ActiveTenants: want 1, got %d", l.ActiveTenants())
	}

	// Update with higher limit.
	cfg.SignalMaxPerSec = 100
	l.UpsertConfig(cfg)
	if l.ActiveTenants() != 1 {
		t.Fatalf("ActiveTenants after upsert: want 1, got %d", l.ActiveTenants())
	}
}

func TestLimiter_LRUEviction(t *testing.T) {
	l := NewLimiter()
	l.maxEntries = 3

	for i := 0; i < 5; i++ {
		tid := "t-" + string(rune('0'+i))
		cfg := DefaultConfig(tid)
		l.AllowSignal(tid, cfg)
	}

	if l.ActiveTenants() > 3 {
		t.Fatalf("ActiveTenants: want <=3, got %d", l.ActiveTenants())
	}
	if l.EvictedCount() < 2 {
		t.Fatalf("EvictedCount: want >=2, got %d", l.EvictedCount())
	}
}

func TestLimiter_EvictIdle(t *testing.T) {
	l := NewLimiter()
	l.idleTimeout = 10 * time.Millisecond

	cfg := DefaultConfig("t-001")
	l.AllowSignal("t-001", cfg)

	time.Sleep(20 * time.Millisecond)

	removed := l.EvictIdle()
	if removed < 1 {
		t.Fatalf("EvictIdle: want >=1, got %d", removed)
	}
}

func TestLimiter_Concurrent(t *testing.T) {
	l := NewLimiter()
	cfg := DefaultConfig("t-shared")
	cfg.SignalMaxPerSec = 5000
	cfg.OrderMaxPerSec = 5000
	cfg.CHWriteBytesPerSec = 100_000_000

	var wg sync.WaitGroup
	var allowed atomic.Int64
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if l.AllowSignal("t-shared", cfg) {
					allowed.Add(1)
				}
				l.AllowOrder("t-shared", cfg)
				l.AllowCHWrite("t-shared", cfg, 100)
			}
		}()
	}
	wg.Wait()

	if allowed.Load() < 5000 {
		t.Logf("allowed=%d (expected ~10000 minus rate-limited rejections)", allowed.Load())
	}
	if l.ActiveTenants() != 1 {
		t.Fatalf("ActiveTenants: want 1, got %d", l.ActiveTenants())
	}
}
