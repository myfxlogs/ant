package strategy

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/backtest"
	"alphaforge/strategy/sdk"
)

func TestBuildBacktestResponse_TradeFieldsAllValid(t *testing.T) {
	initialCapital := decimal.NewFromInt(10000)
	trades := make([]backtest.Trade, 12)
	for i := range trades {
		trades[i] = validTrade()
	}
	// FinalBalance = 10000 + 12*100 - 12*5 - 12*0 = 11140
	finalBalance := decimal.NewFromInt(11140)
	result := makeResult(finalBalance, trades, initialCapital)

	cfg := backtest.Config{
		InitialCapital: initialCapital,
	}
	params := backtestParams{}
	vmRunner := newMinimalVMRunner(t)

	resp, _, _, _ := buildBacktestResponse(result, cfg, params, vmRunner)

	for _, bs := range resp.BlindSpots {
		if bs.Id == "non_positive_price" || bs.Id == "invalid_side" || bs.Id == "time_order_violation" {
			t.Fatalf("expected no trade-field blind spots when all valid, got %s", bs.Id)
		}
	}
}

// Integration: non-positive price → blind spot + IsReliable=false.
func TestBuildBacktestResponse_NonPositivePrice(t *testing.T) {
	initialCapital := decimal.NewFromInt(10000)
	trades := make([]backtest.Trade, 12)
	for i := range trades {
		trades[i] = validTrade()
	}
	trades[5].EntryPrice = decimal.Zero
	finalBalance := decimal.NewFromInt(11140)
	result := makeResult(finalBalance, trades, initialCapital)

	cfg := backtest.Config{
		InitialCapital: initialCapital,
	}
	params := backtestParams{}
	vmRunner := newMinimalVMRunner(t)

	resp, _, _, _ := buildBacktestResponse(result, cfg, params, vmRunner)

	if resp.Risk.IsReliable {
		t.Error("expected IsReliable=false when price invariant is violated")
	}
	found := false
	for _, bs := range resp.BlindSpots {
		if bs.Id == "non_positive_price" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected non_positive_price blind spot in response")
	}
}

// Integration: invalid side → blind spot + IsReliable=false.
func TestBuildBacktestResponse_InvalidSide(t *testing.T) {
	initialCapital := decimal.NewFromInt(10000)
	trades := make([]backtest.Trade, 12)
	for i := range trades {
		trades[i] = validTrade()
	}
	trades[7].Side = sdk.PositionSide(0)
	finalBalance := decimal.NewFromInt(11140)
	result := makeResult(finalBalance, trades, initialCapital)

	cfg := backtest.Config{
		InitialCapital: initialCapital,
	}
	params := backtestParams{}
	vmRunner := newMinimalVMRunner(t)

	resp, _, _, _ := buildBacktestResponse(result, cfg, params, vmRunner)

	if resp.Risk.IsReliable {
		t.Error("expected IsReliable=false when side invariant is violated")
	}
	found := false
	for _, bs := range resp.BlindSpots {
		if bs.Id == "invalid_side" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected invalid_side blind spot in response")
	}
}

// Integration: time order violation → blind spot + IsReliable=false.
func TestBuildBacktestResponse_TimeOrderViolation(t *testing.T) {
	initialCapital := decimal.NewFromInt(10000)
	trades := make([]backtest.Trade, 12)
	for i := range trades {
		trades[i] = validTrade()
	}
	trades[3].EntryTime = time.UnixMilli(9000)
	trades[3].ExitTime = time.UnixMilli(1000)
	finalBalance := decimal.NewFromInt(11140)
	result := makeResult(finalBalance, trades, initialCapital)

	cfg := backtest.Config{
		InitialCapital: initialCapital,
	}
	params := backtestParams{}
	vmRunner := newMinimalVMRunner(t)

	resp, _, _, _ := buildBacktestResponse(result, cfg, params, vmRunner)

	if resp.Risk.IsReliable {
		t.Error("expected IsReliable=false when time order invariant is violated")
	}
	found := false
	for _, bs := range resp.BlindSpots {
		if bs.Id == "time_order_violation" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected time_order_violation blind spot in response")
	}
}

// Integration: empty trades → no trade-field blind spots (vacuously true).
func TestBuildBacktestResponse_TradeFieldsEmptyTrades(t *testing.T) {
	initialCapital := decimal.NewFromInt(10000)
	result := makeResult(initialCapital, nil, initialCapital)

	cfg := backtest.Config{
		InitialCapital: initialCapital,
	}
	params := backtestParams{}
	vmRunner := newMinimalVMRunner(t)

	resp, _, _, _ := buildBacktestResponse(result, cfg, params, vmRunner)

	for _, bs := range resp.BlindSpots {
		if bs.Id == "non_positive_price" || bs.Id == "invalid_side" || bs.Id == "time_order_violation" {
			t.Fatalf("expected no trade-field blind spots for empty trades, got %s", bs.Id)
		}
	}
}
