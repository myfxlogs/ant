package strategy

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/backtest"
	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
)

// --- checkPricePositive tests ---

func TestCheckPricePositive_AllValid(t *testing.T) {
	trades := []backtest.Trade{
		validTrade(),
		makeTradeWithFields(
			decimal.NewFromFloat(0.5),
			decimal.NewFromFloat(0.6),
			sdk.SideSell,
			time.UnixMilli(3000),
			time.UnixMilli(4000),
		),
	}
	result := makeResult(decimal.NewFromInt(10000), trades, decimal.NewFromInt(10000))
	bs := checkPricePositive(result)
	if bs != nil {
		t.Fatalf("expected nil for all positive prices, got %+v", bs)
	}
}

func TestCheckPricePositive_ZeroEntryPrice(t *testing.T) {
	trades := []backtest.Trade{
		validTrade(),
		makeTradeWithFields(
			decimal.Zero,
			decimal.NewFromFloat(1.11),
			sdk.SideBuy,
			time.UnixMilli(1000),
			time.UnixMilli(2000),
		),
	}
	result := makeResult(decimal.NewFromInt(10000), trades, decimal.NewFromInt(10000))
	bs := checkPricePositive(result)
	if bs == nil {
		t.Fatal("expected BlindSpot for zero EntryPrice, got nil")
	}
	if bs.Id != "non_positive_price" {
		t.Errorf("expected Id=non_positive_price, got %s", bs.Id)
	}
}

func TestCheckPricePositive_NegativeExitPrice(t *testing.T) {
	trades := []backtest.Trade{
		makeTradeWithFields(
			decimal.NewFromFloat(1.1),
			decimal.NewFromInt(-1),
			sdk.SideBuy,
			time.UnixMilli(1000),
			time.UnixMilli(2000),
		),
	}
	result := makeResult(decimal.NewFromInt(10000), trades, decimal.NewFromInt(10000))
	bs := checkPricePositive(result)
	if bs == nil {
		t.Fatal("expected BlindSpot for negative ExitPrice, got nil")
	}
}

func TestCheckPricePositive_EntryPositiveExitZero(t *testing.T) {
	trades := []backtest.Trade{
		makeTradeWithFields(
			decimal.NewFromFloat(1.1),
			decimal.Zero,
			sdk.SideBuy,
			time.UnixMilli(1000),
			time.UnixMilli(2000),
		),
	}
	result := makeResult(decimal.NewFromInt(10000), trades, decimal.NewFromInt(10000))
	bs := checkPricePositive(result)
	if bs == nil {
		t.Fatal("expected BlindSpot for zero ExitPrice with positive EntryPrice, got nil")
	}
}

func TestCheckPricePositive_TinyPricePasses(t *testing.T) {
	trades := []backtest.Trade{
		makeTradeWithFields(
			decimal.New(1, -8),
			decimal.New(2, -8),
			sdk.SideBuy,
			time.UnixMilli(1000),
			time.UnixMilli(2000),
		),
	}
	result := makeResult(decimal.NewFromInt(10000), trades, decimal.NewFromInt(10000))
	bs := checkPricePositive(result)
	if bs != nil {
		t.Fatalf("expected nil for tiny but positive prices, got %+v", bs)
	}
}

func TestCheckPricePositive_EmptyTrades(t *testing.T) {
	result := makeResult(decimal.NewFromInt(10000), nil, decimal.NewFromInt(10000))
	bs := checkPricePositive(result)
	if bs != nil {
		t.Fatalf("expected nil for empty trades (vacuously true), got %+v", bs)
	}
}

func TestCheckPricePositive_SingleTrade(t *testing.T) {
	trades := []backtest.Trade{validTrade()}
	result := makeResult(decimal.NewFromInt(10000), trades, decimal.NewFromInt(10000))
	bs := checkPricePositive(result)
	if bs != nil {
		t.Fatalf("expected nil for single valid trade, got %+v", bs)
	}
}

func TestCheckPricePositive_ViolationInMiddle(t *testing.T) {
	trades := []backtest.Trade{
		validTrade(),
		makeTradeWithFields(
			decimal.Zero,
			decimal.NewFromFloat(1.11),
			sdk.SideBuy,
			time.UnixMilli(1000),
			time.UnixMilli(2000),
		),
		validTrade(),
	}
	result := makeResult(decimal.NewFromInt(10000), trades, decimal.NewFromInt(10000))
	bs := checkPricePositive(result)
	if bs == nil {
		t.Fatal("expected BlindSpot when violation is in middle trade, got nil")
	}
}

func TestCheckPricePositive_ViolationAtEnd(t *testing.T) {
	trades := []backtest.Trade{
		validTrade(),
		validTrade(),
		makeTradeWithFields(
			decimal.NewFromFloat(1.1),
			decimal.NewFromInt(-5),
			sdk.SideBuy,
			time.UnixMilli(1000),
			time.UnixMilli(2000),
		),
	}
	result := makeResult(decimal.NewFromInt(10000), trades, decimal.NewFromInt(10000))
	bs := checkPricePositive(result)
	if bs == nil {
		t.Fatal("expected BlindSpot when violation is in last trade, got nil")
	}
}

func TestCheckPricePositive_FieldValidation(t *testing.T) {
	trades := []backtest.Trade{
		makeTradeWithFields(
			decimal.Zero,
			decimal.NewFromFloat(1.11),
			sdk.SideBuy,
			time.UnixMilli(1000),
			time.UnixMilli(2000),
		),
	}
	result := makeResult(decimal.NewFromInt(10000), trades, decimal.NewFromInt(10000))
	bs := checkPricePositive(result)
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

// --- checkSideValid tests ---
