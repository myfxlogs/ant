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
