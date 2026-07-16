// market_data_pg.go — PostgreSQL implementation of MarketDataStore.
// Uses native PG partitioned tables (md_ticks, md_bars) with pgx.
//
// CH→PG query translations:
//   argMax(bid, ts)        → ORDER BY ts DESC LIMIT 1
//   LIMIT 1 BY (a,b,c)     → DISTINCT ON (a,b,c)
//   toFloat64(decimal_col) → direct scan (pgx converts NUMERIC→float64)
//   PrepareBatch           → pgx.CopyFrom

package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// PgMarketDataStore implements MarketDataStore backed by PostgreSQL.
type PgMarketDataStore struct {
	pool *pgxpool.Pool
	log  *zap.Logger
}

// NewPgMarketDataStore creates a PG-backed market data store.
func NewPgMarketDataStore(pool *pgxpool.Pool, log *zap.Logger) *PgMarketDataStore {
	return &PgMarketDataStore{pool: pool, log: log}
}

// ── Read paths ──────────────────────────────────────────────────────────────

// GetKlines returns OHLCV bars from md_bars.
// Replaces CH LIMIT 1 BY with DISTINCT ON for dedup.
//
// When broker is specified, the query filters to that broker and deduplicates
// by (canonical, period, close_ts) only — broker is a constant filter so it
// doesn't belong in the distinct key. When broker is empty, the distinct key
// includes broker so each broker's bars are preserved separately.
//
// Tiebreaker: tick_count DESC ensures that when multiple accounts write the
// same bar (same broker, canonical, period, close_ts), the bar with the most
// underlying ticks is selected — a quality-driven choice rather than arbitrary.
func (s *PgMarketDataStore) GetKlines(ctx context.Context, canonical, broker, period string, from, to *time.Time, limit int32) ([]KlineBar, error) {
	if limit <= 0 {
		limit = 500
	}
	period = normalizeTimeframe(period)

	const lookbackMonths = 6
	args := []any{canonical, period}
	argN := func() string { return fmt.Sprintf("$%d", len(args)+1) }

	// Distinct key and ORDER BY depend on whether broker is pinned.
	var distinctKey, orderClause string
	if broker != "" {
		// Broker is a filter constant — don't include it in the distinct key.
		distinctKey = "canonical, period, close_ts_unix_ms"
		orderClause = "canonical, period, close_ts_unix_ms, tick_count DESC"
	} else {
		// No broker filter — distinct key includes broker to keep per-broker
		// time series separate. This is a legacy fallback; new code should
		// always specify a broker for deterministic results.
		distinctKey = "broker, canonical, period, close_ts_unix_ms"
		orderClause = "broker, canonical, period, close_ts_unix_ms, tick_count DESC"
	}

	query := fmt.Sprintf(`SELECT DISTINCT ON (%s)
			broker, canonical, period, open_ts_unix_ms, close_ts_unix_ms,
			open, high, low, close, volume, tick_count
		FROM md_bars
		WHERE canonical = $1 AND period = $2 AND is_replay = 0`, distinctKey)

	if from != nil {
		query += fmt.Sprintf(` AND close_ts_unix_ms >= %s`, argN())
		args = append(args, from.UnixMilli())
	} else {
		cutoffMs := time.Now().AddDate(0, -lookbackMonths, 0).UnixMilli()
		query += fmt.Sprintf(` AND close_ts_unix_ms >= %s`, argN())
		args = append(args, cutoffMs)
	}
	if to != nil {
		query += fmt.Sprintf(` AND close_ts_unix_ms <= %s`, argN())
		args = append(args, to.UnixMilli())
	}
	if broker != "" {
		query += fmt.Sprintf(` AND broker = %s`, argN())
		args = append(args, broker)
	}
	query += fmt.Sprintf(` ORDER BY %s LIMIT %s`, orderClause, argN())
	args = append(args, limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("pg get klines: %w", err)
	}
	defer rows.Close()
	return scanKlineBars(rows, s.log)
}

// GetLatestTick returns the most recent bid/ask for a symbol.
// Replaces CH argMax() with ORDER BY arrived_unix_ms DESC LIMIT 1.
func (s *PgMarketDataStore) GetLatestTick(ctx context.Context, canonical, broker string) (*LatestTick, error) {
	const lookbackHours = 24
	cutoffMs := time.Now().Add(-lookbackHours * time.Hour).UnixMilli()

	var t LatestTick
	var bidF, askF float64
	var err error

	if broker != "" {
		err = s.pool.QueryRow(ctx,
			`SELECT bid, ask, broker
			 FROM md_ticks
			 WHERE canonical = $1 AND broker = $2 AND is_replay = 0
			   AND arrived_unix_ms >= $3
			 ORDER BY arrived_unix_ms DESC
			 LIMIT 1`,
			canonical, broker, cutoffMs,
		).Scan(&bidF, &askF, &t.Broker)
	} else {
		err = s.pool.QueryRow(ctx,
			`SELECT bid, ask, broker
			 FROM md_ticks
			 WHERE canonical = $1 AND is_replay = 0
			   AND arrived_unix_ms >= $2
			 ORDER BY arrived_unix_ms DESC
			 LIMIT 1`,
			canonical, cutoffMs,
		).Scan(&bidF, &askF, &t.Broker)
	}
	if err != nil {
		return nil, err
	}
	if bidF == 0 && askF == 0 {
		return nil, fmt.Errorf("no recent ticks for %s (last %dh)", canonical, lookbackHours)
	}

	// Match CH format: strconv.FormatFloat with -1 precision (no trailing zeros).
	t.Bid = strconv.FormatFloat(bidF, 'f', -1, 64)
	t.Ask = strconv.FormatFloat(askF, 'f', -1, 64)
	return &t, nil
}

// LoadFinalizedBars returns all existing close_ts values per key within the lookback.
func (s *PgMarketDataStore) LoadFinalizedBars(ctx context.Context, since time.Time) (map[FinalizedKey][]int64, error) {
	result := make(map[FinalizedKey][]int64)
	rows, err := s.pool.Query(ctx,
		`SELECT broker, canonical, period, close_ts_unix_ms
		 FROM md_bars WHERE close_ts_unix_ms >= $1
		 ORDER BY broker, canonical, period, close_ts_unix_ms`,
		since.UnixMilli(),
	)
	if err != nil {
		s.log.Error("pg: load finalized bars FAILED", zap.Error(err))
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var broker, canonical, period string
		var closeTs int64
		if err := rows.Scan(&broker, &canonical, &period, &closeTs); err != nil {
			continue
		}
		fk := FinalizedKey{broker, canonical, period}
		result[fk] = append(result[fk], closeTs)
	}
	s.log.Info("pg: loaded finalized bars", zap.Int("keys", len(result)))
	return result, nil
}

// MaxCloseTs returns the latest close_ts for a (broker, canonical, period) tuple.
func (s *PgMarketDataStore) MaxCloseTs(ctx context.Context, broker, canonical, period string) (int64, error) {
	var ts int64
	err := s.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(close_ts_unix_ms), 0)
		 FROM md_bars
		 WHERE broker = $1 AND canonical = $2 AND period = $3
		   AND close_ts_unix_ms >= $4`,
		broker, canonical, period,
		time.Now().Add(-90*24*time.Hour).UnixMilli(),
	).Scan(&ts)
	if err != nil {
		return 0, nil // treat as no data
	}
	return ts, nil
}

// FetchActualReturn computes the 7-day price return after predictedAt.
func (s *PgMarketDataStore) FetchActualReturn(ctx context.Context, symbol string, predictedAt time.Time) (float64, error) {
	start := predictedAt
	end := predictedAt.Add(7 * 24 * time.Hour)

	var openPrice, closePrice decimal.Decimal
	err := s.pool.QueryRow(ctx,
		`SELECT
			COALESCE((SELECT open FROM md_bars
			  WHERE canonical = $1 AND period = '1d'
			    AND open_ts_unix_ms >= $2 AND open_ts_unix_ms < $3
			  ORDER BY open_ts_unix_ms ASC LIMIT 1), 0),
			COALESCE((SELECT close FROM md_bars
			  WHERE canonical = $1 AND period = '1d'
			    AND close_ts_unix_ms <= $4
			  ORDER BY close_ts_unix_ms DESC LIMIT 1), 0)`,
		symbol,
		start.UnixMilli(), end.UnixMilli(),
		end.UnixMilli(),
	).Scan(&openPrice, &closePrice)
	if err != nil {
		return 0, err
	}
	if !openPrice.GreaterThan(decimal.Zero) {
		return 0, fmt.Errorf("no price data for %s", symbol)
	}
	return closePrice.Sub(openPrice).Div(openPrice).InexactFloat64(), nil
}

// ── Write paths ──────────────────────────────────────────────────────────────

// InsertBars batch-inserts kline bars via COPY protocol.
func (s *PgMarketDataStore) InsertBars(ctx context.Context, bars []KlineBar) error {
	if len(bars) == 0 {
		return nil
	}
	cols := []string{"broker", "symbol_raw", "canonical", "period", "open_ts_unix_ms", "close_ts_unix_ms",
		"open", "high", "low", "close", "volume", "tick_count", "is_replay", "account_id"}
	rows := make([][]any, len(bars))
	for i, b := range bars {
		rows[i] = []any{
			b.Broker, b.Canonical, b.Canonical, b.Period,
			int64(b.OpenTsUnixMs), int64(b.CloseTsUnixMs),
			b.Open, b.High, b.Low, b.Close, b.Volume, int32(b.TickCount),
			int16(0), "",
		}
	}
	return s.copyFrom(ctx, "md_bars", cols, rows)
}

// InsertTicks batch-inserts ticks via COPY protocol.
func (s *PgMarketDataStore) InsertTicks(ctx context.Context, ticks []TickRecord) error {
	if len(ticks) == 0 {
		return nil
	}
	cols := []string{"user_id", "account_id", "broker", "symbol_raw", "canonical",
		"ts_unix_ms", "arrived_unix_ms", "bid", "ask", "bid_volume", "ask_volume", "is_replay"}
	rows := make([][]any, len(ticks))
	for i, t := range ticks {
		replay := int16(0)
		if t.IsReplay {
			replay = 1
		}
		rows[i] = []any{
			t.UserID, t.AccountID, t.Broker, t.SymbolRaw, t.Canonical,
			t.TsUnixMs, t.ArrivedUnixMs, t.Bid, t.Ask, t.BidVolume, t.AskVolume,
			replay,
		}
	}
	return s.copyFrom(ctx, "md_ticks", cols, rows)
}

// copyFrom executes a pgx CopyFrom for batch insertion.
func (s *PgMarketDataStore) copyFrom(ctx context.Context, table string, columns []string, rows [][]any) error {
	_, err := s.pool.CopyFrom(
		ctx,
		pgx.Identifier{table},
		columns,
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("pg copyfrom %s: %w", table, err)
	}
	return nil
}
