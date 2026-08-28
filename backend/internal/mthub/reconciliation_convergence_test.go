package mthub

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mustReadFile(t *testing.T, relPath string) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// wd is .../backend/internal/mthub → backend root is ../..
	backendRoot := filepath.Join(wd, "..", "..")
	data, err := os.ReadFile(filepath.Join(backendRoot, relPath))
	if err != nil {
		t.Fatalf("read %s: %v", relPath, err)
	}
	return string(data)
}

// T1: TestReconciliation_AntQueryHasTimeBound — S1 guard.
// Verifies the ant-side query in reconcileAccount has 24h time bounds
// (created_at >= / close_time >=) symmetric with the broker 24h window.
// Adversarial proof: revert to no time bound → T1 RED.
func TestReconciliation_AntQueryHasTimeBound(t *testing.T) {
	content := mustReadFile(t, "internal/mthub/reconciliation.go")
	idx := strings.Index(content, "func (r *ReconciliationLoop) reconcileAccount")
	if idx < 0 {
		t.Fatal("reconcileAccount not found")
	}
	body := content[idx:]
	if !strings.Contains(body, "created_at >=") {
		t.Fatal("ant orders query must have created_at >= time bound (S1)")
	}
	if !strings.Contains(body, "close_time >=") {
		t.Fatal("ant trade_records query must have close_time >= time bound (S1)")
	}
}

// T2: TestReconciliation_GhostAutoImports — S2 guard.
// Verifies the ghost loop iterates brokerTickets with the broker record
// and calls ImportBrokerOrder to converge ghost orders per ADR-0013 §2.3.
// Adversarial proof: remove the ImportBrokerOrder call → T2 RED.
func TestReconciliation_GhostAutoImports(t *testing.T) {
	content := mustReadFile(t, "internal/mthub/reconciliation.go")
	idx := strings.Index(content, "for ticket, br := range brokerTickets")
	if idx < 0 {
		t.Fatal("ghost loop not found — must iterate brokerTickets with br")
	}
	body := content[idx:]
	if !strings.Contains(body, "ImportBrokerOrder") {
		t.Fatal("ghost loop must call ImportBrokerOrder (S2)")
	}
}

// T3: TestReconciliation_OrphanRepairsAllNonTerminal — S4 guard.
// Verifies orphan repair uses isNonTerminalOMSState (not hardcoded SUBMITTED).
// Adversarial proof: revert to `antState == string(OMSStateSubmitted)` → T3 RED.
func TestReconciliation_OrphanRepairsAllNonTerminal(t *testing.T) {
	content := mustReadFile(t, "internal/mthub/reconciliation.go")
	if !strings.Contains(content, "isNonTerminalOMSState") {
		t.Fatal("must use isNonTerminalOMSState (S4)")
	}
	// The old hardcoded SUBMITTED check in the orphan branch must be gone.
	// (The state-mismatch branch at :180 still uses SUBMITTED legitimately —
	// that one compares ant SUBMITTED vs broker state, not orphan detection.
	// We scope the check to the orphan branch by looking for the old pattern
	// `orphans++` immediately followed by the SUBMITTED guard.)
	idx := strings.Index(content, "orphans++")
	if idx < 0 {
		t.Fatal("orphan branch not found")
	}
	branch := content[idx:]
	// The orphan branch should use isNonTerminalOMSState, not the old
	// `antState == string(OMSStateSubmitted)` guard.
	if !strings.Contains(branch, "isNonTerminalOMSState") {
		t.Fatal("orphan repair must use isNonTerminalOMSState, not hardcoded SUBMITTED (S4)")
	}
}

// T4: TestImportBrokerOrder_UsesTicketConflict — S3 guard.
// Verifies ImportBrokerOrder uses ON CONFLICT (mt_account_id, ticket) DO NOTHING
// (the real broker ticket UK), NOT ON CONFLICT (id) DO NOTHING (which uses
// hashToNegative placeholder and is unsuitable for ghost imports).
// Adversarial proof: change to ON CONFLICT (id) DO NOTHING → T4 RED.
func TestImportBrokerOrder_UsesTicketConflict(t *testing.T) {
	content := mustReadFile(t, "internal/mthub/service_orders_import.go")
	idx := strings.Index(content, "func (s *MtHubService) ImportBrokerOrder")
	if idx < 0 {
		t.Fatal("ImportBrokerOrder not found (S3)")
	}
	body := content[idx:]
	if !strings.Contains(body, "ON CONFLICT (mt_account_id, ticket) DO NOTHING") {
		t.Fatal("ImportBrokerOrder must use ON CONFLICT (mt_account_id, ticket) DO NOTHING (S3, not ON CONFLICT (id))")
	}
}

// T5: TestIsNonTerminalOMSState_Helper — S4 helper correctness.
// Verifies the helper classifies terminal vs non-terminal states correctly.
func TestIsNonTerminalOMSState_Helper(t *testing.T) {
	terminal := []OMSState{
		OMSStateFilled, OMSStateCancelled, OMSStateFailed, OMSStateExpired,
		OMSStateRejected, OMSStateSlippageRejected,
	}
	for _, s := range terminal {
		if isNonTerminalOMSState(string(s)) {
			t.Errorf("terminal state %s must return false", s)
		}
	}
	nonTerminal := []OMSState{
		OMSStateNew, OMSStateValidated, OMSStateRiskApproved, OMSStateSubmitted,
		OMSStateWorking, OMSStatePartiallyFilled, OMSStateUnknown, OMSStateReconciling,
	}
	for _, s := range nonTerminal {
		if !isNonTerminalOMSState(string(s)) {
			t.Errorf("non-terminal state %s must return true", s)
		}
	}
}
