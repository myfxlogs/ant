package strategy

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/repository"
	"alphaforge/strategy/backtest"
	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go"
	"alphaforge/tools/mql2go/interp"
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

	// Bytecode cache omits CoverageReport; recompile from source to recover
	// coverage/blind-spot data when cache hit produced nil coverage.
	if vmRunner.GetCoverage() == nil && params.code != "" {
		if covRunner, _, covErr := mql2go.CompileMQLWithCoverage(params.code); covErr == nil {
			vmRunner.InjectCoverage(covRunner.GetCoverage())
			_ = covRunner // discard; only coverage is needed
		}
	}

	// Persist newly compiled bytecode for future runs.
	if bcData != nil && run.StrategyID != nil && s.importedRepo != nil {
		if saveErr := s.importedRepo.SaveBytecode(ctx, *run.StrategyID, bcData); saveErr != nil {
			s.log.Warn("executeVMBacktest: save bytecode cache failed", zap.Error(saveErr))
		}
	}

	return s.runVMEngine(ctx, vmRunner, params, klines, run)
}

// runVMEngine executes the backtest engine with a pre-compiled VMRunner.
// Shared by both MQL (executeVMBacktest) and Python (executePythonVMBacktest) paths.
func (s *StrategyExecutionServer) runVMEngine(ctx context.Context, vmRunner *mql2go.VMRunner, params backtestParams, klines []*antv1.ExecuteKlineBar, run *repository.BacktestRun) (*antv1.ExecuteBacktestResponse, error) {

	bars := klinesToBars(klines)
	cfg := s.buildBacktestConfig(params, run)

	if len(run.ExtraSymbols) > 0 && s.marketDataRepo != nil {
		s.loadExtraSymbolBars(ctx, run, &cfg)
	}

	s.applySymbolInfo(ctx, run, &cfg, bars)

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

	resp, ruleFindings, covBlindSpots, runtimeBlinds := buildBacktestResponse(result, cfg, params, vmRunner)

	// Persist failure signature if there are diagnostic findings (B2-6).
	if s.failureSigRepo != nil && len(ruleFindings) > 0 {
		totalTrades := 0
		if result.Metrics != nil {
			totalTrades = int(result.Metrics.TotalTrades)
		}
		reproPkg := mql2go.BuildReproPackage(
			params.code, ruleFindings, covBlindSpots, runtimeBlinds,
			totalTrades, run.Symbol, run.Timeframe,
		)
		protoPkg := reproPkg.ToProto()
		var runID *uuid.UUID
		if run.ID != uuid.Nil {
			runID = &run.ID
		}
		if _, err := s.failureSigRepo.SaveFailureSignature(ctx, protoPkg, runID); err != nil {
			s.log.Warn("executeVMBacktest: save failure signature failed", zap.Error(err))
		}
	}

	return resp, nil
}

func klinesToBars(klines []*antv1.ExecuteKlineBar) []sdk.Bar {
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
	return bars
}

func (s *StrategyExecutionServer) buildBacktestConfig(params backtestParams, run *repository.BacktestRun) backtest.Config {
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
	if params.swapRate != "" {
		if sr, err := decimal.NewFromString(params.swapRate); err == nil {
			cfg.SwapRate = sr
		}
	}
	if params.marginCallLevel != "" {
		if mc, err := decimal.NewFromString(params.marginCallLevel); err == nil {
			cfg.MarginCallLevel = mc
		}
	}
	if run.FromTs != nil {
		cfg.StartDate = *run.FromTs
	}
	if run.ToTs != nil {
		cfg.EndDate = *run.ToTs
	}
	return cfg
}

func (s *StrategyExecutionServer) loadExtraSymbolBars(ctx context.Context, run *repository.BacktestRun, cfg *backtest.Config) {
	cfg.ExtraSymbolBars = make(map[string][]sdk.Bar, len(run.ExtraSymbols))
	for _, sym := range run.ExtraSymbols {
		if sym == "" || sym == run.Symbol {
			continue
		}
		extraKlines, err := s.fetchExtraSymbolKlines(ctx, sym, run)
		if err != nil {
			s.log.Warn("fetch extra symbol klines failed",
				zap.String("symbol", sym), zap.Error(err))
			continue
		}
		if len(extraKlines) == 0 {
			continue
		}
		cfg.ExtraSymbolBars[sym] = klinesToBars(extraKlines)
		s.log.Info("loaded extra symbol bars",
			zap.String("symbol", sym), zap.Int("bars", len(cfg.ExtraSymbolBars[sym])))
	}
}

func (s *StrategyExecutionServer) applySymbolInfo(ctx context.Context, run *repository.BacktestRun, cfg *backtest.Config, bars []sdk.Bar) {
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
		backtest.DeriveSymbolInfoFromBars(cfg, bars)
		s.log.Info("executeVMBacktest: derived symbol info from K-lines",
			zap.Int32("digits", cfg.SymbolDigits),
			zap.String("point", cfg.SymbolPoint.String()),
			zap.String("spread", cfg.Spread.String()))
	}
}

func buildBacktestResponse(result *backtest.Result, cfg backtest.Config, params backtestParams, vmRunner *mql2go.VMRunner) (*antv1.ExecuteBacktestResponse, []mql2go.DiagnosticFinding, []mql2go.CoverageBlindSpot, []mql2go.RuntimeBlindSpot) {
	totalTrades := 0
	if result.Metrics != nil {
		totalTrades = int(result.Metrics.TotalTrades)
	}

	// Run diagnostic rule engine
	var ruleFindings []mql2go.DiagnosticFinding
	cov := vmRunner.GetCoverage()
	var covBlindSpots []mql2go.CoverageBlindSpot
	// CoverageReport.BlindSpots is []string; convert to CoverageBlindSpot for the rule engine
	if cov != nil {
		for _, bs := range cov.BlindSpots {
			covBlindSpots = append(covBlindSpots, mql2go.CoverageBlindSpot{
				Builtin:  bs,
				Severity: interp.SeverityForBuiltin(bs),
				Source:   "compile",
			})
		}
	}
	var runtimeBlinds []mql2go.RuntimeBlindSpot
	for _, rbs := range vmRunner.GetRuntimeBlindSpots() {
		runtimeBlinds = append(runtimeBlinds, mql2go.RuntimeBlindSpot{
			Builtin:  rbs.Builtin,
			Severity: rbs.Severity,
			Count:    rbs.Count,
		})
	}
	if params.code != "" {
		engine := mql2go.NewRuleEngine()
		ruleFindings = engine.Run(mql2go.RuleInput{
			Source:        params.code,
			HasOnTick:     vmRunner.HasOnTick(),
			Coverage:      cov,
			BlindSpots:    covBlindSpots,
			TotalTrades:   totalTrades,
			RuntimeBlinds: runtimeBlinds,
		})
	}

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
		totalReturn, err := decimal.NewFromString(result.Metrics.TotalReturn)
		if err != nil {
			totalReturn = decimal.Zero
		}
		totalPnl := cfg.InitialCapital.Mul(totalReturn)
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
		SimulationMode:   "KLINE_RANGE",
		SignalTiming:     "next_bar_open",
		FillRule:         "bar_close",
		ActualCommission: cfg.Commission.String(),
		ActualSlippage:   cfg.Slippage.String(),
		ActualLeverage:   fmt.Sprintf("%d", cfg.Leverage),
		TradeDirection:   tradeDirectionToString(params.tradeDir),
	}
	if cfg.StrictMode {
		resp.ExecutionAssumptions.SignalTiming = "next_bar_open"
	} else {
		resp.ExecutionAssumptions.SignalTiming = "same_bar_close"
		resp.ExecutionAssumptions.MtfFallbackReason = "strict_mode disabled"
	}

	resp.Risk = assessRisk(result.Metrics)
	resp.BlindSpots = attachBlindSpots(cov, vmRunner, ruleFindings)

	// P0 invariant: every trade must have Volume > 0.
	// If violated, the backtest result is unreliable (ADR-0028 §4.2 防线 B).
	if bs := checkVolumeInvariant(result.Trades); bs != nil {
		resp.Risk.IsReliable = false
		resp.BlindSpots = append(resp.BlindSpots, bs)
	}

	// P0 invariant: capital conservation — 期末净值 must equal 本金 + ΣProfit − ΣCommission − ΣSwap.
	// If violated, the backtest result is unreliable (ADR-0028 §4.2 防线 B).
	if bs := checkCapitalConservation(result); bs != nil {
		resp.Risk.IsReliable = false
		resp.BlindSpots = append(resp.BlindSpots, bs)
	}

	// P0 invariant: trade field integrity — prices, side, time order.
	// If violated, the backtest result is unreliable (ADR-0028 §4.2 防线 B).
	if bs := checkPricePositive(result); bs != nil {
		resp.Risk.IsReliable = false
		resp.BlindSpots = append(resp.BlindSpots, bs)
	}
	if bs := checkSideValid(result); bs != nil {
		resp.Risk.IsReliable = false
		resp.BlindSpots = append(resp.BlindSpots, bs)
	}
	if bs := checkTimeOrder(result); bs != nil {
		resp.Risk.IsReliable = false
		resp.BlindSpots = append(resp.BlindSpots, bs)
	}

	return resp, ruleFindings, covBlindSpots, runtimeBlinds
}

// checkPricePositive verifies that every trade has EntryPrice > 0 and ExitPrice > 0.
// Returns a BlindSpot if any price is <= 0, nil otherwise.
// When there are no trades, the invariant is vacuously true (returns nil).
func checkPricePositive(result *backtest.Result) *antv1.BlindSpot {
	for _, t := range result.Trades {
		if !t.EntryPrice.GreaterThan(decimal.Zero) || !t.ExitPrice.GreaterThan(decimal.Zero) {
			return &antv1.BlindSpot{
				Id:          "non_positive_price",
				Category:    "invariant",
				Severity:    interp.SeverityFatal,
				Description: "存在开仓价或平仓价<=0的交易，回测结果不可信",
			}
		}
	}
	return nil
}

// checkSideValid verifies that every trade has Side == sdk.SideBuy or sdk.SideSell.
// Returns a BlindSpot if any trade has an invalid side, nil otherwise.
// When there are no trades, the invariant is vacuously true (returns nil).
func checkSideValid(result *backtest.Result) *antv1.BlindSpot {
	for _, t := range result.Trades {
		if t.Side != sdk.SideBuy && t.Side != sdk.SideSell {
			return &antv1.BlindSpot{
				Id:          "invalid_side",
				Category:    "invariant",
				Severity:    interp.SeverityFatal,
				Description: "存在交易方向非法的交易（非 BUY/SELL），回测结果不可信",
			}
		}
	}
	return nil
}

// checkTimeOrder verifies that every trade has EntryTime <= ExitTime.
// Returns a BlindSpot if any trade has EntryTime after ExitTime, nil otherwise.
// EntryTime == ExitTime is valid (same-bar entry and exit).
// When there are no trades, the invariant is vacuously true (returns nil).
func checkTimeOrder(result *backtest.Result) *antv1.BlindSpot {
	for _, t := range result.Trades {
		if t.EntryTime.After(t.ExitTime) {
			return &antv1.BlindSpot{
				Id:          "time_order_violation",
				Category:    "invariant",
				Severity:    interp.SeverityFatal,
				Description: "存在开仓时间晚于平仓时间的交易，回测结果不可信",
			}
		}
	}
	return nil
}

// checkCapitalConservation verifies the capital conservation identity:
//
//	|FinalBalance − (本金 + ΣProfit − ΣCommission − ΣSwap)| < 容差
//
// FinalBalance is the realized balance (excludes unrealized PnL), so the invariant
// holds regardless of open positions at backtest end.
// Returns a BlindSpot if the identity is violated, nil otherwise.
// 容差 = max(0.01, 1e-4 × 本金) — covers floating-point accumulation and minor swap/commission model discrepancies.
func checkCapitalConservation(result *backtest.Result) *antv1.BlindSpot {
	finalBalance := result.FinalBalance
	initialCapital := result.Config.InitialCapital

	var sumProfit, sumCommission, sumSwap decimal.Decimal
	for _, t := range result.Trades {
		sumProfit = sumProfit.Add(t.Profit)
		sumCommission = sumCommission.Add(t.Commission)
		sumSwap = sumSwap.Add(t.Swap)
	}

	expected := initialCapital.Add(sumProfit).Sub(sumCommission).Sub(sumSwap)
	diff := finalBalance.Sub(expected).Abs()

	tolerance := decimal.New(1, -2) // 0.01
	if scaled := initialCapital.Mul(decimal.New(1, -4)); scaled.GreaterThan(tolerance) {
		tolerance = scaled
	}

	if diff.GreaterThanOrEqual(tolerance) {
		return &antv1.BlindSpot{
			Id:          "capital_not_conserved",
			Category:    "invariant",
			Severity:    interp.SeverityFatal,
			Description: "资金不守恒：期末净值与 本金+Σ盈亏−Σ手续费−Σswap 对不上，回测结果不可信",
		}
	}
	return nil
}

// checkVolumeInvariant verifies that every trade has a strictly positive Volume.
// Returns a BlindSpot if any trade has Volume <= 0, nil otherwise.
// When there are no trades, the invariant is vacuously true (returns nil).
func checkVolumeInvariant(trades []backtest.Trade) *antv1.BlindSpot {
	for _, t := range trades {
		if !t.Volume.GreaterThan(decimal.Zero) {
			return &antv1.BlindSpot{
				Id:          "zero_volume_trade",
				Category:    "invariant",
				Severity:    interp.SeverityFatal,
				Description: "存在手数<=0的交易，回测结果不可信",
			}
		}
	}
	return nil
}

func attachBlindSpots(cov *mql2go.CoverageReport, vmRunner *mql2go.VMRunner, ruleFindings []mql2go.DiagnosticFinding) []*antv1.BlindSpot {
	var spots []*antv1.BlindSpot
	if cov != nil {
		for _, bs := range cov.BlindSpots {
			spots = append(spots, &antv1.BlindSpot{
				Id:          bs,
				Severity:    interp.SeverityForBuiltin(bs),
				Description: bs + " is not fully supported",
			})
		}
	}
	for _, rbs := range vmRunner.GetRuntimeBlindSpots() {
		spots = append(spots, &antv1.BlindSpot{
			Id:          rbs.Builtin,
			Severity:    rbs.Severity,
			Description: fmt.Sprintf("%s hit %d time(s) at runtime", rbs.Builtin, rbs.Count),
		})
	}
	for _, f := range ruleFindings {
		spots = append(spots, &antv1.BlindSpot{
			Id:          f.RuleID,
			Severity:    interp.EnglishToChineseSeverity(f.Severity),
			Description: f.Title + ": " + f.Detail,
		})
	}
	return spots
}
