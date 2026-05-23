package quantengine

import (
	"testing"
)

func TestNewModelRunner(t *testing.T) {
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
}

func TestModelRunner_Fields(t *testing.T) {
	spec := &StrategySpec{
		Name:       "test",
		ModelURI:   "",
		SignalRule: "close > 0 ? 1 : -1",
	}
	mr, _ := NewModelRunner(spec)
	if mr.spec != spec {
		t.Fatal("expected spec to match")
	}
	if !mr.useDSL {
		t.Fatal("expected useDSL to be true when ModelURI is empty")
	}
}

func TestNewModelRunner_WithModelURI(t *testing.T) {
	spec := &StrategySpec{
		Name:       "test_onnx",
		ModelURI:   "s3://models/test.onnx",
		SignalRule: "close > 0 ? 1 : -1",
	}
	mr, err := NewModelRunner(spec)
	if err != nil {
		t.Fatalf("NewModelRunner error: %v", err)
	}
	if mr == nil {
		t.Fatal("NewModelRunner returned nil")
	}
	// Without CGO, should fall back to DSL.
	if !mr.useDSL {
		t.Log("ONNX loaded (cgo enabled)")
	}
}

func TestDirection(t *testing.T) {
	if Direction(1.0) != "long" {
		t.Fatalf("expected long, got %s", Direction(1.0))
	}
	if Direction(-1.0) != "short" {
		t.Fatalf("expected short, got %s", Direction(-1.0))
	}
	if Direction(0.0) != "flat" {
		t.Fatalf("expected flat, got %s", Direction(0.0))
	}
}
