package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/shopspring/decimal"
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


// GetKlines returns OHLCV kline bars for a symbol and period, optionally filtered by broker and time range.
func (r *MarketDataRepository) GetKlines(ctx context.Context, canonical, broker, period string, from, to *time.Time, limit int32) ([]KlineBar, error) {
	if limit <= 0 {
		limit = 500
	}

	// Performance: avoid the FINAL modifier on a wide ReplacingMergeTree —
	// it forces a full part merge before sort/limit, blowing up memory on
	// busy symbols. Instead:
	//   1. narrow scan via close_ts_unix_ms range (from/to, or 6-month fallback)
	//   2. dedup with LIMIT 1 BY on the natural primary key, which CH handles
	//      streaming and bounded.
	const lookbackMonths = 6

	// toFloat64() casts: open/high/low/close are Decimal(18,6) in CH and the
	// clickhouse-go driver cannot scan Decimal directly into *float64.
	query := `SELECT broker, canonical, period, open_ts_unix_ms, close_ts_unix_ms,
	                 toFloat64(open), toFloat64(high), toFloat64(low), toFloat64(close),
	                 volume, tick_count
	          FROM md_bars
	          WHERE canonical = $1 AND period = $2 AND is_replay = 0`
	args := []any{canonical, period}

	// Lower bound: use from if provided, otherwise 6-month lookback.
	if from != nil {
		query += fmt.Sprintf(` AND close_ts_unix_ms >= $%d`, len(args)+1)
		args = append(args, from.UnixMilli())
	} else {
		cutoffMs := time.Now().AddDate(0, -lookbackMonths, 0).UnixMilli()
		query += fmt.Sprintf(` AND close_ts_unix_ms >= $%d`, len(args)+1)
		args = append(args, cutoffMs)
	}

	// Upper bound: use to if provided (e.g. "before" timestamp for historical loads).
	if to != nil {
		query += fmt.Sprintf(` AND close_ts_unix_ms <= $%d`, len(args)+1)
		args = append(args, to.UnixMilli())
	}

	if broker != "" {
		query += fmt.Sprintf(` AND broker = $%d`, len(args)+1)
		args = append(args, broker)
	}

	query += fmt.Sprintf(` ORDER BY close_ts_unix_ms DESC
		           LIMIT 1 BY (broker, canonical, period, close_ts_unix_ms)
		           LIMIT $%d`, len(args)+1)
	args = append(args, limit)

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

// InsertBars writes kline bars into the md_bars table (bypassing Buffer for backfill).
func (r *MarketDataRepository) InsertBars(ctx context.Context, bars []KlineBar) error {
	if len(bars) == 0 {
		return nil
	}
	batch, err := r.ch.PrepareBatch(ctx,
		"INSERT INTO md_bars (broker, symbol_raw, canonical, period, open_ts_unix_ms, close_ts_unix_ms, open, high, low, close, volume, tick_count, is_replay, account_id)")
	if err != nil {
		return fmt.Errorf("insert bars: prepare batch: %w", err)
	}
	defer batch.Abort()
	for _, b := range bars {
		o := decimal.NewFromFloat(b.Open)
		h := decimal.NewFromFloat(b.High)
		l := decimal.NewFromFloat(b.Low)
		c := decimal.NewFromFloat(b.Close)
		if err := batch.Append(b.Broker, b.Canonical, b.Canonical, b.Period, b.OpenTsUnixMs, b.CloseTsUnixMs,
			o, h, l, c, b.Volume, b.TickCount, uint8(0), ""); err != nil {
			return fmt.Errorf("insert bars: append: %w", err)
		}
	}
	return batch.Send()
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
			 WHERE canonical = $1 AND broker = $2 AND is_replay = 0
			   AND arrived_unix_ms >= $3`,
			canonical, broker, cutoffMs,
		).Scan(&bidF, &askF, &brokerOut)
	} else {
		err = r.ch.QueryRow(ctx,
			`SELECT toFloat64(argMax(bid, ts_unix_ms)),
			        toFloat64(argMax(ask, ts_unix_ms)),
			        argMax(broker, ts_unix_ms)
			 FROM md_ticks
			 WHERE canonical = $1 AND is_replay = 0
			   AND arrived_unix_ms >= $2`,
			canonical, cutoffMs,
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

// CloseTime converts the close unix millisecond timestamp to time.Time.
func (b *KlineBar) CloseTime() time.Time {
	return time.UnixMilli(int64(b.CloseTsUnixMs))
}
