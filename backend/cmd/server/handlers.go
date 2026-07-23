package main

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	internalai "alphaforge/internal/ai"
	"alphaforge/internal/chain"
	"alphaforge/internal/config"
	algo "alphaforge/internal/connect/algo"
	"alphaforge/internal/connect/notification"
	strategy "alphaforge/internal/connect/strategy"
	subscriptionhdr "alphaforge/internal/connect/subscription"
	"alphaforge/internal/connect/system"
	"alphaforge/internal/connect/user"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/marketplace"
	"alphaforge/internal/mdgateway"
	"alphaforge/internal/mdgateway/adapter"
	"alphaforge/internal/mdgateway/adapter/brokersearch"
	"alphaforge/internal/mthub"
	notifpubsub "alphaforge/internal/notification"
	"alphaforge/internal/notifier"
	"alphaforge/internal/reconcile"
	"alphaforge/internal/repository"
	"alphaforge/internal/risk"
	"alphaforge/internal/risksvc"
	"alphaforge/internal/service"
	usersvc "alphaforge/internal/service/user"
	antredis "alphaforge/internal/storage/redis"

	"alphaforge/internal/secrets"
	alphasentry "alphaforge/internal/sentry"
	"alphaforge/internal/sweep"

	connectrpc "connectrpc.com/connect"
)

// withSency prepends the Sentry error capture interceptor to the chain.
// This avoids modifying every WithInterceptors call site individually.
func withSency(interceptors ...connectrpc.Interceptor) connectrpc.Option {
	return connectrpc.WithInterceptors(append([]connectrpc.Interceptor{alphasentry.NewErrorInterceptor()}, interceptors...)...)
}

func registerHandlers(
	ctx context.Context,
	mux *http.ServeMux,
	log *zap.Logger,
	pool *pgxpool.Pool,
	store repository.MarketDataStore,
	nc *nats.Conn,
	rdb *antredis.Client,
	cfg *config.Config,
	jwtSecret string,
	accountSvc *service.AccountService,
	platformSvc *service.PlatformService,
	authInterceptor *interceptor.AuthInterceptor,
	adminInterceptor *interceptor.AdminInterceptor,
	rateLimitInterceptor *interceptor.RateLimitInterceptor,
	otelInterceptor connectrpc.Interceptor,
	mthubSvc *mthub.MtHubService,
	hub *mthub.Hub,
	tradeRecordRepo *repository.TradeRecordRepository,
	js nats.JetStreamContext,
	eventStore *mthub.TradeEventStore,
	reconcileGate *mthub.ReconcileGate,
	analyticsCache *service.AnalyticsCache,
	brokerReg *adapter.BrokerRegistry,
	secClient secrets.Client,
	mktplaceSvc *marketplace.Service,
) (*mthub.ReconciliationLoop, *notifier.EmailNotifier, *risksvc.PlatformAggregator, *notifpubsub.Sender, *strategy.ScheduleEngine, func(), *chain.Monitor, *reconcile.Reconciler, *sweep.Worker) {

	// ConnectRPC handlers
	// Repositories for handler→service→repository layering (P1-2).
	userRepo := repository.NewUserRepository(pool)
	convRepo := repository.NewAIConversationRepository(pool)
	session := internalai.NewConversationSession(convRepo)
	templatesRepo := repository.NewAIStrategyTemplatesRepository(pool)
	jobRepo := repository.NewJobRepository(pool)
	schedHealthRepo := repository.NewScheduleHealthRepository(pool)
	marketDataRepo := store
	walletRepo := repository.NewWalletRepository(pool)
	walletSvc := service.NewWalletService(walletRepo, pool, log)
	accountNumberSvc := usersvc.NewAccountNumberService(pool)
	registrationSvc := service.NewRegistrationService(userRepo, accountNumberSvc, walletSvc, log)

	// Create email notifier early so it can be shared between auth and SRE handlers.
	emailNotifier := notifier.NewEmailNotifier(notifier.EmailConfig{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		User:     cfg.SMTPUser,
		Password: cfg.SMTPPassword,
		From:     cfg.SMTPFrom,
		To:       splitAndTrim(cfg.SMTPTo, ","),
	}, log)

	// Create shared broker searcher for account binding and §0 host rediscovery.
	searcher := brokersearch.New("", "")

	// Create HostRediscoverer for §0 broker host lazy rediscovery on connection failure.
	rediscoverer := mdgateway.NewHostRediscoverer(searcher, pool, log)

	// Create MT connection tester early so it can be shared between auth and account handlers.
	mtTester := user.NewMTConnectionTester(cfg.MtapiToken, rediscoverer, log)

	authServer := registerAuthHandler(mux, pool, cfg, jwtSecret, userRepo, registrationSvc, emailNotifier, mtTester, log, otelInterceptor, rateLimitInterceptor, authInterceptor)

	registerWalletHandler(mux, walletSvc, platformSvc, log, otelInterceptor, authInterceptor)

	// Deposit + chain monitor + reconciler + sweep worker (ADR-0026 Phase C).
	dsDeps := setupDepositAndSweep(mux, pool, cfg, walletRepo, platformSvc, log, otelInterceptor, authInterceptor)

	// NOTE: WebAuthn withdrawal authorization temporarily disabled (see rd.md for rationale).
	// wireWebAuthn(mux, pool, log, cfg, walletSvc, walletRepo, emailNotifier, platformSvc,
	// 	dsDeps.sweepBundleRepo, dsDeps.sweepTronClient, dsDeps.adminRepo, otelInterceptor, authInterceptor)

	// P3.1: Subscription service (Free/Pro/Enterprise plans).
	subscriptionRepo := repository.NewSubscriptionRepository(pool)
	subscriptionSvc := service.NewSubscriptionService(subscriptionRepo, walletSvc, pool, log)
	subscriptionSvc.SetUsageRepos(repository.NewAITokenUsageRepository(pool), repository.NewStrategyRunRepository(pool))
	registrationSvc.SetSubscriptionEnsurer(subscriptionSvc)
	subscriptionServer := subscriptionhdr.NewServer(subscriptionSvc, subscriptionSvc, log)
	mux.Handle(antv1c.NewSubscriptionServiceHandler(subscriptionServer, withSency(otelInterceptor, authInterceptor)))

	// P3.1: QuotaChecker — in-memory cache for fast quota lookups.
	quotaChecker := service.NewQuotaChecker(subscriptionRepo, pool, log)
	quotaChecker.LoadAll(ctx)
	quotaChecker.StartRefreshLoop(ctx, 5*time.Minute)

	reconLoop := mthub.NewReconciliationLoop(hub, pool, rdb.Client(), log, reconcileGate)

	mthubServer := system.NewMtHubServer(mthubSvc, platformSvc, marketDataRepo, tradeRecordRepo, log)
	mux.Handle(antv1c.NewMtHubServiceHandler(mthubServer, withSency(otelInterceptor, authInterceptor)))

	accountEventPub := mdgateway.NewAccountEventPublisher(js, log)
	registerAccountHandler(mux, cfg, accountSvc, accountEventPub, hub, mtTester, searcher, log, otelInterceptor, authInterceptor)

	mktplaceHandler := registerMarketplaceHandlers(ctx, mux, nc, log, marketDataRepo, mktplaceSvc, walletRepo, platformSvc, otelInterceptor, authInterceptor)

	subscriptionSvc.StartPlatformRenewalLoop(ctx) // daily platform subscription auto-renewal/expiry

	// M12-A2: Execution Algo handler (TWAP/VWAP/POV/Shortfall).
	// brokerReg is created in main.go before the pipeline starts; gateways register
	// via adapter.RegisterDefaults inside the mdgateway runner after connection.
	algoServer := algo.NewExecutionAlgoServer(brokerReg, log)
	mux.Handle(antv1c.NewExecutionAlgoServiceHandler(algoServer, withSency(otelInterceptor, authInterceptor)))

	logRepo := repository.NewLogRepository(pool)
	logSvc := service.NewLogService(logRepo)
	strategyExperimentRepo := repository.NewStrategyExperimentRepository(pool)
	strategyAssetRepo := repository.NewStrategyAssetRepository(pool)
	backtestRunRepo := repository.NewBacktestRunRepository(pool)

	// AI services: system AI, asset analysis, AI gateway, agent gateway, code assist, strategy plan.
	aiDeps := setupAIServices(ctx, mux, pool, cfg, userRepo, marketDataRepo, platformSvc, mktplaceSvc,
		mktplaceHandler, quotaChecker, walletSvc, convRepo, session, backtestRunRepo, log, otelInterceptor, authInterceptor)

	// Share performance: generate expiring public links for trading results.
	registerShareHandlers(mux, pool, log, tradeRecordRepo, userRepo, mthubSvc,
		jwtSecret, otelInterceptor, authInterceptor)

	streamServer := system.NewStreamServer(mthubSvc, platformSvc, log)
	streamServer.SetMarketDataRepo(marketDataRepo)
	mux.Handle(antv1c.NewStreamServiceHandler(streamServer, withSency(otelInterceptor, authInterceptor)))

	// Strategy + paper + risk + autotrading + schedule engine.
	stratDeps := setupStrategyAndTrading(ctx, mux, pool, cfg, marketDataRepo, mthubSvc, hub, eventStore,
		aiDeps.aiSvc, mktplaceSvc, mktplaceHandler, quotaChecker, templatesRepo, backtestRunRepo,
		log, otelInterceptor, authInterceptor)

	// Phase 2.2: Batch generator — PG NOTIFY-driven AI strategy generation queue.
	if aiDeps.agentGateway.Generator() != nil {
		batchGen := marketplace.NewBatchGenerator(pool, log, aiDeps.agentGateway.Generator(), stratDeps.pgListen, mktplaceSvc)
		mktplaceHandler.SetBatchGenerator(batchGen)
		batchGen.Start(ctx)
	}

	// System service registrations.
	economicDataServer := system.NewEconomicDataServer(log)
	mux.Handle(antv1c.NewEconomicDataServiceHandler(economicDataServer, withSency(otelInterceptor, authInterceptor)))
	jobServer := system.NewJobServer(jobRepo, log)
	jobServer.SetPgListen(stratDeps.pgListen)
	mux.Handle(antv1c.NewJobServiceHandler(jobServer, withSency(otelInterceptor, authInterceptor)))
	logServiceServer := system.NewLogServiceServer(logSvc, log)
	mux.Handle(antv1c.NewLogServiceHandler(logServiceServer, withSency(otelInterceptor, authInterceptor)))
	notifServer := notification.NewNotificationServer(
		repository.NewNotificationRepository(pool), notifpubsub.NewSubscriber(), pool, log)
	mux.Handle(antv1c.NewNotificationServiceHandler(notifServer, withSency(otelInterceptor, authInterceptor)))
	aiDeps.gateEvalServer.SetNotificationSender(stratDeps.notifSender)

	gate := risk.NewDefaultGate()
	gate.SetKillSwitch(func() bool { return cfg.RiskGateKillSwitch })
	gate.SetAutotradeEnabled(func(uid string) bool { return cfg.RiskGateAutotradeEnabled })

	registerAdminHandlers(mux, pool, log, walletSvc, accountNumberSvc, service.NewStrategySvc(pool),
		aiDeps.agentGateway.SettingsStore(), aiDeps.agentGateway.HookEngine(),
		accountEventPub,
		rdb, nc,
		otelInterceptor, authInterceptor, adminInterceptor)

	emailNotifier, workerCleanup := registerSREHandlers(
		userRepo, mux, log, pool, store, nc, rdb, cfg,
		authInterceptor, otelInterceptor, platformSvc, mthubSvc,
		authServer,
		strategyExperimentRepo, strategyAssetRepo, schedHealthRepo,
		analyticsCache,
		aiDeps.aiSvc,
		backtestRunRepo,
		stratDeps.pgListen,
		emailNotifier,
	)

	startBackgroundServices(ctx, pool, emailNotifier, dsDeps.depositAddrRepo, dsDeps.adminRepo, dsDeps.depositSvc, cfg, log)

	return reconLoop, emailNotifier, stratDeps.platformAgg, stratDeps.notifSender, stratDeps.scheduleEngine, workerCleanup, dsDeps.chainMonitor, dsDeps.reconcilerInst, dsDeps.sweepWorker
}
