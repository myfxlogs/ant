package strategy

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"alphaforge/internal/repository"
	antv1 "alphaforge/gen/proto/ant/v1"
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

	var vmRunner *mql2go.VMRunner
	if len(cachedBytecode) > 0 {
		if r, err := mql2go.CompileMQLFromBytecode(cachedBytecode); err == nil {
			vmRunner = r
		}
	}
	if vmRunner == nil {
		r, err := mql2go.CompilePython(params.code)
		if err != nil {
			s.log.Error("executePythonVMBacktest: compile failed", zap.Error(err))
			return nil, fmt.Errorf("compile Python: %w", err)
		}
		vmRunner = r
	}
	s.log.Info("executePythonVMBacktest: compiled successfully")

	// Persist newly compiled bytecode for future runs.
	if run.StrategyID != nil && s.importedRepo != nil {
		if bcData, mErr := mql2go.MarshalBytecode(vmRunner.Bytecode()); mErr == nil {
			if saveErr := s.importedRepo.SaveBytecode(ctx, *run.StrategyID, bcData); saveErr != nil {
				s.log.Warn("executePythonVMBacktest: save bytecode cache failed", zap.Error(saveErr))
			}
		}
	}

	return s.runVMEngine(ctx, vmRunner, params, klines, run)
}
