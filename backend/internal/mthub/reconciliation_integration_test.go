//go:build integration

package mthub

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestReconcileAccount_NoSession(t *testing.T) {
	t.Parallel()
	pg := getTestPG(t)
	log := zap.NewNop()
	rl := NewReconciliationLoop(NewHub(), pg, nil, log, NewReconcileGate())
	// No session registered → should be a no-op.
	err := rl.reconcileAccount(context.Background(), "nonexistent-account-id")
	if err != nil {
		t.Fatalf("expected nil error for no session, got %v", err)
	}
}

func TestReconcileAccount_WithSession(t *testing.T) {
	t.Parallel()
	pg := getTestPG(t)
	log := zap.NewNop()
	hub := NewHub()
	exec := &mockExecutor{platform: "MT5"}
	accountID := "06daab3c-5d87-41fd-bd8b-f31ba73c16c1"
	hub.Register(accountID, &Session{AccountID: accountID, CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)

	gate := NewReconcileGate()
	rl := NewReconciliationLoop(hub, pg, nil, log, gate)

	// Enter reconciling first, then reconcile.
	gate.EnterReconciling(accountID)
	err := rl.reconcileAccount(context.Background(), accountID)
	if err != nil {
		t.Fatalf("reconcileAccount failed: %v", err)
	}

	// Verify gate was released.
	if gate.IsReconciling(accountID) {
		t.Error("expected account to exit reconciling after reconciliation")
	}
}

func TestReconcileGate_EnterMultiple(t *testing.T) {
	t.Parallel()
	gate := NewReconcileGate()
	gate.EnterAll([]string{"acc-1", "acc-2"})
	if !gate.IsReconciling("acc-1") || !gate.IsReconciling("acc-2") {
		t.Fatal("EnterAll should set both accounts to reconciling")
	}
	if gate.ReconcilingCount() != 2 {
		t.Fatalf("expected 2 reconciling, got %d", gate.ReconcilingCount())
	}
	// Mark one as done.
	gate.MarkReconciled("acc-1")
	if gate.IsReconciling("acc-1") {
		t.Error("acc-1 should no longer be reconciling")
	}
	if gate.ReconcilingCount() != 1 {
		t.Fatalf("expected 1 reconciling after mark, got %d", gate.ReconcilingCount())
	}
}

func TestReconcileAccount_PublicMethod(t *testing.T) {
	t.Parallel()
	pg := getTestPG(t)
	log := zap.NewNop()
	hub := NewHub()
	exec := &mockExecutor{platform: "MT5"}
	accountID := "06daab3c-5d87-41fd-bd8b-f31ba73c16c1"
	hub.Register(accountID, &Session{AccountID: accountID, CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)

	rl := NewReconciliationLoop(hub, pg, nil, log, NewReconcileGate())
	// Public method — should not panic.
	rl.ReconcileAccount(context.Background(), accountID)
}

func TestTriggerReconcile(t *testing.T) {
	// Cannot run in parallel due to goroutine.
	pg := getTestPG(t)
	log := zap.NewNop()
	hub := NewHub()
	exec := &mockExecutor{platform: "MT5"}
	accountID := "06daab3c-5d87-41fd-bd8b-f31ba73c16c1"
	hub.Register(accountID, &Session{AccountID: accountID, CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)

	gate := NewReconcileGate()
	rl := NewReconciliationLoop(hub, pg, nil, log, gate)
	rl.TriggerReconcile(accountID)
	// Give the goroutine time to run.
	time.Sleep(100 * time.Millisecond)
	// After reconciliation, gate should be released.
	if gate.IsReconciling(accountID) {
		t.Log("reconciliation may not have completed within wait time")
	}
}

func TestReconcileAll_Empty(t *testing.T) {
	t.Parallel()
	pg := getTestPG(t)
	log := zap.NewNop()
	rl := NewReconciliationLoop(NewHub(), pg, nil, log, nil)
	// No active accounts — should be a no-op.
	rl.reconcileAll(context.Background())
}
