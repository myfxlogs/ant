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

	var gw Gateway
	switch strings.ToLower(cfg.Platform) {
	case "mt4":
		gw = mt4.New(cfg, log)
	case "mt5":
		gw = mt5.New(cfg, log)
	default:
		return nil, fmt.Errorf("unknown platform: %s", cfg.Platform)
	}
	gw.SetBreaker(mgr.GetOrCreateBreaker(cfg))

	if err := gw.Connect(ctx); err != nil {
		// §0: On host errors (connection refused / DNS failure), attempt
		// broker host rediscovery before giving up.
		if mgr.rediscoverer != nil {
			newHost, rerr := mgr.rediscoverer.MaybeRediscover(
				ctx, err, cfg.Broker, cfg.Platform, accID,
				func(host string) error {
					testCfg := cfg
					testCfg.BrokerHost = host
					var testGW Gateway
					switch strings.ToLower(cfg.Platform) {
					case "mt4":
						testGW = mt4.New(testCfg, log)
					case "mt5":
						testGW = mt5.New(testCfg, log)
					}
					if cerr := testGW.Connect(ctx); cerr != nil {
						return cerr
					}
					testGW.Disconnect(ctx)
					return nil
				},
			)
			if rerr == nil && newHost != "" {
				// Rediscovery succeeded — reconnect with the new host.
				cfg.BrokerHost = newHost
				gw = mt4.New(cfg, log)
				if strings.ToLower(cfg.Platform) == "mt5" {
					gw = mt5.New(cfg, log)
				}
				gw.SetBreaker(mgr.GetOrCreateBreaker(cfg))
				if err := gw.Connect(ctx); err != nil {
					return nil, fmt.Errorf("connect after rediscovery: %w", err)
				}
				log.Info("mdgateway: gateway connected after host rediscovery",
					zap.String("account", accID), zap.String("newHost", newHost))
			} else {
				return nil, fmt.Errorf("connect: %w", err)
			}
		} else {
			return nil, fmt.Errorf("connect: %w", err)
		}
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
		_ = gw.Disconnect(ctx)
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
		_ = mgr.RemoveGateway(ctx, accID)
		_ = gw.Disconnect(ctx)
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
