package risk

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestMaxLotSize_Allow(t *testing.T) {
	r := &MaxLotSize{MaxLots: decimal.NewFromInt(10)}
	result := r.Check(context.Background(), intentBuy("5.0"), nil)
	if !result.Allowed {
		t.Errorf("expected allowed, got: %s", result.Reason)
	}
}

func TestMaxLotSize_Block(t *testing.T) {
	r := &MaxLotSize{MaxLots: decimal.NewFromInt(10)}
	result := r.Check(context.Background(), intentBuy("15.0"), nil)
	if result.Allowed {
		t.Error("expected blocked for 15 lots")
	}
}

func TestMaxLotSize_SuggestsAdjustment(t *testing.T) {
	r := &MaxLotSize{MaxLots: decimal.NewFromInt(10)}
	result := r.Check(context.Background(), intentBuy("50.0"), nil)
	if result.AdjustedVolume.IsZero() {
		t.Error("expected adjusted volume suggestion")
	}
}

// ── R2: Max Position Count ────────────────────────────────────────────

func TestMaxPositionCount_Allow(t *testing.T) {
	r := &MaxPositionCount{Max: 20}
	state := &AccountState{OpenPositions: 5}
	result := r.Check(context.Background(), intentBuy("0.1"), state)
	if !result.Allowed {
		t.Errorf("expected allowed, got: %s", result.Reason)
	}
}

func TestMaxPositionCount_Block(t *testing.T) {
	r := &MaxPositionCount{Max: 20}
	state := &AccountState{OpenPositions: 20}
	result := r.Check(context.Background(), intentBuy("0.1"), state)
	if result.Allowed {
		t.Error("expected blocked when at max positions")
	}
}

// ── R3: Max Exposure ──────────────────────────────────────────────────

func TestMaxExposure_Allow(t *testing.T) {
	r := &MaxExposure{MaxRatio: decimal.NewFromFloat(0.5)}
	// 0.01 lots × 1.08500 × 100000 = 1085. 1085 < 5000 (50% of 10000) → allowed.
	state := &AccountState{Balance: decimal.NewFromInt(10000), ContractSize: decimal.NewFromInt(100000)}
	result := r.Check(context.Background(), intentBuy("0.01"), state)
	if !result.Allowed {
		t.Errorf("expected allowed, got: %s", result.Reason)
	}
}

func TestMaxExposure_Block(t *testing.T) {
	r := &MaxExposure{MaxRatio: decimal.NewFromFloat(0.5)}
	state := &AccountState{Balance: decimal.NewFromInt(10000), ContractSize: decimal.NewFromInt(100000)}
	result := r.Check(context.Background(), intentBuy("100.0"), state)
	if result.Allowed {
		t.Error("expected blocked for 100 lots")
	}
}

// ── R4a: Daily Loss Breaker ───────────────────────────────────────────

func TestDailyLoss_Allow(t *testing.T) {
	r := &DailyLossBreaker{MaxDailyLoss: decimal.NewFromInt(500)}
	state := &AccountState{DailyPnL: decimal.NewFromInt(-100)}
	result := r.Check(context.Background(), intentBuy("0.1"), state)
	if !result.Allowed {
		t.Errorf("expected allowed, got: %s", result.Reason)
	}
}

func TestDailyLoss_Block(t *testing.T) {
	r := &DailyLossBreaker{MaxDailyLoss: decimal.NewFromInt(500)}
	state := &AccountState{DailyPnL: decimal.NewFromInt(-600)}
	result := r.Check(context.Background(), intentBuy("0.1"), state)
	if result.Allowed {
		t.Error("expected blocked when daily loss exceeds limit")
	}
}

func TestDailyLoss_Disabled(t *testing.T) {
	r := &DailyLossBreaker{MaxDailyLoss: decimal.Zero}
	state := &AccountState{DailyPnL: decimal.NewFromInt(-99999)}
	result := r.Check(context.Background(), intentBuy("0.1"), state)
	if !result.Allowed {
		t.Errorf("expected allowed when disabled, got: %s", result.Reason)
	}
}

// ── R4b: Drawdown Breaker ─────────────────────────────────────────────

func TestDrawdown_Allow(t *testing.T) {
	r := &DrawdownBreaker{MaxDrawdownPct: decimal.NewFromFloat(0.30)}
	state := &AccountState{Equity: decimal.NewFromInt(9000), PeakEquity: decimal.NewFromInt(10000)}
	result := r.Check(context.Background(), intentBuy("0.1"), state)
	if !result.Allowed {
		t.Errorf("expected allowed (10%% DD), got: %s", result.Reason)
	}
}

func TestDrawdown_Block(t *testing.T) {
	r := &DrawdownBreaker{MaxDrawdownPct: decimal.NewFromFloat(0.30)}
	state := &AccountState{Equity: decimal.NewFromInt(6000), PeakEquity: decimal.NewFromInt(10000)}
	result := r.Check(context.Background(), intentBuy("0.1"), state)
	if result.Allowed {
		t.Error("expected blocked at 40% drawdown")
	}
}

// ── R5: Symbol Whitelist ──────────────────────────────────────────────

func TestSymbolWhitelist_Allow(t *testing.T) {
	r := &SymbolWhitelist{Whitelist: []string{"EURUSD", "GBPUSD"}}
	result := r.Check(context.Background(), intentBuy("0.1"), nil)
	if !result.Allowed {
		t.Errorf("expected allowed, got: %s", result.Reason)
	}
}

func TestSymbolWhitelist_Block(t *testing.T) {
	r := &SymbolWhitelist{Whitelist: []string{"GBPUSD"}}
	result := r.Check(context.Background(), intentBuy("0.1"), nil)
	if result.Allowed {
		t.Error("expected blocked — EURUSD not in whitelist")
	}
}

func TestSymbolWhitelist_EmptyAllowsAll(t *testing.T) {
	r := &SymbolWhitelist{}
	result := r.Check(context.Background(), intentBuy("0.1"), nil)
	if !result.Allowed {
		t.Errorf("empty whitelist should allow all")
	}
}

// ── R6: Leverage Cap ──────────────────────────────────────────────────

func TestLeverageCap_Allow(t *testing.T) {
	r := &LeverageCap{MaxLeverage: 500}
	state := &AccountState{SymbolLeverage: 100}
	result := r.Check(context.Background(), intentBuy("0.1"), state)
	if !result.Allowed {
		t.Errorf("expected allowed, got: %s", result.Reason)
	}
}

func TestLeverageCap_Block(t *testing.T) {
	r := &LeverageCap{MaxLeverage: 500}
	state := &AccountState{SymbolLeverage: 1000}
	result := r.Check(context.Background(), intentBuy("0.1"), state)
	if result.Allowed {
		t.Error("expected blocked for 1000x leverage")
	}
}

// ── R7: Order Frequency Limit ─────────────────────────────────────────

func TestOrderFrequency_Allow(t *testing.T) {
	r := &OrderFrequencyLimit{MaxOrders: 60, Window: time.Minute}
	result := r.Check(context.Background(), intentBuy("0.1"), nil)
	if !result.Allowed {
		t.Errorf("expected allowed, got: %s", result.Reason)
	}
}

func TestOrderFrequency_Block(t *testing.T) {
	r := &OrderFrequencyLimit{MaxOrders: 2, Window: time.Hour}
	r.Check(context.Background(), intentBuy("0.1"), nil)
	r.Check(context.Background(), intentBuy("0.1"), nil)
	result := r.Check(context.Background(), intentBuy("0.1"), nil)
	if result.Allowed {
		t.Error("expected blocked after 2 orders")
	}
}

// ── R8: Duplicate Protection ──────────────────────────────────────────
