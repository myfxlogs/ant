// Package mthub provides the reconcile-before-accept gate (M11-2).
// On broker (re)connect, accounts enter Reconciling state and PlaceOrder
// is rejected until reconciliation completes — preventing ghost/duplicate orders.
package mthub

import "sync"

// ReconcileGate controls whether an account can accept new orders.
// Accounts are "locked" (reconciling) on startup/reconnect and "unlocked"
// once the broker-side position pull + diff completes successfully.
type ReconcileGate struct {
	mu          sync.RWMutex
	reconciling map[string]bool
}

// NewReconcileGate creates a new gate with no accounts reconciling.
func NewReconcileGate() *ReconcileGate {
	return &ReconcileGate{
		reconciling: make(map[string]bool),
	}
}

// EnterReconciling marks an account as reconciling, blocking new orders.
func (g *ReconcileGate) EnterReconciling(accountID string) {
	g.mu.Lock()
	g.reconciling[accountID] = true
	g.mu.Unlock()
}

// MarkReconciled marks an account's reconciliation as complete.
func (g *ReconcileGate) MarkReconciled(accountID string) {
	g.mu.Lock()
	delete(g.reconciling, accountID)
	g.mu.Unlock()
}

// CanAccept returns true if the account is NOT currently reconciling.
func (g *ReconcileGate) CanAccept(accountID string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return !g.reconciling[accountID]
}

// IsReconciling returns true if the account is currently reconciling.
func (g *ReconcileGate) IsReconciling(accountID string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.reconciling[accountID]
}

// ReconcilingCount returns the number of accounts still in reconciliation.
func (g *ReconcileGate) ReconcilingCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.reconciling)
}

// EnterAll marks multiple accounts as reconciling at once.
func (g *ReconcileGate) EnterAll(accountIDs []string) {
	g.mu.Lock()
	for _, id := range accountIDs {
		g.reconciling[id] = true
	}
	g.mu.Unlock()
}
