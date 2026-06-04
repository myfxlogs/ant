// strategy_experiment_worker.go: background worker that processes PENDING experiments.
// Extracts @params, runs grid/random search, creates candidates with placeholder scores.
// Full backtest execution deferred to Phase 2b (DE/TPE + backtest integration).

package strategy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"anttrader/internal/ai"
	"anttrader/internal/repository"
)

// ExperimentWorker polls for PENDING experiments and processes them.
type ExperimentWorker struct {
	repo   *repository.StrategyExperimentRepository
	log    *zap.Logger
	stopCh chan struct{}
}

func NewExperimentWorker(repo *repository.StrategyExperimentRepository, log *zap.Logger) *ExperimentWorker {
	return &ExperimentWorker{repo: repo, log: log, stopCh: make(chan struct{})}
}

// Start begins the worker loop in a background goroutine.
func (w *ExperimentWorker) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-w.stopCh:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := w.processOne(ctx); err != nil {
					w.log.Warn("experiment worker error", zap.Error(err))
				}
			}
		}
	}()
}

// Stop signals the worker to shut down.
func (w *ExperimentWorker) Stop() { close(w.stopCh) }

// processOne claims and processes a single PENDING experiment.
func (w *ExperimentWorker) processOne(ctx context.Context) error {
	exp, err := w.repo.ClaimPendingExperiment(ctx)
	if err != nil {
		return err
	}
	if exp == nil {
		return nil // no pending experiments
	}
	w.log.Info("Processing experiment", zap.String("id", exp.ID.String()))

	// Get strategy code (from template or directly)
	code, err := w.getStrategyCode(ctx, exp)
	if err != nil {
		_ = w.repo.UpdateExperimentStatus(ctx, exp.ID, "FAILED")
		return fmt.Errorf("get code: %w", err)
	}

	// Extract tunable params from @param annotations
	params := ai.ExtractParams(code)
	if len(params) == 0 {
		_ = w.repo.UpdateExperimentStatus(ctx, exp.ID, "COMPLETED")
		w.log.Info("No tunable params found", zap.String("id", exp.ID.String()))
		return nil
	}

	// Run optimizer
	var candidates []map[string]interface{}
	space := ai.NormalizeSpace(params)
	switch exp.SearchMethod {
	case "random":
		candidates = ai.RandomSearch(params, exp.MaxCandidates)
	case "de":
		candidates = runIterativeOptimizer(ai.NewDEOptimizer(space, exp.MaxCandidates), space)
	case "tpe":
		candidates = runIterativeOptimizer(ai.NewTPEOptimizer(space, exp.MaxCandidates), space)
	default:
		candidates = ai.GridSearch(params, exp.MaxCandidates)
	}

	// Create candidate records
	for i, combo := range candidates {
		paramJSON, _ := json.Marshal(combo)
		c := &repository.StrategyExperimentCandidate{
			ID:           uuid.New(),
			ExperimentID: exp.ID,
			Parameters:   paramJSON,
			Rank:         i + 1,
			Summary:      fmt.Sprintf("Candidate %d", i+1),
			Grade:        "C", // placeholder — real grade comes after backtest
		}
		if err := w.repo.CreateCandidate(ctx, c); err != nil {
			w.log.Warn("create candidate failed", zap.Error(err), zap.Int("idx", i))
		}
	}

	_ = w.repo.UpdateExperimentStatus(ctx, exp.ID, "COMPLETED")
	w.log.Info("Experiment completed", zap.String("id", exp.ID.String()), zap.Int("candidates", len(candidates)))
	return nil
}

// getStrategyCode retrieves strategy code. Template-based experiments are not yet supported.
func (w *ExperimentWorker) getStrategyCode(ctx context.Context, exp *repository.StrategyExperiment) (string, error) {
	// TODO: support experiments with base_template_id by fetching code from template service
	if exp.BaseTemplateID != nil && *exp.BaseTemplateID != uuid.Nil {
		return "", fmt.Errorf("template-based experiments not yet supported (template_id=%s)", exp.BaseTemplateID)
	}
	return "", fmt.Errorf("no code source for experiment %s", exp.ID)
}

// runIterativeOptimizer drives an ask/tell optimizer, converting index vectors to overrides.
// Full backtest scoring is deferred to Phase 2b — currently generates candidates with placeholder scores.
func runIterativeOptimizer(opt ai.Optimizer, space ai.ResolvedSpace) []map[string]interface{} {
	var candidates []map[string]interface{}
	for !opt.Done() {
		batch := opt.Ask(0) // 0 = optimizer decides batch size
		for _, indices := range batch {
			overrides := ai.IndexToOverrides(indices, space)
			candidates = append(candidates, overrides)
			// Placeholder: real score comes from backtest execution
			opt.Tell([]ai.OptimizerResult{{Indices: indices, Score: 0}})
		}
	}
	return candidates
}
