package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"anttrader/internal/ai"
	"anttrader/internal/cache"
	"anttrader/internal/config"
	"anttrader/internal/connection"
	"anttrader/internal/monitoring"
	"anttrader/internal/pkg/secretbox"
	"anttrader/internal/repository"
	"anttrader/internal/service"
	"anttrader/internal/service/systemai"
	"anttrader/internal/stream"
	"anttrader/pkg/logger"
)

type Container struct {
	DB                *pgxpool.Pool
	SQLXDB            *sqlx.DB
	RedisClient       *redis.Client
	ConnMgr           *connection.ConnectionManager
	StreamMgr         *stream.Manager
	BacktestWorker    *service.BacktestRunWorker
	ScheduleRunner    *service.StrategyScheduleRunner
	CacheService      *cache.CacheService
	MonitoringService *monitoring.MonitoringService
	SystemCollector   *monitoring.SystemMetricsCollector
	APIKeySvc         *service.APIKeyService
	DynamicConfigSvc  *service.DynamicConfigService
	AICfgSvc          *service.AIConfigService
	AIAgentSvc        *service.AIAgentService
	SystemAISvc       *systemai.Service
	UserRepo          *repository.UserRepository
	AccountRepo       *repository.AccountRepository
	KlineRepo         *repository.KlineRepository
	TradeLogRepo      *repository.TradeLogRepository
	TradeRecordRepo   *repository.TradeRecordRepository
	AnalyticsRepo     *repository.AnalyticsRepository
	StrategyRepo      *repository.StrategyRepository
	AdminRepo         *repository.AdminRepository
	AutoTradingRepo   *repository.AutoTradingRepository
	TemplateRepo      *repository.StrategyTemplateRepository
	ScheduleRepo      *repository.StrategyScheduleRepository
	LogRepo           *repository.LogRepository
	AuthSvc           *service.AuthService
	AccountSvc        *service.AccountService
	TradingSvc        *service.TradingService
	MarketSvc         *service.MarketService
	KlineSvc          *service.KlineService
	AnalyticsSvc      *service.AnalyticsService
	BrokerSvc         *service.BrokerService
	AIManager         *ai.Manager
	AdvisorSvc        *service.AdvisorService
	ReportSvc         *service.AIReportService
	PythonSvc         *service.PythonStrategyService
	BacktestDataset   *service.BacktestDatasetService
	TickDataset       *service.TickDatasetService
	BacktestRunSvc    *service.BacktestRunService
	AdminSvc          *service.AdminService
	AutoTradingSvc    *service.AutoTradingService
	TemplateSvc       *service.StrategyTemplateService
	ScheduleSvc       *service.StrategyScheduleService
	LogSvc            *service.LogService
	EconCalendarSvc   *service.EconomicCalendarService
	DebateV2Svc       *service.DebateV2Service
	AICodeAssist      *service.AICodeAssistService
}

func NewContainer(cfg *config.Config) (*Container, error) {
	db, err := repository.NewDBPool(&cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	sqlxDB, err := repository.NewSQLXDB(&cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database with sqlx: %w", err)
	}
	redisClient := redis.NewClient(&redis.Options{Addr: fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port), Password: cfg.Redis.Password, DB: cfg.Redis.DB, PoolSize: cfg.Redis.PoolSize, MinIdleConns: cfg.Redis.MinIdleConns, MaxRetries: cfg.Redis.MaxRetries, DialTimeout: cfg.Redis.DialTimeout, ReadTimeout: cfg.Redis.ReadTimeout, WriteTimeout: cfg.Redis.WriteTimeout, PoolTimeout: cfg.Redis.PoolTimeout})
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		logger.Warn("Failed to connect to Redis", zap.Error(err))
	}
	var cacheService *cache.CacheService
	if cfg.Cache.Enabled {
		cacheService = cache.NewCacheServiceWithClient(redisClient, &cfg.Redis)
	}
	var monitoringService *monitoring.MonitoringService
	var systemCollector *monitoring.SystemMetricsCollector
	if cfg.Monitoring.Enabled {
		monitoringService = monitoring.NewMonitoringService(redisClient, &cfg.Monitoring)
		systemCollector = monitoring.NewSystemMetricsCollector(redisClient, monitoringService)
	}
	userRepo := repository.NewUserRepository(db)
	accountRepo := repository.NewAccountRepository(sqlxDB)
	tradeLogRepo := repository.NewTradeLogRepository(sqlxDB)
	tradeRecordRepo := repository.NewTradeRecordRepository(sqlxDB)
	klineRepo := repository.NewKlineRepository(sqlxDB)
	analyticsRepo := repository.NewAnalyticsRepository(sqlxDB)
	strategyRepo := repository.NewStrategyRepository(sqlxDB)
	adminRepo := repository.NewAdminRepository(db, sqlxDB)
	autoTradingRepo := repository.NewAutoTradingRepository(sqlxDB)
	templateRepo := repository.NewStrategyTemplateRepository(sqlxDB)
	scheduleRepo := repository.NewStrategyScheduleRepository(sqlxDB)
	logRepo := repository.NewLogRepository(sqlxDB)
	backtestDatasetRepo := repository.NewBacktestDatasetRepository(sqlxDB)
	backtestRunRepo := repository.NewBacktestRunRepository(sqlxDB)
	tickDatasetRepo := repository.NewTickDatasetRepository(sqlxDB)
	systemAIRepo := repository.NewSystemAIConfigRepository(db)
	aiAgentRepo := repository.NewAIAgentDefinitionRepository(db)
	apiKeySvc := service.NewAPIKeyService(repository.NewAPIKeyRepository(sqlxDB))
	authSvc := service.NewAuthService(userRepo)
	connMgr := connection.NewConnectionManager(accountRepo, tradeRecordRepo, logRepo, &cfg.MT4, &cfg.MT5)
	streamMgr := stream.NewManager()
	dynamicConfigSvc := service.NewDynamicConfigService(db)
	logSvc := service.NewLogService(logRepo)
	accountSvc := service.NewAccountService(accountRepo, connMgr, &cfg.MT4, &cfg.MT5, dynamicConfigSvc, logSvc)
	tradingSvc := service.NewTradingService(accountRepo, tradeLogRepo, tradeRecordRepo, connMgr, &cfg.MT4, &cfg.MT5)
	engineSelector, execGateway := newExecutionGateway(tradingSvc, accountRepo, logSvc, redisClient)
	marketSvc := service.NewMarketService(accountRepo, &cfg.MT4, &cfg.MT5)
	if cacheService != nil {
		marketSvc.SetQuoteCache(cache.NewRealtimeQuoteCache(cacheService))
	}
	klineSvc := service.NewKlineService(accountRepo, klineRepo, connMgr, &cfg.MT4, &cfg.MT5)
	analyticsSvc := service.NewAnalyticsService(accountRepo, tradeLogRepo, analyticsRepo)
	econCalendarSvc := service.NewEconomicCalendarService(&cfg.FMP, sqlxDB, dynamicConfigSvc)
	brokerSvc := service.NewBrokerService(&cfg.MT4, &cfg.MT5)
	aiManager := ai.NewManager()
	secretBox := secretbox.New([]byte(cfg.JWT.Secret))
	aiCfgSvc := service.NewAIConfigService(systemAIRepo, secretBox, dynamicConfigSvc).WithPrimaryStore(userRepo)
	aiAgentSvc := service.NewAIAgentService(aiAgentRepo)
	systemAISvc := systemai.NewService(systemAIRepo, secretBox)
	advisorSvc := service.NewAdvisorService(aiManager, klineRepo, accountRepo, klineSvc)
	reportSvc := service.NewAIReportService(aiManager, analyticsRepo, accountRepo, tradeRecordRepo, analyticsSvc)
	autoTradingSvc := service.NewAutoTradingService(autoTradingRepo, strategyRepo, accountRepo, engineSelector, execGateway, connMgr)
	execGateway.SetRiskChecker(autoTradingSvc)
	pythonSvc := service.NewPythonStrategyService(cfg.StrategyService.URL)
	backtestRunSvc := service.NewBacktestRunService(backtestRunRepo)
	backtestDatasetSvc := service.NewBacktestDatasetService(backtestDatasetRepo)
	tickDatasetSvc := service.NewTickDatasetService(tickDatasetRepo, redisClient)
	backtestRunWorker := service.NewBacktestRunWorker(backtestRunSvc, pythonSvc, klineSvc, dynamicConfigSvc, backtestDatasetSvc, tickDatasetSvc)
	templateSvc := service.NewStrategyTemplateService(templateRepo)
	scheduleSvc := service.NewStrategyScheduleService(scheduleRepo, templateRepo, accountRepo, dynamicConfigSvc, pythonSvc, klineSvc, backtestDatasetSvc, backtestRunSvc)
	scheduleRunner := service.NewStrategyScheduleRunner(scheduleRepo, templateRepo, accountRepo, connMgr, klineSvc, pythonSvc, execGateway, tradingSvc, logSvc)
	if sqlxDB != nil {
		scheduleRunner.SetStateStore(service.NewPostgresScheduleRuntimeStateStore(sqlxDB))
	}
	adminSvc := service.NewAdminService(adminRepo, userRepo, dynamicConfigSvc)
	debateV2Svc := service.NewDebateV2Service(repository.NewDebateRepository(sqlxDB), aiCfgSvc, aiAgentSvc).WithSystemAI(newSystemAIProviderAdapter(systemAISvc))
	return &Container{DB: db, SQLXDB: sqlxDB, RedisClient: redisClient, ConnMgr: connMgr, StreamMgr: streamMgr, BacktestWorker: backtestRunWorker, ScheduleRunner: scheduleRunner, CacheService: cacheService, MonitoringService: monitoringService, SystemCollector: systemCollector, APIKeySvc: apiKeySvc, DynamicConfigSvc: dynamicConfigSvc, AICfgSvc: aiCfgSvc, AIAgentSvc: aiAgentSvc, SystemAISvc: systemAISvc, UserRepo: userRepo, AccountRepo: accountRepo, KlineRepo: klineRepo, TradeLogRepo: tradeLogRepo, TradeRecordRepo: tradeRecordRepo, AnalyticsRepo: analyticsRepo, StrategyRepo: strategyRepo, AdminRepo: adminRepo, AutoTradingRepo: autoTradingRepo, TemplateRepo: templateRepo, ScheduleRepo: scheduleRepo, LogRepo: logRepo, AuthSvc: authSvc, AccountSvc: accountSvc, TradingSvc: tradingSvc, MarketSvc: marketSvc, KlineSvc: klineSvc, AnalyticsSvc: analyticsSvc, BrokerSvc: brokerSvc, AIManager: aiManager, AdvisorSvc: advisorSvc, ReportSvc: reportSvc, PythonSvc: pythonSvc, BacktestDataset: backtestDatasetSvc, TickDataset: tickDatasetSvc, BacktestRunSvc: backtestRunSvc, AdminSvc: adminSvc, AutoTradingSvc: autoTradingSvc, TemplateSvc: templateSvc, ScheduleSvc: scheduleSvc, LogSvc: logSvc, EconCalendarSvc: econCalendarSvc, DebateV2Svc: debateV2Svc, AICodeAssist: service.NewAICodeAssistService(aiCfgSvc)}, nil
}

func newExecutionGateway(tradingSvc *service.TradingService, accountRepo *repository.AccountRepository, logSvc *service.LogService, redisClient *redis.Client) (*service.EngineSelector, *service.ExecutionGateway) {
	engineSelector := service.NewEngineSelector(tradingSvc, service.NewTickBacktestEngine(service.BacktestCostConfig{}))
	var execStore service.ExecutionIdempotencyStore
	if redisClient != nil {
		execStore = service.NewRedisExecutionIdempotencyStore(redisClient)
	}
	return engineSelector, service.NewExecutionGateway(engineSelector, accountRepo, nil, logSvc, execStore)
}

type systemAIProviderAdapter struct{ svc *systemai.Service }

func newSystemAIProviderAdapter(svc *systemai.Service) *systemAIProviderAdapter {
	return &systemAIProviderAdapter{svc: svc}
}

var systemProviderTypeAliases = map[string]ai.ProviderType{"openai_compatible": ai.ProviderCustom}

func (a *systemAIProviderAdapter) BuildProviderConfig(ctx context.Context, userID uuid.UUID, providerID string) (*service.AIConfig, error) {
	if a == nil || a.svc == nil {
		return nil, errors.New("system ai service unavailable")
	}
	id := strings.TrimSpace(providerID)
	if id == "" {
		return nil, errors.New("empty provider id")
	}
	row, err := a.svc.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, errors.New("system ai provider not found")
	}
	if !row.Enabled {
		return nil, errors.New("system ai provider not enabled")
	}
	if !row.HasSecret {
		return nil, errors.New("system ai provider has no api key configured")
	}
	apiKey, err := a.svc.GetSecret(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	pt, ok := systemProviderTypeAliases[id]
	if !ok {
		pt = ai.ProviderType(id)
	}
	cfg := &service.AIConfig{Provider: pt, APIKey: apiKey, Model: row.DefaultModel, BaseURL: row.BaseURL, Enabled: true}
	if row.Temperature != 0 {
		cfg.Temperature = sql.NullFloat64{Float64: row.Temperature, Valid: true}
	}
	if row.TimeoutSeconds > 0 {
		cfg.TimeoutSeconds = sql.NullInt32{Int32: int32(row.TimeoutSeconds), Valid: true}
	}
	if row.MaxTokens > 0 {
		cfg.MaxTokens = sql.NullInt32{Int32: int32(row.MaxTokens), Valid: true}
	}
	return cfg, nil
}
