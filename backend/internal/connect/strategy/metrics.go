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
)
