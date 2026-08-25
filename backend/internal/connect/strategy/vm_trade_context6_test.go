package strategy

import (
	"context"
	"strings"
	"testing"
	"time"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"

	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"
)

// ── VM-TRADE-CONTEXT-6 behavior tests (返工第三阶段) ────────────────────

// helper to build a server with a fresh position cache for live context tests.
func newTestServerWithSnapshot(accountID string) *StrategyExecutionServer {
	srv := NewStrategyExecutionServer(nil, nil)
	pc := NewPositionCache(nil)
	snap := &mthub.PositionSnapshot{
		AccountID: accountID, Balance: decimal.NewFromInt(10000), Equity: decimal.NewFromInt(10000),
		Margin: decimal.Zero, FreeMargin: decimal.NewFromInt(10000), Leverage: 100,
		FinancialsAuthoritative: true, FinancialsSource: "account_summary",
		PositionsAuthoritative: true,
		CapturedAt:             time.Now(),
		PositionsCapturedAt:    time.Now(),
		PositionsSource:        "order_stream",
	}
	pc.PutSnapshot(snap, snap.CapturedAt)
	srv.posCache = pc
	return srv
}

// TestBuildLiveContext_InjectsLoginAndCompany verifies that buildLiveContext
// fills Login and Company from the authoritative lookups.
func TestBuildLiveContext_InjectsLoginAndCompany(t *testing.T) {
	srv := newTestServerWithSnapshot("acct-1")
	srv.SetAccountLoginLookup(func(_ context.Context, _ string) (int64, error) {
		return 123456, nil
	})
	srv.SetBrokerCompanyLookup(func(_ context.Context, _ string) (string, error) {
		return "ACME Broker", nil
	})
	srv.SetAccountIsDemoLookup(func(_ context.Context, _ string) (bool, error) {
		return true, nil
	})
	bars := []liveBar{{open: "1", high: "2", low: "0.5", close: "1.5", volume: "100", openTime: 100}}
	lctx, err := srv.buildLiveContext(context.Background(), LiveStrategyConfig{
		Symbol: "EURUSD", Timeframe: "M5", Mode: "paper", AccountID: "acct-1",
	}, bars, nil)
	if err != nil {
		t.Fatalf("buildLiveContext: %v", err)
	}
	if lctx.Login != 123456 {
		t.Errorf("Login=%d, want 123456 (should be injected from accountLoginLookup)", lctx.Login)
	}
	if lctx.Company != "ACME Broker" {
		t.Errorf("Company=%q, want %q (should be injected from brokerCompanyLookup)", lctx.Company, "ACME Broker")
	}
}

// TestBuildLiveContext_LiveModeLookupFailClosed verifies that in live mode,
// missing lookups fail closed (return error) rather than silently degrading
// to 0/"". VM-TRADE-CONTEXT-6 返工: lookup failure must not silently turn
// to 0/empty string.
func TestBuildLiveContext_LiveModeLookupFailClosed(t *testing.T) {
	srv := newTestServerWithSnapshot("acct-1")
	// No lookups configured — live mode must fail closed.
	bars := []liveBar{{open: "1", high: "2", low: "0.5", close: "1.5", volume: "100", openTime: 100}}
	_, err := srv.buildLiveContext(context.Background(), LiveStrategyConfig{
		Symbol: "EURUSD", Timeframe: "M5", Mode: "live", AccountID: "acct-1",
	}, bars, nil)
	if err == nil {
		t.Fatal("live mode with no accountLoginLookup should fail closed, not return nil error")
	}
}

// TestBuildLiveContext_LiveModeLoginZeroFailClosed verifies that in live mode,
// a login lookup returning 0 fails closed.
func TestBuildLiveContext_LiveModeLoginZeroFailClosed(t *testing.T) {
	srv := newTestServerWithSnapshot("acct-1")
	srv.SetAccountLoginLookup(func(_ context.Context, _ string) (int64, error) {
		return 0, nil // lookup failed — account not found
	})
	srv.SetBrokerCompanyLookup(func(_ context.Context, _ string) (string, error) {
		return "ACME Broker", nil
	})
	srv.SetAccountIsDemoLookup(func(_ context.Context, _ string) (bool, error) {
		return false, nil
	})
	bars := []liveBar{{open: "1", high: "2", low: "0.5", close: "1.5", volume: "100", openTime: 100}}
	_, err := srv.buildLiveContext(context.Background(), LiveStrategyConfig{
		Symbol: "EURUSD", Timeframe: "M5", Mode: "live", AccountID: "acct-1",
	}, bars, nil)
	if err == nil {
		t.Fatal("live mode with login=0 should fail closed, not silently use 0")
	}
}

// TestBuildLiveContext_LiveModeCompanyEmptyFailClosed verifies that in live
// mode, a broker company lookup returning "" fails closed.
func TestBuildLiveContext_LiveModeCompanyEmptyFailClosed(t *testing.T) {
	srv := newTestServerWithSnapshot("acct-1")
	srv.SetAccountLoginLookup(func(_ context.Context, _ string) (int64, error) {
		return 123456, nil
	})
	srv.SetBrokerCompanyLookup(func(_ context.Context, _ string) (string, error) {
		return "", nil // lookup failed
	})
	srv.SetAccountIsDemoLookup(func(_ context.Context, _ string) (bool, error) {
		return false, nil
	})
	bars := []liveBar{{open: "1", high: "2", low: "0.5", close: "1.5", volume: "100", openTime: 100}}
	_, err := srv.buildLiveContext(context.Background(), LiveStrategyConfig{
		Symbol: "EURUSD", Timeframe: "M5", Mode: "live", AccountID: "acct-1",
	}, bars, nil)
	if err == nil {
		t.Fatal("live mode with company=\"\" should fail closed, not silently use empty")
	}
}

// TestBuildLiveContext_PaperModeMissingLookupsOk verifies that in paper mode,
// missing lookups degrade gracefully (Login=0, Company="") — not fail closed.
func TestBuildLiveContext_PaperModeMissingLookupsOk(t *testing.T) {
	srv := newTestServerWithSnapshot("acct-1")
	bars := []liveBar{{open: "1", high: "2", low: "0.5", close: "1.5", volume: "100", openTime: 100}}
	lctx, err := srv.buildLiveContext(context.Background(), LiveStrategyConfig{
		Symbol: "EURUSD", Timeframe: "M5", Mode: "paper", AccountID: "acct-1",
	}, bars, nil)
	if err != nil {
		t.Fatalf("paper mode with no lookups should succeed, got: %v", err)
	}
	if lctx.Login != 0 {
		t.Errorf("Login=%d, want 0 (no lookup configured)", lctx.Login)
	}
	if lctx.Company != "" {
		t.Errorf("Company=%q, want empty (no lookup configured)", lctx.Company)
	}
}

// TestVMHandleBar_ArrayLengthMismatch verifies that vmHandleBar returns an
// error response when OHLCV array lengths don't match, instead of panicking.
func TestVMHandleBar_ArrayLengthMismatch(t *testing.T) {
	lctx := &antv1.LiveStrategyContext{
		Close:      []string{"1.5", "2.5"},
		Open:       []string{"1.0"}, // only 1 element — mismatch!
		High:       []string{"2.0", "3.0"},
		Low:        []string{"0.5", "1.5"},
		Volume:     []string{"100", "200"},
		BarTimesMs: []int64{100, 200},
		Symbol:     "EURUSD",
		Timeframe:  "M5",
	}
	resp := vmHandleBar(context.Background(), nil, lctx)
	if resp == nil {
		t.Fatal("vmHandleBar returned nil response")
	}
	if resp.Success {
		t.Fatal("vmHandleBar should return Success=false for array length mismatch")
	}
}

// TestVMHandleBar_NilPositionRejected verifies that a nil position in the
// repeated field is rejected (fail closed), not silently skipped.
// VM-TRADE-CONTEXT-6 返工: reject nil repeated messages.
func TestVMHandleBar_NilPositionRejected(t *testing.T) {
	lctx := &antv1.LiveStrategyContext{
		Close:      []string{"1.5"},
		Open:       []string{"1.0"},
		High:       []string{"2.0"},
		Low:        []string{"0.5"},
		Volume:     []string{"100"},
		BarTimesMs: []int64{100},
		Symbol:     "EURUSD",
		Timeframe:  "M5",
		Positions:  []*antv1.LivePosition{nil}, // nil position!
	}
	resp := vmHandleBar(context.Background(), nil, lctx)
	if resp == nil || resp.Success {
		t.Fatal("vmHandleBar should reject nil position with Success=false")
	}
	if !strings.Contains(resp.Error, "nil") {
		t.Fatalf("error should mention nil, got: %s", resp.Error)
	}
}

// TestVMHandleBar_NilPendingOrderRejected verifies that a nil pending order
// is rejected.
func TestVMHandleBar_NilPendingOrderRejected(t *testing.T) {
	lctx := &antv1.LiveStrategyContext{
		Close:         []string{"1.5"},
		Open:          []string{"1.0"},
		High:          []string{"2.0"},
		Low:           []string{"0.5"},
		Volume:        []string{"100"},
		BarTimesMs:    []int64{100},
		Symbol:        "EURUSD",
		Timeframe:     "M5",
		PendingOrders: []*antv1.LivePendingOrder{nil}, // nil pending order!
	}
	resp := vmHandleBar(context.Background(), nil, lctx)
	if resp == nil || resp.Success {
		t.Fatal("vmHandleBar should reject nil pending order with Success=false")
	}
	if !strings.Contains(resp.Error, "nil") {
		t.Fatalf("error should mention nil, got: %s", resp.Error)
	}
}

// TestVMHandleBar_NilSymbolSeriesRejected verifies that a nil symbol series
// is rejected.
func TestVMHandleBar_NilSymbolSeriesRejected(t *testing.T) {
	lctx := &antv1.LiveStrategyContext{
		Close:      []string{"1.5"},
		Open:       []string{"1.0"},
		High:       []string{"2.0"},
		Low:        []string{"0.5"},
		Volume:     []string{"100"},
		BarTimesMs: []int64{100},
		Symbol:     "EURUSD",
		Timeframe:  "M5",
		Symbols:    []*antv1.LiveSymbolSeries{nil}, // nil symbol series!
	}
	resp := vmHandleBar(context.Background(), nil, lctx)
	if resp == nil || resp.Success {
		t.Fatal("vmHandleBar should reject nil symbol series with Success=false")
	}
	if !strings.Contains(resp.Error, "nil") {
		t.Fatalf("error should mention nil, got: %s", resp.Error)
	}
}

// TestVMHandleBar_InvalidDecimalRejected verifies that an invalid decimal
// string in OHLCV is rejected (fail closed), not silently converted to zero.
// VM-TRADE-CONTEXT-6 返工: parseDecimalStrict in production path.
func TestVMHandleBar_InvalidDecimalRejected(t *testing.T) {
	lctx := &antv1.LiveStrategyContext{
		Close:      []string{"not_a_number"}, // invalid!
		Open:       []string{"1.0"},
		High:       []string{"2.0"},
		Low:        []string{"0.5"},
		Volume:     []string{"100"},
		BarTimesMs: []int64{100},
		Symbol:     "EURUSD",
		Timeframe:  "M5",
	}
	resp := vmHandleBar(context.Background(), nil, lctx)
	if resp == nil || resp.Success {
		t.Fatal("vmHandleBar should reject invalid decimal with Success=false")
	}
	if !strings.Contains(resp.Error, "close") {
		t.Fatalf("error should mention close field, got: %s", resp.Error)
	}
}

// TestVMHandleBar_InvalidVolumeRejected verifies that an invalid integer
// volume string is rejected (fail closed), not silently converted to zero.
// VM-TRADE-CONTEXT-6 返工: parseInt64Strict in production path.
func TestVMHandleBar_InvalidVolumeRejected(t *testing.T) {
	lctx := &antv1.LiveStrategyContext{
		Close:      []string{"1.5"},
		Open:       []string{"1.0"},
		High:       []string{"2.0"},
		Low:        []string{"0.5"},
		Volume:     []string{"not_an_int"}, // invalid!
		BarTimesMs: []int64{100},
		Symbol:     "EURUSD",
		Timeframe:  "M5",
	}
	resp := vmHandleBar(context.Background(), nil, lctx)
	if resp == nil || resp.Success {
		t.Fatal("vmHandleBar should reject invalid volume with Success=false")
	}
	if !strings.Contains(resp.Error, "volume") {
		t.Fatalf("error should mention volume field, got: %s", resp.Error)
	}
}

// TestVMHandleBar_LegitimateZeroAccepted verifies that explicit "0" strings
// are accepted as valid zero values (not rejected as empty/invalid).
// VM-TRADE-CONTEXT-6: "0" is a legitimate zero value, only "" is rejected.
func TestVMHandleBar_LegitimateZeroAccepted(t *testing.T) {
	lctx := &antv1.LiveStrategyContext{
		Close:      []string{"0"},
		Open:       []string{"0"},
		High:       []string{"0"},
		Low:        []string{"0"},
		Volume:     []string{"0"},
		BarTimesMs: []int64{100},
		Symbol:     "EURUSD",
		Timeframe:  "M5",
	}
	// nil runner will panic after validation passes, so we use recover.
	defer func() {
		if r := recover(); r != nil {
			t.Logf("expected panic from nil runner (zero values accepted): %v", r)
		}
	}()
	resp := vmHandleBar(context.Background(), nil, lctx)
	if resp != nil && !resp.Success && (strings.Contains(resp.Error, "invalid decimal") || strings.Contains(resp.Error, "invalid integer")) {
		t.Fatalf("legitimate zero \"0\" should be accepted, got: %s", resp.Error)
	}
}

// TestVMHandleTick_InvalidBidRejected verifies that an invalid bid in tick
// context is rejected (fail closed).
func TestVMHandleTick_InvalidBidRejected(t *testing.T) {
	tctx := &antv1.TickContext{
		Bid:    "not_a_number",
		Ask:    "1.1",
		Symbol: "EURUSD",
	}
	resp := vmHandleTick(context.Background(), nil, tctx)
	if resp == nil || resp.Success {
		t.Fatal("vmHandleTick should reject invalid bid with Success=false")
	}
	if !strings.Contains(resp.Error, "bid") {
		t.Fatalf("error should mention bid, got: %s", resp.Error)
	}
}

// TestVMHandleTrade_InvalidVolumeRejected verifies that an invalid volume
// in trade context is rejected (fail closed).
func TestVMHandleTrade_InvalidVolumeRejected(t *testing.T) {
	evctx := &antv1.TradeContext{
		Side:      "buy",
		EventType: "fill",
		Volume:    "not_a_number",
		Price:     "1.1",
		Symbol:    "EURUSD",
	}
	resp := vmHandleTrade(context.Background(), nil, evctx)
	if resp == nil || resp.Success {
		t.Fatal("vmHandleTrade should reject invalid volume with Success=false")
	}
	if !strings.Contains(resp.Error, "volume") {
		t.Fatalf("error should mention volume, got: %s", resp.Error)
	}
}

// TestValidateFirstBarContext_EmptyArraysRejected verifies that the first bar
// context (used for OnInit) must have at least 1 bar — empty arrays are
// rejected so OnInit does not run with no data.
// VM-TRADE-CONTEXT-6 返工: bad request must not execute OnInit then fail.
func TestValidateFirstBarContext_EmptyArraysRejected(t *testing.T) {
	bctx := &antv1.LiveStrategyContext{
		Close:      []string{},
		Open:       []string{},
		High:       []string{},
		Low:        []string{},
		Volume:     []string{},
		BarTimesMs: []int64{},
	}
	err := validateFirstBarContext(bctx)
	if err == nil {
		t.Fatal("validateFirstBarContext should reject empty arrays for OnInit")
	}
}

// TestValidateFirstBarContext_InvalidDecimalRejected verifies that invalid
// decimals in the first bar context are rejected before OnInit.
func TestValidateFirstBarContext_InvalidDecimalRejected(t *testing.T) {
	bctx := &antv1.LiveStrategyContext{
		Close:      []string{"bad"},
		Open:       []string{"1.0"},
		High:       []string{"2.0"},
		Low:        []string{"0.5"},
		Volume:     []string{"100"},
		BarTimesMs: []int64{100},
	}
	err := validateFirstBarContext(bctx)
	if err == nil {
		t.Fatal("validateFirstBarContext should reject invalid decimal before OnInit")
	}
}

// TestValidateFirstBarContext_NilPositionRejected verifies that nil positions
// in the first bar context are rejected before OnInit.
func TestValidateFirstBarContext_NilPositionRejected(t *testing.T) {
	bctx := &antv1.LiveStrategyContext{
		Close:      []string{"1.5"},
		Open:       []string{"1.0"},
		High:       []string{"2.0"},
		Low:        []string{"0.5"},
		Volume:     []string{"100"},
		BarTimesMs: []int64{100},
		Positions:  []*antv1.LivePosition{nil},
	}
	err := validateFirstBarContext(bctx)
	if err == nil {
		t.Fatal("validateFirstBarContext should reject nil position before OnInit")
	}
}

// TestValidateFirstBarContext_ValidPasses verifies that a valid bar context
// passes validation.
func TestValidateFirstBarContext_ValidPasses(t *testing.T) {
	bctx := &antv1.LiveStrategyContext{
		Close:      []string{"1.5"},
		Open:       []string{"1.0"},
		High:       []string{"2.0"},
		Low:        []string{"0.5"},
		Volume:     []string{"100"},
		BarTimesMs: []int64{100},
	}
	err := validateFirstBarContext(bctx)
	if err != nil {
		t.Fatalf("validateFirstBarContext should pass for valid context, got: %v", err)
	}
}

// TestParseDecimalStrict_Valid verifies that parseDecimalStrict correctly
// parses valid decimal strings.
func TestParseDecimalStrict_Valid(t *testing.T) {
	d, err := parseDecimalStrict("1.23456")
	if err != nil {
		t.Fatalf("parseDecimalStrict: %v", err)
	}
	want, _ := decimal.NewFromString("1.23456")
	if !d.Equal(want) {
		t.Errorf("parseDecimalStrict = %s, want 1.23456", d.String())
	}
}

// TestParseDecimalStrict_LegitimateZero verifies that "0" is accepted.
func TestParseDecimalStrict_LegitimateZero(t *testing.T) {
	d, err := parseDecimalStrict("0")
	if err != nil {
		t.Fatalf("parseDecimalStrict(\"0\") should succeed: %v", err)
	}
	if !d.IsZero() {
		t.Errorf("parseDecimalStrict(\"0\") = %s, want 0", d.String())
	}
}

// TestParseDecimalStrict_Empty verifies that parseDecimalStrict returns
// an error for empty strings (not silent zero).
func TestParseDecimalStrict_Empty(t *testing.T) {
	_, err := parseDecimalStrict("")
	if err == nil {
		t.Fatal("parseDecimalStrict(\"\") should return error, not silent zero")
	}
}

// TestParseDecimalStrict_Invalid verifies that parseDecimalStrict returns
// an error for invalid strings (not silent zero).
func TestParseDecimalStrict_Invalid(t *testing.T) {
	_, err := parseDecimalStrict("not_a_number")
	if err == nil {
		t.Fatal("parseDecimalStrict(\"not_a_number\") should return error")
	}
}

// TestParseInt64Strict_Valid verifies that parseInt64Strict correctly parses
// valid integer strings.
func TestParseInt64Strict_Valid(t *testing.T) {
	n, err := parseInt64Strict("100")
	if err != nil {
		t.Fatalf("parseInt64Strict: %v", err)
	}
	if n != 100 {
		t.Errorf("parseInt64Strict = %d, want 100", n)
	}
}

// TestParseInt64Strict_LegitimateZero verifies that "0" is accepted.
func TestParseInt64Strict_LegitimateZero(t *testing.T) {
	n, err := parseInt64Strict("0")
	if err != nil {
		t.Fatalf("parseInt64Strict(\"0\") should succeed: %v", err)
	}
	if n != 0 {
		t.Errorf("parseInt64Strict(\"0\") = %d, want 0", n)
	}
}

// TestParseInt64Strict_Empty verifies that parseInt64Strict returns an error
// for empty strings.
func TestParseInt64Strict_Empty(t *testing.T) {
	_, err := parseInt64Strict("")
	if err == nil {
		t.Fatal("parseInt64Strict(\"\") should return error")
	}
}

// TestParseInt64Strict_Invalid verifies that parseInt64Strict returns an error
// for invalid strings.
func TestParseInt64Strict_Invalid(t *testing.T) {
	_, err := parseInt64Strict("not_an_int")
	if err == nil {
		t.Fatal("parseInt64Strict(\"not_an_int\") should return error")
	}
}

// TestVMLiveSession_EndToEndAccountNumberReadback verifies the full chain:
// VMLiveSession.Start → OnInit → AccountNumber()/AccountCompany() →
// global readback. VM-TRADE-CONTEXT-6 返工: end-to-end test.
func TestVMLiveSession_EndToEndAccountNumberReadback(t *testing.T) {
	// MQL strategy that stores AccountNumber/AccountCompany in globals
	// during OnInit and exposes them via helper functions.
	source := `
int g_account = 0;
string g_company = "";

int OnInit() {
    g_account = AccountNumber();
    g_company = AccountCompany();
    return 0;
}

int GetStoredAccount() { return g_account; }
string GetStoredCompany() { return g_company; }

int OnBar() { return 0; }
`
	sess, err := NewVMLiveSession(source)
	if err != nil {
		t.Fatalf("NewVMLiveSession: %v", err)
	}

	// Build first request with bar_context containing Login/Company.
	bctx := &antv1.LiveStrategyContext{
		Close:      []string{"1.5"},
		Open:       []string{"1.0"},
		High:       []string{"2.0"},
		Low:        []string{"0.5"},
		Volume:     []string{"100"},
		BarTimesMs: []int64{100},
		Symbol:     "EURUSD",
		Timeframe:  "M5",
		Login:      987654,
		Company:    "TestBroker Ltd",
	}
	req := &antv1.ExecuteLiveRequest{
		RequestType: antv1.RequestType_REQUEST_TYPE_BAR,
		BarContext:  bctx,
	}
	reqBytes, err := proto.Marshal(req)
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}

	respBytes, err := sess.Start(context.Background(), reqBytes)
	if err != nil {
		t.Fatalf("VMLiveSession.Start: %v", err)
	}

	var resp antv1.ExecuteLiveResponse
	if err := proto.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("proto.Unmarshal: %v", err)
	}
	if !resp.Success {
		t.Fatalf("Start response not success: %s", resp.Error)
	}

	// Read back the stored values from the VM globals.
	v, ok := sess.strategy.GetGlobal("g_account")
	if !ok {
		t.Fatal("global g_account not found")
	}
	account := v.ToInt()
	if account != 987654 {
		t.Errorf("AccountNumber readback = %d, want 987654 (OnInit should store AccountNumber)", account)
	}
	cv, ok := sess.strategy.GetGlobal("g_company")
	if !ok {
		t.Fatal("global g_company not found")
	}
	company := cv.ToString()
	if company != "TestBroker Ltd" {
		t.Errorf("AccountCompany readback = %q, want %q (OnInit should store AccountCompany)", company, "TestBroker Ltd")
	}
}
