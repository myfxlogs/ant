package strategy

import (
	"context"
	"fmt"
	"sort"
	"math"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/ai"
	"anttrader/internal/repository"
	systemai "anttrader/internal/service/systemai"
)

// ExperimentWorker polls for PENDING experiments and processes them.
type ExperimentWorker struct {
	repo         *repository.StrategyExperimentRepository
	backtestRepo *repository.BacktestRunRepository
	log          *zap.Logger
	systemAISvc  *systemai.Service // optional: enables AI multi-round proposal
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

// SetAIService injects the system AI service for AI multi-round proposal (search_method="ai").
func (w *ExperimentWorker) SetAIService(svc *systemai.Service) { w.systemAISvc = svc }

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
	case "ai":
		return w.runAIProposal(ctx, params, code, exp)
	case "tpe":
		return w.runIterative(ctx, ai.NewTPEOptimizer(space, exp.MaxCandidates), space, code, exp)
	case "ags":
		return w.runIterative(ctx, ai.NewAnnealedGaussianOptimizer(space, exp.MaxCandidates), space, code, exp)
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

// scoreFromBacktest extracts metrics from proto binary and scores with regime-aware weights.
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
// computeStability returns Spearman rank correlation (0-1) of the equity curve.
// Spearman is optimal for equity monotonicity: it detects any consistent upward trend
// regardless of shape, and is robust to outliers.
func computeStability(equity []float64) float64 {
	if len(equity) < 2 {
		return 0
	}
	n := len(equity)
	// Compute ranks of equity values
	ranks := make([]float64, n)
	type pair struct{ val float64; idx int }
	pairs := make([]pair, n)
	for i, v := range equity {
		pairs[i] = pair{v, i}
	}
	// Sort by value to assign ranks
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].val < pairs[j].val })
	for r, p := range pairs {
		ranks[p.idx] = float64(r + 1)
	}
	// Spearman: correlation of ranks vs time indices (1..n)
	var sumX, sumY, sumXY, sumX2, sumY2 float64
	for i, r := range ranks {
		x := float64(i + 1)
		sumX += x
		sumY += r
		sumXY += x * r
		sumX2 += x * x
		sumY2 += r * r
	}
	nf := float64(n)
	num := nf*sumXY - sumX*sumY
	den := (nf*sumX2 - sumX*sumX) * (nf*sumY2 - sumY*sumY)
	if den <= 0 {
		return 0
	}
	r := num / math.Sqrt(den)
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
func (w *ExperimentWorker) runAIProposal(ctx context.Context, params []ai.TunableParam, code string, exp *repository.StrategyExperiment) ([]candidateResult, error) {
	if w.systemAISvc == nil {
		return nil, fmt.Errorf("AI proposer not configured")
	}
	proposer := &systemAIAdapter{svc: w.systemAISvc, userID: exp.UserID}
	var results []candidateResult
	maxRounds := 3
	for round := 1; round <= maxRounds; round++ {
		req := &ai.ProposeRequest{
			IndicatorCode: code,
			Params:        params,
			Round:         round,
			MaxCandidates: exp.MaxCandidates / maxRounds,
			PrevResults:   make([]ai.ProposePrevResult, len(results)),
		}
		for i, r := range results {
			req.PrevResults[i] = ai.ProposePrevResult{
				Params: r.Overrides, Score: r.Score, Grade: r.Grade,
			}
		}
		proposed, err := ai.ProposeParams(ctx, proposer, req)
		if err != nil {
			w.log.Warn("AI proposal failed", zap.Error(err), zap.Int("round", round))
			continue
		}
		for _, overrides := range proposed {
			r, err := w.backtestAndScore(ctx, code, overrides, exp)
			if err != nil {
				w.log.Warn("AI backtest failed", zap.Error(err))
				continue
			}
			results = append(results, r)
		}
	}
	return results, nil
}
