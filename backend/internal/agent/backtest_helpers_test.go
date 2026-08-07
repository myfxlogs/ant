package agent

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/strategy/backtest"
	"alphaforge/strategy/sdk"
)

// TestBuildBacktestResultProto_InvariantsPopulated verifies that ADR-0028 Defense Line B
// invariant checks are run by buildBacktestResultProto and populate IsReliable + InvariantBlindSpots.
// This is the AGT-2 regression test: agent backtest results must include invariant validation.
func TestBuildBacktestResultProto_InvariantsPopulated(t *testing.T) {
	// Case 1: Valid result — all invariants pass, IsReliable=true, no InvariantBlindSpots.
	validResult := &backtest.Result{
		Config: backtest.Config{
			InitialCapital: decimal.NewFromInt(10000),
		},
		FinalBalance: decimal.NewFromInt(10000),
		Metrics: &antv1.BacktestMetrics{
			TotalReturn:  "0",
			MaxDrawdown:  "0",
			SharpeRatio:  "0",
			WinRate:      "0",
			TotalTrades:  1,
		},
		Trades: []backtest.Trade{
			{
				Side:       sdk.SideBuy,
				EntryTime:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				ExitTime:   time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC),
				EntryPrice: decimal.NewFromFloat(1.1),
				ExitPrice:  decimal.NewFromFloat(1.2),
				Volume:     decimal.NewFromFloat(0.1),
				Profit:     decimal.Zero,
			},
		},
	}

	t.Run("valid_result_is_reliable", func(t *testing.T) {
		proto := buildBacktestResultProto(validResult)
		if !proto.IsReliable {
			t.Fatal("expected IsReliable=true for valid result, got false")
		}
		if len(proto.InvariantBlindSpots) != 0 {
			t.Fatalf("expected 0 InvariantBlindSpots for valid result, got %d", len(proto.InvariantBlindSpots))
		}
	})

	// Case 2: Zero volume trade — volume invariant violated, IsReliable=false.
	zeroVolumeResult := &backtest.Result{
		Config: backtest.Config{
			InitialCapital: decimal.NewFromInt(10000),
		},
		FinalBalance: decimal.NewFromInt(10000),
		Metrics: &antv1.BacktestMetrics{
			TotalReturn:  "0",
			MaxDrawdown:  "0",
			SharpeRatio:  "0",
			WinRate:      "0",
			TotalTrades:  1,
		},
		Trades: []backtest.Trade{
			{
				Side:       sdk.SideBuy,
				EntryTime:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				ExitTime:   time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC),
				EntryPrice: decimal.NewFromFloat(1.1),
				ExitPrice:  decimal.NewFromFloat(1.2),
				Volume:     decimal.Zero, // violation!
				Profit:     decimal.Zero,
			},
		},
	}

	t.Run("zero_volume_is_unreliable", func(t *testing.T) {
		proto := buildBacktestResultProto(zeroVolumeResult)
		if proto.IsReliable {
			t.Fatal("expected IsReliable=false for zero-volume trade, got true")
		}
		if len(proto.InvariantBlindSpots) == 0 {
			t.Fatal("expected at least 1 InvariantBlindSpot for zero-volume trade, got 0")
		}
		found := false
		for _, bs := range proto.InvariantBlindSpots {
			if bs.Id == "zero_volume_trade" {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("expected zero_volume_trade InvariantBlindSpot, not found")
		}
	})

	// Case 3: Capital conservation violated — IsReliable=false.
	capitalViolationResult := &backtest.Result{
		Config: backtest.Config{
			InitialCapital: decimal.NewFromInt(10000),
		},
		FinalBalance: decimal.NewFromInt(9990), // off by 10, tolerance for 10000 is 1.0
		Metrics: &antv1.BacktestMetrics{
			TotalReturn:  "0",
			MaxDrawdown:  "0",
			SharpeRatio:  "0",
			WinRate:      "0",
			TotalTrades:  1,
		},
		Trades: []backtest.Trade{
			{
				Side:       sdk.SideBuy,
				EntryTime:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				ExitTime:   time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC),
				EntryPrice: decimal.NewFromFloat(1.1),
				ExitPrice:  decimal.NewFromFloat(1.2),
				Volume:     decimal.NewFromFloat(0.1),
				Profit:     decimal.Zero, // expected balance = 10000, actual = 9990 → diff=10 > tolerance
			},
		},
	}

	t.Run("capital_not_conserved_is_unreliable", func(t *testing.T) {
		proto := buildBacktestResultProto(capitalViolationResult)
		if proto.IsReliable {
			t.Fatal("expected IsReliable=false for capital conservation violation, got true")
		}
		found := false
		for _, bs := range proto.InvariantBlindSpots {
			if bs.Id == "capital_not_conserved" {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("expected capital_not_conserved InvariantBlindSpot, not found")
		}
	})

	// Case 4: No trades — all invariants vacuously true, IsReliable=true.
	emptyResult := &backtest.Result{
		Config: backtest.Config{
			InitialCapital: decimal.NewFromInt(10000),
		},
		FinalBalance: decimal.NewFromInt(10000),
		Metrics: &antv1.BacktestMetrics{
			TotalReturn: "0",
			TotalTrades: 0,
		},
		Trades: nil,
	}

	t.Run("empty_trades_is_reliable", func(t *testing.T) {
		proto := buildBacktestResultProto(emptyResult)
		if !proto.IsReliable {
			t.Fatal("expected IsReliable=true for empty trades (vacuously true), got false")
		}
		if len(proto.InvariantBlindSpots) != 0 {
			t.Fatalf("expected 0 InvariantBlindSpots for empty trades, got %d", len(proto.InvariantBlindSpots))
		}
	})
}
