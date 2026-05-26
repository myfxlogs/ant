package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

// MarketDataRepository provides read access to ClickHouse market data.
type MarketDataRepository struct {
	ch clickhouse.Conn
}

// NewMarketDataRepository creates a market data repository backed by ClickHouse.
func NewMarketDataRepository(ch clickhouse.Conn) *MarketDataRepository {
	return &MarketDataRepository{ch: ch}
}

// KlineBar represents a single OHLCV bar from ClickHouse.
type KlineBar struct {
	Canonical  string
	Broker     string
	TsUnixMs   int64
	Open       float64
	High       float64
	Low        float64
	Close      float64
	TickVolume int64
}

// GetKlines returns OHLCV kline bars for a symbol and period.
func (r *MarketDataRepository) GetKlines(ctx context.Context, canonical, period string, limit int32) ([]KlineBar, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := r.ch.Query(ctx,
		`SELECT canonical, broker, ts_unix_ms, open, high, low, close, tick_volume
		 FROM kline
		 WHERE canonical = $1 AND period = $2
		 ORDER BY ts_unix_ms DESC
		 LIMIT $3`,
		canonical, period, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("get klines: %w", err)
	}
	defer rows.Close()

	var bars []KlineBar
	for rows.Next() {
		var b KlineBar
		if err := rows.Scan(&b.Canonical, &b.Broker, &b.TsUnixMs, &b.Open, &b.High, &b.Low, &b.Close, &b.TickVolume); err != nil {
			continue
		}
		bars = append(bars, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get klines rows: %w", err)
	}
	if bars == nil {
		bars = []KlineBar{}
	}
	return bars, nil
}

// LatestTick holds the latest bid/ask for a symbol.
type LatestTick struct {
	Bid string
	Ask string
}

// GetLatestTick returns the most recent tick for a symbol.
func (r *MarketDataRepository) GetLatestTick(ctx context.Context, canonical string) (*LatestTick, error) {
	var t LatestTick
	err := r.ch.QueryRow(ctx,
		`SELECT bid, ask FROM tick_raw
		 WHERE canonical = $1
		 ORDER BY ts_unix_ms DESC
		 LIMIT 1`,
		canonical,
	).Scan(&t.Bid, &t.Ask)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// OpenTime converts a unix millisecond timestamp to time.Time.
func (b *KlineBar) OpenTime() time.Time {
	return time.UnixMilli(b.TsUnixMs)
}
