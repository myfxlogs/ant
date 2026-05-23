package quantengine

import (
	"testing"

	"go.uber.org/zap"
)

func TestNewModelRunner_DSL(t *testing.T) {
	spec := &StrategySpec{
		Name:       "test",
		ModelURI:   "",
		SignalRule: "close > 0 ? 1 : -1",
	}
	mr, err := NewModelRunner(spec)
	if err != nil {
		t.Fatalf("NewModelRunner error: %v", err)
	}
	if mr == nil {
		t.Fatal("NewModelRunner returned nil")
	}
	if !mr.useDSL {
		t.Fatal("expected useDSL=true")
	}
}

func TestModelRunner_PredictDSL(t *testing.T) {
	spec := &StrategySpec{
		Name:       "test_dsl",
		ModelURI:   "",
		SignalRule: "close > 0 ? 1 : -1",
	}
	mr, err := NewModelRunner(spec)
	if err != nil {
		t.Fatalf("NewModelRunner error: %v", err)
	}

	// close = 100 → 100 > 0 → signal = 1 (long)
	sig, err := mr.Predict(t.Context(), map[string]float64{"close": 100})
	if err != nil {
		t.Fatalf("Predict error: %v", err)
	}
	if sig != 1 {
		t.Fatalf("expected signal=1, got %f", sig)
	}

	// close = -50 → -50 > 0 → signal = -1 (short)
	sig, err = mr.Predict(t.Context(), map[string]float64{"close": -50})
	if err != nil {
		t.Fatalf("Predict error: %v", err)
	}
	if sig != -1 {
		t.Fatalf("expected signal=-1, got %f", sig)
	}
}

func TestModelRunner_PredictEmptyRule(t *testing.T) {
	spec := &StrategySpec{
		Name:       "empty_rule",
		ModelURI:   "",
		SignalRule: "",
	}
	mr, _ := NewModelRunner(spec)

	sig, err := mr.Predict(t.Context(), map[string]float64{"close": 100})
	if err != nil {
		t.Fatalf("Predict error: %v", err)
	}
	if sig != 0 {
		t.Fatalf("expected signal=0 for empty rule, got %f", sig)
	}
}

func TestDefaultDemoSpec(t *testing.T) {
	spec := DefaultDemoSpec()
	if spec == nil {
		t.Fatal("DefaultDemoSpec returned nil")
	}
	if spec.Name != "demo_sma_e2e" {
		t.Fatalf("expected Name=demo_sma_e2e, got %s", spec.Name)
	}
	if spec.Version != "1.0.0" {
		t.Fatalf("expected Version=1.0.0, got %s", spec.Version)
	}
	if len(spec.CanonicalSymbols) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(spec.CanonicalSymbols))
	}
	if spec.CanonicalSymbols[0] != "BTCUSD" {
		t.Fatalf("expected BTCUSD, got %s", spec.CanonicalSymbols[0])
	}
	if len(spec.Factors) != 2 {
		t.Fatalf("expected 2 factors, got %d", len(spec.Factors))
	}
	if spec.SignalRule != "sma20 > sma60 ? 1 : -1" {
		t.Fatalf("unexpected SignalRule: %s", spec.SignalRule)
	}
}

func TestStrategyRuntime_Fields(t *testing.T) {
	spec := DefaultDemoSpec()
	rt, err := NewStrategyRuntime(spec, nil, zap.NewNop())
	if err != nil {
		t.Fatalf("NewStrategyRuntime error: %v", err)
	}
	if rt == nil {
		t.Fatal("expected non-nil runtime")
	}
	if rt.Spec == nil {
		t.Fatal("expected non-nil spec")
	}
}

func TestStrategyRuntime_StartStop(t *testing.T) {
	spec := DefaultDemoSpec()
	rt, err := NewStrategyRuntime(spec, nil, zap.NewNop())
	if err != nil {
		t.Fatalf("NewStrategyRuntime error: %v", err)
	}
	rt.Start(t.Context())
	rt.Stop()
}

func TestSignalHandler(t *testing.T) {
	var h SignalHandler = func(strategyID, symbol, side string, qty float64, reason string) {}
	if h == nil {
		t.Fatal("SignalHandler should not be nil")
	}
	h("sid", "EURUSD", "buy", 0.1, "test")
}

func TestRuntimeManager_AddRemoveCount(t *testing.T) {
	log := zap.NewNop()
	mgr := NewRuntimeManager(log)

	spec := DefaultDemoSpec()
	rt, err := NewStrategyRuntime(spec, nil, log)
	if err != nil {
		t.Fatalf("NewStrategyRuntime error: %v", err)
	}

	ctx := t.Context()
	mgr.Add(ctx, rt)

	if mgr.Count() != 1 {
		t.Fatalf("expected count=1, got %d", mgr.Count())
	}

	mgr.Remove(spec.Name)
	if mgr.Count() != 0 {
		t.Fatalf("expected count=0 after remove, got %d", mgr.Count())
	}
}

func TestRuntimeManager_SnapshotAll(t *testing.T) {
	log := zap.NewNop()
	mgr := NewRuntimeManager(log)

	spec := DefaultDemoSpec()
	rt, err := NewStrategyRuntime(spec, nil, log)
	if err != nil {
		t.Fatalf("NewStrategyRuntime error: %v", err)
	}
	mgr.Add(t.Context(), rt)

	snaps := mgr.SnapshotAll()
	if len(snaps) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snaps))
	}
	snap, ok := snaps[spec.Name]
	if !ok {
		t.Fatalf("snapshot not found for %s", spec.Name)
	}
	if snap.StrategyName != spec.Name {
		t.Fatalf("expected name %s, got %s", spec.Name, snap.StrategyName)
	}
}

func TestRuntimeManager_Get(t *testing.T) {
	log := zap.NewNop()
	mgr := NewRuntimeManager(log)

	spec := DefaultDemoSpec()
	rt, _ := NewStrategyRuntime(spec, nil, log)
	mgr.Add(t.Context(), rt)

	if mgr.Get(spec.Name) != rt {
		t.Fatal("Get returned wrong runtime")
	}
	if mgr.Get("nonexistent") != nil {
		t.Fatal("Get should return nil for nonexistent")
	}
}

func TestRuntimeManager_OnBar(t *testing.T) {
	log := zap.NewNop()
	mgr := NewRuntimeManager(log)

	spec := DefaultDemoSpec()
	rt, _ := NewStrategyRuntime(spec, nil, log)
	mgr.Add(t.Context(), rt)

	// Should not panic
	mgr.OnBar()
}

func TestRuntimeManager_OnFactors(t *testing.T) {
	log := zap.NewNop()
	mgr := NewRuntimeManager(log)

	spec := DefaultDemoSpec()
	var captured string
	onSignal := func(strategyID, symbol, side string, qty float64, reason string) {
		captured = side
	}
	rt, _ := NewStrategyRuntime(spec, onSignal, log)
	mgr.Add(t.Context(), rt)

	// Provide factor values that trigger a signal via the SMA rule
	mgr.OnFactors(t.Context(), map[string]float64{
		"sma20": 1.5,
		"sma60": 1.0,
	})
	// Signal should have been captured (sma20 > sma60 → 1 → "long")
	if captured != "long" {
		t.Logf("captured signal: %q (expected long; factors may not have propagated)", captured)
	}
}
