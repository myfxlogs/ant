package factorsvc

import (
	"testing"
)

func TestNewFactorMetrics(t *testing.T) {
	m := NewFactorMetrics()
	if m == nil {
		t.Fatal("NewFactorMetrics returned nil")
	}
	if m.EvalCount != 0 {
		t.Fatal("expected EvalCount=0")
	}
}

func TestFactorMetrics_RecordEval(t *testing.T) {
	m := NewFactorMetrics()
	m.RecordEval()
	m.RecordEval()
	if m.EvalCount != 2 {
		t.Fatalf("expected EvalCount=2, got %d", m.EvalCount)
	}
}

func TestFactorMetrics_RecordError(t *testing.T) {
	m := NewFactorMetrics()
	m.RecordError()
	if m.EvalErrors != 1 {
		t.Fatalf("expected EvalErrors=1, got %d", m.EvalErrors)
	}
}

func TestFactorMetrics_RecordBuffer(t *testing.T) {
	m := NewFactorMetrics()
	m.RecordBuffer(42)
	if m.BufferCount != 42 {
		t.Fatalf("expected BufferCount=42, got %d", m.BufferCount)
	}
}
