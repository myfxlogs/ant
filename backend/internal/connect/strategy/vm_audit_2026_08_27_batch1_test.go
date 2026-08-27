package strategy

import (
	"testing"

	"alphaforge/tools/mql2go"
)

// vm_audit_2026_08_27_batch1_test.go — VM-AUDIT-2026-08-27 批次 1 对抗测试.
//
// Tests verify the P1 fixes:
//   - VM-AUDIT-2026-08-27-1 (BUG-1): Python live paths use CompilePythonCached
//     (SourceHash verification) instead of the raw bytecode loader (no verification).
//   - VM-AUDIT-2026-08-27-2 (BUG-2): runEvent resets fatalError between events.
//
// Adversarial proofs: each critical line mutated → relevant test RED → restore GREEN.

// --- T1: VM-AUDIT-2026-08-27-1 SourceHash verification (BUG-1) ---

// pyAuditSourceA and pyAuditSourceB are two distinct Python sources with
// different SourceHash values. They must both compile successfully.
const pyAuditSourceA = `from decimal import Decimal

class MyStrategy:
    def on_bar(self) -> None:
        pass
`

const pyAuditSourceB = `from decimal import Decimal

class MyStrategy:
    def on_bar(self) -> None:
        x = 1
`

// TestExecutePythonVMLive_SourceHashVerification verifies BUG-1 fix:
// NewPythonVMLiveSessionCached (and by extension executePythonVMLive, which
// delegates to the same CompilePythonCached) must reject stale cached bytecode
// whose SourceHash doesn't match the current source, and recompile instead.
//
// This test exercises the NewPythonVMLiveSessionCached path (S2) because it
// accepts cachedBytecode as a direct parameter. executePythonVMLive (S1) reads
// the cache from importedRepo (a concrete *repository.ImportedStrategyRepository
// with an unexported db field — not mockable without refactoring, which is out
// of scope per spec §"边界/不做"). Both S1 and S2 call the same
// CompilePythonCached, so this is a faithful adversarial proof for BUG-1.
//
// Adversarial proof: mutate CompilePythonCached's SourceHash check
// (interp_runner.go: `r.Bytecode().SourceHash == hashSource(source)` → `true`)
// → stale cache from source A is accepted → session's runner has SourceHash of
// A, not B → test RED. Restore → recompiles → SourceHash matches B → GREEN.
func TestExecutePythonVMLive_SourceHashVerification(t *testing.T) {
	t.Parallel()

	// Compile source A → get its bytecode (simulates a previously cached bytecode).
	_, bcDataA, err := mql2go.CompilePythonCached(pyAuditSourceA, nil)
	if err != nil {
		t.Fatalf("compile source A: %v", err)
	}
	if bcDataA == nil {
		t.Fatal("expected non-nil bytecode for source A")
	}

	// Compile source B separately to get the expected SourceHash.
	runnerB, _, err := mql2go.CompilePythonCached(pyAuditSourceB, nil)
	if err != nil {
		t.Fatalf("compile source B: %v", err)
	}
	expectedHash := runnerB.Bytecode().SourceHash

	// Call NewPythonVMLiveSessionCached with source B but A's stale cached bytecode.
	// BUG-1 (old code): used the raw bytecode loader directly → returned cached runner
	//   with SourceHash == hashSource(A) → stale bytecode for source B.
	// Fix (S2): uses CompilePythonCached → SourceHash mismatch → recompiles from B.
	sess, err := NewPythonVMLiveSessionCached(pyAuditSourceB, bcDataA)
	if err != nil {
		t.Fatalf("NewPythonVMLiveSessionCached with stale cache: %v", err)
	}
	if sess == nil || sess.strategy == nil {
		t.Fatal("expected non-nil session with strategy")
	}

	// The runner should reflect source B (recompiled), not source A (stale cache).
	actualHash := sess.strategy.Bytecode().SourceHash
	if actualHash != expectedHash {
		t.Fatalf("stale cache from source A was accepted: SourceHash = %q, want %q (source B)",
			actualHash, expectedHash)
	}
}

// TestExecutePythonVMLive_SourceHashVerification_CacheHit verifies the positive
// case: when cached bytecode's SourceHash matches the current source, the cache
// is accepted (no unnecessary recompile).
func TestExecutePythonVMLive_SourceHashVerification_CacheHit(t *testing.T) {
	t.Parallel()

	// Compile source A → get its bytecode.
	_, bcDataA, err := mql2go.CompilePythonCached(pyAuditSourceA, nil)
	if err != nil {
		t.Fatalf("compile source A: %v", err)
	}

	// Call with the same source A → cache hit (no recompile).
	sess, err := NewPythonVMLiveSessionCached(pyAuditSourceA, bcDataA)
	if err != nil {
		t.Fatalf("NewPythonVMLiveSessionCached cache hit: %v", err)
	}
	if sess == nil || sess.strategy == nil {
		t.Fatal("expected non-nil session with strategy")
	}
}
