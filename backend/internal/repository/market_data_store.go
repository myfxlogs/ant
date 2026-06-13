// market_data_store.go — interface for market data read/write operations.
// Provides a single abstraction over PostgreSQL (primary) and ClickHouse (legacy/deprecated).
//
// All market data — ticks, bars, klines — flows through this interface so the
// underlying storage can be swapped without touching consumers.

package repository

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// MarketDataStore is the unified interface for market data access.
// It covers all read and write paths previously scattered across
// MarketDataRepository, CHWriter, and raw ClickHouse queries.
type MarketDataStore interface {
	// ── Read paths ──

	// GetKlines returns OHLCV kline bars for a symbol and period.
	// Optionally filtered by broker and time range.
	// limit defaults to 500 if <= 0.
	GetKlines(ctx context.Context, canonical, broker, period string, from, to *time.Time, limit int32) ([]KlineBar, error)

	// GetLatestTick returns the most recent bid/ask for a symbol.
	// lookback is bounded to 24h by the implementation.
	GetLatestTick(ctx context.Context, canonical, broker string) (*LatestTick, error)

	// LoadFinalizedBars returns all existing close_ts values per (broker, canonical, period)
	// within the given lookback window. Used by mdgateway at startup for bar dedup.
	// Returns a map of FinalizedKey → []close_ts_unix_ms.
	LoadFinalizedBars(ctx context.Context, since time.Time) (map[FinalizedKey][]int64, error)

	// MaxCloseTs returns the latest close_ts_unix_ms for the given (broker, canonical, period).
	// Used by the backfiller to determine gap boundaries.
	// Returns 0 if no bars found.
	MaxCloseTs(ctx context.Context, broker, canonical, period string) (int64, error)

	// FetchActualReturn computes the 7-day price return after predicted_at.
	// Returns (closePrice - openPrice) / openPrice, or an error if no price data.
	FetchActualReturn(ctx context.Context, symbol string, predictedAt time.Time) (float64, error)

	// ── Write paths ──

	// InsertBars batch-inserts kline bars. Used by backfill and broker history sync.
	InsertBars(ctx context.Context, bars []KlineBar) error

	// InsertTicks batch-inserts raw ticks from the mdgateway pipeline.
	InsertTicks(ctx context.Context, ticks []TickRecord) error
}

// TickRecord represents a single tick for batch insertion.
// Bid/Ask use decimal.Decimal per project precision standard.
type TickRecord struct {
	UserID        string
	AccountID     string
	Broker        string
	SymbolRaw     string
	Canonical     string
	TsUnixMs      int64
	ArrivedUnixMs int64
	Bid           decimal.Decimal
	Ask           decimal.Decimal
	BidVolume     float64
	AskVolume     float64
	IsReplay      bool
}

// FinalizedKey uniquely identifies a bar series for dedup.
type FinalizedKey struct {
	Broker    string
	Canonical string
	Period    string
}
