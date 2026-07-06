package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	internalai "anttrader/internal/ai"
	"anttrader/internal/config"
	"anttrader/internal/connect/strategy"
	"anttrader/internal/mthub"
	notifpubsub "anttrader/internal/notification"
	"anttrader/internal/paper"
	"anttrader/internal/pglisten"
	"anttrader/internal/repository"
	"anttrader/internal/risk"
	"anttrader/internal/risksvc"
	"anttrader/internal/connect/ai"
	systemai "anttrader/internal/service/systemai"
	"anttrader/internal/usermgr"
)

// configureStrategyExecution creates the StrategyExecutionServer with all dependencies
// wired in — bar source, paper engine, notification sender, gate, and auto-gate
// callback. Returns the fully configured server ready for handler registration.
func configureStrategyExecution(
	pool *pgxpool.Pool,
	backtestRunRepo *repository.BacktestRunRepository,
	marketDataRepo repository.MarketDataStore,
	mthubSvc *mthub.MtHubService,
	hub *mthub.Hub,
	paperEngine *paper.PaperEngine,
	notifSender *notifpubsub.Sender,
	aiSvc *systemai.Service,
	pgListen *pglisten.Listener,
	jurisGate *risksvc.JurisdictionGate,
	capStore *risksvc.CapabilityStore,
	cfg *config.Config,
	log *zap.Logger,
) *strategy.StrategyExecutionServer {
	srv := strategy.NewStrategyExecutionServer(backtestRunRepo, log)
	srv.SetPgListen(pgListen)
	srv.SetMarketDataRepo(marketDataRepo)
	srv.SetBarSource(strategy.NewLiveSource(mthubSvc, marketDataRepo))
	srv.SetMtHub(mthubSvc)
	srv.SetGoExecutor(strategy.NewGoExecutor(".", log))
	strategyRunRepo := repository.NewStrategyRunRepository(pool)
	srv.SetRunRepo(strategyRunRepo)
	srv.SetImportedRepo(repository.NewImportedStrategyRepository(pool))
	srv.SetSessionRegistry(strategy.NewSessionRegistry())

	// Clean up runs orphaned by a previous crash/restart.
	if n, err := strategyRunRepo.CleanupStaleRuns(context.Background()); err != nil {
		log.Warn("startup: failed to cleanup stale strategy runs", zap.Error(err))
	} else if n > 0 {
		log.Info("startup: cleaned up stale strategy runs", zap.Int64("count", n))
	}
	srv.SetAccountLookup(func(ctx context.Context, userID string) string {
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
	srv.StartBacktestWorker(context.Background())
	srv.SetPaperEngine(paperEngine)
	srv.SetNotificationSender(notifSender)

	// D6-A: risk Gate — system rules only. User trading constraints are opt-in.
	gate := risk.NewGateWithSystemRules()
	gate.SetKillSwitch(func() bool { return cfg.RiskGateKillSwitch })
	gate.SetAutotradeEnabled(func(uid string) bool { return cfg.RiskGateAutotradeEnabled })

	// Wire KYC/Jurisdiction (legal compliance — non-optional when configured).
	if jurisGate != nil {
		gate.AddRule(&risk.KycJurisdictionGateRule{
			Gate:       jurisGate,
			UserIDFn:   usermgr.GetUserID,
			ClientIPFn: func(ctx context.Context) string { return "" },
		})
	}
	// Wire capability tier (per-user trading limits from DB).
	if capStore != nil {
		gate.AddRule(risk.NewCapabilityTierRule(capStore))
	}
	srv.SetGate(gate)       // live_runner startup guard only (gate runs in mthub now)
	mthubSvc.SetGate(gate)   // D6-A single chokepoint: all orders through mthub
	// Push-first: PositionCache subscribes to PositionSnapshotBroker (no per-bar polling).
	posCache := strategy.NewPositionCache(log)
	srv.SetPositionCache(posCache)
	// T3.2b: Inject AccountStateProvider for live trading.
	// Uses push-based PositionCache when available, falls back to FetchOpenedOrders.
	accountProvider := strategy.NewMTAccountStateProvider(hub, log)
	accountProvider.SetPositionCache(posCache)
	srv.SetAccountProvider(accountProvider)
	log.Info("D6-A: risk.Gate + AccountStateProvider + PositionCache injected into StrategyExecutionServer")

	// Auto-gate: runs gate evaluation after every backtest completion.
	// On failure, spawns async auto-fix (LLM code repair → new backtest).
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
		ai.SendGateNotification(ctx, notifSender, run.UserID, run, result)
		if result.Passed {
			return
		}
		go autoFixCode(context.Background(), run, result, aiSvc, backtestRunRepo, notifSender, log)
	}
	srv.SetOnBacktestComplete(onBacktestComplete)
	return srv
}
