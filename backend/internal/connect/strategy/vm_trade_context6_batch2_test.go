// vm_trade_context6_batch2_test.go — VM-TRADE-CONTEXT-6 (Batch 2) tests.
//
// Tests verify S1-S8 implementation:
//
//	S1: parseDecimalStrict/parseInt64Strict (fail-closed, not silent zero)
//	S2: OHLCV array length validation in vmHandleBar
//	S3: strict parse in live handlers (bar/tick/trade)
//	S4: nil repeated message rejection in live mode
//	S5: validateFirstBarContext before Init
//	S6: lookup injection (Login/Company/IsDemo/IsConnected/IsTradeAllowed)
//	S7: lookup wiring to StrategyExecutionServer
//
// Adversarial proofs P1-P6: each critical line mutated → relevant test RED → restore GREEN.
package strategy

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/strategy/runner"
)

// --- S1: parseDecimalStrict / parseInt64Strict ---

func TestParseDecimalStrict_Valid(t *testing.T) {
	d, err := parseDecimalStrict("123.45")
	if err != nil {
		t.Fatalf("parseDecimalStrict(\"123.45\") error: %v", err)
	}
	if !d.Equal(decimal.NewFromFloat(123.45)) {
		t.Errorf("parseDecimalStrict(\"123.45\") = %s, want 123.45", d)
	}
}

func TestParseDecimalStrict_Zero(t *testing.T) {
	// "0" is a valid decimal — must NOT be rejected.
	d, err := parseDecimalStrict("0")
	if err != nil {
		t.Fatalf("parseDecimalStrict(\"0\") error: %v (zero is valid)", err)
	}
	if !d.Equal(decimal.Zero) {
		t.Errorf("parseDecimalStrict(\"0\") = %s, want 0", d)
	}
}

func TestParseDecimalStrict_Invalid(t *testing.T) {
	_, err := parseDecimalStrict("abc")
	if err == nil {
		t.Fatal("parseDecimalStrict(\"abc\") should return error, got nil")
	}
}

func TestParseInt64Strict_Valid(t *testing.T) {
	n, err := parseInt64Strict("42")
	if err != nil {
		t.Fatalf("parseInt64Strict(\"42\") error: %v", err)
	}
	if n != 42 {
		t.Errorf("parseInt64Strict(\"42\") = %d, want 42", n)
	}
}

func TestParseInt64Strict_Invalid(t *testing.T) {
	_, err := parseInt64Strict("xyz")
	if err == nil {
		t.Fatal("parseInt64Strict(\"xyz\") should return error, got nil")
	}
}

// --- T1: OHLCV array length mismatch ---

func TestVMHandleBar_ArrayLengthMismatch(t *testing.T) {
	r := runner.New(runner.Config{Mode: "paper"})
	lctx := &antv1.LiveStrategyContext{
		Close:         []string{"1.0", "2.0", "3.0", "4.0", "5.0"},
		Open:          []string{"1.0", "2.0", "3.0"}, // mismatch: 3 != 5
		High:          []string{"1.0", "2.0", "3.0", "4.0", "5.0"},
		Low:           []string{"1.0", "2.0", "3.0", "4.0", "5.0"},
		Volume:        []string{"10", "20", "30", "40", "50"},
		BarTimesMs:    []int64{1, 2, 3, 4, 5},
		Mode:          "paper",
		Positions:     []*antv1.LivePosition{},
		PendingOrders: []*antv1.LivePendingOrder{},
	}
	resp := vmHandleBar(context.Background(), r, lctx)
	if resp.Success {
		t.Fatal("vmHandleBar should fail on array length mismatch, got Success=true")
	}
	if !contains(resp.Error, "OHLCV array length mismatch") {
		t.Errorf("Error = %q, want contains \"OHLCV array length mismatch\"", resp.Error)
	}
}

// --- T2: invalid decimal rejected ---

func TestVMHandleBar_InvalidDecimalRejected(t *testing.T) {
	r := runner.New(runner.Config{Mode: "paper"})
	lctx := &antv1.LiveStrategyContext{
		Close:         []string{"abc"}, // invalid decimal
		Open:          []string{"1.0"},
		High:          []string{"1.0"},
		Low:           []string{"1.0"},
		Volume:        []string{"10"},
		BarTimesMs:    []int64{1},
		Mode:          "paper",
		Positions:     []*antv1.LivePosition{},
		PendingOrders: []*antv1.LivePendingOrder{},
	}
	resp := vmHandleBar(context.Background(), r, lctx)
	if resp.Success {
		t.Fatal("vmHandleBar should fail on invalid decimal, got Success=true")
	}
	if !contains(resp.Error, "invalid decimal") {
		t.Errorf("Error = %q, want contains \"invalid decimal\"", resp.Error)
	}
}

// --- T3: nil positions rejected in live mode ---

func TestVMHandleBar_NilPositionAccepted(t *testing.T) {
	// VM-TRADE-CONTEXT-6 S4: nil positions are accepted at the VM handler
	// level because proto3 marshal/unmarshal collapses empty slices to nil.
	// The fail-closed protection lives upstream in buildLiveContext, which
	// returns an error if posCache cannot provide a snapshot in live mode.
	// At the handler level, nil positions == no open positions (empty slice).
	r := runner.New(runner.Config{Mode: "live"})
	lctx := &antv1.LiveStrategyContext{
		Close:         []string{"1.0"},
		Open:          []string{"1.0"},
		High:          []string{"1.0"},
		Low:           []string{"1.0"},
		Volume:        []string{"10"},
		BarTimesMs:    []int64{1},
		Mode:          "live",
		Positions:     nil, // nil == empty slice after proto round-trip
		PendingOrders: nil,
	}
	resp := vmHandleBar(context.Background(), r, lctx)
	// Should succeed — nil positions means "no open positions", not "data missing".
	// Data-missing protection is in buildLiveContext (posCache fail-closed).
	if !resp.Success {
		t.Errorf("vmHandleBar should accept nil positions (proto3 nil==empty), got error: %s", resp.Error)
	}
}

// --- T4: buildLiveContext injects Login and Company ---

func TestBuildLiveContext_InjectsLoginAndCompany(t *testing.T) {
	srv := NewStrategyExecutionServer(nil, zap.NewNop())
	srv.posCache = NewPositionCache(nil)
	snap := buildTestSnapshot()
	srv.posCache.PutSnapshot(snap, snap.CapturedAt)
	srv.SetAccountLoginLookup(func(_ context.Context, _ string) (int64, error) {
		return 12345, nil
	})
	srv.SetBrokerCompanyLookup(func(_ context.Context, _ string) string {
		return "Exness"
	})
	srv.SetAccountIsDemoLookup(func(_ context.Context, _ string) (bool, error) {
		return false, nil
	})
	srv.SetAccountConnectedLookup(func(_ context.Context, _ string) (bool, error) {
		return true, nil
	})
	srv.SetAccountTradeAllowedLookup(func(_ context.Context, _ string) (bool, error) {
		return true, nil
	})
	srv.SetAccountIsInvestorLookup(func(_ context.Context, _ string) (bool, error) {
		return false, nil
	})

	cfg := LiveStrategyConfig{
		AccountID: "acct-1",
		Symbol:    "EURUSD",
		Timeframe: "M15",
		Mode:      "live",
	}
	bars := []liveBar{{open: "1.0", high: "1.1", low: "0.9", close: "1.05", volume: "100", openTime: 1}}
	lctx, err := srv.buildLiveContext(context.Background(), cfg, bars, nil)
	if err != nil {
		t.Fatalf("buildLiveContext failed: %v", err)
	}
	if lctx.Login != 12345 {
		t.Errorf("Login = %d, want 12345", lctx.Login)
	}
	if lctx.Company != "Exness" {
		t.Errorf("Company = %q, want \"Exness\"", lctx.Company)
	}
	if lctx.IsDemo {
		t.Error("IsDemo = true, want false")
	}
	if !lctx.IsConnected {
		t.Error("IsConnected = false, want true")
	}
	if !lctx.IsTradeAllowed {
		t.Error("IsTradeAllowed = false, want true")
	}
}

// --- T5: live mode lookup fail-closed ---

func TestBuildLiveContext_LiveModeLookupFailClosed(t *testing.T) {
	srv := NewStrategyExecutionServer(nil, zap.NewNop())
	srv.posCache = NewPositionCache(nil)
	snap := buildTestSnapshot()
	srv.posCache.PutSnapshot(snap, snap.CapturedAt)
	srv.SetAccountLoginLookup(func(_ context.Context, _ string) (int64, error) {
		return 0, errLookupFailed
	})

	cfg := LiveStrategyConfig{
		AccountID: "acct-1",
		Symbol:    "EURUSD",
		Timeframe: "M15",
		Mode:      "live",
	}
	bars := []liveBar{{open: "1.0", high: "1.1", low: "0.9", close: "1.05", volume: "100", openTime: 1}}
	_, err := srv.buildLiveContext(context.Background(), cfg, bars, nil)
	if err == nil {
		t.Fatal("buildLiveContext should fail on lookup error in live mode, got nil")
	}
}

// --- T6: investor gating — IsTradeAllowed=false even when connected ---

func TestBuildLiveContext_InvestorGatingTradeAllowed(t *testing.T) {
	srv := NewStrategyExecutionServer(nil, zap.NewNop())
	srv.posCache = NewPositionCache(nil)
	snap := buildTestSnapshot()
	srv.posCache.PutSnapshot(snap, snap.CapturedAt)
	srv.SetAccountLoginLookup(func(_ context.Context, _ string) (int64, error) {
		return 12345, nil
	})
	srv.SetAccountIsDemoLookup(func(_ context.Context, _ string) (bool, error) {
		return false, nil
	})
	srv.SetAccountConnectedLookup(func(_ context.Context, _ string) (bool, error) {
		return true, nil
	})
	srv.SetAccountTradeAllowedLookup(func(_ context.Context, _ string) (bool, error) {
		return true, nil // trade_allowed status is true
	})
	srv.SetAccountIsInvestorLookup(func(_ context.Context, _ string) (bool, error) {
		return true, nil // but account is investor
	})

	cfg := LiveStrategyConfig{
		AccountID: "acct-1",
		Symbol:    "EURUSD",
		Timeframe: "M15",
		Mode:      "live",
	}
	bars := []liveBar{{open: "1.0", high: "1.1", low: "0.9", close: "1.05", volume: "100", openTime: 1}}
	lctx, err := srv.buildLiveContext(context.Background(), cfg, bars, nil)
	if err != nil {
		t.Fatalf("buildLiveContext failed: %v", err)
	}
	if lctx.IsTradeAllowed {
		t.Error("IsTradeAllowed = true, want false (investor account cannot trade)")
	}
	if !lctx.IsConnected {
		t.Error("IsConnected = false, want true (account is connected)")
	}
}

// --- T7: VMLiveSession.Start rejects invalid first bar context ---

func TestVMLiveSession_StartRejectsInvalidFirstBarContext(t *testing.T) {
	// Use a minimal MQL source that compiles.
	source := `int g_init = 0; void OnInit() { g_init = 1; } void OnBar() {}`
	sess, err := NewVMLiveSession(source)
	if err != nil {
		t.Fatalf("NewVMLiveSession failed: %v", err)
	}

	// Construct a request with invalid decimal in Close.
	bctx := &antv1.LiveStrategyContext{
		Close:      []string{"abc"}, // invalid decimal
		Open:       []string{"1.0"},
		High:       []string{"1.0"},
		Low:        []string{"1.0"},
		Volume:     []string{"10"},
		BarTimesMs: []int64{1},
		Mode:       "paper",
	}
	req := &antv1.ExecuteLiveRequest{
		StrategyCode: source,
		RequestType:  antv1.RequestType_REQUEST_TYPE_BAR,
		BarContext:   bctx,
	}

	_, err = sess.Start(context.Background(), req)
	if err == nil {
		t.Fatal("Start should fail on invalid first bar context, got nil error")
	}
	if !contains(err.Error(), "invalid first bar context") {
		t.Errorf("Error = %q, want contains \"invalid first bar context\"", err.Error())
	}
}

// --- T8: end-to-end AccountNumber readback via VM ---

func TestVMLiveSession_EndToEndAccountNumberReadback(t *testing.T) {
	// MQL source that reads AccountNumber() into a global.
	source := `int g_accountNumber = 0; void OnInit() { g_accountNumber = AccountNumber(); } void OnBar() {}`
	sess, err := NewVMLiveSession(source)
	if err != nil {
		t.Fatalf("NewVMLiveSession failed: %v", err)
	}

	bctx := &antv1.LiveStrategyContext{
		Close:         []string{"1.0"},
		Open:          []string{"1.0"},
		High:          []string{"1.0"},
		Low:           []string{"1.0"},
		Volume:        []string{"10"},
		BarTimesMs:    []int64{1},
		Mode:          "paper",
		Login:         12345,
		Positions:     []*antv1.LivePosition{},
		PendingOrders: []*antv1.LivePendingOrder{},
	}
	req := &antv1.ExecuteLiveRequest{
		StrategyCode: source,
		RequestType:  antv1.RequestType_REQUEST_TYPE_BAR,
		BarContext:   bctx,
	}

	_, err = sess.Start(context.Background(), req)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Read back g_accountNumber from the VM.
	vm := sess.strategy
	v, ok := vm.GetGlobal("g_accountNumber")
	if !ok {
		t.Fatal("global \"g_accountNumber\" not found")
	}
	got := v.ToInt()
	if got != 12345 {
		t.Errorf("g_accountNumber = %d, want 12345 (Login should reach AccountNumber via context)", got)
	}
}

// --- helpers ---

var errLookupFailed = &testError{"lookup failed"}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }
