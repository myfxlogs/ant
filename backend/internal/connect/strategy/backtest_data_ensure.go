package strategy

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"go.uber.org/zap"

	"alphaforge/internal/repository"
)

// ensureBarData guarantees that PG has bar data covering the requested date range
// for the given symbol+timeframe. If PG data is missing or stale, it auto-fetches
// from the connected MT broker via PriceHistory RPC, persists to PG, then re-queries.
//
// First-principle: the broker is the source of truth for market data; PG is a cache.
// The user should never have to manually manage data availability — the system
// fetches what it needs transparently, and only errors when the broker itself
// cannot provide the data.
//
// Parameters:
//   - accountID: preferred MT account to fetch from (may be empty — falls back to any connected account)
//   - from/to: requested date range (nil = use full available range, no fetch needed)
func (s *StrategyExecutionServer) ensureBarData(ctx context.Context, symbol, timeframe string, from, to *time.Time, accountID string) error {
	if s.marketDataRepo == nil {
		return nil
	}
	if from == nil || to == nil {
		return nil // no date range specified — use whatever PG has
	}

	// Check current PG data coverage for this symbol+timeframe.
	pgBars, err := s.marketDataRepo.GetKlines(ctx, symbol, "", timeframe, nil, nil, 100000)
	if err != nil {
		s.log.Warn("ensureBarData: PG query failed, proceeding to broker fetch",
			zap.String("symbol", symbol), zap.String("timeframe", timeframe), zap.Error(err))
	}

	const gapTolerance = 5 * time.Minute
	needFetch := false
	var fetchFrom, fetchTo time.Time

	if len(pgBars) == 0 {
		needFetch = true
		fetchFrom = *from
		fetchTo = *to
	} else {
		dataStart := time.UnixMilli(int64(pgBars[0].OpenTsUnixMs))
		dataEnd := time.UnixMilli(int64(pgBars[len(pgBars)-1].OpenTsUnixMs))

		startGap := dataStart.After(from.Add(gapTolerance))
		endGap := dataEnd.Before(to.Add(-gapTolerance))
		if startGap || endGap {
			needFetch = true
			// Fetch the full requested range — InsertBars upsert handles overlaps.
			// Fetching only the gap is tricky when both sides are missing;
			// a single full-range call is simpler and correct.
			fetchFrom = *from
			fetchTo = *to
		}
	}

	if !needFetch {
		return nil
	}

	s.log.Info("ensureBarData: PG data gap detected, fetching from broker",
		zap.String("symbol", symbol),
		zap.String("timeframe", timeframe),
		zap.Time("fetchFrom", fetchFrom),
		zap.Time("fetchTo", fetchTo),
		zap.String("accountID", accountID))

	if err := s.fetchAndStoreBars(ctx, symbol, timeframe, fetchFrom, fetchTo, accountID); err != nil {
		return connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("failed to fetch market data for %s %s from broker: %w — please ensure your MT account is connected and the symbol/date range is available",
				symbol, timeframe, err))
	}
	return nil
}

// fetchAndStoreBars calls the MT broker's PriceHistory RPC, converts the bars to
// KlineBar format, and persists them to PG via InsertBars.
func (s *StrategyExecutionServer) fetchAndStoreBars(ctx context.Context, symbol, timeframe string, from, to time.Time, accountID string) error {
	if s.mtHub == nil {
		return fmt.Errorf("no MT account connected (mtHub is nil)")
	}

	// Resolve which account to use: prefer the specified account, fall back to any connected.
	acctID := accountID
	if acctID == "" {
		ids := s.mtHub.ActiveAccountIDs()
		if len(ids) == 0 {
			return fmt.Errorf("no MT account connected")
		}
		acctID = ids[0]
	}

	// Determine broker name for the account (needed for PG insert).
	broker := s.mtHub.Platform(acctID)
	if broker == "" {
		broker = "unknown"
	}

	// MT4 QuoteHistory uses `from` as the END time and fetches `count` bars backwards.
	// MT5 PriceHistory uses `from` and `to` as a range.
	// The mtHub.PriceHistory API abstracts this: from/to are unix seconds, count is max bars.
	fromSec := from.Unix()
	toSec := to.Unix()
	if fromSec >= toSec {
		return fmt.Errorf("invalid date range: from >= to")
	}

	// Single call with full range — broker API caps at ~5000 bars.
	// MT4 QuoteHistory fetches `count` bars ending at `to` (from param computes count).
	// MT5 PriceHistory fetches bars in [from, to] range.
	// InsertBars upsert handles any overlaps with existing PG data.
	// For gaps larger than 5000 bars, the backfiller cron job will fill progressively.
	const maxBars = 5000
	periodMs := periodToMs(timeframe)
	if periodMs == 0 {
		return fmt.Errorf("unsupported timeframe: %s", timeframe)
	}

	s.log.Info("ensureBarData: fetching bars from broker",
		zap.String("symbol", symbol),
		zap.String("timeframe", timeframe),
		zap.Int64("fromSec", fromSec),
		zap.Int64("toSec", toSec),
		zap.Int("maxBars", maxBars))

	mtBars, err := s.mtHub.PriceHistory(ctx, acctID, symbol, timeframe, fromSec, toSec, maxBars)
	if err != nil {
		return fmt.Errorf("broker PriceHistory: %w", err)
	}
	if len(mtBars) == 0 {
		return fmt.Errorf("broker returned no bars for %s %s in range %s to %s",
			symbol, timeframe, from.UTC().Format("2006-01-02"), to.UTC().Format("2006-01-02"))
	}

	allBars := make([]repository.KlineBar, 0, len(mtBars))
	for _, b := range mtBars {
		openMs := b.Time.UnixMilli()
		closeMs := openMs + periodMs
		allBars = append(allBars, repository.KlineBar{
			Broker:        broker,
			SymbolRaw:     symbol,
			Canonical:     symbol,
			Period:        timeframe,
			OpenTsUnixMs:  uint64(openMs),
			CloseTsUnixMs: uint64(closeMs),
			Open:          b.Open,
			High:          b.High,
			Low:           b.Low,
			Close:         b.Close,
			Volume:        b.Volume.InexactFloat64(),
			TickCount:     1,
			IsReplay:      true,
			AccountID:     acctID,
		})
	}

	s.log.Info("ensureBarData: persisting fetched bars to PG",
		zap.String("symbol", symbol),
		zap.String("timeframe", timeframe),
		zap.Int("barCount", len(allBars)))

	if err := s.marketDataRepo.InsertBars(ctx, allBars); err != nil {
		return fmt.Errorf("persist bars to PG: %w", err)
	}

	s.log.Info("ensureBarData: bars persisted successfully",
		zap.String("symbol", symbol),
		zap.String("timeframe", timeframe),
		zap.Int("barCount", len(allBars)))

	return nil
}

// periodToMs converts a timeframe string to milliseconds.
func periodToMs(tf string) int64 {
	switch tf {
	case "1m", "M1":
		return 60 * 1000
	case "5m", "M5":
		return 5 * 60 * 1000
	case "15m", "M15":
		return 15 * 60 * 1000
	case "30m", "M30":
		return 30 * 60 * 1000
	case "1h", "H1":
		return 60 * 60 * 1000
	case "4h", "H4":
		return 4 * 60 * 60 * 1000
	case "1d", "D1":
		return 24 * 60 * 60 * 1000
	case "1w", "W1":
		return 7 * 24 * 60 * 60 * 1000
	default:
		return 0
	}
}
