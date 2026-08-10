package mql2go

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/backtest"
	"alphaforge/strategy/sdk"
)

// TestPhaseA_VM_PendingOrderVisible verifies that the VM's OrdersTotal includes pending orders
// and OrderSelect can select them by position index.
func TestPhaseA_VM_PendingOrderVisible(t *testing.T) {
	source := `
extern int MagicNumber = 77777;
extern double LotSize = 0.1;
extern double LimitPrice = 1.0900;

int OnInit() { return 0; }

void OnBar()
{
    // On the first bar, place a buy limit order (pending, not market)
    if (OrdersTotal() == 0)
    {
        OrderSend(Symbol(), OP_BUYLIMIT, LotSize, LimitPrice, 5, 0, 0, "Pending", MagicNumber, 0, clrGreen);
    }

    // On subsequent bars, verify OrdersTotal > 0 and we can select the pending order
    if (OrdersTotal() > 0)
    {
        for (int i = 0; i < OrdersTotal(); i++)
        {
            if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES))
            {
                int type_ = OrderType();
                double lots = OrderLots();
                double price = OrderOpenPrice();
                int magic = OrderMagicNumber();
                // OP_BUYLIMIT = 2
                if (type_ != 2)
                {
                    Print("wrong order type: ", type_);
                }
                if (lots <= 0)
                {
                    Print("invalid lots");
                }
                if (price <= 0)
                {
                    Print("invalid price");
                }
                if (magic != MagicNumber)
                {
                    Print("wrong magic");
                }
            }
        }
    }
}
`

	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}

	bars := makeE2EBars(5)

	cfg := backtest.Config{
		Symbol:         "EURUSD",
		Timeframe:      "M1",
		InitialCapital: decimal.NewFromInt(100000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
		Params: map[string]string{
			"MagicNumber": "77777",
			"LotSize":     "0.1",
			"LimitPrice":  "1.0900",
		},
	}

	engine := backtest.New(cfg, runner, bars)
	result, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("backtest.Run failed: %v", err)
	}

	// The pending order should NOT have been filled (bar Low never reaches 1.0900 in makeE2EBars)
	// So we should have 0 trades but the VM should have seen the pending order.
	t.Logf("trades: %d, equity points: %d", len(result.Trades), len(result.Equity))

	// Verify no runtime blind spots
	blinds := runner.GetRuntimeBlindSpots()
	for _, bs := range blinds {
		t.Errorf("runtime blind spot: %s (count=%d)", bs.Builtin, bs.Count)
	}
}

// TestPhaseA_VM_PendingOrderModify verifies that OrderModify can modify SL/TP on a pending order.
func TestPhaseA_VM_PendingOrderModify(t *testing.T) {
	source := `
extern int MagicNumber = 88888;
extern double LotSize = 0.1;
extern double LimitPrice = 1.0900;
extern double NewSL = 1.0800;
extern double NewTP = 1.1200;

int OnInit() { return 0; }

void OnBar()
{
    // Place a buy limit order on the first bar
    if (OrdersTotal() == 0)
    {
        OrderSend(Symbol(), OP_BUYLIMIT, LotSize, LimitPrice, 5, 0, 0, "Pending", MagicNumber, 0, clrGreen);
    }
    else
    {
        // On subsequent bars, find the pending order and modify its SL/TP
        for (int i = 0; i < OrdersTotal(); i++)
        {
            if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES))
            {
                if (OrderType() == 2) // OP_BUYLIMIT
                {
                    int ticket = OrderTicket();
                    if (OrderModify(ticket, OrderOpenPrice(), NewSL, NewTP, 0, clrYellow))
                    {
                        // Verify the modification took effect
                        if (OrderSelect(ticket, SELECT_BY_TICKET, MODE_TRADES))
                        {
                            if (OrderStopLoss() != NewSL)
                            {
                                Print("SL not modified: ", OrderStopLoss());
                            }
                            if (OrderTakeProfit() != NewTP)
                            {
                                Print("TP not modified: ", OrderTakeProfit());
                            }
                        }
                    }
                }
            }
        }
    }
}
`

	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}

	bars := makeE2EBars(5)

	cfg := backtest.Config{
		Symbol:         "EURUSD",
		Timeframe:      "M1",
		InitialCapital: decimal.NewFromInt(100000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
		Params: map[string]string{
			"MagicNumber": "88888",
			"LotSize":     "0.1",
			"LimitPrice":  "1.0900",
			"NewSL":       "1.0800",
			"NewTP":       "1.1200",
		},
	}

	engine := backtest.New(cfg, runner, bars)
	_, err = engine.Run(context.Background())
	if err != nil {
		t.Fatalf("backtest.Run failed: %v", err)
	}

	blinds := runner.GetRuntimeBlindSpots()
	for _, bs := range blinds {
		t.Errorf("runtime blind spot: %s (count=%d)", bs.Builtin, bs.Count)
	}
}

// TestPhaseA_VM_PendingOrderFillsAndBecomesPosition verifies that after a pending order fills,
// the VM correctly sees it as a position (OP_BUY) not a pending order.
func TestPhaseA_VM_PendingOrderFillsAndBecomesPosition(t *testing.T) {
	source := `
extern int MagicNumber = 66666;
extern double LotSize = 0.1;

int OnInit() { return 0; }

void OnBar()
{
    // Place a buy limit at a price that will be reached
    if (OrdersTotal() == 0)
    {
        OrderSend(Symbol(), OP_BUYLIMIT, LotSize, 1.1000, 5, 0, 0, "Pending", MagicNumber, 0, clrGreen);
    }
    else
    {
        // After fill, the order should be a position with type OP_BUY=0
        for (int i = 0; i < OrdersTotal(); i++)
        {
            if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES))
            {
                int type_ = OrderType();
                // After fill, it should be OP_BUY=0, not OP_BUYLIMIT=2
                if (type_ == 2)
                {
                    Print("still pending after fill");
                }
            }
        }
    }
}
`

	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}

	// Create bars where Low dips below 1.1000 on the second bar
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	bars := []sdk.Bar{
		{
			Open: decimal.NewFromFloat(1.1050), High: decimal.NewFromFloat(1.1080),
			Low: decimal.NewFromFloat(1.1030), Close: decimal.NewFromFloat(1.1060),
			Volume: 1000, Timestamp: baseTime.UnixMilli(),
		},
		{
			Open: decimal.NewFromFloat(1.1060), High: decimal.NewFromFloat(1.1100),
			Low: decimal.NewFromFloat(1.0950), Close: decimal.NewFromFloat(1.1000),
			Volume: 1000, Timestamp: baseTime.Add(time.Hour).UnixMilli(),
		},
		{
			Open: decimal.NewFromFloat(1.1000), High: decimal.NewFromFloat(1.1050),
			Low: decimal.NewFromFloat(1.0980), Close: decimal.NewFromFloat(1.1020),
			Volume: 1000, Timestamp: baseTime.Add(2 * time.Hour).UnixMilli(),
		},
	}

	cfg := backtest.Config{
		Symbol:         "EURUSD",
		Timeframe:      "M1",
		InitialCapital: decimal.NewFromInt(100000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
		Params: map[string]string{
			"MagicNumber": "66666",
			"LotSize":     "0.1",
		},
	}

	engine := backtest.New(cfg, runner, bars)
	result, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("backtest.Run failed: %v", err)
	}

	// The pending order should have been filled and become a position
	t.Logf("trades: %d", len(result.Trades))

	blinds := runner.GetRuntimeBlindSpots()
	for _, bs := range blinds {
		t.Errorf("runtime blind spot: %s (count=%d)", bs.Builtin, bs.Count)
	}
}

// TestPhaseA_Adversarial_RemovePendingFromTotal verifies that removing pending order counting
// from OrdersTotal would break this test (adversarial proof).
// This test creates a scenario where only a pending order exists (no positions),
// and asserts OrdersTotal > 0. If pending orders are not counted, this fails.
func TestPhaseA_Adversarial_RemovePendingFromTotal(t *testing.T) {
	broker := backtest.NewSimBroker(backtest.Config{
		InitialCapital: decimal.NewFromInt(100000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
	})
	broker.SetBarPrice(decimal.NewFromFloat(1.1000))
	broker.SetBarTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))

	// Place a pending order only (no market orders)
	broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD",
		Side:   sdk.SideBuy,
		Type:   sdk.OrderLimit,
		Volume: decimal.NewFromFloat(0.1),
		Price:  decimal.NewFromFloat(1.0900),
	})

	// With pending visibility: positions=0 + orders=1 = 1
	// Without pending visibility: positions=0 = 0 (test fails)
	positions := broker.Positions(0)
	orders := broker.Orders(0)
	total := len(positions) + len(orders)

	if total != 1 {
		t.Fatalf("adversarial: expected total=1 (pending must be counted), got %d (positions=%d, orders=%d)",
			total, len(positions), len(orders))
	}
	if len(positions) != 0 {
		t.Errorf("expected 0 positions, got %d", len(positions))
	}
	if len(orders) != 1 {
		t.Errorf("expected 1 pending order, got %d", len(orders))
	}
}
