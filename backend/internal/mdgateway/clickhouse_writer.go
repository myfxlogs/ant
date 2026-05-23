// Package mdgateway — ClickHouse async batch writer with spill fallback.
package mdgateway

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// CHWriterConfig holds ClickHouse writer settings.
type CHWriterConfig struct {
	FlushInterval time.Duration // batch flush interval (default 1s)
	MaxBatchSize  int           // max rows per batch (default 1000)
	SpillWriter   *SpillWriter  // optional fallback for CH write failures
}

// DefaultCHWriterConfig returns sensible defaults.
func DefaultCHWriterConfig() CHWriterConfig {
	return CHWriterConfig{
		FlushInterval: time.Second,
		MaxBatchSize:  1000,
	}
}

// CHWriter buffers ticks and flushes them to ClickHouse in batches.
// When CH write fails and a SpillWriter is configured, ticks fall back
// to JSONL spill files for later replay.
type CHWriter struct {
	cfg     CHWriterConfig
	log     *zap.Logger
	chConn  *CHConn
	ticks   chan *Tick
	done    chan struct{}
	wg      sync.WaitGroup
	metrics *MDMetrics
}

// NewCHWriter creates a CHWriter backed by a real ClickHouse connection.
func NewCHWriter(cfg CHWriterConfig, chConn *CHConn, log *zap.Logger) *CHWriter {
	if cfg.FlushInterval == 0 {
		cfg.FlushInterval = time.Second
	}
	if cfg.MaxBatchSize == 0 {
		cfg.MaxBatchSize = 1000
	}
	return &CHWriter{
		cfg:    cfg,
		log:    log,
		chConn: chConn,
		ticks:  make(chan *Tick, cfg.MaxBatchSize*2),
		done:   make(chan struct{}),
	}
}

// SetMetrics attaches Prometheus metrics to the writer.
func (w *CHWriter) SetMetrics(m *MDMetrics) {
	w.metrics = m
}

// Start begins the async flush loop.
func (w *CHWriter) Start(ctx context.Context) {
	w.wg.Add(1)
	go w.loop(ctx)
}

// loop is the main flush loop.
func (w *CHWriter) loop(ctx context.Context) {
	defer w.wg.Done()

	batch := make([]*Tick, 0, w.cfg.MaxBatchSize)
	ticker := time.NewTicker(w.cfg.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.flushBatch(context.Background(), batch)
			return
		case <-w.done:
			w.flushBatch(context.Background(), batch)
			return
		case t := <-w.ticks:
			batch = append(batch, t)
			if len(batch) >= w.cfg.MaxBatchSize {
				w.flushBatch(ctx, batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				w.flushBatch(ctx, batch)
				batch = batch[:0]
			}
		}
	}
}

// Write enqueues a Tick for batch insertion. Non-blocking — drops on full channel.
func (w *CHWriter) Write(tick *Tick) {
	select {
	case w.ticks <- tick:
	default:
		w.log.Warn("chwriter: channel full, dropping tick",
			zap.String("symbol", tick.Symbol),
		)
	}
}

// flushBatch writes a batch of ticks to ClickHouse, spilling on failure.
func (w *CHWriter) flushBatch(ctx context.Context, batch []*Tick) {
	if len(batch) == 0 {
		return
	}

	if err := w.tryFlush(ctx, batch); err != nil {
		w.log.Error("chwriter: flush failed, spilling",
			zap.Int("rows", len(batch)),
			zap.Error(err),
		)
		if w.metrics != nil {
			w.metrics.CHWriteErrors.Inc()
		}
		if w.cfg.SpillWriter != nil {
			for _, t := range batch {
				if err := w.cfg.SpillWriter.Write(t); err != nil {
					w.log.Error("chwriter: spill write failed", zap.Error(err))
				} else if w.metrics != nil {
					w.metrics.SpillWrites.Inc()
				}
			}
		}
	}
}

func (w *CHWriter) tryFlush(ctx context.Context, batch []*Tick) error {
	conn, err := w.chConn.Conn(ctx)
	if err != nil {
		return err
	}

	insCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	chBatch, err := conn.PrepareBatch(insCtx, "INSERT INTO md_ticks")
	if err != nil {
		return err
	}

	for _, t := range batch {
		if err := chBatch.Append(
			t.UserID,
			t.Broker,
			t.Symbol,
			t.Canonical,
			uint64(t.TsUnixMs),
			uint64(t.ArrivedUnixMs),
			t.GetBid().GetValue(),
			t.GetAsk().GetValue(),
			t.BidVolume,
			t.AskVolume,
		); err != nil {
			return err
		}
	}

	return chBatch.Send()
}

// Close flushes remaining ticks and closes the writer.
func (w *CHWriter) Close() error {
	close(w.done)
	w.wg.Wait()
	return nil
}
