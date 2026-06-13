// ch_market_data_store.go — ClickHouse implementation of MarketDataStore.
// Used as an optional read-optimized replica for analytical queries.
// PG is the system of record; CH is disposable — if CH is down, reads fall back to PG.

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

// CHMarketDataStore implements MarketDataStore backed by ClickHouse.
// Optimized for analytical queries (GetKlines, LoadFinalizedBars) at scale.
type CHMarketDataStore struct {
	conn clickhouse.Conn
	log  *zap.Logger
}

// NewCHMarketDataStore creates a CH-backed market data store.
func NewCHMarketDataStore(conn clickhouse.Conn, log *zap.Logger) *CHMarketDataStore {
	return &CHMarketDataStore{conn: conn, log: log}
}

// ── Read paths (analytical queries — CH is optimized for these) ─────────────

func (s *CHMarketDataStore) GetKlines(ctx context.Context, canonical, broker, period string, from, to *time.Time, limit int32) ([]KlineBar, error) {
	if limit <= 0 {
		limit = 500
	}
	period = normalizeTimeframe(period)

	const lookbackMonths = 6
	query := `SELECT broker, canonical, period, open_ts_unix_ms, close_ts_unix_ms,
			toFloat64(open), toFloat64(high), toFloat64(low), toFloat64(close),
			volume, tick_count
		FROM md_bars
		WHERE canonical = $1 AND period = $2 AND is_replay = 0`
	args := []any{canonical, period}

	if from != nil {
		query += fmt.Sprintf(` AND close_ts_unix_ms >= $%d`, len(args)+1)
		args = append(args, from.UnixMilli())
	} else {
		cutoffMs := time.Now().AddDate(0, -lookbackMonths, 0).UnixMilli()
		query += fmt.Sprintf(` AND close_ts_unix_ms >= $%d`, len(args)+1)
		args = append(args, cutoffMs)
	}
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

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ch get klines: %w", err)
	}
	defer rows.Close()
	return scanKlineBars(rows, s.log)
}

func (s *CHMarketDataStore) GetLatestTick(ctx context.Context, canonical, broker string) (*LatestTick, error) {
	const lookbackHours = 24
	cutoffMs := time.Now().Add(-lookbackHours * time.Hour).UnixMilli()

	var t LatestTick
	var bidF, askF float64
	var err error
	if broker != "" {
		err = s.conn.QueryRow(ctx,
			`SELECT toFloat64(argMax(bid, ts_unix_ms)),
				toFloat64(argMax(ask, ts_unix_ms)),
				argMax(broker, ts_unix_ms)
			FROM md_ticks
			WHERE canonical = $1 AND broker = $2 AND is_replay = 0
			AND arrived_unix_ms >= $3`,
			canonical, broker, cutoffMs,
		).Scan(&bidF, &askF, &t.Broker)
	} else {
		err = s.conn.QueryRow(ctx,
			`SELECT toFloat64(argMax(bid, ts_unix_ms)),
				toFloat64(argMax(ask, ts_unix_ms)),
				argMax(broker, ts_unix_ms)
			FROM md_ticks
			WHERE canonical = $1 AND is_replay = 0
			AND arrived_unix_ms >= $2`,
			canonical, cutoffMs,
		).Scan(&bidF, &askF, &t.Broker)
	}
	if err != nil {
		return nil, err
	}
	if bidF == 0 && askF == 0 {
		return nil, fmt.Errorf("no recent ticks for %s (last %dh)", canonical, lookbackHours)
	}
	t.Bid = strconv.FormatFloat(bidF, 'f', -1, 64)
	t.Ask = strconv.FormatFloat(askF, 'f', -1, 64)
	return &t, nil
}

func (s *CHMarketDataStore) LoadFinalizedBars(ctx context.Context, since time.Time) (map[FinalizedKey][]int64, error) {
	result := make(map[FinalizedKey][]int64)
	rows, err := s.conn.Query(ctx,
		`SELECT broker, canonical, period, close_ts_unix_ms
		FROM md_bars WHERE close_ts_unix_ms >= $1`,
		since.UnixMilli(),
	)
	if err != nil {
		s.log.Error("ch: load finalized bars FAILED", zap.Error(err))
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
	s.log.Info("ch: loaded finalized bars", zap.Int("keys", len(result)))
	return result, nil
}

func (s *CHMarketDataStore) MaxCloseTs(ctx context.Context, broker, canonical, period string) (int64, error) {
	var ts int64
	err := s.conn.QueryRow(ctx,
		`SELECT max(close_ts_unix_ms) FROM md_bars
		WHERE broker = $1 AND canonical = $2 AND period = $3
		AND close_ts_unix_ms >= $4`,
		broker, canonical, period,
		time.Now().Add(-90*24*time.Hour).UnixMilli(),
	).Scan(&ts)
	if err != nil {
		return 0, nil
	}
	return ts, nil
}

func (s *CHMarketDataStore) FetchActualReturn(ctx context.Context, symbol string, predictedAt time.Time) (float64, error) {
	start := predictedAt
	end := predictedAt.Add(7 * 24 * time.Hour)

	var openPrice, closePrice float64
	err := s.conn.QueryRow(ctx,
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
	if openPrice <= 0 {
		return 0, fmt.Errorf("no price data for %s", symbol)
	}
	return (closePrice - openPrice) / openPrice, nil
}

// ── Write paths (batch insert for dual-write from PgWriter) ──────────────────

func (s *CHMarketDataStore) InsertBars(ctx context.Context, bars []KlineBar) error {
	if len(bars) == 0 {
		return nil
	}
	batch, err := s.conn.PrepareBatch(ctx,
		"INSERT INTO md_bars (broker, symbol_raw, canonical, period, open_ts_unix_ms, close_ts_unix_ms, open, high, low, close, volume, tick_count, is_replay, account_id)")
	if err != nil {
		return fmt.Errorf("ch insert bars: prepare: %w", err)
	}
	defer batch.Abort()
	for _, b := range bars {
		if err := batch.Append(b.Broker, b.Canonical, b.Canonical, b.Period, b.OpenTsUnixMs, b.CloseTsUnixMs,
			decimal.NewFromFloat(b.Open), decimal.NewFromFloat(b.High), decimal.NewFromFloat(b.Low), decimal.NewFromFloat(b.Close),
			b.Volume, b.TickCount, uint8(0), ""); err != nil {
			return fmt.Errorf("ch insert bars: append: %w", err)
		}
	}
	return batch.Send()
}

func (s *CHMarketDataStore) InsertTicks(ctx context.Context, ticks []TickRecord) error {
	if len(ticks) == 0 {
		return nil
	}
	batch, err := s.conn.PrepareBatch(ctx,
		"INSERT INTO md_ticks (user_id, account_id, broker, symbol_raw, canonical, ts_unix_ms, arrived_unix_ms, bid, ask, bid_volume, ask_volume, is_replay)")
	if err != nil {
		return fmt.Errorf("ch insert ticks: prepare: %w", err)
	}
	defer batch.Abort()
	for _, t := range ticks {
		replay := uint8(0)
		if t.IsReplay {
			replay = 1
		}
		if err := batch.Append(t.UserID, t.AccountID, t.Broker, t.SymbolRaw, t.Canonical,
			t.TsUnixMs, t.ArrivedUnixMs, t.Bid, t.Ask, t.BidVolume, t.AskVolume, replay); err != nil {
			return fmt.Errorf("ch insert ticks: append: %w", err)
		}
	}
	return batch.Send()
}
