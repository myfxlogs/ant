package strategy

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/backtest"
	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
)

// helper: build a trade with explicit price, side, times (other fields defaulted to valid).
func makeTradeWithFields(entryPrice, exitPrice decimal.Decimal, side sdk.PositionSide, entryTime, exitTime time.Time) backtest.Trade {
	return backtest.Trade{
		Symbol:     "EURUSD",
		Side:       side,
		EntryTime:  entryTime,
		ExitTime:   exitTime,
		EntryPrice: entryPrice,
		ExitPrice:  exitPrice,
		Volume:     decimal.NewFromFloat(0.1),
		Profit:     decimal.NewFromInt(100),
		Commission: decimal.NewFromInt(5),
		Swap:       decimal.Zero,
		Comment:    "test",
	}
}

// validTrade returns a trade with all fields valid (positive prices, valid side, correct time order).
func validTrade() backtest.Trade {
	return makeTradeWithFields(
		decimal.NewFromFloat(1.1),
		decimal.NewFromFloat(1.11),
		sdk.SideBuy,
		time.UnixMilli(1000),
		time.UnixMilli(2000),
	)
}

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
