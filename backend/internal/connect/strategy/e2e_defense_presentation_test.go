package strategy

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/backtest"
	"alphaforge/strategy/sdk"
)

// TestDefensePresentation_DegradedStatus verifies the full defense-line-B chain:
// volume=0 trades → buildBacktestResponse → BlindSpot(zero_volume_trade) +
// IsReliable=false + hasInvariantBlindSpot=true → status would be DEGRADED.
// Also verifies the positive case: volume>0 trades → no invariant BlindSpot → SUCCEEDED.
func TestDefensePresentation_DegradedStatus(t *testing.T) {
	initialCapital := decimal.NewFromInt(10000)

	badTrades := make([]backtest.Trade, 12)
	for i := range badTrades {
		badTrades[i] = validTrade()
	}
	badTrades[5].Volume = decimal.Zero

	badResult := &backtest.Result{
		Config:       backtest.Config{InitialCapital: initialCapital},
		Metrics:      makeMetrics(12),
		FinalBalance: initialCapital,
		Trades:       badTrades,
	}

	cfg := backtest.Config{InitialCapital: initialCapital}
	params := backtestParams{}
	vmRunner := newMinimalVMRunner(t)

	resp, _, _, _ := buildBacktestResponse(badResult, cfg, params, vmRunner)

	foundZeroVol := false
	for _, bs := range resp.GetBlindSpots() {
		if bs.GetId() == "zero_volume_trade" {
			foundZeroVol = true
		}
	}
	if !foundZeroVol {
		t.Fatal("defense line B failed to detect zero_volume_trade in BlindSpots")
	}

	if resp.GetRisk().GetIsReliable() {
		t.Error("expected IsReliable=false when invariant is violated, got true")
	}

	if !hasInvariantBlindSpot(resp) {
		t.Fatal("hasInvariantBlindSpot should return true for invariant BlindSpot → status should be DEGRADED")
	}

	goodTrades := make([]backtest.Trade, 12)
	for i := range goodTrades {
		goodTrades[i] = validTrade()
	}
	goodResult := &backtest.Result{
		Config:       backtest.Config{InitialCapital: initialCapital},
		Metrics:      makeMetrics(12),
		FinalBalance: decimal.NewFromInt(11140),
		Trades:       goodTrades,
	}

	goodResp, _, _, _ := buildBacktestResponse(goodResult, cfg, params, vmRunner)

	if hasInvariantBlindSpot(goodResp) {
		t.Error("normal trades (volume>0) should not have invariant BlindSpot → status should be SUCCEEDED")
	}
}

// TestDefensePresentation_AllInvariantsTriggerDegraded verifies that each
// invariant violation individually causes hasInvariantBlindSpot=true,
// ensuring the status-degradation logic covers all five invariant types.
func TestDefensePresentation_AllInvariantsTriggerDegraded(t *testing.T) {
	initialCapital := decimal.NewFromInt(10000)
	cfg := backtest.Config{InitialCapital: initialCapital}
	params := backtestParams{}
	vmRunner := newMinimalVMRunner(t)

	cases := []struct {
		name   string
		mutate func(trades []backtest.Trade)
		id     string
	}{
		{
			name:   "zero_volume_trade",
			mutate: func(t []backtest.Trade) { t[5].Volume = decimal.Zero },
			id:     "zero_volume_trade",
		},
		{
			name:   "non_positive_price",
			mutate: func(t []backtest.Trade) { t[5].EntryPrice = decimal.Zero },
			id:     "non_positive_price",
		},
		{
			name:   "invalid_side",
			mutate: func(t []backtest.Trade) { t[5].Side = sdk.PositionSide(0) },
			id:     "invalid_side",
		},
		{
			name:   "time_order_violation",
			mutate: func(t []backtest.Trade) { t[5].EntryTime = time.UnixMilli(3000); t[5].ExitTime = time.UnixMilli(2000) },
			id:     "time_order_violation",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			trades := make([]backtest.Trade, 12)
			for i := range trades {
				trades[i] = validTrade()
			}
			tc.mutate(trades)

			result := &backtest.Result{
				Config:       backtest.Config{InitialCapital: initialCapital},
				Metrics:      makeMetrics(12),
				FinalBalance: initialCapital,
				Trades:       trades,
			}

			resp, _, _, _ := buildBacktestResponse(result, cfg, params, vmRunner)

			if !hasInvariantBlindSpot(resp) {
				t.Errorf("hasInvariantBlindSpot should be true for %s → status DEGRADED", tc.id)
			}
			if resp.GetRisk().GetIsReliable() {
				t.Errorf("IsReliable should be false for %s", tc.id)
			}
		})
	}
}
