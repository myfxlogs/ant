// Package mdgateway — Prometheus metrics for market data gateway.
package mdgateway

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MDMetrics holds all Prometheus metrics for the mdgateway subsystem.
type MDMetrics struct {
	TickTotal       *prometheus.CounterVec
	TickDropped     *prometheus.CounterVec
	TickOutlier     *prometheus.CounterVec
	BarFlushed      *prometheus.CounterVec
	CHWriteErrors   prometheus.Counter
	SpillWrites     prometheus.Counter
	CircuitState    prometheus.Gauge
	GatewayConnect  *prometheus.CounterVec
}

// NewMDMetrics registers and returns the mdgateway metrics.
func NewMDMetrics(reg prometheus.Registerer) *MDMetrics {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}
	factory := promauto.With(reg)
	return &MDMetrics{
		TickTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "md_tick_total",
			Help: "Total ticks received, by broker and symbol.",
		}, []string{"broker", "symbol"}),
		TickDropped: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "md_tick_dropped_total",
			Help: "Ticks dropped by quality checks, by broker and symbol.",
		}, []string{"broker", "symbol"}),
		TickOutlier: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "md_tick_outlier_total",
			Help: "Ticks flagged as price outliers.",
		}, []string{"broker", "symbol"}),
		BarFlushed: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "md_bar_flushed_total",
			Help: "Completed bars flushed, by period.",
		}, []string{"period"}),
		CHWriteErrors: factory.NewCounter(prometheus.CounterOpts{
			Name: "md_ch_write_errors_total",
			Help: "ClickHouse write errors.",
		}),
		SpillWrites: factory.NewCounter(prometheus.CounterOpts{
			Name: "md_spill_writes_total",
			Help: "Ticks written to spill files.",
		}),
		CircuitState: factory.NewGauge(prometheus.GaugeOpts{
			Name: "md_circuit_state",
			Help: "Circuit breaker state (0=closed, 1=open, 2=half-open).",
		}),
		GatewayConnect: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "md_gateway_connect_total",
			Help: "Gateway connection attempts, by platform and result.",
		}, []string{"platform", "result"}),
	}
}
