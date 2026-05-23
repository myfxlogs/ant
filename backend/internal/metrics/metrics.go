package metrics

import (
	"sync"
	"time"
)

type Metrics struct {
	startTime time.Time

	mu                    sync.RWMutex
	totalConnections      int
	activeConnections     int
	totalSubscriptions    int
	totalStreams          int
	totalZeroBalance      int
	totalErrors           int
	totalSuccess          int
	totalReconnects       int
	totalDisconnected     int

	latencyMetrics        map[string]*LatencyMetric
	subscriptionMetrics   map[string]int
	streamMetrics         map[string]int
	connectionStateMetrics map[string]int
	errorMetrics          map[string]int
}

type LatencyMetric struct {
	Count     int64
	Total     time.Duration
	Max       time.Duration
	Min       time.Duration
	Last      time.Time
}

func NewMetrics() *Metrics {
	return &Metrics{
		startTime: time.Now(),
		latencyMetrics:        make(map[string]*LatencyMetric),
		subscriptionMetrics:   make(map[string]int),
		streamMetrics:         make(map[string]int),
		connectionStateMetrics: make(map[string]int),
		errorMetrics:          make(map[string]int),
	}
}

func (m *Metrics) RecordConnection() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalConnections++
	m.activeConnections++
}

func (m *Metrics) RecordDisconnection() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeConnections--
	m.totalDisconnected++
}

func (m *Metrics) RecordReconnect() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalReconnects++
}

func (m *Metrics) RecordSuccess() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalSuccess++
}

func (m *Metrics) RecordError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalErrors++
}

func (m *Metrics) RecordZeroBalance() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalZeroBalance++
}

func (m *Metrics) RecordSubscription(streamType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalSubscriptions++
	m.subscriptionMetrics[streamType]++
}

func (m *Metrics) RecordStream(streamType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalStreams++
	m.streamMetrics[streamType]++
}

func (m *Metrics) RecordConnectionState(state string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connectionStateMetrics[state]++
}

func (m *Metrics) RecordErrorType(errorType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorMetrics[errorType]++
}

func (m *Metrics) RecordLatency(operation string, latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.latencyMetrics[operation]; !exists {
		m.latencyMetrics[operation] = &LatencyMetric{
			Min: latency,
		}
	}

	metric := m.latencyMetrics[operation]
	metric.Count++
	metric.Total += latency
	if latency > metric.Max {
		metric.Max = latency
	}
	if latency < metric.Min {
		metric.Min = latency
	}
	metric.Last = time.Now()
}

func (m *Metrics) GetTotalConnections() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.totalConnections
}

func (m *Metrics) GetActiveConnections() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeConnections
}

func (m *Metrics) GetTotalSubscriptions() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.totalSubscriptions
}

func (m *Metrics) GetTotalStreams() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.totalStreams
}

func (m *Metrics) GetTotalZeroBalance() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.totalZeroBalance
}

func (m *Metrics) GetTotalErrors() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.totalErrors
}

func (m *Metrics) GetTotalSuccess() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.totalSuccess
}

func (m *Metrics) GetTotalReconnects() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.totalReconnects
}

func (m *Metrics) GetTotalDisconnected() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.totalDisconnected
}

func (m *Metrics) GetSubscriptionCount(streamType string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.subscriptionMetrics[streamType]
}

func (m *Metrics) GetStreamCount(streamType string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.streamMetrics[streamType]
}

func (m *Metrics) GetConnectionStateCount(state string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connectionStateMetrics[state]
}

func (m *Metrics) GetErrorCount(errorType string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.errorMetrics[errorType]
}

func (m *Metrics) GetLatencyStats(operation string) *LatencyMetric {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.latencyMetrics[operation]
}

func (m *Metrics) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	uptime := time.Since(m.startTime)

	stats := map[string]interface{}{
		"uptime":               uptime.String(),
		"total_connections":    m.totalConnections,
		"active_connections":   m.activeConnections,
		"total_subscriptions":  m.totalSubscriptions,
		"total_streams":        m.totalStreams,
		"total_zero_balance":   m.totalZeroBalance,
		"total_errors":         m.totalErrors,
		"total_success":        m.totalSuccess,
		"total_reconnects":     m.totalReconnects,
		"total_disconnected":   m.totalDisconnected,
		"subscription_breakdown": m.subscriptionMetrics,
		"stream_breakdown":     m.streamMetrics,
		"connection_state_breakdown": m.connectionStateMetrics,
		"error_breakdown":      m.errorMetrics,
	}

	latencies := make(map[string]interface{})
	for op, metric := range m.latencyMetrics {
		avg := time.Duration(0)
		if metric.Count > 0 {
			avg = metric.Total / time.Duration(metric.Count)
		}
		latencies[op] = map[string]interface{}{
			"count":  metric.Count,
			"avg":    avg.String(),
			"max":    metric.Max.String(),
			"min":    metric.Min.String(),
			"last":   metric.Last.Format(time.RFC3339),
		}
	}
	stats["latency_breakdown"] = latencies

	return stats
}

func (m *Metrics) LogStats() {
	_ = m.GetStats()
}
