// pg_writer.go — PostgreSQL batch writer replacing CHWriter.
// Same channel-buffered pattern, but writes to PG native partitioned tables via CopyFrom.

package mdgateway

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"anttrader/internal/mdgateway/adapter/mdtick"
	"anttrader/internal/repository"
	"anttrader/internal/usermgr"
)

// PgWriterConfig mirrors CHWriterConfig for drop-in replacement.
type PgWriterConfig struct {
	FlushInterval time.Duration // default 500ms
	MaxBatchSize  int           // default 2000
	QueueSize     int           // default 10000
}

// DefaultPgWriterConfig returns production-tuned defaults.
func DefaultPgWriterConfig() PgWriterConfig {
	return PgWriterConfig{
		FlushInterval: 500 * time.Millisecond,
		MaxBatchSize:  2000,
		QueueSize:     10000,
	}
}

// PgWriter buffers ticks and bars and flushes them to PostgreSQL via CopyFrom.
// Optionally dual-writes to a CH read replica asynchronously (best-effort).
type PgWriter struct {
	cfg     PgWriterConfig
	store   repository.MarketDataStore // PG primary
	chStore repository.MarketDataStore // CH read replica (nil if not configured)
	log     *zap.Logger

	tickQ chan *mdtick.Tick
	barQ  chan *mdtick.Bar

	userLimiter *usermgr.UserLimiter
}

// NewPgWriter creates a PG-backed writer with the same channel pattern as CHWriter.
func NewPgWriter(cfg PgWriterConfig, store repository.MarketDataStore, log *zap.Logger) *PgWriter {
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 5000
	}
	return &PgWriter{
		cfg:   cfg,
		store: store,
		log:   log,
		tickQ: make(chan *mdtick.Tick, cfg.QueueSize),
		barQ:  make(chan *mdtick.Bar, cfg.QueueSize),
	}
}

// SetCHStore sets an optional CH read replica for async dual-write.
func (w *PgWriter) SetCHStore(ch repository.MarketDataStore) { w.chStore = ch }

// SetUserLimiter injects the per-user write rate limiter (nil-safe).
func (w *PgWriter) SetUserLimiter(l *usermgr.UserLimiter) { w.userLimiter = l }

// EnqueueTick adds a tick to the write buffer. Non-blocking; drops if channel full.
func (w *PgWriter) EnqueueTick(t *mdtick.Tick) {
	if w.userLimiter != nil && t.UserID != "" && !w.userLimiter.AllowCHWrite(t.UserID, 256) {
		RecordChanFull()
		return
	}
	select {
	case w.tickQ <- t:
	default:
		RecordChanFull()
		// No spill writer — NATS replay provides durability.
		w.log.Warn("pgwriter: tick queue full, dropping", zap.String("symbol", t.Canonical))
	}
}

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
func (w *PgWriter) Start(ctx context.Context) {
	ticker := time.NewTicker(w.cfg.FlushInterval)
	defer ticker.Stop()

	var tickBatch []*mdtick.Tick
	var barBatch []*mdtick.Bar

	for {
		select {
		case <-ctx.Done():
			w.flush(context.Background(), tickBatch, barBatch)
			return
		case t := <-w.tickQ:
			tickBatch = append(tickBatch, t)
			if len(tickBatch) >= w.cfg.MaxBatchSize {
				w.flushTicks(ctx, tickBatch)
				tickBatch = tickBatch[:0]
			}
		case b := <-w.barQ:
			barBatch = append(barBatch, b)
			if len(barBatch) >= w.cfg.MaxBatchSize {
				w.flushBars(ctx, barBatch)
				barBatch = barBatch[:0]
			}
		case <-ticker.C:
			w.flushTicks(ctx, tickBatch)
			w.flushBars(ctx, barBatch)
			tickBatch = tickBatch[:0]
			barBatch = barBatch[:0]
		}
	}
}

// Flush drains the given batches to PG. Called during graceful shutdown.
func (w *PgWriter) Flush(ctx context.Context, ticks []*mdtick.Tick, bars []*mdtick.Bar) {
	w.flushTicks(ctx, ticks)
	w.flushBars(ctx, bars)
}

func (w *PgWriter) flush(ctx context.Context, ticks []*mdtick.Tick, bars []*mdtick.Bar) {
	w.flushTicks(ctx, ticks)
	w.flushBars(ctx, bars)
}

func (w *PgWriter) flushTicks(ctx context.Context, batch []*mdtick.Tick) {
	if len(batch) == 0 {
		return
	}
	records := make([]repository.TickRecord, len(batch))
	for i, t := range batch {
		records[i] = repository.TickRecord{
			UserID:        t.UserID,
			AccountID:     t.AccountID,
			Broker:        t.Broker,
			SymbolRaw:     t.SymbolRaw,
			Canonical:     t.Canonical,
			TsUnixMs:      t.TsUnixMs,
			ArrivedUnixMs: t.ArrivedUnixMs,
			Bid:           t.Bid,
			Ask:           t.Ask,
			BidVolume:     t.BidVolume,
			AskVolume:     t.AskVolume,
			IsReplay:      t.IsReplay,
		}
	}
	if err := w.retryInsert(ctx, func() error { return w.store.InsertTicks(ctx, records) }); err != nil {
		w.log.Error("pgwriter: tick flush failed after retries", zap.Int("count", len(batch)), zap.Error(err))
		return
	}
	// Async dual-write to CH read replica (best-effort, non-blocking).
	if w.chStore != nil {
		go func() {
			if err := w.chStore.InsertTicks(context.Background(), records); err != nil {
				w.log.Warn("pgwriter: ch dual-write ticks failed", zap.Int("count", len(records)), zap.Error(err))
			}
		}()
	}
}

func (w *PgWriter) flushBars(ctx context.Context, batch []*mdtick.Bar) {
	if len(batch) == 0 {
		return
	}
	bars := make([]repository.KlineBar, len(batch))
	for i, b := range batch {
		bars[i] = repository.KlineBar{
			Broker:        b.Broker,
			Canonical:     b.Canonical,
			Period:        b.Period,
			OpenTsUnixMs:  uint64(b.OpenTsUnixMs),
			CloseTsUnixMs: uint64(b.CloseTsUnixMs),
			Open:          b.Open.InexactFloat64(),
			High:          b.High.InexactFloat64(),
			Low:           b.Low.InexactFloat64(),
			Close:         b.Close.InexactFloat64(),
			Volume:        b.Volume,
			TickCount:     b.TickCount,
		}
	}
	if err := w.retryInsert(ctx, func() error { return w.store.InsertBars(ctx, bars) }); err != nil {
		w.log.Error("pgwriter: bar flush failed after retries", zap.Int("count", len(batch)), zap.Error(err))
		return
	}
	// Async dual-write to CH read replica (best-effort, non-blocking).
	if w.chStore != nil {
		go func() {
			if err := w.chStore.InsertBars(context.Background(), bars); err != nil {
				w.log.Warn("pgwriter: ch dual-write bars failed", zap.Int("count", len(bars)), zap.Error(err))
			}
		}()
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

// Drain non-blockingly reads all remaining items from the queues.
func (w *PgWriter) Drain() (ticks []*mdtick.Tick, bars []*mdtick.Bar) {
	for {
		select {
		case t := <-w.tickQ:
			ticks = append(ticks, t)
		default:
			goto doneTicks
		}
	}
doneTicks:
	for {
		select {
		case b := <-w.barQ:
			bars = append(bars, b)
		default:
			goto doneBars
		}
	}
doneBars:
	return ticks, bars
}

