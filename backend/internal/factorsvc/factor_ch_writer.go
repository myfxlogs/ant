package factorsvc

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// FactorCHWriterConfig holds ClickHouse writer settings for factor values.
type FactorCHWriterConfig struct {
	FlushInterval time.Duration
	MaxBatchSize  int
}

// DefaultFactorCHWriterConfig returns sensible defaults.
func DefaultFactorCHWriterConfig() FactorCHWriterConfig {
	return FactorCHWriterConfig{
		FlushInterval: 5 * time.Second,
		MaxBatchSize:  500,
	}
}

// CHConn is a placeholder interface for ClickHouse connection.
// Replace with clickhouse.Conn when the CH driver is available.
type CHConn interface{}

// FactorCHWriter buffers factor values and flushes them to ClickHouse.
// Async batch insert — currently stubbed with a no-op CH connection.
type FactorCHWriter struct {
	cfg  FactorCHWriterConfig
	log  *zap.Logger
	ch   chan factorRow
	conn CHConn
	done chan struct{}
	wg   sync.WaitGroup
}

type factorRow struct {
	UserID string
	Factor   string
	Symbol   string
	TS       int64
	Value    float64
}

// NewFactorCHWriter creates a FactorCHWriter.
func NewFactorCHWriter(cfg FactorCHWriterConfig, log *zap.Logger) *FactorCHWriter {
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 5 * time.Second
	}
	if cfg.MaxBatchSize == 0 {
		cfg.MaxBatchSize = 500
	}
	return &FactorCHWriter{
		cfg:  cfg,
		log:  log,
		ch:   make(chan factorRow, cfg.MaxBatchSize*2),
		done: make(chan struct{}),
	}
}

// WithConn sets the ClickHouse connection (stub — accepts interface{}).
func (w *FactorCHWriter) WithConn(conn CHConn) *FactorCHWriter {
	w.conn = conn
	return w
}

// Start begins the async flush loop.
func (w *FactorCHWriter) Start(ctx context.Context) {
	w.wg.Add(1)
	go w.loop(ctx)
}

func (w *FactorCHWriter) loop(ctx context.Context) {
	defer w.wg.Done()

	batch := make([]factorRow, 0, w.cfg.MaxBatchSize)
	ticker := time.NewTicker(w.cfg.FlushInterval)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		if w.conn == nil {
			w.log.Debug("factor_ch_writer: flush skipped (no CH conn)",
				zap.Int("rows", len(batch)),
			)
			batch = batch[:0]
			return
		}
		w.log.Info("factor_ch_writer: flushed to CH (stub)",
			zap.Int("rows", len(batch)),
		)
		batch = batch[:0]
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case <-w.done:
			flush()
			return
		case r := <-w.ch:
			batch = append(batch, r)
			if len(batch) >= w.cfg.MaxBatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

// Write enqueues a factor value for batch insertion.
func (w *FactorCHWriter) Write(ctx context.Context, userID, factor, symbol string, tsMs int64, value float64) {
	select {
	case w.ch <- factorRow{UserID: userID, Factor: factor, Symbol: symbol, TS: tsMs, Value: value}:
	default:
		w.log.Warn("factor_ch_writer: channel full, dropping value",
			zap.String("factor", factor), zap.String("symbol", symbol),
		)
	}
}

// Close flushes remaining values and stops the writer.
func (w *FactorCHWriter) Close() error {
	close(w.done)
	w.wg.Wait()
	return nil
}
