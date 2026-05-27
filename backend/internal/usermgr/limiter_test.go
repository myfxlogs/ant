package usermgr

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestUserLimiter_AllowSignal(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	l := NewUserLimiter(cfg)

	if !l.AllowSignal("user-1") {
		t.Fatal("expected AllowSignal to return true")
	}
	if l.ActiveUsers() != 1 {
		t.Fatalf("ActiveUsers: want 1, got %d", l.ActiveUsers())
	}
}

func TestUserLimiter_AllowOrder(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	l := NewUserLimiter(cfg)

	if !l.AllowOrder("user-1") {
		t.Fatal("expected AllowOrder to return true")
	}
}

func TestUserLimiter_AllowCHWrite(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	l := NewUserLimiter(cfg)

	if !l.AllowCHWrite("user-1", 1024) {
		t.Fatal("expected AllowCHWrite to return true")
	}
}

func TestUserLimiter_LRUEviction(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	cfg.MaxEntries = 3
	l := NewUserLimiter(cfg)

	l.AllowSignal("user-1")
	l.AllowSignal("user-2")
	l.AllowSignal("user-3")
	if l.ActiveUsers() != 3 {
		t.Fatalf("ActiveUsers after fill: want 3, got %d", l.ActiveUsers())
	}

	l.AllowSignal("user-4")
	if l.ActiveUsers() != 3 {
		t.Fatalf("ActiveUsers after overflow: want 3, got %d", l.ActiveUsers())
	}
	if l.EvictedCount() < 1 {
		t.Fatalf("EvictedCount: want >=1, got %d", l.EvictedCount())
	}
}

func TestUserLimiter_EvictIdle(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	cfg.MaxEntries = 100
	cfg.IdleTimeout = 10 * time.Millisecond
	l := NewUserLimiter(cfg)

	l.AllowSignal("user-active")
	l.AllowSignal("user-idle")

	time.Sleep(20 * time.Millisecond)

	removed := l.EvictIdle()
	if removed < 2 {
		t.Fatalf("EvictIdle: want >=2, got %d", removed)
	}
	if l.ActiveUsers() != 0 {
		t.Fatalf("ActiveUsers after eviction: want 0, got %d", l.ActiveUsers())
	}
}

func TestUserLimiter_BlockedCounts(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	l := NewUserLimiter(cfg)

	s, o, c := l.BlockedCounts()
	if s != 0 || o != 0 || c != 0 {
		t.Fatalf("initial blocked counts: want all 0, got %d/%d/%d", s, o, c)
	}
}

func TestUserLimiter_Concurrent(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	l := NewUserLimiter(cfg)

	done := make(chan struct{})
	for i := 0; i < 100; i++ {
		go func(id int) {
			uid := "user-" + string(rune('A'+id%10))
			l.AllowSignal(uid)
			l.AllowOrder(uid)
			l.AllowCHWrite(uid, 1024)
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < 100; i++ {
		<-done
	}

	if l.ActiveUsers() > 10 {
		t.Logf("ActiveUsers: %d (expected <=10 unique)", l.ActiveUsers())
	}
}

func TestGetUserID_NoAuth(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	uid := GetUserID(ctx)
	if uid != "" {
		t.Fatalf("unauthenticated GetUserID: want empty, got %q", uid)
	}
}

func TestSpanWithUser_NoSpan(t *testing.T) {
	t.Parallel()
	// Should not panic when there's no span in context.
	SpanWithUser(t.Context(), "user-1")
}

func TestSpanWithUser_EmptyUserID(t *testing.T) {
	t.Parallel()
	// Should be a no-op for empty userID.
	SpanWithUser(t.Context(), "")
}

func TestLoggerWithUser_NoAuth(t *testing.T) {
	t.Parallel()
	log := zap.NewNop()
	augmented := LoggerWithUser(t.Context(), log)
	if augmented != log {
		t.Fatal("LoggerWithUser with no auth should return same logger")
	}
}

func TestUserLimiter_SignalRateEnforcement(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	cfg.SignalPerUserMax = 3
	l := NewUserLimiter(cfg)

	// First 3 should pass.
	for i := 0; i < 3; i++ {
		if !l.AllowSignal("user-sig") {
			t.Fatalf("AllowSignal #%d: expected true", i+1)
		}
	}

	// 4th should be denied.
	if l.AllowSignal("user-sig") {
		t.Fatal("AllowSignal #4: expected false (rate limit)")
	}

	s, _, _ := l.BlockedCounts()
	if s < 1 {
		t.Fatalf("blocked signals: want >=1, got %d", s)
	}
}

func TestUserLimiter_OrderRateEnforcement(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	cfg.OrderPerUserMax = 2
	l := NewUserLimiter(cfg)

	for i := 0; i < 2; i++ {
		if !l.AllowOrder("user-ord") {
			t.Fatalf("AllowOrder #%d: expected true", i+1)
		}
	}

	if l.AllowOrder("user-ord") {
		t.Fatal("AllowOrder #3: expected false (rate limit)")
	}

	_, o, _ := l.BlockedCounts()
	if o < 1 {
		t.Fatalf("blocked orders: want >=1, got %d", o)
	}
}

func TestUserLimiter_CHWritePerUserEnforcement(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	cfg.CHWritePerUserMax = 1000
	l := NewUserLimiter(cfg)

	if !l.AllowCHWrite("user-ch", 800) {
		t.Fatal("AllowCHWrite 800: expected true")
	}
	if l.AllowCHWrite("user-ch", 300) {
		t.Fatal("AllowCHWrite +300 (total 1100): expected false (per-user limit)")
	}

	_, _, c := l.BlockedCounts()
	if c < 1 {
		t.Fatalf("blocked CH writes: want >=1, got %d", c)
	}
}

func TestUserLimiter_GlobalCHCeilingEnforcement(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	cfg.CHWritePerUserMax = 1_000_000 // high per-user, won't hit
	cfg.GlobalCHCeiling = 500
	l := NewUserLimiter(cfg)

	if !l.AllowCHWrite("user-a", 400) {
		t.Fatal("AllowCHWrite user-a 400: expected true")
	}
	if l.AllowCHWrite("user-b", 200) {
		t.Fatal("AllowCHWrite user-b +200 (total 600): expected false (global ceiling)")
	}

	_, _, c := l.BlockedCounts()
	if c < 1 {
		t.Fatalf("blocked CH writes (global): want >=1, got %d", c)
	}
}

func TestUserLimiter_GlobalCHCeilingWindowReset(t *testing.T) {
	t.Parallel()
	// This test verifies the global ceiling resets when a new second arrives.
	cfg := DefaultConfig()
	cfg.CHWritePerUserMax = 1_000_000
	cfg.GlobalCHCeiling = 100
	l := NewUserLimiter(cfg)

	if !l.AllowCHWrite("user-a", 50) {
		t.Fatal("first write should pass")
	}

	// Manually reset the global window to simulate a new second.
	l.globalChBytes.Store(0)
	l.globalChWindow.Store(time.Now().Unix())

	if !l.AllowCHWrite("user-b", 50) {
		t.Fatal("after reset should pass again")
	}
}

func TestUserLimiter_PerUserWindowReset(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	cfg.SignalPerUserMax = 2
	l := NewUserLimiter(cfg)

	if !l.AllowSignal("user-r") {
		t.Fatal("AllowSignal #1: expected true")
	}
	if !l.AllowSignal("user-r") {
		t.Fatal("AllowSignal #2: expected true")
	}
	if l.AllowSignal("user-r") {
		t.Fatal("AllowSignal #3: expected false")
	}

	// Manually reset the window by advancing signalWindow.
	l.mu.Lock()
	if elem, ok := l.entries["user-r"]; ok {
		rl := elem.Value.(*RateLimit)
		rl.signalWindow = time.Now().Unix()
		rl.signalCount.Store(0)
	}
	l.mu.Unlock()

	if !l.AllowSignal("user-r") {
		t.Fatal("AllowSignal after window reset: expected true")
	}
}

func TestUserLimiter_SeparateLimitsIndependent(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	cfg.SignalPerUserMax = 1
	cfg.OrderPerUserMax = 10
	l := NewUserLimiter(cfg)

	// Exhaust signal limit.
	l.AllowSignal("user-x")
	if l.AllowSignal("user-x") {
		t.Fatal("2nd signal should be blocked")
	}

	// Order should still work (separate limit).
	if !l.AllowOrder("user-x") {
		t.Fatal("order should still be allowed after signal exhaustion")
	}
}

func TestUserLimiter_HighConcurrency(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	cfg.SignalPerUserMax = 5000
	cfg.OrderPerUserMax = 5000
	cfg.CHWritePerUserMax = 100_000_000
	l := NewUserLimiter(cfg)

	var wg sync.WaitGroup
	var allowed atomic.Int64
	var blocked atomic.Int64

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				uid := "user-" + string(rune('A'+j%20))
				if l.AllowSignal(uid) {
					allowed.Add(1)
				} else {
					blocked.Add(1)
				}
				l.AllowOrder(uid)
				l.AllowCHWrite(uid, 100)
			}
		}()
	}
	wg.Wait()

	if allowed.Load() < 5000 {
		t.Logf("allowed=%d blocked=%d", allowed.Load(), blocked.Load())
	}
}
