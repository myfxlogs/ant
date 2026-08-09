package strategy

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

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
			Summary:         fmt.Sprintf("%s score=%.1f grade=%s", c.Summary, c.Score, c.Grade),
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

func (w *ExperimentWorker) runOOSValidation(ctx context.Context, exp *repository.StrategyExperiment, code string, candidates []candidateResult, regime ai.MarketRegime) {
	const oosTopK = 5
	oosVal := ai.DefaultOOSValidator()
	symbol := exp.Symbol
	if symbol == "" {
		symbol = "XAUUSDm"
	}
	tf := exp.Timeframe
	if tf == "" {
		tf = "1h"
	}
	fromTs := time.UnixMilli(exp.FromTsUnixMs)
	toTs := time.UnixMilli(exp.ToTsUnixMs)
	if exp.FromTsUnixMs == 0 {
		fromTs = time.Now().AddDate(0, -1, 0)
	}
	if exp.ToTsUnixMs == 0 {
		toTs = time.Now()
	}
	windows := oosVal.ComputeWindows(fromTs, toTs)
	if windows == nil || len(candidates) == 0 {
		return
	}
	topIndices := selectTopK(candidates, oosTopK)
	for _, idx := range topIndices {
		c := &candidates[idx]
		oosScored, err := w.runSingleBacktest(
			ctx, code, c.Overrides, exp.UserID, symbol, tf,
			windows.OOSStart, windows.OOSEnd, regime,
		)
		if err != nil {
			w.log.Warn("OOS backtest failed",
				zap.Error(err),
				zap.Int("candidateIdx", idx),
				zap.Float64("isScore", c.Score))
			continue
		}
		validation := oosVal.Validate(c.Score, oosScored.Score)
		c.OOSScore = &oosScored.Score
		c.OOSTotalReturn = &oosScored.TotalReturn
		c.OOSSharpeRatio = &oosScored.SharpeRatio
		c.DegradationPct = &validation.Degradation
		c.IsOverfit = validation.IsOverfit
	}
}

type candidateResult struct {
	Overrides       map[string]interface{}
	Score           float64
	Grade           string
	ScoreComponents map[string]float64
	Summary         string
	BacktestRunID   *uuid.UUID
	// Raw backtest metrics (original values, not scored).
	TotalReturn  float64
	AnnualReturn float64
	SharpeRatio  float64
	MaxDrawdown  float64
	WinRate      float64
	ProfitFactor float64
	TotalTrades  int
	// OOS validation (nil when not in top-K or window too short)
	OOSScore       *float64
	OOSTotalReturn *float64
	OOSSharpeRatio *float64
	DegradationPct *float64
	IsOverfit      bool
}

func (w *ExperimentWorker) runOptimizer(
	ctx context.Context, exp *repository.StrategyExperiment,
	params []ai.TunableParam, space ai.ResolvedSpace, code string,
	regime ai.MarketRegime,
) ([]candidateResult, error) {
	switch exp.SearchMethod {
	case "de":
		return w.runIterative(ctx, ai.NewDEOptimizer(space, exp.MaxCandidates), space, code, exp, regime)
	case "ai":
		return w.runAIProposal(ctx, params, code, exp, regime)
	case "tpe":
		return w.runIterative(ctx, ai.NewTPEOptimizer(space, exp.MaxCandidates), space, code, exp, regime)
	case "ags":
		return w.runIterative(ctx, ai.NewAnnealedGaussianOptimizer(space, exp.MaxCandidates), space, code, exp, regime)
	case "random":
		return w.runOneShot(ctx, ai.RandomSearchSpace(space, exp.MaxCandidates), code, exp, regime)
	default:
		return w.runOneShot(ctx, ai.GridSearchSpace(space, exp.MaxCandidates), code, exp, regime)
	}
}

// runOneShot processes a batch of candidates from grid/random search.
func (w *ExperimentWorker) runOneShot(ctx context.Context, overridesList []map[string]interface{}, code string, exp *repository.StrategyExperiment, regime ai.MarketRegime) ([]candidateResult, error) {
	var results []candidateResult
	for _, overrides := range overridesList {
		r, err := w.backtestAndScore(ctx, code, overrides, exp, regime)
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
	code string, exp *repository.StrategyExperiment, regime ai.MarketRegime,
) ([]candidateResult, error) {
	var results []candidateResult
	for !opt.Done() {
		batch := opt.Ask(0)
		for _, indices := range batch {
			overrides := ai.IndexToOverrides(indices, space)
			r, err := w.backtestAndScore(ctx, code, overrides, exp, regime)
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

// resolvedSpaceFromParamSpace builds a ResolvedSpace from a proto structpb.Struct
// stored in exp.ParameterSpace. The struct format is { "paramName": [v1, v2, ...] }
// as submitted by the frontend tuning UI.
func resolvedSpaceFromParamSpace(raw []byte) ai.ResolvedSpace {
	if len(raw) == 0 {
		return ai.ResolvedSpace{}
	}
	var ps structpb.Struct
	if err := proto.Unmarshal(raw, &ps); err != nil {
		return ai.ResolvedSpace{}
	}
	keys := make([]string, 0, len(ps.Fields))
	vals := make(map[string][]float64, len(ps.Fields))
	for k, v := range ps.Fields {
		listVal := v.GetListValue()
		if listVal == nil || len(listVal.Values) == 0 {
			continue
		}
		var floatVals []float64
		for _, item := range listVal.Values {
			f := item.GetNumberValue()
			floatVals = append(floatVals, f)
		}
		if len(floatVals) > 0 {
			keys = append(keys, k)
			vals[k] = floatVals
		}
	}
	sort.Strings(keys) // deterministic order
	return ai.ResolvedSpace{Keys: keys, ValuesByKey: vals}
}

// paramsFromSpace creates pseudo TunableParam entries from a ResolvedSpace
// so that GridSearch/RandomSearch can operate. Each key becomes a "choice"
// param with Min/Max/Step derived from the value array.
func paramsFromSpace(space ai.ResolvedSpace) []ai.TunableParam {
	out := make([]ai.TunableParam, 0, len(space.Keys))
	for _, key := range space.Keys {
		vals := space.ValuesByKey[key]
		if len(vals) == 0 {
			continue
		}
		minVal, maxVal := vals[0], vals[0]
		for _, v := range vals {
			if v < minVal {
				minVal = v
			}
			if v > maxVal {
				maxVal = v
			}
		}
		step := 1.0
		if len(vals) > 1 {
			step = (maxVal - minVal) / float64(len(vals)-1)
		}
		out = append(out, ai.TunableParam{
			Name:    key,
			Type:    "float",
			Default: vals[0],
			Min:     minVal,
			Max:     maxVal,
			Step:    step,
		})
	}
	return out
}
