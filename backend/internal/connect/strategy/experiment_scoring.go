package strategy

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/ai"
	"anttrader/internal/repository"
)

// runSingleBacktest executes one backtest with the given time window,
// polls for completion, scores with regime-aware weights, and returns the scored result.
// Used for both in-sample (full window) and out-of-sample (OOS window) backtests.
func (w *ExperimentWorker) runSingleBacktest(
	ctx context.Context, code string, overrides map[string]interface{},
	userID uuid.UUID, symbol, timeframe string, fromTs, toTs time.Time,
	regime ai.MarketRegime,
) (*ai.ScoredResult, error) {
	modifiedCode := code
	overridesBytes, err := marshalOverrides(overrides)
	if err != nil {
		return nil, fmt.Errorf("marshal overrides: %w", err)
	}
	run := &repository.BacktestRun{
		ID:                 uuid.New(),
		UserID:             userID,
		AccountID:          uuid.Nil,
		Symbol:             symbol,
		Timeframe:          timeframe,
		FromTs:             &fromTs,
		ToTs:               &toTs,
		Mode:               "KLINE_RANGE",
		Status:             StatusPending,
		StrategyCode:       &modifiedCode,
		InitialCapital:     f64Ptr(10000),
		Commission:         f64Ptr(0.001),
		Slippage:           f64Ptr(0),
		Leverage:           f64Ptr(1),
		TradeDirection:     strPtr("both"),
		StrictMode:         boolPtr(true),
		StrategyCodeHash:   "",
		Error:              "",
		ExtraSymbols:       []string{},
		ParameterOverrides: overridesBytes,
	}

	runID, err := w.backtestRepo.Create(ctx, run)
	if err != nil {
		return nil, fmt.Errorf("create backtest: %w", err)
	}

	// Poll for completion
	for i := 0; i < 120; i++ { // 10 minutes max timeout
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("backtest %s cancelled: %w", runID, ctx.Err())
		case <-time.After(5 * time.Second):
		}
		bt, err := w.backtestRepo.GetByID(ctx, userID, runID)
		if err != nil {
			return nil, fmt.Errorf("get backtest: %w", err)
		}
		if bt.Status == StatusSucceeded || bt.Status == StatusFailed {
			if bt.Status == StatusFailed {
				return nil, fmt.Errorf("backtest failed: %s", bt.Error)
			}
			btMetrics := extractBacktestMetrics(bt.ProtoResponse)
			scored := ai.Score(btMetrics, regime)
			return scored, nil
		}
	}
	return nil, fmt.Errorf("backtest %s timed out", runID)
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
	}, nil
}

// scoreFromBacktest extracts metrics from proto binary and scores with the given regime.
func (w *ExperimentWorker) scoreFromBacktest(bt *repository.BacktestRun, overrides map[string]interface{}, regime ai.MarketRegime) candidateResult {
	btMetrics := extractBacktestMetrics(bt.ProtoResponse)
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

// detectRegime fetches K-lines for the backtest run and detects market regime.
// Falls back to Transition if data is insufficient or unavailable.
func (w *ExperimentWorker) detectRegime(ctx context.Context, bt *repository.BacktestRun) ai.MarketRegime {
	if w.marketDataRepo == nil || bt.Symbol == "" || bt.Timeframe == "" {
		return ai.RegimeTransition
	}
	bars, err := w.marketDataRepo.GetKlines(
		ctx, bt.Symbol, "", bt.Timeframe, bt.FromTs, bt.ToTs, 2000,
	)
	if err != nil || len(bars) < 30 {
		return ai.RegimeTransition
	}
	ohlc := make([]ai.OHLCBar, len(bars))
	for i := 0; i < len(bars); i++ {
		b := bars[len(bars)-1-i] // reverse DESC→ASC
		ohlc[i] = ai.OHLCBar{Open: b.Open, High: b.High, Low: b.Low, Close: b.Close, Volume: b.Volume}
	}
	result := ai.DetectRegime(ohlc)
	return result.Regime
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
		ohlc[i] = ai.OHLCBar{Open: b.Open, High: b.High, Low: b.Low, Close: b.Close, Volume: b.Volume}
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

