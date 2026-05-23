package factorsvc

// Metrics is a stub for factor evaluation metrics.
// The original implementation used Prometheus; replace with the ant metrics
// package when the observability layer is defined.
type FactorMetrics struct {
	EvalCount   int64
	EvalErrors  int64
	BufferCount int64
}

// NewFactorMetrics returns a stub metrics collector.
func NewFactorMetrics() *FactorMetrics {
	return &FactorMetrics{}
}

// RecordEval increments the evaluation counter.
func (m *FactorMetrics) RecordEval() {
	m.EvalCount++
}

// RecordError increments the error counter.
func (m *FactorMetrics) RecordError() {
	m.EvalErrors++
}

// RecordBuffer records the current buffer count.
func (m *FactorMetrics) RecordBuffer(n int64) {
	m.BufferCount = n
}
