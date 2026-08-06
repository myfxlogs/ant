package strategy

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/strategy/backtest"
	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go"
	"alphaforge/tools/mql2go/interp"
)

func newMinimalVMRunner(t *testing.T) *mql2go.VMRunner {
	t.Helper()
	return mql2go.NewVMRunner(&mql2go.Bytecode{})
}

func makeTrade(volume string) backtest.Trade {
	return backtest.Trade{
		Symbol:     "EURUSD",
		Side:       sdk.SideBuy,
		EntryTime:  time.UnixMilli(1000),
		ExitTime:   time.UnixMilli(2000),
		EntryPrice: decimal.NewFromFloat(1.1),
		ExitPrice:  decimal.NewFromFloat(1.11),
		Volume:     parseDecimal(volume),
		Profit:     decimal.NewFromFloat(10),
		Comment:    "test",
	}
}

func makeMetrics(totalTrades int32) *antv1.BacktestMetrics {
	return &antv1.BacktestMetrics{
		TotalTrades:  totalTrades,
		TotalReturn:  "0.1",
		MaxDrawdown:  "0.05",
		SharpeRatio:  "1.5",
		WinRate:      "0.6",
		ProfitFactor: "2.0",
	}
}

// --- checkVolumeInvariant unit tests ---

// Positive: all trades have Volume > 0 → invariant passes (nil returned).
func TestCheckVolumeInvariant_AllPositive(t *testing.T) {
	trades := []backtest.Trade{
		makeTrade("0.01"),
		makeTrade("0.1"),
		makeTrade("1.0"),
		makeTrade("10"),
	}
	bs := checkVolumeInvariant(trades)
	if bs != nil {
		t.Fatalf("expected nil BlindSpot for all-positive volumes, got %+v", bs)
	}
}

// Negative: one trade with Volume == 0 → invariant triggers.
func TestCheckVolumeInvariant_ZeroVolume(t *testing.T) {
	trades := []backtest.Trade{
		makeTrade("0.1"),
		makeTrade("0"),
		makeTrade("0.01"),
	}
	bs := checkVolumeInvariant(trades)
	if bs == nil {
		t.Fatal("expected BlindSpot for zero-volume trade, got nil")
	}
	if bs.Id != "zero_volume_trade" {
		t.Errorf("expected Id=zero_volume_trade, got %s", bs.Id)
	}
	if bs.Severity != interp.SeverityFatal {
		t.Errorf("expected Severity=%s, got %s", interp.SeverityFatal, bs.Severity)
	}
	if bs.Description == "" {
		t.Error("expected non-empty Description")
	}
}

// Negative: one trade with negative Volume → invariant triggers.
func TestCheckVolumeInvariant_NegativeVolume(t *testing.T) {
	trades := []backtest.Trade{
		makeTrade("0.1"),
		makeTrade("-0.05"),
	}
	bs := checkVolumeInvariant(trades)
	if bs == nil {
		t.Fatal("expected BlindSpot for negative-volume trade, got nil")
	}
	if bs.Id != "zero_volume_trade" {
		t.Errorf("expected Id=zero_volume_trade, got %s", bs.Id)
	}
}

// Edge: no trades → vacuously true, no blind spot.
func TestCheckVolumeInvariant_NoTrades(t *testing.T) {
	trades := []backtest.Trade{}
	bs := checkVolumeInvariant(trades)
	if bs != nil {
		t.Fatalf("expected nil for empty trades (vacuously true), got %+v", bs)
	}
}

// Edge: nil trades slice → vacuously true.
func TestCheckVolumeInvariant_NilTrades(t *testing.T) {
	bs := checkVolumeInvariant(nil)
	if bs != nil {
		t.Fatalf("expected nil for nil trades (vacuously true), got %+v", bs)
	}
}

// Edge: single trade with Volume == 0 → triggers.
func TestCheckVolumeInvariant_SingleZeroVolume(t *testing.T) {
	trades := []backtest.Trade{
		makeTrade("0"),
	}
	bs := checkVolumeInvariant(trades)
	if bs == nil {
		t.Fatal("expected BlindSpot for single zero-volume trade")
	}
}

// Edge: single trade with very small positive Volume → passes.
func TestCheckVolumeInvariant_SmallPositiveVolume(t *testing.T) {
	trades := []backtest.Trade{
		makeTrade("0.00000001"),
	}
	bs := checkVolumeInvariant(trades)
	if bs != nil {
		t.Fatalf("expected nil for small positive volume, got %+v", bs)
	}
}

// Edge: first trade valid, last trade zero → triggers (checks all trades, not just first).
func TestCheckVolumeInvariant_LastTradeZero(t *testing.T) {
	trades := []backtest.Trade{
		makeTrade("0.1"),
		makeTrade("0.2"),
		makeTrade("0.3"),
		makeTrade("0"),
	}
	bs := checkVolumeInvariant(trades)
	if bs == nil {
		t.Fatal("expected BlindSpot when last trade has zero volume")
	}
}

// Edge: first trade zero → triggers immediately.
func TestCheckVolumeInvariant_FirstTradeZero(t *testing.T) {
	trades := []backtest.Trade{
		makeTrade("0"),
		makeTrade("0.1"),
	}
	bs := checkVolumeInvariant(trades)
	if bs == nil {
		t.Fatal("expected BlindSpot when first trade has zero volume")
	}
}

// Edge: all trades zero → triggers.
func TestCheckVolumeInvariant_AllZero(t *testing.T) {
	trades := []backtest.Trade{
		makeTrade("0"),
		makeTrade("0"),
		makeTrade("0"),
	}
	bs := checkVolumeInvariant(trades)
	if bs == nil {
		t.Fatal("expected BlindSpot when all trades have zero volume")
	}
}

// Edge: mix of zero and negative volumes → triggers.
func TestCheckVolumeInvariant_MixZeroAndNegative(t *testing.T) {
	trades := []backtest.Trade{
		makeTrade("0.1"),
		makeTrade("0"),
		makeTrade("-1"),
		makeTrade("0.01"),
	}
	bs := checkVolumeInvariant(trades)
	if bs == nil {
		t.Fatal("expected BlindSpot for mix of zero and negative volumes")
	}
}

// --- Integration: buildBacktestResponse with volume invariant ---

// Positive: all trades valid → IsReliable not overridden by invariant.
func TestBuildBacktestResponse_VolumeInvariantPass(t *testing.T) {
	trades := []backtest.Trade{
		makeTrade("0.1"),
		makeTrade("0.2"),
	}
	for i := range trades {
		trades[i].Side = sdk.SideBuy
	}

	result := &backtest.Result{
		Metrics: makeMetrics(2),
		Trades:  trades,
	}
	cfg := backtest.Config{
		InitialCapital: decimal.NewFromInt(10000),
	}
	params := backtestParams{}
	vmRunner := newMinimalVMRunner(t)

	resp, _, _, _ := buildBacktestResponse(result, cfg, params, vmRunner)

	// With only 2 trades, assessRisk already sets IsReliable=false (< 10 trades).
	// The invariant should NOT add a zero_volume_trade blind spot.
	for _, bs := range resp.BlindSpots {
		if bs.Id == "zero_volume_trade" {
			t.Fatal("expected no zero_volume_trade blind spot when all volumes are positive")
		}
	}
}

// Negative: trade with zero volume → IsReliable=false, blind spot present.
func TestBuildBacktestResponse_VolumeInvariantFail(t *testing.T) {
	trades := []backtest.Trade{
		makeTrade("0.1"),
		makeTrade("0"),
		makeTrade("0.2"),
	}

	result := &backtest.Result{
		Metrics: makeMetrics(15), // 15 trades reported, but one has zero volume
		Trades:  trades,
	}
	cfg := backtest.Config{
		InitialCapital: decimal.NewFromInt(10000),
	}
	params := backtestParams{}
	vmRunner := newMinimalVMRunner(t)

	resp, _, _, _ := buildBacktestResponse(result, cfg, params, vmRunner)

	if resp.Risk.IsReliable {
		t.Error("expected IsReliable=false when a trade has zero volume")
	}

	found := false
	for _, bs := range resp.BlindSpots {
		if bs.Id == "zero_volume_trade" {
			found = true
			if bs.Severity != interp.SeverityFatal {
				t.Errorf("expected Severity=%s, got %s", interp.SeverityFatal, bs.Severity)
			}
			if bs.Description == "" {
				t.Error("expected non-empty Description for zero_volume_trade blind spot")
			}
		}
	}
	if !found {
		t.Fatal("expected zero_volume_trade blind spot in response")
	}
}

// Edge: no trades → invariant vacuously true, IsReliable not overridden by invariant.
func TestBuildBacktestResponse_VolumeInvariantNoTrades(t *testing.T) {
	result := &backtest.Result{
		Metrics: makeMetrics(0),
		Trades:  nil,
	}
	cfg := backtest.Config{
		InitialCapital: decimal.NewFromInt(10000),
	}
	params := backtestParams{}
	vmRunner := newMinimalVMRunner(t)

	resp, _, _, _ := buildBacktestResponse(result, cfg, params, vmRunner)

	for _, bs := range resp.BlindSpots {
		if bs.Id == "zero_volume_trade" {
			t.Fatal("expected no zero_volume_trade blind spot when there are no trades")
		}
	}
}

// Edge: assessRisk says reliable (>= 10 trades) but invariant overrides to false.
func TestBuildBacktestResponse_InvariantOverridesReliable(t *testing.T) {
	trades := make([]backtest.Trade, 12)
	for i := range trades {
		trades[i] = makeTrade("0.1")
	}
	trades[5] = makeTrade("0") // slip in a zero-volume trade

	result := &backtest.Result{
		Metrics: makeMetrics(12),
		Trades:  trades,
	}
	cfg := backtest.Config{
		InitialCapital: decimal.NewFromInt(10000),
	}
	params := backtestParams{}
	vmRunner := newMinimalVMRunner(t)

	resp, _, _, _ := buildBacktestResponse(result, cfg, params, vmRunner)

	// assessRisk would set IsReliable=true (12 >= 10), but invariant must override.
	if resp.Risk.IsReliable {
		t.Error("expected IsReliable=false overridden by volume invariant despite >= 10 trades")
	}
}
