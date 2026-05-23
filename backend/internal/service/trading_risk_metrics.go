package service

import (
	"fmt"
	"sync"
	"time"
)

type tradeRiskMetrics struct {
	mu          sync.Mutex
	counters    map[string]int64
	latencyData map[string]latencyMetric
}

type latencyMetric struct {
	Count int64
	Total time.Duration
	Max   time.Duration
	Min   time.Duration
}

var globalTradeRiskMetrics = &tradeRiskMetrics{
	counters:    make(map[string]int64),
	latencyData: make(map[string]latencyMetric),
}

func recordRiskValidateMetric(result, code, platform, triggerSource string, latency time.Duration) {
	key := fmt.Sprintf("risk_validate_total|result=%s|code=%s|platform=%s|trigger_source=%s", result, code, platform, triggerSource)
	globalTradeRiskMetrics.incrementCounter(key)
	globalTradeRiskMetrics.recordLatency("risk_validate_latency_ms", latency)
}

func recordOrderSendMetric(result, code string) {
	key := fmt.Sprintf("order_send_total|result=%s|code=%s", result, code)
	globalTradeRiskMetrics.incrementCounter(key)
}

func recordOrderCloseMetric(result, code string) {
	key := fmt.Sprintf("order_close_total|result=%s|code=%s", result, code)
	globalTradeRiskMetrics.incrementCounter(key)
}

func (m *tradeRiskMetrics) incrementCounter(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[key]++
}

func (m *tradeRiskMetrics) recordLatency(name string, d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	lm := m.latencyData[name]
	lm.Count++
	lm.Total += d
	if lm.Max == 0 || d > lm.Max {
		lm.Max = d
	}
	if lm.Min == 0 || d < lm.Min {
		lm.Min = d
	}
	m.latencyData[name] = lm
}

func SnapshotTradeRiskMetrics() map[string]interface{} {
	globalTradeRiskMetrics.mu.Lock()
	defer globalTradeRiskMetrics.mu.Unlock()
	out := map[string]interface{}{
		"counters": make(map[string]int64),
		"latency":  make(map[string]map[string]interface{}),
	}
	counters := out["counters"].(map[string]int64)
	for k, v := range globalTradeRiskMetrics.counters {
		counters[k] = v
	}
	latency := out["latency"].(map[string]map[string]interface{})
	for k, v := range globalTradeRiskMetrics.latencyData {
		avg := time.Duration(0)
		if v.Count > 0 {
			avg = v.Total / time.Duration(v.Count)
		}
		latency[k] = map[string]interface{}{
			"count": v.Count,
			"avg":   avg.Milliseconds(),
			"max":   v.Max.Milliseconds(),
			"min":   v.Min.Milliseconds(),
		}
	}
	return out
}
