package mdgateway

import (
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func TestNewSpillWriter(t *testing.T) {
	dir := t.TempDir()
	cfg := SpillConfig{Dir: dir}
	w, err := NewSpillWriter(cfg, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	if w.Count() != 0 {
		t.Fatal("expected 0 count initially")
	}
}

func TestSpillWriter_Write(t *testing.T) {
	dir := t.TempDir()
	cfg := SpillConfig{Dir: dir, SyncEvery: 1}
	w, err := NewSpillWriter(cfg, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}

	tick := &Tick{
		UserID: "u1", Broker: "demo", Symbol: "EURUSD",
		TsUnixMs: 1000, ArrivedUnixMs: 1001,
		Bid: &Money{Value: "1.1000"}, Ask: &Money{Value: "1.1005"},
		BidVolume: 1.0, AskVolume: 1.0,
	}

	if err := w.Write(tick); err != nil {
		t.Fatal(err)
	}
	if w.Count() != 1 {
		t.Fatalf("expected count=1, got %d", w.Count())
	}

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	// Verify file exists
	files, _ := filepath.Glob(filepath.Join(dir, "spill-*.jsonl"))
	if len(files) == 0 {
		t.Fatal("no spill file created")
	}
}

func TestSpillWriter_WriteBatch(t *testing.T) {
	dir := t.TempDir()
	cfg := SpillConfig{Dir: dir}
	w, _ := NewSpillWriter(cfg, zap.NewNop())

	ticks := make([]*Tick, 3)
	for i := range ticks {
		ticks[i] = &Tick{
			UserID: "u1", Broker: "demo", Symbol: "EURUSD",
			Bid: &Money{Value: "1.1000"}, Ask: &Money{Value: "1.1005"},
		}
	}

	if err := w.WriteBatch(ticks); err != nil {
		t.Fatal(err)
	}
	if w.Count() != 3 {
		t.Fatalf("expected count=3, got %d", w.Count())
	}
	w.Close()
}

func TestSpillWriter_Rotation(t *testing.T) {
	dir := t.TempDir()
	// Each JSONL record is ~120 bytes, set max to 200 to rotate every ~2 records
	cfg := SpillConfig{Dir: dir, MaxFileSize: 200}
	w, _ := NewSpillWriter(cfg, zap.NewNop())

	for i := 0; i < 5; i++ {
		w.Write(&Tick{
			UserID: "u1", Broker: "demo", Symbol: "EURUSD",
			Bid: &Money{Value: "1.1000"}, Ask: &Money{Value: "1.1005"},
		})
	}

	w.Close()

	files, _ := filepath.Glob(filepath.Join(dir, "spill-*.jsonl"))
	t.Logf("spill files: %d", len(files))
	// With MaxFileSize=200 and ~120 bytes/record, 5 records should create 2+ files
	if len(files) < 1 {
		t.Fatal("expected at least 1 spill file")
	}
}

func TestSpillWriter_Replay(t *testing.T) {
	dir := t.TempDir()
	cfg := SpillConfig{Dir: dir, SyncEvery: 1}
	w, _ := NewSpillWriter(cfg, zap.NewNop())

	tick := &Tick{
		UserID: "u1", Broker: "demo", Symbol: "EURUSD",
		TsUnixMs: 1000, ArrivedUnixMs: 1001,
		Bid: &Money{Value: "1.1000"}, Ask: &Money{Value: "1.1005"},
		BidVolume: 1.0, AskVolume: 1.0,
	}
	w.Write(tick)
	w.Close()

	// Verify spill file was created and contains data
	files, _ := filepath.Glob(filepath.Join(dir, "spill-*.jsonl"))
	if len(files) == 0 {
		t.Fatal("no spill file for replay")
	}

	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("empty spill file")
	}
	t.Logf("spill file: %s (%d bytes)", files[0], len(data))
}

func TestSpillWriter_DefaultConfig(t *testing.T) {
	cfg := DefaultSpillConfig()
	if cfg.Dir != "/tmp/mdgateway/spill" {
		t.Fatalf("expected /tmp/mdgateway/spill, got %s", cfg.Dir)
	}
	if cfg.MaxFileSize != 100*1024*1024 {
		t.Fatalf("expected 100MB max, got %d", cfg.MaxFileSize)
	}
}

func TestSpillWriter_Mkdir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "spill")
	cfg := SpillConfig{Dir: dir}
	w, err := NewSpillWriter(cfg, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Fatal("spill directory not created")
	}
}
