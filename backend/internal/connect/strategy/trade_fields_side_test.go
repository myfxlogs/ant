package strategy

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/backtest"
	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
)

func TestCheckSideValid_BuyAndSell(t *testing.T) {
	trades := []backtest.Trade{
		validTrade(),
		makeTradeWithFields(
			decimal.NewFromFloat(1.1),
			decimal.NewFromFloat(1.11),
			sdk.SideSell,
			time.UnixMilli(1000),
			time.UnixMilli(2000),
		),
	}
	result := makeResult(decimal.NewFromInt(10000), trades, decimal.NewFromInt(10000))
	bs := checkSideValid(result)
	if bs != nil {
		t.Fatalf("expected nil for SideBuy and SideSell, got %+v", bs)
	}
}

func TestCheckSideValid_InvalidZero(t *testing.T) {
	trade := validTrade()
	trade.Side = sdk.PositionSide(0)
	trades := []backtest.Trade{trade}
	result := makeResult(decimal.NewFromInt(10000), trades, decimal.NewFromInt(10000))
	bs := checkSideValid(result)
	if bs == nil {
		t.Fatal("expected BlindSpot for Side=0, got nil")
	}
	if bs.Id != "invalid_side" {
		t.Errorf("expected Id=invalid_side, got %s", bs.Id)
	}
}

func TestCheckSideValid_InvalidArbitrary(t *testing.T) {
	trade := validTrade()
	trade.Side = sdk.PositionSide(99)
	trades := []backtest.Trade{trade}
	result := makeResult(decimal.NewFromInt(10000), trades, decimal.NewFromInt(10000))
	bs := checkSideValid(result)
	if bs == nil {
		t.Fatal("expected BlindSpot for Side=99, got nil")
	}
}

func TestCheckSideValid_EmptyTrades(t *testing.T) {
	result := makeResult(decimal.NewFromInt(10000), nil, decimal.NewFromInt(10000))
	bs := checkSideValid(result)
	if bs != nil {
		t.Fatalf("expected nil for empty trades, got %+v", bs)
	}
}

func TestCheckSideValid_SingleTrade(t *testing.T) {
	trades := []backtest.Trade{validTrade()}
	result := makeResult(decimal.NewFromInt(10000), trades, decimal.NewFromInt(10000))
	bs := checkSideValid(result)
	if bs != nil {
		t.Fatalf("expected nil for single valid trade, got %+v", bs)
	}
}

func TestCheckSideValid_ViolationInMiddle(t *testing.T) {
	trades := []backtest.Trade{
		validTrade(),
		func() backtest.Trade {
			t := validTrade()
			t.Side = sdk.PositionSide(0)
			return t
		}(),
		validTrade(),
	}
	result := makeResult(decimal.NewFromInt(10000), trades, decimal.NewFromInt(10000))
	bs := checkSideValid(result)
	if bs == nil {
		t.Fatal("expected BlindSpot when invalid side is in middle trade, got nil")
	}
}

func TestCheckSideValid_ViolationAtEnd(t *testing.T) {
	trades := []backtest.Trade{
		validTrade(),
		validTrade(),
		func() backtest.Trade {
			t := validTrade()
			t.Side = sdk.PositionSide(42)
			return t
		}(),
	}
	result := makeResult(decimal.NewFromInt(10000), trades, decimal.NewFromInt(10000))
	bs := checkSideValid(result)
	if bs == nil {
		t.Fatal("expected BlindSpot when invalid side is in last trade, got nil")
	}
}

func TestCheckSideValid_FieldValidation(t *testing.T) {
	trade := validTrade()
	trade.Side = sdk.PositionSide(0)
	trades := []backtest.Trade{trade}
	result := makeResult(decimal.NewFromInt(10000), trades, decimal.NewFromInt(10000))
	bs := checkSideValid(result)
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

// --- checkTimeOrder tests ---

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
