package strategy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/repository"
	"connectrpc.com/connect"
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
	if s.backtestClient == nil {
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

	_ = time.Now() // dates now passed as Unix ms via fromMs/toMs
	

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

	// Call Python backtest via ConnectRPC (proto contract).
	klines := make([]*antv1.ExecuteKlineBar, 0)
	if s.marketDataRepo != nil && run.Symbol != "" && run.Timeframe != "" {
		chBars, _ := s.marketDataRepo.GetKlines(ctx, run.Symbol, "", run.Timeframe, run.FromTs, run.ToTs, 2000)
		for _, b := range chBars {
			klines = append(klines, &antv1.ExecuteKlineBar{
				OpenTimeMs: int64(b.OpenTsUnixMs), CloseTimeMs: int64(b.CloseTsUnixMs),
				Open: b.Open, High: b.High, Low: b.Low, Close: b.Close, Volume: b.Volume,
			})
		}
	}

	fromMs := int64(0)
	toMs := int64(0)
	if run.FromTs != nil { fromMs = run.FromTs.UnixMilli() }
	if run.ToTs != nil { toMs = run.ToTs.UnixMilli() }

	resp, err := s.backtestClient.RunBacktest(execCtx,
		connect.NewRequest(&antv1.ExecuteBacktestRequest{
			StrategyId: run.ID.String(), StrategyCode: code,
			Symbol: run.Symbol, Timeframe: run.Timeframe,
			StartDateMs: fromMs, EndDateMs: toMs,
			InitialCapital: initialCapital, Commission: 0,
			Klines: klines,
			StrategyParamsJson: string(run.ParameterOverrides),
		}))
	if err != nil {
		if execCtx.Err() != nil {
			s.log.Info("backtest worker: run cancelled", zap.String("run_id", run.ID.String()))
			now := time.Now()
			_ = s.backtestRepo.UpdateAsyncFields(ctx, run.UserID, run.ID, "CANCELED", "cancelled by user", nil, &now, nil, nil)
			return
		}
		s.log.Error("backtest worker: python backtest failed", zap.String("run_id", run.ID.String()), zap.Error(err))
		s.failRun(ctx, run.ID, fmt.Sprintf("backtest execution failed: %v", err))
		return
	}

	result := resp.Msg
	if !result.GetSuccess() {
		s.failRun(ctx, run.ID, result.GetError())
		return
	}

	// Build metrics JSON for storage
	m := result.Metrics
	metricsDoc := map[string]interface{}{
		"total_return":    m.TotalReturn,
		"annual_return":   m.AnnualReturn,
		"max_drawdown":    m.MaxDrawdown,
		"sharpe_ratio":    m.SharpeRatio,
		"win_rate":        m.WinRate,
		"profit_factor":   m.ProfitFactor,
		"total_trades":    m.TotalTrades,
		"winning_trades":  m.WinningTrades,
		"losing_trades":   m.LosingTrades,
		"average_profit":  m.AverageProfit,
		"average_loss":    m.AverageLoss,
		"risk_score":      result.GetRisk().GetScore(),
		"risk_level":      result.GetRisk().GetLevel(),
		"risk_reasons":    result.GetRisk().GetReasons(),
		"risk_warnings":   result.GetRisk().GetWarnings(),
		"is_reliable":     result.GetRisk().GetIsReliable(),
	}
	metricsJSON, err := json.Marshal(metricsDoc)
	if err != nil {
		s.failRun(ctx, run.ID, fmt.Sprintf("metrics marshal failed: %v", err))
		return
	}
	equityJSON, _ := json.Marshal(result.GetEquityCurve())

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
		zap.Float64("total_return", result.GetMetrics().GetTotalReturn()),
		zap.Float64("sharpe", m.SharpeRatio))
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
	if s.backtestClient == nil {
		s.log.Warn("backtest worker: Python client not configured, workers will not start")
		return
	}
	s.log.Info("backtest worker: starting pool", zap.Int("workers", defaultWorkers))
	for i := 0; i < defaultWorkers; i++ {
		go s.backtestWorker(ctx, i)
	}
}
