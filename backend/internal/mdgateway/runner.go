package mdgateway

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	anttrace "anttrader/internal/trace"
	"anttrader/internal/mdgateway/adapter/mdtick"
	"anttrader/internal/mdgateway/adapter"
	"anttrader/internal/mdgateway/adapter/mt4"
	"anttrader/internal/mdgateway/adapter/mt5"
	"anttrader/internal/mdgateway/backfiller"
	"anttrader/internal/mthub"
	"anttrader/internal/secrets"
)

// RunnerDeps collects all infrastructure needed to start mdgateway.
type RunnerDeps struct {
	Log            *zap.Logger
	PG             *pgxpool.Pool
	CH             clickhouse.Conn
	NATSConn       *nats.Conn
	SpillDir       string          // default /var/lib/ant/spill
	Secrets        secrets.Client  // decrypts account passwords and mtapi tokens
	OnAccountProfit     func(accountID, userID string, p *mdtick.ProfitUpdate)    // receives real-time balance/equity from mtapi OnOrderProfit
	OnOrderUpdate       func(accountID, userID string, o *mdtick.OrderUpdate)     // receives real-time order/position changes from mtapi OnOrderUpdate
	OnAccountDisconnect func(accountID string)                                     // B-1.3: called when gateway stops/fails for an account
	OnBrokerInfo        func(accountID, platform, broker string, info *mdtick.BrokerInfo) // B-2.2: called once after successful Connect
	OnBar               func(bar *mdtick.Bar)                                               // called when a bar is finalized (for realtime SSE push)
	Hub                 *mthub.Hub
	BrokerRegistry      *adapter.BrokerRegistry // M12-C2: multi-broker registry; gateways registered on start
}

// Run assembles and starts the full mdgateway pipeline. Blocks until ctx.Done.
func Run(ctx context.Context, deps RunnerDeps) error {
	log := deps.Log
	log.Info("mdgateway: starting", zap.String("spill_dir", deps.SpillDir))

	// --- OTel trace (ADR-0010 §2.3) ---
	tracer := anttrace.New()
	defer tracer.Shutdown(context.Background())
	log.Info("mdgateway: trace", zap.Bool("enabled", tracer.Enabled()))

	// --- SpillWriter ---
	spillCfg := DefaultSpillConfig()
	if deps.SpillDir != "" {
		spillCfg.Dir = deps.SpillDir
	}
	spillWriter, err := NewSpillWriter(spillCfg, log)
	if err != nil {
		return fmt.Errorf("create spill writer: %w", err)
	}
	defer spillWriter.Close()

	// --- CHWriter ---
	chCfg := DefaultCHWriterConfig()
	chWriter := NewCHWriter(chCfg, deps.CH, spillWriter, log)

	// --- Publisher ---
	var js nats.JetStreamContext
	if deps.NATSConn != nil {
		js, err = deps.NATSConn.JetStream()
		if err != nil {
			log.Warn("mdgateway: JetStream not available", zap.Error(err))
		}
	}
	publisher := NewPublisher(js)

	// --- BarAggregator with finalized bars ---
	aggregator := NewBarAggregator()
	finalized, err := loadFinalizedBars(ctx, deps.CH, log)
	if err != nil {
		return fmt.Errorf("load finalized bars from ClickHouse: %w", err)
	}
	aggregator.LoadFinalizedBars(finalized)

	// --- SpillReplay (runs after aggregator for finality dedup) ---
	replay := NewSpillReplay(spillCfg.Dir, publisher, chWriter, aggregator, log)
	if n, err := replay.Run(ctx); err != nil {
		log.Warn("mdgateway: spill_replay errors", zap.Error(err))
	} else {
		log.Info("mdgateway: spill_replay complete", zap.Int("rows", n))
	}

	// --- Normalizer + Quality + Dedup ---
	normalizer := NewNormalizer(deps.PG)
	quality := NewQuality(DefaultQualityConfig())
	dedup := NewTickDedup(0) // default size (1000)

	// --- Start CHWriter background loop ---
	// #nosec G118 — pipeline ctx is the correct lifecycle scope for CHWriter
	go chWriter.Start(ctx)

	// --- Backfiller (initial scan + 6h cron) ---
	bf, srcMap := startBackfiller(ctx, deps, aggregator, publisher, chWriter, log)

	// --- NormalizerInvalidator (PG LISTEN) ---
	invalidator := NewNormalizerInvalidator(log, deps.PG, func(broker, symbolRaw string) {
		normalizer.InvalidateCache(broker, symbolRaw)
	})
	invalidator.Start(ctx, newPGListener(ctx, deps.PG, log))

	// --- Manager (wires HandleTick pipeline) ---
	mgr := NewManager(ManagerDeps{
		Normalizer:  normalizer,
		Quality:     quality,
		Dedup:       dedup,
		Aggregator:  aggregator,
		Publisher:   publisher,
		CHWriter:    chWriter,
		SpillWriter: spillWriter,
		OnBar:       deps.OnBar,
		Log:         log,
	})
	mgr.SetOTelTracer(tracer)
	mgr.SetBaseContext(ctx)

	chWriter.SetOnSpillFail(func(brokerKey string, err error) {
		log.Warn("mdgateway: spill failed", zap.String("broker", brokerKey), zap.Error(err))
	})

	// --- Open bar ticker (500ms) for real-time price updates ---
	// #nosec G118 — pipeline ctx is the correct lifecycle scope for open bar ticker
	go mgr.StartOpenBarTicker(ctx)

	// --- Health monitor (start before gateways so accounts with no ticks are caught) ---
	// #nosec G118 — pipeline ctx is the correct lifecycle scope for health monitor
	go healthMonitor(ctx, mgr, chWriter, log, deps.OnAccountDisconnect)

	// --- Load active accounts and start gateways ---
	var cfgs []mdtick.AccountConfig
	var loadErr error
	for attempt := 0; attempt < 5; attempt++ {
		cfgs, loadErr = loadAccountConfigs(ctx, deps)
		if loadErr == nil {
			break
		}
		log.Warn("mdgateway: load account configs failed, retrying",
			zap.Int("attempt", attempt+1),
			zap.Error(loadErr))
		time.Sleep(time.Duration(1<<attempt) * time.Second)
	}
	if loadErr != nil {
		return fmt.Errorf("load account configs after retries: %w", loadErr)
	}
	var firstMT4 *mt4.Gateway
	var firstMT5 *mt5.Gateway
	for _, cfg := range cfgs {
		accID := cfg.AccountID
		log.Info("mdgateway: starting gateway",
			zap.String("account", accID),
			zap.String("platform", cfg.Platform),
			zap.String("broker", cfg.Broker))

		gw, err := startGatewayForAccount(ctx, cfg, deps, mgr, log)
		if err != nil {
			log.Error("mdgateway: gateway start failed",
				zap.String("account", accID), zap.Error(err))
			continue
		}
		if bfSrc, ok := gw.(backfiller.MTAPIBarSource); ok {
			srcMap.gws[accID] = bfSrc
		}
		if firstMT4 == nil {
			if mt4gw, ok := gw.(*mt4.Gateway); ok {
				firstMT4 = mt4gw
			}
		}
		if firstMT5 == nil {
			if mt5gw, ok := gw.(*mt5.Gateway); ok {
				firstMT5 = mt5gw
			}
		}
	}

	// M12-C2: register collected gateways in the multi-broker registry.
	if deps.BrokerRegistry != nil {
		if err := adapter.RegisterDefaults(deps.BrokerRegistry, firstMT4, firstMT5); err != nil {
			log.Error("mdgateway: failed to register broker defaults", zap.Error(err))
		} else {
			log.Info("mdgateway: broker registry populated",
				zap.Bool("mt4", firstMT4 != nil),
				zap.Bool("mt5", firstMT5 != nil))
		}
	}

	// --- Account event subscriber (NATS: account.connect/disconnect/reconnect) ---
	startAccountEventSubscriber(ctx, deps, mgr, log)

	// --- Wait for shutdown ---
	<-ctx.Done()
	log.Info("mdgateway: shutting down")

	ticks, bars := chWriter.drain()
	chWriter.Flush(ctx, ticks, bars)
	_ = invalidator
	_ = bf
	log.Info("mdgateway: stopped")
	return nil
}

// startGatewayForAccount connects a single account's gateway to the broker,
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

	// Persist connected status so the frontend stops showing "Connecting".
	if deps.PG != nil {
		if _, err := deps.PG.Exec(ctx,
			`UPDATE mt_accounts SET account_status = 'connected', updated_at = CURRENT_TIMESTAMP WHERE id = $1`,
			accID); err != nil {
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
