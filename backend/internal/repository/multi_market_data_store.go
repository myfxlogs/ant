// multi_market_data_store.go — multi-backend market data store.
// Routes heavy analytical reads to ClickHouse, writes+light reads to PostgreSQL.
// If CH is unavailable (not configured or down), falls back to PG transparently.

package repository

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// MultiMarketDataStore routes reads to the optimal backend and writes to PG.
// CH is an optional read-optimized replica for analytical queries.
// PG is the system of record for all writes and light reads.
type MultiMarketDataStore struct {
	pg     MarketDataStore // system of record
	ch     MarketDataStore // read-optimized replica (nil if not configured)
	log    *zap.Logger
}

// NewMultiMarketDataStore creates a multi-backend store.
// ch may be nil if ClickHouse is not configured.
func NewMultiMarketDataStore(pg, ch MarketDataStore, log *zap.Logger) *MultiMarketDataStore {
	return &MultiMarketDataStore{pg: pg, ch: ch, log: log}
}

// chOrPg returns ch if available and healthy, otherwise pg.
// Analytical queries benefit from CH columnar storage at scale.
func (s *MultiMarketDataStore) analytical() MarketDataStore {
	if s.ch != nil {
		return s.ch
	}
	return s.pg
}

// ── Analytical reads (route to CH if available) ──────────────────────────────

func (s *MultiMarketDataStore) GetKlines(ctx context.Context, canonical, broker, period string, from, to *time.Time, limit int32) ([]KlineBar, error) {
	bars, err := s.analytical().GetKlines(ctx, canonical, broker, period, from, to, limit)
	if err != nil && s.ch != nil {
		s.log.Warn("multi: CH GetKlines failed, falling back to PG", zap.Error(err))
		return s.pg.GetKlines(ctx, canonical, broker, period, from, to, limit)
	}
	return bars, err
}

func (s *MultiMarketDataStore) GetLatestTick(ctx context.Context, canonical, broker string) (*LatestTick, error) {
	tick, err := s.analytical().GetLatestTick(ctx, canonical, broker)
	if err != nil && s.ch != nil {
		return s.pg.GetLatestTick(ctx, canonical, broker)
	}
	return tick, err
}

func (s *MultiMarketDataStore) LoadFinalizedBars(ctx context.Context, since time.Time) (map[FinalizedKey][]int64, error) {
	bars, err := s.analytical().LoadFinalizedBars(ctx, since)
	if err != nil && s.ch != nil {
		s.log.Warn("multi: CH LoadFinalizedBars failed, falling back to PG", zap.Error(err))
		return s.pg.LoadFinalizedBars(ctx, since)
	}
	return bars, err
}

func (s *MultiMarketDataStore) MaxCloseTs(ctx context.Context, broker, canonical, period string) (int64, error) {
	ts, err := s.analytical().MaxCloseTs(ctx, broker, canonical, period)
	if err != nil && s.ch != nil {
		return s.pg.MaxCloseTs(ctx, broker, canonical, period)
	}
	return ts, err
}

func (s *MultiMarketDataStore) GetLatestBars(ctx context.Context, since time.Time) ([]KlineBar, error) {
	bars, err := s.analytical().GetLatestBars(ctx, since)
	if err != nil && s.ch != nil {
		s.log.Warn("multi: CH GetLatestBars failed, falling back to PG", zap.Error(err))
		return s.pg.GetLatestBars(ctx, since)
	}
	return bars, err
}

// ── Light reads (always PG — no analytical benefit from CH) ──────────────────

func (s *MultiMarketDataStore) FetchActualReturn(ctx context.Context, symbol string, predictedAt time.Time) (float64, error) {
	return s.pg.FetchActualReturn(ctx, symbol, predictedAt)
}

// ── Writes (always PG — system of record) ────────────────────────────────────

func (s *MultiMarketDataStore) InsertBars(ctx context.Context, bars []KlineBar) error {
	return s.pg.InsertBars(ctx, bars)
}

func (s *MultiMarketDataStore) InsertTicks(ctx context.Context, ticks []TickRecord) error {
	return s.pg.InsertTicks(ctx, ticks)
}
