package system

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"anttrader/internal/repository"
)

func periodSeconds(period string) int64 {
	switch period {
	case "1m":
		return 60
	case "5m":
		return 5 * 60
	case "15m":
		return 15 * 60
	case "30m":
		return 30 * 60
	case "1h":
		return 3600
	case "4h":
		return 4 * 3600
	case "1d":
		return 86400
	case "1w":
		return 7 * 86400
	default:
		return 3600
	}
}

// backfillKlines pulls historical bars from the broker for ALL timeframes
// (1m/5m/15m/30m/1h/4h/1d/1w). Each period is fetched and inserted
// independently — one period failing does not affect the others.
// Deduplicates: only one backfill per account+symbol runs at a time.
func (s *MtHubServer) backfillKlines(accountID, rawSymbol string) {
	key := accountID + ":" + rawSymbol
	s.backfillMu.Lock()
	if s.backfilling[key] {
		s.backfillMu.Unlock()
		return
	}
	s.backfilling[key] = true
	s.backfillMu.Unlock()
	defer func() {
		s.backfillMu.Lock()
		delete(s.backfilling, key)
		s.backfillMu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	broker, err := s.platform.GetAccountBroker(ctx, accountID)
	if err != nil || broker == "" {
		s.log.Warn("backfill: get account broker failed",
			zap.String("account", accountID), zap.Error(err))
		return
	}

	now := time.Now().Unix()
	periods := []string{"1m", "5m", "15m", "30m", "1h", "4h", "1d", "1w"}

	var (
		mu       sync.Mutex
		inserted int
		failed   int
		total    int
		eg       errgroup.Group
	)
	// Limit concurrent broker PriceHistory calls. Some MT servers silently
	// drop concurrent requests for the same symbol, causing data loss (e.g.
	// brokerFallback grabbed the 1h slot while backfill was fetching it).
	eg.SetLimit(4)

	for _, period := range periods {
		p := period // capture
		eg.Go(func() error {
			from := now - 300*periodSeconds(p)
			// Cap at 2-year retention (md_bars partition TTL).
			if minFrom := now - 730*24*3600; from < minFrom {
				from = minFrom
			}

			bars, err := s.svc.PriceHistory(ctx, accountID, rawSymbol, p, from, now, 500)
			if err != nil {
				s.log.Warn("backfill: fetch failed",
					zap.String("symbol", rawSymbol), zap.String("period", p), zap.Error(err))
				mu.Lock()
				failed++
				mu.Unlock()
				return nil
			}
			if len(bars) == 0 {
				return nil
			}

			// Convert to KlineBar.
			closeMs := uint64(periodSeconds(p) * 1000)
			klines := make([]repository.KlineBar, len(bars))
			for i, b := range bars {
				klines[i] = repository.KlineBar{
					Broker:        broker,
					Canonical:     rawSymbol,
					Period:        p,
					OpenTsUnixMs:  uint64(b.Time.UnixMilli()),
					CloseTsUnixMs: uint64(b.Time.UnixMilli()) + closeMs,
					Open:          b.Open,
					High:          b.High,
					Low:           b.Low,
					Close:         b.Close,
					Volume:        b.Volume,
				}
			}

			// Insert per-period — a partition gap in 1w won't kill 1m/1h/etc.
			if err := s.marketData.InsertBars(ctx, klines); err != nil {
				s.log.Warn("backfill: insert failed",
					zap.String("symbol", rawSymbol), zap.String("period", p),
					zap.Int("bars", len(klines)), zap.Error(err))
				mu.Lock()
				failed++
				mu.Unlock()
				return nil
			}

			s.log.Info("backfill: inserted",
				zap.String("symbol", rawSymbol), zap.String("period", p),
				zap.Int("bars", len(klines)))
			mu.Lock()
			inserted++
			total += len(klines)
			mu.Unlock()
			return nil
		})
	}

	_ = eg.Wait()
	s.log.Info("backfill: complete",
		zap.String("symbol", rawSymbol),
		zap.Int("total_bars", total),
		zap.Int("periods_ok", inserted),
		zap.Int("periods_failed", failed))
}

// brokerFallback fetches K-line bars directly from the broker for a single
// period when the database has no data yet. Returns []repository.KlineBar so
// the caller can reuse the same OHLCV conversion path as GetKlines.
func (s *MtHubServer) brokerFallback(
	ctx context.Context, accountID, symbol, period string, limit int,
) []repository.KlineBar {
	key := accountID + ":" + symbol

	// If backfill is in progress for this symbol, wait for it to finish
	// and re-query the database. Avoids duplicate broker calls that cause
	// MT connection contention — the backfill's concurrent PriceHistory
	// call for the same period would otherwise return empty.
	s.backfillMu.Lock()
	bfRunning := s.backfilling[key]
	s.backfillMu.Unlock()

	if bfRunning {
		for i := 0; i < 20; i++ {
			time.Sleep(500 * time.Millisecond)
			s.backfillMu.Lock()
			stillRunning := s.backfilling[key]
			s.backfillMu.Unlock()
			if !stillRunning {
				break
			}
		}
		// Re-query: backfill should have populated the database.
		dbBars, err := s.marketData.GetKlines(ctx, symbol, "", period, nil, nil, int32(limit))
		if err == nil && len(dbBars) >= 50 {
			return dbBars
		}
	}

	// Database still insufficient — call broker directly.
	now := time.Now().Unix()
	from := now - int64(limit)*periodSeconds(period)
	bars, err := s.svc.PriceHistory(ctx, accountID, symbol, period, from, now, limit)
	if err != nil || len(bars) == 0 {
		return nil
	}
	s.log.Info("PriceHistory: broker fallback",
		zap.String("symbol", symbol), zap.String("period", period),
		zap.Int("bars", len(bars)))
	out := make([]repository.KlineBar, 0, len(bars))
	for _, b := range bars {
		closeMs := uint64(b.Time.UnixMilli()) + uint64(periodSeconds(period)*1000)
		out = append(out, repository.KlineBar{
			Canonical:     symbol,
			Period:        period,
			OpenTsUnixMs:  uint64(b.Time.UnixMilli()),
			CloseTsUnixMs: closeMs,
			Open:          b.Open,
			High:          b.High,
			Low:           b.Low,
			Close:         b.Close,
			Volume:        b.Volume,
		})
	}
	return out
}

// needsBrokerFallback returns true when database data is insufficient or has
// large discontinuities (e.g., account disconnected for days — old cached bars
// + new bars pass the count check but span a gap). Broker fallback fills the gap.
func (s *MtHubServer) needsBrokerFallback(bars []repository.KlineBar, period string, limit int) bool {
	if len(bars) < 50 {
		return true
	}
	if period == "" || len(bars) == 0 {
		return false
	}
	secs := periodSeconds(period)
	if secs <= 0 {
		return false
	}
	now := time.Now().Unix()
	// Stale check: latest bar close > 2x period behind now
	lastClose := int64(bars[len(bars)-1].CloseTsUnixMs / 1000)
	if now-lastClose > 2*secs {
		return true
	}
	// Internal gap check: any adjacent bars with gap > 2x period
	for i := 1; i < len(bars); i++ {
		gap := int64(bars[i].OpenTsUnixMs/1000) - int64(bars[i-1].CloseTsUnixMs/1000)
		if gap > 2*secs {
			return true
		}
	}
	return false
}
