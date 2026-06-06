package mdgateway

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// StartOpenBarTicker periodically pushes in-progress bar snapshots for
// real-time chart updates. Runs every 500ms.
func (m *Manager) StartOpenBarTicker(ctx context.Context) {
	ticker := Clk.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	logTicker := Clk.NewTicker(10 * time.Second)
	defer logTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-logTicker.C():
			if m.onBar == nil {
				m.log.Warn("open_bar_ticker: onBar nil, skipping")
				continue
			}
			bars := m.aggregator.GetOpenBars()
			if len(bars) == 0 {
				m.log.Warn("open_bar_ticker: no open bars")
			} else {
				sample := bars[0]
				m.log.Info("open_bar_ticker",
					zap.Int("open_bars", len(bars)),
					zap.String("sample_account", sample.AccountID),
					zap.String("sample_symbol", sample.Canonical),
					zap.String("sample_period", sample.Period),
				)
			}
		case <-ticker.C():
			if m.onBar == nil {
				continue
			}
			for _, b := range m.aggregator.GetOpenBars() {
				b.IsReplay = false
				m.onBar(b)
			}
		}
	}
}

// Health returns health status for all registered gateways.
func (m *Manager) Health() []AccountHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()
	now := Clk.Now().UnixMilli()
	var result []AccountHealth
	for _, gw := range m.gateways {
		lastAt := m.lastTickAt[gw.AccountID()]
		state := "healthy"
		if lastAt == 0 {
			state = "no_data"
		} else if now-lastAt > 15*60*1000 {
			state = "dead"
		} else if now-lastAt > 5*60*1000 {
			state = "stale"
		}
		result = append(result, AccountHealth{
			AccountID:  gw.AccountID(),
			Platform:   gw.Platform(),
			State:      state,
			LastTickAt: lastAt,
		})
	}
	return result
}

type AccountHealth struct {
	AccountID    string
	Broker       string
	Platform     string
	State        string
	LastTickAt   int64
	CircuitState string
	TickRate1m   float64
}
