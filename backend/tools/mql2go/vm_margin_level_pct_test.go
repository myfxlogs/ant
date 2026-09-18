// vm_margin_level_pct_test.go — ACCOUNT-MARGIN-LEVEL-PCT-1 (2026-09-18).
//
// AccountInfoDouble(ACCOUNT_MARGIN_LEVEL) returned equity/margin as a RATIO;
// real MQL5 documents it as a PERCENTAGE (equity/margin*100 — official
// example 9921.24/1000 → 992.12). Any margin-call/stop-out threshold check
// was off by 100×. These pins lock the corrected percentage semantics and
// the zero-margin boundary (real MT5 reports 0 with no positions).
//
// Adversarial (M1): remove the .Mul(100) → T1 REDs (9.92124 instead of
// 992.124); T2 is boundary-independent and stays green.
package mql2go

import (
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/tools/mql2go/interp"
)

// TestMarginLevel_IsPercentage — T1: equity 9921.24 / margin 1000 →
// ACCOUNT_MARGIN_LEVEL == 992.12 (percentage), NOT 9.92124 (ratio).
func TestMarginLevel_IsPercentage(t *testing.T) {
	bc := &Bytecode{OnBar: -1, Builtins: make(map[string]BuiltinID)}
	vm := NewVM(bc)
	vm.ctx = &accountNoopTestContext{
		accountStatusTestContext: &accountStatusTestContext{},
		equity:                   decimal.NewFromFloat(9921.24),
		margin:                   decimal.NewFromInt(1000),
	}

	v, err := builtinAccountInfoDouble(vm, []interp.Value{
		interp.IntVal(6), // ACCOUNT_MARGIN_LEVEL (real MQL5 enum value)
	})
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	// 9921.24/1000 = 9.92124 ratio × 100 = 992.124 exactly (the dispatch's
	// "992.12" is the official doc's rounded display).
	want := decimal.NewFromFloat(992.124)
	if !v.ToDecimal().Equal(want) {
		t.Fatalf("ACCOUNT_MARGIN_LEVEL = %s, want %s (MQL5 percentage semantics: equity/margin*100 — a ratio of 9.92124 means the ×100 fix is missing)", v.ToDecimal(), want)
	}
}

// TestMarginLevel_ZeroMarginStaysZero — T2: Margin=0 → 0 (real MT5
// convention with no positions); guards against regressions to error/inf.
func TestMarginLevel_ZeroMarginStaysZero(t *testing.T) {
	bc := &Bytecode{OnBar: -1, Builtins: make(map[string]BuiltinID)}
	vm := NewVM(bc)
	vm.ctx = &accountNoopTestContext{
		accountStatusTestContext: &accountStatusTestContext{},
		equity:                   decimal.NewFromFloat(9921.24),
		margin:                   decimal.Zero,
	}

	v, err := builtinAccountInfoDouble(vm, []interp.Value{interp.IntVal(6)})
	if err != nil {
		t.Fatalf("err = %v, want nil (zero margin is a documented 0, not an error)", err)
	}
	if !v.ToDecimal().IsZero() {
		t.Fatalf("ACCOUNT_MARGIN_LEVEL with Margin=0 = %s, want 0 (documented boundary preserved)", v.ToDecimal())
	}
}
