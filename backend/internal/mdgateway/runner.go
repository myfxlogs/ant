package mdgateway

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"alphaforge/internal/mdgateway/adapter"
	"alphaforge/internal/mdgateway/adapter/brokersearch"
	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/mdgateway/adapter/mt4"
	"alphaforge/internal/mdgateway/adapter/mt5"
	"alphaforge/internal/mdgateway/backfiller"
	"alphaforge/internal/mthub"
	"alphaforge/internal/repository"
	"alphaforge/internal/secrets"
	anttrace "alphaforge/internal/trace"
)

// RunnerDeps collects all infrastructure needed to start mdgateway.
type RunnerDeps struct {
	Log                 *zap.Logger
	PG                  *pgxpool.Pool
	Store               repository.MarketDataStore // PG market data store
	NATSConn            *nats.Conn
	RedisClient         *goredis.Client                                                   // ADR-0012: latest quote cache
	SpillDir            string                                                            // default /var/lib/ant/spill
	Secrets             secrets.Client                                                    // decrypts account passwords and mtapi tokens
	OnAccountProfit     func(accountID, userID string, p *mdtick.ProfitUpdate)            // receives real-time balance/equity from mtapi OnOrderProfit
	OnOrderUpdate       func(accountID, userID string, o *mdtick.OrderUpdate)             // receives real-time order/position changes from mtapi OnOrderUpdate
	OnAccountDisconnect func(accountID string)                                            // B-1.3: called when gateway stops/fails for an account
	OnBrokerInfo        func(accountID, platform, broker string, info *mdtick.BrokerInfo) // B-2.2: called once after successful Connect
	OnBar               func(bar *mdtick.Bar)                                             // called when a bar is finalized (for bar broker push to strategy runner)
	OnAccountStatus     func(accountID, userID, status, message string)                   // called when gateway connection state changes (connected/reconnecting/disconnected)
	OnBreakerTrip       func(accountID, userID, status, message string)                   // called when circuit breaker state changes (circuit_open/circuit_half_open/circuit_closed)
	Hub                 *mthub.Hub
	BrokerRegistry      *adapter.BrokerRegistry // M12-C2: multi-broker registry; gateways registered on start
	Searcher            *brokersearch.Searcher  // §0: broker host rediscovery
}

// Run assembles and starts the full mdgateway pipeline. Blocks until ctx.Done.
func Run(ctx context.Context, deps RunnerDeps) error {
	log := deps.Log
	log.Info("mdgateway: starting")

	// --- OTel trace (ADR-0010 §2.3) ---
	tracer := anttrace.New()
	defer func() { _ = tracer.Shutdown(context.Background()) }()
	log.Info("mdgateway: trace", zap.Bool("enabled", tracer.Enabled()))

	// --- Publisher ---
	var js nats.JetStreamContext
	if deps.NATSConn != nil {
		var jetErr error
		js, jetErr = deps.NATSConn.JetStream()
		if jetErr != nil {
			log.Warn("mdgateway: JetStream not available", zap.Error(jetErr))
		}
	}
	publisher := NewPublisher(js)

	// --- BarAggregator with finalized bars ---
	aggregator := NewBarAggregator()
	finalized, err := loadFinalizedBars(ctx, deps.Store, log)
	if err != nil {
		return fmt.Errorf("load finalized bars: %w", err)
	}
	aggregator.LoadFinalizedBars(finalized)

	// --- Restore in-progress bar state (R1a: bar_aggregator restart recovery) ---
	latestBars, err := deps.Store.GetLatestBars(ctx, time.Now().Add(-30*24*time.Hour))
	if err != nil {
		log.Warn("mdgateway: load latest bars for open bar restore FAILED", zap.Error(err))
	} else {
		restored := aggregator.RestoreOpenBars(latestBars, time.Now().UnixMilli())
		if restored > 0 {
			log.Info("mdgateway: restored open bars after restart", zap.Int("bars", restored))
		}
	}

	// --- PgWriter (sole writer — PG is the only storage backend) ---
	pgCfg := DefaultPgWriterConfig()
	pgWriter := NewPgWriter(pgCfg, deps.Store, log)
	go func(ctx context.Context) { pgWriter.Start(ctx) }(ctx)

	// --- Normalizer + Quality + Dedup ---
	normalizer := NewNormalizer(deps.PG)
	quality := NewQuality(DefaultQualityConfig())
	dedup := NewTickDedup(0) // default size (1000)

	// --- Backfiller (initial scan + 6h cron) ---
	bf, srcMap := startBackfiller(ctx, deps, aggregator, publisher, pgWriter, log)
	_ = bf // backfiller goroutines run until ctx is done; cleanup handled by ctx cancellation

	// --- NormalizerInvalidator (PG LISTEN) ---
	invalidator := NewNormalizerInvalidator(log, deps.PG, func(broker, symbolRaw string) {
		normalizer.InvalidateCache(broker, symbolRaw)
	})
	invalidator.Start(ctx, newPGListener(ctx, deps.PG, log))

	// --- Manager (wires HandleTick pipeline) ---
	mgr := NewManager(ManagerDeps{
		Normalizer:    normalizer,
		Quality:       quality,
		Dedup:         dedup,
		Aggregator:    aggregator,
		Publisher:     publisher,
		PgWriter:      pgWriter,
		OnBar:         deps.OnBar,
		OnBreakerTrip: deps.OnBreakerTrip,
		Log:           log,
		RedisClient:   deps.RedisClient,
	})
	mgr.SetOTelTracer(tracer)
	mgr.SetBaseContext(ctx)

	// Wire synchronous gateway removal for account deletion.
	fullRestart := func(ctx context.Context, accountID string) error {
		_ = mgr.RemoveGateway(ctx, accountID)
		cfg, err := loadSingleAccountConfig(ctx, deps.PG, deps.Secrets, accountID)
		if err != nil || cfg == nil {
			return fmt.Errorf("full restart: load config failed for %s: %w", accountID, err)
		}
		_, err = startGatewayForAccount(ctx, *cfg, deps, mgr, log)
		return err
	}
	if deps.Hub != nil {
		deps.Hub.RemoveGateway = mgr.RemoveGateway
		deps.Hub.ReconnectGateway = fullRestart
	}
	mgr.SetFullRestart(fullRestart)

	// --- Host rediscoverer (§0: broker_host lazy rediscovery) ---
	rediscoverer := NewHostRediscoverer(deps.Searcher, deps.PG, log)
	mgr.SetRediscoverer(rediscoverer)

	// --- Health monitor (start before gateways so accounts with no ticks are caught) ---
	go healthMonitor(ctx, mgr, nil, log, deps.OnAccountDisconnect)

	// --- Load active accounts and start gateways ---
	cfgs, err := loadAccountConfigsWithRetry(ctx, deps, log)
	if err != nil {
		return fmt.Errorf("load account configs after retries: %w", err)
	}
	firstMT4, firstMT5 := startAllGateways(ctx, cfgs, deps, mgr, log, srcMap)

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

	pbars := pgWriter.Drain()
	pgWriter.Flush(ctx, pbars)
	log.Info("mdgateway: stopped")
	return nil
}

func loadAccountConfigsWithRetry(ctx context.Context, deps RunnerDeps, log *zap.Logger) ([]mdtick.AccountConfig, error) {
	for attempt := 0; attempt < 5; attempt++ {
		cfgs, err := loadAccountConfigs(ctx, deps)
		if err == nil {
			return cfgs, nil
		}
		log.Warn("mdgateway: load account configs failed, retrying",
			zap.Int("attempt", attempt+1),
			zap.Error(err))
		time.Sleep(time.Duration(1<<attempt) * time.Second)
	}
	return nil, fmt.Errorf("load account configs failed after 5 retries")
}

func startAllGateways(ctx context.Context, cfgs []mdtick.AccountConfig, deps RunnerDeps, mgr *Manager, log *zap.Logger, srcMap *gatewaySourceMap) (*mt4.Gateway, *mt5.Gateway) {
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
			msg := err.Error()
			if len(msg) > 512 {
				msg = msg[:512]
			}
			if deps.PG != nil {
				_, _ = deps.PG.Exec(ctx,
					`UPDATE mt_accounts SET account_status = 'disconnected',
						 last_error = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1 AND deleted_at IS NULL`,
					accID, msg)
			}
			if deps.OnAccountStatus != nil {
				deps.OnAccountStatus(accID, cfg.UserID, "disconnected", msg)
			}
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
	return firstMT4, firstMT5
}

// startGatewayForAccount connects a single account's gateway to the broker,
// registers it with the manager and hub, fetches broker info, and subscribes
// to tick/profit/order-update streams. Used by both startup load and dynamic
// subscriber to eliminate duplication.
