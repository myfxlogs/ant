package mthub

import (
	"sync"
	"testing"
)

func TestReconcileGate_DefaultAccept(t *testing.T) {
	g := NewReconcileGate()
	if !g.CanAccept("acc-1") {
		t.Fatal("new gate should accept orders")
	}
}

func TestReconcileGate_EnterBlocks(t *testing.T) {
	g := NewReconcileGate()
	g.EnterReconciling("acc-1")

	if g.CanAccept("acc-1") {
		t.Fatal("reconciling account should be blocked")
	}
	if !g.IsReconciling("acc-1") {
		t.Fatal("IsReconciling should return true")
	}
}

func TestReconcileGate_MarkReconciledUnblocks(t *testing.T) {
	g := NewReconcileGate()
	g.EnterReconciling("acc-1")
	g.MarkReconciled("acc-1")

	if !g.CanAccept("acc-1") {
		t.Fatal("reconciled account should be unblocked")
	}
}

func TestReconcileGate_IndependentAccounts(t *testing.T) {
	g := NewReconcileGate()
	g.EnterReconciling("acc-1")

	if !g.CanAccept("acc-2") {
		t.Fatal("unrelated account should not be blocked")
	}
}

func TestReconcileGate_EnterAll(t *testing.T) {
	g := NewReconcileGate()
	g.EnterAll([]string{"acc-1", "acc-2", "acc-3"})

	if g.ReconcilingCount() != 3 {
		t.Fatalf("want 3 reconciling, got %d", g.ReconcilingCount())
	}
}

func TestReconcileGate_Count(t *testing.T) {
	g := NewReconcileGate()
	g.EnterReconciling("acc-1")
	g.EnterReconciling("acc-2")
	if g.ReconcilingCount() != 2 {
		t.Fatal("count should be 2")
	}
	g.MarkReconciled("acc-1")
	if g.ReconcilingCount() != 1 {
		t.Fatal("count should be 1")
	}
}

func TestReconcileGate_Concurrent(t *testing.T) {
	g := NewReconcileGate()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			g.EnterReconciling("shared")
			g.MarkReconciled("shared")
		}()
	}
	wg.Wait()
	if g.ReconcilingCount() != 0 {
		t.Fatalf("concurrent enter/mark should leave count 0, got %d", g.ReconcilingCount())
	}
}
