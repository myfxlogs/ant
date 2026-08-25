// strategy_experiment_worker_validation.go — OOS validation and optimizer helpers extracted from strategy_experiment_worker.go.
package strategy

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"alphaforge/internal/ai"
	"alphaforge/internal/repository"
)

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
	backtestRunIDStr := ""
	if exp.BacktestRunID != nil {
		backtestRunIDStr = exp.BacktestRunID.String()
	}
	topIndices := selectTopK(candidates, oosTopK)
	for _, idx := range topIndices {
		c := &candidates[idx]
		oosScored, err := w.runSingleBacktest(
			ctx, code, c.Overrides, exp.UserID, symbol, tf,
			windows.OOSStart, windows.OOSEnd, regime, backtestRunIDStr,
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
