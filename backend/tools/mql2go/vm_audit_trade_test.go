package mql2go

import (
	"context"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/backtest"
	"alphaforge/strategy/sdk"
)

func TestVM_Audit_OrderCacheInvalidatedAfterMutation(t *testing.T) {
	const source = `
int g_after = -1;
int OnInit() { return 0; }
void OnTick() {
    if (OrdersTotal() == 0)
        OrderSend(Symbol(), OP_BUY, 0.1, Ask, 0, 0, 0, "cache", 1, 0, clrGreen);
    g_after = OrdersTotal();
}
`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	engine := backtest.New(backtest.Config{
		Symbol: "EURUSD", Timeframe: "M1", InitialCapital: decimal.NewFromInt(100000), Leverage: 100,
	}, runner, auditBars(3))
	if _, err := engine.Run(context.Background()); err != nil {
		t.Fatalf("Engine.Run: %v", err)
	}
	value, ok := runner.GetGlobal("g_after")
	if !ok || value.ToInt() != 1 {
		t.Fatalf("OrdersTotal after OrderSend = %v, want 1", value)
	}
}

func TestVM_Audit_CTradeMagicReachesBroker(t *testing.T) {
	const source = `
#include <Trade/Trade.mqh>
CTrade trade;
int OnInit() {
    trade.SetExpertMagicNumber(12345);
    return 0;
}
void OnTick() { trade.Buy(0.1, Symbol()); }
`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	engine := backtest.New(backtest.Config{
		Symbol: "EURUSD", Timeframe: "M1", InitialCapital: decimal.NewFromInt(100000), Leverage: 100,
	}, runner, auditBars(3))
	if _, err := engine.Run(context.Background()); err != nil {
		t.Fatalf("Engine.Run: %v", err)
	}
	positions := engine.Broker().Positions(0)
	if len(positions) == 0 {
		t.Fatal("CTrade.Buy did not create a position")
	}
	if positions[0].Magic != 12345 {
		t.Fatalf("CTrade position magic = %d, want 12345", positions[0].Magic)
	}
}

func TestVM_Audit_CTradeDeviationReachesVM(t *testing.T) {
	// Verify SetDeviationInPoints is wired through the CTrade namespace
	// (CTrade.SetDeviationInPoints, not bare SetDeviationInPoints) and
	// stores the value in VM state for subsequent Buy/Sell calls.
	const source = `
#include <Trade/Trade.mqh>
CTrade trade;
int OnInit() {
    trade.SetDeviationInPoints(50);
    return 0;
}
void OnTick() { trade.Buy(0.1, Symbol()); }
`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	if err := runner.vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit: %v", err)
	}
	if runner.vm.tradeDeviation != 50 {
		t.Fatalf("CTrade deviation = %d, want 50 (SetDeviationInPoints not wired through CTrade namespace)", runner.vm.tradeDeviation)
	}
}

func TestVM_Audit_ModifySignalIsDispatched(t *testing.T) {
	cfg := backtest.Config{
		Symbol:         "EURUSD",
		Timeframe:      "M1",
		InitialCapital: decimal.NewFromInt(10000),
		Leverage:       100,
		SignalTiming:   "same_bar_close",
	}
	strategy := &auditModifySignalStrategy{}
	engine := backtest.New(cfg, strategy, auditBars(3))
	if _, err := engine.Run(context.Background()); err != nil {
		t.Fatalf("backtest: %v", err)
	}
	pos := engine.Broker().Positions(0)
	if len(pos) != 1 || !pos[0].StopLoss.Equal(decimal.NewFromInt(99)) || !pos[0].TakeProfit.Equal(decimal.NewFromInt(101)) {
		t.Fatalf("modified position = %+v", pos)
	}
}

func TestVM_Audit_VMOrderSendErrorsPropagate(t *testing.T) {
	runner, err := CompileMQL(`int OnInit() { return 0; } void OnTick() { OrderSend(Symbol(), OP_BUY, 1, Ask, 0, 0, 0); }`)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	_, err = backtest.New(backtest.Config{
		Symbol:         "EURUSD",
		Timeframe:      "M1",
		InitialCapital: decimal.NewFromInt(10000),
		Leverage:       1,
		ContractSize:   decimal.NewFromInt(100000),
	}, runner, auditBars(2)).Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "OrderSend") {
		t.Fatalf("VM OrderSend error was swallowed: %v", err)
	}
}

func TestVM_Audit_RejectedOrderDoesNotChargeCommission(t *testing.T) {
	broker := backtest.NewSimBroker(backtest.Config{
		InitialCapital: decimal.NewFromInt(10000),
		Leverage:       1,
		Commission:     decimal.NewFromFloat(0.01),
		ContractSize:   decimal.NewFromInt(100000),
	})
	broker.SetBarPrice(decimal.NewFromInt(1))
	if _, err := broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideBuy, Type: sdk.OrderMarket,
		Volume: decimal.NewFromInt(1), Price: decimal.NewFromInt(1),
	}); err == nil {
		t.Fatal("insufficient-margin order was accepted")
	}
	account := broker.Account()
	if !account.Balance.Equal(decimal.NewFromInt(10000)) || len(broker.Positions(0)) != 0 {
		t.Fatalf("rejected order mutated account: balance=%s positions=%d", account.Balance, len(broker.Positions(0)))
	}
}

func TestVM_Audit_BrokerSignalErrorsPropagate(t *testing.T) {
	_, err := backtest.New(backtest.Config{
		Symbol:         "EURUSD",
		Timeframe:      "M1",
		InitialCapital: decimal.NewFromInt(10000),
		Leverage:       100,
		SignalTiming:   "same_bar_close",
	}, auditInvalidSignalStrategy{}, auditBars(2)).Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "PositionClose") {
		t.Fatalf("broker signal error was swallowed: %v", err)
	}
}

func TestVM_Audit_CanceledBacktestDoesNotReturnSuccess(t *testing.T) {
	runner, err := CompileMQL(`int OnInit() { return 0; } void OnTick() {}`)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = backtest.New(backtest.Config{
		Symbol:         "EURUSD",
		Timeframe:      "M1",
		InitialCapital: decimal.NewFromInt(10000),
		Leverage:       100,
	}, runner, auditBars(3)).Run(ctx)
	if err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("canceled backtest returned err=%v", err)
	}
}

func TestVM_Audit_RuntimeFatalModeStopsExecution(t *testing.T) {
	const source = `
int g_after = 0;
int OnInit() { return 0; }
void OnTick() {
    g_after = 1;
    double plus = iADX(Symbol(), 0, 14, PRICE_CLOSE, MODE_PLUSDI, 0);
    g_after = 2;
}
`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	engine := backtest.New(backtest.Config{
		Symbol: "EURUSD", Timeframe: "M1", InitialCapital: decimal.NewFromInt(100000), Leverage: 100,
	}, runner, auditBars(3))
	if _, err := engine.Run(context.Background()); err == nil {
		t.Fatal("unsupported fatal indicator mode completed without an execution error")
	}
	value, ok := runner.GetGlobal("g_after")
	if !ok || value.ToInt() != 1 {
		t.Fatalf("execution continued after fatal indicator mode: g_after=%v, want 1", value)
	}
}

func TestVM_Audit_BuiltinErrorStopsExecution(t *testing.T) {
	// When a builtin returns a Go error (e.g. OrderSend with invalid volume),
	// the VM must stop immediately — post-error instructions must not execute.
	const source = `
int g_after = 0;
int OnInit() { return 0; }
void OnTick() {
    // OrderSend with volume=0 triggers a builtin Go error
    // (SimBroker rejects zero volume).
    int ticket = OrderSend(Symbol(), OP_BUY, 0, 0, 5, 0, 0, "", 0);
    g_after = 42;
}`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	engine := backtest.New(backtest.Config{
		Symbol: "EURUSD", Timeframe: "M1", InitialCapital: decimal.NewFromInt(100000), Leverage: 100,
	}, runner, auditBars(3))
	_, err = engine.Run(context.Background())
	if err == nil {
		t.Fatal("builtin error (OrderSend volume=0) completed without an execution error")
	}
	value, ok := runner.GetGlobal("g_after")
	if !ok || value.ToInt() != 0 {
		t.Fatalf("execution continued after builtin error: g_after=%v, want 0", value)
	}
}

func TestVM_Audit_InvalidMutationDoesNotChangeCapital(t *testing.T) {
	// When OrderSend fails (invalid volume), the broker's balance and
	// positions must remain unchanged — no partial mutation.
	const source = `
int OnInit() { return 0; }
void OnTick() {
    OrderSend(Symbol(), OP_BUY, 0, 0, 5, 0, 0, "", 0);
}`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	cfg := backtest.Config{
		Symbol: "EURUSD", Timeframe: "M1", InitialCapital: decimal.NewFromInt(100000), Leverage: 100,
	}
	engine := backtest.New(cfg, runner, auditBars(3))
	_, err = engine.Run(context.Background())
	if err == nil {
		t.Fatal("invalid OrderSend should have caused an error")
	}
	// Balance must be unchanged — no partial mutation from the failed order.
	broker := engine.Broker()
	if !broker.Account().Balance.Equal(cfg.InitialCapital) {
		t.Fatalf("balance changed after invalid OrderSend: %s, want %s", broker.Account().Balance, cfg.InitialCapital)
	}
	if len(broker.Positions(0)) != 0 {
		t.Fatalf("positions created after invalid OrderSend: %d", len(broker.Positions(0)))
	}
}

func TestVM_Audit_FatalBlindSpotFromHandlerNotPushedToStack(t *testing.T) {
	// When a builtin handler sets fatalError via recordBlindSpot (e.g. iADX
	// MODE_PLUSDI), callBuiltin must return an error immediately — the zero
	// result must NOT be pushed to the stack. This verifies the defense-in-depth
	// check in callBuiltin (not just the OP_CALL_BUILTIN opcode handler check).
	const source = `
double g_result = 99.0;
int g_after = 0;
int OnInit() { return 0; }
void OnTick() {
    g_result = iADX(Symbol(), 0, 14, PRICE_CLOSE, MODE_PLUSDI, 0);
    g_after = 1;
}`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	engine := backtest.New(backtest.Config{
		Symbol: "EURUSD", Timeframe: "M1", InitialCapital: decimal.NewFromInt(100000), Leverage: 100,
	}, runner, auditBars(3))
	_, err = engine.Run(context.Background())
	if err == nil {
		t.Fatal("fatal blind spot (iADX MODE_PLUSDI) completed without an error")
	}
	// g_result should still be 99.0 (the assignment never completed because
	// callBuiltin returned an error before the result was pushed).
	result, ok := runner.GetGlobal("g_result")
	if !ok || !result.ToDecimal().Equal(decimal.NewFromFloat(99.0)) {
		t.Fatalf("g_result = %v, want 99.0 (fatal blind spot result should not be pushed)", result)
	}
	after, ok := runner.GetGlobal("g_after")
	if !ok || after.ToInt() != 0 {
		t.Fatalf("g_after = %v, want 0 (post-error instruction should not execute)", after)
	}
}

func TestVM_Audit_BuiltinErrorPropagatesToEngine(t *testing.T) {
	// Verify that a builtin Go error propagates through the full pipeline:
	// VM → VMRunner.OnBar → backtest.Engine.Run → error result.
	const source = `
int OnInit() { return 0; }
void OnTick() {
    // Unsupported order command (cmd=99) triggers a builtin Go error.
    OrderSend(Symbol(), 99, 0.1, 0, 5, 0, 0, "", 0);
}`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	engine := backtest.New(backtest.Config{
		Symbol: "EURUSD", Timeframe: "M1", InitialCapital: decimal.NewFromInt(100000), Leverage: 100,
	}, runner, auditBars(3))
	result, err := engine.Run(context.Background())
	if err == nil {
		t.Fatal("builtin error should propagate to Engine.Run error")
	}
	if result != nil {
		t.Fatalf("Engine.Run returned non-nil result with error: %v", result)
	}
	if !strings.Contains(err.Error(), "backtest: strategy event failed") {
		t.Fatalf("error should be wrapped as strategy event failure, got: %v", err)
	}
}
