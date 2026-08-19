package strategy

import (
	"context"
	"time"

	"go.uber.org/zap"

	"alphaforge/internal/mthub"
)

// subscribeSymbolsWithRetry ensures the gateway subscribes to the strategy's
// symbol, retrying until the gateway session is available (handles startup
// race where strategy launches before gateway connects).
//
// After the first successful subscription, it continues to periodically
// re-confirm (every 60s). This handles gateway rebuilds (e.g. NATS
// disconnect/reconnect events) where the new gateway starts with empty
// subscribedSymbols and the strategy's symbol would otherwise be lost.
func subscribeSymbolsWithRetry(ctx context.Context, mtHub *mthub.MtHubService, accountID, symbol string, log *zap.Logger) {
	if mtHub == nil || symbol == "" {
		return
	}
	go func() {
		backoff := 2 * time.Second
		maxBackoff := 30 * time.Second
		const reConfirmInterval = 60 * time.Second
		for {
			if ctx.Err() != nil {
				return
			}
			err := mtHub.SubscribeSymbols(ctx, accountID, []string{symbol})
			if err == nil {
				// Success: wait reConfirmInterval before re-confirming.
				// AddSymbols is idempotent — re-subscribing an already-subscribed
				// symbol is a no-op on the mtapi side.
				select {
				case <-ctx.Done():
					return
				case <-time.After(reConfirmInterval):
				}
				continue
			}
			log.Warn("LiveStrategyRunner: SubscribeSymbols retry",
				zap.String("symbol", symbol), zap.String("bar_account", accountID), zap.Error(err))
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}()
}
