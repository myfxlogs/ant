package mdgateway

import (
	"context"
	"path/filepath"
	"testing"

	"go.uber.org/zap"

	"anttrader/internal/mdgateway/adapter/mdtick"
)

func TestSpillReplayDualWrite(t *testing.T) {
	dir := t.TempDir()

	// Create a spill file with one tick and one bar.
	spillCfg := DefaultSpillConfig()
	spillCfg.Dir = dir
	spillCfg.MaxFileBytes = 10 * 1024 * 1024
	sw, err := NewSpillWriter(spillCfg, zap.NewNop())
	if err != nil {
		t.Fatalf("NewSpillWriter: %v", err)
	}

	tick := &mdtick.Tick{
		Broker: "test-broker", Canonical: "EURUSD",
		TsUnixMs: 1000, ArrivedUnixMs: 1000,
		Bid: requireDecimal(t, "1.08000"), Ask: requireDecimal(t, "1.08002"),
		BidVolume: 1000, AskVolume: 500,
	}
	if err := sw.WriteTick(tick); err != nil {
		t.Fatalf("WriteTick: %v", err)
	}
	sw.Close()

	// Create SpillReplay with nil publisher and nil CHWriter (dual-write no-op).
	sr := NewSpillReplay(dir, nil, nil, zap.NewNop())

	n, err := sr.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 row replayed, got %d", n)
	}

	// Check that the file was moved to processed/.
	files, _ := filepath.Glob(filepath.Join(dir, "processed", "spill-*.jsonl"))
	if len(files) == 0 {
		t.Error("spill file should be moved to processed/")
	}
	if len(files) > 0 {
		t.Logf("SpillReplayDualWrite: replay succeeded, file moved to %s", files[0])
	}
}
