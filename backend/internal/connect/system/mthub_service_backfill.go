package system

import (
	"context"
	"time"

	"go.uber.org/zap"

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
// (1m/5m/15m/30m/1h/4h/1d/1w) and inserts them into ClickHouse.
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

	allBars, gotCount := s.fetchAllPeriods(ctx, accountID, rawSymbol, broker)
	if len(allBars) == 0 {
		s.log.Info("backfill: no bars for any period",
			zap.String("symbol", rawSymbol), zap.String("account", accountID))
		return
	}
	if err := s.marketData.InsertBars(ctx, allBars); err != nil {
		s.log.Warn("backfill: insert bars failed",
			zap.String("symbol", rawSymbol), zap.Int("count", len(allBars)), zap.Error(err))
		return
	}
	s.log.Info("backfill: inserted bars",
		zap.String("symbol", rawSymbol),
		zap.Int("total", len(allBars)), zap.Int("periods", gotCount))
}

// fetchAllPeriods calls broker PriceHistory for each timeframe and returns
// ClickHouse-ready bars. Each period gets a lookback of ~300 bars.
func (s *MtHubServer) fetchAllPeriods(
	ctx context.Context, accountID, symbol, broker string,
) ([]repository.KlineBar, int) {
	now := time.Now().Unix()
	periods := []string{"1m", "5m", "15m", "30m", "1h", "4h", "1d", "1w"}
	var all []repository.KlineBar
	got := 0

	for _, period := range periods {
		from := now - 300*periodSeconds(period)
		bars, err := s.svc.PriceHistory(ctx, accountID, symbol, period, from, now, 500)
		if err != nil {
			s.log.Warn("backfill: period failed",
				zap.String("period", period), zap.String("symbol", symbol), zap.Error(err))
			continue
		}
		if len(bars) == 0 {
			continue
		}
		for _, b := range bars {
			all = append(all, repository.KlineBar{
				Broker:        broker,
				Canonical:     symbol,
				Period:        period,
				OpenTsUnixMs:  uint64(b.Time.UnixMilli()),
				CloseTsUnixMs: uint64(b.Time.UnixMilli()),
				Open:          b.Open,
				High:          b.High,
				Low:           b.Low,
				Close:         b.Close,
				Volume:        b.Volume,
			})
		}
		got++
	}
	return all, got
}

// brokerFallback fetches K-line bars directly from the broker for a single
// period when ClickHouse has no data yet. Returns []repository.KlineBar so
// the caller can reuse the same OHLCV conversion path as GetKlines.
func (s *MtHubServer) brokerFallback(
	ctx context.Context, accountID, symbol, period string, limit int,
) []repository.KlineBar {
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
		out = append(out, repository.KlineBar{
			Canonical:     symbol,
			Period:        period,
			OpenTsUnixMs:  uint64(b.Time.UnixMilli()),
			CloseTsUnixMs: uint64(b.Time.UnixMilli()),
			Open:          b.Open,
			High:          b.High,
			Low:           b.Low,
			Close:         b.Close,
			Volume:        b.Volume,
		})
	}
	return out
}
