package quantengine

import (
	"net/http"
	"testing"

	"go.uber.org/zap"
)

func TestRunQuantEngine(t *testing.T) {
	mux := http.NewServeMux()
	log := zap.NewNop()

	specs := []*StrategySpec{DefaultDemoSpec()}
	mgr := RunQuantEngine(mux, specs, log)

	if mgr == nil {
		t.Fatal("RunQuantEngine returned nil manager")
	}
	if mgr.Count() != 1 {
		t.Fatalf("expected 1 runtime, got %d", mgr.Count())
	}

	// Clean up
	for name := range mgr.SnapshotAll() {
		mgr.Remove(name)
	}
}

func TestRunQuantEngine_NilSpecs(t *testing.T) {
	mux := http.NewServeMux()
	log := zap.NewNop()

	mgr := RunQuantEngine(mux, nil, log)
	if mgr == nil {
		t.Fatal("expected non-nil manager even with nil specs")
	}
	if mgr.Count() != 0 {
		t.Fatalf("expected 0 runtimes, got %d", mgr.Count())
	}
}

func TestRunQuantEngine_WithSignalHandler(t *testing.T) {
	mux := http.NewServeMux()
	log := zap.NewNop()

	var captured string
	onSignal := func(strategyID, symbol, side string, qty float64, reason string) {
		captured = side
	}

	specs := []*StrategySpec{DefaultDemoSpec()}
	mgr := RunQuantEngineWithSignalHandler(mux, specs, onSignal, log)

	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}

	// Trigger evaluation
	mgr.OnBar()

	mgr.Remove(DefaultDemoSpec().Name)
	_ = captured
}
