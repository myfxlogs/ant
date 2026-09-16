package strategy

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// BacktestRunsTotal counts backtest runs by status (started/completed/failed/canceled).
	BacktestRunsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ant_strategy_backtest_runs_total",
			Help: "Total number of backtest runs by status.",
		},
		[]string{"status"},
	)

	// BacktestDuration tracks backtest execution time in seconds.
	BacktestDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "ant_strategy_backtest_duration_seconds",
			Help:    "Backtest execution duration in seconds.",
			Buckets: []float64{0.5, 1, 2, 5, 10, 30, 60, 120, 300},
		},
	)

	// ExperimentRunsTotal counts strategy experiments by status.
	ExperimentRunsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ant_strategy_experiment_runs_total",
			Help: "Total number of strategy experiments by status.",
		},
		[]string{"status"},
	)

	// SSEConnectionsActive tracks active SSE watch connections.
	SSEConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "ant_strategy_sse_connections_active",
			Help: "Number of active SSE watch connections for backtest/experiment streams.",
		},
	)

	// VMEventDurationSeconds tracks single-event VM execution time in seconds,
	// labeled by event type (bar/tick/trade/trade_transaction/timer).
	// Live path only — backtest fires thousands of events/sec and histogram
	// cost itself would skew the measurement. QS-3-BASELINE.
	VMEventDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ant_strategy_vm_event_duration_seconds",
			Help:    "VM single-event execution duration in seconds (live path only).",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5},
		},
		[]string{"event"},
	)

	// VMEventInstructions tracks bytecode instruction count per event
	// (source: vm.ticks, read via Runner.EventStats immediately post-event).
	// Buckets span up to MaxTicks=1e7. QS-3-BASELINE.
	VMEventInstructions = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ant_strategy_vm_event_instructions",
			Help:    "VM bytecode instructions executed per event.",
			Buckets: []float64{10, 100, 1000, 10000, 100000, 1000000, 10000000},
		},
		[]string{"event"},
	)

	// VMFatalTotal counts events that ended with vm.fatalError set.
	// QS-3-BASELINE.
	VMFatalTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ant_strategy_vm_fatal_total",
			Help: "VM events ending with fatalError set, by event type.",
		},
		[]string{"event"},
	)
)
