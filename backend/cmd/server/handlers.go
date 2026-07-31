package main

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
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
	"alphaforge/internal/marketplace"
	"alphaforge/internal/mdgateway"
	"alphaforge/internal/mdgateway/adapter/brokersearch"
	"alphaforge/internal/mthub"
	notifpubsub "alphaforge/internal/notification"
	"alphaforge/internal/notifier"
	"alphaforge/internal/pglisten"
	"alphaforge/internal/reconcile"
	"alphaforge/internal/repository"
	"alphaforge/internal/risk"
	"alphaforge/internal/risksvc"
	"alphaforge/internal/service"
	usersvc "alphaforge/internal/service/user"

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
	d handlerDeps,
) (*mthub.ReconciliationLoop, *notifier.EmailNotifier, *risksvc.PlatformAggregator, *notifpubsub.Sender, *strategy.ScheduleEngine, func(), *chain.Monitor, *reconcile.Reconciler, *sweep.Worker) {
	mux := d.Mux
	log := d.Log
	pool := d.Pool
	cfg := d.Cfg

	// ConnectRPC handlers
	// Repositories for handler→service→repository layering (P1-2).
	userRepo := repository.NewUserRepository(pool)
	convRepo := repository.NewAIConversationRepository(pool)
	session := internalai.NewConversationSession(convRepo)
	templatesRepo := repository.NewAIStrategyTemplatesRepository(pool)
	jobRepo := repository.NewJobRepository(pool)
	schedHealthRepo := repository.NewScheduleHealthRepository(pool)
	marketDataRepo := d.Store
	walletRepo := repository.NewWalletRepository(pool)
	walletSvc := service.NewWalletService(walletRepo, pool, log)
	accountNumberSvc := usersvc.NewAccountNumberService(pool)
	registrationSvc := service.NewRegistrationService(userRepo, accountNumberSvc, walletSvc, log)
	emailNotifier := newEmailNotifier(cfg, log)
	searcher := brokersearch.New("", "")
	rediscoverer := mdgateway.NewHostRediscoverer(searcher, pool, log)
	mtTester := user.NewMTConnectionTester(cfg.MtapiToken, rediscoverer, log)

	authServer := registerAuthHandler(authHandlerParams{
		Mux: mux, Pool: pool, Cfg: cfg, JWTSecret: d.JWTSecret,
		UserRepo: userRepo, RegistrationSvc: registrationSvc,
		EmailNotifier: emailNotifier, MTTester: mtTester, Log: log,
		OtelInterceptor: d.OtelInterceptor, RateLimitInterceptor: d.RateLimitInterceptor,
		AuthInterceptor: d.AuthInterceptor,
	})

	registerWalletHandler(mux, walletSvc, d.PlatformSvc, log, d.OtelInterceptor, d.AuthInterceptor)

	// Deposit + chain monitor + reconciler + sweep worker (ADR-0026 Phase C).
	dsDeps := setupDepositAndSweep(mux, pool, cfg, walletRepo, d.PlatformSvc, log, d.OtelInterceptor, d.AuthInterceptor)

	// NOTE: WebAuthn withdrawal authorization temporarily disabled (see rd.md for rationale).
	// wireWebAuthn(mux, pool, log, cfg, walletSvc, walletRepo, emailNotifier, platformSvc,
	// 	dsDeps.sweepBundleRepo, dsDeps.sweepTronClient, dsDeps.adminRepo, otelInterceptor, authInterceptor)

	// P3.1: Subscription service + QuotaChecker.
	subscriptionSvc, quotaChecker := setupSubscription(ctx, mux, pool, log, walletSvc, registrationSvc, d.OtelInterceptor, d.AuthInterceptor)

	reconLoop := mthub.NewReconciliationLoop(d.Hub, pool, d.RDB.Client(), log, d.ReconcileGate)

	mthubServer := system.NewMtHubServer(d.MthubSvc, d.PlatformSvc, marketDataRepo, d.TradeRecordRepo, log)
	mux.Handle(antv1c.NewMtHubServiceHandler(mthubServer, withSency(d.OtelInterceptor, d.AuthInterceptor)))

	accountEventPub := mdgateway.NewAccountEventPublisher(d.JS, log)
	registerAccountHandler(mux, cfg, d.AccountSvc, accountEventPub, d.Hub, mtTester, searcher, quotaChecker, log, d.OtelInterceptor, d.AuthInterceptor)

	mktplaceHandler := registerMarketplaceHandlers(ctx, mux, d.NC, log, marketDataRepo, d.MktplaceSvc, walletRepo, d.PlatformSvc, d.OtelInterceptor, d.AuthInterceptor)

	subscriptionSvc.StartPlatformRenewalLoop(ctx) // daily platform subscription auto-renewal/expiry

	// M12-A2: Execution Algo handler (TWAP/VWAP/POV/Shortfall).
	// brokerReg is created in main.go before the pipeline starts; gateways register
	// via adapter.RegisterDefaults inside the mdgateway runner after connection.
	algoServer := algo.NewExecutionAlgoServer(d.BrokerReg, log)
	mux.Handle(antv1c.NewExecutionAlgoServiceHandler(algoServer, withSency(d.OtelInterceptor, d.AuthInterceptor)))

	logRepo := repository.NewLogRepository(pool)
	logSvc := service.NewLogService(logRepo)
	strategyExperimentRepo := repository.NewStrategyExperimentRepository(pool)
	strategyAssetRepo := repository.NewStrategyAssetRepository(pool)
	backtestRunRepo := repository.NewBacktestRunRepository(pool)

	// AI services: system AI, asset analysis, AI gateway, agent gateway, code assist, strategy plan.
	aiDeps := setupAIServices(aiServicesParams{
		Ctx: ctx, Mux: mux, Pool: pool, Cfg: cfg, UserRepo: userRepo,
		MarketDataRepo: marketDataRepo, PlatformSvc: d.PlatformSvc, MktplaceSvc: d.MktplaceSvc,
		MktplaceHandler: mktplaceHandler, QuotaChecker: quotaChecker, WalletSvc: walletSvc,
		ConvRepo: convRepo, Session: session, BacktestRunRepo: backtestRunRepo,
		Log: log, OtelInterceptor: d.OtelInterceptor, AuthInterceptor: d.AuthInterceptor,
	})

	// Share performance: generate expiring public links for trading results.
	registerShareHandlers(mux, pool, log, d.TradeRecordRepo, userRepo, d.MthubSvc,
		d.JWTSecret, d.OtelInterceptor, d.AuthInterceptor)

	streamServer := system.NewStreamServer(d.MthubSvc, d.PlatformSvc, log)
	streamServer.SetMarketDataRepo(marketDataRepo)
	mux.Handle(antv1c.NewStreamServiceHandler(streamServer, withSency(d.OtelInterceptor, d.AuthInterceptor)))

	// Strategy + paper + risk + autotrading + schedule engine.
	stratDeps := setupStrategyAndTrading(strategyTradingParams{
		Ctx: ctx, Mux: mux, Pool: pool, Cfg: cfg, MarketDataRepo: marketDataRepo,
		MthubSvc: d.MthubSvc, Hub: d.Hub, EventStore: d.EventStore,
		AISvc: aiDeps.aiSvc, MktplaceSvc: d.MktplaceSvc, MktplaceHandler: mktplaceHandler,
		QuotaChecker: quotaChecker, TemplatesRepo: templatesRepo, BacktestRunRepo: backtestRunRepo,
		Log: log, OtelInterceptor: d.OtelInterceptor, AuthInterceptor: d.AuthInterceptor,
	})

	// Phase 2.2: Batch generator — PG NOTIFY-driven AI strategy generation queue.
	if aiDeps.agentGateway.Generator() != nil {
		batchGen := marketplace.NewBatchGenerator(pool, log, aiDeps.agentGateway.Generator(), stratDeps.pgListen, d.MktplaceSvc)
		mktplaceHandler.SetBatchGenerator(batchGen)
		batchGen.Start(ctx)
	}

	// System service registrations.
	registerSystemServices(mux, pool, log, jobRepo, logSvc, stratDeps.pgListen, d.OtelInterceptor, d.AuthInterceptor)
	aiDeps.gateEvalServer.SetNotificationSender(stratDeps.notifSender)

	gate := risk.NewDefaultGate()
	gate.SetKillSwitch(func() bool { return cfg.RiskGateKillSwitch })
	gate.SetAutotradeEnabled(func(uid string) bool { return cfg.RiskGateAutotradeEnabled })

	registerAdminHandlers(adminHandlerDeps{
		Mux: mux, Pool: pool, Log: log, WalletSvc: walletSvc,
		AccountNumberSvc: accountNumberSvc, StrategySvc: service.NewStrategySvc(pool),
		SettingsStore: aiDeps.agentGateway.SettingsStore(), HookEngine: aiDeps.agentGateway.HookEngine(),
		AccountEventPub: accountEventPub, RDB: d.RDB, NC: d.NC,
		Interceptors: interceptorSet{otel: d.OtelInterceptor, auth: d.AuthInterceptor, admin: d.AdminInterceptor},
	})

	emailNotifier, workerCleanup := registerSREHandlers(sreHandlerParams{
		UserRepo: userRepo, Mux: mux, Log: log, Pool: pool, Store: d.Store,
		NC: d.NC, RDB: d.RDB, Cfg: cfg, AuthInterceptor: d.AuthInterceptor,
		OtelInterceptor: d.OtelInterceptor, PlatformSvc: d.PlatformSvc, MthubSvc: d.MthubSvc,
		AuthServer: authServer, StrategyExperimentRepo: strategyExperimentRepo,
		StrategyAssetRepo: strategyAssetRepo, SchedHealthRepo: schedHealthRepo,
		AnalyticsCache: d.AnalyticsCache, AISvc: aiDeps.aiSvc, BacktestRunRepo: backtestRunRepo,
		PgListen: stratDeps.pgListen, EmailNotifier: emailNotifier,
	})

	startBackgroundServices(ctx, pool, emailNotifier, dsDeps.depositAddrRepo, dsDeps.adminRepo, dsDeps.depositSvc, cfg, log)

	return reconLoop, emailNotifier, stratDeps.platformAgg, stratDeps.notifSender, stratDeps.scheduleEngine, workerCleanup, dsDeps.chainMonitor, dsDeps.reconcilerInst, dsDeps.sweepWorker
}

func newEmailNotifier(cfg *config.Config, log *zap.Logger) *notifier.EmailNotifier {
	return notifier.NewEmailNotifier(notifier.EmailConfig{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		User:     cfg.SMTPUser,
		Password: cfg.SMTPPassword,
		From:     cfg.SMTPFrom,
		To:       splitAndTrim(cfg.SMTPTo, ","),
	}, log)
}

func setupSubscription(ctx context.Context, mux *http.ServeMux, pool *pgxpool.Pool, log *zap.Logger, walletSvc *service.WalletService, registrationSvc *service.RegistrationService, otel, auth connectrpc.Interceptor) (*service.SubscriptionService, *service.QuotaChecker) {
	subscriptionRepo := repository.NewSubscriptionRepository(pool)
	subscriptionSvc := service.NewSubscriptionService(subscriptionRepo, walletSvc, pool, log)
	subscriptionSvc.SetUsageRepos(repository.NewAITokenUsageRepository(pool), repository.NewStrategyRunRepository(pool))
	registrationSvc.SetSubscriptionEnsurer(subscriptionSvc)
	subscriptionServer := subscriptionhdr.NewServer(subscriptionSvc, subscriptionSvc, log)
	mux.Handle(antv1c.NewSubscriptionServiceHandler(subscriptionServer, withSency(otel, auth)))
	quotaChecker := service.NewQuotaChecker(subscriptionRepo, pool, log)
	_ = quotaChecker.LoadAll(ctx)
	return subscriptionSvc, quotaChecker
}

func registerSystemServices(mux *http.ServeMux, pool *pgxpool.Pool, log *zap.Logger, jobRepo *repository.JobRepository, logSvc *service.LogService, pgListen *pglisten.Listener, otel, auth connectrpc.Interceptor) {
	economicDataServer := system.NewEconomicDataServer(log)
	mux.Handle(antv1c.NewEconomicDataServiceHandler(economicDataServer, withSency(otel, auth)))
	jobServer := system.NewJobServer(jobRepo, log)
	jobServer.SetPgListen(pgListen)
	mux.Handle(antv1c.NewJobServiceHandler(jobServer, withSency(otel, auth)))
	logServiceServer := system.NewLogServiceServer(logSvc, log)
	mux.Handle(antv1c.NewLogServiceHandler(logServiceServer, withSency(otel, auth)))
	notifServer := notification.NewNotificationServer(
		repository.NewNotificationRepository(pool), notifpubsub.NewSubscriber(), pool, log)
	mux.Handle(antv1c.NewNotificationServiceHandler(notifServer, withSency(otel, auth)))
}
