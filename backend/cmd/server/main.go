package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"alphaforge/internal/chain"
	"alphaforge/internal/config"
	"alphaforge/internal/connect/strategy"
	"alphaforge/internal/hdwallet"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/marketplace"
	"alphaforge/internal/mdgateway/adapter"
	"alphaforge/internal/mthub"
	notifpubsub "alphaforge/internal/notification"
	"alphaforge/internal/notifier"
	"alphaforge/internal/reconcile"
	"alphaforge/internal/repository"
	"alphaforge/internal/risksvc"
	"alphaforge/internal/secrets"
	alphasentry "alphaforge/internal/sentry"
	"alphaforge/internal/server"
	"alphaforge/internal/service"
	antredis "alphaforge/internal/storage/redis"
	"alphaforge/internal/sweep"
	anttrace "alphaforge/internal/trace"

	"connectrpc.com/otelconnect"
)

func splitAndTrim(s, sep string) []string {
	var out []string
	for _, part := range strings.Split(s, sep) {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func main() {
	log, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer func() { _ = log.Sync() }()

	// ── Sentry: error tracking for production observability ──
	sentryCleanup := alphasentry.Init(log)
	defer sentryCleanup()

	cfg := config.Load()

	// ── ENV-TO-PG-1 boot 链（ADR-0031）：连库 → 密钥客户端 → seed-once + DB-wins overlay ──
	// 必须先于任何 C/D 档消费点；Validate 后移（JWT_SECRET 首选来自 platform_secrets 解密，
	// env 仅过渡兜底）。pool 前移创建，后续 initInfrastructure 复用。
	pool := connectPostgres(cfg, log)
	defer pool.Close()
	secClient := newSecretsClient(cfg, log)
	if err := seedAndOverlayConfig(context.Background(), pool, secClient, cfg); err != nil {
		log.Fatal("config seed/overlay failed", zap.Error(err))
	}
	if err := cfg.Validate(); err != nil {
		log.Fatal("invalid config", zap.Error(err))
	}

	// ── OpenTelemetry: unified tracer provider for all ConnectRPC + pipeline spans ──
	// Configured via standard OTel env vars: OTEL_EXPORTER_OTLP_ENDPOINT, OTEL_SERVICE_NAME.
	traceShutdown, err := anttrace.InitGlobalProvider(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	if err != nil {
		log.Warn("OpenTelemetry init failed, tracing disabled", zap.Error(err))
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := traceShutdown(ctx); err != nil {
			log.Warn("OpenTelemetry shutdown failed", zap.Error(err))
		}
	}()

	// ConnectRPC OpenTelemetry interceptor — creates spans for every RPC call.
	// Uses the global TracerProvider set by InitGlobalProvider above.
	otelInterceptor, err := otelconnect.NewInterceptor(
		otelconnect.WithTrustRemote(),
	)
	if err != nil {
		log.Warn("otelconnect interceptor creation failed", zap.Error(err))
	}

	// Connect to NATS, Redis, and core services (PG pool + secrets client created earlier
	// by the ENV-TO-PG-1 boot chain).
	nc, rdb, accountSvc, platformSvc, jwtSecret, mdStore := initInfrastructure(cfg, log, pool, secClient)
	defer nc.Close()
	defer func() { _ = rdb.Close() }()

	authInterceptor := interceptor.NewAuthInterceptor(jwtSecret, nil)
	adminInterceptor := interceptor.NewAdminInterceptor(platformSvc, log)
	rateLimitInterceptor := interceptor.NewRateLimitInterceptor(cfg.RateLimitLoginPerMinute, cfg.RateLimitEnabled)

	hub := mthub.NewHub()
	eventBroker := mthub.NewOrderEventBroker()
	accountBroker := mthub.NewAccountProfitBroker()
	snapshotBroker := mthub.NewPositionSnapshotBroker()
	barBroker := mthub.NewBarBroker()
	tickBroker := mthub.NewTickBroker(64, log)
	tradeBroker := mthub.NewTradeBroker(64, log)
	statusBroker := mthub.NewAccountStatusBroker()
	idemGuard := mthub.NewIdempotencyGuard(rdb.Client())
	reconcileGate := mthub.NewReconcileGate()
	var reconLoop *mthub.ReconciliationLoop // H17: declared early so OnBrokerInfo callback can trigger reconciliation
	js, err := nc.JetStream()
	if err != nil {
		log.Fatal("nats jetstream failed", zap.Error(err))
	}
	eventStore := mthub.NewTradeEventStore(js)
	mthubSvc := mthub.NewMtHubService(hub, eventBroker, accountBroker, snapshotBroker, idemGuard, reconcileGate, eventStore)
	mthubSvc.SetLogger(log)
	mthubSvc.SetBarBroker(barBroker)
	mthubSvc.SetTickBroker(tickBroker)
	mthubSvc.SetTradeBroker(tradeBroker)
	mthubSvc.SetStatusBroker(statusBroker)

	// --- Analytics cache ---
	analyticsCache := service.NewAnalyticsCache(rdb.Client(), log)

	// --- mdgateway pipeline (M10 runner) ---
	tradeRecordRepo := repository.NewTradeRecordRepository(pool)
	accountSyncSvc := service.NewAccountSyncService(tradeRecordRepo, mthubSvc, analyticsCache, log)
	accountSyncSvc.SetScheduleResolver(repository.NewStrategyScheduleRepository(pool))

	spillDir := cfg.SpillDir
	pipelineCtx, pipelineCancel := context.WithCancel(context.Background())
	defer pipelineCancel()

	// Phase 0.2: Position snapshot persistence for mtapi disconnection display.
	snapshotPersister := mthub.NewSnapshotPersister(snapshotBroker, pool, log)
	go snapshotPersister.Start(pipelineCtx)

	var emailNotifier *notifier.EmailNotifier   // set after creation; referenced by OnAccountProfit closure
	var platformAgg *risksvc.PlatformAggregator // set after creation; referenced by OnOrderUpdate closure
	var notifSender *notifpubsub.Sender         // set after creation; referenced by CheckMarginCall closure
	var workerCleanup func()                    // set after creation; calls worker.Stop() on shutdown
	var scheduleEngine *strategy.ScheduleEngine // set after creation; started below
	var chainMonitor *chain.Monitor             // set after creation; started below
	var reconcilerInst *reconcile.Reconciler    // set after creation; started below
	var sweepWorker *sweep.Worker               // set after creation; started below

	// M12-C2: multi-broker registry created early so both handler wiring
	// and the mdgateway pipeline can reference the same instance.
	brokerReg := adapter.NewBrokerRegistry()
	mthubSvc.SetBrokerRegistry(brokerReg)

	mktplaceSvc := marketplace.New(pool, nil, log)

	livePerfCollector := marketplace.NewLivePerformanceCollector(mktplaceSvc, log)
	mktplaceSvc.SetLivePerfCollector(livePerfCollector)
	go func() {
		_ = startMdGatewayPipeline(mdGatewayPipelineDeps{
			pipelineCtx:       pipelineCtx,
			log:               log,
			pool:              pool,
			store:             mdStore,
			nc:                nc,
			rdb:               rdb.Client(),
			spillDir:          spillDir,
			secClient:         secClient,
			hub:               hub,
			accountSvc:        accountSvc,
			mthubSvc:          mthubSvc,
			accountSyncSvc:    accountSyncSvc,
			tradeRecordRepo:   tradeRecordRepo,
			snapshotBroker:    snapshotBroker,
			accountBroker:     accountBroker,
			barBroker:         barBroker,
			eventStore:        eventStore,
			emailNotifier:     &emailNotifier,
			platformAgg:       &platformAgg,
			reconLoop:         &reconLoop,
			brokerReg:         brokerReg,
			mtapiMT4Host:      cfg.MtapiMT4Host,
			mtapiMT5Host:      cfg.MtapiMT5Host,
			livePerfCollector: livePerfCollector,
			scheduleResolver:  repository.NewStrategyScheduleRepository(pool),
		})
	}()

	// Graceful shutdown context — created before registerHandlers so background
	// goroutines spawned there can observe shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ADR-0026 R5: Verify xpub fingerprint at startup to detect key substitution.
	// Resolve xpub from DB (system_config) first, fall back to env var.
	// DB is the canonical source — survives server migration (PG volume), editable via admin UI.
	resolveDepositXpub(context.Background(), pool, cfg, log)
	if cfg.DepositXpub != "" {
		fp, err := hdwallet.XpubFingerprint(cfg.DepositXpub)
		if err != nil {
			log.Fatal("startup: invalid deposit xpub — refusing to start", zap.Error(err))
		}
		if cfg.DepositXpubFingerprint != "" && fp != cfg.DepositXpubFingerprint {
			log.Fatal("startup: xpub fingerprint mismatch — potential key substitution",
				zap.String("expected", cfg.DepositXpubFingerprint),
				zap.String("actual", fp),
			)
		}
		log.Info("startup: deposit xpub verified", zap.String("fingerprint", fp))
	}

	mux := http.NewServeMux()
	reconLoop, emailNotifier, platformAgg, notifSender, scheduleEngine, workerCleanup, chainMonitor, reconcilerInst, sweepWorker = registerHandlers(ctx, handlerDeps{
		Mux: mux, Log: log, Pool: pool, Store: mdStore, NC: nc, RDB: rdb, Cfg: cfg,
		JWTSecret: jwtSecret, AccountSvc: accountSvc, PlatformSvc: platformSvc,
		AuthInterceptor: authInterceptor, AdminInterceptor: adminInterceptor,
		RateLimitInterceptor: rateLimitInterceptor, OtelInterceptor: otelInterceptor,
		MthubSvc: mthubSvc, Hub: hub, TradeRecordRepo: tradeRecordRepo, JS: js,
		EventStore: eventStore, ReconcileGate: reconcileGate, AnalyticsCache: analyticsCache,
		BrokerReg: brokerReg, SecClient: secClient, MktplaceSvc: mktplaceSvc,
	})
	accountSyncSvc.SetNotificationSender(notifSender)
	mktplaceSvc.SetNotificationSender(notifSender)

	// Task 2 (SUBMIT-STUCK-RACE): wire reconciliation trigger so
	// TransitionOrderByTicket can fall back to reconciliation when
	// a broker fill event arrives before ticket backfill.
	mthubSvc.SetReconcileTrigger(reconLoop.TriggerReconcile)

	go func() { _ = scheduleEngine.Start(ctx) }()
	defer workerCleanup()

	// Start reconciliation loop (cancelled on shutdown)
	go reconLoop.Start(ctx)

	// Start chain monitor for USDT deposit detection (cancelled on shutdown).
	// Gated by CHAIN_MONITOR_ENABLED — TRON track currently paused.
	if cfg.ChainMonitorEnabled {
		go func() { _ = chainMonitor.Run(ctx) }()
	} else {
		log.Info("chain monitor disabled by CHAIN_MONITOR_ENABLED=false")
	}

	// Start deposit reconciler + sweep worker — both touch TronGrid, so they
	// share the CHAIN_MONITOR_ENABLED gate while the TRON track is paused.
	if cfg.ChainMonitorEnabled {
		go func() { _ = reconcilerInst.Run(ctx) }()
		if sweepWorker != nil {
			go func() { _ = sweepWorker.Run(ctx) }()
		}
	}

	// Daily data retention cleanup — prevents unbounded disk growth.
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				cleanCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				_ = accountSvc.CleanupOldSnapshots(cleanCtx, log)
				cancel()
			case <-ctx.Done():
				return
			}
		}
	}()

	port := cfg.Port
	log.Info("ant v2 starting", zap.String("port", port), zap.String("nats", cfg.NATSURL))

	go func() {
		<-ctx.Done()
		log.Info("shutting down")
		pipelineCancel()
	}()

	// Wrap with Sentry panic recovery — captures panics in all HTTP handlers.
	sentryHandler := sentryhttp.New(sentryhttp.Options{Repanic: false, WaitForDelivery: true})
	sentryWrapped := sentryHandler.Handle(mux)

	// Wrap with SSE stream limit before keepalive to enforce per-user max streams.
	limitedHandler := interceptor.SSEStreamLimitMiddleware(5)(sentryWrapped)

	// Wrap with SSE keepalive to prevent Cloudflare/nginx from closing idle streams.
	keepaliveHandler := interceptor.SSEKeepaliveMiddleware(10 * time.Second)(limitedHandler)
	if err := server.Run(ctx, keepaliveHandler, port, log); err != nil {
		log.Fatal("server failed", zap.Error(err))
	}

}

func initInfrastructure(cfg *config.Config, log *zap.Logger, pool *pgxpool.Pool, secClient secrets.Client) (
	nc *nats.Conn, rdb *antredis.Client,
	accountSvc *service.AccountService,
	platformSvc *service.PlatformService, jwtSecret string,
	mdStore repository.MarketDataStore,
) {
	var err error
	if err := repository.MigrateScheduleProtoColumns(context.Background(), pool); err != nil {
		log.Warn("schedule proto migration skipped", zap.Error(err))
	}
	if err := repository.MigrateNotificationDataProto(context.Background(), pool); err != nil {
		log.Warn("notification data proto migration skipped", zap.Error(err))
	}

	pgStore := repository.NewPgMarketDataStore(pool, log)
	mdStore = pgStore
	repository.EnsureMarketDataPartitions(context.Background(), pool, log)

	nc, err = nats.Connect(cfg.NATSURL)
	if err != nil {
		log.Fatal("nats connect failed", zap.Error(err))
	}

	redisCfg := antredis.Config{
		Host: cfg.RedisHost, Port: 6379, Password: cfg.RedisPassword,
		DB: 0, PoolSize: 10, MinIdleConns: 3, MaxRetries: 3,
		DialTimeout: 5 * time.Second, ReadTimeout: 3 * time.Second,
	}
	if p := cfg.RedisPort; p != "" {
		_, _ = fmt.Sscanf(p, "%d", &redisCfg.Port)
	}
	rdb, err = antredis.Connect(context.Background(), redisCfg)
	if err != nil {
		log.Fatal("redis connect failed", zap.Error(err))
	}
	pgStore.SetRedisClient(rdb.Client())

	accountSvc = service.NewAccountService(pool, secClient)
	accountSvc.SetLogger(log)
	if n, err := accountSvc.BackfillPlaintextCredentials(context.Background()); err != nil {
		log.Warn("account backfill failed", zap.Error(err))
	} else if n > 0 {
		log.Info("account backfill migrated plaintext credentials", zap.Int("count", n))
	}
	platformSvc = service.NewPlatformService(pool, accountSvc)
	platformSvc.SetLogger(log)
	jwtSecret = cfg.JWTSecret
	return
}
