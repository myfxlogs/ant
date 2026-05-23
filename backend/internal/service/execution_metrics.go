package service

import (
	"sync"
	"sync/atomic"
)

type ExecutionMetrics struct {
	totalCalls       atomic.Int64
	successCalls     atomic.Int64
	engineErrors     atomic.Int64
	timeouts         atomic.Int64
	riskRejections   atomic.Int64
	limitExceeded    atomic.Int64
	circuitOpen      atomic.Int64
	idempotencyHits  atomic.Int64
	idempotencyMiss  atomic.Int64

	latencyMu     sync.Mutex
	latencySumMs  int64
	latencyCount  int64
	latencyMaxMs  int64
}

func NewExecutionMetrics() *ExecutionMetrics {
	return &ExecutionMetrics{}
}

func (m *ExecutionMetrics) RecordCall()            { m.totalCalls.Add(1) }
func (m *ExecutionMetrics) RecordSuccess()         { m.successCalls.Add(1) }
func (m *ExecutionMetrics) RecordEngineError()     { m.engineErrors.Add(1) }
func (m *ExecutionMetrics) RecordTimeout()         { m.timeouts.Add(1) }
func (m *ExecutionMetrics) RecordRiskRejection()   { m.riskRejections.Add(1) }
func (m *ExecutionMetrics) RecordLimitExceeded()   { m.limitExceeded.Add(1) }
func (m *ExecutionMetrics) RecordCircuitOpen()     { m.circuitOpen.Add(1) }
func (m *ExecutionMetrics) RecordIdempotencyHit()  { m.idempotencyHits.Add(1) }
func (m *ExecutionMetrics) RecordIdempotencyMiss() { m.idempotencyMiss.Add(1) }

func (m *ExecutionMetrics) RecordLatency(ms int64) {
	m.latencyMu.Lock()
	m.latencySumMs += ms
	m.latencyCount++
	if ms > m.latencyMaxMs {
		m.latencyMaxMs = ms
	}
	m.latencyMu.Unlock()
}

type ExecutionMetricsSnapshot struct {
	TotalCalls      int64   `json:"total_calls"`
	SuccessCalls    int64   `json:"success_calls"`
	EngineErrors    int64   `json:"engine_errors"`
	Timeouts        int64   `json:"timeouts"`
	RiskRejections  int64   `json:"risk_rejections"`
	LimitExceeded   int64   `json:"limit_exceeded"`
	CircuitOpen     int64   `json:"circuit_open"`
	IdempotencyHits int64   `json:"idempotency_hits"`
	IdempotencyMiss int64   `json:"idempotency_miss"`
	AvgLatencyMs    float64 `json:"avg_latency_ms"`
	MaxLatencyMs    int64   `json:"max_latency_ms"`
}

func (m *ExecutionMetrics) Snapshot() ExecutionMetricsSnapshot {
	m.latencyMu.Lock()
	var avg float64
	if m.latencyCount > 0 {
		avg = float64(m.latencySumMs) / float64(m.latencyCount)
	}
	maxMs := m.latencyMaxMs
	m.latencyMu.Unlock()

	return ExecutionMetricsSnapshot{
		TotalCalls:      m.totalCalls.Load(),
		SuccessCalls:    m.successCalls.Load(),
		EngineErrors:    m.engineErrors.Load(),
		Timeouts:        m.timeouts.Load(),
		RiskRejections:  m.riskRejections.Load(),
		LimitExceeded:   m.limitExceeded.Load(),
		CircuitOpen:     m.circuitOpen.Load(),
		IdempotencyHits: m.idempotencyHits.Load(),
		IdempotencyMiss: m.idempotencyMiss.Load(),
		AvgLatencyMs:    avg,
		MaxLatencyMs:    maxMs,
	}
}

func (m *ExecutionMetrics) Reset() {
	m.totalCalls.Store(0)
	m.successCalls.Store(0)
	m.engineErrors.Store(0)
	m.timeouts.Store(0)
	m.riskRejections.Store(0)
	m.limitExceeded.Store(0)
	m.circuitOpen.Store(0)
	m.idempotencyHits.Store(0)
	m.idempotencyMiss.Store(0)
	m.latencyMu.Lock()
	m.latencySumMs = 0
	m.latencyCount = 0
	m.latencyMaxMs = 0
	m.latencyMu.Unlock()
}
