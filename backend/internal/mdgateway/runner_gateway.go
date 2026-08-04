package mdgateway

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/mdgateway/adapter/mt4"
	"alphaforge/internal/mdgateway/adapter/mt5"
	"alphaforge/internal/mthub"
)

// startGatewayForAccount creates a gateway for a single account,
// registers it with the manager and hub, fetches broker info, and subscribes
// to tick/profit/order-update streams. Used by both startup load and dynamic
// subscriber to eliminate duplication.
func startGatewayForAccount(ctx context.Context, cfg mdtick.AccountConfig, deps RunnerDeps, mgr *Manager, log *zap.Logger) (Gateway, error) {
	accID := cfg.AccountID

	gw := newGateway(cfg, log)
	if gw == nil {
		return nil, fmt.Errorf("unknown platform: %s", cfg.Platform)
	}
	gw.SetBreaker(mgr.GetOrCreateBreaker(cfg))

	if err := connectWithRediscovery(ctx, &cfg, gw, mgr, log); err != nil {
		return nil, err
	}

	if err := postConnectSetup(ctx, cfg, gw, deps, mgr, log); err != nil {
		return gw, err
	}

	log.Info("mdgateway: gateway active", zap.String("account", accID), zap.String("platform", cfg.Platform))
	return gw, nil
}

func newGateway(cfg mdtick.AccountConfig, log *zap.Logger) Gateway {
	switch strings.ToLower(cfg.Platform) {
	case "mt4":
		return mt4.New(cfg, log)
	case "mt5":
		return mt5.New(cfg, log)
	}
	return nil
}

func connectWithRediscovery(ctx context.Context, cfg *mdtick.AccountConfig, gw Gateway, mgr *Manager, log *zap.Logger) error {
	if err := gw.Connect(ctx); err != nil {
		if mgr.rediscoverer == nil {
			return fmt.Errorf("connect: %w", err)
		}
		newHost, rerr := mgr.rediscoverer.MaybeRediscover(
			ctx, err, cfg.Broker, cfg.Platform, cfg.AccountID,
			func(host string) error {
				testCfg := *cfg
				testCfg.BrokerHost = host
				testGW := newGateway(testCfg, log)
				if testGW == nil {
					return fmt.Errorf("unknown platform: %s", cfg.Platform)
				}
				if cerr := testGW.Connect(ctx); cerr != nil {
					return cerr
				}
				_ = testGW.Disconnect(ctx)
				return nil
			},
		)
		if rerr != nil || newHost == "" {
			return fmt.Errorf("connect: %w", err)
		}
		cfg.BrokerHost = newHost
		gw = newGateway(*cfg, log)
		gw.SetBreaker(mgr.GetOrCreateBreaker(*cfg))
		if err := gw.Connect(ctx); err != nil {
			return fmt.Errorf("connect after rediscovery: %w", err)
		}
		log.Info("mdgateway: gateway connected after host rediscovery",
			zap.String("account", cfg.AccountID), zap.String("newHost", newHost))
	}
	return nil
}

func postConnectSetup(ctx context.Context, cfg mdtick.AccountConfig, gw Gateway, deps RunnerDeps, mgr *Manager, log *zap.Logger) error {
	accID := cfg.AccountID

	gw.SetStatusCallback(func(status, message string) {
		if deps.OnAccountStatus != nil {
			deps.OnAccountStatus(accID, cfg.UserID, status, message)
		}
	})

	if deps.PG != nil {
		updateAccountStatusOnConnect(ctx, accID, gw, deps, log)
	}

	if err := mgr.AddGateway(ctx, gw, nil); err != nil {
		_ = gw.Disconnect(ctx)
		return fmt.Errorf("add gateway: %w", err)
	}

	if deps.Hub != nil {
		if exec, ok := gw.(mthub.OrderExecutor); ok {
			deps.Hub.Register(accID,
				&mthub.Session{AccountID: accID, CreatedAt: Clk.Now()}, exec)
		}
	}

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

	syms := cfg.Symbols
	if len(syms) == 0 {
		syms = defaultQuoteSymbols()
	}
	if err := gw.Subscribe(ctx, syms, mgr.HandleTick); err != nil {
		_ = mgr.RemoveGateway(ctx, accID)
		_ = gw.Disconnect(ctx)
		return fmt.Errorf("tick subscribe: %w", err)
	}

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
	return nil
}

func updateAccountStatusOnConnect(ctx context.Context, accID string, gw Gateway, deps RunnerDeps, log *zap.Logger) {
	isInvestor := false
	accountInfoErr := ""
	if infoProvider, ok := gw.(mdtick.AccountInfoProvider); ok {
		if info, err := infoProvider.GetAccountInfo(ctx); err == nil && info != nil {
			isInvestor = info.IsInvestor
		} else if err != nil {
			accountInfoErr = err.Error()
			log.Warn("mdgateway: GetAccountInfo failed during postConnectSetup",
				zap.String("account", accID), zap.Error(err))
		}
	}
	if accountInfoErr != "" {
		msg := accountInfoErr
		if len(msg) > 512 {
			msg = msg[:512]
		}
		if _, err := deps.PG.Exec(ctx,
			`UPDATE mt_accounts SET account_status = 'reconnecting',
			 last_error = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1 AND deleted_at IS NULL`,
			accID, msg); err != nil {
			log.Warn("mdgateway: failed to update account status to reconnecting",
				zap.String("account", accID), zap.Error(err))
		}
		return
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
