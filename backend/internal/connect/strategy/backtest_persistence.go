package strategy

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
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

func (s *StrategyExecutionServer) saveBacktestResult(ctx context.Context, run *repository.BacktestRun, result *antv1.ExecuteBacktestResponse) {
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
		go s.onBacktestComplete(context.Background(), run)
	}
}

func (s *StrategyExecutionServer) failRun(ctx context.Context, run *repository.BacktestRun, errMsg string) {
	now := time.Now()
	BacktestRunsTotal.WithLabelValues(StatusFailed).Inc()
	status := StatusFailed
	if err := s.backtestRepo.UpdateAsyncFields(ctx, run.UserID, run.ID, status, errMsg, nil, &now, nil); err != nil {
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
	pnlF, _ := strconv.ParseFloat(pnlStr, 64)
	if pnlF == 0 {
		pnlF, _ = strconv.ParseFloat(m.GetTotalReturn(), 64) // legacy: percentage as proxy
	}
	winRateF, _ := strconv.ParseFloat(m.GetWinRate(), 64)
	_, err := s.backtestRepo.DB().Exec(ctx,
		`UPDATE marketplace_strategies SET win_rate = $1, total_pnl = $2, updated_at = now()
		 WHERE strategy_id = $3`,
		winRateF, pnlF, *run.TemplateID,
	)
	if err != nil {
		s.log.Debug("marketplace sync: template not published or update failed",
			zap.String("template_id", run.TemplateID.String()), zap.Error(err))
	}
}
