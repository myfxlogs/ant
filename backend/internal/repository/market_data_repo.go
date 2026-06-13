// market_data_repo.go — shared market data types and helpers.
// Implementation: PgMarketDataStore (market_data_pg.go).

package repository

import (
	"time"

	"go.uber.org/zap"
)

// KlineBar represents a single OHLCV bar.
type KlineBar struct {
	Broker        string
	Canonical     string
	Period        string
	OpenTsUnixMs  uint64
	CloseTsUnixMs uint64
	Open          float64
	High          float64
	Low           float64
	Close         float64
	Volume        float64
	TickCount     uint32
}

// LatestTick holds the latest bid/ask for a symbol.
type LatestTick struct {
	Bid    string
	Ask    string
	Broker string
}

// timeframeAliases maps MT-standard format to internal format.
// Internal format is canonical: 1m, 5m, 15m, 30m, 1h, 4h, 1d, 1w.
var timeframeAliases = map[string]string{
	"M1": "1m", "M5": "5m", "M15": "15m", "M30": "30m",
	"H1": "1h", "H4": "4h", "D1": "1d", "W1": "1w", "MN": "1mo",
}

// normalizeTimeframe converts MT-standard timeframe strings to internal format.
// Idempotent: passing an already-internal format is a no-op.
func normalizeTimeframe(period string) string {
	if mapped, ok := timeframeAliases[period]; ok {
		return mapped
	}
	return period
}

// scanKlineBars scans a row iterator into a slice of KlineBar.
// Used by both CH (legacy) and PG implementations.
func scanKlineBars(rows interface{ Scan(...interface{}) error; Next() bool; Err() error }, log *zap.Logger) ([]KlineBar, error) {
	var bars []KlineBar
	for rows.Next() {
		var b KlineBar
		if err := rows.Scan(&b.Broker, &b.Canonical, &b.Period, &b.OpenTsUnixMs, &b.CloseTsUnixMs,
			&b.Open, &b.High, &b.Low, &b.Close, &b.Volume, &b.TickCount); err != nil {
			if log != nil {
				log.Warn("scan kline bars: scan row failed", zap.Error(err))
			}
			continue
		}
		bars = append(bars, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if bars == nil {
		bars = []KlineBar{}
	}
	return bars, nil
}

// OpenTime converts a unix millisecond timestamp to time.Time.
func (b *KlineBar) OpenTime() time.Time {
	return time.UnixMilli(int64(b.OpenTsUnixMs))
}

// CloseTime converts the close unix millisecond timestamp to time.Time.
func (b *KlineBar) CloseTime() time.Time {
	return time.UnixMilli(int64(b.CloseTsUnixMs))
}
