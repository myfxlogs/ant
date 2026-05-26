package mdgateway

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"go.uber.org/zap"

	"anttrader/internal/mdgateway/adapter/mdtick"
	"anttrader/internal/usermgr"
)

type CHWriterConfig struct {
	// M10 默认值（ADR-0011 §2.1）；env 可覆盖
	FlushInterval time.Duration // 默认 500ms（M7 旧值 1s）
	MaxBatchSize  int           // 默认 10000（M7 旧值 1000）
	QueueSize     int           // 默认 50000（M7 旧值 5000）；满则走 spill
}

func DefaultCHWriterConfig() CHWriterConfig {
	// M10 ADR-0011 §2.1: tuned for 100-account peak (25k tick/s).
	return CHWriterConfig{
		FlushInterval: 500 * time.Millisecond,
		MaxBatchSize:  10000,
		QueueSize:     50000,
	}
}

type CHWriter struct {
	cfg    CHWriterConfig
	conn   clickhouse.Conn
	log    *zap.Logger
	spill  *SpillWriter

	tickQ chan *mdtick.Tick
	barQ  chan *mdtick.Bar

	onSpillFail    func(brokerKey string, err error)
	spillFailCount int
	mu             sync.Mutex

	// S-2: dynamic buffer bypass for OOM auto-degradation.
	// Defaults to ANT_CH_BUFFER_ENABLED env (true if unset).
	bufferEnabled atomic.Bool

	// M10-BASE-A4: per-user CH write rate limiter (nil if not configured).
	userLimiter *usermgr.UserLimiter
}

func NewCHWriter(cfg CHWriterConfig, conn clickhouse.Conn, spill *SpillWriter, log *zap.Logger) *CHWriter {
	if cfg.QueueSize <= 0 { cfg.QueueSize = 5000 }
	w := &CHWriter{
		cfg:   cfg,
		conn:  conn,
		log:   log,
		spill: spill,
		tickQ: make(chan *mdtick.Tick, cfg.QueueSize),
		barQ:  make(chan *mdtick.Bar, cfg.QueueSize),
	}
	// S-2: init buffer bypass from env; default = buffer enabled.
	w.bufferEnabled.Store(os.Getenv("ANT_CH_BUFFER_ENABLED") != "false")
	return w
}

func (w *CHWriter) SetOnSpillFail(fn func(brokerKey string, err error)) {
	w.onSpillFail = fn
}

// SetUserLimiter injects the per-user CH write rate limiter (nil-safe).
func (w *CHWriter) SetUserLimiter(l *usermgr.UserLimiter) { w.userLimiter = l }

func (w *CHWriter) EnqueueTick(t *mdtick.Tick) {
	if w.userLimiter != nil && t.UserID != "" && !w.userLimiter.AllowCHWrite(t.UserID, 256) {
		RecordChanFull()
		return
	}
	select {
	case w.tickQ <- t:
	default:
		RecordChanFull()
		w.writeSpillTick(t)
	}
}

func (w *CHWriter) EnqueueBar(b *mdtick.Bar) {
	if w.userLimiter != nil && b.UserID != "" && !w.userLimiter.AllowCHWrite(b.UserID, 512) {
		RecordChanFull()
		return
	}
	select {
	case w.barQ <- b:
	default:
		RecordChanFull()
		w.writeSpillBar(b)
	}
}

func (w *CHWriter) Start(ctx context.Context) {
	ticker := Clk.NewTicker(w.cfg.FlushInterval)
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
		case <-ticker.C():
			w.flushTicks(ctx, tickBatch)
			w.flushBars(ctx, barBatch)
			tickBatch = tickBatch[:0]
			barBatch = barBatch[:0]
		}
	}
}

// Flush drains the given batches to CH. Called during graceful shutdown.
func (w *CHWriter) Flush(ctx context.Context, ticks []*mdtick.Tick, bars []*mdtick.Bar) {
	w.flushTicks(ctx, ticks)
	w.flushBars(ctx, bars)
}

func (w *CHWriter) flush(ctx context.Context, ticks []*mdtick.Tick, bars []*mdtick.Bar) {
	w.flushTicks(ctx, ticks)
	w.flushBars(ctx, bars)
}

func (w *CHWriter) flushTicks(ctx context.Context, batch []*mdtick.Tick) {
	if len(batch) == 0 { return }
	if err := w.insertTicks(ctx, batch); err != nil {
		w.log.Warn("chwriter: tick flush failed, spilling", zap.Int("count", len(batch)), zap.Error(err))
		for _, t := range batch {
			w.writeSpillTick(t)
		}
	}
}

func (w *CHWriter) flushBars(ctx context.Context, batch []*mdtick.Bar) {
	if len(batch) == 0 { return }
	if err := w.insertBars(ctx, batch); err != nil {
		w.log.Warn("chwriter: bar flush failed, spilling", zap.Int("count", len(batch)), zap.Error(err))
		for _, b := range batch {
			w.writeSpillBar(b)
		}
	}
}

// SetBufferEnabled toggles the buffer engine bypass at runtime.
// When false, INSERTs target md_ticks/md_bars directly (Buffer bypass).
// This is the S-2 auto-degradation hook: called by the memory monitor when
// CH Buffer engine memory pressure exceeds threshold.
func (w *CHWriter) SetBufferEnabled(enabled bool) {
	prev := w.bufferEnabled.Swap(enabled)
	if prev != enabled {
		w.log.Warn("chwriter: buffer bypass toggled",
			zap.Bool("enabled", enabled),
			zap.Bool("previous", prev))
	}
}

// BufferEnabled returns whether the Buffer engine is currently in use.
func (w *CHWriter) BufferEnabled() bool { return w.bufferEnabled.Load() }

// tickTargetTable returns the CH target table for tick INSERTs.
// ADR-0011: default = md_ticks_buffer (Buffer engine).
// M10.5-10: ANT_CH_BUFFER_ENABLED=false → direct-write md_ticks (Buffer bypass).
// S-2: dynamic toggle via SetBufferEnabled — no restart required.
func (w *CHWriter) tickTargetTable() string {
	if !w.bufferEnabled.Load() {
		return "md_ticks"
	}
	return "md_ticks_buffer"
}

// barTargetTable returns the CH target table for bar INSERTs (see tickTargetTable).
func (w *CHWriter) barTargetTable() string {
	if !w.bufferEnabled.Load() {
		return "md_bars"
	}
	return "md_bars_buffer"
}

func (w *CHWriter) insertTicks(ctx context.Context, ticks []*mdtick.Tick) error {
	targetTable := w.tickTargetTable()
	batch, err := w.conn.PrepareBatch(ctx,
		"INSERT INTO "+targetTable+" (user_id, account_id, broker, symbol_raw, canonical, ts_unix_ms, arrived_unix_ms, bid, ask, bid_volume, ask_volume, is_replay)")
	if err != nil { return err }
	defer batch.Abort()

	nowMs := Clk.Now().UnixMilli()
	for _, t := range ticks {
		replayBit := uint8(0)
		if t.IsReplay {
			replayBit = 1
		}
		if err := batch.Append(t.UserID, t.AccountID, t.Broker, t.SymbolRaw, t.Canonical,
			t.TsUnixMs, t.ArrivedUnixMs, t.Bid, t.Ask, t.BidVolume, t.AskVolume, replayBit,
		); err != nil {
			return fmt.Errorf("append tick to CH batch: %w", err)
		}
		// ADR-0010 §2.2: record e2e latency (mdgateway arrival → CH flush).
		ObserveE2eLatency(float64(nowMs-t.ArrivedUnixMs) / 1000.0)
	}
	return batch.Send()
}

func (w *CHWriter) insertBars(ctx context.Context, bars []*mdtick.Bar) error {
	// ADR-0008 §2.2 + ADR-0009 §2.2: close_ts_unix_ms is set from ArrivedUnixMs by bar_aggregator;
	// open_ts_unix_ms follows the same clock source for consistency.
	barsTarget := w.barTargetTable()
	batch, err := w.conn.PrepareBatch(ctx,
		"INSERT INTO "+barsTarget+" (user_id, account_id, broker, symbol_raw, canonical, period, open_ts_unix_ms, close_ts_unix_ms, open, high, low, close, volume, tick_count, is_replay)")
	if err != nil { return err }
	defer batch.Abort()

	for _, b := range bars {
		replayBit := uint8(0)
		if b.IsReplay {
			replayBit = 1
		}
		if err := batch.Append(b.UserID, b.AccountID, b.Broker, "", b.Canonical, b.Period,
			b.OpenTsUnixMs, b.CloseTsUnixMs, b.Open, b.High, b.Low, b.Close, b.Volume, b.TickCount, replayBit,
		); err != nil {
			return fmt.Errorf("append bar to CH batch: %w", err)
		}
	}
	return batch.Send()
}

func (w *CHWriter) writeSpillTick(t *mdtick.Tick) {
	if w.spill == nil { return }
	if err := w.spill.WriteTick(t); err != nil {
		w.spillFailed(t.Broker, err)
	}
}

func (w *CHWriter) writeSpillBar(b *mdtick.Bar) {
	if w.spill == nil { return }
	if err := w.spill.WriteBar(b); err != nil {
		w.spillFailed(b.Broker, err)
	}
}

func (w *CHWriter) spillFailed(broker string, err error) {
	w.mu.Lock()
	w.spillFailCount++
	count := w.spillFailCount
	w.mu.Unlock()

	if count >= 3 && w.onSpillFail != nil {
		w.onSpillFail(broker, err)
	}
}

var _ driver.Batch = nil
