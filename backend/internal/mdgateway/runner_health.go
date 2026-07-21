package mdgateway

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// healthMonitor checks gateway health every 30s, monitors memory pressure
// for S-2 auto-degradation, and emits stale/dead account metrics.
func healthMonitor(ctx context.Context, mgr *Manager, _ interface{}, log *zap.Logger, onDisconnect func(accountID string)) {
	ticker := Clk.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C():
			var stale, dead int64
			handleDeadAccount := func(h AccountHealth) {
				dead++
				log.Error("mdgateway: dead account — attempting reconnect",
					zap.String("account", h.AccountID),
					zap.String("platform", h.Platform))

				// Attempt reconnect before removing. This covers transient
				// failures (e.g. mtapi proxy restart, network blip) without
				// losing the gateway and its subscriptions.
				if err := mgr.ReconnectGateway(ctx, h.AccountID); err != nil {
					log.Error("mdgateway: reconnect failed, removing dead gateway",
						zap.String("account", h.AccountID), zap.Error(err))
					mgr.MarkDisconnecting(h.AccountID)
					if onDisconnect != nil {
						onDisconnect(h.AccountID)
					}
					if err := mgr.RemoveGateway(ctx, h.AccountID); err != nil {
						log.Warn("mdgateway: remove dead gateway failed",
							zap.String("account", h.AccountID), zap.Error(err))
					}
					mgr.UnmarkDisconnecting(h.AccountID)
					time.Sleep(100 * time.Millisecond)
				} else {
					log.Info("mdgateway: dead account reconnected successfully",
						zap.String("account", h.AccountID),
						zap.String("platform", h.Platform))
					mgr.ResetLastTickAt(h.AccountID)
				}
			}

			// Collect stale accounts for parallel Ping/Health checks.
			// Running them sequentially could block the health loop for
			// N * 3s (timeout) when multiple accounts are stale.
			type staleEntry struct {
				h  AccountHealth
				gw Gateway
			}
			var stales []staleEntry
			for _, h := range mgr.Health() {
				switch h.State {
				case "stale":
					if gw := mgr.GetGateway(h.AccountID); gw != nil {
						stales = append(stales, staleEntry{h, gw})
					} else {
						stale++
					}
				case "dead":
					handleDeadAccount(h)
				case "no_data":
					log.Debug("mdgateway: no data yet",
						zap.String("account", h.AccountID))
				default:
					log.Debug("mdgateway: health",
						zap.String("account", h.AccountID),
						zap.String("state", h.State))
				}
			}

			if len(stales) > 0 {
				var wg sync.WaitGroup
				failures := make(chan AccountHealth, len(stales))
				for _, se := range stales {
					stale++
					wg.Add(1)
					go func(e staleEntry) {
						defer wg.Done()
						if err := e.gw.HealthCheck(ctx); err != nil {
							failures <- e.h
						}
					}(se)
				}
				wg.Wait()
				close(failures)

				for _, se := range stales {
					log.Warn("mdgateway: stale account — no ticks for >5 min",
						zap.String("account", se.h.AccountID),
						zap.String("platform", se.h.Platform))
				}
				for h := range failures {
					log.Warn("mdgateway: stale account failed health check — promoting to dead",
						zap.String("account", h.AccountID),
						zap.String("platform", h.Platform))
					handleDeadAccount(h)
				}
			}
			SetStaleAccountCount(stale, dead)

		}
	}
}

// defaultQuoteSymbols returns a broad set of symbols for mtapi SymbolSubscribe
// when an account has no configured symbols. Kept in sync with frontend COMMON_SYMBOLS.
func defaultQuoteSymbols() []string {
	return []string{
		// Forex majors
		"EURUSDm", "GBPUSDm", "USDJPYm", "AUDUSDm", "NZDUSDm", "USDCADm", "USDCHFm",
		// Forex crosses
		"EURGBPm", "EURJPYm", "GBPJPYm", "AUDJPYm", "NZDJPYm", "CADJPYm", "CHFJPYm",
		"EURCHFm", "EURAUDm", "EURNZDm", "GBPCHFm", "GBPAUDm", "GBPNZDm",
		"GBPCADm", "AUDCADm", "AUDCHFm", "AUDNZDm", "NZDCADm", "NZDCHFm", "CADCHFm",
		// Metals
		"XAUUSDm", "XAGUSDm", "XAUJPYm",
		// Crypto
		"BTCUSDm", "ETHUSDm", "XRPUSDm", "SOLUSDm", "BNBUSDm",
		// Indices
		"US30m", "US100m", "GER40m",
	}
}
