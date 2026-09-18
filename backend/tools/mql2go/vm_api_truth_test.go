package mql2go

import (
	"strings"
	"testing"

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
