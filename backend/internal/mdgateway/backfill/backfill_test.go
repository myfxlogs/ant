package backfill

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.BatchSize != 1000 {
		t.Fatalf("expected BatchSize=1000, got %d", cfg.BatchSize)
	}
	if cfg.Concurrency != 4 {
		t.Fatalf("expected Concurrency=4, got %d", cfg.Concurrency)
	}
	if cfg.Period != "1m" {
		t.Fatalf("expected Period=1m, got %s", cfg.Period)
	}
}

func TestConfig_Fields(t *testing.T) {
	cfg := Config{
		BatchSize:   500,
		Concurrency: 2,
		FromTime:    time.Unix(1000, 0),
		ToTime:      time.Unix(2000, 0),
		Symbols:     []string{"EURUSD", "GBPJPY"},
		Period:      "1h",
	}
	if cfg.BatchSize != 500 {
		t.Fatalf("expected BatchSize=500, got %d", cfg.BatchSize)
	}
	if len(cfg.Symbols) != 2 {
		t.Fatalf("expected 2 symbols, got %d", len(cfg.Symbols))
	}
}

func TestNewRunner(t *testing.T) {
	r := New(nil, nil, nil)
	if r == nil {
		t.Fatal("New returned nil")
	}
}
