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

	gate := setupRiskGate(cfg, jurisGate, capStore)
	srv.SetGate(gate)
	mthubSvc.SetGate(gate)
	posCache := strategy.NewPositionCache(log)
	srv.SetPositionCache(posCache)
	accountProvider := strategy.NewMTAccountStateProvider(hub, log)
	accountProvider.SetPositionCache(posCache)
	srv.SetAccountProvider(accountProvider)
	log.Info("D6-A: risk.Gate + AccountStateProvider + PositionCache injected into StrategyExecutionServer")

	gateEvalRepo := repository.NewGateEvaluationRepository(pool)
	srv.SetOnBacktestComplete(makeOnBacktestComplete(gateEvalRepo, mktplaceSvc, notifSender, aiSvc, backtestRunRepo, log))
	srv.SetQualityValidator(mktplaceSvc)
	srv.SetGateEvalRepo(gateEvalRepo)
	return srv
}

func setupRiskGate(cfg *config.Config, jurisGate *risksvc.JurisdictionGate, capStore *risksvc.CapabilityStore) *risk.Gate {
	gate := risk.NewGateWithSystemRules()
	gate.SetKillSwitch(func() bool { return cfg.RiskGateKillSwitch })
	gate.SetAutotradeEnabled(func(uid string) bool { return cfg.RiskGateAutotradeEnabled })
	if jurisGate != nil {
		gate.AddRule(&risk.KycJurisdictionGateRule{
			Gate:       jurisGate,
			UserIDFn:   usermgr.GetUserID,
			ClientIPFn: func(ctx context.Context) string { return "" },
		})
	}
	if capStore != nil {
		gate.AddRule(risk.NewCapabilityTierRule(capStore))
	}
	return gate
}

func makeOnBacktestComplete(
	gateEvalRepo *repository.GateEvaluationRepository,
	mktplaceSvc *marketplace.Service,
	notifSender *notifpubsub.Sender,
	aiSvc *systemai.Service,
	backtestRunRepo *repository.BacktestRunRepository,
	log *zap.Logger,
) func(ctx context.Context, run *repository.BacktestRun) {
	return func(ctx context.Context, run *repository.BacktestRun) {
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

		dailyReturns := internalai.EquityCurveToDailyReturns(run.ProtoResponse)
		if len(dailyReturns) < 10 {
			return
		}
		input := strategy.BuildPipelineInputFromRepo(ctx, gateEvalRepo, run, dailyReturns)
		result := internalai.Pipeline(input)

		if run.AutoGate {
			persistAutoGateResults(ctx, gateEvalRepo, mktplaceSvc, run, result, log)
		}

		ai.SendGateNotification(ctx, notifSender, run.UserID, run, result)
		if result.Passed {
			return
		}
		go autoFixCode(context.WithoutCancel(ctx), run, result, aiSvc, backtestRunRepo, notifSender, log)
	}
}

func persistAutoGateResults(
	ctx context.Context,
	gateEvalRepo *repository.GateEvaluationRepository,
	mktplaceSvc *marketplace.Service,
	run *repository.BacktestRun,
	result internalai.PipelineResult,
	log *zap.Logger,
) {
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
