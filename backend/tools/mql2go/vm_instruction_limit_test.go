package mql2go

import (
	"context"
	"strings"
	"testing"
)

// TestVM_InstructionLimit_DiagnosesInfiniteLoop ensures that when a strategy
// exceeds the per-event MaxTicks budget (an infinite loop), the VM returns an
// error that names the offending function and explains the cause — instead of
// the old opaque "strategy exceeded instruction limit".
//
// Regression guard for the instruction-limit diagnostic.
// See docs/runbook/mql2go-known-pitfalls.md and the MQL EA Compatibility Proposal.
func TestVM_InstructionLimit_DiagnosesInfiniteLoop(t *testing.T) {
	// The infinite loop lives in a named user function so the diagnostic can
	// point the EA author at it. (A loop directly in OnTick would report the
	// event handler instead — covered conceptually by the same code path.)
	source := `
int OnInit() { return 0; }

void InfiniteLoop()
{
    while (true)
    {
    }
}

void OnTick()
{
    InfiniteLoop();
}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}
	bc, err := CompileAST(ir)
	if err != nil {
		t.Fatalf("CompileAST failed: %v", err)
	}

	vm := NewVM(bc)
	err = vm.RunOnTick(context.Background())
	if err == nil {
		t.Fatal("expected instruction-limit error for infinite loop, got nil")
	}

	msg := err.Error()
	for _, want := range []string{"infinite loop", "InfiniteLoop", "pc="} {
		if !strings.Contains(msg, want) {
			t.Errorf("instruction-limit error missing %q; got: %s", want, msg)
		}
	}
	t.Logf("diagnostic error: %s", msg)
}
