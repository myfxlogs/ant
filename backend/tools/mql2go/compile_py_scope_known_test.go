// Package pin tests for documented Python-subset scope deviations
// (registry PY-SCOPE-KNOWN-1). These tests PIN the current behavior as
// documented — they are NOT approvals of the semantics. If a future fix
// corrects a deviation, the corresponding pin test must be updated in the
// same commit that changes the behavior.
// ③ for-range exit value is already pinned by TestQS13_ForLoopVarSurvivesLoop.
package mql2go

import (
	"context"
	"strings"
	"testing"

	"alphaforge/tools/mql2go/interp"
)

// runScopeKnownSource compiles python source, runs OnBar, and returns the
// runner (for GetGlobal / Coverage assertions).
func runScopeKnownSource(t *testing.T, body string) *VMRunner {
	t.Helper()
	source := `class S:
    def on_bar(self) -> None:
` + body + `        return
`
	vmRunner, err := CompilePython(source)
	if err != nil {
		t.Fatalf("CompilePython failed: %v", err)
	}
	if err := vmRunner.vm.RunOnBar(context.Background()); err != nil {
		t.Fatalf("RunOnBar failed: %v", err)
	}
	return vmRunner
}

// TestPyScopeKnown_SelfBareWriteClobbersField — pin deviation ①a (write side):
// a bare `x = 1` inside a function hits the same global slot declared for
// `self.x`, overwriting the field (Python semantics: self.x stays 10, the
// bare x is function-local).
//
// Discriminator: GetGlobal("x") == 1. (GetGlobal("r") == 1 is documented but
// NOT discriminating — with a local write present, the self.x read also
// resolves to the local via resolveVar's local-first rule.)
//
// Adversarial (M1): comment out resolveAssignTarget's isDeclaredGlobal block
// → bare x becomes a local → GetGlobal("x") = 10 → RED.
func TestPyScopeKnown_SelfBareWriteClobbersField(t *testing.T) {
	vmRunner := runScopeKnownSource(t, `        self.x: int = 10
        x: int = 1
        self.r = self.x
`)
	if got := getGlobalInt(t, vmRunner, "x"); got != 1 {
		t.Fatalf("GetGlobal(x) = %d, want 1 (documented deviation: bare write clobbers the field slot; Python self.x should stay 10)", got)
	}
	// Documented (non-discriminating): r reads back the clobbered value.
	if got := getGlobalInt(t, vmRunner, "r"); got != 1 {
		t.Fatalf("GetGlobal(r) = %d, want 1 (documented deviation)", got)
	}
}

// TestPyScopeKnown_SelfReadShadowedByLocal — pin deviation ①b (read side):
// a same-named local (the for-loop variable) shadows the self.x field READ,
// while the field value itself stays intact in its slot (shadowing, not
// clobbering).
//
// Adversarial (M1b): restore compileDecl's python base-scope assignment to
// the inner scope → the for variable dies with the loop scope → r = 10 → RED
// (expected collateral: QS-1.3 v3 family tests RED too — that is the v3 fix
// surface).
func TestPyScopeKnown_SelfReadShadowedByLocal(t *testing.T) {
	vmRunner := runScopeKnownSource(t, `        self.x: int = 10
        for x in range(3):
            pass
        self.r = self.x
`)
	if got := getGlobalInt(t, vmRunner, "r"); got != 3 {
		t.Fatalf("GetGlobal(r) = %d, want 3 (documented deviation: self.x read shadowed by the loop local; Python field read should be 10)", got)
	}
	// Shadowing, not clobbering: the field slot keeps 10.
	if got := getGlobalInt(t, vmRunner, "x"); got != 10 {
		t.Fatalf("GetGlobal(x) = %d, want 10 (shadowing must not overwrite the field slot)", got)
	}
}

// TestPyScopeKnown_ImplicitReadYieldsNone — pin deviation ②: reading an
// undeclared variable compiles and runs (no NameError), yields ValNone, and
// is reported through the coverage blind-spot channel (not fully silent).
//
// Adversarial (M2): turn resolveVar's python implicit-registration branch
// into a compile error → CompilePython fails → RED.
func TestPyScopeKnown_ImplicitReadYieldsNone(t *testing.T) {
	vmRunner := runScopeKnownSource(t, `        self.r = never_declared
`)
	v, ok := vmRunner.GetGlobal("r")
	if !ok {
		t.Fatal("global r not found")
	}
	if v.Kind != interp.ValNone {
		t.Fatalf("r kind = %v, want ValNone (documented deviation: undeclared reads yield None, not NameError)", v.Kind)
	}
	found := false
	for _, bs := range vmRunner.vm.bc.Coverage.BlindSpots {
		if strings.Contains(bs, "implicit variable: never_declared") {
			found = true
		}
	}
	if !found {
		t.Fatalf("Coverage.BlindSpots = %v, want it to contain 'implicit variable: never_declared' (deviation must surface via coverage)", vmRunner.vm.bc.Coverage.BlindSpots)
	}
}
