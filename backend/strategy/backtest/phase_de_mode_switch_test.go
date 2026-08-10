package backtest

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

// TestEngineRun_OHLCPath_vs_KlineRange_ModeSwitch is the adversarial integration test
// that verifies the engine.Run() main loop correctly routes to checkSLTPPath vs checkSLTP
// based on SimulationMode. This test goes through the full engine.Run() path — not direct
// function calls.
//
// Setup: Buy position with SL=1.0960, TP=1.1110. Bullish bar O=1.1000, H=1.1120, L=1.0950, C=1.1080.
// - OHLC_PATH: TP triggers first (segment O→H ascending, TP=1.1110 in range) → close at TP
// - KLINE_RANGE: SL triggers first (conservative, SL always checked first) → close at SL
//
// Adversarial proof: if the main loop always calls checkSLTP (ignoring SimulationMode),
// the OHLC_PATH test would close at SL (1.0960) instead of TP (1.1110), and this test fails.
func TestEngineRun_OHLCPath_vs_KlineRange_ModeSwitch(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Bars: bar 0 is skipped by engine (loop starts at i=1).
	// Bar 1: barsSeen=1, no signal.
	// Bar 2: barsSeen=2, buy signal → same_bar_close → position opens at Close=1.1000.
	// Bar 3: SL/TP test bar (bullish: O=1.1000, H=1.1120, L=1.0950, C=1.1080).
	//   OHLC_PATH: TP=1.1110 in segment 1 (O→H ascending) → TP triggers first.
	//   KLINE_RANGE: SL=1.0960 checked first (conservative) → SL triggers first.
	// Bar 4: extra bar to ensure position closure is recorded.
	bars := []sdk.Bar{
		{
			Open: df(1.1000), High: df(1.1020), Low: df(1.0990), Close: df(1.1010),
			Volume: 1000, Timestamp: baseTime.UnixMilli(),
		},
		{
			Open: df(1.1010), High: df(1.1030), Low: df(1.1005), Close: df(1.1020),
			Volume: 1000, Timestamp: baseTime.Add(time.Hour).UnixMilli(),
		},
		{
			Open: df(1.1020), High: df(1.1040), Low: df(1.1010), Close: df(1.1000),
			Volume: 1000, Timestamp: baseTime.Add(2 * time.Hour).UnixMilli(),
		},
		// SL/TP bar: bullish O=1.1000, H=1.1120, L=1.0950, C=1.1080
		{
			Open: df(1.1000), High: df(1.1120), Low: df(1.0950), Close: df(1.1080),
			Volume: 1000, Timestamp: baseTime.Add(3 * time.Hour).UnixMilli(),
		},
		{
			Open: df(1.1080), High: df(1.1100), Low: df(1.1070), Close: df(1.1090),
			Volume: 1000, Timestamp: baseTime.Add(4 * time.Hour).UnixMilli(),
		},
	}

	sl := df(1.0960)
	tp := df(1.1110)

	// --- OHLC_PATH mode ---
	ohlcStrategy := &signalStrategy{
		buyAtBar:   2, // buy on bar 2 (barsSeen==2)
		stopLoss:   sl,
		takeProfit: tp,
	}
	ohlcCfg := Config{
		Symbol:         "EURUSD",
		Timeframe:      "H1",
		InitialCapital: decimal.NewFromInt(100000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
		SimulationMode: "OHLC_PATH",
		SignalTiming:   "same_bar_close",
	}
	ohlcEngine := New(ohlcCfg, ohlcStrategy, bars)
	ohlcResult, err := ohlcEngine.Run(context.Background())
	if err != nil {
		t.Fatalf("OHLC_PATH engine.Run failed: %v", err)
	}

	// --- KLINE_RANGE mode ---
	klineStrategy := &signalStrategy{
		buyAtBar:   2,
		stopLoss:   sl,
		takeProfit: tp,
	}
	klineCfg := Config{
		Symbol:         "EURUSD",
		Timeframe:      "H1",
		InitialCapital: decimal.NewFromInt(100000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
		SimulationMode: "KLINE_RANGE",
		SignalTiming:   "same_bar_close",
	}
	klineEngine := New(klineCfg, klineStrategy, bars)
	klineResult, err := klineEngine.Run(context.Background())
	if err != nil {
		t.Fatalf("KLINE_RANGE engine.Run failed: %v", err)
	}

	// Both should have at least 1 trade (position opened and closed)
	if len(ohlcResult.Trades) == 0 {
		t.Fatalf("OHLC_PATH: expected at least 1 trade, got 0")
	}
	if len(klineResult.Trades) == 0 {
		t.Fatalf("KLINE_RANGE: expected at least 1 trade, got 0")
	}

	// OHLC_PATH: position should close at TP (1.1110)
	ohlcLastTrade := ohlcResult.Trades[len(ohlcResult.Trades)-1]
	if !ohlcLastTrade.ExitPrice.Equal(tp) {
		t.Errorf("OHLC_PATH: expected exit at TP=%s, got %s (TP should trigger before SL in bullish bar)",
			tp.String(), ohlcLastTrade.ExitPrice.String())
	}

	// KLINE_RANGE: position should close at SL (1.0960)
	klineLastTrade := klineResult.Trades[len(klineResult.Trades)-1]
	if !klineLastTrade.ExitPrice.Equal(sl) {
		t.Errorf("KLINE_RANGE: expected exit at SL=%s, got %s (SL should trigger first, conservative)",
			sl.String(), klineLastTrade.ExitPrice.String())
	}

	// The two modes must produce different results (adversarial: if main loop
	// always calls checkSLTP, both would close at SL and this assertion fails)
	if ohlcLastTrade.ExitPrice.Equal(klineLastTrade.ExitPrice) {
		t.Errorf("adversarial: OHLC_PATH and KLINE_RANGE should produce different exit prices, "+
			"both got %s (main loop mode switch not working)", ohlcLastTrade.ExitPrice.String())
	}

	// FinalBalance must differ (TP profit vs SL loss)
	if ohlcResult.FinalBalance.Equal(klineResult.FinalBalance) {
		t.Errorf("adversarial: OHLC_PATH and KLINE_RANGE should produce different FinalBalance, "+
			"both got %s", ohlcResult.FinalBalance.String())
	}
}
