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

	// VM-CACHE-INTEGRITY-2: use CompilePythonCached which verifies SourceHash
	// before accepting cached bytecode, preventing stale cache execution.
	vmRunner, bcData, err := mql2go.CompilePythonCached(params.code, cachedBytecode)
	if err != nil {
		s.log.Error("executePythonVMBacktest: compile failed", zap.Error(err))
		return nil, fmt.Errorf("compile Python: %w", err)
	}
	s.log.Info("executePythonVMBacktest: compiled successfully")

	// Bytecode cache omits CoverageReport; recompile from source to recover
	// coverage/blind-spot data when cache hit produced nil coverage.
	// Mirrors backtest_worker_vm.go:42-48 (MQL path).
	if vmRunner.GetCoverage() == nil && params.code != "" {
		if covRunner, cov, covErr := mql2go.CompilePythonWithCoverage(params.code); covErr == nil {
			vmRunner.InjectCoverage(covRunner.GetCoverage())
			vmRunner.InjectDefenseAViolations(cov.DefenseAViolations)
			_ = covRunner
		}
	}

	// Persist newly compiled bytecode for future runs.
	if bcData != nil && run.StrategyID != nil && s.importedRepo != nil {
		if saveErr := s.importedRepo.SaveBytecode(ctx, *run.StrategyID, bcData); saveErr != nil {
			s.log.Warn("executePythonVMBacktest: save bytecode cache failed", zap.Error(saveErr))
		}
	}

	return s.runVMEngine(ctx, vmRunner, params, klines, run)
}
