package backtest

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

func df(f float64) decimal.Decimal {
	return decimal.NewFromFloat(f)
}

// TestOHLCPath_Buy_TPBeforeSL_BullishBar verifies that in a bullish bar (O→H→L→C),
// TP in the H segment triggers before SL in the L segment.
func TestOHLCPath_Buy_TPBeforeSL_BullishBar(t *testing.T) {
	// Bullish bar: O=1.1000, H=1.1120, L=1.0950, C=1.1080
	// Path: O→H (ascending), H→L (descending), L→C (ascending)
	// Buy position entry=1.1000, SL=1.0960, TP=1.1110
	// TP=1.1110 is in segment 1 (O→H: 1.1000→1.1120, ascending, TP in range)
	// SL=1.0960 is in segment 2 (H→L: 1.1120→1.0950, descending, SL in range)
	// TP should trigger first (segment 1 before segment 2)
	o, h, l, c := df(1.1000), df(1.1120), df(1.0950), df(1.1080)
	sl, tp := df(1.0960), df(1.1110)

	closed, closePrice := checkBuySLTPPath(o, h, l, c, sl, tp)
	if !closed {
		t.Fatal("expected position to close")
	}
	if !closePrice.Equal(tp) {
		t.Errorf("expected close at TP=%s, got %s (TP should trigger before SL in bullish bar)", tp.String(), closePrice.String())
	}
}

// TestOHLCPath_Buy_SLBeforeTP_BearishBar verifies that in a bearish bar (O→L→H→C),
// SL in the L segment triggers before TP in the H segment.
func TestOHLCPath_Buy_SLBeforeTP_BearishBar(t *testing.T) {
	// Bearish bar: O=1.1080, L=1.0950, H=1.1120, C=1.1000
	// Path: O→L (descending), L→H (ascending), H→C (descending)
	// Buy position entry=1.1080, SL=1.0960, TP=1.1110
	// SL=1.0960 is in segment 1 (O→L: 1.1080→1.0950, descending, SL in range)
	// TP=1.1110 is in segment 2 (L→H: 1.0950→1.1120, ascending, TP in range)
	// SL should trigger first (segment 1 before segment 2)
	o, h, l, c := df(1.1080), df(1.1120), df(1.0950), df(1.1000)
	sl, tp := df(1.0960), df(1.1110)

	closed, closePrice := checkBuySLTPPath(o, h, l, c, sl, tp)
	if !closed {
		t.Fatal("expected position to close")
	}
	if !closePrice.Equal(sl) {
		t.Errorf("expected close at SL=%s, got %s (SL should trigger before TP in bearish bar)", sl.String(), closePrice.String())
	}
}

// TestOHLCPath_Sell_TPBeforeSL_BearishBar verifies sell position in bearish bar:
// O→L→H→C path, TP in L segment (descending) before SL in H segment (ascending).
func TestOHLCPath_Sell_TPBeforeSL_BearishBar(t *testing.T) {
	// Bearish bar: O=1.1080, L=1.0950, H=1.1120, C=1.1000
	// Path: O→L (descending), L→H (ascending), H→C (descending)
	// Sell position entry=1.1080, SL=1.1110, TP=1.0960
	// TP=1.0960 is in segment 1 (O→L: 1.1080→1.0950, descending, TP in range for sell)
	// SL=1.1110 is in segment 2 (L→H: 1.0950→1.1120, ascending, SL in range for sell)
	// TP should trigger first
	o, h, l, c := df(1.1080), df(1.1120), df(1.0950), df(1.1000)
	sl, tp := df(1.1110), df(1.0960)

	closed, closePrice := checkSellSLTPPath(o, h, l, c, sl, tp)
	if !closed {
		t.Fatal("expected position to close")
	}
	if !closePrice.Equal(tp) {
		t.Errorf("expected close at TP=%s, got %s (TP should trigger before SL for sell in bearish bar)", tp.String(), closePrice.String())
	}
}

// TestOHLCPath_Sell_SLBeforeTP_BullishBar verifies sell position in bullish bar:
// O→H→L→C path, SL in H segment (ascending) before TP in L segment (descending).
func TestOHLCPath_Sell_SLBeforeTP_BullishBar(t *testing.T) {
	// Bullish bar: O=1.1000, H=1.1120, L=1.0950, C=1.1080
	// Path: O→H (ascending), H→L (descending), L→C (ascending)
	// Sell position entry=1.1000, SL=1.1110, TP=1.0960
	// SL=1.1110 is in segment 1 (O→H: 1.1000→1.1120, ascending, SL in range for sell)
	// TP=1.0960 is in segment 2 (H→L: 1.1120→1.0950, descending, TP in range for sell)
	// SL should trigger first
	o, h, l, c := df(1.1000), df(1.1120), df(1.0950), df(1.1080)
	sl, tp := df(1.1110), df(1.0960)

	closed, closePrice := checkSellSLTPPath(o, h, l, c, sl, tp)
	if !closed {
		t.Fatal("expected position to close")
	}
	if !closePrice.Equal(sl) {
		t.Errorf("expected close at SL=%s, got %s (SL should trigger before TP for sell in bullish bar)", sl.String(), closePrice.String())
	}
}

// TestOHLCPath_GapOpen_FillsAtOpen verifies that when Open gaps through SL,
// the fill price is Open (not SL).
func TestOHLCPath_GapOpen_FillsAtOpen(t *testing.T) {
	// Buy position entry=1.1000, SL=1.1050
	// Bar opens at 1.1060 (above SL for buy = gap up, but SL for buy is below entry)
	// Actually for buy: SL is below entry. Gap down: Open=1.0900, SL=1.0950
	// Open(1.0900) <= SL(1.0950) → SL hit at Open price
	o, h, l, c := df(1.0900), df(1.0920), df(1.0880), df(1.0910)
	sl, tp := df(1.0950), decimal.Zero

	closed, closePrice := checkBuySLTPPath(o, h, l, c, sl, tp)
	if !closed {
		t.Fatal("expected position to close on gap")
	}
	if !closePrice.Equal(o) {
		t.Errorf("expected close at Open=%s (gap), got %s", o.String(), closePrice.String())
	}
}

// TestOHLCPath_NoHit verifies no trigger when SL/TP are outside the bar range.
func TestOHLCPath_NoHit(t *testing.T) {
	o, h, l, c := df(1.1000), df(1.1080), df(1.0950), df(1.1050)
	sl, tp := df(1.0900), df(1.1200) // both outside bar range

	closed, _ := checkBuySLTPPath(o, h, l, c, sl, tp)
	if closed {
		t.Error("expected no close when SL/TP outside bar range")
	}
}

// TestOHLCPath_SameSegment_NearerFirst tests the defensive same-segment rule.
// With valid SL/TP this scenario is unreachable. Even with invalid SL/TP (SL>entry for buy),
// the gap-at-open check fires first (Open <= SL), so the same-segment code is never reached.
// This test verifies the gap-at-open correctly handles the invalid configuration.
func TestOHLCPath_SameSegment_NearerFirst(t *testing.T) {
	// Defensive test: buy position with SL=1.1150 (above entry=1.1000, invalid for buy)
	// Bullish bar: O=1.1000, H=1.1200, L=1.1050, C=1.1100
	// Open(1.1000) <= SL(1.1150) → gap-at-open triggers at Open price before segments are checked.
	// This confirms the defensive same-segment code is unreachable even with invalid SL/TP.
	o, h, l, c := df(1.1000), df(1.1200), df(1.1050), df(1.1100)
	sl, tp := df(1.1150), df(1.1100) // invalid: SL > entry for buy

	closed, closePrice := checkBuySLTPPath(o, h, l, c, sl, tp)
	if !closed {
		t.Fatal("expected position to close (defensive test)")
	}
	// Gap-at-open: SL hit at Open price (not SL price)
	if !closePrice.Equal(o) {
		t.Errorf("defensive: expected gap-at-open fill at Open=%s, got %s", o.String(), closePrice.String())
	}
}

// TestKlineRange_BehaviorUnchanged verifies that KLINE_RANGE mode still uses
// the existing checkSLTP logic (SL always first when both in range).
func TestKlineRange_BehaviorUnchanged(t *testing.T) {
	// Same bullish bar as TestOHLCPath_Buy_TPBeforeSL_BullishBar
	// KLINE_RANGE: SL always first → should return SL, not TP
	o, h, l := df(1.1000), df(1.1120), df(1.0950)
	sl, tp := df(1.0960), df(1.1110)

	closed, closePrice := checkBuySLTP(o, h, l, sl, tp)
	if !closed {
		t.Fatal("expected position to close")
	}
	// KLINE_RANGE: SL always first (conservative)
	if !closePrice.Equal(sl) {
		t.Errorf("KLINE_RANGE: expected SL first (conservative), got %s", closePrice.String())
	}
}

// TestOHLCPath_Adversarial_PathOrderMatters verifies that removing path-based
// checking (reverting to KLINE_RANGE) would give a different result for this case.
func TestOHLCPath_Adversarial_PathOrderMatters(t *testing.T) {
	o, h, l, c := df(1.1000), df(1.1120), df(1.0950), df(1.1080)
	sl, tp := df(1.0960), df(1.1110)

	// OHLC_PATH: TP first (segment 1 before segment 2)
	_, pathPrice := checkBuySLTPPath(o, h, l, c, sl, tp)
	// KLINE_RANGE: SL first (conservative)
	_, rangePrice := checkBuySLTP(o, h, l, sl, tp)

	if pathPrice.Equal(rangePrice) {
		t.Errorf("adversarial: OHLC_PATH and KLINE_RANGE should differ for this case, both gave %s", pathPrice.String())
	}
	if !pathPrice.Equal(tp) {
		t.Errorf("OHLC_PATH should return TP=%s, got %s", tp.String(), pathPrice.String())
	}
	if !rangePrice.Equal(sl) {
		t.Errorf("KLINE_RANGE should return SL=%s, got %s", sl.String(), rangePrice.String())
	}
}

// TestOHLCPath_EngineIntegration verifies the engine routes to checkSLTPPath
// when SimulationMode == "OHLC_PATH".
func TestOHLCPath_EngineIntegration(t *testing.T) {
	broker := NewSimBroker(Config{
		Symbol:         "EURUSD",
		InitialCapital: decimal.NewFromInt(100000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
		SimulationMode: "OHLC_PATH",
	})
	broker.SetBarPrice(df(1.1000))
	broker.SetBarTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))

	// Open a buy position with SL and TP
	res, _ := broker.OrderSend(sdk.OrderRequest{
		Symbol:     "EURUSD",
		Side:       sdk.SideBuy,
		Type:       sdk.OrderMarket,
		Volume:     df(0.1),
		Price:      df(1.1000),
		StopLoss:   df(1.0960),
		TakeProfit: df(1.1110),
	})

	// Find the position and set its SL/TP (OrderSend may not set them on market orders)
	for _, pos := range broker.positions {
		if pos.Ticket == res.Ticket {
			pos.StopLoss = df(1.0960)
			pos.TakeProfit = df(1.1110)
		}
	}

	// Bullish bar where TP is in H segment and SL is in L segment
	// OHLC_PATH should trigger TP first
	bar := sdk.Bar{
		Open: df(1.1000), High: df(1.1120),
		Low: df(1.0950), Close: df(1.1080),
		Volume: 1000, Timestamp: time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC).UnixMilli(),
	}

	engine := &Engine{broker: broker}
	engine.checkSLTPPath(bar)

	// Position should be closed at TP price (1.1110), not SL (1.0960)
	if len(broker.Positions(0)) != 0 {
		t.Fatalf("expected position to be closed, still have %d", len(broker.Positions(0)))
	}
	history := broker.HistoryOrders(0, 0)
	if len(history) == 0 {
		t.Fatal("expected closed position in history")
	}
	last := history[len(history)-1]
	if !last.ClosePrice.Equal(df(1.1110)) {
		t.Errorf("expected close at TP=1.1110, got %s", last.ClosePrice.String())
	}
}
