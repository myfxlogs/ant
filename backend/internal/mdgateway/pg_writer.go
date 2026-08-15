// pg_writer.go — PostgreSQL batch writer for market data.
// Channel-buffered pattern, writes to PG native partitioned tables.

package mdgateway

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/repository"
	"alphaforge/internal/usermgr"
)

// PgWriterConfig controls batch sizes and queue capacity.
type PgWriterConfig struct {
	MaxBatchSize int           // default 2000
	QueueSize    int           // default 10000
	FlushEvery   time.Duration // default 30s; time-based flush so low bar rates
	// (on-demand subscriptions produce ~1 bar/min) still land in PG promptly.
	// Without it a 2000-bar batch at 1 bar/min takes ~33h to reach PG.
}

// DefaultPgWriterConfig returns production-tuned defaults.
func DefaultPgWriterConfig() PgWriterConfig {
	return PgWriterConfig{
		MaxBatchSize: 2000,
		QueueSize:    10000,
		FlushEvery:   30 * time.Second,
	}
}

// PgWriter buffers bars and flushes them to PostgreSQL via batch INSERT.
type PgWriter struct {
	cfg     PgWriterConfig
	store   repository.MarketDataStore // PG sole storage (ADR-0012)
	log     *zap.Logger

	barQ chan *mdtick.Bar

	userLimiter *usermgr.UserLimiter
}

// NewPgWriter creates a PG-backed writer with channel-buffered batch flushing.
func NewPgWriter(cfg PgWriterConfig, store repository.MarketDataStore, log *zap.Logger) *PgWriter {
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 5000
	}
	return &PgWriter{
		cfg:   cfg,
		store: store,
		log:   log,
		barQ:  make(chan *mdtick.Bar, cfg.QueueSize),
	}
}

// SetUserLimiter injects the per-user write rate limiter (nil-safe).
func (w *PgWriter) SetUserLimiter(l *usermgr.UserLimiter) { w.userLimiter = l }

// EnqueueBar adds a bar to the write buffer. Non-blocking; drops if channel full.
func (w *PgWriter) EnqueueBar(b *mdtick.Bar) {
	if w.userLimiter != nil && b.UserID != "" && !w.userLimiter.AllowCHWrite(b.UserID, 512) {
		RecordChanFull()
		return
	}
	select {
	case w.barQ <- b:
	default:
		RecordChanFull()
		w.log.Warn("pgwriter: bar queue full, dropping", zap.String("symbol", b.Canonical))
	}
}

// Start begins the flush loop. Blocks until ctx.Done.
// Flush triggers: batch reaches MaxBatchSize, FlushEvery elapses (partial
// batch), or shutdown. The time trigger keeps PG fresh under low bar rates
// (on-demand subscriptions) where a full 2000-bar batch may take a day.
func (w *PgWriter) Start(ctx context.Context) {
	var barBatch []*mdtick.Bar
	flushEvery := w.cfg.FlushEvery
	if flushEvery <= 0 {
		flushEvery = 30 * time.Second
	}
	ticker := time.NewTicker(flushEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.flushBars(ctx, barBatch)
			return
		case b := <-w.barQ:
			barBatch = append(barBatch, b)
			if len(barBatch) >= w.cfg.MaxBatchSize {
				w.flushBars(ctx, barBatch)
				barBatch = barBatch[:0]
			}
		case <-ticker.C:
			if len(barBatch) > 0 {
				w.flushBars(ctx, barBatch)
				barBatch = barBatch[:0]
			}
		}
	}
}

// Flush drains the given bars to PG. Called during graceful shutdown.
func (w *PgWriter) Flush(ctx context.Context, bars []*mdtick.Bar) {
	w.flushBars(ctx, bars)
}

func (w *PgWriter) flushBars(ctx context.Context, batch []*mdtick.Bar) {
	if len(batch) == 0 {
		return
	}
	bars := make([]repository.KlineBar, len(batch))
	for i, b := range batch {
		bars[i] = repository.KlineBar{
			Broker:        b.Broker,
			SymbolRaw:     b.SymbolRaw,
			Canonical:     b.Canonical,
			Period:        b.Period,
			OpenTsUnixMs:  uint64(b.OpenTsUnixMs),
			CloseTsUnixMs: uint64(b.CloseTsUnixMs),
			Open:          b.Open,
			High:          b.High,
			Low:           b.Low,
			Close:         b.Close,
			Volume:        b.Volume,
			TickCount:     b.TickCount,
			IsReplay:      b.IsReplay,
			AccountID:     b.AccountID,
		}
	}
	if err := w.retryInsert(ctx, func() error { return w.store.InsertBars(ctx, bars) }); err != nil {
		w.log.Error("pgwriter: bar flush failed after retries", zap.Int("count", len(batch)), zap.Error(err))
		return
	}
}

// retryInsert retries an insert operation up to 3 times with exponential backoff.
func (w *PgWriter) retryInsert(ctx context.Context, fn func() error) error {
	var err error
	for attempt := 1; attempt <= 3; attempt++ {
		err = fn()
		if err == nil {
			return nil
		}
		if attempt < 3 {
			backoff := time.Duration(200*(1<<(attempt-1))) * time.Millisecond
			time.Sleep(backoff)
		}
	}
	return fmt.Errorf("pgwriter: insert failed after 3 attempts: %w", err)
}

// Drain non-blockingly reads all remaining bars from the queue.
func (w *PgWriter) Drain() []*mdtick.Bar {
	var bars []*mdtick.Bar
	for {
		select {
		case b := <-w.barQ:
			bars = append(bars, b)
		default:
			return bars
		}
	}
}
