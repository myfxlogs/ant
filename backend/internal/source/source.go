// Package source provides unified interfaces for bar and factor data consumption.
// LiveSource uses NATS JetStream; ReplaySource uses ClickHouse (backfill/backtest).
package source

import (
	"context"

	"anttrader/internal/mdgateway/adapter/mdtick"
)

// BarSource subscribes to OHLC bar data for a symbol/period pair.
type BarSource interface {
	Subscribe(ctx context.Context, canonical, period string) (<-chan *mdtick.Bar, error)
}

// FactorSource subscribes to factor values for a symbol.
type FactorSource interface {
	Subscribe(ctx context.Context, canonical string) (<-chan *FactorValue, error)
}

// FactorValue is a single factor output.
type FactorValue struct {
	Canonical string
	Name      string
	Value     float64
	BarTsMs   int64
}
