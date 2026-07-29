package marketplace

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// runOptimizationGeneration invokes the AI generator to produce an optimized
// version of the strategy referenced by the task, then writes the result back
// to the marketplace_strategy_optimization_tasks row.
// It is intended to be run in a background goroutine.
func (s *Service) runOptimizationGeneration(ctx context.Context, taskID, publisherID uuid.UUID) {
	log := s.log.With(zap.String("task_id", taskID.String()))

	// Mark the task as generating.
	tag, err := s.pg.Exec(ctx,
		`UPDATE marketplace_strategy_optimization_tasks
		 SET status = 'generating', updated_at = now()
		 WHERE id = $1 AND publisher_id = $2 AND status IN ('pending', 'generating')`,
		taskID, publisherID,
	)
	if err != nil {
		log.Warn("optimization generation: failed to mark generating", zap.Error(err))
		return
	}
	if tag.RowsAffected() == 0 {
		log.Warn("optimization generation: task not found or not pending")
		return
	}

	// Fetch task details + source code + metadata.
	_, triggerReason, sourceCode, symbol, timeframe, err := s.fetchOptimizationTask(ctx, taskID, publisherID, log)
	if err != nil {
		_ = s.markOptimizationFailed(ctx, taskID, publisherID, err.Error())
		return
	}

	if s.optimizer == nil {
		_ = s.markOptimizationFailed(ctx, taskID, publisherID, "optimizer not configured")
		return
	}

	msg := &antv1.AgentGenerateStrategyRequest{
		Message: fmt.Sprintf("Optimize this strategy to address alpha decay. Trigger: %s", triggerReason),
		Locale:  "en",
		BacktestConfig: &antv1.AgentBacktestConfig{
			Symbol:         symbol,
			Timeframe:      timeframe,
			InitialCapital: "10000",
			Commission:     "0.0003",
			Slippage:       "0.0001",
			Leverage:       "100",
		},
	}
	// If the original source is available, pass it as CurrentCode so the agent
	// can iterate on the existing implementation rather than starting from scratch.
	if sourceCode != "" {
		msg.CurrentCode = sourceCode
	}

	var pythonSource string
	var backtestResult *antv1.AgentBacktestResult
	genErr := s.optimizer.Generate(ctx, publisherID, msg, func(chunk *antv1.AgentGenerateStrategyChunk) error {
		if chunk == nil {
			return nil
		}
		if chunk.Phase == "done" {
			pythonSource = chunk.PythonSource
			backtestResult = chunk.Result
			if chunk.Error != "" {
				return fmt.Errorf("generator error: %s", chunk.Error)
			}
		}
		return nil
	})

	if genErr != nil {
		log.Warn("optimization generation: generator failed", zap.Error(genErr))
		_ = s.markOptimizationFailed(ctx, taskID, publisherID, genErr.Error())
		return
	}
	if pythonSource == "" {
		log.Warn("optimization generation: no source produced")
		_ = s.markOptimizationFailed(ctx, taskID, publisherID, "no optimized source produced")
		return
	}

	// Store the completed result.
	s.storeOptimizationResult(ctx, taskID, publisherID, pythonSource, symbol, timeframe, backtestResult, log)
}

func (s *Service) fetchOptimizationTask(ctx context.Context, taskID, publisherID uuid.UUID, log *zap.Logger) (strategyID uuid.UUID, triggerReason, sourceCode, symbol, timeframe string, err error) {
	err = s.pg.QueryRow(ctx,
		`SELECT strategy_id, trigger_reason
		 FROM marketplace_strategy_optimization_tasks
		 WHERE id = $1 AND publisher_id = $2`,
		taskID, publisherID,
	).Scan(&strategyID, &triggerReason)
	if err != nil {
		log.Warn("optimization generation: failed to fetch task", zap.Error(err))
		return uuid.Nil, "", "", "", "", fmt.Errorf("failed to fetch task")
	}

	err = s.pg.QueryRow(ctx,
		`SELECT source_code FROM imported_strategies WHERE id = $1`,
		strategyID,
	).Scan(&sourceCode)
	if err != nil {
		log.Warn("optimization generation: failed to load source code", zap.Error(err))
		return uuid.Nil, "", "", "", "", fmt.Errorf("source code not found")
	}

	var symbols []string
	_ = s.pg.QueryRow(ctx,
		`SELECT symbols, timeframe FROM marketplace_strategies WHERE strategy_id = $1`,
		strategyID,
	).Scan(&symbols, &timeframe)
	symbol = "EURUSD"
	if len(symbols) > 0 && symbols[0] != "" {
		symbol = symbols[0]
	}
	if timeframe == "" {
		timeframe = "H1"
	}
	return strategyID, triggerReason, sourceCode, symbol, timeframe, nil
}

func (s *Service) storeOptimizationResult(ctx context.Context, taskID, publisherID uuid.UUID, pythonSource, symbol, timeframe string, backtestResult *antv1.AgentBacktestResult, log *zap.Logger) {
	var snapshotBytes []byte
	if backtestResult != nil {
		snap := &antv1.BacktestSnapshot{
			TotalReturn:  backtestResult.TotalReturn,
			WinRate:      backtestResult.WinRate,
			MaxDrawdown:  backtestResult.MaxDrawdown,
			SharpeRatio:  backtestResult.SharpeRatio,
			TotalTrades:  backtestResult.TotalTrades,
			Symbol:       symbol,
			Timeframe:    timeframe,
			SnapshotAt:   timestamppb.Now(),
		}
		snapshotBytes, _ = proto.Marshal(snap)
	}

	changeSummary := "AI-optimized strategy generated"
	if backtestResult != nil {
		changeSummary = fmt.Sprintf(
			"Optimized strategy: total_return=%s sharpe=%s win_rate=%s trades=%d",
			backtestResult.TotalReturn, backtestResult.SharpeRatio, backtestResult.WinRate, backtestResult.TotalTrades,
		)
	}

	_, err := s.pg.Exec(ctx,
		`UPDATE marketplace_strategy_optimization_tasks
		 SET status = 'completed', suggested_code = $3, suggested_params = $4,
		     change_summary = $5, backtest_snapshot = $6,
		     completed_at = now(), updated_at = now()
		 WHERE id = $1 AND publisher_id = $2 AND status = 'generating'`,
		taskID, publisherID, pythonSource, "", changeSummary, snapshotBytes,
	)
	if err != nil {
		log.Warn("optimization generation: failed to store result", zap.Error(err))
		_ = s.markOptimizationFailed(ctx, taskID, publisherID, err.Error())
	}
}

// markOptimizationFailed records a terminal failure for the optimization task.
func (s *Service) markOptimizationFailed(ctx context.Context, taskID, publisherID uuid.UUID, reason string) error {
	_, err := s.pg.Exec(ctx,
		`UPDATE marketplace_strategy_optimization_tasks
		 SET status = 'failed', change_summary = $3, updated_at = now()
		 WHERE id = $1 AND publisher_id = $2`,
		taskID, publisherID, reason,
	)
	return err
}
