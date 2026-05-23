package factorsvc

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestWindowBufferPush(t *testing.T) {
	wb := NewWindowBuffer(5, zap.NewNop())

	for i := 0; i < 3; i++ {
		bar := &Bar{
			UserID:        "t1",
			Symbol:        "EURUSD",
			Period:        "M1",
			Open:          "1.0",
			High:          "1.0",
			Low:           "1.0",
			Close:         "1.0",
			CloseTsUnixMs: int64(i * 60000),
		}
		wb.Push("t1", "EURUSD", "M1", bar)
	}

	bars := wb.Snapshot("t1", "EURUSD", "M1", 10)
	if len(bars) != 3 {
		t.Fatalf("expected 3 bars, got %d", len(bars))
	}
}

func TestWindowBufferEviction(t *testing.T) {
	wb := NewWindowBuffer(3, zap.NewNop())

	for i := 0; i < 5; i++ {
		bar := &Bar{
			UserID:        "t1",
			Symbol:        "EURUSD",
			Period:        "M1",
			Open:          "1.0",
			High:          "1.0",
			Low:           "1.0",
			Close:         "1.0",
			CloseTsUnixMs: int64(i * 60000),
		}
		wb.Push("t1", "EURUSD", "M1", bar)
	}

	bars := wb.Snapshot("t1", "EURUSD", "M1", 10)
	if len(bars) != 3 {
		t.Fatalf("expected 3 bars after eviction, got %d", len(bars))
	}
}

func TestWindowBufferBootstrap(t *testing.T) {
	wb := NewWindowBuffer(10, zap.NewNop())
	specs := []BootstrapSpec{
		{UserID: "t1", Symbol: "EURUSD", Period: "D1", Limit: 20},
	}
	wb.Bootstrap(context.Background(), specs)

	bars := wb.Snapshot("t1", "EURUSD", "D1", 10)
	if len(bars) != 0 {
		t.Fatalf("expected 0 bars after bootstrap, got %d", len(bars))
	}
}

func TestWindowBufferIsolation(t *testing.T) {
	wb := NewWindowBuffer(10, zap.NewNop())

	bar1 := &Bar{UserID: "t1", Symbol: "EURUSD", Period: "M1", CloseTsUnixMs: 1,
		Open: "1.0", Close: "1.0", High: "1.0", Low: "1.0"}
	bar2 := &Bar{UserID: "t2", Symbol: "EURUSD", Period: "M1", CloseTsUnixMs: 1,
		Open: "2.0", Close: "2.0", High: "2.0", Low: "2.0"}

	wb.Push("t1", "EURUSD", "M1", bar1)
	wb.Push("t2", "EURUSD", "M1", bar2)

	bars1 := wb.Snapshot("t1", "EURUSD", "M1", 10)
	bars2 := wb.Snapshot("t2", "EURUSD", "M1", 10)

	if len(bars1) != 1 || len(bars2) != 1 {
		t.Fatal("tenant isolation broken")
	}
}

func TestWindowBufferPushRaw(t *testing.T) {
	wb := NewWindowBuffer(5, zap.NewNop())

	wb.PushRaw("t1", "EURUSD", "M1", 1.0, 1.5, 0.9, 1.2, 1000, 60000)
	wb.PushRaw("t1", "EURUSD", "M1", 1.1, 1.6, 1.0, 1.3, 1100, 120000)

	bars := wb.Snapshot("t1", "EURUSD", "M1", 10)
	if len(bars) != 2 {
		t.Fatalf("expected 2 bars, got %d", len(bars))
	}
	if bars[0].Close != 1.2 {
		t.Fatalf("expected first close=1.2, got %f", bars[0].Close)
	}
	if bars[1].Close != 1.3 {
		t.Fatalf("expected second close=1.3, got %f", bars[1].Close)
	}
}

func TestWindowBufferSnapshotLimit(t *testing.T) {
	wb := NewWindowBuffer(10, zap.NewNop())

	for i := 0; i < 5; i++ {
		bar := &Bar{
			UserID:        "t1",
			Symbol:        "EURUSD",
			Period:        "M1",
			Open:          "1.0",
			High:          "1.0",
			Low:           "1.0",
			Close:         "1.0",
			CloseTsUnixMs: int64(i * 60000),
		}
		wb.Push("t1", "EURUSD", "M1", bar)
	}

	bars := wb.Snapshot("t1", "EURUSD", "M1", 2)
	if len(bars) != 2 {
		t.Fatalf("expected 2 bars with limit, got %d", len(bars))
	}
}

func TestWindowBufferEmptySnapshot(t *testing.T) {
	wb := NewWindowBuffer(10, zap.NewNop())
	bars := wb.Snapshot("nonexistent", "EURUSD", "M1", 10)
	if bars != nil {
		t.Fatal("expected nil for non-existent key")
	}
}
