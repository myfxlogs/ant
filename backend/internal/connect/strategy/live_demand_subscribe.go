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
func subscribeSymbolsWithRetry(ctx context.Context, mtHub *mthub.MtHubService, accountID, symbol string, log *zap.Logger) {
	if mtHub == nil || symbol == "" {
		return
	}
	go func() {
		backoff := 2 * time.Second
		maxBackoff := 30 * time.Second
		for {
			if ctx.Err() != nil {
				return
			}
			if err := mtHub.SubscribeSymbols(ctx, accountID, []string{symbol}); err == nil {
				return
			} else {
				log.Warn("LiveStrategyRunner: SubscribeSymbols retry",
					zap.String("symbol", symbol), zap.String("bar_account", accountID), zap.Error(err))
			}
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
