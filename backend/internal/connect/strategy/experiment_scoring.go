package strategy

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/ai"
	"anttrader/internal/repository"
)

func (w *ExperimentWorker) backtestAndScore(
	ctx context.Context, code string, overrides map[string]interface{},
	exp *repository.StrategyExperiment,
) (candidateResult, error) {
	modifiedCode := code
	symbol := exp.Symbol
	if symbol == "" {
		symbol = "XAUUSDm" // backward compat for experiments created before migration
	}
	tf := exp.Timeframe
	if tf == "" {
		tf = "1h"
	}
	var fromTs, toTs *time.Time
	if exp.FromTsUnixMs > 0 {
		ft := time.UnixMilli(exp.FromTsUnixMs)
		fromTs = &ft
	}
	if exp.ToTsUnixMs > 0 {
		tt := time.UnixMilli(exp.ToTsUnixMs)
		toTs = &tt
	}
	if fromTs == nil {
		ft := time.Now().AddDate(0, -1, 0)
		fromTs = &ft
	}
	if toTs == nil {
		tt := time.Now()
		toTs = &tt
	}
	overridesBytes, err := marshalOverrides(overrides)
	if err != nil {
		return candidateResult{}, fmt.Errorf("marshal overrides: %w", err)
	}
	run := &repository.BacktestRun{
		ID:                 uuid.New(),
		UserID:             exp.UserID,
		AccountID:          uuid.Nil,
		Symbol:             symbol,
		Timeframe:          tf,
		FromTs:             fromTs,
		ToTs:               toTs,
		Mode:               "KLINE_RANGE",
		Status:             StatusPending,
		StrategyCode:       &modifiedCode,
		InitialCapital:     f64Ptr(10000),
			Commission:        f64Ptr(0.001),
			Slippage:          f64Ptr(0),
			Leverage:          f64Ptr(1),
			TradeDirection:    strPtr("both"),
			StrictMode:        boolPtr(true),
		StrategyCodeHash:   "",
		Error:              "",
		ExtraSymbols:       []string{},
		ParameterOverrides: overridesBytes,
	}

	runID, err := w.backtestRepo.Create(ctx, run)
	if err != nil {
		return candidateResult{}, fmt.Errorf("create backtest: %w", err)
	}

	// Poll for completion
	for i := 0; i < 120; i++ { // 10 minutes max timeout
		select {
		case <-ctx.Done():
			return candidateResult{}, fmt.Errorf("backtest %s cancelled: %w", runID, ctx.Err())
		case <-time.After(5 * time.Second):
		}
		bt, err := w.backtestRepo.GetByID(ctx, exp.UserID, runID)
		if err != nil {
			return candidateResult{}, fmt.Errorf("get backtest: %w", err)
		}
		if bt.Status == StatusSucceeded || bt.Status == StatusFailed {
			if bt.Status == StatusFailed {
				return candidateResult{}, fmt.Errorf("backtest failed: %s", bt.Error)
			}
			return w.scoreFromBacktest(ctx, bt, overrides), nil
		}
	}
	return candidateResult{}, fmt.Errorf("backtest %s timed out", runID)
}

// scoreFromBacktest extracts metrics from proto binary and scores with regime-aware weights.
func (w *ExperimentWorker) scoreFromBacktest(ctx context.Context, bt *repository.BacktestRun, overrides map[string]interface{}) candidateResult {
	btMetrics := extractBacktestMetrics(bt.ProtoResponse)
	regime := w.detectRegime(ctx, bt)
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

