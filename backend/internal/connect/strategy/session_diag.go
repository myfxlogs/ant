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
		// L3: Order truth
		VmOrdersTotal:      int32(s.VmOrdersTotal),
		BrokerAccountOrders: int32(s.BrokerAccountOrders),
		StrategyMagicOrders: int32(s.StrategyMagicOrders),
		PendingBrokerOrders: int32(s.PendingBrokerOrders),
		ScheduleMagic:       s.ScheduleMagic,
		// L3: Execution state
		ExecutionState:   s.ExecutionState,
		OrderLifecycle:   s.OrderLifecycle,
		LastBrokerTicket: s.LastBrokerTicket,
		// L3: Financial freshness
		FinancialSource:     s.FinancialSource,
		FinancialCapturedAt: s.FinancialCapturedAt,
		FinancialAgeMs:      s.FinancialAgeMs,
		FinancialFresh:      s.FinancialFresh,
		// L3: Positions freshness
		PositionsSource:     s.PositionsSource,
		PositionsCapturedAt: s.PositionsCapturedAt,
		PositionsAgeMs:      s.PositionsAgeMs,
		PositionsFresh:      s.PositionsFresh,
		// L3: Data availability
		DataAvailable: s.DataAvailable,
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

	// LIVE-DIAG-TRUTH-1 rework: persisted lifecycle + ticket.
	// Barrier state is transient — Release() clears it to idle/ticket=0.
	// These fields persist the last known lifecycle so diagnostics
	// continue to show "order_confirmed" etc. after Release().
	lastLifecycle     string
	lastBrokerTicket  int64
}

func newSessionDiag() *sessionDiag {
	return &sessionDiag{
		indicators:    make(map[string][]decimal.Decimal),
		lastLifecycle: "signal_generated",
	}
}

// RecordLifecycle persists the last order lifecycle stage and broker ticket.
// Called from logOrderLifecycle at every transition point (before Release).
func (d *sessionDiag) RecordLifecycle(lifecycle string, ticket int64) {
	d.mu.Lock()
	d.lastLifecycle = lifecycle
	d.lastBrokerTicket = ticket
	d.mu.Unlock()
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
//
// LIVE-DIAG-TRUTH-1 fix: Empty indicator values must NOT block ordersTotal
// update. OnTick-only strategies produce empty indicator maps on bar events,
// but OrdersTotal is still valid and must be recorded. The throttle window
// is only burned when there are actual indicator values to write — empty
// calls update ordersTotal without consuming the throttle slot.
func (d *sessionDiag) RecordIndicators(values map[string]decimal.Decimal, ordersTotal int) {
	now := time.Now().UnixMilli()
	d.mu.Lock()
	defer d.mu.Unlock()

	// Always update ordersTotal — it's valid even when indicators are empty.
	d.ordersTotalSeen = ordersTotal

	if len(values) == 0 {
		return // No indicator values to write; don't burn throttle window.
	}

	if now-d.lastWriteAt < int64(diagWriteInterval/time.Millisecond) {
		return // Throttled — indicator ring buffer write skipped.
	}
	d.lastWriteAt = now
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

	// L3: Order truth — computed at snapshot time from PositionCache + barrier.
	VmOrdersTotal       int
	BrokerAccountOrders int
	StrategyMagicOrders int
	PendingBrokerOrders int
	ScheduleMagic       int32

	// L3: Execution state
	ExecutionState   string
	OrderLifecycle   string
	LastBrokerTicket int64

	// L3: Financial freshness
	FinancialSource     string
	FinancialCapturedAt int64
	FinancialAgeMs      int64
	FinancialFresh      bool

	// L3: Positions freshness
	PositionsSource     string
	PositionsCapturedAt int64
	PositionsAgeMs      int64
	PositionsFresh      bool

	// L3: Data availability — distinguishes "no broker data source"
	// (paper mode / not wired) from "stale data". When false, freshness
	// fields are meaningless and frontend shows N/A instead of Stale.
	DataAvailable bool
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
		// Persisted lifecycle (survives Release)
		OrderLifecycle:   d.lastLifecycle,
		LastBrokerTicket: d.lastBrokerTicket,
	}
}
