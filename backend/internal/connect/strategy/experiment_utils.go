package strategy

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/ai"
	"anttrader/internal/repository"
)

func marshalOverrides(overrides map[string]interface{}) ([]byte, error) {
	b, err := proto.Marshal(paramsToProto(overrides))
	if err != nil {
		return nil, fmt.Errorf("proto.Marshal overrides: %w", err)
	}
	return b, nil
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
func (w *ExperimentWorker) runAIProposal(ctx context.Context, params []ai.TunableParam, code string, exp *repository.StrategyExperiment, regime ai.MarketRegime) ([]candidateResult, error) {
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
			r, err := w.backtestAndScore(ctx, code, overrides, exp, regime)
			if err != nil {
				w.log.Warn("AI backtest failed", zap.Error(err))
				continue
			}
			results = append(results, r)
		}
	}
	return results, nil
}
