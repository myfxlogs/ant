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
	"alphaforge/internal/marketplace"
	"alphaforge/internal/mthub"
	notifpubsub "alphaforge/internal/notification"
	"alphaforge/internal/pglisten"
	papereng "alphaforge/internal/paper"
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

// setupStrategyAndTrading wires strategy service, paper trading, risk pipeline,
// autotrading, and schedule engine. Returns deps needed by downstream handlers.
func setupStrategyAndTrading(
	ctx context.Context,
	mux *http.ServeMux,
	pool *pgxpool.Pool,
	cfg *config.Config,
	marketDataRepo repository.MarketDataStore,
	mthubSvc *mthub.MtHubService,
	hub *mthub.Hub,
	eventStore *mthub.TradeEventStore,
	aiSvc *systemai.Service,
	mktplaceSvc *marketplace.Service,
	mktplaceHandler *mktplace.MarketplaceServer,
	quotaChecker *service.QuotaChecker,
	templatesRepo *repository.AIStrategyTemplatesRepository,
	backtestRunRepo *repository.BacktestRunRepository,
	log *zap.Logger,
	otelInterceptor connectrpc.Interceptor,
	authInterceptor connectrpc.Interceptor,
) strategyRuntimeDeps {
	strategySvc := service.NewStrategySvc(pool)
	strategyServer := strategy.NewStrategyServer(strategySvc, log)
	strategyServer.SetCodeAccessChecker(mktplaceSvc)
	pgListen := pglisten.New(pool, log)
	strategyServer.SetPgListen(pgListen)
	mktplaceHandler.SetPgListen(pgListen)
	mktplaceHandler.SetPgPool(pool)
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

	scheduleRepo := repository.NewStrategyScheduleRepository(pool)
	scheduleEngine := strategy.NewScheduleEngine(scheduleRepo, templatesRepo,
		strategyExecServer,
		func(userID uuid.UUID) bool {
			settings, err := autoTradingRepo.GetGlobalSettingsByUserID(context.Background(), userID)
			if err != nil {
				return true
			}
			return settings.AutoTradeEnabled
		},
		log)
	strategyServer.SetEngine(scheduleEngine)

	return strategyRuntimeDeps{
		strategyServer:     strategyServer,
		strategyExecServer: strategyExecServer,
		notifSender:        notifSender,
		pgListen:           pgListen,
		scheduleEngine:     scheduleEngine,
		platformAgg:        platformAgg,
	}
}
