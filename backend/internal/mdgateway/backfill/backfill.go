// Package backfill provides historical market data backfill from PostgreSQL
// to ClickHouse. Used during migration and periodic data repair.
package backfill

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Config holds backfill configuration.
type Config struct {
	BatchSize    int           // rows per ClickHouse batch insert
	Concurrency  int           // max concurrent symbol workers
	FromTime     time.Time     // backfill start (inclusive)
	ToTime       time.Time     // backfill end (exclusive)
	Symbols      []string      // symbols to backfill; empty = all from pg
	Period       string        // bar period ("1m", "5m", "1h", "1d")
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		BatchSize:   1000,
		Concurrency: 4,
		Period:      "1m",
	}
}

// Runner executes backfill jobs.
type Runner struct {
	pgPool *pgxpool.Pool
	chConn clickhouse.Conn
	log    *zap.Logger
}

// New creates a backfill runner.
func New(pgPool *pgxpool.Pool, chConn clickhouse.Conn, log *zap.Logger) *Runner {
	return &Runner{pgPool: pgPool, chConn: chConn, log: log}
}

// RunBars backfills OHLCV bars from PostgreSQL (md_bars_raw) to ClickHouse (md_bars).
func (r *Runner) RunBars(ctx context.Context, cfg Config) (int, error) {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 1000
	}

	query := `SELECT broker, symbol, period, open_ts_ms, close_ts_ms,
		open, high, low, close, volume, tick_count
		FROM md_bars_raw
		WHERE ($1::timestamptz IS NULL OR open_ts_ms >= $1)
		  AND ($2::timestamptz IS NULL OR close_ts_ms < $2)
		  AND ($3::text = '' OR symbol = ANY($4::text[]))
		  AND period = $5
		ORDER BY open_ts_ms`

	var fromPtr, toPtr *time.Time
	if !cfg.FromTime.IsZero() {
		fromPtr = &cfg.FromTime
	}
	if !cfg.ToTime.IsZero() {
		toPtr = &cfg.ToTime
	}

	rows, err := r.pgPool.Query(ctx, query, fromPtr, toPtr, cfg.Symbols, cfg.Period)
	if err != nil {
		return 0, fmt.Errorf("backfill: query pg: %w", err)
	}
	defer rows.Close()

	total := 0
	batch, err := r.chConn.PrepareBatch(ctx, "INSERT INTO md_bars")
	if err != nil {
		return 0, fmt.Errorf("backfill: prepare ch batch: %w", err)
	}
	batchCnt := 0

	for rows.Next() {
		var broker, symbol, period string
		var openTs, closeTs int64
		var open, high, low, close, volume float64
		var tickCount uint32
		if err := rows.Scan(&broker, &symbol, &period, &openTs, &closeTs,
			&open, &high, &low, &close, &volume, &tickCount); err != nil {
			r.log.Warn("backfill: scan row", zap.Error(err))
			continue
		}
		if err := batch.Append(broker, symbol, symbol, period, uint64(openTs),
			uint64(closeTs), open, high, low, close, volume, tickCount); err != nil {
			r.log.Warn("backfill: append to batch", zap.Error(err))
			continue
		}
		batchCnt++
		total++

		if batchCnt >= cfg.BatchSize {
			if err := batch.Send(); err != nil {
				return total, fmt.Errorf("backfill: send ch batch: %w", err)
			}
			batch, err = r.chConn.PrepareBatch(ctx, "INSERT INTO md_bars")
			if err != nil {
				return total, fmt.Errorf("backfill: prepare next ch batch: %w", err)
			}
			batchCnt = 0
		}
	}

	if batchCnt > 0 {
		if err := batch.Send(); err != nil {
			return total, fmt.Errorf("backfill: send final ch batch: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return total, fmt.Errorf("backfill: rows iteration: %w", err)
	}

	r.log.Info("backfill: bars complete",
		zap.Int("total_rows", total),
		zap.String("period", cfg.Period),
	)
	return total, nil
}

// RunTicks backfills tick data from PostgreSQL to ClickHouse.
func (r *Runner) RunTicks(ctx context.Context, cfg Config) (int, error) {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 1000
	}

	query := `SELECT user_id, broker, symbol, ts_unix_ms, arrived_unix_ms,
		bid, ask, bid_volume, ask_volume
		FROM md_ticks_raw
		WHERE ($1::timestamptz IS NULL OR ts_unix_ms >= $1)
		  AND ($2::timestamptz IS NULL OR ts_unix_ms < $2)
		ORDER BY ts_unix_ms`

	var fromPtr, toPtr *time.Time
	if !cfg.FromTime.IsZero() {
		fromPtr = &cfg.FromTime
	}
	if !cfg.ToTime.IsZero() {
		toPtr = &cfg.ToTime
	}

	rows, err := r.pgPool.Query(ctx, query, fromPtr, toPtr)
	if err != nil {
		return 0, fmt.Errorf("backfill: query pg: %w", err)
	}
	defer rows.Close()

	total := 0
	batch, err := r.chConn.PrepareBatch(ctx, "INSERT INTO md_ticks")
	if err != nil {
		return 0, fmt.Errorf("backfill: prepare ch batch: %w", err)
	}
	batchCnt := 0

	for rows.Next() {
		var userID, broker, symbol string
		var tsUnixMs, arrivedUnixMs int64
		var bid, ask, bidVol, askVol float64
		if err := rows.Scan(&userID, &broker, &symbol, &tsUnixMs, &arrivedUnixMs,
			&bid, &ask, &bidVol, &askVol); err != nil {
			r.log.Warn("backfill: scan row", zap.Error(err))
			continue
		}
		if err := batch.Append(userID, broker, symbol, symbol,
			uint64(tsUnixMs), uint64(arrivedUnixMs), bid, ask, bidVol, askVol); err != nil {
			r.log.Warn("backfill: append to batch", zap.Error(err))
			continue
		}
		batchCnt++
		total++

		if batchCnt >= cfg.BatchSize {
			if err := batch.Send(); err != nil {
				return total, fmt.Errorf("backfill: send ch batch: %w", err)
			}
			batch, err = r.chConn.PrepareBatch(ctx, "INSERT INTO md_ticks")
			if err != nil {
				return total, fmt.Errorf("backfill: prepare next ch batch: %w", err)
			}
			batchCnt = 0
		}
	}

	if batchCnt > 0 {
		if err := batch.Send(); err != nil {
			return total, fmt.Errorf("backfill: send final ch batch: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return total, fmt.Errorf("backfill: rows iteration: %w", err)
	}

	r.log.Info("backfill: ticks complete", zap.Int("total_rows", total))
	return total, nil
}
