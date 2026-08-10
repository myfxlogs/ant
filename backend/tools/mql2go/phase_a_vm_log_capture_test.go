package mql2go

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/backtest"
	"alphaforge/strategy/sdk"
)

// TestPhaseA_VM_LogCapture_OrdersTotal verifies that the VM's OrdersTotal
// includes pending orders, using log capture (not just blind-spot checking).
// The MQL EA prints "orders_total=N" which is captured in Result.Logs.
// Adversarial proof: if +len(vm.cachedOrders) is removed from builtinOrdersTotal,
// the log would show "orders_total=0" and this test would fail.
func TestPhaseA_VM_LogCapture_OrdersTotal(t *testing.T) {
	source := `
extern double LotSize = 0.1;
extern double LimitPrice = 1.0900;

int OnInit() { return 0; }

void OnBar()
{
    if (OrdersTotal() == 0)
    {
        OrderSend(Symbol(), OP_BUYLIMIT, LotSize, LimitPrice, 5, 0, 0, "Pending", 12345, 0, clrGreen);
    }

    // Print the count so we can verify via log capture
    Print("orders_total=", OrdersTotal());

    // Iterate and print each order's type
    for (int i = 0; i < OrdersTotal(); i++)
    {
        if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES))
        {
            Print("order_type=", OrderType());
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
			"LotSize":    "0.1",
			"LimitPrice": "1.0900",
		},
	}

	engine := backtest.New(cfg, runner, bars)
	result, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("backtest.Run failed: %v", err)
	}

	// Find logs that contain "orders_total=" after the first bar (where order is placed)
	// MQL Print joins args with space: Print("orders_total=", N) → "orders_total= N"
	foundOrdersTotal := false
	foundOrderType := false
	for _, log := range result.Logs {
		if strings.Contains(log, "orders_total= 1") {
			foundOrdersTotal = true
		}
		if strings.Contains(log, "order_type= 2") {
			// OP_BUYLIMIT = 2
			foundOrderType = true
		}
	}

	if !foundOrdersTotal {
		t.Errorf("expected log containing 'orders_total=1' (pending order must be counted), logs: %v", result.Logs)
	}
	if !foundOrderType {
		t.Errorf("expected log containing 'order_type=2' (OP_BUYLIMIT), logs: %v", result.Logs)
	}
}

// TestPhaseA_VM_LogCapture_Adversarial verifies that if pending orders were NOT
// counted in OrdersTotal, the log would show "orders_total=0" after placing a pending order.
// This is the adversarial proof: the test asserts orders_total=1, so removing
// +len(vm.cachedOrders) from builtinOrdersTotal would make it fail.
func TestPhaseA_VM_LogCapture_Adversarial(t *testing.T) {
	source := `
extern double LotSize = 0.1;
extern double LimitPrice = 1.0900;

int OnInit() { return 0; }

void OnBar()
{
    if (OrdersTotal() == 0)
    {
        OrderSend(Symbol(), OP_BUYLIMIT, LotSize, LimitPrice, 5, 0, 0, "Pending", 12345, 0, clrGreen);
    }
    // After placing, OrdersTotal must be > 0
    if (OrdersTotal() > 0)
    {
        Print("pending_visible=true");
    }
    else
    {
        Print("pending_visible=false");
    }
}
`

	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}

	// Bars where Low never reaches 1.0900 → pending order stays pending
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	bars := []sdk.Bar{
		{Open: decimal.NewFromFloat(1.1050), High: decimal.NewFromFloat(1.1080),
			Low: decimal.NewFromFloat(1.1030), Close: decimal.NewFromFloat(1.1060),
			Volume: 1000, Timestamp: baseTime.UnixMilli()},
		{Open: decimal.NewFromFloat(1.1060), High: decimal.NewFromFloat(1.1090),
			Low: decimal.NewFromFloat(1.1040), Close: decimal.NewFromFloat(1.1070),
			Volume: 1000, Timestamp: baseTime.Add(time.Hour).UnixMilli()},
		{Open: decimal.NewFromFloat(1.1070), High: decimal.NewFromFloat(1.1100),
			Low: decimal.NewFromFloat(1.1050), Close: decimal.NewFromFloat(1.1080),
			Volume: 1000, Timestamp: baseTime.Add(2 * time.Hour).UnixMilli()},
	}

	cfg := backtest.Config{
		Symbol:         "EURUSD",
		Timeframe:      "M1",
		InitialCapital: decimal.NewFromInt(100000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
		Params: map[string]string{
			"LotSize":    "0.1",
			"LimitPrice": "1.0900",
		},
	}

	engine := backtest.New(cfg, runner, bars)
	result, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("backtest.Run failed: %v", err)
	}

	foundVisible := false
	for _, log := range result.Logs {
		if strings.Contains(log, "pending_visible=true") {
			foundVisible = true
		}
		// If we see pending_visible=false after placing an order, that's a bug
		if strings.Contains(log, "pending_visible=false") {
			t.Errorf("adversarial: pending order not visible in OrdersTotal (logs: %v)", result.Logs)
		}
	}
	if !foundVisible {
		t.Errorf("expected log 'pending_visible=true' (pending order must be counted in OrdersTotal), logs: %v", result.Logs)
	}
}
