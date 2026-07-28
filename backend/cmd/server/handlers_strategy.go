package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	internalai "alphaforge/internal/ai"
	"alphaforge/internal/config"
	"alphaforge/internal/connect/ai"
	"alphaforge/internal/connect/strategy"
	"alphaforge/internal/marketplace"
	"alphaforge/internal/mthub"
	notifpubsub "alphaforge/internal/notification"
	"alphaforge/internal/paper"
	"alphaforge/internal/pglisten"
	"alphaforge/internal/repository"
	"alphaforge/internal/risk"
	"alphaforge/internal/risksvc"
	"alphaforge/internal/service"
	systemai "alphaforge/internal/service/systemai"
	"alphaforge/internal/usermgr"
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
	quotaChecker *service.QuotaChecker,
	mktplaceSvc *marketplace.Service,
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
	srv.SetVersionRepo(repository.NewStrategyVersionRepository(pool))
	srv.SetSessionRegistry(strategy.NewSessionRegistry())
	srv.SetQuotaChecker(quotaChecker)

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
	srv.SetGate(gate)      // live_runner startup guard only (gate runs in mthub now)
	mthubSvc.SetGate(gate) // D6-A single chokepoint: all orders through mthub
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
	// This callback is the SINGLE source of truth for pipeline computation:
	// it always computes the 7-gate pipeline, persists results for auto_gate runs
	// (so restoreGateEvaluation can read from DB), and sends notification + auto-fix.
	// If a concurrent restoreGateEvaluation already persisted results, it reads from DB
	// to avoid duplicate computation.
	gateEvalRepo := repository.NewGateEvaluationRepository(pool)
	onBacktestComplete := func(ctx context.Context, run *repository.BacktestRun) {
		// For auto_gate runs: check if restoreGateEvaluation already persisted.
		if run.AutoGate {
			ge, err := gateEvalRepo.GetByRunID(ctx, run.ID)
			if err == nil && ge != nil {
				result := internalai.PipelineResult{
					Passed:    ge.Passed,
					FirstFail: internalai.GateName(ge.FirstFail),
					Summary:   ge.Summary,
				}
				ai.SendGateNotification(ctx, notifSender, run.UserID, run, result)
				if !ge.Passed {
					go autoFixCode(context.WithoutCancel(ctx), run, result, aiSvc, backtestRunRepo, notifSender, log)
				}
				return
			}
		}

		// Compute pipeline (always — for both auto_gate and non-auto-gate runs).
		// Use the same buildPipelineInput as sendAutoGateUpdate for consistency.
		dailyReturns := internalai.EquityCurveToDailyReturns(run.ProtoResponse)
		if len(dailyReturns) < 10 {
			return
		}
		input := strategy.BuildPipelineInputFromRepo(ctx, gateEvalRepo, run, dailyReturns)
		result := internalai.Pipeline(input)

		// For auto_gate runs: persist gate results so restoreGateEvaluation reads from DB.
		if run.AutoGate {
			// Compute quality preview (same as sendAutoGateUpdate).
			var qualityPreview *antv1.MarketplaceQualityPreview
			if mktplaceSvc != nil && len(run.BacktestSnapshot) > 0 {
				strategyID := ""
				if run.TemplateID != nil {
					strategyID = run.TemplateID.String()
				}
				violations, err := mktplaceSvc.ValidateBacktestQuality(ctx, run.BacktestSnapshot, strategyID)
				if err != nil {
					log.Warn("onBacktestComplete: marketplace quality preview failed", zap.Error(err))
				} else {
					qualityPreview = strategy.ViolationsToPreview(violations)
				}
			}
			// Unify publishability: 7-gate pass AND no marketplace violations.
			if qualityPreview != nil && !result.Passed {
				qualityPreview.Publishable = false
			}

			gateSummary := strategy.BuildGateSummaryProto(&result)
			gateBytes, _ := proto.Marshal(gateSummary)
			gateList := strategy.BuildGateListProto(&result)
			var gateResultsBytes []byte
			if gateList != nil {
				gateResultsBytes, _ = proto.Marshal(gateList)
			}
			var qualityBytes []byte
			publishable := false
			if qualityPreview != nil {
				qualityBytes, _ = proto.Marshal(qualityPreview)
				publishable = qualityPreview.Publishable
			}
			if err := gateEvalRepo.Upsert(ctx, run.UserID, run.ID, gateBytes, gateResultsBytes, qualityBytes, result.Passed, string(result.FirstFail), result.Summary, publishable); err != nil {
				log.Warn("onBacktestComplete: failed to persist gate evaluation", zap.Error(err))
			}
		}

		ai.SendGateNotification(ctx, notifSender, run.UserID, run, result)
		if result.Passed {
			return
		}
		go autoFixCode(context.WithoutCancel(ctx), run, result, aiSvc, backtestRunRepo, notifSender, log)
	}
	srv.SetOnBacktestComplete(onBacktestComplete)
	srv.SetQualityValidator(mktplaceSvc)
	srv.SetGateEvalRepo(repository.NewGateEvaluationRepository(pool))
	return srv
}
