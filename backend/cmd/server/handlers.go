package main

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/agent"
	internalai "alphaforge/internal/ai"
	"alphaforge/internal/analysis"
	"alphaforge/internal/chain"
	"alphaforge/internal/config"
	"alphaforge/internal/connect/admin"
	"alphaforge/internal/connect/ai"
	algo "alphaforge/internal/connect/algo"
	assetanalysis "alphaforge/internal/connect/asset_analysis"
	"alphaforge/internal/connect/autotrading"
	"alphaforge/internal/connect/gateway"
	mktplace "alphaforge/internal/connect/marketplace"
	"alphaforge/internal/connect/notification"
	paperhdr "alphaforge/internal/connect/paper"
	"alphaforge/internal/connect/strategy"
	subscriptionhdr "alphaforge/internal/connect/subscription"
	"alphaforge/internal/connect/system"
	"alphaforge/internal/connect/user"
	"alphaforge/internal/reconcile"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/marketplace"
	"alphaforge/internal/mdgateway"
	"alphaforge/internal/mdgateway/adapter"
	"alphaforge/internal/mdgateway/adapter/brokersearch"
	"alphaforge/internal/mthub"
	notifpubsub "alphaforge/internal/notification"
	"alphaforge/internal/notifier"
	papereng "alphaforge/internal/paper"
	"alphaforge/internal/pglisten"
	"alphaforge/internal/pkg/secretbox"
	"alphaforge/internal/repository"
	"alphaforge/internal/risk"
	"alphaforge/internal/risksvc"
	"alphaforge/internal/service"
	systemai "alphaforge/internal/service/systemai"
	usersvc "alphaforge/internal/service/user"
	antredis "alphaforge/internal/storage/redis"
	"alphaforge/internal/usermgr"

	alphasentry "alphaforge/internal/sentry"
	"alphaforge/internal/secrets"
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

	authServer := user.NewAuthServer(userRepo, jwtSecret, log)
	authServer.SetInsecureCookies(true) // no TLS in Docker deployment
	authServer.WithRegistration(registrationSvc)
	// Wire email verification if SMTP is configured.
	if emailNotifier != nil {
		emailVerifSvc := service.NewEmailVerificationService(pool, emailNotifier, cfg.AppURL, log)
		authServer.WithEmailVerification(emailVerifSvc)
		registrationSvc.SetEmailVerification(emailVerifSvc)
	}
	authServer.SetRequireEmailVerification(cfg.RequireEmailVerification)
	mux.Handle(antv1c.NewAuthServiceHandler(authServer, withSency(otelInterceptor, rateLimitInterceptor, authInterceptor)))

	walletServer := user.NewWalletServer(walletSvc, platformSvc, log)
	mux.Handle(antv1c.NewWalletServiceHandler(walletServer, withSency(otelInterceptor, authInterceptor)))

	// USDT deposit service: HD wallet with per-user addresses + auto-confirmation.
	depositAddrRepo := repository.NewDepositAddressRepository(pool)
	depositRepo := repository.NewDepositRepository(pool)
	depositSvc := service.NewDepositService(depositAddrRepo, depositRepo, walletRepo, pool, cfg.DepositXpub, log)

	// Chain monitor: scans TronGrid for USDT transfers to user deposit addresses.
	adminRepo := repository.NewAdminRepository(pool)
	tronGrid := chain.NewTronGridClient(cfg.TrongridAPIKey)
	tronScan := chain.NewTronScanClient(cfg.TronscanAPIKey)
	chainMonitor := chain.NewMonitor(tronGrid, tronScan, pool, depositAddrRepo, adminRepo, depositSvc, depositRepo, log)
	depositSvc.OnAddressClaimed = chainMonitor.RegisterAddress
	// Reconciler: two-phase deposit reconciliation (internal 6h + on-chain 24h).
	reconcileRepo := repository.NewReconcileRepository(pool)
	reconcilerInst := reconcile.NewReconciler(reconcileRepo, adminRepo, depositAddrRepo, tronGrid, log)
	// Sweep worker: builds unsigned bundles, broadcasts signed bundles, crash recovery (ADR-0026 Phase C).
	sweepLogRepo := repository.NewSweepLogRepository(pool)
	sweepTronClient, err := sweep.NewTronClient(cfg.TronGridGRPCEndpoint, cfg.TrongridAPIKey)
	if err != nil {
		log.Warn("sweep tron client: failed to connect (sweep worker disabled)", zap.Error(err))
	}
	var sweepWorker *sweep.Worker
	var sweepBundleRepo *sweep.BundleRepository
	if sweepTronClient != nil {
		sweepBuilder := sweep.NewBuilder(sweepTronClient, depositAddrRepo, adminRepo, log)
		sweepBroadcaster := sweep.NewBroadcaster(sweepTronClient, sweepLogRepo, depositAddrRepo, adminRepo, log)
		sweepState := sweep.NewStateMachine(sweepTronClient, sweepLogRepo, tronGrid, adminRepo, depositAddrRepo, log)
		sweepBundleRepo = sweep.NewBundleRepository(pool)
		sweepWorker = sweep.NewWorker(sweepBuilder, sweepBroadcaster, sweepState, sweepBundleRepo,
			sweepLogRepo, depositRepo, depositAddrRepo, adminRepo, pool, log)
	}

	// ADR-0026 Phase E: WebAuthn withdrawal authorization (after sweep setup for WithdrawalBuilder).
	wireWebAuthn(mux, pool, log, cfg, walletSvc, walletRepo, emailNotifier, platformSvc,
		sweepBundleRepo, sweepTronClient, adminRepo, otelInterceptor, authInterceptor)

	// Deposit server (with sweep worker wired in for admin sweep RPCs).
	depositServer := user.NewDepositServer(depositSvc, platformSvc, sweepWorker, log)
	mux.Handle(antv1c.NewDepositServiceHandler(depositServer, withSency(otelInterceptor, authInterceptor)))

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

	searcher := brokersearch.New("", "")
	accountEventPub := mdgateway.NewAccountEventPublisher(js, log)
	mtTester := user.NewMTConnectionTester(cfg.MtapiToken, log)
	accountServer := user.NewAccountServer(accountSvc, searcher, accountEventPub, mtTester, log).
		WithSessionWaiter(hub).
		WithStopGateway(hub.RemoveGateway)
	mux.Handle(antv1c.NewAccountServiceHandler(accountServer, withSency(otelInterceptor, authInterceptor)))

	mktServer := mktplace.NewMarketServer(platformSvc, marketDataRepo, nc, log)
	mux.Handle(antv1c.NewMarketServiceHandler(mktServer, withSency(otelInterceptor, authInterceptor)))

	mktplaceSvc.SetWalletRepo(walletRepo)
	mktplaceHandler := mktplace.NewMarketplaceServer(mktplaceSvc, platformSvc, log)
	mux.Handle(antv1c.NewMarketplaceServiceHandler(mktplaceHandler, withSency(otelInterceptor, authInterceptor)))

	mktplaceSvc.StartRenewalLoop(ctx, log) // daily subscription renewal (background goroutine)
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

	aiRepo := repository.NewSystemAIConfigRepository(pool)
	var aiBox *secretbox.Box
	if mk := cfg.AntMasterKey; mk != "" {
		aiBox = secretbox.New([]byte(mk))
	}
	aiSvc := systemai.NewService(aiRepo, aiBox)
	aiSvc.SetUserRepo(userRepo)
	aiSvc.SetCircuitBreakerDB(&pgxCB{p: pool}) // persistent, shared across instances
	agentDefRepo := repository.NewAIAgentDefinitionRepository(pool)
	aiServer := ai.NewAIServer(aiSvc, convRepo, session, log)
	aiServer.SetAgentDefRepo(agentDefRepo)
	mux.Handle(antv1c.NewAIServiceHandler(aiServer, withSency(otelInterceptor, authInterceptor)))
	// Agent definition CRUD (no proto RPC yet — raw HTTP).
	mux.Handle(antv1c.NewAgentDefinitionServiceHandler(aiServer, withSency(otelInterceptor, authInterceptor)))

	// P3: AI Asset Analysis — MTF outlook, S/R levels, volatility, AI recommendation.
	assetAnalyzer := analysis.NewAnalyzer(marketDataRepo, log)
	assetAnalysisServer := assetanalysis.NewAssetAnalysisServer(assetAnalyzer, aiSvc, platformSvc, log)
	mux.Handle(antv1c.NewAssetAnalysisServiceHandler(assetAnalysisServer, withSency(otelInterceptor, authInterceptor)))

	// Share performance: generate expiring public links for trading results.
	registerShareHandlers(mux, pool, log, tradeRecordRepo, userRepo, mthubSvc,
		jwtSecret, otelInterceptor, authInterceptor)

	// AI Gateway: platform-operated AI model relay with token billing.
	gatewayProviderRepo := repository.NewSystemAIProviderRepository(pool)
	gatewayModelRepo := repository.NewAIModelRepository(pool)
	gatewayTokenUsageRepo := repository.NewAITokenUsageRepository(pool)
	gatewayServer := gateway.NewAIGatewayServer(gatewayProviderRepo, gatewayModelRepo, gatewayTokenUsageRepo, walletSvc, aiBox, log)
	mux.Handle(antv1c.NewAIGatewayServiceHandler(gatewayServer, withSency(otelInterceptor, authInterceptor)))

	// Wire AI Gateway fallback: if user has no own API key, use system providers.
	aiSvc.SetGatewayProviderRepo(gatewayProviderRepo)

	wireAIBilling(aiSvc, walletSvc, gatewayServer, gatewayModelRepo, quotaChecker, gatewayTokenUsageRepo)

	// ADR-0024: Agent Gateway — strategy submission → compile → backtest → LLM analysis.
	agentGateway := agent.NewGatewayServer(pool, marketDataRepo, aiSvc, log)
	mux.Handle(antv1c.NewAgentGatewayServiceHandler(agentGateway, withSency(otelInterceptor, authInterceptor)))

	// ADR-0025 §8: Load persisted hook configs from DB at startup.
	if pool != nil && agentGateway.HookEngine() != nil {
		if err := admin.LoadHookConfigsFromDB(ctx, pool, agentGateway.HookEngine()); err != nil {
			log.Warn("failed to load hook configs from DB", zap.Error(err))
		}
	}

	// ADR-0025 §5.2: Wire model whitelist filter from managed settings into systemai.Service.
	if ss := agentGateway.SettingsStore(); ss != nil {
		aiSvc.SetModelFilter(func(ctx context.Context, userID uuid.UUID, model string) bool {
			rs, err := ss.ResolveSettings(ctx, userID)
			if err != nil || !rs.Loaded {
				return true // fail-open if settings can't load (fail-closed is handled elsewhere)
			}
			if !rs.Managed.EnforceAllowedModels {
				return true
			}
			if len(rs.Managed.AllowedModels) == 0 {
				return false // empty whitelist + enforce = deny all
			}
			for _, allowed := range rs.Managed.AllowedModels {
				if allowed == model {
					return true
				}
			}
			return false
		})
	}

	streamServer := system.NewStreamServer(mthubSvc, platformSvc, log)
	streamServer.SetMarketDataRepo(marketDataRepo)
	mux.Handle(antv1c.NewStreamServiceHandler(streamServer, withSency(otelInterceptor, authInterceptor)))

	strategySvc := service.NewStrategySvc(pool)
	strategyServer := strategy.NewStrategyServer(strategySvc, log)
	strategyServer.SetCodeAccessChecker(mktplaceSvc) // marketplace code protection
	pgListen := pglisten.New(pool, log)
	strategyServer.SetPgListen(pgListen)
	mktplaceHandler.SetPgListen(pgListen) // marketplace SSE streaming
	mux.Handle(antv1c.NewStrategyServiceHandler(strategyServer, withSency(otelInterceptor, authInterceptor)))

	// Paper trading + notification deps created early — both needed by strategy execution config.
	paperRepo := repository.NewPaperRepo(pool)
	guard := risk.NewGuard(&risk.GuardConfig{
		KillSwitch: func() bool { return cfg.RiskGateKillSwitch },
	})
	paperEngine := papereng.New(paperRepo, mthubSvc, log)
	paperEngine.SetGuard(guard)
	notifSub := notifpubsub.NewSubscriber()
	notifRepo := repository.NewNotificationRepository(pool)
	notifSender := notifpubsub.NewSender(notifRepo, notifSub, log)

	jurisGate, capStore, platformAgg := initRiskPipeline(pool, log, mthubSvc, hub, eventStore, cfg, guard)
	strategyExecServer := configureStrategyExecution(pool, backtestRunRepo, marketDataRepo, mthubSvc, hub,
		paperEngine, notifSender, aiSvc, pgListen, jurisGate, capStore, quotaChecker, cfg, log)
	mux.Handle(antv1c.NewStrategyRuntimeServiceHandler(strategyExecServer,
		withSency(otelInterceptor, authInterceptor)))

	paperHandler := paperhdr.NewHandler(paperRepo, paperEngine, strategyExecServer, log,
		func(ctx context.Context, userID string) string {
			var mt4ID string
			err := pool.QueryRow(ctx,
				`SELECT id::text FROM mt_accounts
				 WHERE user_id = $1::uuid AND account_status != 'frozen'
				 ORDER BY created_at LIMIT 1`,
				userID).Scan(&mt4ID)
			if err != nil {
				return ""
			}
			return mt4ID
		})
	mux.Handle(antv1c.NewPaperTradingServiceHandler(paperHandler,
		withSency(otelInterceptor, authInterceptor)))
	codeAssistServer := ai.NewCodeAssistServer(aiSvc, session, log)
	// CodeAssist uses LLM-only code analysis and generation.
	mux.Handle(antv1c.NewCodeAssistServiceHandler(codeAssistServer, withSency(otelInterceptor, authInterceptor)))
	systemAIServer := ai.NewSystemAIServer(aiSvc, log)
	mux.Handle(antv1c.NewSystemAIServiceHandler(systemAIServer, withSency(otelInterceptor, authInterceptor)))
	aiPrimaryServer := ai.NewAIPrimaryServer(aiSvc, log)
	mux.Handle(antv1c.NewAIPrimaryServiceHandler(aiPrimaryServer, withSency(otelInterceptor, authInterceptor)))
	backtestTradesServer := strategy.NewBacktestTradesServer(backtestRunRepo, log)
	mux.Handle(antv1c.NewBacktestTradesServiceHandler(backtestTradesServer, withSency(otelInterceptor, authInterceptor)))
	gateEvalServer := ai.NewGateEvalServer(backtestRunRepo, log)
	mux.Handle(antv1c.NewGateServiceHandler(gateEvalServer, withSency(otelInterceptor, authInterceptor)))
	// Claude Code style: separated plan → execute pipeline
	strategyPlanServer := ai.NewStrategyPlanServer(aiSvc, backtestRunRepo, convRepo, marketDataRepo, log)
	strategyPlanServer.SetPoolAdapter(
		func(ctx context.Context, sql string, args ...any) error {
			_, e := pool.Exec(ctx, sql, args...)
			return e
		},
		func(ctx context.Context, sql string, args ...any) (string, error) {
			var v string
			row := pool.QueryRow(ctx, sql, args...)
			err := row.Scan(&v)
			return v, err
		},
	)
	mux.Handle(antv1c.NewStrategyPlanServiceHandler(strategyPlanServer, withSency(otelInterceptor, authInterceptor)))

	economicDataServer := system.NewEconomicDataServer(log)
	mux.Handle(antv1c.NewEconomicDataServiceHandler(economicDataServer, withSency(otelInterceptor, authInterceptor)))
	jobServer := system.NewJobServer(jobRepo, log)
	jobServer.SetPgListen(pgListen)
	mux.Handle(antv1c.NewJobServiceHandler(jobServer, withSency(otelInterceptor, authInterceptor)))
	logServiceServer := system.NewLogServiceServer(logSvc, log)
	mux.Handle(antv1c.NewLogServiceHandler(logServiceServer, withSency(otelInterceptor, authInterceptor)))
	notifServer := notification.NewNotificationServer(notifRepo, notifSub, log)
	mux.Handle(antv1c.NewNotificationServiceHandler(notifServer, withSency(otelInterceptor, authInterceptor)))
	gateEvalServer.SetNotificationSender(notifSender)

	gate := risk.NewDefaultGate()
	gate.SetKillSwitch(func() bool { return cfg.RiskGateKillSwitch })
	gate.SetAutotradeEnabled(func(uid string) bool { return cfg.RiskGateAutotradeEnabled })

	registerAdminHandlers(mux, pool, log, walletSvc, accountNumberSvc, strategySvc,
		agentGateway.SettingsStore(), agentGateway.HookEngine(),
		accountEventPub,
		rdb, nc,
		otelInterceptor, authInterceptor, adminInterceptor)

	// S1.1-S1.3: Wire SignalPipeline, rate limiter, cost estimator, OMS writer.

	// AutoTradingService handler — leverages existing pipeline + repositories.
	autoTradingRepo := repository.NewAutoTradingRepository(pool)
	autoTradingServer := autotrading.NewAutoTradingServer(autoTradingRepo, nil, log)

	// Wire per-user risk config into the Gate (frontend settings → Gate evaluation).
	// Tries RiskConfig (account-level) first, falls back to GlobalSettings (user-level).
	strategyExecServer.AddGateRule(&risk.UserRiskConfigRule{Store: func(ctx context.Context, accountID string) (*risk.UserRiskConfig, error) {
		aid, err := uuid.Parse(accountID)
		if err != nil {
			return nil, nil
		}
		// 1. Try account-level RiskConfig.
		if rc, err := autoTradingRepo.GetRiskConfigByAccountID(ctx, aid); err == nil && rc != nil {
			return &risk.UserRiskConfig{
				MaxLotSize: rc.MaxLotSize, MaxPositions: rc.MaxPositions,
				MaxDailyLoss:       rc.MaxDailyLoss,
				MaxDrawdownPercent: rc.MaxDrawdownPercent,
				MaxRiskPercent:     rc.MaxRiskPercent,
			}, nil
		}
		// 2. Fall back to user-level GlobalSettings.
		uid, _ := uuid.Parse(usermgr.GetUserID(ctx))
		if uid != uuid.Nil {
			if gs, err := autoTradingRepo.GetGlobalSettingsByUserID(ctx, uid); err == nil && gs != nil {
				return &risk.UserRiskConfig{
					MaxLotSize: gs.MaxLotSize, MaxPositions: int(gs.MaxPositions),
					MaxDailyLoss:       gs.MaxDailyLoss,
					MaxDrawdownPercent: gs.MaxDrawdownPercent,
					MaxRiskPercent:     gs.MaxRiskPercent,
				}, nil
			}
		}
		return nil, nil // no config = no restriction
	}})
	mux.Handle(antv1c.NewAutoTradingServiceHandler(autoTradingServer,
		withSency(otelInterceptor, authInterceptor)))

	// Schedule execution engine — timer-driven loop that dispatches due schedules
	// to RunLiveStrategy (bar stream → Go-native executor → signal → OMS).
	scheduleRepo := repository.NewStrategyScheduleRepository(pool)
	scheduleEngine := strategy.NewScheduleEngine(scheduleRepo, templatesRepo,
		strategyExecServer,
		func(userID uuid.UUID) bool {
			settings, err := autoTradingRepo.GetGlobalSettingsByUserID(context.Background(), userID)
			if err != nil {
				return true // default to enabled if settings not found
			}
			return settings.AutoTradeEnabled
		},
		log)
	strategyServer.SetEngine(scheduleEngine)

	emailNotifier, workerCleanup := registerSREHandlers(
		userRepo, mux, log, pool, store, nc, rdb, cfg,
		authInterceptor, otelInterceptor, platformSvc, mthubSvc,
		authServer,
		strategyExperimentRepo, strategyAssetRepo, schedHealthRepo,
		analyticsCache,
		aiSvc,
		backtestRunRepo,
		pgListen,
		emailNotifier,
	)

	startBackgroundServices(ctx, pool, emailNotifier, depositAddrRepo, adminRepo, depositSvc, cfg, log)

	return reconLoop, emailNotifier, platformAgg, notifSender, scheduleEngine, workerCleanup, chainMonitor, reconcilerInst, sweepWorker
}
