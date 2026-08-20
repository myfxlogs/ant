package main

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/config"
	"alphaforge/internal/connect/autotrading"
	mktplace "alphaforge/internal/connect/marketplace"
	paperhdr "alphaforge/internal/connect/paper"
	"alphaforge/internal/connect/strategy"
	"alphaforge/internal/knowledgebase"
	"alphaforge/internal/marketplace"
	"alphaforge/internal/mthub"
	notifpubsub "alphaforge/internal/notification"
	papereng "alphaforge/internal/paper"
	"alphaforge/internal/pglisten"
	"alphaforge/internal/repository"
	"alphaforge/internal/risk"
	"alphaforge/internal/risksvc"
	"alphaforge/internal/service"
	systemai "alphaforge/internal/service/systemai"
	"alphaforge/internal/usermgr"

	connectrpc "connectrpc.com/connect"
)

// strategyRuntimeDeps holds strategy + trading dependencies created by setupStrategyAndTrading.
type strategyRuntimeDeps struct {
	strategyServer     *strategy.StrategyServer
	strategyExecServer *strategy.StrategyExecutionServer
	notifSender        *notifpubsub.Sender
	pgListen           *pglisten.Listener
	scheduleEngine     *strategy.ScheduleEngine
	platformAgg        *risksvc.PlatformAggregator
}

// strategyTradingParams holds parameters for setupStrategyAndTrading.
type strategyTradingParams struct {
	Ctx             context.Context
	Mux             *http.ServeMux
	Pool            *pgxpool.Pool
	Cfg             *config.Config
	MarketDataRepo  repository.MarketDataStore
	MthubSvc        *mthub.MtHubService
	Hub             *mthub.Hub
	EventStore      *mthub.TradeEventStore
	AISvc           *systemai.Service
	MktplaceSvc     *marketplace.Service
	MktplaceHandler *mktplace.MarketplaceServer
	QuotaChecker    *service.QuotaChecker
	BacktestRunRepo *repository.BacktestRunRepository
	BoundSvc        *service.BoundAccountService
	Log             *zap.Logger
	OtelInterceptor connectrpc.Interceptor
	AuthInterceptor connectrpc.Interceptor
}

// setupStrategyAndTrading wires strategy service, paper trading, risk pipeline,
// autotrading, and schedule engine. Returns deps needed by downstream handlers.
func setupStrategyAndTrading(p strategyTradingParams) strategyRuntimeDeps {
	ctx := p.Ctx
	mux := p.Mux
	pool := p.Pool
	cfg := p.Cfg
	log := p.Log
	marketDataRepo := p.MarketDataRepo
	mthubSvc := p.MthubSvc
	hub := p.Hub
	eventStore := p.EventStore
	aiSvc := p.AISvc
	mktplaceSvc := p.MktplaceSvc
	mktplaceHandler := p.MktplaceHandler
	quotaChecker := p.QuotaChecker
	backtestRunRepo := p.BacktestRunRepo
	boundSvc := p.BoundSvc
	otelInterceptor := p.OtelInterceptor
	authInterceptor := p.AuthInterceptor
	strategySvc := service.NewStrategySvc(pool)
	strategyServer := strategy.NewStrategyServer(strategySvc, log)
	strategyServer.SetCodeAccessChecker(mktplaceSvc)
	strategyServer.SetBoundSvc(boundSvc)
	pgListen := pglisten.New(pool, log)
	strategyServer.SetPgListen(pgListen)
	mktplaceHandler.SetPgListen(pgListen)
	mktplaceHandler.SetPgPool(pool)

	kbSvc := initKnowledgeBase(ctx, pool, pgListen, log)
	mktplaceSvc.SetDemandRecorder(kbSvc)
	quotaChecker.SetPgListen(pgListen)
	quotaChecker.StartRefreshLoop(ctx)
	mux.Handle(antv1c.NewStrategyServiceHandler(strategyServer, withSency(otelInterceptor, authInterceptor)))

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
	strategyExecServer := configureStrategyExecution(strategyExecDeps{
		pool:            pool,
		backtestRunRepo: backtestRunRepo,
		marketDataRepo:  marketDataRepo,
		mthubSvc:        mthubSvc,
		hub:             hub,
		paperEngine:     paperEngine,
		notifSender:     notifSender,
		aiSvc:           aiSvc,
		pgListen:        pgListen,
		jurisGate:       jurisGate,
		capStore:        capStore,
		quotaChecker:    quotaChecker,
		mktplaceSvc:     mktplaceSvc,
		boundSvc:        boundSvc,
		strategyServer:  strategyServer,
		cfg:             cfg,
		log:             log,
	})
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

	autoTradingRepo, autoTradingServer := setupAutoTrading(pool, mux, strategyExecServer, log, otelInterceptor, authInterceptor)

	scheduleRepo := repository.NewStrategyScheduleRepository(pool)
	scheduleEngine := strategy.NewScheduleEngine(scheduleRepo, strategySvc,
		strategyExecServer,
		func(userID uuid.UUID) bool {
			settings, err := autoTradingRepo.GetGlobalSettingsByUserID(context.Background(), userID)
			if err != nil {
				return true
			}
			return settings.AutoTradeEnabled
		},
		func(ctx context.Context, userID, strategyID string) bool {
			ok, _ := mktplaceSvc.CanAccessCode(ctx, userID, strategyID)
			return ok
		},
		log)
	strategyServer.SetEngine(scheduleEngine)

	// SCHEDULE-HOTLOOP-1: wire autoTrade cache invalidation callback.
	// ToggleAutoTrade and UpdateGlobalSettings call this after DB success to
	// invalidate the schedule engine's autoTrade cache + recompute the timer,
	// preventing a 30s TTL window where disabled autoTrade still dispatches.
	autoTradingServer.SetOnAutoTradeChanged(func(userID uuid.UUID) {
		scheduleEngine.InvalidateAutoTradeCache(userID)
		scheduleEngine.Notify()
	})

	registerDivergenceAndDecayServices(ctx, mux, pool, backtestRunRepo, mktplaceSvc, pgListen, log, otelInterceptor, authInterceptor)

	return strategyRuntimeDeps{
		strategyServer:     strategyServer,
		strategyExecServer: strategyExecServer,
		notifSender:        notifSender,
		pgListen:           pgListen,
		scheduleEngine:     scheduleEngine,
		platformAgg:        platformAgg,
	}
}

func registerDivergenceAndDecayServices(
	ctx context.Context,
	mux *http.ServeMux,
	pool *pgxpool.Pool,
	backtestRunRepo *repository.BacktestRunRepository,
	mktplaceSvc *marketplace.Service,
	pgListen *pglisten.Listener,
	log *zap.Logger,
	otelInterceptor, authInterceptor connectrpc.Interceptor,
) {
	tradeRecordRepo := repository.NewTradeRecordRepository(pool)
	divergenceServer := strategy.NewDivergenceServer(backtestRunRepo, tradeRecordRepo, log)
	divergenceServer.SetPgListen(pgListen)
	mux.Handle(antv1c.NewLiveBacktestDivergenceServiceHandler(divergenceServer,
		withSency(otelInterceptor, authInterceptor)))

	walkForwardServer := strategy.NewWalkForwardServer(backtestRunRepo, log)
	walkForwardServer.SetPgListen(pgListen)
	mux.Handle(antv1c.NewWalkForwardServiceHandler(walkForwardServer,
		withSency(otelInterceptor, authInterceptor)))

	decayMonitor := marketplace.NewDecayMonitor(mktplaceSvc, pgListen, log)
	go decayMonitor.Start(ctx)
}

func initKnowledgeBase(ctx context.Context, pool *pgxpool.Pool, pgListen *pglisten.Listener, log *zap.Logger) *knowledgebase.Service {
	kbSvc := knowledgebase.New(pool, pgListen, log)
	if err := kbSvc.Start(ctx); err != nil {
		log.Warn("kb service startup failed, falling back to built-in constants", zap.Error(err))
	}
	return kbSvc
}

func setupAutoTrading(pool *pgxpool.Pool, mux *http.ServeMux, strategyExecServer *strategy.StrategyExecutionServer, log *zap.Logger, otelInterceptor, authInterceptor connectrpc.Interceptor) (*repository.AutoTradingRepository, *autotrading.AutoTradingServer) {
	autoTradingRepo := repository.NewAutoTradingRepository(pool)
	autoTradingServer := autotrading.NewAutoTradingServer(autoTradingRepo, nil, log)
	strategyExecServer.AddGateRule(&risk.UserRiskConfigRule{Store: func(ctx context.Context, accountID string) (*risk.UserRiskConfig, error) {
		aid, err := uuid.Parse(accountID)
		if err != nil {
			return nil, nil
		}
		if rc, err := autoTradingRepo.GetRiskConfigByAccountID(ctx, aid); err == nil && rc != nil {
			return &risk.UserRiskConfig{
				MaxLotSize: rc.MaxLotSize, MaxPositions: rc.MaxPositions,
				MaxDailyLoss:       rc.MaxDailyLoss,
				MaxDrawdownPercent: rc.MaxDrawdownPercent,
				MaxRiskPercent:     rc.MaxRiskPercent,
			}, nil
		}
		uid, _ := uuid.Parse(usermgr.GetUserID(ctx))
		if uid != uuid.Nil {
			if gs, err := autoTradingRepo.GetGlobalSettingsByUserID(ctx, uid); err == nil && gs != nil {
				return &risk.UserRiskConfig{
					MaxLotSize: gs.MaxLotSize, MaxPositions: gs.MaxPositions,
					MaxDailyLoss:       gs.MaxDailyLoss,
					MaxDrawdownPercent: gs.MaxDrawdownPercent,
					MaxRiskPercent:     gs.MaxRiskPercent,
				}, nil
			}
		}
		return nil, nil
	}})
	mux.Handle(antv1c.NewAutoTradingServiceHandler(autoTradingServer,
		withSency(otelInterceptor, authInterceptor)))
	return autoTradingRepo, autoTradingServer
}
