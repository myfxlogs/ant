package mql2go

// LIVE-MQL-ORDER-CONTEXT-1 Rework: Real VM/MQL integration tests.
//
// These tests compile actual MQL source code, set up a live harness Runner
// with positions + pending orders (via UpdateLiveState), execute the MQL
// code through the VM, and verify that OrdersTotal/OrderSelect/OrderMagicNumber/
// OrderType/OrderSymbol return correct broker-original values.
//
// Adversarial acceptance criteria from builder-handoff-2026-08-21.md:
//   - buy/sell/buy_limit/sell_stop → Positions=2, Orders=2, OrdersTotal=4
//   - magic=1699507621 must reach OrderMagicNumber end-to-end through the VM
//   - deleting any MQL builtin mapping must make tests RED
//
// The previous round's tests only checked Go-layer broker.Positions/Orders
// and manual len(positions)+len(orders). Changing builtinOrderMagicNumber
// to return 0 did NOT make them RED. These tests fix that by executing
// real MQL code through the VM.

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/runner"
	"alphaforge/strategy/sdk"
)

// vmTestMagic is the magic number used across all VM integration tests.
// Matches the builder handoff spec: magic=1699507621 must reach OrderMagicNumber.
const vmTestMagic int32 = 1699507621

// vmTestSource is MQL4 source that reads order context via VM builtins
// and stores results in global variables for the test to read.
const vmTestSource = `
int g_total = -1;
int g_magic0 = -1;
int g_magic1 = -1;
int g_magic2 = -1;
int g_magic3 = -1;
int g_type0 = -1;
int g_type1 = -1;
int g_type2 = -1;
int g_type3 = -1;
string g_symbol0 = "";
string g_symbol1 = "";
string g_symbol2 = "";
string g_symbol3 = "";
double g_lots0 = 0;
double g_lots2 = 0;
double g_openPrice0 = 0;
double g_openPrice2 = 0;
double g_sl0 = 0;
double g_tp0 = 0;
string g_comment0 = "";

int OnInit() { return 0; }

void OnTick()
{
    g_total = OrdersTotal();

    for (int i = 0; i < OrdersTotal(); i++)
    {
        if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES))
        {
            int magic = OrderMagicNumber();
            int type_ = OrderType();
            string sym = OrderSymbol();
            double lots = OrderLots();
            double price = OrderOpenPrice();
            double sl = OrderStopLoss();
            double tp = OrderTakeProfit();
            string cmt = OrderComment();

            if (i == 0) {
                g_magic0 = magic; g_type0 = type_; g_symbol0 = sym;
                g_lots0 = lots; g_openPrice0 = price; g_sl0 = sl; g_tp0 = tp;
                g_comment0 = cmt;
            }
            if (i == 1) { g_magic1 = magic; g_type1 = type_; g_symbol1 = sym; }
            if (i == 2) {
                g_magic2 = magic; g_type2 = type_; g_symbol2 = sym;
                g_lots2 = lots; g_openPrice2 = price;
            }
            if (i == 3) { g_magic3 = magic; g_type3 = type_; g_symbol3 = sym; }
        }
    }
}
`

// setupVMTestRunner creates a Runner with live harness state containing
// 2 market positions (buy/sell) + 2 pending orders (buy_limit/sell_stop),
// all with vmTestMagic. Returns the runner and the compiled VMRunner.
func setupVMTestRunner(t *testing.T) (*runner.Runner, *VMRunner) {
	t.Helper()

	vmRunner, err := CompileMQL(vmTestSource)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}

	r := runner.New(runner.Config{})
	r.SetStrategy(vmRunner)

	// Set up live harness state with 2 positions + 2 pending orders
	now := time.Now()
	positions := []sdk.Position{
		{
			Ticket: 1001, Symbol: "EURUSD", Side: sdk.SideBuy, Magic: vmTestMagic,
			Volume: decimal.NewFromFloat(0.1), OpenPrice: decimal.NewFromFloat(1.1),
			StopLoss: decimal.NewFromFloat(1.0), TakeProfit: decimal.NewFromFloat(1.2),
			Profit: decimal.NewFromFloat(50), Comment: "test-buy", OpenTime: now,
		},
		{
			Ticket: 1002, Symbol: "GBPUSD", Side: sdk.SideSell, Magic: vmTestMagic,
			Volume: decimal.NewFromFloat(0.2), OpenPrice: decimal.NewFromFloat(1.3),
			StopLoss: decimal.NewFromFloat(1.4), TakeProfit: decimal.NewFromFloat(1.2),
			Profit: decimal.NewFromFloat(-30), Comment: "test-sell", OpenTime: now,
		},
	}
	pendingOrders := []sdk.PendingOrder{
		{
			Ticket: 2001, Symbol: "EURUSD", Type: sdk.OrderLimit, Side: sdk.SideBuy, Magic: vmTestMagic,
			Volume: decimal.NewFromFloat(0.1), Price: decimal.NewFromFloat(1.05),
			StopLoss: decimal.NewFromFloat(0.95), TakeProfit: decimal.NewFromFloat(1.15),
			Comment: "test-buy-limit", OpenTime: now,
		},
		{
			Ticket: 2002, Symbol: "GBPUSD", Type: sdk.OrderStop, Side: sdk.SideSell, Magic: vmTestMagic,
			Volume: decimal.NewFromFloat(0.15), Price: decimal.NewFromFloat(1.25),
			StopLoss: decimal.NewFromFloat(1.35), TakeProfit: decimal.NewFromFloat(1.15),
			Comment: "test-sell-stop", OpenTime: now,
		},
	}

	r.UpdateLiveState("10000", "10500", "500", "9500", positions, pendingOrders)

	return r, vmRunner
}

// runVMTestTick executes one OnTick on the runner and returns the VMRunner
// so globals can be read.
func runVMTestTick(t *testing.T, r *runner.Runner, vm *VMRunner) *VMRunner {
	t.Helper()
	ctx := context.Background()
	_, err := r.OnTick(ctx, decimal.NewFromFloat(1.1), decimal.NewFromFloat(1.1001))
	if err != nil {
		t.Fatalf("OnTick failed: %v", err)
	}
	return vm
}

// getGlobalInt reads a global int variable from the VMRunner.
func getGlobalInt(t *testing.T, vm *VMRunner, name string) int32 {
	t.Helper()
	v, ok := vm.GetGlobal(name)
	if !ok {
		t.Fatalf("global %q not found", name)
	}
	return v.ToInt()
}

// getGlobalString reads a global string variable from the VMRunner.
func getGlobalString(t *testing.T, vm *VMRunner, name string) string {
	t.Helper()
	v, ok := vm.GetGlobal(name)
	if !ok {
		t.Fatalf("global %q not found", name)
	}
	return v.ToString()
}

// getGlobalDecimal reads a global decimal variable from the VMRunner.
func getGlobalDecimal(t *testing.T, vm *VMRunner, name string) decimal.Decimal {
	t.Helper()
	v, ok := vm.GetGlobal(name)
	if !ok {
		t.Fatalf("global %q not found", name)
	}
	return v.ToDecimal()
}

// TestLIVE_MQL_VM_OrdersTotal_PositionsPlusPending verifies that the VM's
// OrdersTotal() returns 4 (2 market positions + 2 pending orders) when
// the live harness has 2 positions and 2 pending orders.
//
// Adversarial: delete pending orders from brokerImpl.Orders harness →
// g_total becomes 2 instead of 4 → test RED.
func TestLIVE_MQL_VM_OrdersTotal_PositionsPlusPending(t *testing.T) {
	r, vmRunner := setupVMTestRunner(t)
	vm := runVMTestTick(t, r, vmRunner)

	total := getGlobalInt(t, vm, "g_total")
	if total != 4 {
		t.Fatalf("VM OrdersTotal = %d, want 4 (2 positions + 2 pending orders)", total)
	}

	// Verify no runtime blind spots (unknown builtins returning 0)
	for _, bs := range vm.GetRuntimeBlindSpots() {
		t.Errorf("runtime blind spot: %s (count=%d)", bs.Builtin, bs.Count)
	}
}

// TestLIVE_MQL_VM_OrderMagicNumber_EndToEnd verifies that magic=1699507621
// from the live harness state reaches OrderMagicNumber() through the VM
// for ALL 4 orders (2 positions + 2 pending).
//
// Adversarial: delete Magic mapping from vmPositionsToSdk OR change
// builtinOrderMagicNumber to return 0 → g_magic0-3 become 0 → test RED.
func TestLIVE_MQL_VM_OrderMagicNumber_EndToEnd(t *testing.T) {
	r, vmRunner := setupVMTestRunner(t)
	vm := runVMTestTick(t, r, vmRunner)

	for i := 0; i < 4; i++ {
		var name string
		switch i {
		case 0:
			name = "g_magic0"
		case 1:
			name = "g_magic1"
		case 2:
			name = "g_magic2"
		case 3:
			name = "g_magic3"
		}
		magic := getGlobalInt(t, vm, name)
		if magic != vmTestMagic {
			t.Fatalf("OrderMagicNumber[%d] = %d, want %d (magic lost in VM chain)", i, magic, vmTestMagic)
		}
	}
}

// TestLIVE_MQL_VM_OrderType_MarketVsPending verifies that OrderType()
// returns correct MQL4 OP_* constants:
//   - Position 0 (buy): OP_BUY = 0
//   - Position 1 (sell): OP_SELL = 1
//   - Pending 0 (buy_limit): OP_BUYLIMIT = 2
//   - Pending 1 (sell_stop): OP_SELLSTOP = 5
func TestLIVE_MQL_VM_OrderType_MarketVsPending(t *testing.T) {
	r, vmRunner := setupVMTestRunner(t)
	vm := runVMTestTick(t, r, vmRunner)

	type0 := getGlobalInt(t, vm, "g_type0") // buy position
	if type0 != 0 {                         // OP_BUY = 0
		t.Fatalf("OrderType[0] (buy position) = %d, want 0 (OP_BUY)", type0)
	}
	type1 := getGlobalInt(t, vm, "g_type1") // sell position
	if type1 != 1 {                         // OP_SELL = 1
		t.Fatalf("OrderType[1] (sell position) = %d, want 1 (OP_SELL)", type1)
	}
	type2 := getGlobalInt(t, vm, "g_type2") // buy_limit pending
	if type2 != 2 {                         // OP_BUYLIMIT = 2
		t.Fatalf("OrderType[2] (buy_limit pending) = %d, want 2 (OP_BUYLIMIT)", type2)
	}
	type3 := getGlobalInt(t, vm, "g_type3") // sell_stop pending
	if type3 != 5 {                         // OP_SELLSTOP = 5
		t.Fatalf("OrderType[3] (sell_stop pending) = %d, want 5 (OP_SELLSTOP)", type3)
	}
}

// TestLIVE_MQL_VM_OrderSymbol_EndToEnd verifies that OrderSymbol() returns
// the correct symbol for each order through the VM.
func TestLIVE_MQL_VM_OrderSymbol_EndToEnd(t *testing.T) {
	r, vmRunner := setupVMTestRunner(t)
	vm := runVMTestTick(t, r, vmRunner)

	sym0 := getGlobalString(t, vm, "g_symbol0")
	if sym0 != "EURUSD" {
		t.Fatalf("OrderSymbol[0] = %q, want %q", sym0, "EURUSD")
	}
	sym1 := getGlobalString(t, vm, "g_symbol1")
	if sym1 != "GBPUSD" {
		t.Fatalf("OrderSymbol[1] = %q, want %q", sym1, "GBPUSD")
	}
	sym2 := getGlobalString(t, vm, "g_symbol2")
	if sym2 != "EURUSD" {
		t.Fatalf("OrderSymbol[2] = %q, want %q", sym2, "EURUSD")
	}
}

// TestLIVE_MQL_VM_OrderLotsAndPrice_EndToEnd verifies that OrderLots()
// and OrderOpenPrice() return correct values through the VM for both
// market positions and pending orders.
func TestLIVE_MQL_VM_OrderLotsAndPrice_EndToEnd(t *testing.T) {
	r, vmRunner := setupVMTestRunner(t)
	vm := runVMTestTick(t, r, vmRunner)

	lots0 := getGlobalDecimal(t, vm, "g_lots0")
	if !lots0.Equal(decimal.NewFromFloat(0.1)) {
		t.Fatalf("OrderLots[0] = %s, want 0.1", lots0)
	}
	price0 := getGlobalDecimal(t, vm, "g_openPrice0")
	if !price0.Equal(decimal.NewFromFloat(1.1)) {
		t.Fatalf("OrderOpenPrice[0] = %s, want 1.1", price0)
	}

	// Pending order: lots and price
	lots2 := getGlobalDecimal(t, vm, "g_lots2")
	if !lots2.Equal(decimal.NewFromFloat(0.1)) {
		t.Fatalf("OrderLots[2] (pending) = %s, want 0.1", lots2)
	}
	price2 := getGlobalDecimal(t, vm, "g_openPrice2")
	if !price2.Equal(decimal.NewFromFloat(1.05)) {
		t.Fatalf("OrderOpenPrice[2] (pending) = %s, want 1.05", price2)
	}
}

// TestLIVE_MQL_VM_OrderSLTPComment_EndToEnd verifies that OrderStopLoss,
// OrderTakeProfit, and OrderComment return correct values through the VM.
func TestLIVE_MQL_VM_OrderSLTPComment_EndToEnd(t *testing.T) {
	r, vmRunner := setupVMTestRunner(t)
	vm := runVMTestTick(t, r, vmRunner)

	sl0 := getGlobalDecimal(t, vm, "g_sl0")
	if !sl0.Equal(decimal.NewFromFloat(1.0)) {
		t.Fatalf("OrderStopLoss[0] = %s, want 1.0", sl0)
	}
	tp0 := getGlobalDecimal(t, vm, "g_tp0")
	if !tp0.Equal(decimal.NewFromFloat(1.2)) {
		t.Fatalf("OrderTakeProfit[0] = %s, want 1.2", tp0)
	}
	cmt0 := getGlobalString(t, vm, "g_comment0")
	if cmt0 != "test-buy" {
		t.Fatalf("OrderComment[0] = %q, want %q", cmt0, "test-buy")
	}
}

// TestLIVE_MQL_VM_OrderSelect_ByTicket verifies that OrderSelect by ticket
// works correctly for both market positions and pending orders through the VM.
func TestLIVE_MQL_VM_OrderSelect_ByTicket(t *testing.T) {
	source := `
int g_found1001 = 0;
int g_found2001 = 0;
int g_magic1001 = -1;
int g_magic2001 = -1;
int g_type1001 = -1;
int g_type2001 = -1;

int OnInit() { return 0; }

void OnTick()
{
    if (OrderSelect(1001, SELECT_BY_TICKET, MODE_TRADES))
    {
        g_found1001 = 1;
        g_magic1001 = OrderMagicNumber();
        g_type1001 = OrderType();
    }
    if (OrderSelect(2001, SELECT_BY_TICKET, MODE_TRADES))
    {
        g_found2001 = 1;
        g_magic2001 = OrderMagicNumber();
        g_type2001 = OrderType();
    }
}
`
	vmRunner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}

	r := runner.New(runner.Config{})
	r.SetStrategy(vmRunner)

	now := time.Now()
	positions := []sdk.Position{
		{Ticket: 1001, Symbol: "EURUSD", Side: sdk.SideBuy, Magic: vmTestMagic,
			Volume: decimal.NewFromFloat(0.1), OpenPrice: decimal.NewFromFloat(1.1), OpenTime: now},
	}
	pendingOrders := []sdk.PendingOrder{
		{Ticket: 2001, Symbol: "EURUSD", Type: sdk.OrderLimit, Side: sdk.SideBuy, Magic: vmTestMagic,
			Volume: decimal.NewFromFloat(0.1), Price: decimal.NewFromFloat(1.05), OpenTime: now},
	}
	r.UpdateLiveState("10000", "10500", "500", "9500", positions, pendingOrders)

	ctx := context.Background()
	_, err = r.OnTick(ctx, decimal.NewFromFloat(1.1), decimal.NewFromFloat(1.1001))
	if err != nil {
		t.Fatalf("OnTick failed: %v", err)
	}

	// Position ticket 1001 should be found
	if getGlobalInt(t, vmRunner, "g_found1001") != 1 {
		t.Fatal("OrderSelect(1001, SELECT_BY_TICKET) failed for market position")
	}
	if getGlobalInt(t, vmRunner, "g_magic1001") != vmTestMagic {
		t.Fatalf("OrderMagicNumber for ticket 1001 = %d, want %d", getGlobalInt(t, vmRunner, "g_magic1001"), vmTestMagic)
	}
	if getGlobalInt(t, vmRunner, "g_type1001") != 0 { // OP_BUY
		t.Fatalf("OrderType for ticket 1001 = %d, want 0 (OP_BUY)", getGlobalInt(t, vmRunner, "g_type1001"))
	}

	// Pending order ticket 2001 should be found
	if getGlobalInt(t, vmRunner, "g_found2001") != 1 {
		t.Fatal("OrderSelect(2001, SELECT_BY_TICKET) failed for pending order")
	}
	if getGlobalInt(t, vmRunner, "g_magic2001") != vmTestMagic {
		t.Fatalf("OrderMagicNumber for ticket 2001 = %d, want %d", getGlobalInt(t, vmRunner, "g_magic2001"), vmTestMagic)
	}
	if getGlobalInt(t, vmRunner, "g_type2001") != 2 { // OP_BUYLIMIT
		t.Fatalf("OrderType for ticket 2001 = %d, want 2 (OP_BUYLIMIT)", getGlobalInt(t, vmRunner, "g_type2001"))
	}
}

// TestLIVE_MQL_VM_EmptyState_OrdersTotalZero verifies that OrdersTotal
// returns 0 when the live harness has no positions and no pending orders.
func TestLIVE_MQL_VM_EmptyState_OrdersTotalZero(t *testing.T) {
	source := `
int g_total = -1;
int OnInit() { return 0; }
void OnTick() { g_total = OrdersTotal(); }
`
	vmRunner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}

	r := runner.New(runner.Config{})
	r.SetStrategy(vmRunner)
	r.UpdateLiveState("10000", "10500", "500", "9500", nil, nil)

	ctx := context.Background()
	_, err = r.OnTick(ctx, decimal.NewFromFloat(1.1), decimal.NewFromFloat(1.1001))
	if err != nil {
		t.Fatalf("OnTick failed: %v", err)
	}

	total := getGlobalInt(t, vmRunner, "g_total")
	if total != 0 {
		t.Fatalf("VM OrdersTotal with empty state = %d, want 0", total)
	}
}

// TestLIVE_MQL_VM_OnlyPending_OrdersTotal verifies that OrdersTotal
// returns 2 when the live harness has only pending orders (no positions).
func TestLIVE_MQL_VM_OnlyPending_OrdersTotal(t *testing.T) {
	source := `
int g_total = -1;
int OnInit() { return 0; }
void OnTick() { g_total = OrdersTotal(); }
`
	vmRunner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}

	r := runner.New(runner.Config{})
	r.SetStrategy(vmRunner)

	now := time.Now()
	pendingOrders := []sdk.PendingOrder{
		{Ticket: 2001, Symbol: "EURUSD", Type: sdk.OrderLimit, Side: sdk.SideBuy, Magic: vmTestMagic,
			Volume: decimal.NewFromFloat(0.1), Price: decimal.NewFromFloat(1.05), OpenTime: now},
		{Ticket: 2002, Symbol: "GBPUSD", Type: sdk.OrderStop, Side: sdk.SideSell, Magic: vmTestMagic,
			Volume: decimal.NewFromFloat(0.15), Price: decimal.NewFromFloat(1.25), OpenTime: now},
	}
	r.UpdateLiveState("10000", "10500", "500", "9500", nil, pendingOrders)

	ctx := context.Background()
	_, err = r.OnTick(ctx, decimal.NewFromFloat(1.1), decimal.NewFromFloat(1.1001))
	if err != nil {
		t.Fatalf("OnTick failed: %v", err)
	}

	total := getGlobalInt(t, vmRunner, "g_total")
	if total != 2 {
		t.Fatalf("VM OrdersTotal with only pending = %d, want 2", total)
	}
}
