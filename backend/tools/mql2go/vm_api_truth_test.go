package mql2go

import (
	"fmt"
	"strings"
	"testing"

	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"

	"github.com/shopspring/decimal"
)

// VM-API-TRUTH-1: MQL5 order/deal/history 22 API reclassified StatusUnsupported.
// These were stubs returning safe defaults (true/0/""/false) — strategies could
// run on fake data. Now the compiler rejects them at compile time (fail-closed).

// unsupportedMQL5History lists the 22 MQL5 order/deal/history API names that
// VM-API-TRUTH-1 reclassified from implemented (stub) to StatusUnsupported.
var unsupportedMQL5History = []string{
	"OrderCalcMargin", "OrderCalcProfit", "OrderCheck",
	"OrderGetTicket", "OrderGetDouble", "OrderGetInteger", "OrderGetString",
	"OrdersTotalMQL5",
	"HistorySelect", "HistorySelectByPosition",
	"HistoryDealsTotal", "HistoryDealSelect", "HistoryDealGetTicket",
	"HistoryDealGetDouble", "HistoryDealGetInteger", "HistoryDealGetString",
	"HistoryOrdersTotal", "HistoryOrderSelect", "HistoryOrderGetTicket",
	"HistoryOrderGetDouble", "HistoryOrderGetInteger", "HistoryOrderGetString",
}

// TestVM_API_TRUTH_1_MQL5HistoryRejected verifies each of the 22 APIs causes a
// compile-time error (not silent acceptance with fake runtime data).
//
// Adversarial: restore any of the 22 to implementedMQL5Position + remove from
// unsupportedSymbols → CompileMQL succeeds → RED.
func TestVM_API_TRUTH_1_MQL5HistoryRejected(t *testing.T) {
	for _, api := range unsupportedMQL5History {
		t.Run(api, func(t *testing.T) {
			// Minimal source: call the API with zero args. The compiler looks up
			// the name in the API registry before checking arg count, so an
			// unsupported function is rejected regardless of arg count.
			src := "int OnInit() { return 0; }\nvoid OnTick() { " + api + "(); }"
			_, err := CompileMQL(src)
			if err == nil {
				t.Fatalf("%s: expected compile error (StatusUnsupported), got nil — API silently accepted", api)
			}
			// Error message must mention the API name (or "unsupported").
			msg := strings.ToLower(err.Error())
			if !strings.Contains(msg, "unsupported") && !strings.Contains(msg, strings.ToLower(api)) {
				t.Fatalf("%s: error message must mention 'unsupported' or API name, got: %v", api, err)
			}
		})
	}
}

// TestVM_API_TRUTH_1_RegistryConsistency verifies the API registry reflects
// the reclassification: each of the 22 APIs is StatusUnsupported with a
// non-empty reason, IsAPIImplemented=false, IsAPIUnsupported=true.
//
// Adversarial: remove the 22 entries from unsupportedSymbols → LookupAPI
// returns false (or StatusImplemented if also in implementedMQL5Position) → RED.
func TestVM_API_TRUTH_1_RegistryConsistency(t *testing.T) {
	for _, api := range unsupportedMQL5History {
		t.Run(api, func(t *testing.T) {
			sym, ok := interp.LookupAPI(api)
			if !ok {
				t.Fatalf("%s: LookupAPI returned not-found — missing from registry", api)
			}
			if sym.Status != interp.StatusUnsupported {
				t.Fatalf("%s: status = %v, want StatusUnsupported", api, sym.Status)
			}
			if sym.Reason == "" {
				t.Fatalf("%s: Reason is empty — must explain why unsupported", api)
			}
			if interp.IsAPIImplemented(api) {
				t.Fatalf("%s: IsAPIImplemented=true, want false", api)
			}
			if !interp.IsAPIUnsupported(api) {
				t.Fatalf("%s: IsAPIUnsupported=false, want true", api)
			}
		})
	}
}

// TestVM_API_TRUTH_1_PositionSelectStillImplemented verifies the reclassification
// only affects the 22 MQL5 order/deal/history APIs — PositionSelect (which has a
// real implementation delegating to PositionSelectByTicket) is NOT误伤.
//
// Adversarial: accidentally remove PositionSelect from implementedMQL5Position
// or add it to unsupportedSymbols → IsAPIImplemented=false → RED.
func TestVM_API_TRUTH_1_PositionSelectStillImplemented(t *testing.T) {
	sym, ok := interp.LookupAPI("PositionSelect")
	if !ok {
		t.Fatal("PositionSelect: LookupAPI returned not-found — accidentally removed from registry")
	}
	if sym.Status != interp.StatusImplemented {
		t.Fatalf("PositionSelect: status = %v, want StatusImplemented (VM-API-TRUTH-1 must not误伤 PositionSelect)", sym.Status)
	}
	if !interp.IsAPIImplemented("PositionSelect") {
		t.Fatal("PositionSelect: IsAPIImplemented=false, want true (VM-API-TRUTH-1 must not误伤 PositionSelect)")
	}
	if interp.IsAPIUnsupported("PositionSelect") {
		t.Fatal("PositionSelect: IsAPIUnsupported=true, want false (VM-API-TRUTH-1 must not误伤 PositionSelect)")
	}
}

// VM-API-TRUTH-1 batch 2a: platform checkup 12 API reclassified StatusUnsupported.
// These were fixed-value stubs (true/false/0/""/NoneVal) — strategies could
// run on fake platform/terminal data. Now the compiler rejects them.

// unsupportedPlatformCheckup lists the 12 platform checkup API names that
// VM-API-TRUTH-1 batch 2a reclassified from implemented (fixed-value stub)
// to StatusUnsupported.
var unsupportedPlatformCheckup = []string{
	"IsDllsAllowed", "IsExpertEnabled", "IsLibrariesAllowed",
	"IsTradeContextBusy", "IsStopped", "UninitializeReason",
	"MQLInfoInteger", "MQLInfoString",
	"TerminalInfoDouble", "TerminalInfoInteger", "TerminalInfoString",
	"SetReturnError",
}

// TestVM_API_TRUTH_1_PlatformCheckupRejected verifies each of the 12 APIs
// causes a compile-time error (not silent acceptance with fake platform data).
//
// Adversarial: restore any of the 12 to implementedPlatform + remove from
// unsupportedSymbols → CompileMQL succeeds → RED.
func TestVM_API_TRUTH_1_PlatformCheckupRejected(t *testing.T) {
	for _, api := range unsupportedPlatformCheckup {
		t.Run(api, func(t *testing.T) {
			src := "int OnInit() { return 0; }\nvoid OnTick() { " + api + "(); }"
			_, err := CompileMQL(src)
			if err == nil {
				t.Fatalf("%s: expected compile error (StatusUnsupported), got nil — API silently accepted", api)
			}
			msg := strings.ToLower(err.Error())
			if !strings.Contains(msg, "unsupported") && !strings.Contains(msg, strings.ToLower(api)) {
				t.Fatalf("%s: error message must mention 'unsupported' or API name, got: %v", api, err)
			}
		})
	}
}

// TestVM_API_TRUTH_1_PlatformCheckupRegistryConsistency verifies the API
// registry reflects the batch 2a reclassification: each of the 12 APIs is
// StatusUnsupported with a non-empty reason, IsAPIImplemented=false,
// IsAPIUnsupported=true.
//
// Adversarial: remove the 12 entries from unsupportedSymbols → LookupAPI
// returns not-found (or StatusImplemented if also in implementedPlatform)
// → RED.
func TestVM_API_TRUTH_1_PlatformCheckupRegistryConsistency(t *testing.T) {
	for _, api := range unsupportedPlatformCheckup {
		t.Run(api, func(t *testing.T) {
			sym, ok := interp.LookupAPI(api)
			if !ok {
				t.Fatalf("%s: LookupAPI returned not-found — missing from registry", api)
			}
			if sym.Status != interp.StatusUnsupported {
				t.Fatalf("%s: status = %v, want StatusUnsupported", api, sym.Status)
			}
			if sym.Reason == "" {
				t.Fatalf("%s: Reason is empty — must explain why unsupported", api)
			}
			if interp.IsAPIImplemented(api) {
				t.Fatalf("%s: IsAPIImplemented=true, want false", api)
			}
			if !interp.IsAPIUnsupported(api) {
				t.Fatalf("%s: IsAPIUnsupported=false, want true", api)
			}
		})
	}
}

// TestVM_API_TRUTH_1_PlatformCheckupRealStillImplemented verifies the batch 2a
// reclassification only affects the 12 fixed-value stubs — the real
// implementations (IsConnected/IsDemo/IsTradeAllowed via VM-API-TRUTH-3,
// GetLastError/ResetLastError/SetUserError via lastError state machine,
// CurTime/GetTickCount* via real time sources, IsTesting/IsOptimization/
// IsVisualMode via backtest semantics) are NOT误伤.
//
// Adversarial: accidentally remove any of these from implementedPlatform or
// add to unsupportedSymbols → IsAPIImplemented=false → RED.
func TestVM_API_TRUTH_1_PlatformCheckupRealStillImplemented(t *testing.T) {
	realImplemented := []string{
		"IsConnected", "IsDemo", "IsTradeAllowed",
		"GetLastError", "ResetLastError", "SetUserError",
		"CurTime", "GetTickCount", "GetTickCount64", "GetMicrosecondCount",
		"IsTesting", "IsOptimization", "IsVisualMode",
	}
	for _, api := range realImplemented {
		t.Run(api, func(t *testing.T) {
			if !interp.IsAPIImplemented(api) {
				t.Fatalf("%s: IsAPIImplemented=false, want true (VM-API-TRUTH-1 batch 2a must not误伤 real implementations)", api)
			}
			if interp.IsAPIUnsupported(api) {
				t.Fatalf("%s: IsAPIUnsupported=true, want false (VM-API-TRUTH-1 batch 2a must not误伤 real implementations)", api)
			}
		})
	}
}

// VM-API-TRUTH-1 batch 2b: account/symbol stub 5 API reclassified
// StatusUnsupported. AccountStopoutMode/AccountCredit returned fixed 0 with no
// authoritative data; SymbolInfoMarginRate/SymbolInfoSessionQuote/
// SymbolInfoSessionTrade returned true without filling their by-reference
// outputs. Now the compiler rejects them (fail-closed).

// unsupportedAccountSymbolStub lists the 5 account/symbol stub API names that
// VM-API-TRUTH-1 batch 2b reclassified from implemented (fixed-value stub /
// by-reference unfilled) to StatusUnsupported.
var unsupportedAccountSymbolStub = []string{
	"AccountStopoutMode", "AccountCredit",
	"SymbolInfoMarginRate", "SymbolInfoSessionQuote", "SymbolInfoSessionTrade",
}

// TestVM_API_TRUTH_1_AccountSymbolStubRejected verifies each of the 5 APIs
// causes a compile-time error (not silent acceptance with fake data).
//
// Adversarial: restore any of the 5 to implementedAccount/implementedPlatform
// + remove from unsupportedSymbols → CompileMQL succeeds → RED.
func TestVM_API_TRUTH_1_AccountSymbolStubRejected(t *testing.T) {
	for _, api := range unsupportedAccountSymbolStub {
		t.Run(api, func(t *testing.T) {
			src := "int OnInit() { return 0; }\nvoid OnTick() { " + api + "(); }"
			_, err := CompileMQL(src)
			if err == nil {
				t.Fatalf("%s: expected compile error (StatusUnsupported), got nil — API silently accepted", api)
			}
			msg := strings.ToLower(err.Error())
			if !strings.Contains(msg, "unsupported") && !strings.Contains(msg, strings.ToLower(api)) {
				t.Fatalf("%s: error message must mention 'unsupported' or API name, got: %v", api, err)
			}
		})
	}
}

// TestVM_API_TRUTH_1_AccountSymbolStubRegistryConsistency verifies the API
// registry reflects the batch 2b reclassification: each of the 5 APIs is
// StatusUnsupported with a non-empty reason, IsAPIImplemented=false,
// IsAPIUnsupported=true.
//
// Adversarial: remove the 5 entries from unsupportedSymbols → LookupAPI
// returns not-found (or StatusImplemented if also in implemented* lists)
// → RED.
func TestVM_API_TRUTH_1_AccountSymbolStubRegistryConsistency(t *testing.T) {
	for _, api := range unsupportedAccountSymbolStub {
		t.Run(api, func(t *testing.T) {
			sym, ok := interp.LookupAPI(api)
			if !ok {
				t.Fatalf("%s: LookupAPI returned not-found — missing from registry", api)
			}
			if sym.Status != interp.StatusUnsupported {
				t.Fatalf("%s: status = %v, want StatusUnsupported", api, sym.Status)
			}
			if sym.Reason == "" {
				t.Fatalf("%s: Reason is empty — must explain why unsupported", api)
			}
			if interp.IsAPIImplemented(api) {
				t.Fatalf("%s: IsAPIImplemented=true, want false", api)
			}
			if !interp.IsAPIUnsupported(api) {
				t.Fatalf("%s: IsAPIUnsupported=false, want true", api)
			}
		})
	}
}

// TestVM_API_TRUTH_1_AccountSymbolStubRealStillImplemented verifies the batch
// 2b reclassification only affects the 5 stubs — AccountInfoDouble/Integer/
// String (mixed real+stub, batch 2c scope), SymbolSelect/SymbolsTotal/
// SymbolIsSynchronized (fixed values are correct backtest semantics for a
// single-symbol run, like IsTesting), SymbolInfoTick/SymbolName (real
// implementations reading vm.ctx) and AccountBalance/Equity/Margin/Leverage
// (real implementations) are NOT误伤.
//
// Adversarial: accidentally remove any of these from implemented* lists or
// add to unsupportedSymbols → IsAPIImplemented=false → RED.
func TestVM_API_TRUTH_1_AccountSymbolStubRealStillImplemented(t *testing.T) {
	realImplemented := []string{
		// Batch 2c scope: mixed AccountInfo* (real BALANCE/EQUITY/MARGIN/LEVERAGE
		// branches must stay while stub branches get fixed).
		"AccountInfoDouble", "AccountInfoInteger", "AccountInfoString",
		// Backtest-semantics fixed values (single-symbol run).
		"SymbolSelect", "SymbolsTotal", "SymbolIsSynchronized",
		// Real implementations reading vm.ctx.
		"SymbolInfoTick", "SymbolName",
		// Real account implementations (vm_builtin_account.go).
		"AccountBalance", "AccountEquity", "AccountMargin", "AccountLeverage",
	}
	for _, api := range realImplemented {
		t.Run(api, func(t *testing.T) {
			if !interp.IsAPIImplemented(api) {
				t.Fatalf("%s: IsAPIImplemented=false, want true (VM-API-TRUTH-1 batch 2b must not误伤 real/semantically-valid implementations)", api)
			}
			if interp.IsAPIUnsupported(api) {
				t.Fatalf("%s: IsAPIUnsupported=true, want false (VM-API-TRUTH-1 batch 2b must not误伤 real/semantically-valid implementations)", api)
			}
		})
	}
}

// VM-API-TRUTH-1 batch 2c: AccountInfoDouble/Integer/String fake branches
// fail-closed or rewired to authoritative sources; prop numbering aligned to
// the real MQL5 enums; named enum constants added so real MQL5 sources
// compile. NOT a reclassification — the 3 functions stay implemented.

// TestVM_API_TRUTH_1_AccountInfoLeverageReal verifies the named constant
// ACCOUNT_LEVERAGE compiles, resolves to the real MQL5 value 2, and the
// leverage branch reads ctx.Account().Leverage.
//
// Adversarial: revert case 2 to old fake numbering case 32 → prop 2 falls to
// default → OnInit error → RED; delete "ACCOUNT_LEVERAGE" from constants.go →
// unknown identifier compile error → RED.
func TestVM_API_TRUTH_1_AccountInfoLeverageReal(t *testing.T) {
	src := "long g=0; int OnInit(){ g=AccountInfoInteger(ACCOUNT_LEVERAGE); return 0; }"
	runner := compileAndInit(t, src, &accountStatusTestContext{leverage: 25})
	v, ok := runner.GetGlobal("g")
	if !ok {
		t.Fatal("global \"g\" not found")
	}
	if got := v.ToInt(); got != 25 {
		t.Errorf("g = %d, want 25 (ctx leverage)", got)
	}
}

// TestVM_API_TRUTH_1_AccountInfoTradeAllowedReadsSource verifies
// ACCOUNT_TRADE_ALLOWED reads ctx.Account().IsTradeAllowed instead of the old
// fixed BoolVal(true) (which also leaked a bool through a long-returning API).
//
// Adversarial: restore case 5 to `return interp.BoolVal(true), nil` →
// IsTradeAllowed=false subtest reads 1 → RED.
func TestVM_API_TRUTH_1_AccountInfoTradeAllowedReadsSource(t *testing.T) {
	src := "long g=1; int OnInit(){ g=AccountInfoInteger(ACCOUNT_TRADE_ALLOWED); return 0; }"
	for _, tc := range []struct {
		allowed bool
		want    int32
	}{
		{false, 0},
		{true, 1},
	} {
		t.Run(fmt.Sprintf("IsTradeAllowed=%v", tc.allowed), func(t *testing.T) {
			runner := compileAndInit(t, src, &accountStatusTestContext{isTradeAllowed: tc.allowed})
			v, ok := runner.GetGlobal("g")
			if !ok {
				t.Fatal("global \"g\" not found")
			}
			if got := v.ToInt(); got != tc.want {
				t.Errorf("g = %d, want %d (IsTradeAllowed=%v)", got, tc.want, tc.allowed)
			}
		})
	}
}

// TestVM_API_TRUTH_1_AccountInfoNewRealBranches verifies the newly wired real
// branches read their authoritative sources.
//
// Adversarial: break any branch (wrong enum number, ignored source field) →
// the subtest reads the old fixed value → RED.
func TestVM_API_TRUTH_1_AccountInfoNewRealBranches(t *testing.T) {
	cases := []struct {
		name string
		prop string
		ctx  *accountStatusTestContext
		want int32
	}{
		{"LOGIN", "ACCOUNT_LOGIN", &accountStatusTestContext{login: 77}, 77},
		{"TRADE_MODE_DEMO", "ACCOUNT_TRADE_MODE", &accountStatusTestContext{isDemo: true}, 0},
		{"TRADE_MODE_REAL", "ACCOUNT_TRADE_MODE", &accountStatusTestContext{isDemo: false}, 2},
		{"MARGIN_MODE_HEDGING", "ACCOUNT_MARGIN_MODE", &accountStatusTestContext{mode: sdk.ModeHedging}, 2},
		{"MARGIN_MODE_NETTING", "ACCOUNT_MARGIN_MODE", &accountStatusTestContext{mode: sdk.ModeNetting}, 0},
		{"HEDGE_ALLOWED_HEDGING", "ACCOUNT_HEDGE_ALLOWED", &accountStatusTestContext{mode: sdk.ModeHedging}, 1},
		{"HEDGE_ALLOWED_NETTING", "ACCOUNT_HEDGE_ALLOWED", &accountStatusTestContext{mode: sdk.ModeNetting}, 0},
		{"TRADE_EXPERT_ALLOWED", "ACCOUNT_TRADE_EXPERT", &accountStatusTestContext{isTradeAllowed: true}, 1},
		{"TRADE_EXPERT_DENIED", "ACCOUNT_TRADE_EXPERT", &accountStatusTestContext{isTradeAllowed: false}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := "long g=-1; int OnInit(){ g=AccountInfoInteger(" + tc.prop + "); return 0; }"
			runner := compileAndInit(t, src, tc.ctx)
			v, ok := runner.GetGlobal("g")
			if !ok {
				t.Fatalf("global \"g\" not found (%s)", tc.prop)
			}
			if got := v.ToInt(); got != tc.want {
				t.Errorf("AccountInfoInteger(%s) = %d, want %d", tc.prop, got, tc.want)
			}
		})
	}
}

// TestVM_API_TRUTH_1_AccountInfoFakePropsError verifies props without an
// authoritative source fail-closed: the builtin returns an error →
// callBuiltin sets fatalError → OnInit reports the error.
//
// Adversarial: restore any fake branch (fixed 0/true/"") → OnInit succeeds
// silently on fake data → RED.
func TestVM_API_TRUTH_1_AccountInfoFakePropsError(t *testing.T) {
	cases := []struct {
		call string
	}{
		{"AccountInfoDouble(1)"},
		{"AccountInfoDouble(7)"},
		{"AccountInfoDouble(99)"},
		{"AccountInfoInteger(3)"},
		{"AccountInfoInteger(4)"},
		{"AccountInfoInteger(8)"},
		{"AccountInfoInteger(9)"},
		{"AccountInfoInteger(99)"},
		{"AccountInfoString(0)"},
		{"AccountInfoString(1)"},
		{"AccountInfoString(99)"},
	}
	for _, tc := range cases {
		t.Run(tc.call, func(t *testing.T) {
			src := "int OnInit(){ " + tc.call + "; return 0; }"
			runner, err := CompileMQL(src)
			if err != nil {
				t.Fatalf("CompileMQL failed: %v", err)
			}
			runner.SetSignalMode(true)
			if err := runner.OnInit(&accountStatusTestContext{}); err == nil {
				t.Fatalf("%s: OnInit err = nil, want error (prop without authoritative source must fail-closed, not return fake value)", tc.call)
			}
		})
	}
}

// TestVM_API_TRUTH_1_AccountInfoEnumNumbering pins the enum renumbering: the
// old fake numbering (32=LEVERAGE) now errors, and the real
// ENUM_ACCOUNT_INFO_STRING numbering (CURRENCY=2) reads the source with
// fail-closed on empty.
//
// Adversarial: restore case 32 → AccountInfoInteger(32) returns leverage
// silently → OnInit succeeds → RED; restore fixed "USD" → currency="" subtest
// returns "USD" → RED.
func TestVM_API_TRUTH_1_AccountInfoEnumNumbering(t *testing.T) {
	// Old fake prop 32 (was LEVERAGE) is out of the real enum → error.
	runner, err := CompileMQL("int OnInit(){ AccountInfoInteger(32); return 0; }")
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}
	runner.SetSignalMode(true)
	if err := runner.OnInit(&accountStatusTestContext{leverage: 25}); err == nil {
		t.Fatal("AccountInfoInteger(32): OnInit err = nil, want error (old fake numbering must fail-closed)")
	}

	// Real ACCOUNT_CURRENCY (=2) reads the source.
	src := "string g=\"\"; int OnInit(){ g=AccountInfoString(ACCOUNT_CURRENCY); return 0; }"
	runner2 := compileAndInit(t, src, &accountStatusTestContext{currency: "EUR"})
	v, ok := runner2.GetGlobal("g")
	if !ok {
		t.Fatal("global \"g\" not found")
	}
	if got := v.ToString(); got != "EUR" {
		t.Errorf("g = %q, want %q (ctx currency)", got, "EUR")
	}

	// Empty currency = missing fact → fail-closed.
	runner3, err := CompileMQL(src)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}
	runner3.SetSignalMode(true)
	if err := runner3.OnInit(&accountStatusTestContext{}); err == nil {
		t.Fatal("AccountInfoString(ACCOUNT_CURRENCY) with empty ctx currency: OnInit err = nil, want error (empty = missing fact)")
	}
}

// TestVM_API_TRUTH_1_AccountInfoConstantValues pins the corrected enum
// constant values — the double constants SO_CALL/SO_SO/INITIAL/MAINTENANCE
// were mutually transposed before batch 2c.
//
// Adversarial: change any value back to the old wrong numbering → RED.
func TestVM_API_TRUTH_1_AccountInfoConstantValues(t *testing.T) {
	cases := []struct {
		name string
		want int32
	}{
		// ENUM_ACCOUNT_INFO_INTEGER
		{"ACCOUNT_LOGIN", 0},
		{"ACCOUNT_LEVERAGE", 2},
		{"ACCOUNT_TRADE_ALLOWED", 5},
		{"ACCOUNT_TRADE_EXPERT", 6},
		// ENUM_ACCOUNT_INFO_DOUBLE (SO_*/INITIAL/MAINTENANCE were transposed)
		{"ACCOUNT_MARGIN_SO_CALL", 7},
		{"ACCOUNT_MARGIN_SO_SO", 8},
		{"ACCOUNT_MARGIN_INITIAL", 9},
		{"ACCOUNT_MARGIN_MAINTENANCE", 10},
		// ENUM_ACCOUNT_INFO_STRING
		{"ACCOUNT_NAME", 0},
		{"ACCOUNT_SERVER", 1},
		{"ACCOUNT_CURRENCY", 2},
		{"ACCOUNT_COMPANY", 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, ok := interp.LookupMQLConstant(tc.name)
			if !ok {
				t.Fatalf("%s: LookupMQLConstant returned not-found — constant missing", tc.name)
			}
			if got := v.ToInt(); got != tc.want {
				t.Errorf("%s = %d, want %d (real MQL5 enum value)", tc.name, got, tc.want)
			}
		})
	}
}

// TestVM_API_TRUTH_1_AccountInfoStillImplemented verifies batch 2c did NOT
// reclassify the 3 AccountInfo* functions — they stay implemented (fake
// branches were fixed in place, not rejected).
//
// Adversarial: accidentally move them to unsupportedSymbols or remove from
// implementedAccount → IsAPIImplemented=false → RED.
func TestVM_API_TRUTH_1_AccountInfoStillImplemented(t *testing.T) {
	for _, api := range []string{"AccountInfoDouble", "AccountInfoInteger", "AccountInfoString"} {
		t.Run(api, func(t *testing.T) {
			if !interp.IsAPIImplemented(api) {
				t.Fatalf("%s: IsAPIImplemented=false, want true (batch 2c fixes branches in place, must not reclassify)", api)
			}
			if interp.IsAPIUnsupported(api) {
				t.Fatalf("%s: IsAPIUnsupported=true, want false (batch 2c fixes branches in place, must not reclassify)", api)
			}
		})
	}
}

// VM-API-TRUTH-1 batch 2d: timeseries stubs reclassified StatusUnsupported.
// CopyBuffer returned a fake success count without filling its by-reference
// array (and misread the array arg as count); CopyRates passed close prices
// off as MqlRates structs; iSpread/CopySpread returned fixed 0 (forex bars
// have real spread); CopyTicks returned fixed 0 (no tick source);
// BarsCalculated ignored its handle arg; SeriesInfoInteger returned fixed 0
// for every prop. The VM has no indicator-handle subsystem (Devin CLI ruling:
// not extended), so the compiler now rejects them (fail-closed).

// unsupportedTimeseriesNoSource lists the 7 timeseries API names that
// VM-API-TRUTH-1 batch 2d reclassified from implemented (fixed values /
// fake success counts / handle semantics) to StatusUnsupported.
var unsupportedTimeseriesNoSource = []string{
	"CopyBuffer", "CopyRates", "iSpread", "CopySpread",
	"CopyTicks", "BarsCalculated", "SeriesInfoInteger",
}

// TestVM_API_TRUTH_1_TimeseriesNoSourceRejected verifies each of the 7 APIs
// causes a compile-time error (not silent acceptance with fake runtime data).
//
// Adversarial: restore any of the 7 to implementedPlatform + remove from
// unsupportedSymbols → CompileMQL succeeds → RED.
func TestVM_API_TRUTH_1_TimeseriesNoSourceRejected(t *testing.T) {
	for _, api := range unsupportedTimeseriesNoSource {
		t.Run(api, func(t *testing.T) {
			src := "int OnInit() { return 0; }\nvoid OnTick() { " + api + "(); }"
			_, err := CompileMQL(src)
			if err == nil {
				t.Fatalf("%s: expected compile error (StatusUnsupported), got nil — API silently accepted", api)
			}
			msg := strings.ToLower(err.Error())
			if !strings.Contains(msg, "unsupported") && !strings.Contains(msg, strings.ToLower(api)) {
				t.Fatalf("%s: error message must mention 'unsupported' or API name, got: %v", api, err)
			}
		})
	}
}

// TestVM_API_TRUTH_1_TimeseriesNoSourceRegistryConsistency verifies the API
// registry reflects the batch 2d reclassification: each of the 7 APIs is
// StatusUnsupported with a non-empty reason, IsAPIImplemented=false,
// IsAPIUnsupported=true.
//
// Adversarial: remove the 7 entries from unsupportedSymbols → LookupAPI
// returns not-found (or StatusImplemented if also in implementedPlatform)
// → RED.
func TestVM_API_TRUTH_1_TimeseriesNoSourceRegistryConsistency(t *testing.T) {
	for _, api := range unsupportedTimeseriesNoSource {
		t.Run(api, func(t *testing.T) {
			sym, ok := interp.LookupAPI(api)
			if !ok {
				t.Fatalf("%s: LookupAPI returned not-found — missing from registry", api)
			}
			if sym.Status != interp.StatusUnsupported {
				t.Fatalf("%s: status = %v, want StatusUnsupported", api, sym.Status)
			}
			if sym.Reason == "" {
				t.Fatalf("%s: Reason is empty — must explain why unsupported", api)
			}
			if interp.IsAPIImplemented(api) {
				t.Fatalf("%s: IsAPIImplemented=true, want false", api)
			}
			if !interp.IsAPIUnsupported(api) {
				t.Fatalf("%s: IsAPIUnsupported=false, want true", api)
			}
		})
	}
}

// TestVM_API_TRUTH_1_TimeseriesRealStillImplemented verifies the batch 2d
// reclassification only affects the 7 sourceless/handle stubs — the real
// BarSeries/Volume channel functions stay implemented, including
// iRealVolume/CopyRealVolume (venue-fact semantics: backtest simulates
// non-exchange instruments where real volume is legitimately 0, like
// SymbolSelect=1/IsTesting=true).
//
// Adversarial: accidentally remove any of these from implementedPlatform or
// add to unsupportedSymbols → IsAPIImplemented=false → RED.
func TestVM_API_TRUTH_1_TimeseriesRealStillImplemented(t *testing.T) {
	realImplemented := []string{
		"iTickVolume", "iVolume",
		// venue-fact semantics (real volume is 0 for non-exchange instruments)
		"iRealVolume", "CopyRealVolume",
		"CopyClose", "CopyHigh", "CopyLow", "CopyOpen", "CopyTime", "CopyTickVolume",
		"Bars", "iBarShift", "iHighest", "iLowest",
	}
	for _, api := range realImplemented {
		t.Run(api, func(t *testing.T) {
			if !interp.IsAPIImplemented(api) {
				t.Fatalf("%s: IsAPIImplemented=false, want true (batch 2d must not误伤 real/venue-semantics implementations)", api)
			}
			if interp.IsAPIUnsupported(api) {
				t.Fatalf("%s: IsAPIUnsupported=true, want false (batch 2d must not误伤 real/venue-semantics implementations)", api)
			}
		})
	}
}

// VM-API-TRUTH-1 batch 2e: account noop whole-API disposition — 4 stubs
// (AccountName/AccountServer/AccountStopoutLevel/AccountFreeMarginMode) had no
// authoritative source and are reclassified StatusUnsupported; 4
// (AccountProfit/AccountCurrency/AccountCompany/AccountFreeMarginCheck) were
// re-wired from noop fixed values to real ctx sources.

// unsupportedAccountNoop lists the 4 account API names that VM-API-TRUTH-1
// batch 2e reclassified from implemented (noop fixed-value stub) to
// StatusUnsupported.
var unsupportedAccountNoop = []string{
	"AccountName", "AccountServer", "AccountStopoutLevel", "AccountFreeMarginMode",
}

// TestVM_API_TRUTH_1_AccountNoopRejected verifies each of the 4 APIs causes a
// compile-time error (not silent acceptance with fake runtime data).
//
// Adversarial: restore any of the 4 to implementedAccount + remove from
// unsupportedSymbols → CompileMQL succeeds → RED.
func TestVM_API_TRUTH_1_AccountNoopRejected(t *testing.T) {
	for _, api := range unsupportedAccountNoop {
		t.Run(api, func(t *testing.T) {
			src := "int OnInit() { return 0; }\nvoid OnTick() { " + api + "(); }"
			_, err := CompileMQL(src)
			if err == nil {
				t.Fatalf("%s: expected compile error (StatusUnsupported), got nil — API silently accepted", api)
			}
			msg := strings.ToLower(err.Error())
			if !strings.Contains(msg, "unsupported") && !strings.Contains(msg, strings.ToLower(api)) {
				t.Fatalf("%s: error message must mention 'unsupported' or API name, got: %v", api, err)
			}
		})
	}
}

// accountNoopTestContext extends accountStatusTestContext with the balance/
// equity/freeMargin/ask/bid/broker controls the batch 2e real-impl tests need.
type accountNoopTestContext struct {
	*accountStatusTestContext
	balance    decimal.Decimal
	equity     decimal.Decimal
	freeMargin decimal.Decimal
	ask        decimal.Decimal
	bid        decimal.Decimal
	broker     sdk.Broker
}

func (c *accountNoopTestContext) Account() sdk.AccountInfo {
	info := c.accountStatusTestContext.Account()
	info.Balance = c.balance
	info.Equity = c.equity
	info.FreeMargin = c.freeMargin
	return info
}
func (c *accountNoopTestContext) Ask() decimal.Decimal { return c.ask }
func (c *accountNoopTestContext) Bid() decimal.Decimal { return c.bid }
func (c *accountNoopTestContext) Broker() sdk.Broker   { return c.broker }

// accountNoopTestBroker serves a fixed ContractSize for the
// AccountFreeMarginCheck margin formula test. Only SymbolInfo is called.
type accountNoopTestBroker struct {
	sdk.Broker   // nil embed: non-SymbolInfo methods are never called here
	contractSize decimal.Decimal
}

func (b *accountNoopTestBroker) SymbolInfo(string) (sdk.SymbolInfo, error) {
	return sdk.SymbolInfo{ContractSize: b.contractSize}, nil
}

// TestVM_API_TRUTH_1_AccountNoopRealImpl verifies the 4 re-wired APIs read
// their authoritative sources instead of noop fixed values.
//
// Adversarial: re-wire any builtin back to builtinNoop* in
// vm_builtin_impls.go → the subtest reads the old fixed value (or OnInit
// unexpectedly succeeds on missing data) → RED.
func TestVM_API_TRUTH_1_AccountNoopRealImpl(t *testing.T) {
	t.Run("AccountProfit reads Equity-Balance", func(t *testing.T) {
		src := "double g=0; int OnInit(){ g=AccountProfit(); return 0; }"
		runner := compileAndInit(t, src, &accountNoopTestContext{
			accountStatusTestContext: &accountStatusTestContext{leverage: 100},
			balance:                  decimal.NewFromInt(9000),
			equity:                   decimal.NewFromInt(10000),
		})
		v, ok := runner.GetGlobal("g")
		if !ok {
			t.Fatal("global \"g\" not found")
		}
		want := decimal.NewFromInt(1000)
		if !v.ToDecimal().Equal(want) {
			t.Errorf("AccountProfit() = %s, want %s (Equity-Balance)", v.ToDecimal(), want)
		}
	})

	t.Run("AccountCurrency reads source", func(t *testing.T) {
		src := "string g=\"\"; int OnInit(){ g=AccountCurrency(); return 0; }"
		runner := compileAndInit(t, src, &accountNoopTestContext{
			accountStatusTestContext: &accountStatusTestContext{currency: "EUR"},
		})
		v, ok := runner.GetGlobal("g")
		if !ok {
			t.Fatal("global \"g\" not found")
		}
		if got := v.ToString(); got != "EUR" {
			t.Errorf("AccountCurrency() = %q, want %q (ctx currency)", got, "EUR")
		}
	})

	t.Run("AccountCompany reads source", func(t *testing.T) {
		src := "string g=\"\"; int OnInit(){ g=AccountCompany(); return 0; }"
		runner := compileAndInit(t, src, &accountNoopTestContext{
			accountStatusTestContext: &accountStatusTestContext{company: "TestCo"},
		})
		v, ok := runner.GetGlobal("g")
		if !ok {
			t.Fatal("global \"g\" not found")
		}
		if got := v.ToString(); got != "TestCo" {
			t.Errorf("AccountCompany() = %q, want %q (ctx company)", got, "TestCo")
		}
	})

	t.Run("AccountFreeMarginCheck formula", func(t *testing.T) {
		// required = volume·ContractSize·Ask/Leverage = 1·100000·1.25/100 = 1250
		// FreeMarginCheck = FreeMargin − required = 5000 − 1250 = 3750
		src := "double g=0; int OnInit(){ g=AccountFreeMarginCheck(\"EURUSD\", 0, 1); return 0; }"
		runner := compileAndInit(t, src, &accountNoopTestContext{
			accountStatusTestContext: &accountStatusTestContext{leverage: 100},
			freeMargin:               decimal.NewFromInt(5000),
			ask:                      decimal.NewFromFloat(1.25),
			broker:                   &accountNoopTestBroker{contractSize: decimal.NewFromInt(100000)},
		})
		v, ok := runner.GetGlobal("g")
		if !ok {
			t.Fatal("global \"g\" not found")
		}
		want := decimal.NewFromInt(3750)
		if !v.ToDecimal().Equal(want) {
			t.Errorf("AccountFreeMarginCheck = %s, want %s (FreeMargin − volume·ContractSize·Ask/Leverage)", v.ToDecimal(), want)
		}
	})

	// Missing-fact and invalid-arg cases must fail-closed, not return fake 0.
	errorCases := []struct {
		name string
		src  string
		ctx  *accountNoopTestContext
	}{
		{"AccountCurrency empty errors", "int OnInit(){ AccountCurrency(); return 0; }",
			&accountNoopTestContext{accountStatusTestContext: &accountStatusTestContext{}}},
		{"AccountCompany empty errors", "int OnInit(){ AccountCompany(); return 0; }",
			&accountNoopTestContext{accountStatusTestContext: &accountStatusTestContext{}}},
		{"AccountFreeMarginCheck invalid cmd errors", "int OnInit(){ AccountFreeMarginCheck(\"EURUSD\", 9, 1); return 0; }",
			&accountNoopTestContext{
				accountStatusTestContext: &accountStatusTestContext{leverage: 100},
				broker:                   &accountNoopTestBroker{contractSize: decimal.NewFromInt(100000)},
			}},
		{"AccountFreeMarginCheck nil broker errors", "int OnInit(){ AccountFreeMarginCheck(\"EURUSD\", 0, 1); return 0; }",
			&accountNoopTestContext{accountStatusTestContext: &accountStatusTestContext{leverage: 100}}},
	}
	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			runner, err := CompileMQL(tc.src)
			if err != nil {
				t.Fatalf("CompileMQL failed: %v", err)
			}
			runner.SetSignalMode(true)
			if err := runner.OnInit(tc.ctx); err == nil {
				t.Fatalf("%s: OnInit err = nil, want error (must fail-closed, not return fake value)", tc.name)
			}
		})
	}
}

// TestVM_API_TRUTH_1_AccountNoopConsistency verifies the registry reflects the
// batch 2e disposition: the 4 no-source stubs are StatusUnsupported, and the
// 4 re-wired APIs plus the pre-existing real implementations stay implemented.
//
// Adversarial: remove the 4 entries from unsupportedSymbols → LookupAPI
// returns not-found → RED; accidentally reclassify any real/rewired API →
// IsAPIImplemented=false → RED.
func TestVM_API_TRUTH_1_AccountNoopConsistency(t *testing.T) {
	for _, api := range unsupportedAccountNoop {
		t.Run(api+"/unsupported", func(t *testing.T) {
			sym, ok := interp.LookupAPI(api)
			if !ok {
				t.Fatalf("%s: LookupAPI returned not-found — missing from registry", api)
			}
			if sym.Status != interp.StatusUnsupported {
				t.Fatalf("%s: status = %v, want StatusUnsupported", api, sym.Status)
			}
			if sym.Reason == "" {
				t.Fatalf("%s: Reason is empty — must explain why unsupported", api)
			}
			if interp.IsAPIImplemented(api) {
				t.Fatalf("%s: IsAPIImplemented=true, want false", api)
			}
			if !interp.IsAPIUnsupported(api) {
				t.Fatalf("%s: IsAPIUnsupported=false, want true", api)
			}
		})
	}
	realOrRewired := []string{
		// re-wired to real sources in batch 2e
		"AccountProfit", "AccountCurrency", "AccountCompany", "AccountFreeMarginCheck",
		// pre-existing real implementations (must not be误伤)
		"AccountBalance", "AccountLeverage", "AccountNumber",
	}
	for _, api := range realOrRewired {
		t.Run(api+"/implemented", func(t *testing.T) {
			if !interp.IsAPIImplemented(api) {
				t.Fatalf("%s: IsAPIImplemented=false, want true (batch 2e must not误伤 real/rewired implementations)", api)
			}
			if interp.IsAPIUnsupported(api) {
				t.Fatalf("%s: IsAPIUnsupported=true, want false (batch 2e must not误伤 real/rewired implementations)", api)
			}
		})
	}
}

// VM-ENUM-NUMBERING-1: SymbolInfoDouble/SymbolInfoInteger/MarketInfo prop
// numbering aligned to the real MQL5 ENUM_SYMBOL_INFO_DOUBLE/INTEGER and the
// real MQL4 MarketInfo modes; sourceless branches fail-closed;
// SymbolInfoString reclassified StatusUnsupported (the real STRING enum has
// no NAME member and sdk.SymbolInfo carries no string fields).

// symbolEnumTestBroker serves a fully-populated sdk.SymbolInfo whose values
// are all distinct, so a mis-numbered prop cannot hit a neighbor's value.
type symbolEnumTestBroker struct{ sdk.Broker }

func (b *symbolEnumTestBroker) SymbolInfo(string) (sdk.SymbolInfo, error) {
	return sdk.SymbolInfo{
		Name:         "EURUSD",
		Digits:       5,
		Point:        decimal.NewFromFloat(0.0001),
		VolumeMin:    decimal.NewFromInt(1),
		VolumeMax:    decimal.NewFromInt(100),
		VolumeStep:   decimal.NewFromFloat(0.01),
		StopsLevel:   30,
		Spread:       12,
		TickValue:    decimal.NewFromFloat(1.1),
		TickSize:     decimal.NewFromFloat(0.00001),
		SwapLong:     decimal.NewFromFloat(0.5),
		SwapShort:    decimal.NewFromFloat(-1.3),
		ContractSize: decimal.NewFromInt(100000),
	}, nil
}

// symbolEnumTestContext adds quote/time/broker controls to the account
// context (Symbol() = "EURUSD" comes from the embedded context).
type symbolEnumTestContext struct {
	*accountStatusTestContext
	ask        decimal.Decimal
	bid        decimal.Decimal
	serverTime int64
	broker     sdk.Broker
}

func (c *symbolEnumTestContext) Ask() decimal.Decimal { return c.ask }
func (c *symbolEnumTestContext) Bid() decimal.Decimal { return c.bid }
func (c *symbolEnumTestContext) ServerTime() int64    { return c.serverTime }
func (c *symbolEnumTestContext) Broker() sdk.Broker   { return c.broker }

func newSymbolEnumVM(t *testing.T) *VM {
	t.Helper()
	vm := NewVM(&Bytecode{OnBar: -1, Builtins: make(map[string]BuiltinID)})
	vm.ctx = &symbolEnumTestContext{
		accountStatusTestContext: &accountStatusTestContext{leverage: 100, isTradeAllowed: true},
		ask:                      decimal.NewFromFloat(1.25),
		bid:                      decimal.NewFromFloat(1.2),
		serverTime:               1000000, // fits int32 so both TIME (s) and TIME_MSC (ms) are assertable
		broker:                   &symbolEnumTestBroker{},
	}
	return vm
}

// enumConst resolves a named MQL constant — the S1↔S2 consistency check:
// calling builtins with the *named* constant must hit the matching real branch.
func enumConst(t *testing.T, name string) int32 {
	t.Helper()
	v, ok := interp.LookupMQLConstant(name)
	if !ok {
		t.Fatalf("MQL constant %s missing from the registry", name)
	}
	return v.ToInt()
}

// TestVM_ENUM_NUMBERING_1_SymbolInfoStringRejected verifies the
// SymbolInfoString reclassification: compile-time rejection + registry
// consistency.
//
// Adversarial: comment the SymbolInfoString entry in unsupportedSymbols →
// compile succeeds / LookupAPI not-found → RED (×2).
func TestVM_ENUM_NUMBERING_1_SymbolInfoStringRejected(t *testing.T) {
	src := "int OnInit() { return 0; }\nvoid OnTick() { SymbolInfoString(\"EURUSD\", 1); }"
	_, err := CompileMQL(src)
	if err == nil {
		t.Fatal("SymbolInfoString: expected compile error (StatusUnsupported), got nil — API silently accepted")
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "unsupported") && !strings.Contains(msg, "symbolinfostring") {
		t.Fatalf("error must mention 'unsupported' or the API name, got: %v", err)
	}

	sym, ok := interp.LookupAPI("SymbolInfoString")
	if !ok {
		t.Fatal("SymbolInfoString: LookupAPI returned not-found — missing from registry")
	}
	if sym.Status != interp.StatusUnsupported {
		t.Fatalf("status = %v, want StatusUnsupported", sym.Status)
	}
	if sym.Reason == "" {
		t.Fatal("Reason is empty — must explain why unsupported")
	}
	if interp.IsAPIImplemented("SymbolInfoString") {
		t.Fatal("IsAPIImplemented=true, want false")
	}
	if !interp.IsAPIUnsupported("SymbolInfoString") {
		t.Fatal("IsAPIUnsupported=false, want true")
	}
	if !interp.IsAPIImplemented("SymbolInfoDouble") || !interp.IsAPIImplemented("SymbolInfoInteger") || !interp.IsAPIImplemented("MarketInfo") {
		t.Fatal("SymbolInfoDouble/SymbolInfoInteger/MarketInfo must stay implemented (not误伤)")
	}
}

// TestVM_ENUM_NUMBERING_1_RealBranchesReadSource is the mis-numbering matrix:
// every real branch, called with the *named* constant, must return the
// injected source value. Key cases that used to hit the wrong branch:
// SymbolInfoDouble(SYMBOL_POINT) returned VolumeMax, SYMBOL_BID returned Point.
//
// Adversarial: revert any constant value (SYMBOL_POINT → 2) or any switch
// case number → the builtin reads the wrong source → RED.
func TestVM_ENUM_NUMBERING_1_RealBranchesReadSource(t *testing.T) {
	vm := newSymbolEnumVM(t)

	doubleCases := []struct {
		constName string
		want      decimal.Decimal
	}{
		{"SYMBOL_BID", decimal.NewFromFloat(1.2)},
		{"SYMBOL_ASK", decimal.NewFromFloat(1.25)},
		{"SYMBOL_POINT", decimal.NewFromFloat(0.0001)},
		{"SYMBOL_TRADE_TICK_VALUE", decimal.NewFromFloat(1.1)},
		{"SYMBOL_TRADE_TICK_SIZE", decimal.NewFromFloat(0.00001)},
		{"SYMBOL_TRADE_CONTRACT_SIZE", decimal.NewFromInt(100000)},
		{"SYMBOL_VOLUME_MIN", decimal.NewFromInt(1)},
		{"SYMBOL_VOLUME_MAX", decimal.NewFromInt(100)},
		{"SYMBOL_VOLUME_STEP", decimal.NewFromFloat(0.01)},
		{"SYMBOL_SWAP_LONG", decimal.NewFromFloat(0.5)},
		{"SYMBOL_SWAP_SHORT", decimal.NewFromFloat(-1.3)},
	}
	for _, tc := range doubleCases {
		t.Run("Double/"+tc.constName, func(t *testing.T) {
			v, err := builtinSymbolInfoDouble(vm, []interp.Value{
				interp.StringVal("EURUSD"), interp.IntVal(enumConst(t, tc.constName)),
			})
			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if !v.ToDecimal().Equal(tc.want) {
				t.Fatalf("got %s, want %s (prop %s=%d must read its own source)",
					v.ToDecimal(), tc.want, tc.constName, enumConst(t, tc.constName))
			}
		})
	}

	integerCases := []struct {
		constName string
		want      int32
	}{
		{"SYMBOL_DIGITS", 5},
		{"SYMBOL_SPREAD", 12},
		{"SYMBOL_TRADE_STOPS_LEVEL", 30},
		{"SYMBOL_TIME", 1000},        // ServerTime(1e6 ms)/1000
		{"SYMBOL_TIME_MSC", 1000000}, // native ms
	}
	for _, tc := range integerCases {
		t.Run("Integer/"+tc.constName, func(t *testing.T) {
			v, err := builtinSymbolInfoInteger(vm, []interp.Value{
				interp.StringVal("EURUSD"), interp.IntVal(enumConst(t, tc.constName)),
			})
			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if v.ToInt() != tc.want {
				t.Fatalf("got %d, want %d (prop %s=%d must read its own source)",
					v.ToInt(), tc.want, tc.constName, enumConst(t, tc.constName))
			}
		})
	}

	marketCases := []struct {
		constName string
		want      decimal.Decimal
	}{
		{"MODE_BID", decimal.NewFromFloat(1.2)},
		{"MODE_ASK", decimal.NewFromFloat(1.25)},
		{"MODE_POINT", decimal.NewFromFloat(0.0001)},
		{"MODE_DIGITS", decimal.NewFromInt(5)},
		{"MODE_SPREAD", decimal.NewFromInt(12)},
		{"MODE_STOPLEVEL", decimal.NewFromInt(30)},
		{"MODE_LOTSIZE", decimal.NewFromInt(100000)},
		{"MODE_TICKVALUE", decimal.NewFromFloat(1.1)},    // renumbered 17→16
		{"MODE_TICKSIZE", decimal.NewFromFloat(0.00001)}, // renumbered 18→17
		{"MODE_SWAPLONG", decimal.NewFromFloat(0.5)},     // new case 18
		{"MODE_SWAPSHORT", decimal.NewFromFloat(-1.3)},   // new case 19
		{"MODE_MINLOT", decimal.NewFromInt(1)},           // renumbered 20→23
		{"MODE_LOTSTEP", decimal.NewFromFloat(0.01)},     // renumbered 22→24
		{"MODE_MAXLOT", decimal.NewFromInt(100)},         // renumbered 21→25
		{"MODE_TIME", decimal.NewFromInt(1000)},
		{"MODE_TRADEALLOWED", decimal.NewFromInt(1)},
		// MARGININIT/MARGINREQUIRED: ContractSize·Ask/Leverage = 100000·1.25/100
		{"MODE_MARGININIT", decimal.NewFromInt(1250)},
		{"MODE_MARGINREQUIRED", decimal.NewFromInt(1250)},
	}
	for _, tc := range marketCases {
		t.Run("MarketInfo/"+tc.constName, func(t *testing.T) {
			v, err := builtinMarketInfo(vm, []interp.Value{
				interp.StringVal("EURUSD"), interp.IntVal(enumConst(t, tc.constName)),
			})
			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if !v.ToDecimal().Equal(tc.want) {
				t.Fatalf("got %s, want %s (mode %s=%d must read its own source)",
					v.ToDecimal(), tc.want, tc.constName, enumConst(t, tc.constName))
			}
		})
	}
}

// TestVM_ENUM_NUMBERING_1_VenueValues pins the venue-fact branches (same
// semantics family as iRealVolume: fixed values that are true facts of the
// backtest venue, not missing data).
//
// Adversarial: change any venue branch value (e.g. SYMBOL_EXIST returns 0) →
// RED.
func TestVM_ENUM_NUMBERING_1_VenueValues(t *testing.T) {
	vm := newSymbolEnumVM(t)

	doubleVenue := []struct {
		name string
		prop int32
	}{
		{"SYMBOL_LAST", 6}, {"SYMBOL_LASTHIGH", 7}, {"SYMBOL_LASTLOW", 8},
		{"SYMBOL_VOLUME_REAL", 9}, {"SYMBOL_VOLUMEHIGH_REAL", 10}, {"SYMBOL_VOLUMELOW_REAL", 11},
	}
	for _, tc := range doubleVenue {
		t.Run("Double/"+tc.name, func(t *testing.T) {
			v, err := builtinSymbolInfoDouble(vm, []interp.Value{interp.StringVal("EURUSD"), interp.IntVal(tc.prop)})
			if err != nil {
				t.Fatalf("err = %v, want nil (venue 0 is a fact, not missing data)", err)
			}
			if !v.ToDecimal().IsZero() {
				t.Fatalf("got %s, want 0 (venue fact)", v.ToDecimal())
			}
		})
	}

	integerVenue := []struct {
		name string
		prop int32
		want int32
	}{
		{"SYMBOL_CUSTOM", 3, 0},
		{"SYMBOL_CHART_MODE", 5, 0},
		{"SYMBOL_EXIST", 6, 1},
		{"SYMBOL_SELECT", 7, 1},
		{"SYMBOL_VISIBLE", 8, 1},
		{"SYMBOL_SPREAD_FLOAT", 18, 0},
		{"SYMBOL_TICKS_BOOKDEPTH", 20, 0},
		{"SYMBOL_TRADE_CALC_MODE", 21, 0},
		{"SYMBOL_TRADE_MODE", 22, 4},
		{"SYMBOL_START_TIME", 23, 0},
		{"SYMBOL_EXPIRATION_TIME", 24, 0},
		{"SYMBOL_TRADE_FREEZE_LEVEL", 26, 0},
		{"SYMBOL_TRADE_EXEMODE", 27, 2},
		{"SYMBOL_SWAP_MODE", 28, 1},
		{"SYMBOL_ORDER_MODE", 33, 63},
		{"SYMBOL_ORDER_GTC_MODE", 34, 0},
	}
	for _, tc := range integerVenue {
		t.Run("Integer/"+tc.name, func(t *testing.T) {
			v, err := builtinSymbolInfoInteger(vm, []interp.Value{interp.StringVal("EURUSD"), interp.IntVal(tc.prop)})
			if err != nil {
				t.Fatalf("err = %v, want nil (venue value is a fact, not missing data)", err)
			}
			if v.ToInt() != tc.want {
				t.Fatalf("got %d, want %d (venue fact)", v.ToInt(), tc.want)
			}
		})
	}

	marketVenue := []struct {
		name string
		mode int32
	}{
		{"MODE_STARTING", 20}, {"MODE_EXPIRATION", 21},
		{"MODE_FREEZELEVEL", 32}, {"MODE_CLOSEBY_ALLOWED", 33},
	}
	for _, tc := range marketVenue {
		t.Run("MarketInfo/"+tc.name, func(t *testing.T) {
			v, err := builtinMarketInfo(vm, []interp.Value{interp.StringVal("EURUSD"), interp.IntVal(tc.mode)})
			if err != nil {
				t.Fatalf("err = %v, want nil (venue 0 is a fact, not missing data)", err)
			}
			if !v.ToDecimal().IsZero() {
				t.Fatalf("got %s, want 0 (venue fact)", v.ToDecimal())
			}
		})
	}

	// TRADEALLOWED mirrors ctx.Account().IsTradeAllowed both ways.
	vmFalse := newSymbolEnumVM(t)
	vmFalse.ctx.(*symbolEnumTestContext).isTradeAllowed = false
	v, err := builtinMarketInfo(vmFalse, []interp.Value{interp.StringVal("EURUSD"), interp.IntVal(22)})
	if err != nil || !v.ToDecimal().IsZero() {
		t.Fatalf("MODE_TRADEALLOWED with IsTradeAllowed=false: got %s err=%v, want 0/nil", v.ToDecimal(), err)
	}
}

// TestVM_ENUM_NUMBERING_1_FailClosed verifies sourceless props error instead
// of returning fake values, and that constants for non-real modes/props are
// gone (compile-time rejection).
//
// Adversarial: restore any fake 0-return for these props → OnInit-style
// direct calls succeed → RED.
func TestVM_ENUM_NUMBERING_1_FailClosed(t *testing.T) {
	vm := newSymbolEnumVM(t)

	if _, err := builtinSymbolInfoDouble(vm, []interp.Value{interp.StringVal("EURUSD"), interp.IntVal(48)}); err == nil {
		t.Fatal("SymbolInfoDouble(prop 48 MARGIN_HEDGED): err = nil, want error (no hedged-margin model)")
	}
	if _, err := builtinSymbolInfoInteger(vm, []interp.Value{interp.StringVal("EURUSD"), interp.IntVal(1)}); err == nil {
		t.Fatal("SymbolInfoInteger(prop 1 SECTOR): err = nil, want error (no source)")
	}
	if _, err := builtinMarketInfo(vm, []interp.Value{interp.StringVal("EURUSD"), interp.IntVal(1)}); err == nil {
		t.Fatal("MarketInfo(mode 1 MODE_LOW): err = nil, want error (daily aggregate, no source)")
	}

	for _, name := range []string{"SYMBOL_SWAP_ROLLOVER3DAYS", "MODE_SWAPTYPE", "MODE_PROFITCALCMODE", "SYMBOL_MARGIN_INITIAL"} {
		t.Run("const/"+name, func(t *testing.T) {
			// Registry level: the removed constant no longer resolves.
			if _, ok := interp.LookupMQLConstant(name); ok {
				t.Fatalf("%s still resolvable — removed constant must not resolve", name)
			}
			// NOTE: an unknown identifier in source does NOT fail compilation —
			// the front end auto-registers it as an implicit global (value 0).
			// For props where 0 is an unsupported prop the call then fails
			// closed at runtime; SYMBOL_MARGIN_INITIAL/MARGIN_MAINTENANCE are
			// exempt because Double prop 0 is the real BID branch. The implicit
			//-variable front-end behavior is tracked with
			// VM-GLOBAL-ARRAY-DECL-1's front-end family.
			runtimeErrorCases := map[string]string{
				"SYMBOL_SWAP_ROLLOVER3DAYS": "SymbolInfoInteger",
				"MODE_SWAPTYPE":             "MarketInfo",
				"MODE_PROFITCALCMODE":       "MarketInfo",
			}
			fn, ok := runtimeErrorCases[name]
			if !ok {
				return
			}
			src := "int OnInit(){ " + fn + "(\"EURUSD\", " + name + "); return 0; }"
			runner, err := CompileMQL(src)
			if err != nil {
				t.Fatalf("CompileMQL failed: %v", err)
			}
			runner.SetSignalMode(true)
			if err := runner.OnInit(&symbolEnumTestContext{
				accountStatusTestContext: &accountStatusTestContext{leverage: 100, isTradeAllowed: true},
				ask:                      decimal.NewFromFloat(1.25),
				bid:                      decimal.NewFromFloat(1.2),
				serverTime:               1000000,
				broker:                   &symbolEnumTestBroker{},
			}); err == nil {
				t.Fatalf("%s references a removed constant whose value 0 is an unsupported prop: OnInit err = nil, want error (fail-closed)", name)
			}
		})
	}
}

// TestVM_ENUM_NUMBERING_1_CurrentSymbolGuards verifies the quote/time guards:
// BID/ASK/TIME only resolve for the run's own symbol with a non-zero source.
//
// Adversarial: remove a guard → the call silently returns a (possibly wrong)
// value → RED.
func TestVM_ENUM_NUMBERING_1_CurrentSymbolGuards(t *testing.T) {
	vm := newSymbolEnumVM(t)

	if _, err := builtinSymbolInfoDouble(vm, []interp.Value{interp.StringVal("OTHER"), interp.IntVal(enumConst(t, "SYMBOL_BID"))}); err == nil {
		t.Fatal("SymbolInfoDouble(\"OTHER\", SYMBOL_BID): err = nil, want error (no quote source for another symbol)")
	}
	if _, err := builtinMarketInfo(vm, []interp.Value{interp.StringVal("OTHER"), interp.IntVal(enumConst(t, "MODE_ASK"))}); err == nil {
		t.Fatal("MarketInfo(\"OTHER\", MODE_ASK): err = nil, want error")
	}

	vmNoTime := newSymbolEnumVM(t)
	vmNoTime.ctx.(*symbolEnumTestContext).serverTime = 0
	if _, err := builtinSymbolInfoInteger(vmNoTime, []interp.Value{interp.StringVal("EURUSD"), interp.IntVal(enumConst(t, "SYMBOL_TIME"))}); err == nil {
		t.Fatal("SymbolInfoInteger(SYMBOL_TIME) with ServerTime=0: err = nil, want error")
	}
	if _, err := builtinSymbolInfoInteger(vmNoTime, []interp.Value{interp.StringVal("EURUSD"), interp.IntVal(enumConst(t, "SYMBOL_TIME_MSC"))}); err == nil {
		t.Fatal("SymbolInfoInteger(SYMBOL_TIME_MSC) with ServerTime=0: err = nil, want error")
	}

	vmNoBid := newSymbolEnumVM(t)
	vmNoBid.ctx.(*symbolEnumTestContext).bid = decimal.Zero
	if _, err := builtinSymbolInfoDouble(vmNoBid, []interp.Value{interp.StringVal("EURUSD"), interp.IntVal(enumConst(t, "SYMBOL_BID"))}); err == nil {
		t.Fatal("SymbolInfoDouble(SYMBOL_BID) with Bid=0: err = nil, want error (zero quote is not a fact)")
	}
}
