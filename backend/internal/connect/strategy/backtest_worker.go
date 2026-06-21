package strategy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"connectrpc.com/connect"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/repository"
)

const (
	listenTimeout  = 30 * time.Second // fallback poll interval when no PG notification received
	leaseFor       = 60 * time.Second
	defaultWorkers = 3
)

// backtestWorker waits for new PENDING backtest runs via PG LISTEN/NOTIFY,
// with a 30s fallback ticker for safety (Push-First pattern). This replaces
// the previous 3-second polling, which violated the project's push-first rule.
func (s *PythonStrategyServer) backtestWorker(ctx context.Context, workerID int) {
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

// fetchBars retrieves K-line data via BarSource when available,
// falling back to direct ClickHouse fetch for backward compatibility.
func (s *PythonStrategyServer) fetchBars(ctx context.Context, run *repository.BacktestRun) []*antv1.ExecuteKlineBar {
	if s.barSource != nil {
		klines, _ := s.barSource.Fetch(ctx, run.Symbol, run.Timeframe, run.FromTs, run.ToTs)
		return klines
	}
	return s.fetchBacktestKlines(ctx, run)
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
	if s.backtestClient == nil { s.failRun(ctx, run, "Python strategy service not available"); return }
	params, err := extractBacktestParams(run)
	if err != nil { s.failRun(ctx, run, err.Error()); return }
	execCtx := s.startBacktestWatchers(ctx, run, leaseFor)
	defer func() { _ = execCtx }()

	req := buildBacktestRequest(run, params, s.fetchBars(ctx, run))
	resp, err := s.backtestClient.RunBacktest(execCtx, connect.NewRequest(req))
	if err != nil {
		s.handleBacktestError(ctx, run, execCtx, err)
		return
	}
	s.saveBacktestResult(ctx, run, resp.Msg)
}

func buildBacktestRequest(run *repository.BacktestRun, params backtestParams, klines []*antv1.ExecuteKlineBar) *antv1.ExecuteBacktestRequest {
	fromMs, toMs := int64(0), int64(0)
	if run.FromTs != nil { fromMs = run.FromTs.UnixMilli() }
	if run.ToTs != nil { toMs = run.ToTs.UnixMilli() }
	return &antv1.ExecuteBacktestRequest{
		StrategyId: run.ID.String(), StrategyCode: params.code,
		Symbol: run.Symbol, Timeframe: run.Timeframe,
		StartDateMs: fromMs, EndDateMs: toMs,
		InitialCapital: params.initialCapital, Commission: params.commission,
		SlippageRate: params.slippage, SlippageMode: "fixed", SlippageSeed: 42,
		Leverage: params.leverage, TradeDirection: params.tradeDir,
		StrictMode: params.strictMode, StrategyConfig: params.strategyCfg,
		Klines: klines, StrategyParamsJson: paramsProtoToJSON(run.ParameterOverrides),
	}
}

func (s *PythonStrategyServer) handleBacktestError(ctx context.Context, run *repository.BacktestRun, execCtx context.Context, err error) {
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
}

// persistBacktestTrades converts proto trades to DB rows and writes them via batch insert.
// Trade persistence is best-effort: failure is logged but does not fail the run,
// because the authoritative trade data lives in ProtoResponse.
func (s *PythonStrategyServer) persistBacktestTrades(ctx context.Context, runID uuid.UUID, trades []*antv1.ExecuteBacktestTrade) {
	if len(trades) == 0 {
		return
	}
	dbTrades := make([]*repository.BacktestRunTrade, 0, len(trades))
	for _, t := range trades {
		dbTrades = append(dbTrades, &repository.BacktestRunTrade{
			RunID:      runID,
			Ticket:     t.GetTicket(),
			Side:       t.GetSide(),
			Volume:     t.GetVolume(),
			OpenTs:     t.GetOpenTsMs(),
			OpenPrice:  t.GetOpenPrice(),
			CloseTs:    t.GetCloseTsMs(),
			ClosePrice: t.GetClosePrice(),
			PnL:        t.GetPnl(),
			Commission: t.GetCommission(),
			Reason:     t.GetReason(),
		})
	}
	if err := s.backtestRepo.BatchCreateTrades(ctx, dbTrades); err != nil {
		s.log.Error("backtest worker: persist trades failed",
			zap.String("runID", runID.String()), zap.Error(err))
	}
}

func (s *PythonStrategyServer) saveBacktestResult(ctx context.Context, run *repository.BacktestRun, result *antv1.ExecuteBacktestResponse) {
	if !result.GetSuccess() { s.failRun(ctx, run, result.GetError()); return }
	protoResp, err := proto.Marshal(result)
	if err != nil { s.failRun(ctx, run, fmt.Sprintf("proto marshal failed: %v", err)); return }
	now := time.Now()
	BacktestRunsTotal.WithLabelValues(StatusSucceeded).Inc()
	if err := s.backtestRepo.UpdateAsyncFields(ctx, run.UserID, run.ID, StatusSucceeded, "", &now, &now, protoResp); err != nil {
		s.log.Error("backtest worker: UpdateAsyncFields failed", zap.String("runID", run.ID.String()), zap.Error(err))
		return
	}

	s.persistBacktestTrades(ctx, run.ID, result.GetTrades())

	// Sync performance metrics to marketplace_strategies if published.
	s.syncMarketplacePerformance(ctx, run, result)

	s.log.Info("backtest worker: run completed", zap.String("runID", run.ID.String()),
		zap.Float64("total_return", result.GetMetrics().GetTotalReturn()),
		zap.Float64("sharpe", result.GetMetrics().GetSharpeRatio()))

	// Emit notification for completed backtest.
	if s.notifSender != nil {
		totalReturn := result.GetMetrics().GetTotalReturn()
		sharpe := result.GetMetrics().GetSharpeRatio()
		data, _ := json.Marshal(map[string]interface{}{
			"run_id":       run.ID.String(),
			"symbol":       run.Symbol,
			"timeframe":    run.Timeframe,
			"total_return": totalReturn,
			"sharpe":       sharpe,
		})
		_, _ = s.notifSender.Send(ctx, run.UserID, "backtest_completed",
			fmt.Sprintf("Backtest Complete: %s %s", run.Symbol, run.Timeframe),
			fmt.Sprintf("Strategy on %s %s: return %.2f%%, Sharpe %.2f", run.Symbol, run.Timeframe, totalReturn, sharpe),
			string(data))
	}

	// Trigger auto-gate evaluation after backtest completes.
	if s.onBacktestComplete != nil {
		go s.onBacktestComplete(context.Background(), run)
	}
}

func (s *PythonStrategyServer) failRun(ctx context.Context, run *repository.BacktestRun, errMsg string) {
	now := time.Now()
	BacktestRunsTotal.WithLabelValues(StatusFailed).Inc()
	status := StatusFailed
	if err := s.backtestRepo.UpdateAsyncFields(ctx, run.UserID, run.ID, status, errMsg, nil, &now, nil); err != nil {
		s.log.Error("backtest worker: failRun UpdateAsyncFields",
			zap.String("run_id", run.ID.String()), zap.Error(err))
	}

	// Emit notification for failed backtest.
	if s.notifSender != nil {
		data, _ := json.Marshal(map[string]interface{}{
			"run_id":    run.ID.String(),
			"symbol":    run.Symbol,
			"timeframe": run.Timeframe,
			"error":     errMsg,
		})
		_, _ = s.notifSender.Send(ctx, run.UserID, "backtest_failed",
			fmt.Sprintf("Backtest Failed: %s %s", run.Symbol, run.Timeframe),
			errMsg,
			string(data))
	}
}

// syncMarketplacePerformance updates marketplace_strategies with the latest backtest
// metrics when the run is associated with a published strategy template.
func (s *PythonStrategyServer) syncMarketplacePerformance(ctx context.Context, run *repository.BacktestRun, result *antv1.ExecuteBacktestResponse) {
	if run.TemplateID == nil {
		return
	}
	m := result.GetMetrics()
	// Use total_pnl_absolute (correct absolute PnL), fall back to total_return percentage.
	pnl := m.GetTotalPnlAbsolute()
	if pnl == 0 {
		pnl = m.GetTotalReturn() // legacy: percentage as proxy
	}
	_, err := s.backtestRepo.DB().Exec(ctx,
		`UPDATE marketplace_strategies SET win_rate = $1, total_pnl = $2, updated_at = now()
		 WHERE strategy_id = $3`,
		m.GetWinRate(), pnl, *run.TemplateID,
	)
	if err != nil {
		s.log.Debug("marketplace sync: template not published or update failed",
			zap.String("template_id", run.TemplateID.String()), zap.Error(err))
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

