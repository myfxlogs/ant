// data_source.go — BarSource interface + BacktestSource / LiveSource implementations.
//
// BarSource abstracts market data delivery for strategy execution, enabling
// the same strategy code to run in both backtest (pre-loaded historical bars)
// and live (streaming real-time bars) modes.
//
// Architecture:
//   - BacktestSource: loads bars from ClickHouse via MarketDataRepository
//   - LiveSource: subscribes to real-time bar updates via MtHubService
//   - LiveBarSubscriber: optional streaming interface for live/paper modes

package strategy

import (
	"context"
	"time"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/mthub"
	"anttrader/internal/repository"
)

// BarSource provides K-line bars for strategy execution.
type BarSource interface {
	// Name returns a human-readable identifier ("backtest", "live") for logging/metrics.
	Name() string
	// Fetch returns historical bars. For live sources this returns nil — bars arrive via Subscribe.
	Fetch(ctx context.Context, symbol, timeframe string, from, to *time.Time) ([]*antv1.ExecuteKlineBar, error)
}

// LiveBarSubscriber is an optional interface for sources that support streaming.
// LiveSource implements this; BacktestSource does not.
type LiveBarSubscriber interface {
	BarSource
	// Subscribe returns a channel emitting bars as they arrive, plus a cancel function.
	Subscribe(accountID string) (<-chan *mthub.BarUpdate, func())
}

// ── BacktestSource: loads historical bars from ClickHouse ──

type BacktestSource struct {
	marketDataRepo *repository.MarketDataRepository
}

func NewBacktestSource(repo *repository.MarketDataRepository) *BacktestSource {
	return &BacktestSource{marketDataRepo: repo}
}

func (s *BacktestSource) Name() string { return "backtest" }

func (s *BacktestSource) Fetch(ctx context.Context, symbol, timeframe string, from, to *time.Time) ([]*antv1.ExecuteKlineBar, error) {
	if s.marketDataRepo == nil || symbol == "" || timeframe == "" {
		return nil, nil
	}
	chBars, _ := s.marketDataRepo.GetKlines(ctx, symbol, "", timeframe, from, to, 2000)
	klines := make([]*antv1.ExecuteKlineBar, 0, len(chBars))
	for i := len(chBars) - 1; i >= 0; i-- {
		b := chBars[i]
		klines = append(klines, &antv1.ExecuteKlineBar{
			OpenTimeMs:  int64(b.OpenTsUnixMs),
			CloseTimeMs: int64(b.CloseTsUnixMs),
			Open:        b.Open, High: b.High, Low: b.Low, Close: b.Close, Volume: b.Volume,
		})
	}
	return klines, nil
}

// ── LiveSource: subscribes to real-time bar updates via MtHubService ──

type LiveSource struct {
	hub *mthub.MtHubService
}

func NewLiveSource(hub *mthub.MtHubService) *LiveSource {
	return &LiveSource{hub: hub}
}

func (s *LiveSource) Name() string { return "live" }

func (s *LiveSource) Fetch(_ context.Context, _, _ string, _, _ *time.Time) ([]*antv1.ExecuteKlineBar, error) {
	return nil, nil // live mode: bars arrive via Subscribe
}

func (s *LiveSource) Subscribe(accountID string) (<-chan *mthub.BarUpdate, func()) {
	return s.hub.SubscribeBarUpdates(accountID)
}
