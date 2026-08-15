package main

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	internalai "alphaforge/internal/ai"
	"alphaforge/internal/config"
	"alphaforge/internal/connect/ai"
	"alphaforge/internal/connect/strategy"
	"alphaforge/internal/interceptor"
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
type strategyExecDeps struct {
	pool            *pgxpool.Pool
	backtestRunRepo *repository.BacktestRunRepository
	marketDataRepo  repository.MarketDataStore
	mthubSvc        *mthub.MtHubService
	hub             *mthub.Hub
	paperEngine     *paper.PaperEngine
	notifSender     *notifpubsub.Sender
	aiSvc           *systemai.Service
	pgListen        *pglisten.Listener
	jurisGate       *risksvc.JurisdictionGate
	capStore        *risksvc.CapabilityStore
	quotaChecker    *service.QuotaChecker
	mktplaceSvc     *marketplace.Service
	boundSvc        *service.BoundAccountService
	strategyServer  *strategy.StrategyServer
	cfg             *config.Config
	log             *zap.Logger
}

func configureStrategyExecution(d strategyExecDeps) *strategy.StrategyExecutionServer {
	srv := strategy.NewStrategyExecutionServer(d.backtestRunRepo, d.log)
	srv.SetPgListen(d.pgListen)
	srv.SetMarketDataRepo(d.marketDataRepo)
	srv.SetBarSource(strategy.NewLiveSource(d.mthubSvc, d.marketDataRepo))
	srv.SetMtHub(d.mthubSvc)
	// GoExecutor removed per Gap 3 — all strategies route through Bytecode VM.
	strategyRunRepo := repository.NewStrategyRunRepository(d.pool)
	srv.SetRunRepo(strategyRunRepo)
	srv.SetImportedRepo(repository.NewImportedStrategyRepository(d.pool))
	srv.SetVersionRepo(repository.NewStrategyVersionRepository(d.pool))
	srv.SetFailureSignatureRepo(repository.NewFailureSignatureRepository(d.pool))
	reg := strategy.NewSessionRegistry()
	reg.SetLogger(d.log)
	reg.SetLogRepository(repository.NewLogRepository(d.pool))
	reg.SubscribeToMthub(d.mthubSvc)
	srv.SetSessionRegistry(reg)
	if d.strategyServer != nil {
		d.strategyServer.SetSessionRegistry(reg)
	}
	srv.SetScheduleNameLookup(func(ctx context.Context, scheduleID uuid.UUID) string {
		var name string
		err := d.pool.QueryRow(ctx, `SELECT name FROM strategy_schedules WHERE id = $1`, scheduleID).Scan(&name)
		if err != nil {
			return ""
		}
		return name
	})
	srv.SetBrokerCompanyLookup(func(ctx context.Context, accountID string) string {
		var broker string
		err := d.pool.QueryRow(ctx,
			`SELECT COALESCE(broker_company,'') FROM mt_accounts WHERE id = $1::uuid AND deleted_at IS NULL`,
			accountID).Scan(&broker)
		if err != nil {
			return ""
		}
		return broker
	})
	srv.SetQuotaChecker(d.quotaChecker)
	if d.boundSvc != nil {
		srv.SetBoundSvc(d.boundSvc)
	}

	// Task 4 (CLEANUP-MISFIRE): exclude currently active runs from cleanup.
	// At startup, ListAll() is empty (no sessions registered yet), so all
	// running rows are cleaned up — semantically correct for orphan recovery.
	// Wrongly-marked rows self-heal: registerLiveSession calls runRepo.MarkRunning.
	excludeIDs := make([]uuid.UUID, 0, len(reg.ListAll()))
	for _, sess := range reg.ListAll() {
		excludeIDs = append(excludeIDs, sess.RunID)
	}
	if n, err := strategyRunRepo.CleanupStaleRuns(context.Background(), excludeIDs); err != nil {
		d.log.Warn("startup: failed to cleanup stale strategy runs", zap.Error(err))
	} else if n > 0 {
		d.log.Info("startup: cleaned up stale strategy runs", zap.Int64("count", n))
	}
	srv.SetAccountLookup(func(ctx context.Context, userID string) string {
		var mt4ID string
		err := d.pool.QueryRow(ctx,
			`SELECT id::text FROM mt_accounts
			 WHERE user_id = $1::uuid AND account_status = 'connected'
			 ORDER BY created_at LIMIT 1`,
			userID).Scan(&mt4ID)
		if err != nil {
			return ""
		}
		return mt4ID
	})
	srv.StartBacktestWorker(context.Background())
	srv.SetPaperEngine(d.paperEngine)
	srv.SetNotificationSender(d.notifSender)

	gate := setupRiskGate(d.cfg, d.jurisGate, d.capStore)
	srv.SetGate(gate)
	d.mthubSvc.SetGate(gate)
	posCache := strategy.NewPositionCache(d.log)
	srv.SetPositionCache(posCache)
	if d.strategyServer != nil {
		d.strategyServer.SetPositionCache(posCache)
	}
	accountProvider := strategy.NewMTAccountStateProvider(d.hub, d.log)
	accountProvider.SetPositionCache(posCache)
	srv.SetAccountProvider(accountProvider)
	d.log.Info("D6-A: risk.Gate + AccountStateProvider + PositionCache injected into StrategyExecutionServer")

	gateEvalRepo := repository.NewGateEvaluationRepository(d.pool)
	srv.SetOnBacktestComplete(makeOnBacktestComplete(gateEvalRepo, d.mktplaceSvc, d.notifSender, d.aiSvc, d.backtestRunRepo, d.log))
	srv.SetQualityValidator(d.mktplaceSvc)
	srv.SetCoverageChecker(d.mktplaceSvc)
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
			ClientIPFn: interceptor.GetClientIP,
		})
	}
	if capStore != nil {
		gate.AddRule(risk.NewCapabilityTierRule(capStore))
	}
	// D6-A: consolidate all pre-trade risk checks into the single Gate chokepoint.
	// These rules replace the former risksvc.PreCheck call in submitToBroker.
	gate.AddRule(&risk.MaxPositionCount{Max: 20})
	gate.AddRule(&risk.MaxLotSize{MaxLots: decimal.NewFromInt(100000)})
	gate.AddRule(&risk.MarginPreCheck{MaxMarginRatio: decimal.NewFromFloat(0.80)})
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
