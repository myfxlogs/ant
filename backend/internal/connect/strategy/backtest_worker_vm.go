package strategy

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/repository"
	"anttrader/strategy/backtest"
	"anttrader/strategy/sdk"
	"anttrader/tools/mql2go"
)

// executeVMBacktest runs a backtest via the in-process Bytecode VM:
// MQL source → CompileMQL → VMRunner → backtest.Engine → ExecuteBacktestResponse.
// Uses bytecode cache from imported_strategies when strategy_id is available.
func (s *StrategyExecutionServer) executeVMBacktest(ctx context.Context, params backtestParams, klines []*antv1.ExecuteKlineBar, run *repository.BacktestRun) (*antv1.ExecuteBacktestResponse, error) {
	s.log.Info("executeVMBacktest: starting",
		zap.Int("klines", len(klines)),
		zap.String("symbol", run.Symbol),
		zap.String("timeframe", run.Timeframe))

	var cachedBytecode []byte
	if run.StrategyID != nil && s.importedRepo != nil {
		cachedBytecode, _ = s.importedRepo.GetBytecode(ctx, *run.StrategyID)
	}

	vmRunner, bcData, err := mql2go.CompileMQLCached(params.code, cachedBytecode)
	if err != nil {
		s.log.Error("executeVMBacktest: compile failed", zap.Error(err))
		return nil, fmt.Errorf("compile MQL: %w", err)
	}
	s.log.Info("executeVMBacktest: compiled successfully")

	// Persist newly compiled bytecode for future runs.
	if bcData != nil && run.StrategyID != nil && s.importedRepo != nil {
		if saveErr := s.importedRepo.SaveBytecode(ctx, *run.StrategyID, bcData); saveErr != nil {
			s.log.Warn("executeVMBacktest: save bytecode cache failed", zap.Error(saveErr))
		}
	}

	bars := make([]sdk.Bar, len(klines))
	for i, k := range klines {
		bars[i] = sdk.Bar{
			Open:      parseDecimal(k.Open),
			High:      parseDecimal(k.High),
			Low:       parseDecimal(k.Low),
			Close:     parseDecimal(k.Close),
			Volume:    parseInt64(k.Volume),
			Timestamp: k.OpenTimeMs,
		}
	}

	cfg := backtest.Config{
		Symbol:         run.Symbol,
		Timeframe:      run.Timeframe,
		InitialCapital: parseDecimal(params.initialCapital),
		Leverage:       parseInt32(params.leverage),
		Commission:     parseDecimal(params.commission),
		Slippage:       parseDecimal(params.slippage),
		SwapRate:       decimal.NewFromFloat(0.00001),
		StrictMode:     params.strictMode,
		Params:         paramsProtoToMap(run.ParameterOverrides),
	}
	if run.FromTs != nil {
		cfg.StartDate = *run.FromTs
	}
	if run.ToTs != nil {
		cfg.EndDate = *run.ToTs
	}

	symbolInfo := s.fetchSymbolInfo(ctx, run)
	if symbolInfo != nil {
		cfg.SymbolDigits = symbolInfo.Digits
		if p, err := decimal.NewFromString(symbolInfo.Point); err == nil {
			cfg.SymbolPoint = p
		}
		if v, err := decimal.NewFromString(symbolInfo.VolumeMin); err == nil {
			cfg.VolumeMin = v
		}
		if v, err := decimal.NewFromString(symbolInfo.VolumeMax); err == nil {
			cfg.VolumeMax = v
		}
		if v, err := decimal.NewFromString(symbolInfo.VolumeStep); err == nil {
			cfg.VolumeStep = v
		}
		if v, err := decimal.NewFromString(symbolInfo.ContractSize); err == nil {
			cfg.ContractSize = v
		}
	} else if len(bars) > 0 {
		closeStr := bars[0].Close.String()
		dotIdx := -1
		for i, c := range closeStr {
			if c == '.' {
				dotIdx = i
				break
			}
		}
		digits := int32(0)
		if dotIdx >= 0 {
			digits = int32(len(closeStr) - dotIdx - 1)
		}
		if digits > 8 {
			digits = 8
		}
		cfg.SymbolDigits = digits
		point := decimal.NewFromFloat(1)
		for i := int32(0); i < digits; i++ {
			point = point.Div(decimal.NewFromInt(10))
		}
		cfg.SymbolPoint = point
		cfg.Spread = point.Mul(decimal.NewFromInt(10))
		s.log.Info("executeVMBacktest: derived symbol info from K-lines",
			zap.Int32("digits", digits),
			zap.String("point", point.String()),
			zap.String("spread", cfg.Spread.String()))
	}

	engine := backtest.New(cfg, vmRunner, bars)
	result, err := engine.Run(ctx)
	if err != nil {
		s.log.Error("executeVMBacktest: engine.Run failed", zap.Error(err), zap.Int("bars", len(bars)))
		return &antv1.ExecuteBacktestResponse{Success: false, Error: err.Error()}, nil
	}
	s.log.Info("executeVMBacktest: engine.Run completed",
		zap.Int("trades", len(result.Trades)),
		zap.Int("equity_points", len(result.Equity)),
		zap.Int("bars_processed", len(bars)))

	resp := &antv1.ExecuteBacktestResponse{
		Success: true,
		Metrics: &antv1.ExecuteBacktestMetrics{
			TotalReturn:   result.Metrics.TotalReturn,
			AnnualReturn:  result.Metrics.AnnualReturn,
			MaxDrawdown:   result.Metrics.MaxDrawdown,
			SharpeRatio:   result.Metrics.SharpeRatio,
			WinRate:       result.Metrics.WinRate,
			ProfitFactor:  result.Metrics.ProfitFactor,
			TotalTrades:   result.Metrics.TotalTrades,
			WinningTrades: result.Metrics.WinningTrades,
			LosingTrades:  result.Metrics.LosingTrades,
		},
	}
	if result.Metrics != nil {
		totalPnl := cfg.InitialCapital.Mul(decimal.NewFromFloat(result.Metrics.TotalReturn))
		resp.Metrics.TotalPnlAbsolute = totalPnl.String()
	}
	for _, ep := range result.Equity {
		resp.EquityCurve = append(resp.EquityCurve, ep.Equity.String())
		resp.EquityTimesMs = append(resp.EquityTimesMs, ep.Time.UnixMilli())
	}
	for i, t := range result.Trades {
		side := "BUY"
		if t.Side == sdk.SideSell {
			side = "SELL"
		}
		resp.Trades = append(resp.Trades, &antv1.ExecuteBacktestTrade{
			Ticket:     int64(i + 1),
			Side:       side,
			Volume:     t.Volume.String(),
			OpenTsMs:   t.EntryTime.UnixMilli(),
			OpenPrice:  t.EntryPrice.String(),
			CloseTsMs:  t.ExitTime.UnixMilli(),
			ClosePrice: t.ExitPrice.String(),
			Pnl:        t.Profit.String(),
			Commission: t.Commission.String(),
			Reason:     t.Comment,
		})
	}

	resp.ExecutionAssumptions = &antv1.ExecutionAssumptions{
		SimulationMode:  "KLINE_RANGE",
		SignalTiming:    "next_bar_open",
		FillRule:        "bar_close",
		ActualCommission: cfg.Commission.String(),
		ActualSlippage:  cfg.Slippage.String(),
		ActualLeverage:  fmt.Sprintf("%d", cfg.Leverage),
		TradeDirection:  "both",
	}
	if cfg.StrictMode {
		resp.ExecutionAssumptions.SignalTiming = "next_bar_open"
	} else {
		resp.ExecutionAssumptions.SignalTiming = "same_bar_close"
		resp.ExecutionAssumptions.MtfFallbackReason = "strict_mode disabled"
	}

	resp.Risk = assessRisk(result.Metrics)

	return resp, nil
}
