package strategy

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/ai"
	"alphaforge/internal/repository"
)

// runSingleBacktest executes one backtest in-process via BacktestExecutor,
// scores with regime-aware weights, and returns the scored result.
// No backtest_runs record is created — used for experiment candidate scoring.
func (w *ExperimentWorker) runSingleBacktest(
	ctx context.Context, code string, overrides map[string]interface{},
	userID uuid.UUID, symbol, timeframe string, fromTs, toTs time.Time,
	regime ai.MarketRegime,
) (*ai.ScoredResult, error) {
	if w.executor == nil {
		return nil, fmt.Errorf("backtest executor not configured")
	}
	overridesBytes, err := marshalOverrides(overrides)
	if err != nil {
		return nil, fmt.Errorf("marshal overrides: %w", err)
	}
	resp, err := w.executor.ExecuteBacktestDirect(ctx, code, overridesBytes, symbol, timeframe, fromTs, toTs)
	if err != nil {
		return nil, fmt.Errorf("execute backtest direct: %w", err)
	}
	if !resp.GetSuccess() {
		return nil, fmt.Errorf("backtest failed: %s", resp.GetError())
	}
	btMetrics := extractBacktestMetricsFromResponse(resp)
	scored := ai.Score(btMetrics, regime)
	return scored, nil
}

// backtestAndScore executes an in-sample backtest on the full experiment time window.
func (w *ExperimentWorker) backtestAndScore(
	ctx context.Context, code string, overrides map[string]interface{},
	exp *repository.StrategyExperiment, regime ai.MarketRegime,
) (candidateResult, error) {
	symbol := exp.Symbol
	if symbol == "" {
		symbol = "XAUUSDm" // backward compat for experiments created before migration
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

	scored, err := w.runSingleBacktest(ctx, code, overrides, exp.UserID, symbol, tf, fromTs, toTs, regime)
	if err != nil {
		return candidateResult{}, err
	}

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
		TotalReturn:     scored.TotalReturn,
		AnnualReturn:    scored.AnnualReturn,
		SharpeRatio:     scored.SharpeRatio,
		MaxDrawdown:     scored.MaxDrawdown,
		WinRate:         scored.WinRate,
		ProfitFactor:    scored.ProfitFactor,
		TotalTrades:     scored.Trades,
	}, nil
}

// extractBacktestMetricsFromResponse parses ExecuteBacktestResponse → BacktestMetrics.
func extractBacktestMetricsFromResponse(resp *antv1.ExecuteBacktestResponse) *ai.BacktestMetrics {
	m := resp.GetMetrics()
	eq := resp.GetEquityCurve()
	return &ai.BacktestMetrics{
		TotalReturn:  parseFloat(m.GetTotalReturn()),
		AnnualReturn: parseFloat(m.GetAnnualReturn()),
		SharpeRatio:  parseFloat(m.GetSharpeRatio()),
		MaxDrawdown:  parseFloat(m.GetMaxDrawdown()),
		WinRate:      parseFloat(m.GetWinRate()),
		ProfitFactor: parseFloat(m.GetProfitFactor()),
		TotalTrades:  int(m.GetTotalTrades()),
		Stability:    computeStability(equityCurveToFloat64(eq)),
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
	m := resp.GetMetrics()
	eq := resp.GetEquityCurve()
	return &ai.BacktestMetrics{
		TotalReturn:  parseFloat(m.GetTotalReturn()),
		AnnualReturn: parseFloat(m.GetAnnualReturn()),
		SharpeRatio:  parseFloat(m.GetSharpeRatio()),
		MaxDrawdown:  parseFloat(m.GetMaxDrawdown()),
		WinRate:      parseFloat(m.GetWinRate()),
		ProfitFactor: parseFloat(m.GetProfitFactor()),
		TotalTrades:  int(m.GetTotalTrades()),
		Stability:    computeStability(equityCurveToFloat64(eq)),
	}
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// detectRegimeForExperiment fetches K-lines for the experiment's symbol/timeframe/time-window
// and classifies the market regime. Called once per experiment, not per candidate.
func (w *ExperimentWorker) detectRegimeForExperiment(
	ctx context.Context, exp *repository.StrategyExperiment,
) ai.MarketRegime {
	if w.marketDataRepo == nil || exp.Symbol == "" || exp.Timeframe == "" {
		return ai.RegimeTransition
	}
	fromTs := time.UnixMilli(exp.FromTsUnixMs)
	toTs := time.UnixMilli(exp.ToTsUnixMs)
	if exp.FromTsUnixMs == 0 {
		ft := time.Now().AddDate(0, -1, 0)
		fromTs = ft
	}
	if exp.ToTsUnixMs == 0 {
		toTs = time.Now()
	}
	bars, err := w.marketDataRepo.GetKlines(
		ctx, exp.Symbol, "", exp.Timeframe, &fromTs, &toTs, 2000,
	)
	if err != nil || len(bars) < 30 {
		return ai.RegimeTransition
	}
	ohlc := make([]ai.OHLCBar, len(bars))
	for i := 0; i < len(bars); i++ {
		b := bars[len(bars)-1-i] // reverse DESC→ASC
		ohlc[i] = ai.OHLCBar{Open: b.Open.InexactFloat64(), High: b.High.InexactFloat64(), Low: b.Low.InexactFloat64(), Close: b.Close.InexactFloat64(), Volume: b.Volume}
	}
	result := ai.DetectRegime(ohlc)
	return result.Regime
}

// selectTopK returns the indices of the top-K candidates sorted by score descending.
func selectTopK(candidates []candidateResult, k int) []int {
	if k > len(candidates) {
		k = len(candidates)
	}
	if k <= 0 {
		return nil
	}
	type indexed struct {
		idx   int
		score float64
	}
	items := make([]indexed, len(candidates))
	for i, c := range candidates {
		items[i] = indexed{idx: i, score: c.Score}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].score > items[j].score })
	indices := make([]int, k)
	for i := 0; i < k; i++ {
		indices[i] = items[i].idx
	}
	return indices
}
