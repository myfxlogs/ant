package mdgateway

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/repository"
)

// recordingStore records InsertBars calls (implements repository.MarketDataStore).
type recordingStore struct {
	mu    sync.Mutex
	bars  []string // canonical of each inserted bar
	batch int
}

func (m *recordingStore) GetKlines(_ context.Context, _, _, _ string, _, _ *time.Time, _ int32) ([]repository.KlineBar, error) {
	return nil, nil
}
func (m *recordingStore) GetLatestTick(_ context.Context, _, _ string) (*repository.LatestTick, error) {
	return nil, nil
}
func (m *recordingStore) LoadFinalizedBars(_ context.Context, _ time.Time) (map[repository.FinalizedKey][]int64, error) {
	return nil, nil
}
func (m *recordingStore) MaxCloseTs(_ context.Context, _, _, _ string) (int64, error) {
	return 0, nil
}
func (m *recordingStore) GetLatestBars(_ context.Context, _ time.Time) ([]repository.KlineBar, error) {
	return nil, nil
}
func (m *recordingStore) FetchActualReturn(_ context.Context, _ string, _ time.Time) (float64, error) {
	return 0, nil
}
func (m *recordingStore) InsertBars(_ context.Context, bars []repository.KlineBar) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.batch++
	for _, b := range bars {
		m.bars = append(m.bars, b.Canonical)
	}
	return nil
}

func (m *recordingStore) InsertBarsCalled() (int, []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.batch, append([]string(nil), m.bars...)
}

// TestPGWRITER_TimeBasedFlush_LandsPromptly verifies that a SINGLE bar (low
// bar rate, e.g. on-demand subscription) is flushed to the store within
// FlushEvery — not held until the 2000-bar batch fills (~33h at 1 bar/min).
// Remove the ticker case in PgWriter.Start → bar never lands → RED.
func TestPGWRITER_TimeBasedFlush_LandsPromptly(t *testing.T) {
	store := &recordingStore{}
	w := NewPgWriter(PgWriterConfig{
		MaxBatchSize: 2000,
		QueueSize:    100,
		FlushEvery:   60 * time.Millisecond,
	}, store, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.Start(ctx)

	w.EnqueueBar(&mdtick.Bar{Canonical: "BTCUSDm", Period: "1m", Broker: "test"})

	deadline := time.Now().Add(2 * time.Second)
	for {
		n, syms := store.InsertBarsCalled()
		if n > 0 && len(syms) > 0 && syms[0] == "BTCUSDm" {
			return // GREEN: single bar landed via time-based flush
		}
		if time.Now().After(deadline) {
			t.Fatal("PGWRITER: single bar not flushed within FlushEvery — time-based flush missing (bar held for 2000-batch)")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
