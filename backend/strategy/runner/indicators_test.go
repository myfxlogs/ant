package runner

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

// --- Indicator stub tests ---

func TestIndicatorSet_StubIndicators(t *testing.T) {
	r := New(Config{Symbol: "EURUSD"})
	bars := sdk.BarsToSlice([]sdk.Bar{
		{Close: dec("1.0"), High: dec("2.0"), Low: dec("0.5"), Open: dec("0.8"), Volume: 100, Timestamp: 1000},
		{Close: dec("1.1"), High: dec("2.1"), Low: dec("0.6"), Open: dec("0.9"), Volume: 110, Timestamp: 2000},
		{Close: dec("1.2"), High: dec("2.2"), Low: dec("0.7"), Open: dec("1.0"), Volume: 120, Timestamp: 3000},
		{Close: dec("1.3"), High: dec("2.3"), Low: dec("0.8"), Open: dec("1.1"), Volume: 130, Timestamp: 4000},
		{Close: dec("1.4"), High: dec("2.4"), Low: dec("0.9"), Open: dec("1.2"), Volume: 140, Timestamp: 5000},
	})
	r.ctx.setBars(bars)
	ind := r.ctx.Indicators()

	// Call a few real indicator methods to exercise the cache path.
	_ = ind.MA(3, 0, "SMA", 0)
	_ = ind.EMA(3, 0)
	_ = ind.RSI(3, 0, 0)

	// Call all stub indicators — they all return 0 but should not panic.
	ind.Alligator(13, 8, 8, 5, 5, 3, "SMA", 1, 0)
	ind.Ichimoku(9, 26, 52, 0)
	ind.Envelopes(14, dec("0.1"), "SMA", 1, 0)
	ind.DeMarker(14, 0)
	ind.OsMA(12, 26, 9, 1, 0)
	ind.RVI(10, 0)
	ind.Force(13, "EMA", 1, 0)
	ind.Fractals(0)
	ind.Gator(13, 8, 8, 5, 5, 3, "SMA", 1, 0)
	ind.AC(0)
	ind.AD(0)
	ind.AO(0)
	ind.BearsPower(13, 1, 0)
	ind.BullsPower(13, 1, 0)
	ind.BWMFI(0)

	// MQL5-only stubs.
	ind.AMA(9, 2, 30, 1, 0)
	ind.DEMA(14, 1, 0)
	ind.TEMA(14, 1, 0)
	ind.FrAMA(14, 1, 0)
	ind.VIDyA(9, 0, 14, 0, 1, 0)
	ind.TriX(14, 1, 0)
	ind.ADXWilder(14, 0)
	ind.Chaikin(3, 10, 0)
	ind.Volumes(0)
}

// --- Runner optional interface tests with real implementations ---

type fullStrategy struct {
	barOnlyStrategy
	tickCalled    bool
	tradeCalled   bool
	timerCalled   bool
	transCalled   bool
	bookCalled    bool
}

func (s *fullStrategy) OnTick(ctx sdk.Context, bid, ask decimal.Decimal) (*sdk.Signal, error) {
	s.tickCalled = true
	return &sdk.Signal{Action: sdk.ActionBuy}, nil
}

func (s *fullStrategy) OnTrade(ctx sdk.Context, event sdk.TradeEvent) (*sdk.Signal, error) {
	s.tradeCalled = true
	return nil, nil
}

func (s *fullStrategy) OnTimer(ctx sdk.Context) (*sdk.Signal, error) {
	s.timerCalled = true
	return nil, nil
}

func (s *fullStrategy) OnTradeTransaction(ctx sdk.Context) (*sdk.Signal, error) {
	s.transCalled = true
	return nil, nil
}

func (s *fullStrategy) OnBookEvent(ctx sdk.Context) (*sdk.Signal, error) {
	s.bookCalled = true
	return nil, nil
}

func TestRunner_OnTick_WithTickStrategy(t *testing.T) {
	r := New(Config{})
	strat := &fullStrategy{}
	r.SetStrategy(strat)

	sig, err := r.OnTick(context.Background(), dec("1.1"), dec("1.2"))
	if err != nil {
		t.Fatalf("OnTick failed: %v", err)
	}
	if sig == nil || sig.Action != sdk.ActionBuy {
		t.Errorf("OnTick signal = %v, want ActionBuy", sig)
	}
	if !strat.tickCalled {
		t.Error("OnTick was not called on strategy")
	}
}

func TestRunner_OnTrade_WithTradeStrategy(t *testing.T) {
	r := New(Config{})
	strat := &fullStrategy{}
	r.SetStrategy(strat)

	_, err := r.OnTrade(context.Background(), sdk.TradeEvent{Ticket: 1})
	if err != nil {
		t.Fatalf("OnTrade failed: %v", err)
	}
	if !strat.tradeCalled {
		t.Error("OnTrade was not called on strategy")
	}
}

func TestRunner_OnTimerTick_WithTimerStrategy(t *testing.T) {
	r := New(Config{})
	strat := &fullStrategy{}
	r.SetStrategy(strat)

	_, err := r.OnTimerTick(context.Background())
	if err != nil {
		t.Fatalf("OnTimerTick failed: %v", err)
	}
	if !strat.timerCalled {
		t.Error("OnTimer was not called on strategy")
	}
}

func TestRunner_OnTradeTransaction_WithImpl(t *testing.T) {
	r := New(Config{})
	strat := &fullStrategy{}
	r.SetStrategy(strat)

	_, err := r.OnTradeTransaction(context.Background())
	if err != nil {
		t.Fatalf("OnTradeTransaction failed: %v", err)
	}
	if !strat.transCalled {
		t.Error("OnTradeTransaction was not called on strategy")
	}
	if !r.HasOnTradeTransaction() {
		t.Error("HasOnTradeTransaction should be true for fullStrategy")
	}
}

func TestRunner_OnBookEvent_WithImpl(t *testing.T) {
	r := New(Config{})
	strat := &fullStrategy{}
	r.SetStrategy(strat)

	_, err := r.OnBookEvent(context.Background())
	if err != nil {
		t.Fatalf("OnBookEvent failed: %v", err)
	}
	if !strat.bookCalled {
		t.Error("OnBookEvent was not called on strategy")
	}
	if !r.HasOnBookEvent() {
		t.Error("HasOnBookEvent should be true for fullStrategy")
	}
}

// --- bar_source.go coverage ---

func TestRunnerBarSource(t *testing.T) {
	bars := sdk.BarsToSlice([]sdk.Bar{
		{Open: dec("1.0"), High: dec("2.0"), Low: dec("0.5"), Close: dec("1.5"), Volume: 100, Timestamp: 1000},
		{Open: dec("1.5"), High: dec("2.5"), Low: dec("1.0"), Close: dec("2.0"), Volume: 200, Timestamp: 2000},
	})
	src := &runnerBarSource{bars: bars}
	if src.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", src.Len())
	}
	if !src.Close(0).Equal(dec("2.0")) {
		t.Errorf("Close(0) = %s, want 2.0", src.Close(0))
	}
}
