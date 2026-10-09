package strategy

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/backtest"
	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
)

func TestCheckTimeOrder_EntryBeforeExit(t *testing.T) {
	trades := []backtest.Trade{validTrade()}
	result := makeResult(decimal.NewFromInt(10000), trades, decimal.NewFromInt(10000))
	bs := checkTimeOrder(result)
	if bs != nil {
		t.Fatalf("expected nil for EntryTime < ExitTime, got %+v", bs)
	}
}

func TestCheckTimeOrder_EntryEqualsExit(t *testing.T) {
	ts := time.UnixMilli(5000)
	trades := []backtest.Trade{
		makeTradeWithFields(
			decimal.NewFromFloat(1.1),
			decimal.NewFromFloat(1.11),
			sdk.SideBuy,
			ts, ts,
		),
	}
	result := makeResult(decimal.NewFromInt(10000), trades, decimal.NewFromInt(10000))
	bs := checkTimeOrder(result)
	if bs != nil {
		t.Fatalf("expected nil for EntryTime == ExitTime (same-bar), got %+v", bs)
	}
}

func TestCheckTimeOrder_EntryAfterExit(t *testing.T) {
	trades := []backtest.Trade{
		makeTradeWithFields(
			decimal.NewFromFloat(1.1),
			decimal.NewFromFloat(1.11),
			sdk.SideBuy,
			time.UnixMilli(3000),
			time.UnixMilli(1000),
		),
	}
	result := makeResult(decimal.NewFromInt(10000), trades, decimal.NewFromInt(10000))
	bs := checkTimeOrder(result)
	if bs == nil {
		t.Fatal("expected BlindSpot for EntryTime > ExitTime, got nil")
	}
	if bs.Id != "time_order_violation" {
		t.Errorf("expected Id=time_order_violation, got %s", bs.Id)
	}
}

func TestCheckTimeOrder_EmptyTrades(t *testing.T) {
	result := makeResult(decimal.NewFromInt(10000), nil, decimal.NewFromInt(10000))
	bs := checkTimeOrder(result)
	if bs != nil {
		t.Fatalf("expected nil for empty trades, got %+v", bs)
	}
}

func TestCheckTimeOrder_SingleTrade(t *testing.T) {
	trades := []backtest.Trade{validTrade()}
	result := makeResult(decimal.NewFromInt(10000), trades, decimal.NewFromInt(10000))
	bs := checkTimeOrder(result)
	if bs != nil {
		t.Fatalf("expected nil for single valid trade, got %+v", bs)
	}
}

func TestCheckTimeOrder_ViolationInMiddle(t *testing.T) {
	trades := []backtest.Trade{
		validTrade(),
		makeTradeWithFields(
			decimal.NewFromFloat(1.1),
			decimal.NewFromFloat(1.11),
			sdk.SideBuy,
			time.UnixMilli(5000),
			time.UnixMilli(1000),
		),
		validTrade(),
	}
	result := makeResult(decimal.NewFromInt(10000), trades, decimal.NewFromInt(10000))
	bs := checkTimeOrder(result)
	if bs == nil {
		t.Fatal("expected BlindSpot when time violation is in middle trade, got nil")
	}
}

func TestCheckTimeOrder_ViolationAtEnd(t *testing.T) {
	trades := []backtest.Trade{
		validTrade(),
		validTrade(),
		makeTradeWithFields(
			decimal.NewFromFloat(1.1),
			decimal.NewFromFloat(1.11),
			sdk.SideBuy,
			time.UnixMilli(9000),
			time.UnixMilli(8000),
		),
	}
	result := makeResult(decimal.NewFromInt(10000), trades, decimal.NewFromInt(10000))
	bs := checkTimeOrder(result)
	if bs == nil {
		t.Fatal("expected BlindSpot when time violation is in last trade, got nil")
	}
}

func TestCheckTimeOrder_FieldValidation(t *testing.T) {
	trades := []backtest.Trade{
		makeTradeWithFields(
			decimal.NewFromFloat(1.1),
			decimal.NewFromFloat(1.11),
			sdk.SideBuy,
			time.UnixMilli(3000),
			time.UnixMilli(1000),
		),
	}
	result := makeResult(decimal.NewFromInt(10000), trades, decimal.NewFromInt(10000))
	bs := checkTimeOrder(result)
	if bs == nil {
		t.Fatal("expected BlindSpot, got nil")
	}
	if bs.Category != "invariant" {
		t.Errorf("expected Category=invariant, got %s", bs.Category)
	}
	if bs.Severity != interp.SeverityFatal {
		t.Errorf("expected Severity=%s, got %s", interp.SeverityFatal, bs.Severity)
	}
	if bs.Description == "" {
		t.Error("expected non-empty Description")
	}
}

// --- Integration: buildBacktestResponse with trade field invariants ---

// Integration: all fields valid → no trade-field blind spots.
