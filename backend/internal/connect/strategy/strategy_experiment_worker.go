package strategy

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/ai"
	"anttrader/internal/repository"
)

// ExperimentWorker polls for PENDING experiments and processes them.
type ExperimentWorker struct {
	repo         *repository.StrategyExperimentRepository
	backtestRepo *repository.BacktestRunRepository
	log          *zap.Logger
	stopCh       chan struct{}
}

func NewExperimentWorker(
	repo *repository.StrategyExperimentRepository,
	backtestRepo *repository.BacktestRunRepository,
	log *zap.Logger,
) *ExperimentWorker {
	return &ExperimentWorker{
		repo:         repo,
		backtestRepo: backtestRepo,
		log:          log,
		stopCh:       make(chan struct{}),
	}
}

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

func (w *ExperimentWorker) Stop() { close(w.stopCh) }

// processOne claims and processes a single PENDING experiment.
func (w *ExperimentWorker) processOne(ctx context.Context) error {
	exp, err := w.repo.ClaimPendingExperiment(ctx)
	if err != nil {
		return err
	}
	if exp == nil {
		return nil
	}

	w.log.Info("Processing experiment", zap.String("id", exp.ID.String()),
		zap.String("method", exp.SearchMethod), zap.Int("maxCandidates", exp.MaxCandidates))

	code := exp.StrategyCode
	if code == "" {
		_ = w.repo.UpdateExperimentStatus(ctx, exp.ID, "FAILED")
		return fmt.Errorf("experiment %s has no strategy_code", exp.ID)
	}

	params := ai.ExtractParams(code)
	if len(params) == 0 {
		_ = w.repo.UpdateExperimentStatus(ctx, exp.ID, "COMPLETED")
		w.log.Info("No @params found", zap.String("id", exp.ID.String()))
		return nil
	}

	space := ai.NormalizeSpace(params)
	candidates, err := w.runOptimizer(ctx, exp, params, space, code)
	if err != nil {
		w.log.Error("optimizer failed", zap.Error(err))
		_ = w.repo.UpdateExperimentStatus(ctx, exp.ID, "FAILED")
		return err
	}

	// Create candidate records with scores
	for i, c := range candidates {
		paramProto, _ := proto.Marshal(paramsToProto(c.Overrides))
		scoreProto, _ := proto.Marshal(scoreComponentsToProto(c.ScoreComponents))
		record := &repository.StrategyExperimentCandidate{
			ID:              uuid.New(),
			ExperimentID:    exp.ID,
			Parameters:      paramProto,
			Rank:            i + 1,
			Score:           c.Score,
			Grade:           c.Grade,
			ScoreComponents: scoreProto,
			Summary:         fmt.Sprintf("%s score=%.1f grade=%s", c.Summary, c.Score, c.Grade),
		}
		if err := w.repo.CreateCandidate(ctx, record); err != nil {
			w.log.Warn("create candidate failed", zap.Error(err), zap.Int("idx", i))
		}
	}

	_ = w.repo.UpdateExperimentStatus(ctx, exp.ID, "COMPLETED")
	w.log.Info("Experiment completed", zap.String("id", exp.ID.String()),
		zap.Int("candidates", len(candidates)))
	return nil
}

type candidateResult struct {
	Overrides       map[string]interface{}
	Score           float64
	Grade           string
	ScoreComponents map[string]float64
	Summary         string
}

func (w *ExperimentWorker) runOptimizer(
	ctx context.Context, exp *repository.StrategyExperiment,
	params []ai.TunableParam, space ai.ResolvedSpace, code string,
) ([]candidateResult, error) {
	switch exp.SearchMethod {
	case "de":
		return w.runIterative(ctx, ai.NewDEOptimizer(space, exp.MaxCandidates), space, code, exp)
	case "tpe":
		return w.runIterative(ctx, ai.NewTPEOptimizer(space, exp.MaxCandidates), space, code, exp)
	case "random":
		return w.runOneShot(ai.RandomSearch(params, exp.MaxCandidates), code, exp)
	default:
		return w.runOneShot(ai.GridSearch(params, exp.MaxCandidates), code, exp)
	}
}

// runOneShot processes a batch of candidates from grid/random search.
func (w *ExperimentWorker) runOneShot(overridesList []map[string]interface{}, code string, exp *repository.StrategyExperiment) ([]candidateResult, error) {
	var results []candidateResult
	for _, overrides := range overridesList {
		r, err := w.backtestAndScore(context.Background(), code, overrides, exp)
		if err != nil {
			w.log.Warn("backtest failed", zap.Error(err))
			continue
		}
		results = append(results, r)
	}
	return results, nil
}

// runIterative drives an ask/tell optimizer (DE/TPE) with real backtest scoring.
func (w *ExperimentWorker) runIterative(
	ctx context.Context, opt ai.Optimizer, space ai.ResolvedSpace,
	code string, exp *repository.StrategyExperiment,
) ([]candidateResult, error) {
	var results []candidateResult
	for !opt.Done() {
		batch := opt.Ask(0)
		for _, indices := range batch {
			overrides := ai.IndexToOverrides(indices, space)
			r, err := w.backtestAndScore(ctx, code, overrides, exp)
			if err != nil {
				w.log.Warn("backtest failed", zap.Error(err))
				opt.Tell([]ai.OptimizerResult{{Indices: indices, Score: 0}})
				continue
			}
			opt.Tell([]ai.OptimizerResult{{Indices: indices, Score: r.Score}})
			results = append(results, r)
		}
	}
	return results, nil
}

// backtestAndScore executes a single backtest with parameter overrides applied.
func (w *ExperimentWorker) backtestAndScore(
	ctx context.Context, code string, overrides map[string]interface{},
	exp *repository.StrategyExperiment,
) (candidateResult, error) {
	modifiedCode := code
	run := &repository.BacktestRun{
		ID:                 uuid.New(),
		UserID:             exp.UserID,
		AccountID:          uuid.Nil,
		Symbol:             "XAUUSDm",
		Timeframe:          "1h",
		FromTs:             timePtr(time.Now().AddDate(0, -1, 0)),
		ToTs:               timePtr(time.Now()),
		Mode:               "KLINE_RANGE",
		Status:             "PENDING",
		StrategyCode:       &modifiedCode,
		InitialCapital:     f64Ptr(10000),
		StrategyCodeHash:   "",
		Error:              "",
		ExtraSymbols:       []string{},
		ParameterOverrides: marshalOverrides(overrides),
	}

	runID, err := w.backtestRepo.Create(ctx, run)
	if err != nil {
		return candidateResult{}, fmt.Errorf("create backtest: %w", err)
	}

	// Poll for completion
	for i := 0; i < 120; i++ { // 10 minutes timeout
		time.Sleep(5 * time.Second)
		bt, err := w.backtestRepo.GetByID(ctx, exp.UserID, runID)
		if err != nil {
			return candidateResult{}, fmt.Errorf("get backtest: %w", err)
		}
		if bt.Status == "SUCCEEDED" || bt.Status == "FAILED" {
			if bt.Status == "FAILED" {
				return candidateResult{}, fmt.Errorf("backtest failed: %s", bt.Error)
			}
			return w.scoreFromBacktest(bt, overrides), nil
		}
	}
	return candidateResult{}, fmt.Errorf("backtest %s timed out", runID)
}

// scoreFromBacktest extracts metrics from proto binary ExecuteBacktestResponse.
func (w *ExperimentWorker) scoreFromBacktest(bt *repository.BacktestRun, overrides map[string]interface{}) candidateResult {
	btMetrics := extractBacktestMetrics(bt.ProtoResponse)
	regime := ai.RegimeTransition
	scored := ai.Score(btMetrics, regime)

	summary := "param search"
	if scored.Trades < 5 {
		summary = fmt.Sprintf("only %d trades", scored.Trades)
	}

	return candidateResult{
		Overrides:       overrides,
		Score:           scored.Score,
		Grade:           scored.Grade,
		ScoreComponents: scored.Components,
		Summary:         summary,
	}
}

// extractBacktestMetrics parses proto binary ExecuteBacktestResponse → BacktestMetrics.
func extractBacktestMetrics(protoResp []byte) *ai.BacktestMetrics {
	if len(protoResp) == 0 {
		return &ai.BacktestMetrics{TotalTrades: 0}
	}
	var resp antv1.ExecuteBacktestResponse
	if err := proto.Unmarshal(protoResp, &resp); err != nil {
		return &ai.BacktestMetrics{TotalTrades: 0}
	}
	m := resp.GetMetrics(); eq := resp.GetEquityCurve()
	return &ai.BacktestMetrics{
		TotalReturn:  m.GetTotalReturn(),
		AnnualReturn: m.GetAnnualReturn(),
		SharpeRatio:  m.GetSharpeRatio(),
		MaxDrawdown:  m.GetMaxDrawdown(),
		WinRate:      m.GetWinRate(),
		ProfitFactor: m.GetProfitFactor(),
		TotalTrades:  int(m.GetTotalTrades()),
		Stability:    computeStability(eq),
	}
}

func marshalOverrides(overrides map[string]interface{}) []byte {
	b, _ := proto.Marshal(paramsToProto(overrides))
	return b
}

func timePtr(t time.Time) *time.Time { return &t }

// computeStability returns the R² of linear regression on the equity curve (0–1).
// A value near 1 means the equity curve is close to a straight line (stable growth).
func computeStability(equity []float64) float64 {
	if len(equity) < 2 {
		return 0
	}
	n := float64(len(equity))
	var sumX, sumY, sumXY, sumX2, sumY2 float64
	for i, y := range equity {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
		sumY2 += y * y
	}
	denom := (n*sumX2 - sumX*sumX) * (n*sumY2 - sumY*sumY)
	if denom <= 0 {
		return 0
	}
	r := (n*sumXY - sumX*sumY) / denom // Pearson's r
	// Clamp to [0,1]; negative r (declining equity) → 0 stability.
	if r < 0 {
		return 0
	}
	if r > 1 {
		return 1
	}
	return r
}

// paramsToProto converts a map of parameter overrides to StrategyParams proto.
func paramsToProto(overrides map[string]interface{}) *antv1.StrategyParams {
	p := &antv1.StrategyParams{Values: make(map[string]float64)}
	for k, v := range overrides {
		switch val := v.(type) {
		case float64:
			p.Values[k] = val
		case int:
			p.Values[k] = float64(val)
		case int64:
			p.Values[k] = float64(val)
		}
	}
	return p
}

// scoreComponentsToProto converts a score components map to ScoreComponents proto.
func scoreComponentsToProto(components map[string]float64) *antv1.ScoreComponents {
	return &antv1.ScoreComponents{Components: components}
}
