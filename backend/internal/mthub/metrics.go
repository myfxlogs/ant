package mthub

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Order status label values for mthub_orders_placed_total.
const (
	orderStatusOK       = "ok"
	orderStatusRejected = "rejected"
	orderStatusErr      = "err"
)

// Money-path metrics for mthub — implements 15-observability §3.2.
// REUSE: promauto pattern from strategy/metrics.go.

var (
	// OrdersPlacedTotal counts orders placed by broker and outcome status.
	// status ∈ {ok, rejected, err}
	OrdersPlacedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mthub_orders_placed_total",
			Help: "Total orders placed by broker and status (ok/rejected/err).",
		},
		[]string{"broker", "status"},
	)

	// PlaceLatencySeconds tracks broker order placement latency.
	PlaceLatencySeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mthub_place_latency_seconds",
			Help:    "Order placement latency from submit to broker ack.",
			Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5},
		},
		[]string{"broker"},
	)

	// SessionActive tracks active MT sessions by account and broker.
	SessionActive = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mthub_session_active",
			Help: "Active MT sessions (1=connected, 0=disconnected).",
		},
		[]string{"account_id", "broker"},
	)

	// EventPublishedTotal counts trade events published by event type.
	EventPublishedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mthub_event_published_total",
			Help: "Total trade events published by event type.",
		},
		[]string{"event_type"},
	)
)
