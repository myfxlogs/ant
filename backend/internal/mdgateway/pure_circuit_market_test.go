package mdgateway

import (
	"testing"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"

	"github.com/shopspring/decimal"
)

func TestStateString_Unknown(t *testing.T) {
	t.Parallel()
	if got := State(99).String(); got != "unknown" {
		t.Errorf("State(99).String() = %q, want \"unknown\"", got)
	}
}

func TestCircuitBreaker_OpenBlocks(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(1, 2, time.Hour)
	cb.Allow()
	cb.OnFailure()
	if cb.Allow() {
		t.Error("open circuit should deny calls")
	}
	if cb.State() != StateOpen {
		t.Error("circuit should be open")
	}
}

func TestCircuitBreaker_HalfOpenTransition(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(1, 2, time.Millisecond)
	cb.Allow()
	cb.OnFailure()
	time.Sleep(10 * time.Millisecond)
	if !cb.Allow() {
		t.Error("half-open should allow one probe call")
	}
	if cb.State() != StateHalfOpen {
		t.Error("should be half-open")
	}
}

func TestCircuitBreaker_HalfOpenToClosed(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(1, 1, time.Millisecond)
	cb.Allow()
	cb.OnFailure()
	time.Sleep(10 * time.Millisecond)
	cb.Allow()
	cb.OnSuccess()
	if cb.State() != StateClosed {
		t.Errorf("should transition to closed, got %v", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenToOpen(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(1, 1, time.Millisecond)
	cb.Allow()
	cb.OnFailure()
	time.Sleep(10 * time.Millisecond)
	cb.Allow()
	cb.OnFailure()
	if cb.State() != StateOpen {
		t.Errorf("should stay open after failed probe, got %v", cb.State())
	}
}

// --- market_state.go ---

func TestDefaultMarketStateConfig(t *testing.T) {
	t.Parallel()
	cfg := DefaultMarketStateConfig()
	if cfg.MaxQuoteAgeMs <= 0 {
		t.Errorf("MaxQuoteAgeMs = %d, want >0", cfg.MaxQuoteAgeMs)
	}
}

func TestMarketState_GetAll(t *testing.T) {
	t.Parallel()
	ms := NewMarketStateTracker(DefaultMarketStateConfig())
	ms.Update(&mdtick.Tick{Broker: "broker", Canonical: "EURUSD", TsUnixMs: time.Now().UnixMilli()})
	if len(ms.All()) == 0 {
		t.Error("All should return non-empty slice")
	}
	if ms.Get("broker", "EURUSD") == nil {
		t.Error("Get should return non-nil state")
	}
}

func TestMarketState_RefreshAges(t *testing.T) {
	t.Parallel()
	ms := NewMarketStateTracker(DefaultMarketStateConfig())
	ms.Update(&mdtick.Tick{Broker: "broker", Canonical: "EURUSD", TsUnixMs: time.Now().UnixMilli()})
	ms.RefreshAges(time.Now())
}

// --- user_metrics_flusher.go ---

func TestEvaluateTradeable_Stale(t *testing.T) {
	t.Parallel()
	ms := NewMarketStateTracker(DefaultMarketStateConfig())
	state := &MarketState{QuoteAgeMs: 100_000} // very stale
	if ms.evaluateTradeable(state) {
		t.Error("stale quote should not be tradeable")
	}
}

// --- normalizer_invalidator.go tickerLoop ---

func TestMarketState_UpdateStale(t *testing.T) {
	t.Parallel()
	ms := NewMarketStateTracker(DefaultMarketStateConfig())
	now := time.Now()
	ms.Update(&mdtick.Tick{
		Broker: "broker", Canonical: "EURUSD",
		TsUnixMs: now.UnixMilli() - 100_000, ArrivedUnixMs: now.UnixMilli(),
		Bid: decimal.NewFromFloat(1.1000), Ask: decimal.NewFromFloat(1.1001),
	})
	ms.RefreshAges(now)
}
