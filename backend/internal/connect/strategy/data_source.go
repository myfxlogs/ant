// data_source.go — BarSource interface + LiveSource implementation.
//
// BarSource abstracts market data delivery for strategy execution, enabling
// the same strategy code to run in both backtest (pre-loaded historical bars)
// and live (streaming real-time bars) modes.
//
// Architecture:
//   - LiveSource: unified bar source implementing both BarSource and LiveBarSubscriber.
//     Fetch() loads historical bars from ClickHouse via MarketDataRepository.
//     Subscribe() streams real-time bars via MtHubService.
//   - LiveBarSubscriber: streaming interface for live/paper modes.

package strategy

import (
	"context"
	"strconv"
	"time"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/mthub"
	"anttrader/internal/repository"
)

// BarSource provides K-line bars for strategy execution.
type BarSource interface {
	// Name returns a human-readable identifier ("backtest", "live") for logging/metrics.
	Name() string
	// Fetch returns historical bars from ClickHouse via MarketDataRepository.
	Fetch(ctx context.Context, symbol, timeframe string, from, to *time.Time) ([]*antv1.ExecuteKlineBar, error)
}

// LiveBarSubscriber is an optional interface for sources that support streaming.
// LiveSource implements this.
type LiveBarSubscriber interface {
	BarSource
	// Subscribe returns a channel emitting bars as they arrive, plus a cancel function.
	Subscribe(accountID string) (<-chan *mthub.BarUpdate, func())
}

// ── LiveSource: unified bar source for live + backtest execution ──
//
// Implements both BarSource (Fetch for backtest) and LiveBarSubscriber (Subscribe for live).
// Fetch delegates to MarketDataRepository (ClickHouse), Subscribe delegates to MtHubService.

type LiveSource struct {
	hub     *mthub.MtHubService
	mktRepo repository.MarketDataStore
}

func NewLiveSource(hub *mthub.MtHubService, mktRepo repository.MarketDataStore) *LiveSource {
	return &LiveSource{hub: hub, mktRepo: mktRepo}
}

func (s *LiveSource) Name() string { return "live" }

func (s *LiveSource) Fetch(ctx context.Context, symbol, timeframe string, from, to *time.Time) ([]*antv1.ExecuteKlineBar, error) {
	if s.mktRepo == nil || symbol == "" || timeframe == "" {
		return nil, nil
	}
	chBars, err := s.mktRepo.GetKlines(ctx, symbol, "", timeframe, from, to, 100000)
	if err != nil {
		return nil, err
	}
	return klineBarsToProto(chBars), nil
}

func (s *LiveSource) Subscribe(accountID string) (<-chan *mthub.BarUpdate, func()) {
	return s.hub.SubscribeBarUpdates(accountID)
}

// klineBarsToProto converts repository KlineBars to proto ExecuteKlineBars.
// Bars are reversed to chronological order (oldest first).
func klineBarsToProto(chBars []repository.KlineBar) []*antv1.ExecuteKlineBar {
	klines := make([]*antv1.ExecuteKlineBar, 0, len(chBars))
	for i := len(chBars) - 1; i >= 0; i-- {
		b := chBars[i]
		klines = append(klines, &antv1.ExecuteKlineBar{
			OpenTimeMs:  int64(b.OpenTsUnixMs),
			CloseTimeMs: int64(b.CloseTsUnixMs),
			Open:        b.Open.String(),
			High:        b.High.String(),
			Low:         b.Low.String(),
			Close:       b.Close.String(),
			Volume:      strconv.FormatFloat(b.Volume, 'f', -1, 64),
		})
	}
	return klines
}
