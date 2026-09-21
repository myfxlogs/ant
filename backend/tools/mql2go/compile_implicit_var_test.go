package mql2go

import (
	"strings"
	"testing"
)

// VM-IMPLICIT-VAR-READ-1: an undeclared identifier in a READ position must be
// a compile error for MQL4/MQL5. Previously resolveVar silently registered a
// zero-valued global, so e.g. SymbolInfoDouble(Symbol(), SYMBOL_TICK_VALUE)
// (a nonexistent constant alias) compiled and evaluated as property 0
// (SYMBOL_BID), returning the bid price instead of a tick value.
//
// Implicit declaration is preserved only at WRITE positions (`x = expr`),
// matching the documented MQL4 compatibility shim.
//
// Adversarial (M1): restoring the mql4 implicit branch in resolveVar must
// turn the read-position tests RED.
func TestCompile_ImplicitVarRead_MQL4_Rejected(t *testing.T) {
	cases := map[string]string{
		"bare read": `int OnInit() {
	int x = undeclared_name_xyz;
	return 0;
}`,
		"call arg (probe SYMBOL_TICK_VALUE alias)": `void OnTick() {
	double v = SymbolInfoDouble(Symbol(), SYMBOL_TICK_VALUE);
}`,
		"update ++": `void OnTick() {
	undeclared_ctr++;
}`,
		"compound +=": `void OnTick() {
	undeclared_acc += 1.5;
}`,
		"array read": `void OnTick() {
	double v = undeclared_arr[0];
}`,
	}
	for name, src := range cases {
		_, err := CompileMQL(src)
		if err == nil {
			t.Errorf("%s: undeclared read compiled silently — must be a compile error", name)
			continue
		}
		if !strings.Contains(err.Error(), "undeclared") && !strings.Contains(err.Error(), "SYMBOL_TICK_VALUE") {
			t.Errorf("%s: error %q does not name the undeclared identifier", name, err)
		}
	}
}

// Write positions keep the implicit-declaration shim: `x = expr` on an
// undeclared name still declares a global (existing compatibility behavior,
// exercised by live strategies that assign without a declaration).
func TestCompile_ImplicitVarWrite_MQL4_StillAllowed(t *testing.T) {
	cases := map[string]string{
		"plain assign": `void OnTick() {
	implicit_total = 1;
}`,
		"assign from call": `void OnTick() {
	implicit_ticket = OrderSend(Symbol(), OP_BUY, 0.01, Ask, 3, 0, 0);
}`,
		"array store": `void OnTick() {
	implicit_buf[0] = 1.5;
}`,
	}
	for name, src := range cases {
		if _, err := CompileMQL(src); err != nil {
			t.Errorf("%s: implicit-declaration write must still compile, got %v", name, err)
		}
	}
}
