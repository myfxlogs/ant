package backtest

import (
	"context"
	"math"
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
	mql2go "alphaforge/tools/mql2go"
)

// D3 Production Metamorphic Tests
//
// These tests verify metamorphic relations through the FULL production pipeline:
// MQL source → CompileMQL → VMRunner → backtest.Engine → Result
//
// Unlike d3_metamorphic_test.go which tests reference implementations,
// these tests compile and execute actual MQL4 EAs to verify that the
// production pipeline satisfies mathematical invariants.

// d3MakeConfig creates a standard backtest config for metamorphic tests.
func d3MakeConfig() Config {
	return Config{
		Symbol:         "EURUSD",
		Timeframe:      "H1",
		InitialCapital: decimal.NewFromInt(10000),
		Leverage:       100,
		SymbolDigits:   5,
		SymbolPoint:    decimal.NewFromFloat(0.00001),
		ContractSize:   decimal.NewFromInt(100000),
	}
}

// ── MR-P1: iMA period=1 identity ─────────────────────────────────────
// iMA(NULL,0,1,0,MODE_SMA,PRICE_CLOSE,0) should equal Close[0].
// SMA with period 1 is just the close price itself.

func TestD3_MR_P1_MA_Period1_Identity(t *testing.T) {
	const ea = `#property strict
double g_ma;
double g_close;

int OnInit() { return(INIT_SUCCEEDED); }

void OnTick() {
    g_ma = iMA(Symbol(), 0, 1, 0, MODE_SMA, PRICE_CLOSE, 0);
    g_close = iClose(Symbol(), 0, 0);
}
`
	bars := d3GenerateBars(50)
	runner, err := mql2go.CompileMQL(ea)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}

	engine := New(d3MakeConfig(), runner, bars)
	if _, err := engine.Run(context.Background()); err != nil {
		t.Fatalf("Engine.Run: %v", err)
	}

	maVal, ok := runner.GetGlobal("g_ma")
	if !ok {
		t.Fatal("GetGlobal(g_ma) not found")
	}
	closeVal, ok := runner.GetGlobal("g_close")
	if !ok {
		t.Fatal("GetGlobal(g_close) not found")
	}

	got := maVal.Decimal.InexactFloat64()
	want := closeVal.Decimal.InexactFloat64()
	if math.Abs(got-want) > 1e-6 {
		t.Errorf("MA(1) identity: got %v, want %v (diff %v)", got, want, math.Abs(got-want))
	}
}

// ── MR-P2: Trade PnL identity ────────────────────────────────────────
// For a BUY: PnL = (exitPrice - entryPrice) * volume * contractSize
// For a SELL: PnL = (entryPrice - exitPrice) * volume * contractSize

func TestD3_MR_P2_TradePnL_Identity(t *testing.T) {
	const ea = `#property strict
extern double Lots = 0.1;

int OnInit() { return(INIT_SUCCEEDED); }

void OnTick() {
    if (OrdersTotal() == 0 && Bars > 10) {
        OrderSend(Symbol(), OP_BUY, Lots, Close[0], 0, 0, 0, "MR-P2", 0, 0, clrGreen);
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
}
`
	bars := d3GenerateBars(30)
	cfg := d3MakeConfig()
	runner, err := mql2go.CompileMQL(ea)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}

	engine := New(cfg, runner, bars)
	result, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("Engine.Run: %v", err)
	}

	if len(result.Trades) == 0 {
		t.Skip("No trades produced — skipping PnL identity check")
	}

	for i, trade := range result.Trades {
		priceDiff := trade.ExitPrice.Sub(trade.EntryPrice)
		if trade.Side == sdk.SideSell {
			priceDiff = trade.EntryPrice.Sub(trade.ExitPrice)
		}
		expectedPnL := priceDiff.Mul(trade.Volume).Mul(cfg.ContractSize)
		actualPnL := trade.Profit

		diff := actualPnL.Sub(expectedPnL).InexactFloat64()
		if math.Abs(diff) > 0.01 {
			t.Errorf("Trade %d PnL identity: actual=%s, expected=%s (diff=%v)",
				i, actualPnL.String(), expectedPnL.String(), diff)
		}
	}
}

// ── MR-P3: iHighest boundary constraint ──────────────────────────────
// iHighest returns the index of the highest bar in a range.
// The high price at that index must be >= all other highs in the range.

func TestD3_MR_P3_iHighest_Boundary(t *testing.T) {
	const ea = `#property strict
int g_idx;
double g_high;

int OnInit() { return(INIT_SUCCEEDED); }

void OnTick() {
    g_idx = iHighest(Symbol(), 0, MODE_HIGH, 10, 0);
    if (g_idx >= 0) {
        g_high = iHigh(Symbol(), 0, g_idx);
    }
}
`
	bars := d3GenerateBars(50)
	runner, err := mql2go.CompileMQL(ea)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}

	engine := New(d3MakeConfig(), runner, bars)
	if _, err := engine.Run(context.Background()); err != nil {
		t.Fatalf("Engine.Run: %v", err)
	}

	idxVal, ok := runner.GetGlobal("g_idx")
	if !ok {
		t.Fatal("GetGlobal(g_idx) not found")
	}
	highVal, ok := runner.GetGlobal("g_high")
	if !ok {
		t.Fatal("GetGlobal(g_high) not found")
	}

	idx := int(idxVal.Int)
	if idx < 0 {
		t.Skip("iHighest returned -1")
	}

	// In SDK convention, shift=0 is the latest bar.
	// iHighest with count=10, shift=0 looks at the last 10 bars.
	// The returned index is a shift value (0-based from the end).
	highAtIdx := highVal.Decimal.InexactFloat64()

	// Verify that the high at the returned index is >= all other highs in the 10-bar window.
	for s := 0; s < 10; s++ {
		barIdx := len(bars) - 1 - s
		if barIdx < 0 {
			break
		}
		barHigh := bars[barIdx].High.InexactFloat64()
		if barHigh > highAtIdx+1e-10 {
			t.Errorf("iHighest boundary: bar at shift %d has high %v > returned high %v",
				s, barHigh, highAtIdx)
		}
	}
}

// ── MR-P4: BUY/SELL PnL symmetry ─────────────────────────────────────
// A BUY and SELL with identical parameters should have PnL that are
// exact negatives of each other (ignoring commission).

func TestD3_MR_P4_BuySell_Symmetry(t *testing.T) {
	const buyEA = `#property strict
extern double Lots = 0.1;

int OnInit() { return(INIT_SUCCEEDED); }

void OnTick() {
    if (OrdersTotal() == 0 && Bars > 5) {
        OrderSend(Symbol(), OP_BUY, Lots, Close[0], 0, 0, 0, "MR-P4-BUY", 0, 0, clrGreen);
    }
    if (OrdersTotal() > 0 && Bars > 15) {
        int total = OrdersTotal();
        for (int i = 0; i < total; i++) {
            if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES)) {
                OrderClose(OrderTicket(), OrderLots(), Close[0], 0, clrRed);
                break;
            }
        }
    }
}
`
	const sellEA = `#property strict
extern double Lots = 0.1;

int OnInit() { return(INIT_SUCCEEDED); }

void OnTick() {
    if (OrdersTotal() == 0 && Bars > 5) {
        OrderSend(Symbol(), OP_SELL, Lots, Close[0], 0, 0, 0, "MR-P4-SELL", 0, 0, clrRed);
    }
    if (OrdersTotal() > 0 && Bars > 15) {
        int total = OrdersTotal();
        for (int i = 0; i < total; i++) {
            if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES)) {
                OrderClose(OrderTicket(), OrderLots(), Close[0], 0, clrGreen);
                break;
            }
        }
    }
}
`
	bars := d3GenerateBars(20)
	cfg := d3MakeConfig()
	cfg.Commission = decimal.Zero // exclude commission for symmetry check

	// Run BUY
	buyRunner, err := mql2go.CompileMQL(buyEA)
	if err != nil {
		t.Fatalf("CompileMQL(buy): %v", err)
	}
	buyResult, err := New(cfg, buyRunner, bars).Run(context.Background())
	if err != nil {
		t.Fatalf("BUY Engine.Run: %v", err)
	}

	// Run SELL
	sellRunner, err := mql2go.CompileMQL(sellEA)
	if err != nil {
		t.Fatalf("CompileMQL(sell): %v", err)
	}
	sellResult, err := New(cfg, sellRunner, bars).Run(context.Background())
	if err != nil {
		t.Fatalf("SELL Engine.Run: %v", err)
	}

	if len(buyResult.Trades) == 0 || len(sellResult.Trades) == 0 {
		t.Skip("No trades produced")
	}

	buyPnL := buyResult.Trades[0].Profit
	sellPnL := sellResult.Trades[0].Profit
	sum := buyPnL.Add(sellPnL)

	if math.Abs(sum.InexactFloat64()) > 0.01 {
		t.Errorf("BUY/SELL symmetry: BUY PnL=%s, SELL PnL=%s, sum=%s (should be ~0)",
			buyPnL.String(), sellPnL.String(), sum.String())
	}
}

// ── MR-P5: Lots linearity ────────────────────────────────────────────
// Doubling the lot size should double the PnL (linear scaling).

func TestD3_MR_P5_Lots_Linearity(t *testing.T) {
	const ea1 = `#property strict
extern double Lots = 0.1;

int OnInit() { return(INIT_SUCCEEDED); }

void OnTick() {
    if (OrdersTotal() == 0 && Bars > 5) {
        OrderSend(Symbol(), OP_BUY, Lots, Close[0], 0, 0, 0, "MR-P5-1", 0, 0, clrGreen);
    }
    if (OrdersTotal() > 0 && Bars > 15) {
        int total = OrdersTotal();
        for (int i = 0; i < total; i++) {
            if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES)) {
                OrderClose(OrderTicket(), OrderLots(), Close[0], 0, clrRed);
                break;
            }
        }
    }
}
`
	const ea2 = `#property strict
extern double Lots = 0.2;

int OnInit() { return(INIT_SUCCEEDED); }

void OnTick() {
    if (OrdersTotal() == 0 && Bars > 5) {
        OrderSend(Symbol(), OP_BUY, Lots, Close[0], 0, 0, 0, "MR-P5-2", 0, 0, clrGreen);
    }
    if (OrdersTotal() > 0 && Bars > 15) {
        int total = OrdersTotal();
        for (int i = 0; i < total; i++) {
            if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES)) {
                OrderClose(OrderTicket(), OrderLots(), Close[0], 0, clrRed);
                break;
            }
        }
    }
}
`
	bars := d3GenerateBars(20)
	cfg := d3MakeConfig()
	cfg.Commission = decimal.Zero // exclude commission for linearity check

	// Run with 0.1 lots
	runner1, err := mql2go.CompileMQL(ea1)
	if err != nil {
		t.Fatalf("CompileMQL(ea1): %v", err)
	}
	result1, err := New(cfg, runner1, bars).Run(context.Background())
	if err != nil {
		t.Fatalf("Engine.Run(0.1): %v", err)
	}

	// Run with 0.2 lots
	runner2, err := mql2go.CompileMQL(ea2)
	if err != nil {
		t.Fatalf("CompileMQL(ea2): %v", err)
	}
	result2, err := New(cfg, runner2, bars).Run(context.Background())
	if err != nil {
		t.Fatalf("Engine.Run(0.2): %v", err)
	}

	if len(result1.Trades) == 0 || len(result2.Trades) == 0 {
		t.Skip("No trades produced")
	}

	pnL1 := result1.Trades[0].Profit
	pnL2 := result2.Trades[0].Profit

	// pnL2 should be approximately 2x pnL1
	ratio := pnL2.Div(pnL1).InexactFloat64()
	if math.Abs(ratio-2.0) > 0.01 {
		t.Errorf("Lots linearity: PnL(0.2)=%s, PnL(0.1)=%s, ratio=%v (expected ~2.0)",
			pnL2.String(), pnL1.String(), ratio)
	}
}
