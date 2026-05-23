package mdgateway

import (
	"testing"

	"go.uber.org/zap"
)

func TestCHWriter_ConfigDefaults(t *testing.T) {
	cfg := DefaultCHWriterConfig()
	if cfg.FlushInterval == 0 {
		t.Fatal("FlushInterval should have default")
	}
	if cfg.MaxBatchSize != 1000 {
		t.Fatalf("expected MaxBatchSize=1000, got %d", cfg.MaxBatchSize)
	}
}

func TestCHWriter_New(t *testing.T) {
	cfg := DefaultCHWriterConfig()
	w := NewCHWriter(cfg, nil, zap.NewNop())
	if w == nil {
		t.Fatal("NewCHWriter returned nil")
	}
	if cap(w.ticks) != 2000 {
		t.Fatalf("expected cap=2000, got %d", cap(w.ticks))
	}
}

func TestCHWriter_SetMetrics(t *testing.T) {
	cfg := DefaultCHWriterConfig()
	w := NewCHWriter(cfg, nil, zap.NewNop())
	m := NewMDMetrics(nil)
	w.SetMetrics(m)
	if w.metrics == nil {
		t.Fatal("metrics should be set")
	}
}

func TestCHWriter_WriteNilTick(t *testing.T) {
	cfg := DefaultCHWriterConfig()
	w := NewCHWriter(cfg, nil, zap.NewNop())
	// nil tick guard
	w.Write(nil)
	// should not panic
}

func TestCHWriter_Write(t *testing.T) {
	cfg := DefaultCHWriterConfig()
	w := NewCHWriter(cfg, nil, zap.NewNop())

	tick := &Tick{UserID: "u1", Symbol: "EURUSD"}
	w.Write(tick)
	// non-blocking write should succeed
}

func TestCHWriter_Close(t *testing.T) {
	cfg := DefaultCHWriterConfig()
	w := NewCHWriter(cfg, nil, zap.NewNop())
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}
