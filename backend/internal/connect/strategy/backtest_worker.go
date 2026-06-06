package strategy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/repository"
	"connectrpc.com/connect"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

const (
	pollInterval   = 3 * time.Second
	leaseFor       = 60 * time.Second
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

// executeBacktestRun orchestrates a full backtest life cycle: parameter extraction,
// K-line fetching, Python engine execution, and result persistence.
// TODO(P2-1): Split into sub-functions — fetchKlineData, buildBacktestRequest, callPythonEngine.
// Current 181-line body exceeds the 50-line function limit (3.6x over).

// backtestParams holds extracted parameters from a BacktestRun for execution.
type backtestParams struct {
	code           string
	initialCapital float64
	commission     float64
	slippage       float64
	leverage       float64
	tradeDir       antv1.TradeDirection
	strictMode     bool
	strategyCfg    *antv1.StrategyConfig
}

// extractBacktestParams extracts and validates parameters from a BacktestRun.
func extractBacktestParams(run *repository.BacktestRun) (backtestParams, error) {
	p := backtestParams{
		initialCapital: 10000, commission: 0.001, slippage: 0, leverage: 1,
		tradeDir: antv1.TradeDirection_TRADE_DIRECTION_BOTH, strictMode: true,
	}
	if run.StrategyCode != nil {
		p.code = *run.StrategyCode
	}
	if p.code == "" {
		return p, fmt.Errorf("strategy code is empty")
	}
	if run.InitialCapital != nil {
		p.initialCapital = *run.InitialCapital
	}
	if run.Commission != nil {
		p.commission = *run.Commission
	}
	if run.Slippage != nil {
		p.slippage = *run.Slippage
	}
	if run.Leverage != nil && *run.Leverage > 0 {
		p.leverage = *run.Leverage
	}
	if run.TradeDirection != nil {
		p.tradeDir = stringToTradeDirection(*run.TradeDirection)
	}
	if run.StrictMode != nil {
		p.strictMode = *run.StrictMode
	}
	if len(run.ConfigSnapshot) > 0 {
		var ec antv1.BacktestExecutionConfig
		if err := proto.Unmarshal(run.ConfigSnapshot, &ec); err == nil {
			p.strategyCfg = ec.GetStrategyConfig()
		}
	}
	return p, nil
}

// startBacktestWatchers starts lease heartbeat and cancel polling goroutines.
// Returns a derived context that is cancelled when the user requests cancellation.
func (s *PythonStrategyServer) startBacktestWatchers(ctx context.Context, run *repository.BacktestRun, leaseFor time.Duration) context.Context {
	execCtx, execCancel := context.WithCancel(ctx)
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
					s.log.Warn("extend lease failed", zap.String("runID", run.ID.String()), zap.Error(err))
				}
			}
		}
	}()
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
					s.log.Warn("cancel poll failed", zap.String("runID", run.ID.String()), zap.Error(err))
					continue
				}
				if status == StatusCancelRequested {
					s.log.Info("cancelling run per user request", zap.String("runID", run.ID.String()))
					execCancel()
					return
				}
			}
		}
	}()
	return execCtx
}

// fetchBacktestKlines retrieves K-line data from ClickHouse and converts to proto format.
func (s *PythonStrategyServer) fetchBacktestKlines(ctx context.Context, run *repository.BacktestRun) []*antv1.ExecuteKlineBar {
	if s.marketDataRepo == nil || run.Symbol == "" || run.Timeframe == "" {
		return nil
	}
	chBars, _ := s.marketDataRepo.GetKlines(ctx, run.Symbol, "", run.Timeframe, run.FromTs, run.ToTs, 2000)
	klines := make([]*antv1.ExecuteKlineBar, 0, len(chBars))
	for i := len(chBars) - 1; i >= 0; i-- {
		b := chBars[i]
		klines = append(klines, &antv1.ExecuteKlineBar{
			OpenTimeMs:  int64(b.OpenTsUnixMs),
			CloseTimeMs: int64(b.CloseTsUnixMs),
			Open:        b.Open, High: b.High, Low: b.Low, Close: b.Close, Volume: b.Volume,
		})
	}
	return klines
}
func (s *PythonStrategyServer) executeBacktestRun(ctx context.Context, run *repository.BacktestRun, leaseFor time.Duration) {
	if s.backtestClient == nil {
		s.failRun(ctx, run, "Python strategy service not available")
		return
	}
	params, err := extractBacktestParams(run)
	if err != nil {
		s.failRun(ctx, run, err.Error())
		return
	}
	execCtx := s.startBacktestWatchers(ctx, run, leaseFor)
	defer func() { _ = execCtx }() // capture; cancel is handled by watcher goroutines

	klines := s.fetchBacktestKlines(ctx, run)
	fromMs, toMs := int64(0), int64(0)
	if run.FromTs != nil { fromMs = run.FromTs.UnixMilli() }
	if run.ToTs != nil { toMs = run.ToTs.UnixMilli() }

	resp, err := s.backtestClient.RunBacktest(execCtx,
		connect.NewRequest(&antv1.ExecuteBacktestRequest{
			StrategyId:        run.ID.String(),
			StrategyCode:      params.code,
			Symbol:            run.Symbol,
			Timeframe:         run.Timeframe,
			StartDateMs:       fromMs,
			EndDateMs:         toMs,
			InitialCapital:    params.initialCapital,
			Commission:        params.commission,
			SlippageRate:      params.slippage,
			SlippageMode:      "fixed",
			SlippageSeed:      42,
			Leverage:          params.leverage,
			TradeDirection:    params.tradeDir,
			StrictMode:        params.strictMode,
			StrategyConfig:    params.strategyCfg,
			Klines:            klines,
			StrategyParamsJson: paramsProtoToJSON(run.ParameterOverrides),
		}))
	if err != nil {
		if execCtx.Err() != nil {
			s.log.Info("backtest worker: run cancelled", zap.String("runID", run.ID.String()))
			now := time.Now()
			if uerr := s.backtestRepo.UpdateAsyncFields(ctx, run.UserID, run.ID, StatusCanceled, "cancelled by user", nil, &now, nil); uerr != nil {
				s.log.Error("update backtest run to CANCELED failed", zap.Error(uerr), zap.String("runID", run.ID.String()))
			}
			return
		}
		s.log.Error("backtest worker: python backtest failed", zap.String("runID", run.ID.String()), zap.Error(err))
		s.failRun(ctx, run, fmt.Sprintf("backtest execution failed: %v", err))
		return
	}

	result := resp.Msg
	if !result.GetSuccess() {
		s.failRun(ctx, run, result.GetError())
		return
	}

	protoResp, err := proto.Marshal(result)
	if err != nil {
		s.failRun(ctx, run, fmt.Sprintf("proto marshal failed: %v", err))
		return
	}

	now := time.Now()
	BacktestRunsTotal.WithLabelValues(StatusSucceeded).Inc()
	if err := s.backtestRepo.UpdateAsyncFields(ctx, run.UserID, run.ID, StatusSucceeded, "", &now, &now, protoResp); err != nil {
		s.log.Error("backtest worker: UpdateAsyncFields failed", zap.String("runID", run.ID.String()), zap.Error(err))
		return
	}
	s.log.Info("backtest worker: run completed",
		zap.String("runID", run.ID.String()),
		zap.Float64("total_return", result.GetMetrics().GetTotalReturn()),
		zap.Float64("sharpe", result.GetMetrics().GetSharpeRatio()))
}

func (s *PythonStrategyServer) failRun(ctx context.Context, run *repository.BacktestRun, errMsg string) {
	now := time.Now()
	BacktestRunsTotal.WithLabelValues(StatusFailed).Inc()
	status := StatusFailed
	if err := s.backtestRepo.UpdateAsyncFields(ctx, run.UserID, run.ID, status, errMsg, nil, &now, nil); err != nil {
		s.log.Error("backtest worker: failRun UpdateAsyncFields",
			zap.String("run_id", run.ID.String()), zap.Error(err))
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
// paramsToJSON converts proto binary StrategyParams to JSON string for Python engine compat.
 func paramsProtoToJSON(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var sp antv1.StrategyParams
	if err := proto.Unmarshal(raw, &sp); err != nil {
		return ""
	}
	if len(sp.GetValues()) == 0 {
		return ""
	}
	b, _ := json.Marshal(sp.GetValues())
	return string(b)
}

