package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"anttrader/internal/app"
	"anttrader/internal/config"
	connectsvc "anttrader/internal/connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/repository"
	"anttrader/internal/service"
	connecttransport "anttrader/internal/transport/connect"
	"anttrader/internal/transport/rest"
)

type Server struct {
	cfg        *config.Config
	httpServer *http.Server
	container  *app.Container
	streamSvc  *connectsvc.StreamService
	lifecycle  *app.Lifecycle
}

func newExecutionGateway(tradingSvc *service.TradingService, accountRepo *repository.AccountRepository, logSvc *service.LogService, redisClient *redis.Client) (*service.EngineSelector, *service.ExecutionGateway) {
	engineSelector := service.NewEngineSelector(tradingSvc, service.NewTickBacktestEngine(service.BacktestCostConfig{}))
	var execStore service.ExecutionIdempotencyStore
	if redisClient != nil {
		execStore = service.NewRedisExecutionIdempotencyStore(redisClient)
	}
	execGateway := service.NewExecutionGateway(engineSelector, accountRepo, nil, logSvc, execStore)
	return engineSelector, execGateway
}

func New(cfg *config.Config) (*Server, error) {
	c, err := app.NewContainer(cfg)
	if err != nil {
		return nil, err
	}
	if os.Getenv("ANTRADER_SEED_DEFAULT_STRATEGY_TEMPLATES") == "true" {
		seedDefaultStrategyTemplates(context.Background(), c.SQLXDB, c.TemplateRepo)
	}

	// M1-3: seed canonical_symbols on every startup (idempotent).
	seedCanonicalSymbols(context.Background(), c.SQLXDB)

	s := &Server{
		cfg:       cfg,
		container: c,
		lifecycle: app.NewLifecycle(cfg, c),
	}
	return s, nil
}
func (s *Server) Start() error {
	if s.lifecycle != nil {
		s.lifecycle.Start(context.Background())
	}

	connectHandler := s.setupConnectHandlers()
	addr := fmt.Sprintf("0.0.0.0:%d", s.cfg.Server.HTTPPort)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      h2c.NewHandler(connectHandler, &http2.Server{}),
		ReadTimeout:  s.cfg.Server.ReadTimeout,
		WriteTimeout: 0,
		IdleTimeout:  0,
	}

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}
	return nil
}

// setupConnectHandlers configures the HTTP mux for connect-rpc and health.
func (s *Server) setupConnectHandlers() http.Handler {
	c := s.container
	mux := http.NewServeMux()

	rest.RegisterHealth(mux)

	authConnect := connectsvc.NewAuthService(c.UserRepo, s.cfg.JWT.Secret)
	streamConnect := connectsvc.NewStreamService(c.AccountRepo, c.TradeRecordRepo, c.AnalyticsRepo, c.ConnMgr, &s.cfg.MT4, &s.cfg.MT5)
	instanceID := os.Getenv("INSTANCE_ID")
	if instanceID == "" {
		instanceID = uuid.New().String()
	}
	streamConnect.SetSessionLeader(c.RedisClient, instanceID)
	s.streamSvc = streamConnect
	accountConnect := connectsvc.NewAccountService(c.AccountRepo, c.ConnMgr, c.BrokerSvc, streamConnect, c.AccountSvc)
	tradingConnect := connectsvc.NewTradingService(c.AccountRepo, c.TradingSvc, c.ConnMgr, c.LogSvc, streamConnect)
	marketConnect := connectsvc.NewMarketService(c.AccountRepo, c.MarketSvc, c.KlineSvc)
	strategyConnect := connectsvc.NewStrategyService(nil, c.ScheduleSvc, c.TemplateSvc, c.PythonSvc, c.LogSvc)
	convRepo := repository.NewAIConversationRepository(c.SQLXDB)
	wfRepo := repository.NewAIWorkflowRepository(c.SQLXDB)
	aiConnect := connectsvc.NewAIService(c.ReportSvc, c.AICfgSvc, c.AIAgentSvc, c.AIManager, convRepo, wfRepo, c.PythonSvc)
	adminConnect := connectsvc.NewAdminService(c.AdminSvc)
	analyticsConnect := connectsvc.NewAnalyticsService(c.AnalyticsSvc)
	pythonConnect := connectsvc.NewPythonStrategyService(c.PythonSvc, c.KlineSvc, c.DynamicConfigSvc, c.BacktestDataset, c.TickDataset, c.BacktestRunSvc, streamConnect)
	datasetConnect := connectsvc.NewBacktestDatasetService(c.BacktestDataset, c.KlineSvc, c.DynamicConfigSvc)
	backtestTradesConnect := connectsvc.NewBacktestTradesService(c.BacktestRunSvc)
	debateV2Connect := connectsvc.NewDebateV2Service(c.DebateV2Svc)
	debateV2StreamConnect := connectsvc.NewDebateV2StreamService(c.DebateV2Svc)
	aiPrimaryConnect := connectsvc.NewAIPrimaryService(c.AICfgSvc)
	codeAssistConnect := connectsvc.NewCodeAssistService(c.AICodeAssist, c.PythonSvc)
	logConnect := connectsvc.NewLogService(c.LogSvc)
	scheduleHealthConnect := connectsvc.NewScheduleHealthService(c.LogSvc, c.DynamicConfigSvc)
	autoTradingConnect := connectsvc.NewAutoTradingService(c.AutoTradingSvc, c.ScheduleSvc)
	systemAIConnect := connectsvc.NewSystemAIService(c.SystemAISvc)
	objectiveScoreConnect := connectsvc.NewObjectiveScoreService(c.PythonSvc)
	indicatorCatalogConnect := connectsvc.NewIndicatorCatalogService()
	economicDataConnect := connectsvc.NewEconomicDataService(c.EconCalendarSvc)
	agentConnect := connectsvc.NewAgentService(repository.NewAgentRepository(c.SQLXDB))
	jobConnect := connectsvc.NewJobService(repository.NewJobRepository(c.SQLXDB))
	strategyExperimentConnect := connectsvc.NewStrategyExperimentService(repository.NewStrategyExperimentRepository(c.SQLXDB), repository.NewJobRepository(c.SQLXDB), c.TemplateSvc)
	strategyAssetConnect := connectsvc.NewStrategyAssetService(repository.NewStrategyAssetRepository(c.SQLXDB), c.TemplateSvc)
	marketRegimeConnect := connectsvc.NewMarketRegimeService(repository.NewMarketRegimeRepository(c.SQLXDB), c.KlineSvc)

	authInterceptor := interceptor.NewAuthInterceptor(s.cfg.JWT.Secret, c.APIKeySvc)
	registerDebateV2AdvanceSSE(mux, authInterceptor, c.DebateV2Svc)
	registerDebateV2ChatSSE(mux, authInterceptor, c.DebateV2Svc)

	connecttransport.Register(mux, connecttransport.Services{
		Auth:               authConnect,
		Account:            accountConnect,
		Trading:            tradingConnect,
		Market:             marketConnect,
		Stream:             streamConnect,
		Strategy:           strategyConnect,
		AI:                 aiConnect,
		AdminUser:          adminConnect,
		AdminAccount:       adminConnect,
		AdminTrading:       adminConnect,
		AdminConfig:        adminConnect,
		AdminLog:           adminConnect,
		AdminSystem:        adminConnect,
		Analytics:          analyticsConnect,
		PythonStrategy:     pythonConnect,
		BacktestDataset:    datasetConnect,
		BacktestTrades:     backtestTradesConnect,
		DebateV2:           debateV2Connect,
		DebateV2Stream:     debateV2StreamConnect,
		AIPrimary:          aiPrimaryConnect,
		CodeAssist:         codeAssistConnect,
		AutoTrading:        autoTradingConnect,
		Log:                logConnect,
		ScheduleHealth:     scheduleHealthConnect,
		SystemAI:           systemAIConnect,
		ObjectiveScore:     objectiveScoreConnect,
		IndicatorCatalog:   indicatorCatalogConnect,
		EconomicData:       economicDataConnect,
		Agent:              agentConnect,
		Job:                jobConnect,
		StrategyExperiment: strategyExperimentConnect,
		StrategyAsset:      strategyAssetConnect,
		MarketRegime:       marketRegimeConnect,
	}, authInterceptor)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization,Connect-Protocol-Version,Connect-Timeout-Ms,Grpc-Timeout,Connect-Accept-Encoding")
		w.Header().Set("Access-Control-Expose-Headers", "Connect-Error-Code,Connect-Error-Message")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		mux.ServeHTTP(w, r)
	})
}

func (s *Server) Stop(ctx context.Context) error {
	if s.streamSvc != nil {
		s.streamSvc.StopAllSupervisors()
	}

	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				_ = s.httpServer.Close()
			} else {
				return fmt.Errorf("failed to shutdown HTTP server: %w", err)
			}
		}
	}

	if s.lifecycle != nil {
		s.lifecycle.Stop()
	}

	if s.container != nil && s.container.DB != nil {
		s.container.DB.Close()
	}

	if s.container != nil && s.container.SQLXDB != nil {
		s.container.SQLXDB.Close()
	}

	if s.container != nil && s.container.RedisClient != nil {
		s.container.RedisClient.Close()
	}

	return nil
}
