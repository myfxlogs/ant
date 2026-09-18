package mql2go

import (
	"fmt"
	"strings"
	"testing"

	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
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
