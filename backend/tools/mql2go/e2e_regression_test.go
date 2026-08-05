package mql2go

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/backtest"
)

// TestE2E_ClassicStartEntry verifies that MQL4's legacy start() function
// is correctly mapped to OnTick and executes in backtest.
func TestE2E_ClassicStartEntry(t *testing.T) {
	source := `
extern int MagicNumber = 11111;
extern double LotSize = 0.1;

int start()
{
    if (OrdersTotal() == 0)
    {
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "Start", MagicNumber, 0, clrGreen);
    }
    return 0;
}
`

	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL failed (start() should map to OnTick): %v", err)
	}

	bars := makeE2EBars(30)

	cfg := backtest.Config{
		Symbol:         "EURUSD",
		Timeframe:      "M1",
		InitialCapital: decimal.NewFromInt(10000),
		Leverage:       100,
		Params: map[string]string{
			"MagicNumber": "11111",
			"LotSize":     "0.1",
		},
	}

	engine := backtest.New(cfg, runner, bars)
	result, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("backtest.Run failed: %v", err)
	}

	if len(result.Equity) == 0 {
		t.Fatal("expected equity points — start() should have executed")
	}

	t.Logf("start() entry: trades=%d, equity points=%d", len(result.Trades), len(result.Equity))
}

// TestE2E_OrderSelectHistory verifies that OrderSelect with MODE_HISTORY
// correctly iterates closed orders after a position is opened and closed.
func TestE2E_OrderSelectHistory(t *testing.T) {
	source := `
extern int MagicNumber = 22222;
extern double LotSize = 0.1;

double g_closedPrice;
double g_closedTime;

int OnInit() { return 0; }

void OnBar()
{
    // Open a position on the first bar
    if (OrdersTotal() == 0 && OrdersHistoryTotal() == 0)
    {
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "Hist", MagicNumber, 0, clrGreen);
        return;
    }

    // Close the position on the next bar
    if (OrdersTotal() > 0)
    {
        for (int i = 0; i < OrdersTotal(); i++)
        {
            if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES))
            {
                OrderClose(OrderTicket(), OrderLots(), Bid, 5);
            }
        }
        return;
    }

    // After closing, iterate history pool
    if (OrdersHistoryTotal() > 0)
    {
        for (int i = 0; i < OrdersHistoryTotal(); i++)
        {
            if (OrderSelect(i, SELECT_BY_POS, MODE_HISTORY))
            {
                g_closedPrice = OrderClosePrice();
                g_closedTime = OrderCloseTime();
            }
        }
    }
}
`

	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}

	bars := makeE2EBars(40)

	cfg := backtest.Config{
		Symbol:         "EURUSD",
		Timeframe:      "M1",
		InitialCapital: decimal.NewFromInt(10000),
		Leverage:       100,
		Params: map[string]string{
			"MagicNumber": "22222",
			"LotSize":     "0.1",
		},
	}

	engine := backtest.New(cfg, runner, bars)
	result, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("backtest.Run failed: %v", err)
	}

	t.Logf("MODE_HISTORY: trades=%d, equity points=%d", len(result.Trades), len(result.Equity))

	// Verify no blind spots for Order* history functions
	blinds := runner.GetRuntimeBlindSpots()
	for _, bs := range blinds {
		if bs.Builtin == "OrdersHistoryTotal" || bs.Builtin == "OrderSelect" ||
			bs.Builtin == "OrderClosePrice" || bs.Builtin == "OrderCloseTime" {
			t.Errorf("unexpected blind spot for history function: %s (count=%d)", bs.Builtin, bs.Count)
		}
	}
}

// TestE2E_IADX_Mode verifies that iADX mode parameter is handled correctly:
// MODE_MAIN returns ADX value (no blind spot), MODE_PLUSDI/MODE_MINUSDI
// return 0 and record blind spots.
func TestE2E_IADX_Mode(t *testing.T) {
	source := `
extern int ADXPeriod = 14;

double g_adxMain;
double g_adxPlusDI;
double g_adxMinusDI;

int OnInit() { return 0; }

void OnBar()
{
    g_adxMain = iADX(NULL, 0, ADXPeriod, PRICE_CLOSE, MODE_MAIN, 0);
    g_adxPlusDI = iADX(NULL, 0, ADXPeriod, PRICE_CLOSE, MODE_PLUSDI, 0);
    g_adxMinusDI = iADX(NULL, 0, ADXPeriod, PRICE_CLOSE, MODE_MINUSDI, 0);
}
`

	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL failed (MODE_MAIN/PLUSDI/MINUSDI should be known constants): %v", err)
	}

	bars := makeE2EBars(50)

	cfg := backtest.Config{
		Symbol:         "EURUSD",
		Timeframe:      "M1",
		InitialCapital: decimal.NewFromInt(10000),
		Leverage:       100,
		Params: map[string]string{
			"ADXPeriod": "14",
		},
	}

	engine := backtest.New(cfg, runner, bars)
	_, err = engine.Run(context.Background())
	if err != nil {
		t.Fatalf("backtest.Run failed: %v", err)
	}

	blinds := runner.GetRuntimeBlindSpots()

	// MODE_MAIN should NOT produce a blind spot
	// MODE_PLUSDI and MODE_MINUSDI SHOULD produce blind spots
	hasPlusDI := false
	hasMinusDI := false
	for _, bs := range blinds {
		if bs.Builtin == "iADX:MODE_PLUSDI" {
			hasPlusDI = true
		}
		if bs.Builtin == "iADX:MODE_MINUSDI" {
			hasMinusDI = true
		}
		if bs.Builtin == "iADX:MODE_MAIN" {
			t.Error("MODE_MAIN should not produce a blind spot — it returns the ADX line value")
		}
	}

	if !hasPlusDI {
		t.Error("expected blind spot for iADX:MODE_PLUSDI")
	}
	if !hasMinusDI {
		t.Error("expected blind spot for iADX:MODE_MINUSDI")
	}
}
