package mdgateway

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"math"

	"go.uber.org/zap"

	"alphaforge/internal/mdgateway/adapter/mdtick"
)

// DLQWriter logs dropped ticks with reason-based sampling.
// ADR-0010 §2.2: parse_error=100%, bid_gt_ask/non_positive=1%.
// M10.5-10: writes are async via buffered channel + background goroutine.
// ADR-0012: writes structured log entries (no DB insertion).
type DLQWriter struct {
	log  *zap.Logger
	dlqQ chan dlqEntry // buffered async write queue
}

type dlqEntry struct {
	tick       *mdtick.Tick
	reason     string
	sampledPct float32
	rawPayload string
}

// NewDLQWriter creates a DLQ writer that logs to structured logger.
func NewDLQWriter(log *zap.Logger) *DLQWriter {
	d := &DLQWriter{
		log:  log,
		dlqQ: make(chan dlqEntry, 1000),
	}
	go d.flushLoop()
	return d
}

func (d *DLQWriter) flushLoop() {
	for entry := range d.dlqQ {
		d.writeTick(entry.tick, entry.reason, entry.sampledPct, entry.rawPayload)
	}
}

// WriteTick samples and logs a dropped tick.
// reason: "parse_error" | "bid_gt_ask" | "non_positive"
func (d *DLQWriter) WriteTick(ctx context.Context, t *mdtick.Tick, reason string, rawPayload string) {
	sampledPct := d.sampleRate(reason)
	if !d.shouldSample(sampledPct) {
		return
	}
	select {
	case d.dlqQ <- dlqEntry{tick: t, reason: reason, sampledPct: sampledPct, rawPayload: rawPayload}:
	default:
		d.log.Debug("dlq: queue full, dropping entry", zap.String("reason", reason))
	}
}

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
	var buf [4]byte
	_, _ = rand.Read(buf[:])
	v := float32(math.Mod(float64(binary.LittleEndian.Uint32(buf[:])), 100.0))
	return v < pct
}

func (d *DLQWriter) writeTick(t *mdtick.Tick, reason string, pct float32, raw string) {
	d.log.Warn("dlq: dropped tick",
		zap.String("broker", t.Broker),
		zap.String("canonical", t.Canonical),
		zap.String("reason", reason),
		zap.Float32("sampled_pct", pct),
		zap.String("bid", t.Bid.String()),
		zap.String("ask", t.Ask.String()),
	)
}
