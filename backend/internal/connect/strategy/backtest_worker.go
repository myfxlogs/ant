package strategy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/repository"
	"anttrader/internal/strategysvc"
	"go.uber.org/zap"
)

const (
	pollInterval = 3 * time.Second
	leaseFor     = 60 * time.Second
	defaultWorkers = 3
)

// backtestWorker polls for PENDING backtest runs and executes them via the Python engine.
func (s *PythonStrategyServer) backtestWorker(ctx context.Context, workerID int) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		// Claim next pending run (atomic SKIP LOCKED — concurrent-safe).
		leaseUntil := time.Now().Add(leaseFor)
		run, err := s.backtestRepo.ClaimNextForWork(ctx, leaseUntil)
		if err != nil {
			s.log.Warn("backtest worker: ClaimNextForWork",
				zap.Int("worker", workerID), zap.Error(err))
			continue
		}
		if run == nil {
			continue // no pending work
		}

		s.log.Info("backtest worker: claimed run",
			zap.Int("worker", workerID),
			zap.String("run_id", run.ID.String()),
			zap.String("symbol", run.Symbol),
			zap.String("timeframe", run.Timeframe))

		s.executeBacktestRun(ctx, run, leaseFor)
	}
}

func (s *PythonStrategyServer) executeBacktestRun(ctx context.Context, run *repository.BacktestRun, leaseFor time.Duration) {
	if s.client == nil {
		s.failRun(ctx, run.ID, "Python strategy service not available")
		return
	}
	code := ""
	if run.StrategyCode != nil {
		code = *run.StrategyCode
	}
	if code == "" {
		s.failRun(ctx, run.ID, "strategy code is empty")
		return
	}

	startDate := time.Now().AddDate(0, -3, 0).Format("2006-01-02")
	endDate := time.Now().Format("2006-01-02")
	if run.FromTs != nil {
		startDate = run.FromTs.Format("2006-01-02")
	}
	if run.ToTs != nil {
		endDate = run.ToTs.Format("2006-01-02")
	}

	initialCapital := 10000.0
	if run.InitialCapital != nil {
		initialCapital = *run.InitialCapital
	}

	// Cancellable context: lease heartbeat + cancel polling share a derived context.
	execCtx, execCancel := context.WithCancel(ctx)
	defer execCancel()

	// Extend lease during execution — runs every leaseFor/2.
	go func() {
		ticker := time.NewTicker(leaseFor / 2)
		defer ticker.Stop()
		for {
			select {
			case <-execCtx.Done():
				return
			case <-ticker.C:
				newLease := time.Now().Add(leaseFor)
				if err := s.backtestRepo.ExtendLease(ctx, run.UserID, run.ID, newLease); err != nil {
					s.log.Warn("backtest worker: extend lease failed", zap.String("run_id", run.ID.String()), zap.Error(err))
				}
			}
		}
	}()

	// Cancel polling: check DB every 5s for CANCEL_REQUESTED status.
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-execCtx.Done():
				return
			case <-ticker.C:
				status, _, err := s.backtestRepo.GetStatusAndCancelRequestedAt(ctx, run.UserID, run.ID)
				if err != nil {
					s.log.Warn("backtest worker: cancel poll failed",
						zap.String("run_id", run.ID.String()), zap.Error(err))
					continue
				}
				if status == "CANCEL_REQUESTED" {
					s.log.Info("backtest worker: cancelling run per user request",
						zap.String("run_id", run.ID.String()))
					execCancel()
					return
				}
			}
		}
	}()

	// Call Python engine with cancellable context.
	// Fetch K-lines from ClickHouse md_bars.
	klines := []strategysvc.KlineBar{}
	if s.marketDataRepo != nil && run.Symbol != "" && run.Timeframe != "" {
		chBars, _ := s.marketDataRepo.GetKlines(ctx, run.Symbol, "", run.Timeframe, run.FromTs, run.ToTs, 2000)
		for _, b := range chBars {
			klines = append(klines, strategysvc.KlineBar{
				OpenTime: time.UnixMilli(int64(b.OpenTsUnixMs)).Format(time.RFC3339),
				CloseTime: time.UnixMilli(int64(b.CloseTsUnixMs)).Format(time.RFC3339),
				Open: b.Open, High: b.High, Low: b.Low, Close: b.Close, Volume: b.Volume,
			})
		}
	}

	result, err := s.client.Backtest(execCtx, &strategysvc.BacktestRequest{
		Code:       code,
		Symbol:    run.Symbol,
		Timeframe: run.Timeframe,
		StartDate: startDate,
		EndDate:   endDate,
		Capital:   initialCapital,
		Commission: 0,
		Klines:     klines,
	})
	if err != nil {
		if execCtx.Err() != nil {
			// Cancelled by user — mark as cancelled not failed.
			s.log.Info("backtest worker: run cancelled",
				zap.String("run_id", run.ID.String()))
			now := time.Now()
			status := "CANCELED"
			_ = s.backtestRepo.UpdateAsyncFields(ctx, run.UserID, run.ID, status, "cancelled by user", nil, &now, nil, nil)
			return
		}
		s.log.Error("backtest worker: python backtest failed",
			zap.String("run_id", run.ID.String()), zap.Error(err))
		s.failRun(ctx, run.ID, fmt.Sprintf("backtest execution failed: %v", err))
		return
	}

	if !result.Success {
		s.failRun(ctx, run.ID, result.Error)
		return
	}

	// Build metrics JSON for storage
	metricsDoc := map[string]interface{}{
		"total_return":    result.TotalReturn,
		"annual_return":   result.AnnualReturn,
		"max_drawdown":    result.MaxDrawdown,
		"sharpe_ratio":    result.SharpeRatio,
		"win_rate":        result.WinRate,
		"profit_factor":   result.ProfitFactor,
		"total_trades":    result.TotalTrades,
		"winning_trades":  result.WinningTrades,
		"losing_trades":   result.LosingTrades,
		"average_profit":  result.AverageProfit,
		"average_loss":    result.AverageLoss,
		"risk_score":      result.RiskScore,
		"risk_level":      result.RiskLevel,
		"risk_reasons":    result.RiskReasons,
		"risk_warnings":   result.RiskWarnings,
		"is_reliable":     result.IsReliable,
	}
	metricsJSON, err := json.Marshal(metricsDoc)
	if err != nil {
		s.failRun(ctx, run.ID, fmt.Sprintf("metrics marshal failed: %v", err))
		return
	}
	equityJSON, _ := json.Marshal(result.EquityCurve)

	// Validate metrics can round-trip through proto types — catch schema drift early
	var checkMetrics antv1.BacktestMetrics
	var checkRisk antv1.BacktestRisk
	if err := json.Unmarshal(metricsJSON, &checkMetrics); err != nil {
		s.failRun(ctx, run.ID, fmt.Sprintf("metrics validation failed: %v", err))
		return
	}
	if err := json.Unmarshal(metricsJSON, &checkRisk); err != nil {
		s.failRun(ctx, run.ID, fmt.Sprintf("risk validation failed: %v", err))
		return
	}
	if checkMetrics.TotalTrades == 0 && checkRisk.Score == 0 {
		s.log.Warn("backtest worker: metrics JSON parsed but appears empty",
			zap.String("run_id", run.ID.String()))
	}

	now := time.Now()
	status := "SUCCEEDED"
	if err := s.backtestRepo.UpdateAsyncFields(ctx, run.UserID, run.ID, status, "", &now, &now, metricsJSON, equityJSON); err != nil {
		s.log.Error("backtest worker: UpdateAsyncFields failed",
			zap.String("run_id", run.ID.String()), zap.Error(err))
		return
	}

	s.log.Info("backtest worker: run completed",
		zap.String("run_id", run.ID.String()),
		zap.Float64("total_return", result.TotalReturn),
		zap.Float64("sharpe", result.SharpeRatio))
}

func (s *PythonStrategyServer) failRun(ctx context.Context, runID uuid.UUID, errMsg string) {
	now := time.Now()
	status := "FAILED"
	if err := s.backtestRepo.UpdateAsyncFields(ctx, uuid.Nil, runID, status, errMsg, nil, &now, nil, nil); err != nil {
		s.log.Error("backtest worker: failRun UpdateAsyncFields",
			zap.String("run_id", runID.String()), zap.Error(err))
	}
}

// StartBacktestWorker launches a pool of background workers that poll for PENDING backtest
// runs and execute them via the Python strategy engine. Call this once during server startup.
// Defaults to 3 concurrent workers; each claims a run via SKIP LOCKED.
func (s *PythonStrategyServer) StartBacktestWorker(ctx context.Context) {
	if s.client == nil {
		s.log.Warn("backtest worker: Python client not configured, workers will not start")
		return
	}
	s.log.Info("backtest worker: starting pool", zap.Int("workers", defaultWorkers))
	for i := 0; i < defaultWorkers; i++ {
		go s.backtestWorker(ctx, i)
	}
}
