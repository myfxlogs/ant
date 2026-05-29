package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"go.uber.org/zap"
)

// MarketDataRepository provides read access to ClickHouse market data.
type MarketDataRepository struct {
	ch  clickhouse.Conn
	log *zap.Logger
}

// NewMarketDataRepository creates a market data repository backed by ClickHouse.
func NewMarketDataRepository(ch clickhouse.Conn, log *zap.Logger) *MarketDataRepository {
	return &MarketDataRepository{ch: ch, log: log}
}

// KlineBar represents a single OHLCV bar from ClickHouse.
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

// canonicalVariants expands a clean canonical symbol into the set of
// suffixed variants we may see in legacy ClickHouse rows that were ingested
// before the Normalizer was wired in. This lets read queries cover both
// post-fix data (canonical = "BTCUSD") and legacy data (canonical = "BTCUSDM",
// "BTCUSD.m", "BTCUSDpro", etc.) without expensive ALTER TABLE migrations
// — `canonical` is part of the ORDER BY key and therefore not UPDATEable.
//
// Suffix list is kept in sync with stripSuffix() in mdgateway/normalizer.go.
func canonicalVariants(canon string) []string {
	if canon == "" {
		return nil
	}
	// Note: deliberately exclude "T" / "R" / "_i" / "_r" variants here even
	// though stripSuffix() strips them, because BTCUSDT, EURJPY_R etc are
	// real, distinct symbols. Read-side variant expansion must be conservative
	// to avoid bleeding unrelated symbols into a user's chart.
	suffixes := []string{"M", "m", "Pro", "pro", "X", "x", "C", "c"}
	dotSuffixes := []string{"m", "pro", "x", "c"}
	out := make([]string, 0, 1+len(suffixes)+len(dotSuffixes))
	out = append(out, canon)
	for _, s := range suffixes {
		out = append(out, canon+s)
	}
	for _, s := range dotSuffixes {
		out = append(out, canon+"."+s)
	}
	return out
}

// GetKlines returns OHLCV kline bars for a symbol and period, optionally filtered by broker.
func (r *MarketDataRepository) GetKlines(ctx context.Context, canonical, broker, period string, limit int32) ([]KlineBar, error) {
	if limit <= 0 {
		limit = 500
	}

	// Performance: avoid the FINAL modifier on a wide ReplacingMergeTree —
	// it forces a full part merge before sort/limit, blowing up memory on
	// busy symbols. Instead:
	//   1. narrow scan to the last 6 partitions via close_ts_unix_ms cutoff
	//      (table is partitioned by toYYYYMM(close_ts_unix_ms/1000));
	//   2. dedup with LIMIT 1 BY on the natural primary key, which CH handles
	//      streaming and bounded.
	// 6 months covers any reasonable (limit × period) request — 500 bars even
	// at H4 cadence is < 3 months.
	const lookbackMonths = 6
	cutoffMs := time.Now().AddDate(0, -lookbackMonths, 0).UnixMilli()

	variants := canonicalVariants(canonical)
	// toFloat64() casts: open/high/low/close are Decimal(18,6) in CH and the
	// clickhouse-go driver cannot scan Decimal directly into *float64.
	query := `SELECT broker, canonical, period, open_ts_unix_ms, close_ts_unix_ms,
	                 toFloat64(open), toFloat64(high), toFloat64(low), toFloat64(close),
	                 volume, tick_count
	          FROM md_bars
	          WHERE canonical IN $1 AND period = $2 AND is_replay = 0
	            AND close_ts_unix_ms >= $3`
	args := []any{variants, period, cutoffMs}
	if broker != "" {
		query += ` AND broker = $4
		           ORDER BY close_ts_unix_ms DESC
		           LIMIT 1 BY (broker, canonical, period, close_ts_unix_ms)
		           LIMIT $5`
		args = append(args, broker, limit)
	} else {
		query += ` ORDER BY close_ts_unix_ms DESC
		           LIMIT 1 BY (broker, canonical, period, close_ts_unix_ms)
		           LIMIT $4`
		args = append(args, limit)
	}

	rows, err := r.ch.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get klines: %w", err)
	}
	defer rows.Close()

	var bars []KlineBar
	for rows.Next() {
		var b KlineBar
		if err := rows.Scan(&b.Broker, &b.Canonical, &b.Period, &b.OpenTsUnixMs, &b.CloseTsUnixMs,
			&b.Open, &b.High, &b.Low, &b.Close, &b.Volume, &b.TickCount); err != nil {
			if r.log != nil {
				r.log.Warn("get klines: scan row failed", zap.Error(err))
			}
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
	Bid    string
	Ask    string
	Broker string
}

// GetLatestTick returns the most recent tick for a symbol, optionally filtered by broker.
//
// Avoids FINAL + ORDER BY DESC LIMIT 1 on the wide ReplacingMergeTree — that
// pattern forces a full part merge in memory and OOMs on busy symbols. We use
// argMax(field, ts) which streams in bounded memory, scoped to a 1-day window
// via the partition key (md_ticks is partitioned monthly on arrived_unix_ms).
func (r *MarketDataRepository) GetLatestTick(ctx context.Context, canonical, broker string) (*LatestTick, error) {
	const lookbackHours = 24
	cutoffMs := time.Now().Add(-lookbackHours * time.Hour).UnixMilli()
	variants := canonicalVariants(canonical)

	var t LatestTick
	var bidF, askF float64
	var brokerOut string
	var err error
	if broker != "" {
		err = r.ch.QueryRow(ctx,
			`SELECT toFloat64(argMax(bid, ts_unix_ms)),
			        toFloat64(argMax(ask, ts_unix_ms)),
			        argMax(broker, ts_unix_ms)
			 FROM md_ticks
			 WHERE canonical IN $1 AND broker = $2 AND is_replay = 0
			   AND arrived_unix_ms >= $3`,
			variants, broker, cutoffMs,
		).Scan(&bidF, &askF, &brokerOut)
	} else {
		err = r.ch.QueryRow(ctx,
			`SELECT toFloat64(argMax(bid, ts_unix_ms)),
			        toFloat64(argMax(ask, ts_unix_ms)),
			        argMax(broker, ts_unix_ms)
			 FROM md_ticks
			 WHERE canonical IN $1 AND is_replay = 0
			   AND arrived_unix_ms >= $2`,
			variants, cutoffMs,
		).Scan(&bidF, &askF, &brokerOut)
	}
	if err != nil {
		return nil, err
	}
	if bidF == 0 && askF == 0 {
		// argMax over an empty set returns zero values without an error.
		return nil, fmt.Errorf("no recent ticks for %s (last %dh)", canonical, lookbackHours)
	}
	t.Bid = strconv.FormatFloat(bidF, 'f', -1, 64)
	t.Ask = strconv.FormatFloat(askF, 'f', -1, 64)
	t.Broker = brokerOut
	return &t, nil
}

// OpenTime converts a unix millisecond timestamp to time.Time.
func (b *KlineBar) OpenTime() time.Time {
	return time.UnixMilli(int64(b.OpenTsUnixMs))
}
