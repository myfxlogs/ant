package system

import (
	"context"
	"time"

	"go.uber.org/zap"

	"alphaforge/internal/repository"
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
			Volume:        b.Volume.InexactFloat64(),
		})
	}
	return out
}

// needsBrokerFallback returns true when database data is insufficient or has
// large discontinuities (e.g., account disconnected for days — old cached bars
// + new bars pass the count check but span a gap). Broker fallback fills the gap.
func (s *MtHubServer) needsBrokerFallback(bars []repository.KlineBar, period string) bool {
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
