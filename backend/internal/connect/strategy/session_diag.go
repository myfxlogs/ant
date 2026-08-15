// session_diag.go — Runtime diagnostics for active strategy sessions.
// L1 evaluation counters + L2 indicator ring buffer with 5s write throttling.
// Fine-grained locking: diag.mu is independent from session global locks.
package strategy

import (
	antv1 "alphaforge/gen/proto/ant/v1"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

const (
	maxIndicatorKeys  = 32
	indicatorRingCap  = 64
	diagWriteInterval = 5 * time.Second
)

// Eval kind constants for RecordEval.
const (
	evalKindBar   = 0
	evalKindTick  = 1
	evalKindTrade = 2
)

// diagToProto converts a DiagSnapshot to a proto StrategyDiagnostics message.
func diagToProto(s DiagSnapshot) *antv1.StrategyDiagnostics {
	pb := &antv1.StrategyDiagnostics{
		EvalCount:   s.EvalCount,
		TickCount:   s.TickCount,
		BarCount:    s.BarCount,
		LastEvalAt:  s.LastEvalAt,
		WindowBars:  int32(s.WindowBars),
		OrdersTotal: int32(s.OrdersTotalSeen),
	}
	if len(s.IndicatorOrder) > 0 {
		pb.Indicators = make([]*antv1.DiagIndicatorSeries, 0, len(s.IndicatorOrder))
		for _, key := range s.IndicatorOrder {
			vals := s.Indicators[key]
			strVals := make([]string, len(vals))
			for i, v := range vals {
				strVals[i] = v.String()
			}
			pb.Indicators = append(pb.Indicators, &antv1.DiagIndicatorSeries{
				Key:    key,
				Values: strVals,
			})
		}
	}
	return pb
}

// sessionDiag holds runtime diagnostics for a single active strategy run.
// Fields are protected by mu — never reuse session global locks (hot path).
type sessionDiag struct {
	mu                sync.Mutex
	evalCount         int64
	tickCount         int64
	barCount          int64
	lastEvalAt        int64 // unix ms
	windowBars        int
	ordersTotalSeen   int
	indicators        map[string][]decimal.Decimal
	indicatorKeyOrder []string
	lastWriteAt       int64 // unix ms, for 5s throttling
}

func newSessionDiag() *sessionDiag {
	return &sessionDiag{
		indicators: make(map[string][]decimal.Decimal),
	}
}

// RecordEval increments the evaluation counter for the given event kind.
func (d *sessionDiag) RecordEval(kind int) {
	d.mu.Lock()
	d.evalCount++
	switch kind {
	case evalKindBar:
		d.barCount++
	case evalKindTick:
		d.tickCount++
	}
	d.lastEvalAt = time.Now().UnixMilli()
	d.mu.Unlock()
}

// RecordWindow sets the server-side bar window length.
func (d *sessionDiag) RecordWindow(n int) {
	d.mu.Lock()
	d.windowBars = n
	d.mu.Unlock()
}

// RecordIndicators writes indicator values and ordersTotal to the ring buffer
// with 5s server-side throttling. VM records every event (zero cost);
// this method is called from vmHandleX tail to throttle ring buffer writes.
func (d *sessionDiag) RecordIndicators(values map[string]decimal.Decimal, ordersTotal int) {
	now := time.Now().UnixMilli()
	d.mu.Lock()
	defer d.mu.Unlock()
	if now-d.lastWriteAt < int64(diagWriteInterval/time.Millisecond) {
		return
	}
	d.lastWriteAt = now
	d.ordersTotalSeen = ordersTotal
	for key, val := range values {
		ring := d.indicators[key]
		if ring == nil {
			if len(d.indicators) >= maxIndicatorKeys {
				continue
			}
			d.indicatorKeyOrder = append(d.indicatorKeyOrder, key)
		}
		ring = append(ring, val)
		if len(ring) > indicatorRingCap {
			ring = ring[len(ring)-indicatorRingCap:]
		}
		d.indicators[key] = ring
	}
}

// DiagSnapshot is a point-in-time copy of diagnostics for proto conversion.
// Returned by SnapshotDiag — safe to use outside the lock.
type DiagSnapshot struct {
	EvalCount       int64
	TickCount       int64
	BarCount        int64
	LastEvalAt      int64
	WindowBars      int
	OrdersTotalSeen int
	Indicators      map[string][]decimal.Decimal
	IndicatorOrder  []string
}

// SnapshotDiag returns a copy of the current diagnostics state.
// The indicators map and slices are deep-copied to prevent lock leakage.
func (d *sessionDiag) SnapshotDiag() DiagSnapshot {
	d.mu.Lock()
	defer d.mu.Unlock()
	indicators := make(map[string][]decimal.Decimal, len(d.indicators))
	order := make([]string, len(d.indicatorKeyOrder))
	copy(order, d.indicatorKeyOrder)
	for k, v := range d.indicators {
		cp := make([]decimal.Decimal, len(v))
		copy(cp, v)
		indicators[k] = cp
	}
	return DiagSnapshot{
		EvalCount:       d.evalCount,
		TickCount:       d.tickCount,
		BarCount:        d.barCount,
		LastEvalAt:      d.lastEvalAt,
		WindowBars:      d.windowBars,
		OrdersTotalSeen: d.ordersTotalSeen,
		Indicators:      indicators,
		IndicatorOrder:  order,
	}
}
