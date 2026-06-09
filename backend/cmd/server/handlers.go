package main

import (
	"context"
	"net/http"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	internalai "anttrader/internal/ai"
	"anttrader/internal/analysis"
	"anttrader/internal/config"
	"anttrader/internal/connect/admin"
	"anttrader/internal/connect/ai"
	algo "anttrader/internal/connect/algo"
	assetanalysis "anttrader/internal/connect/asset_analysis"
	"anttrader/internal/connect/autotrading"
	mktplace "anttrader/internal/connect/marketplace"
	"anttrader/internal/connect/notification"
	"anttrader/internal/connect/strategy"
	"anttrader/internal/connect/system"
	"anttrader/internal/connect/user"
	paperhdr "anttrader/internal/connect/paper"
	papereng "anttrader/internal/paper"
	"anttrader/internal/interceptor"
	"anttrader/internal/marketplace"
	"anttrader/internal/mdgateway"
	"anttrader/internal/mdgateway/adapter"
	"anttrader/internal/mdgateway/adapter/brokersearch"
	"anttrader/internal/mthub"
	notifpubsub "anttrader/internal/notification"
	"anttrader/internal/notifier"
	"anttrader/internal/pglisten"
	"anttrader/internal/pkg/secretbox"
	"anttrader/internal/repository"
	"anttrader/internal/risksvc"
	"anttrader/internal/service"
	systemai "anttrader/internal/service/systemai"
	antredis "anttrader/internal/storage/redis"

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
	otelInterceptor connectrpc.Interceptor,
	mthubSvc *mthub.MtHubService,
	hub *mthub.Hub,
	tradeRecordRepo *repository.TradeRecordRepository,
	js nats.JetStreamContext,
	eventStore *mthub.TradeEventStore,
	reconcileGate *mthub.ReconcileGate,
	analyticsCache *service.AnalyticsCache,
	brokerReg *adapter.BrokerRegistry,
) (*mthub.ReconciliationLoop, *notifier.EmailNotifier, *risksvc.PlatformAggregator, *notifpubsub.Sender, func()) {

	// ConnectRPC handlers
	// Repositories for handler→service→repository layering (P1-2).
	userRepo := repository.NewUserRepository(pool)
	convRepo := repository.NewAIConversationRepository(pool)
	session := internalai.NewConversationSession(convRepo)
	templatesRepo := repository.NewAIStrategyTemplatesRepository(pool)
	jobRepo := repository.NewJobRepository(pool)
	schedHealthRepo := repository.NewScheduleHealthRepository(pool)
	marketDataRepo := repository.NewMarketDataRepository(ch, log)

	authServer := user.NewAuthServer(userRepo, jwtSecret, log)
	authServer.SetInsecureCookies(true) // no TLS in Docker deployment
	mux.Handle(antv1c.NewAuthServiceHandler(authServer, connectrpc.WithInterceptors(otelInterceptor,rateLimitInterceptor, authInterceptor)))

	reconLoop := mthub.NewReconciliationLoop(hub, pool, rdb.Client(), log, reconcileGate)

	mthubServer := system.NewMtHubServer(mthubSvc, platformSvc, marketDataRepo, tradeRecordRepo, log)
	mux.Handle(antv1c.NewMtHubServiceHandler(mthubServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))

	searcher := brokersearch.New("", "")
	accountEventPub := mdgateway.NewAccountEventPublisher(js, log)
	mtTester := user.NewMTConnectionTester(cfg.MtapiToken, log)
	accountServer := user.NewAccountServer(accountSvc, searcher, accountEventPub, mtTester, log).
		WithSessionWaiter(hub)
	mux.Handle(antv1c.NewAccountServiceHandler(accountServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))

	mktServer := mktplace.NewMarketServer(platformSvc, marketDataRepo, nc, log)
	mux.Handle(antv1c.NewMarketServiceHandler(mktServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))

	mktplaceSvc := marketplace.New(pool)
	mktplaceHandler := mktplace.NewMarketplaceServer(mktplaceSvc, platformSvc, log)
	mux.Handle(antv1c.NewMarketplaceServiceHandler(mktplaceHandler, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))

	// M12-A2: Execution Algo handler (TWAP/VWAP/POV/Shortfall).
	// brokerReg is created in main.go before the pipeline starts; gateways register
	// via adapter.RegisterDefaults inside the mdgateway runner after connection.
	algoServer := algo.NewExecutionAlgoServer(brokerReg, log)
	mux.Handle(antv1c.NewExecutionAlgoServiceHandler(algoServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))

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
	agentDefRepo := repository.NewAIAgentDefinitionRepository(pool)
	aiServer := ai.NewAIServer(aiSvc, convRepo, session, log)
	aiServer.SetAgentDefRepo(agentDefRepo)
	mux.Handle(antv1c.NewAIServiceHandler(aiServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
	// Agent definition CRUD (no proto RPC yet — raw HTTP).
	mux.Handle(antv1c.NewAgentDefinitionServiceHandler(aiServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))

	// P3: AI Asset Analysis — MTF outlook, S/R levels, volatility, AI recommendation.
	assetAnalyzer := analysis.NewAnalyzer(marketDataRepo, log)
	assetAnalysisServer := assetanalysis.NewAssetAnalysisServer(assetAnalyzer, aiSvc, log)
	mux.Handle(antv1c.NewAssetAnalysisServiceHandler(assetAnalysisServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))

	streamServer := system.NewStreamServer(mthubSvc, platformSvc, log)
	streamServer.SetMarketDataRepo(marketDataRepo)
	mux.Handle(antv1c.NewStreamServiceHandler(streamServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))

	strategySvc := service.NewStrategySvc(pool)
	strategyServer := strategy.NewStrategyServer(strategySvc, log)
	strategy.SetProtoLog(log)
	pgListen := pglisten.New(pool, log)
	strategyServer.SetPgListen(pgListen)
	mux.Handle(antv1c.NewStrategyServiceHandler(strategyServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))

	// Mock/stub handlers — return mock data for services not yet connected to real backends.
	// Real: SystemAI, AIPrimary, Job, ScheduleHealth
	// Mock: PythonStrategy, CodeAssist, BacktestTrades, EconomicData
	pythonStrategyServer := strategy.NewPythonStrategyServer(backtestRunRepo, log)
	pythonStrategyServer.SetPgListen(pgListen)
	if cfg.StrategyServiceURL != "" {
		connectClient := antv1c.NewPythonStrategyServiceClient(http.DefaultClient, cfg.StrategyServiceURL)
		pythonStrategyServer.SetConnectClient(connectClient)
		backtestClient := antv1c.NewBacktestServiceClient(http.DefaultClient, cfg.StrategyServiceURL)
		pythonStrategyServer.SetBacktestClient(backtestClient)
		pythonStrategyServer.SetMarketDataRepo(marketDataRepo)
		pythonStrategyServer.SetBarSource(strategy.NewBacktestSource(marketDataRepo))
		pythonStrategyServer.SetMtHub(mthubSvc) // for live strategy order dispatch
		strategyServer.SetBacktestClient(backtestClient)
		strategyServer.SetMarketDataRepo(marketDataRepo)
		pythonStrategyServer.StartBacktestWorker(context.Background()) // Background worker for async backtest runs
		objScoreClient := antv1c.NewObjectiveScoreServiceClient(http.DefaultClient, cfg.StrategyServiceURL)
		objectiveScoreServer := strategy.NewObjectiveScoreServer(objScoreClient, log)
		mux.Handle(antv1c.NewObjectiveScoreServiceHandler(objectiveScoreServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
		log.Info("Python strategy client configured", zap.String("url", cfg.StrategyServiceURL))
	}
	mux.Handle(antv1c.NewPythonStrategyServiceHandler(pythonStrategyServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))

	// Paper trading — virtual trading engine + ConnectRPC handler.
	paperRepo := repository.NewPaperRepo(pool)
	paperEngine := papereng.New(paperRepo, mthubSvc, log)
	pythonStrategyServer.SetPaperEngine(paperEngine)
	paperHandler := paperhdr.NewHandler(paperRepo, paperEngine, pythonStrategyServer, log)
	mux.Handle(antv1c.NewPaperTradingServiceHandler(paperHandler, connectrpc.WithInterceptors(otelInterceptor, authInterceptor)))
	codeAssistServer := ai.NewCodeAssistServer(aiSvc, session, log)
	if cfg.StrategyServiceURL != "" {
		codeAssistServer.SetPythonStrategyClient(antv1c.NewPythonStrategyServiceClient(http.DefaultClient, cfg.StrategyServiceURL))
	}
	mux.Handle(antv1c.NewCodeAssistServiceHandler(codeAssistServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
	systemAIServer := ai.NewSystemAIServer(aiSvc, log)
	mux.Handle(antv1c.NewSystemAIServiceHandler(systemAIServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
	aiPrimaryServer := ai.NewAIPrimaryServer(aiSvc, log)
	mux.Handle(antv1c.NewAIPrimaryServiceHandler(aiPrimaryServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
	backtestTradesServer := strategy.NewBacktestTradesServer(backtestRunRepo, log)
	mux.Handle(antv1c.NewBacktestTradesServiceHandler(backtestTradesServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
	gateEvalServer := ai.NewGateEvalServer(backtestRunRepo, log)
	mux.Handle(antv1c.NewGateServiceHandler(gateEvalServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
	strategyGenServer := ai.NewStrategyGenServer(aiSvc, templatesRepo, convRepo, backtestRunRepo, log)
	mux.Handle(antv1c.NewStrategyGenerationServiceHandler(strategyGenServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
	economicDataServer := system.NewEconomicDataServer(log)
	mux.Handle(antv1c.NewEconomicDataServiceHandler(economicDataServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
	jobServer := system.NewJobServer(jobRepo, log)
	mux.Handle(antv1c.NewJobServiceHandler(jobServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
	logServiceServer := system.NewLogServiceServer(logSvc, log)
	mux.Handle(antv1c.NewLogServiceHandler(logServiceServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
	notifSub := notifpubsub.NewSubscriber()
	notifRepo := repository.NewNotificationRepository(pool)
	notifServer := notification.NewNotificationServer(notifRepo, notifSub, log)
	mux.Handle(antv1c.NewNotificationServiceHandler(notifServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
	notifSender := notifpubsub.NewSender(notifRepo, notifSub, log)
	pythonStrategyServer.SetNotificationSender(notifSender)
	gateEvalServer.SetNotificationSender(notifSender)

	// Auto-gate callback: runs gate evaluation after every backtest completion.
	// If gate fails, spawns async auto-fix (LLM code repair → new backtest).
	onBacktestComplete := func(ctx context.Context, run *repository.BacktestRun) {
		dailyReturns := internalai.EquityCurveToDailyReturns(run.ProtoResponse)
		if len(dailyReturns) < 10 {
			return
		}
		input := internalai.PipelineInput{
			DailyReturns: dailyReturns,
			NumAttempts:  1,
		}
		result := internalai.Pipeline(input)

		// Send gate notification (shared with GateEvalServer.RunEvaluation).
		ai.SendGateNotification(ctx, notifSender, run.UserID, run, result)

		if result.Passed {
			return
		}

		// Spawn async auto-fix: LLM generates improved code, creates new backtest run.
		go autoFixCode(context.Background(), run, result, aiSvc, backtestRunRepo, notifSender, log)
	}
	pythonStrategyServer.SetOnBacktestComplete(onBacktestComplete)

	adminRepo := repository.NewAdminRepository(pool)
	passwordResetRepo := repository.NewPasswordResetRepo(pool)
	adminTradingServer := admin.NewAdminTradingServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminTradingServiceHandler(adminTradingServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))
	adminConfigServer := admin.NewAdminConfigServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminConfigServiceHandler(adminConfigServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))
	adminLogServer := admin.NewAdminLogServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminLogServiceHandler(adminLogServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))
	adminAccountServer := admin.NewAdminAccountServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminAccountServiceHandler(adminAccountServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))
	adminUserServer := admin.NewAdminUserServer(adminRepo, passwordResetRepo, log)
	mux.Handle(antv1c.NewAdminUserServiceHandler(adminUserServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))
	adminSystemServer := admin.NewAdminSystemServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminSystemServiceHandler(adminSystemServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))

	// S1.1-S1.3: Wire SignalPipeline, rate limiter, cost estimator, OMS writer.
	pipeline, platformAgg := initRiskPipeline(pool, log, mthubSvc, eventStore, cfg)

	// AutoTradingService handler — leverages existing pipeline + repositories.
	autoTradingRepo := repository.NewAutoTradingRepository(pool)
	autoTradingServer := autotrading.NewAutoTradingServer(autoTradingRepo, pipeline, log)
	mux.Handle(antv1c.NewAutoTradingServiceHandler(autoTradingServer,
		connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))

	// Factor subscriber activation prerequisites (M10-BASE-B6):
	// When ready to wire, create and start:
	//   factorSub := factor.NewSubscriber(factor.DefaultSubscriberConfig(), log)
	//   go factorSub.Start(pipelineCtx)
	// Required before activation:
	//   (1) Factor registry that registers DSL strategies
	//   (2) Bar-stream subscription from mdgateway → factorSub.Push()
	//   (3) Evaluation results → signal/order pipeline

	adminJurisdictionServer := admin.NewAdminJurisdictionServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminJurisdictionServiceHandler(adminJurisdictionServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))

	emailNotifier, workerCleanup := registerSREHandlers(
		mux, log, pool, ch, nc, rdb, cfg,
		authInterceptor, otelInterceptor, platformSvc, mthubSvc,
		authServer,
		strategyExperimentRepo, strategyAssetRepo, schedHealthRepo,
		analyticsCache,
		aiSvc,
	)

	return reconLoop, emailNotifier, platformAgg, notifSender, workerCleanup
}
