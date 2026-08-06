package strategy

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/repository"
)

// persistBacktestTrades converts proto trades to DB rows and writes them via batch insert.
// Trade persistence is best-effort: failure is logged but does not fail the run,
// because the authoritative trade data lives in ProtoResponse.
func (s *StrategyExecutionServer) persistBacktestTrades(ctx context.Context, runID uuid.UUID, trades []*antv1.ExecuteBacktestTrade) {
	if len(trades) == 0 {
		return
	}
	dbTrades := make([]*repository.BacktestRunTrade, 0, len(trades))
	for _, t := range trades {
		dbTrades = append(dbTrades, &repository.BacktestRunTrade{
			RunID:      runID,
			Ticket:     t.GetTicket(),
			Side:       t.GetSide(),
			Volume:     parseDecimal(t.GetVolume()),
			OpenTs:     t.GetOpenTsMs(),
			OpenPrice:  parseDecimal(t.GetOpenPrice()),
			CloseTs:    t.GetCloseTsMs(),
			ClosePrice: parseDecimal(t.GetClosePrice()),
			PnL:        parseDecimal(t.GetPnl()),
			Commission: parseDecimal(t.GetCommission()),
			Reason:     t.GetReason(),
		})
	}
	if err := s.backtestRepo.BatchCreateTrades(ctx, dbTrades); err != nil {
		s.log.Error("backtest worker: persist trades failed",
			zap.String("runID", runID.String()), zap.Error(err))
	}
}

// invariantBlindSpotIDs lists BlindSpot IDs that indicate a defense-line-B
// invariant violation, making the backtest result unreliable.
var invariantBlindSpotIDs = []string{
	"zero_volume_trade",
	"capital_not_conserved",
	"non_positive_price",
	"invalid_side",
	"time_order_violation",
}

// hasInvariantBlindSpot returns true if resp contains any invariant-class BlindSpot.
func hasInvariantBlindSpot(resp *antv1.ExecuteBacktestResponse) bool {
	for _, bs := range resp.GetBlindSpots() {
		for _, id := range invariantBlindSpotIDs {
			if bs.GetId() == id {
				return true
			}
		}
	}
	return false
}

func (s *StrategyExecutionServer) saveBacktestResult(ctx context.Context, run *repository.BacktestRun, result *antv1.ExecuteBacktestResponse) {
	if !result.GetSuccess() {
		s.failRun(ctx, run, result.GetError())
		return
	}
	protoResp, err := proto.Marshal(result)
	if err != nil {
		s.failRun(ctx, run, fmt.Sprintf("proto marshal failed: %v", err))
		return
	}

	// Build server-generated BacktestSnapshot from actual backtest metrics.
	// This is the tamper-proof source for marketplace quality gate validation.
	snapshotBytes := buildBacktestSnapshot(run, result)

	status := StatusSucceeded
	if hasInvariantBlindSpot(result) {
		status = StatusDegraded
	}

	now := time.Now()
	BacktestRunsTotal.WithLabelValues(status).Inc()
	if err := s.backtestRepo.UpdateAsyncFields(ctx, run.UserID, run.ID, status, "", &now, &now, protoResp, snapshotBytes); err != nil {
		s.log.Error("backtest worker: UpdateAsyncFields failed", zap.String("runID", run.ID.String()), zap.Error(err))
		return
	}

	s.persistBacktestTrades(ctx, run.ID, result.GetTrades())

	// Sync performance metrics to marketplace_strategies if published.
	s.syncMarketplacePerformance(ctx, run, result)

	s.log.Info("backtest worker: run completed", zap.String("runID", run.ID.String()),
		zap.String("total_return", result.GetMetrics().GetTotalReturn()),
		zap.String("sharpe", result.GetMetrics().GetSharpeRatio()))

	// Emit notification for completed backtest.
	if s.notifSender != nil {
		totalReturn := result.GetMetrics().GetTotalReturn()
		sharpe := result.GetMetrics().GetSharpeRatio()
		data, _ := structpb.NewStruct(map[string]interface{}{
			"run_id":       run.ID.String(),
			"symbol":       run.Symbol,
			"timeframe":    run.Timeframe,
			"total_return": totalReturn,
			"sharpe":       sharpe,
		})
		_, _ = s.notifSender.Send(ctx, run.UserID, "backtest_completed",
			fmt.Sprintf("Backtest Complete: %s %s", run.Symbol, run.Timeframe),
			fmt.Sprintf("Strategy on %s %s: return %s%%, Sharpe %s", run.Symbol, run.Timeframe, totalReturn, sharpe),
			data)
	}

	// Trigger auto-gate evaluation after backtest completes.
	if s.onBacktestComplete != nil {
		go s.onBacktestComplete(context.WithoutCancel(ctx), run)
	}
}

func (s *StrategyExecutionServer) failRun(ctx context.Context, run *repository.BacktestRun, errMsg string) {
	now := time.Now()
	BacktestRunsTotal.WithLabelValues(StatusFailed).Inc()
	status := StatusFailed
	if err := s.backtestRepo.UpdateAsyncFields(ctx, run.UserID, run.ID, status, errMsg, nil, &now, nil, nil); err != nil {
		s.log.Error("backtest worker: failRun UpdateAsyncFields",
			zap.String("run_id", run.ID.String()), zap.Error(err))
	}

	// Emit notification for failed backtest.
	if s.notifSender != nil {
		data, _ := structpb.NewStruct(map[string]interface{}{
			"run_id":    run.ID.String(),
			"symbol":    run.Symbol,
			"timeframe": run.Timeframe,
			"error":     errMsg,
		})
		_, _ = s.notifSender.Send(ctx, run.UserID, "backtest_failed",
			fmt.Sprintf("Backtest Failed: %s %s", run.Symbol, run.Timeframe),
			errMsg,
			data)
	}
}

// syncMarketplacePerformance updates marketplace_strategies with the latest backtest
// metrics when the run is associated with a published strategy template.
func (s *StrategyExecutionServer) syncMarketplacePerformance(ctx context.Context, run *repository.BacktestRun, result *antv1.ExecuteBacktestResponse) {
	if run.TemplateID == nil {
		return
	}
	m := result.GetMetrics()
	// Use total_pnl_absolute (correct absolute PnL), fall back to total_return percentage.
	pnlStr := m.GetTotalPnlAbsolute()
	if pnlStr == "" {
		pnlStr = m.GetTotalReturn()
	}
	pnl, err := decimal.NewFromString(pnlStr)
	if err != nil {
		pnl = decimal.Zero
	}
	winRate, err := decimal.NewFromString(m.GetWinRate())
	if err != nil {
		winRate = decimal.Zero
	}
	_, err = s.backtestRepo.DB().Exec(ctx,
		`UPDATE marketplace_strategies SET win_rate = $1, total_pnl = $2, updated_at = now()
		 WHERE strategy_id = $3`,
		winRate.String(), pnl.String(), *run.TemplateID,
	)
	if err != nil {
		s.log.Debug("marketplace sync: template not published or update failed",
			zap.String("template_id", run.TemplateID.String()), zap.Error(err))
	}
}

// buildBacktestSnapshot constructs a tamper-proof BacktestSnapshot proto from
// the actual backtest run metrics. Called on success, stored in backtest_runs.backtest_snapshot.
// The marketplace quality gate reads this snapshot instead of trusting client-supplied data.
func buildBacktestSnapshot(run *repository.BacktestRun, result *antv1.ExecuteBacktestResponse) []byte {
	m := result.GetMetrics()
	snap := &antv1.BacktestSnapshot{
		TotalReturn:  m.GetTotalReturn(),
		AnnualReturn: m.GetAnnualReturn(),
		MaxDrawdown:  m.GetMaxDrawdown(),
		SharpeRatio:  m.GetSharpeRatio(),
		WinRate:      m.GetWinRate(),
		TotalTrades:  m.GetTotalTrades(),
		Symbol:       run.Symbol,
		Timeframe:    run.Timeframe,
	}
	b, err := proto.Marshal(snap)
	if err != nil {
		return nil
	}
	return b
}
