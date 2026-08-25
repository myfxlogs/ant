package strategy

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/repository"
	"alphaforge/tools/mql2go"
)

// executePythonVMBacktest runs a backtest via the in-process Bytecode VM for Python strategies.
// Python source → CompilePython → VMRunner → backtest.Engine → ExecuteBacktestResponse.
// Uses bytecode cache from imported_strategies when strategy_id is available.
func (s *StrategyExecutionServer) executePythonVMBacktest(ctx context.Context, params backtestParams, klines []*antv1.ExecuteKlineBar, run *repository.BacktestRun) (*antv1.ExecuteBacktestResponse, error) {
	s.log.Info("executePythonVMBacktest: starting",
		zap.Int("klines", len(klines)),
		zap.String("symbol", run.Symbol),
		zap.String("timeframe", run.Timeframe))

	var cachedBytecode []byte
	if run.StrategyID != nil && s.importedRepo != nil {
		cachedBytecode, _ = s.importedRepo.GetBytecode(ctx, *run.StrategyID)
	}

	// VM-CACHE-INTEGRITY-2: Use CompilePythonCached which verifies SourceHash
	// before accepting cached bytecode, mirroring the MQL path. This prevents
	// stale bytecode from a different source version from being used silently.
	vmRunner, bcData, err := mql2go.CompilePythonCached(params.code, cachedBytecode)
	if err != nil {
		s.log.Error("executePythonVMBacktest: compile failed", zap.Error(err))
		return nil, fmt.Errorf("compile Python: %w", err)
	}
	s.log.Info("executePythonVMBacktest: compiled successfully")

	// VM-CACHE-INTEGRITY-2: Bytecode cache omits CoverageResult; recompile from
	// source to restore severity-aware blind spots and Defense A data on cache
	// hits. Mirrors the MQL path in executeVMBacktest.
	if vmRunner.GetCoverageResult() == nil && params.code != "" {
		covRunner, cov, covErr := mql2go.CompilePythonWithCoverage(params.code)
		if covErr != nil {
			s.log.Error("executePythonVMBacktest: restore coverage failed", zap.Error(covErr))
			return nil, fmt.Errorf("restore Python coverage: %w", covErr)
		}
		vmRunner.InjectCoverage(covRunner.GetCoverage())
		vmRunner.InjectCoverageResult(cov)
	}

	// Persist newly compiled bytecode for future runs.
	if bcData != nil && run.StrategyID != nil && s.importedRepo != nil {
		if saveErr := s.importedRepo.SaveBytecode(ctx, *run.StrategyID, bcData); saveErr != nil {
			s.log.Warn("executePythonVMBacktest: save bytecode cache failed", zap.Error(saveErr))
		}
	}

	return s.runVMEngine(ctx, vmRunner, params, klines, run)
}
