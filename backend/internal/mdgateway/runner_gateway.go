package mdgateway

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"anttrader/internal/mdgateway/adapter/mdtick"
	"anttrader/internal/mdgateway/adapter/mt4"
	"anttrader/internal/mdgateway/adapter/mt5"
	"anttrader/internal/mthub"
)

// startGatewayForAccount creates a gateway for a single account,
// registers it with the manager and hub, fetches broker info, and subscribes
// to tick/profit/order-update streams. Used by both startup load and dynamic
// subscriber to eliminate duplication.
func startGatewayForAccount(ctx context.Context, cfg mdtick.AccountConfig, deps RunnerDeps, mgr *Manager, log *zap.Logger) (Gateway, error) {
	accID := cfg.AccountID

	var gw Gateway
	switch strings.ToLower(cfg.Platform) {
	case "mt4":
		gw = mt4.New(cfg, log)
	case "mt5":
		gw = mt5.New(cfg, log)
	default:
		return nil, fmt.Errorf("unknown platform: %s", cfg.Platform)
	}

	if err := gw.Connect(ctx); err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	// Wire gateway connection state changes → OnAccountStatus callback.
	gw.SetStatusCallback(func(status, message string) {
		if deps.OnAccountStatus != nil {
			deps.OnAccountStatus(accID, cfg.UserID, status, message)
		}
	})

	// Persist connected status + account metadata (investor flag, method).
	if deps.PG != nil {
		isInvestor := false
		if infoProvider, ok := gw.(mdtick.AccountInfoProvider); ok {
			if info, err := infoProvider.GetAccountInfo(ctx); err == nil && info != nil {
				isInvestor = info.IsInvestor
			}
		}
		accountMethod := "master"
		if isInvestor {
			accountMethod = "investor"
		}
		if _, err := deps.PG.Exec(ctx,
			`UPDATE mt_accounts SET account_status = 'connected',
			 is_investor = $2, account_method = $3, last_connected_at = CURRENT_TIMESTAMP,
			 last_error = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = $1 AND deleted_at IS NULL`,
			accID, isInvestor, accountMethod); err != nil {
			log.Warn("mdgateway: failed to update account status to connected",
				zap.String("account", accID), zap.Error(err))
		}
	}

	if err := mgr.AddGateway(ctx, gw, nil); err != nil {
		gw.Disconnect(ctx)
		return gw, fmt.Errorf("add gateway: %w", err)
	}

	// Register with Hub BEFORE FetchBrokerInfo so syncHistory can find the session.
	if deps.Hub != nil {
		if exec, ok := gw.(mthub.OrderExecutor); ok {
			deps.Hub.Register(accID,
				&mthub.Session{AccountID: accID, CreatedAt: Clk.Now()}, exec)
		}
	}

	// Fetch broker-level margin thresholds after Hub registration.
	if deps.OnBrokerInfo != nil {
		if fetcher, ok := gw.(mdtick.BrokerInfoFetcher); ok {
			info, ferr := fetcher.FetchBrokerInfo(ctx)
			if ferr != nil {
				log.Warn("mdgateway: FetchBrokerInfo failed",
					zap.String("account", accID), zap.Error(ferr))
			}
			if info != nil {
				deps.OnBrokerInfo(accID, cfg.Platform, cfg.Broker, info)
			}
		}
	}

	// Subscribe to tick stream.
	syms := cfg.Symbols
	if len(syms) == 0 {
		syms = defaultQuoteSymbols()
	}
	if err := gw.Subscribe(ctx, syms, mgr.HandleTick); err != nil {
		mgr.RemoveGateway(ctx, accID)
		gw.Disconnect(ctx)
		return gw, fmt.Errorf("tick subscribe: %w", err)
	}

	// Subscribe to profit and order-update streams.
	if deps.OnAccountProfit != nil {
		uid, aid := cfg.UserID, accID
		if err := gw.SubscribeProfit(ctx, func(p *mdtick.ProfitUpdate) { deps.OnAccountProfit(aid, uid, p) }); err != nil {
			log.Warn("mdgateway: SubscribeProfit failed",
				zap.String("account", accID), zap.Error(err))
		}
	}
	if deps.OnOrderUpdate != nil {
		uid, aid := cfg.UserID, accID
		if err := gw.SubscribeOrderUpdate(ctx, func(o *mdtick.OrderUpdate) { deps.OnOrderUpdate(aid, uid, o) }); err != nil {
			log.Warn("mdgateway: SubscribeOrderUpdate failed",
				zap.String("account", accID), zap.Error(err))
		}
	}

	log.Info("mdgateway: gateway active", zap.String("account", accID), zap.String("platform", cfg.Platform))
	return gw, nil
}
