package main

import (
	"context"
	"net/http"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/config"
	"anttrader/internal/connect/admin"
	"anttrader/internal/connect/ai"
	mktplace "anttrader/internal/connect/marketplace"
	"anttrader/internal/connect/strategy"
	"anttrader/internal/connect/system"
	"anttrader/internal/connect/user"
	"anttrader/internal/costsvc"
	"anttrader/internal/factor"
	"anttrader/internal/interceptor"
	"anttrader/internal/marketplace"
	"anttrader/internal/mdgateway"
	"anttrader/internal/mdgateway/adapter/brokersearch"
	"anttrader/internal/mthub"
	"anttrader/internal/notifier"
	"anttrader/internal/pkg/secretbox"
	"anttrader/internal/repository"
	"anttrader/internal/risksvc"
	"anttrader/internal/service"
	systemai "anttrader/internal/service/systemai"
	antredis "anttrader/internal/storage/redis"
	"anttrader/internal/strategysvc"
	"anttrader/internal/usermgr"

	connectrpc "connectrpc.com/connect"
)

func registerHandlers(
	mux *http.ServeMux,
	log *zap.Logger,
	pool *pgxpool.Pool,
	ch clickhouse.Conn,
	nc *nats.Conn,
	rdb *antredis.Client,
	cfg *config.Config,
	jwtSecret string,
	accountSvc *service.AccountService,
	platformSvc *service.PlatformService,
	authInterceptor *interceptor.AuthInterceptor,
	adminInterceptor *interceptor.AdminInterceptor,
	rateLimitInterceptor *interceptor.RateLimitInterceptor,
	mthubSvc *mthub.MtHubService,
	hub *mthub.Hub,
	tradeRecordRepo *repository.TradeRecordRepository,
	js nats.JetStreamContext,
	eventStore *mthub.TradeEventStore,
	reconcileGate *mthub.ReconcileGate,
	analyticsCache *service.AnalyticsCache,
) (*mthub.ReconciliationLoop, *notifier.EmailNotifier, *risksvc.PlatformAggregator) {

	// ConnectRPC handlers
	// Repositories for handler→service→repository layering (P1-2).
	userRepo := repository.NewUserRepository(pool)
	convRepo := repository.NewAIConversationRepository(pool)
	jobRepo := repository.NewJobRepository(pool)
	schedHealthRepo := repository.NewScheduleHealthRepository(pool)
	marketDataRepo := repository.NewMarketDataRepository(ch, log)

	authServer := user.NewAuthServer(userRepo, jwtSecret, log)
	authServer.SetInsecureCookies(true) // no TLS in Docker deployment
	mux.Handle(antv1c.NewAuthServiceHandler(authServer, connectrpc.WithInterceptors(rateLimitInterceptor, authInterceptor)))

	reconLoop := mthub.NewReconciliationLoop(hub, pool, rdb.Client(), log, reconcileGate)

	mthubServer := system.NewMtHubServer(mthubSvc, platformSvc, marketDataRepo, tradeRecordRepo, log)
	mux.Handle(antv1c.NewMtHubServiceHandler(mthubServer, connectrpc.WithInterceptors(authInterceptor)))

	searcher := brokersearch.New("", "")
	accountEventPub := mdgateway.NewAccountEventPublisher(js, log)
	mtTester := user.NewMTConnectionTester(cfg.MtapiToken, log)
	accountServer := user.NewAccountServer(accountSvc, searcher, accountEventPub, mtTester, log)
	mux.Handle(antv1c.NewAccountServiceHandler(accountServer, connectrpc.WithInterceptors(authInterceptor)))

	mktServer := mktplace.NewMarketServer(platformSvc, marketDataRepo, nc, log)
	mux.Handle(antv1c.NewMarketServiceHandler(mktServer, connectrpc.WithInterceptors(authInterceptor)))

	mktplaceSvc := marketplace.New(pool)
	mktplaceHandler := mktplace.NewMarketplaceServer(mktplaceSvc, log)
	mux.Handle(antv1c.NewMarketplaceServiceHandler(mktplaceHandler, connectrpc.WithInterceptors(authInterceptor)))

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
	aiServer := ai.NewAIServer(aiSvc, convRepo, log)
	mux.Handle(antv1c.NewAIServiceHandler(aiServer, connectrpc.WithInterceptors(authInterceptor)))

	// Debate V2 service (multi-expert AI strategy generation).
	debateV2Svc := service.NewDebateV2Service(pool, jobRepo, log)
	debateV2Server := ai.NewDebateV2Server(debateV2Svc, log)
	mux.Handle(antv1c.NewDebateV2ServiceHandler(debateV2Server, connectrpc.WithInterceptors(authInterceptor)))

	gateProgressServer := ai.NewGateProgressServer(log)

	streamServer := system.NewStreamServer(mthubSvc, platformSvc, log)
	mux.Handle(antv1c.NewStreamServiceHandler(streamServer, connectrpc.WithInterceptors(authInterceptor)))

	strategySvc := service.NewStrategySvc(pool)
	strategyServer := strategy.NewStrategyServer(strategySvc, log)
	mux.Handle(antv1c.NewStrategyServiceHandler(strategyServer, connectrpc.WithInterceptors(authInterceptor)))

	// Mock/stub handlers — return mock data for services not yet connected to real backends.
	// Real: SystemAI, AIPrimary, Job, ScheduleHealth, DebateV2
	// Mock: PythonStrategy, CodeAssist, BacktestTrades, EconomicData
	pythonStrategyServer := strategy.NewPythonStrategyServer(backtestRunRepo, log)
	if cfg.StrategyServiceURL != "" {
		pythonClient := strategysvc.NewPythonClient(cfg.StrategyServiceURL)
		pythonStrategyServer.SetClient(pythonClient)
		strategyServer.SetClient(pythonClient) // S2.5: real RunBacktest
		log.Info("Python strategy client configured", zap.String("url", cfg.StrategyServiceURL))
	}
	mux.Handle(antv1c.NewPythonStrategyServiceHandler(pythonStrategyServer, connectrpc.WithInterceptors(authInterceptor)))
	codeAssistServer := ai.NewCodeAssistServer(aiSvc, log)
	mux.Handle(antv1c.NewCodeAssistServiceHandler(codeAssistServer, connectrpc.WithInterceptors(authInterceptor)))
	systemAIServer := ai.NewSystemAIServer(aiSvc, log)
	mux.Handle(antv1c.NewSystemAIServiceHandler(systemAIServer, connectrpc.WithInterceptors(authInterceptor)))
	aiPrimaryServer := ai.NewAIPrimaryServer(aiSvc, log)
	mux.Handle(antv1c.NewAIPrimaryServiceHandler(aiPrimaryServer, connectrpc.WithInterceptors(authInterceptor)))
	backtestTradesServer := strategy.NewBacktestTradesServer(backtestRunRepo, log)
	mux.Handle(antv1c.NewBacktestTradesServiceHandler(backtestTradesServer, connectrpc.WithInterceptors(authInterceptor)))
	economicDataServer := system.NewEconomicDataServer(log)
	mux.Handle(antv1c.NewEconomicDataServiceHandler(economicDataServer, connectrpc.WithInterceptors(authInterceptor)))
	jobServer := system.NewJobServer(jobRepo, log)
	mux.Handle(antv1c.NewJobServiceHandler(jobServer, connectrpc.WithInterceptors(authInterceptor)))
	logServiceServer := system.NewLogServiceServer(logSvc, log)
	mux.Handle(antv1c.NewLogServiceHandler(logServiceServer, connectrpc.WithInterceptors(authInterceptor)))
	adminRepo := repository.NewAdminRepository(pool)
	passwordResetRepo := repository.NewPasswordResetRepo(pool)
	adminTradingServer := admin.NewAdminTradingServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminTradingServiceHandler(adminTradingServer, connectrpc.WithInterceptors(authInterceptor, adminInterceptor)))
	adminConfigServer := admin.NewAdminConfigServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminConfigServiceHandler(adminConfigServer, connectrpc.WithInterceptors(authInterceptor, adminInterceptor)))
	adminLogServer := admin.NewAdminLogServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminLogServiceHandler(adminLogServer, connectrpc.WithInterceptors(authInterceptor, adminInterceptor)))
	adminAccountServer := admin.NewAdminAccountServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminAccountServiceHandler(adminAccountServer, connectrpc.WithInterceptors(authInterceptor, adminInterceptor)))
	adminUserServer := admin.NewAdminUserServer(adminRepo, passwordResetRepo, log)
	mux.Handle(antv1c.NewAdminUserServiceHandler(adminUserServer, connectrpc.WithInterceptors(authInterceptor, adminInterceptor)))
	adminSystemServer := admin.NewAdminSystemServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminSystemServiceHandler(adminSystemServer, connectrpc.WithInterceptors(authInterceptor, adminInterceptor)))

	// --- M11-16 Jurisdictional Gate ---
	jurisStore := risksvc.NewPgJurisdictionStore(pool)
	geoipResolver := risksvc.NewMaxMindGeoIPResolver(cfg.GeoIPDBPath)
	jurisGate := &risksvc.JurisdictionGate{
		Store:               jurisStore,
		GeoIP:               geoipResolver,
		RequireKYC:           cfg.RequireKYC,
		RequireDisclaimer:    cfg.RequireDisclaimer,
		RequireQuestionnaire: cfg.RequireQuestionnaire,
	}
	// S1.1: Wire SignalPipeline for pre-trade risk checks (capability → hardlimit → platform → engine → sizer).
	capStore := risksvc.NewCapabilityStore()
	rows, err := pool.Query(context.Background(),
		`SELECT user_id, COALESCE(capability_tier, 0),
		        COALESCE(order_types_allowed, '{}'),
		        lot_per_order_max, daily_order_max, leverage_max,
		        COALESCE(symbol_whitelist, '{}'),
		        COALESCE(killswitch_enabled, false)
		 FROM user_risk_profiles`)
	if err != nil {
		log.Error("capability LoadFromPG query failed, using defaults", zap.Error(err))
	} else {
		if err := capStore.LoadFromPG(context.Background(), rows); err != nil {
			log.Error("capability LoadFromPG scan failed, using defaults", zap.Error(err))
		}
		log.Info("capability store loaded", zap.Int("users", capStore.Count()))
	}
	hardLimit := risksvc.NewHardLimitEvaluator(&risksvc.KycJurisdictionRule{Gate: jurisGate})
	platformAgg := risksvc.NewPlatformAggregator()
	platformAgg.StartRefreshLoop(5 * time.Second) // B-1.4: async recalculate on dirty
	platformLimits := risksvc.DefaultPlatformLimits()
	riskEngine := risksvc.NewEngine(
		&risksvc.MaxPosition{Max: 20},
		&risksvc.Margin{MinLevel: 1.5},
	)
	sizer := &risksvc.VolTargetSizer{RiskBudgetPct: 0.01}
	allocator := &risksvc.ProRataAllocator{}

	pipeline := risksvc.NewSignalPipeline(risksvc.PipelineConfig{
		CapStore:  capStore,
		HardLimit: hardLimit,
		Platform:  platformAgg,
		Limits:    platformLimits,
		Engine:    riskEngine,
		Sizer:     sizer,
		Allocator: allocator,
	})
	mthubSvc.SetRiskPipeline(pipeline)

	// Account state provider queries PG for balance/equity/margin per account.
	mthubSvc.SetAccountStateProvider(func(ctx context.Context, accountID string) (*mthub.AccountState, error) {
		var state mthub.AccountState
		var positions int64
		err := pool.QueryRow(ctx,
			`SELECT balance, equity, free_margin, COALESCE(margin, 0)::float8,
			        COALESCE((SELECT count(*) FROM positions WHERE mt_account_id = $1), 0)::int
			 FROM mt_accounts WHERE id = $1::uuid`,
			accountID,
		).Scan(&state.Balance, &state.Equity, &state.FreeMargin, &state.Margin, &positions)
		if err != nil {
			return nil, err
		}
		state.Positions = int(positions)
		return &state, nil
	})

	// S1.3: Wire per-user rate limiter (10 orders/sec/user, 100 signals/sec/user).
	limiter := usermgr.NewUserLimiter(usermgr.DefaultConfig())
	mthubSvc.SetUserLimiter(limiter)

	// S1.3: Wire pre-trade cost estimator (EURUSD default model).
	eurtusdModel := &costsvc.CostModel{
		Symbol:           "EURUSD",
		SpreadPips:       1.0,
		PipSize:          0.0001,
		PipValue:         10.0,
		CommissionPerLot: 7.0,
	}
	estimator := &costsvc.StaticEstimator{Model: eurtusdModel}
	mthubSvc.SetCostEstimator(estimator)

	// S1.2: OMS state writer for order lifecycle tracking.
	omsWriter := mthub.NewOmsWriter(pool, eventStore)
	mthubSvc.SetOmsWriter(omsWriter)

	// D-1: Wire factor subscriber for bar-event DSL evaluation (M10-BASE-B6).
	// TODO(H16): factorSub is created and started but not yet connected to a
	// factor evaluation pipeline. Needs: (1) a factor registry that registers DSL
	// strategies, (2) a bar-stream subscription from the market data gateway,
	// (3) evaluation results fed back into the signal/order pipeline.
	factorSub := factor.NewSubscriber(factor.DefaultSubscriberConfig(), log)
	go factorSub.Start(context.Background())

	adminJurisdictionServer := admin.NewAdminJurisdictionServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminJurisdictionServiceHandler(adminJurisdictionServer, connectrpc.WithInterceptors(authInterceptor, adminInterceptor)))

	emailNotifier := registerSREHandlers(
		mux, log, pool, ch, nc, rdb, cfg,
		authInterceptor, platformSvc, mthubSvc,
		authServer, debateV2Server, gateProgressServer,
		strategyExperimentRepo, strategyAssetRepo, schedHealthRepo,
		analyticsCache,
	)

	return reconLoop, emailNotifier, platformAgg
}
