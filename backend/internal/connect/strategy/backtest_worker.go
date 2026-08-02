package strategy

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/repository"
)

const (
	listenTimeout  = 30 * time.Second // fallback poll interval when no PG notification received
	leaseFor       = 60 * time.Second
	defaultWorkers = 3
)

// backtestWorker waits for new PENDING backtest runs via PG LISTEN/NOTIFY,
// with a 30s fallback ticker for safety (Push-First pattern). This replaces
// the previous 3-second polling, which violated the project's push-first rule.
func (s *StrategyExecutionServer) backtestWorker(ctx context.Context, workerID int) {
	// Subscribe to PG notifications for new pending runs.
	notifCh, listenCancel, _ := s.pgListen.Listen(ctx, "backtest_pending")
	if listenCancel != nil {
		defer listenCancel()
	}

	ticker := time.NewTicker(listenTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-notifCh:
			// New pending run inserted — wake up immediately.
		case <-ticker.C:
			// Safety fallback in case notification was missed.
		}

		// Claim next pending run (atomic SKIP LOCKED — concurrent-safe).
		leaseUntil := time.Now().Add(leaseFor)
		run, err := s.backtestRepo.ClaimNextForWork(ctx, leaseUntil)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue // no pending work — normal on fallback ticker
			}
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

// backtestParams holds extracted parameters from a BacktestRun for execution.
type backtestParams struct {
	code           string
	initialCapital string
	commission     string
	slippage       string
	leverage       string
	tradeDir       antv1.TradeDirection
	strictMode     bool
	strategyCfg    *antv1.StrategyConfig
}

// extractBacktestParams extracts and validates parameters from a BacktestRun.
func extractBacktestParams(run *repository.BacktestRun) (backtestParams, error) {
	p := backtestParams{
		initialCapital: "10000", commission: "0.001", slippage: "0", leverage: "1",
		tradeDir: antv1.TradeDirection_TRADE_DIRECTION_BOTH, strictMode: true,
	}
	if run.StrategyCode != nil {
		p.code = *run.StrategyCode
	}
	if p.code == "" {
		return p, fmt.Errorf("strategy code is empty")
	}
	if run.InitialCapital != nil {
		p.initialCapital = run.InitialCapital.String()
	}
	if run.Commission != nil {
		p.commission = run.Commission.String()
	}
	if run.Slippage != nil {
		p.slippage = run.Slippage.String()
	}
	if run.Leverage != nil && run.Leverage.GreaterThan(decimal.Zero) {
		p.leverage = run.Leverage.String()
	}
	if run.TradeDirection != nil {
		p.tradeDir = stringToTradeDirection(*run.TradeDirection)
	}
	if run.StrictMode != nil {
		p.strictMode = *run.StrictMode
	}
	if len(run.ConfigSnapshot) > 0 {
		var ec antv1.BacktestExecutionConfig
		opts := proto.UnmarshalOptions{DiscardUnknown: true}
		if err := opts.Unmarshal(run.ConfigSnapshot, &ec); err == nil {
			p.strategyCfg = ec.GetStrategyConfig()
		}
	}
	return p, nil
}

func (s *StrategyExecutionServer) executeBacktestRun(ctx context.Context, run *repository.BacktestRun, leaseFor time.Duration) {
	defer func() {
		if r := recover(); r != nil {
			s.log.Error("backtest worker: panic recovered",
				zap.String("runID", run.ID.String()), zap.Any("panic", r))
			s.failRun(ctx, run, fmt.Sprintf("internal panic: %v", r))
		}
	}()

	params, err := extractBacktestParams(run)
	if err != nil {
		s.failRun(ctx, run, err.Error())
		return
	}
	execCtx, execCancel := s.startBacktestWatchers(ctx, run, leaseFor)
	defer execCancel()

	klines, err := s.fetchBars(ctx, run)
	if err != nil {
		s.failRun(ctx, run, fmt.Sprintf("fetch bars: %v", err))
		return
	}
	if len(klines) == 0 {
		s.failRun(ctx, run, "no K-line data available for the specified symbol/timeframe/range")
		return
	}

	// Go-native backtest engine execution.
	result, err := s.executeGoBacktest(execCtx, run, params, klines)
	if err != nil {
		s.handleBacktestError(ctx, run, execCtx, err)
		return
	}
	s.saveBacktestResult(ctx, run, result)
}

// executeGoBacktest runs a backtest using the Go-native engine.
// MQL source → VMRunner (in-process Bytecode VM) + backtest.Engine.
// Generated Go strategy → GoExecutor (subprocess go run).
func (s *StrategyExecutionServer) executeGoBacktest(ctx context.Context, run *repository.BacktestRun, params backtestParams, klines []*antv1.ExecuteKlineBar) (*antv1.ExecuteBacktestResponse, error) {
	// MQL path: in-process Bytecode VM execution.
	if isMQLStrategy(params.code) {
		return s.executeVMBacktest(ctx, params, klines, run)
	}

		// GoExecutor removed (Gap 3). Go strategies must be converted to MQL.
	return nil, fmt.Errorf("Go strategy backtest has been retired — please convert your strategy to MQL")
}

// StartBacktestWorker launches a pool of background workers that poll for PENDING backtest
// runs and execute them via the Go-native backtest engine. Call this once during server startup.
// Defaults to 3 concurrent workers; each claims a run via SKIP LOCKED.
func (s *StrategyExecutionServer) StartBacktestWorker(ctx context.Context) {
	s.log.Info("backtest worker: starting pool", zap.Int("workers", defaultWorkers))

	// Push-first: shared LISTEN for cancel notifications (single connection).
	go s.startCancelListener(ctx)

	for i := 0; i < defaultWorkers; i++ {
		go s.backtestWorker(ctx, i)
	}
}

// startCancelListener subscribes to backtest_cancel PG notifications and
// cancels the corresponding active run's context. Single shared listener
// for all workers — no per-run LISTEN needed.
func (s *StrategyExecutionServer) startCancelListener(ctx context.Context) {
	if s.pgListen == nil {
		s.log.Warn("backtest cancel listener: pgListen unavailable, cancel relies on ticker fallback")
		return
	}
	notifCh, listenCancel, _ := s.pgListen.Listen(ctx, "backtest_cancel")
	if listenCancel != nil {
		defer listenCancel()
	}
	for {
		select {
		case <-ctx.Done():
			return
		case runIDStr, ok := <-notifCh:
			if !ok {
				return
			}
			runID, err := uuid.Parse(runIDStr)
			if err != nil {
				continue
			}
			s.activeCancelsMu.Lock()
			cancelFn, exists := s.activeCancels[runID]
			s.activeCancelsMu.Unlock()
			if exists {
				s.log.Info("backtest cancel listener: cancelling run", zap.String("runID", runIDStr))
				cancelFn()
			}
		}
	}
}
