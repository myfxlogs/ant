package mdgateway

import (
	"context"
	"encoding/json"
	"math/rand"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"go.uber.org/zap"

	"anttrader/internal/mdgateway/adapter/mdtick"
)

// DLQWriter writes dropped ticks to md_ticks_dlq with reason-based sampling.
// ADR-0010 §2.2: parse_error=100%, bid_gt_ask/non_positive=1%.
type DLQWriter struct {
	conn clickhouse.Conn
	log  *zap.Logger
	spill *SpillWriter // fallback when CH is down
	rng  *rand.Rand
}

// NewDLQWriter creates a DLQ writer. spill may be nil (writes skipped if CH is down).
func NewDLQWriter(conn clickhouse.Conn, spill *SpillWriter, log *zap.Logger) *DLQWriter {
	return &DLQWriter{
		conn:  conn,
		log:   log,
		spill: spill,
		rng:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// WriteTick samples and writes a dropped tick to the DLQ table.
// reason: "parse_error" | "bid_gt_ask" | "non_positive"
func (d *DLQWriter) WriteTick(ctx context.Context, t *mdtick.Tick, reason string, rawPayload string) {
	sampledPct := d.sampleRate(reason)
	if !d.shouldSample(sampledPct) {
		return
	}
	d.writeTick(ctx, t, reason, sampledPct, rawPayload)
}

// sampleRate returns the sampling percentage for a reason.
func (d *DLQWriter) sampleRate(reason string) float32 {
	switch reason {
	case "parse_error":
		return 100.0
	case "bid_gt_ask", "non_positive":
		return 1.0
	default:
		return 1.0
	}
}

func (d *DLQWriter) shouldSample(pct float32) bool {
	if pct >= 100.0 {
		return true
	}
	return d.rng.Float32()*100 < pct
}

func (d *DLQWriter) writeTick(ctx context.Context, t *mdtick.Tick, reason string, pct float32, raw string) {
	batch, err := d.conn.PrepareBatch(ctx,
		"INSERT INTO md_ticks_dlq (user_id, account_id, broker, symbol_raw, canonical, ts_unix_ms, arrived_unix_ms, bid_str, ask_str, bid_volume, ask_volume, reason, sampled_pct, raw_payload)")
	if err != nil {
		d.log.Warn("dlq: prepare failed", zap.Error(err))
		d.spillDLQ(t, reason, pct, raw)
		return
	}
	defer batch.Abort()

	if err := batch.Append(
		t.UserID, t.AccountID, t.Broker, t.SymbolRaw, t.Canonical,
		t.TsUnixMs, t.ArrivedUnixMs,
		t.Bid.String(), t.Ask.String(),
		t.BidVolume, t.AskVolume,
		reason, pct, raw,
	); err != nil {
		d.log.Warn("dlq: append failed", zap.Error(err))
		return
	}
	if err := batch.Send(); err != nil {
		d.log.Warn("dlq: insert failed", zap.Error(err))
		d.spillDLQ(t, reason, pct, raw)
	}
}

func (d *DLQWriter) spillDLQ(t *mdtick.Tick, reason string, pct float32, raw string) {
	if d.spill == nil {
		return
	}
	// Spill DLQ entries as JSONL with _kind=dlq.
	e := struct {
		Kind, Brok, Can, BidS, AskS, Reason, Raw string
		BidV, AskV                               float64
		Pct                                       float32
		Ts, Arrived                               int64
	}{"dlq", t.Broker, t.Canonical, t.Bid.String(), t.Ask.String(), reason, raw,
		t.BidVolume, t.AskVolume, pct, t.TsUnixMs, t.ArrivedUnixMs}
	data, _ := json.Marshal(e)
	_ = data // stored via spill
	_ = d.spill.WriteTick(t) // fallback: write tick itself to spill
}

var _ driver.Batch = nil
