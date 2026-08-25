package strategy

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"alphaforge/internal/ai"
	"alphaforge/internal/pglisten"
	"alphaforge/internal/repository"
	systemai "alphaforge/internal/service/systemai"
)

// ExperimentWorker polls for PENDING experiments and processes them.
type ExperimentWorker struct {
	repo           *repository.StrategyExperimentRepository
	marketDataRepo repository.MarketDataStore
	executor       BacktestExecutor // in-process backtest execution (no DB record)
	log            *zap.Logger
	systemAISvc    *systemai.Service  // optional: enables AI multi-round proposal
	pgListen       *pglisten.Listener // optional: push-first experiment dispatch
	stopCh         chan struct{}
}

func NewExperimentWorker(
	repo *repository.StrategyExperimentRepository,
	marketDataRepo repository.MarketDataStore,
	log *zap.Logger,
) *ExperimentWorker {
	return &ExperimentWorker{
		repo:           repo,
		marketDataRepo: marketDataRepo,
		log:            log,
		stopCh:         make(chan struct{}),
	}
}

func (w *ExperimentWorker) Start(ctx context.Context) {
	go func() {
		notifCh, listenCancel, _ := w.pgListen.Listen(ctx, "experiment_status")
		if listenCancel != nil {
			defer listenCancel()
		}
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-w.stopCh:
				return
			case <-ctx.Done():
				return
			case <-notifCh:
				w.processOneSafe(ctx)
			case <-ticker.C:
				w.processOneSafe(ctx)
			}
		}
	}()
}

// SetPgListen enables push-first dispatch via PG LISTEN/NOTIFY.
func (w *ExperimentWorker) SetPgListen(l *pglisten.Listener) { w.pgListen = l }

// processOneSafe wraps processOne with panic recovery so a single corrupted
// experiment cannot take down the entire worker loop.
func (w *ExperimentWorker) processOneSafe(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			w.log.Error("experiment worker panic recovered", zap.Any("panic", r))
		}
	}()
	if err := w.processOne(ctx); err != nil {
		w.log.Warn("experiment worker error", zap.Error(err))
	}
}

func (w *ExperimentWorker) Stop() { close(w.stopCh) }

// SetAIService injects the system AI service for AI multi-round proposal (search_method="ai").
func (w *ExperimentWorker) SetAIService(svc *systemai.Service) { w.systemAISvc = svc }

// SetExecutor injects the in-process backtest executor for direct candidate scoring.
func (w *ExperimentWorker) SetExecutor(e BacktestExecutor) { w.executor = e }

// processOne claims and processes a single PENDING experiment.
func (w *ExperimentWorker) processOne(ctx context.Context) error {
	exp, err := w.repo.ClaimPendingExperiment(ctx)
	if err != nil {
		return err
	}
	if exp == nil {
		return nil
	}

	w.log.Info("Processing experiment",
		zap.String("expID", exp.ID.String()),
		zap.String("userID", exp.UserID.String()),
		zap.String("method", exp.SearchMethod),
		zap.Int("maxCandidates", exp.MaxCandidates))

	code := exp.StrategyCode
	if code == "" {
		if err := w.repo.UpdateExperimentStatus(ctx, exp.ID, StatusFailed); err != nil {
			w.log.Error("update experiment status to FAILED failed", zap.Error(err), zap.String("expID", exp.ID.String()))
		}
		ExperimentRunsTotal.WithLabelValues(StatusFailed).Inc()
		return fmt.Errorf("experiment %s has no strategy_code", exp.ID)
	}

	params := ai.ExtractParamsWithAnnotations(code)

	// Detect and persist regime once per experiment
	regime := w.detectRegimeForExperiment(ctx, exp)
	if exp.MarketRegimeRef == "" {
		if err := w.repo.UpdateMarketRegime(ctx, exp.ID, regime.String()); err != nil {
			w.log.Warn("failed to persist regime", zap.Error(err))
		}
	}

	var space ai.ResolvedSpace
	if len(params) > 0 {
		space = ai.NormalizeSpace(params)
	} else {
		// No @param annotations in code — fall back to frontend-submitted parameterSpace.
		space = resolvedSpaceFromParamSpace(exp.ParameterSpace)
		if len(space.Keys) == 0 {
			if err := w.repo.UpdateExperimentStatus(ctx, exp.ID, StatusCompleted); err != nil {
				w.log.Error("update experiment status to COMPLETED failed", zap.Error(err), zap.String("expID", exp.ID.String()))
			}
			ExperimentRunsTotal.WithLabelValues(StatusCompleted).Inc()
			w.log.Info("No tunable params found (no @param annotations and no parameterSpace)", zap.String("id", exp.ID.String()))
			return nil
		}
		// Build pseudo-params from space keys so runOptimizer can use GridSearch/RandomSearch.
		params = paramsFromSpace(space)
		w.log.Info("Using frontend parameterSpace (no @param annotations in code)",
			zap.String("expID", exp.ID.String()), zap.Int("dims", len(space.Keys)))
	}

	candidates, err := w.runOptimizer(ctx, exp, params, space, code, regime)
	if err != nil {
		w.log.Error("optimizer failed", zap.Error(err))
		if err := w.repo.UpdateExperimentStatus(ctx, exp.ID, StatusFailed); err != nil {
			w.log.Error("update experiment status to FAILED failed", zap.Error(err), zap.String("expID", exp.ID.String()))
		}
		ExperimentRunsTotal.WithLabelValues(StatusFailed).Inc()
		return err
	}

	// ── OOS validation for top-K candidates ──
	w.runOOSValidation(ctx, exp, code, candidates, regime)

	// Create candidate records with scores
	for i, c := range candidates {
		paramProto, err := proto.Marshal(paramsToProto(c.Overrides))
		if err != nil {
			w.log.Error("marshal candidate params failed", zap.Error(err), zap.String("expID", exp.ID.String()), zap.Int("idx", i))
			continue
		}
		scoreProto, err := proto.Marshal(scoreComponentsToProto(c.ScoreComponents))
		if err != nil {
			w.log.Error("marshal candidate score failed", zap.Error(err), zap.String("expID", exp.ID.String()), zap.Int("idx", i))
			continue
		}
		record := &repository.StrategyExperimentCandidate{
			ID:              uuid.New(),
			ExperimentID:    exp.ID,
			Parameters:      paramProto,
			Rank:            i + 1,
			Score:           c.Score,
			Grade:           c.Grade,
			ScoreComponents: scoreProto,
			Summary:         c.Summary,
			BacktestRunID:   c.BacktestRunID,
			TotalReturn:     c.TotalReturn,
			AnnualReturn:    c.AnnualReturn,
			SharpeRatio:     c.SharpeRatio,
			MaxDrawdown:     c.MaxDrawdown,
			WinRate:         c.WinRate,
			ProfitFactor:    c.ProfitFactor,
			TotalTrades:     c.TotalTrades,
			OOSScore:        c.OOSScore,
			OOSTotalReturn:  c.OOSTotalReturn,
			OOSSharpeRatio:  c.OOSSharpeRatio,
			DegradationPct:  c.DegradationPct,
			IsOverfit:       c.IsOverfit,
		}
		if err := w.repo.CreateCandidate(ctx, record); err != nil {
			w.log.Warn("create candidate failed", zap.Error(err), zap.Int("idx", i))
		}
	}

	if err := w.repo.UpdateExperimentStatus(ctx, exp.ID, StatusCompleted); err != nil {
		w.log.Error("update experiment status to COMPLETED failed", zap.Error(err), zap.String("expID", exp.ID.String()))
	}
	w.log.Info("Experiment completed", zap.String("id", exp.ID.String()),
		zap.Int("candidates", len(candidates)))
	return nil
}
