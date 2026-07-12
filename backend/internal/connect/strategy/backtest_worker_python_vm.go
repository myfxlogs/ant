package strategy

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/repository"
	"alphaforge/tools/mql2go"
)

// executePythonVMBacktest runs a backtest via the in-process Bytecode VM for Python subset:
// Python source → CompilePython → VMRunner → backtest.Engine → ExecuteBacktestResponse.
func (s *StrategyExecutionServer) executePythonVMBacktest(ctx context.Context, params backtestParams, klines []*antv1.ExecuteKlineBar, run *repository.BacktestRun) (*antv1.ExecuteBacktestResponse, error) {
	s.log.Info("executePythonVMBacktest: starting",
		zap.Int("klines", len(klines)),
		zap.String("symbol", run.Symbol),
		zap.String("timeframe", run.Timeframe))

	vmRunner, err := mql2go.CompilePython(params.code)
	if err != nil {
		s.log.Error("executePythonVMBacktest: compile failed", zap.Error(err))
		return nil, fmt.Errorf("compile Python: %w", err)
	}
	s.log.Info("executePythonVMBacktest: compiled successfully")

	return s.runVMEngine(ctx, vmRunner, params, klines, run)
}
