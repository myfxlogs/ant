package backtest

import (
	"context"
	"testing"

	mql2go "alphaforge/tools/mql2go"
)

// TestBTVolume_NewBarGuard_ProducesTrades verifies that the common MQL4
// new-bar guard `if(Volume[0]>1) return;` works correctly in bar-based
// backtest. In MT4 "Open prices only" mode, the current bar starts with
// Volume=1 (first tick), so the guard passes and the EA trades.
//
// Without the btBarSeries Volume(0)=1 override, Volume[0] returns the bar's
// full tick volume (1000+), the guard always returns, and the EA never trades.
//
// Regression: backtest 95fbd896 (Moving Average EA) SUCCEEDED with 0 trades
// because Volume[0]>1 was always true.
//
// Adversarial proof: remove the Volume(0)=1 override in btBarSeries →
// Volume[0] returns 1000+ → guard returns → no OrderSend → 0 trades → RED.
func TestBTVolume_NewBarGuard_ProducesTrades(t *testing.T) {
	// EA uses the Volume[0]>1 new-bar guard (standard MetaQuotes pattern),
	// opens a BUY when no positions, closes when Bars > 20.
	const ea = `#property strict
extern double Lots = 0.1;
int g_ticket = -1;

int OnInit() { return(INIT_SUCCEEDED); }

void OnTick() {
    if (Volume[0] > 1) return;
    if (Bars < 10) return;
    if (OrdersTotal() == 0) {
        g_ticket = OrderSend(Symbol(), OP_BUY, Lots, Close[0], 0, 0, 0, "VOL-GUARD", 0, 0, clrGreen);
    }
    if (OrdersTotal() > 0 && Bars > 20) {
        int total = OrdersTotal();
        for (int i = 0; i < total; i++) {
            if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES)) {
                OrderClose(OrderTicket(), OrderLots(), Close[0], 0, clrRed);
                break;
            }
        }
    }
}`
	bars := d3GenerateBars(30)
	// d3GenerateBars sets Volume = 1000+i%500, so without the fix
	// Volume[0] > 1 is always true → no trades.

	runner, err := mql2go.CompileMQL(ea)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}

	cfg := d3MakeConfig()
	engine := New(cfg, runner, bars)
	result, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("Engine.Run: %v", err)
	}

	if len(result.Trades) == 0 {
		t.Fatalf("expected trades with Volume[0]>1 guard, got 0 — Volume(0) override not working")
	}
	t.Logf("produced %d trades with Volume[0]>1 new-bar guard", len(result.Trades))
}

// TestBTVolume_CurrentBarVolumeIsOne directly verifies that the backtest
// bar series returns Volume=1 for shift=0 (current bar) and the actual
// volume for shift>0 (historical bars).
func TestBTVolume_CurrentBarVolumeIsOne(t *testing.T) {
	bars := d3GenerateBars(20)
	// d3GenerateBars sets Volume = 1000+i%500, all > 1.

	btCtx := &backtestContext{
		bars:     bars,
		barIndex: len(bars) - 1,
	}
	bs := btCtx.Bars()

	if v := bs.Volume(0); v != 1 {
		t.Errorf("Volume(0) = %d, want 1 (MT4 Open prices only: current bar just opened)", v)
	}
	// Historical bar should keep its actual volume.
	if v := bs.Volume(1); v == 1 {
		t.Errorf("Volume(1) = %d, expected actual bar volume (not 1)", v)
	}
	if v := bs.Volume(1); v != bars[len(bars)-2].Volume {
		t.Errorf("Volume(1) = %d, want %d (actual historical bar volume)", v, bars[len(bars)-2].Volume)
	}
}

// TestBTVolume_NoGuard_AlwaysTrades is a control: without the Volume[0]>1
// guard, the EA trades — confirming the guard is what prevents trading
// (not some other issue).
func TestBTVolume_NoGuard_AlwaysTrades(t *testing.T) {
	const ea = `#property strict
extern double Lots = 0.1;

int OnInit() { return(INIT_SUCCEEDED); }

void OnTick() {
    if (Bars < 10) return;
    if (OrdersTotal() == 0) {
        OrderSend(Symbol(), OP_BUY, Lots, Close[0], 0, 0, 0, "NO-GUARD", 0, 0, clrGreen);
    }
    if (OrdersTotal() > 0 && Bars > 20) {
        int total = OrdersTotal();
        for (int i = 0; i < total; i++) {
            if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES)) {
                OrderClose(OrderTicket(), OrderLots(), Close[0], 0, clrRed);
                break;
            }
        }
    }
}`
	bars := d3GenerateBars(30)
	runner, err := mql2go.CompileMQL(ea)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}

	cfg := d3MakeConfig()
	engine := New(cfg, runner, bars)
	result, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("Engine.Run: %v", err)
	}

	if len(result.Trades) == 0 {
		t.Fatalf("expected trades without Volume guard, got 0 — test setup is broken")
	}
	t.Logf("control: %d trades without Volume guard (confirming setup validity)", len(result.Trades))
}
