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

	// Bytecode cache omits CoverageResult; recompile from source to restore
	// severity-aware blind spots and Defense A data on cache hits.
	if vmRunner.GetCoverageResult() == nil && params.code != "" {
		covRunner, cov, covErr := mql2go.CompileMQLWithCoverage(params.code)
		if covErr != nil {
			s.log.Error("executeVMBacktest: restore coverage failed", zap.Error(covErr))
			return nil, fmt.Errorf("restore MQL coverage: %w", covErr)
		}
		vmRunner.InjectCoverage(covRunner.GetCoverage())
		vmRunner.InjectCoverageResult(cov)
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

	s.log.Info("runVMEngine config",
		zap.Int32("leverage", cfg.Leverage),
		zap.String("initialCapital", cfg.InitialCapital.String()),
		zap.String("commission", cfg.Commission.String()),
		zap.String("slippage", cfg.Slippage.String()),
		zap.String("swapRate", cfg.SwapRate.String()),
		zap.Int32("digits", cfg.SymbolDigits),
		zap.String("point", cfg.SymbolPoint.String()),
		zap.String("contractSize", cfg.ContractSize.String()),
		zap.String("volumeMin", cfg.VolumeMin.String()),
		zap.String("volumeStep", cfg.VolumeStep.String()),
		zap.Bool("strictMode", cfg.StrictMode),
		zap.Int("bars", len(bars)),
		zap.Int("paramsCount", len(cfg.Params)))

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
	signalTiming := params.signalTiming
	if signalTiming == "" {
		if params.strictMode {
			signalTiming = "next_bar_open"
		} else {
			signalTiming = "same_bar_close"
		}
	}
	fillRule := params.fillRule
	if fillRule == "" {
		fillRule = "bar_close"
	}
	simulationMode := params.simulationMode
	if simulationMode == "" {
		simulationMode = "KLINE_RANGE"
	}
	cfg := backtest.Config{
		Symbol:         run.Symbol,
		Timeframe:      run.Timeframe,
		InitialCapital: parseDecimal(params.initialCapital),
		Leverage:       parseInt32(params.leverage),
		Commission:     parseDecimal(params.commission),
		Slippage:       parseDecimal(params.slippage),
		SwapRate:       decimal.NewFromFloat(0.00001),
		StrictMode:     signalTiming == "next_bar_open",
		SignalTiming:   signalTiming,
		FillRule:       fillRule,
		SimulationMode: simulationMode,
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
