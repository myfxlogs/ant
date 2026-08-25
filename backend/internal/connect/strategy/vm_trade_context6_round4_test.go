package strategy

import (
	"context"
	"strings"
	"testing"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/tools/mql2go"

	"google.golang.org/protobuf/proto"
)

// ── VM-TRADE-CONTEXT-6 round 4 behavior tests ────────────────────────
// Tests for: dispatchVMLive validates before Init, extra symbol OHLCV
// validation, financial field strict parsing, unknown enum fail-closed,
// negative volume rejection.

// validBarContext builds a minimal valid LiveStrategyContext for testing.
func validBarContext() *antv1.LiveStrategyContext {
	return &antv1.LiveStrategyContext{
		Open:       []string{"1.0"},
		High:       []string{"2.0"},
		Low:        []string{"0.5"},
		Close:      []string{"1.5"},
		Volume:     []string{"100"},
		BarTimesMs: []int64{100},
		Symbol:     "EURUSD",
		Timeframe:  "M5",
	}
}

// TestDispatchVMLive_RejectsInvalidBeforeInit verifies that dispatchVMLive
// validates the bar context BEFORE calling r.Init, so that an invalid first
// request does not execute OnInit (g_init must NOT be set).
// VM-TRADE-CONTEXT-6 round 4: the key blocker — dispatchVMLive was calling
// r.Init before vmHandleBar validation.
func TestDispatchVMLive_RejectsInvalidBeforeInit(t *testing.T) {
	srv := NewStrategyExecutionServer(nil, nil)
	source := `
int g_init = 0;
int OnInit() { g_init = 1; return 0; }
int OnBar() { return 0; }
`
	strategy, err := mql2go.CompileMQL(source)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	strategy.SetSignalMode(true)

	// Construct an invalid bar context (bad decimal in Close).
	bctx := validBarContext()
	bctx.Close = []string{"bad_decimal"}
	req := &antv1.ExecuteLiveRequest{
		RequestType:  antv1.RequestType_REQUEST_TYPE_BAR,
		BarContext:   bctx,
		StrategyCode: source,
	}

	resp, err := srv.dispatchVMLive(context.Background(), req, strategy)
	if err != nil {
		t.Fatalf("dispatchVMLive error: %v", err)
	}
	if resp == nil || resp.Success {
		t.Fatal("dispatchVMLive should reject invalid bar context")
	}
	if !strings.Contains(resp.Error, "invalid") && !strings.Contains(resp.Error, "bad_decimal") {
		t.Fatalf("error should mention invalid decimal, got: %s", resp.Error)
	}
	// Critical: g_init must NOT be set — OnInit must not have executed.
	// After Init, GetGlobal returns ok=true with g_init=1. If validation
	// rejected before Init, ok=false (globals not populated).
	v, ok := strategy.GetGlobal("g_init")
	if ok && v.ToInt() == 1 {
		t.Fatalf("g_init = %v (ok=%v), want uninitialized — OnInit must not execute with invalid bar context", v, ok)
	}
}

// TestValidateBarContext_ExtraSymbolBadDecimal verifies that validateBarContext
// rejects invalid decimals in extra symbol OHLCV data.
// VM-TRADE-CONTEXT-6 round 4: validateFirstBarContext was not checking
// extra symbol OHLCV — OnInit could run with bad extra symbol data.
func TestValidateBarContext_ExtraSymbolBadDecimal(t *testing.T) {
	bctx := validBarContext()
	bctx.Symbols = []*antv1.LiveSymbolSeries{
		{
			Symbol: "GBPUSD",
			Open:   []string{"1.2"},
			High:   []string{"1.3"},
			Low:    []string{"1.1"},
			Close:  []string{"not_a_number"}, // invalid!
			Volume: []string{"50"},
		},
	}
	err := validateBarContext(bctx)
	if err == nil {
		t.Fatal("validateBarContext should reject invalid extra symbol decimal")
	}
	if !strings.Contains(err.Error(), "GBPUSD") {
		t.Fatalf("error should mention symbol name, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "close") {
		t.Fatalf("error should mention close field, got: %s", err.Error())
	}
}

// TestValidateBarContext_ExtraSymbolLengthMismatch verifies that
// validateBarContext rejects length mismatch in extra symbol OHLCV.
func TestValidateBarContext_ExtraSymbolLengthMismatch(t *testing.T) {
	bctx := validBarContext()
	bctx.Symbols = []*antv1.LiveSymbolSeries{
		{
			Symbol: "GBPUSD",
			Open:   []string{"1.2", "1.3"}, // 2 bars
			High:   []string{"1.3"},
			Low:    []string{"1.1"},
			Close:  []string{"1.25"},
			Volume: []string{"50"},
		},
	}
	err := validateBarContext(bctx)
	if err == nil {
		t.Fatal("validateBarContext should reject extra symbol length mismatch")
	}
	if !strings.Contains(err.Error(), "GBPUSD") {
		t.Fatalf("error should mention symbol name, got: %s", err.Error())
	}
}

// TestValidateBarContext_InvalidFinancialField verifies that
// validateBarContext rejects invalid financial field strings.
// VM-TRADE-CONTEXT-6 round 4: Balance/Equity/Margin/FreeMargin were passed
// directly to UpdateLiveState → mustDecimal converts invalid to -1 (fail-open).
func TestValidateBarContext_InvalidFinancialField(t *testing.T) {
	bctx := validBarContext()
	bctx.Balance = "not_a_number"
	err := validateBarContext(bctx)
	if err == nil {
		t.Fatal("validateBarContext should reject invalid balance")
	}
	if !strings.Contains(err.Error(), "balance") {
		t.Fatalf("error should mention balance, got: %s", err.Error())
	}
}

// TestValidateBarContext_EmptyFinancialFieldsOk verifies that empty
// financial fields are allowed (field not provided by broker).
func TestValidateBarContext_EmptyFinancialFieldsOk(t *testing.T) {
	bctx := validBarContext()
	bctx.Balance = ""
	bctx.Equity = ""
	bctx.Margin = ""
	bctx.FreeMargin = ""
	err := validateBarContext(bctx)
	if err != nil {
		t.Fatalf("empty financial fields should be allowed, got: %v", err)
	}
}

// TestValidateBarContext_NegativeVolumeRejected verifies that negative
// volume is rejected by validateBarContext.
// VM-TRADE-CONTEXT-6 round 4: domain validation — negative volume is invalid.
func TestValidateBarContext_NegativeVolumeRejected(t *testing.T) {
	bctx := validBarContext()
	bctx.Volume = []string{"-100"}
	err := validateBarContext(bctx)
	if err == nil {
		t.Fatal("validateBarContext should reject negative volume")
	}
	if !strings.Contains(err.Error(), "negative") {
		t.Fatalf("error should mention negative, got: %s", err.Error())
	}
}

// TestValidateBarContext_ExtraSymbolNegativeVolumeRejected verifies that
// negative volume in extra symbol data is rejected.
func TestValidateBarContext_ExtraSymbolNegativeVolumeRejected(t *testing.T) {
	bctx := validBarContext()
	bctx.Symbols = []*antv1.LiveSymbolSeries{
		{
			Symbol: "GBPUSD",
			Open:   []string{"1.2"},
			High:   []string{"1.3"},
			Low:    []string{"1.1"},
			Close:  []string{"1.25"},
			Volume: []string{"-50"}, // negative!
		},
	}
	err := validateBarContext(bctx)
	if err == nil {
		t.Fatal("validateBarContext should reject negative extra symbol volume")
	}
	if !strings.Contains(err.Error(), "negative") {
		t.Fatalf("error should mention negative, got: %s", err.Error())
	}
}

// TestVMHandleTrade_UnknownSideRejected verifies that an unknown side
// string is rejected, not silently mapped to buy.
// VM-TRADE-CONTEXT-6 round 4: unknown enum must fail-closed.
func TestVMHandleTrade_UnknownSideRejected(t *testing.T) {
	evctx := &antv1.TradeContext{
		Side:      "unknown_side",
		EventType: "fill",
		Volume:    "0.1",
		Price:     "1.1",
		Symbol:    "EURUSD",
	}
	resp := vmHandleTrade(context.Background(), nil, evctx)
	if resp == nil || resp.Success {
		t.Fatal("vmHandleTrade should reject unknown side")
	}
	if !strings.Contains(resp.Error, "side") {
		t.Fatalf("error should mention side, got: %s", resp.Error)
	}
}

// TestVMHandleTrade_UnknownEventTypeRejected verifies that an unknown
// trade event type is rejected, not silently mapped to filled.
// VM-TRADE-CONTEXT-6 round 4: unknown enum must fail-closed.
func TestVMHandleTrade_UnknownEventTypeRejected(t *testing.T) {
	evctx := &antv1.TradeContext{
		Side:      "buy",
		EventType: "unknown_event",
		Volume:    "0.1",
		Price:     "1.1",
		Symbol:    "EURUSD",
	}
	resp := vmHandleTrade(context.Background(), nil, evctx)
	if resp == nil || resp.Success {
		t.Fatal("vmHandleTrade should reject unknown event type")
	}
	if !strings.Contains(resp.Error, "event_type") {
		t.Fatalf("error should mention event_type, got: %s", resp.Error)
	}
}

// TestVMHandleTrade_NegativeVolumeRejected verifies that negative volume
// in trade context is rejected.
// VM-TRADE-CONTEXT-6 round 4: domain validation.
func TestVMHandleTrade_NegativeVolumeRejected(t *testing.T) {
	evctx := &antv1.TradeContext{
		Side:      "buy",
		EventType: "fill",
		Volume:    "-0.1",
		Price:     "1.1",
		Symbol:    "EURUSD",
	}
	resp := vmHandleTrade(context.Background(), nil, evctx)
	if resp == nil || resp.Success {
		t.Fatal("vmHandleTrade should reject negative volume")
	}
	if !strings.Contains(resp.Error, "negative") {
		t.Fatalf("error should mention negative, got: %s", resp.Error)
	}
}

// TestVMHandleTrade_InvalidFinancialField verifies that invalid financial
// fields in trade context are rejected before UpdateLiveState.
func TestVMHandleTrade_InvalidFinancialField(t *testing.T) {
	evctx := &antv1.TradeContext{
		Side:       "buy",
		EventType:  "fill",
		Volume:     "0.1",
		Price:      "1.1",
		StopLoss:   "0",
		TakeProfit: "0",
		Profit:     "0",
		Commission: "0",
		Swap:       "0",
		Symbol:     "EURUSD",
		Balance:    "not_a_number",
	}
	resp := vmHandleTrade(context.Background(), nil, evctx)
	if resp == nil || resp.Success {
		t.Fatal("vmHandleTrade should reject invalid balance")
	}
	if !strings.Contains(resp.Error, "balance") {
		t.Fatalf("error should mention balance, got: %s", resp.Error)
	}
}

// TestVmPositionsToSdk_UnknownSideRejected verifies that an unknown side
// in a position is rejected, not silently mapped to buy.
func TestVmPositionsToSdk_UnknownSideRejected(t *testing.T) {
	lps := []*antv1.LivePosition{
		{
			Side:       "unknown",
			Volume:     "0.1",
			OpenPrice:  "1.1",
			Sl:         "0",
			Tp:         "0",
			Profit:     "0",
			Swap:       "0",
			Commission: "0",
		},
	}
	_, err := vmPositionsToSdk(lps)
	if err == nil {
		t.Fatal("vmPositionsToSdk should reject unknown side")
	}
	if !strings.Contains(err.Error(), "side") {
		t.Fatalf("error should mention side, got: %s", err.Error())
	}
}

// TestVmPendingOrdersToSdk_UnknownOrderTypeRejected verifies that an
// unknown order type is rejected, not silently mapped to market.
func TestVmPendingOrdersToSdk_UnknownOrderTypeRejected(t *testing.T) {
	lpos := []*antv1.LivePendingOrder{
		{
			Side:      "buy",
			OrderType: "unknown_type",
			Volume:    "0.1",
			Price:     "1.1",
			Sl:        "0",
			Tp:        "0",
		},
	}
	_, err := vmPendingOrdersToSdk(lpos)
	if err == nil {
		t.Fatal("vmPendingOrdersToSdk should reject unknown order type")
	}
	if !strings.Contains(err.Error(), "order_type") {
		t.Fatalf("error should mention order_type, got: %s", err.Error())
	}
}

// TestVmPositionsToSdk_NegativeVolumeRejected verifies that negative
// volume in a position is rejected.
func TestVmPositionsToSdk_NegativeVolumeRejected(t *testing.T) {
	lps := []*antv1.LivePosition{
		{
			Side:       "buy",
			Volume:     "-0.1",
			OpenPrice:  "1.1",
			Sl:         "0",
			Tp:         "0",
			Profit:     "0",
			Swap:       "0",
			Commission: "0",
		},
	}
	_, err := vmPositionsToSdk(lps)
	if err == nil {
		t.Fatal("vmPositionsToSdk should reject negative volume")
	}
	if !strings.Contains(err.Error(), "negative") {
		t.Fatalf("error should mention negative, got: %s", err.Error())
	}
}

// TestVMLiveSession_StartRejectsInvalidExtraSymbol verifies that
// VMLiveSession.Start rejects invalid extra symbol data before OnInit.
func TestVMLiveSession_StartRejectsInvalidExtraSymbol(t *testing.T) {
	source := `
int g_init = 0;
int OnInit() { g_init = 1; return 0; }
int OnBar() { return 0; }
`
	sess, err := NewVMLiveSession(source)
	if err != nil {
		t.Fatalf("NewVMLiveSession: %v", err)
	}
	bctx := validBarContext()
	bctx.Symbols = []*antv1.LiveSymbolSeries{
		{
			Symbol: "GBPUSD",
			Open:   []string{"1.2"},
			High:   []string{"1.3"},
			Low:    []string{"1.1"},
			Close:  []string{"bad"}, // invalid!
			Volume: []string{"50"},
		},
	}
	req := &antv1.ExecuteLiveRequest{
		RequestType: antv1.RequestType_REQUEST_TYPE_BAR,
		BarContext:  bctx,
	}
	reqBytes, _ := proto.Marshal(req)
	_, err = sess.Start(context.Background(), reqBytes)
	if err == nil {
		t.Fatal("Start should reject invalid extra symbol data")
	}
	// g_init must NOT be set — OnInit must not have executed.
	v, ok := sess.strategy.GetGlobal("g_init")
	if ok && v.ToInt() == 1 {
		t.Fatalf("g_init = %v (ok=%v), want uninitialized — OnInit must not execute with invalid extra symbol", v, ok)
	}
}

// TestValidateBarContext_ValidPasses verifies that a fully valid context
// with all fields populated passes validation.
func TestValidateBarContext_ValidPasses(t *testing.T) {
	bctx := validBarContext()
	bctx.Balance = "10000"
	bctx.Equity = "10000"
	bctx.Margin = "0"
	bctx.FreeMargin = "10000"
	bctx.Symbols = []*antv1.LiveSymbolSeries{
		{
			Symbol: "GBPUSD",
			Open:   []string{"1.2"},
			High:   []string{"1.3"},
			Low:    []string{"1.1"},
			Close:  []string{"1.25"},
			Volume: []string{"50"},
		},
	}
	bctx.Positions = []*antv1.LivePosition{
		{Side: "buy", Volume: "0.1", OpenPrice: "1.1", Sl: "0", Tp: "0", Profit: "0", Swap: "0", Commission: "0"},
	}
	bctx.PendingOrders = []*antv1.LivePendingOrder{
		{Side: "buy", OrderType: "buy_limit", Volume: "0.1", Price: "1.0", Sl: "0", Tp: "0"},
	}
	err := validateBarContext(bctx)
	if err != nil {
		t.Fatalf("valid context should pass, got: %v", err)
	}
}
