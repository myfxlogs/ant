package strategy

import (
	"context"
	"errors"
	"math"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/ai"
	"alphaforge/internal/pglisten"
	"alphaforge/internal/repository"
)

// WalkForwardServer implements WalkForwardService.
// It runs walk-forward validation on completed backtest runs and reports
// IS/OOS degradation metrics (FEAT-4 walk-forward validation).
type WalkForwardServer struct {
	backtestRepo *repository.BacktestRunRepository
	pgListen     *pglisten.Listener
	log          *zap.Logger
}

var _ antv1c.WalkForwardServiceHandler = (*WalkForwardServer)(nil)

func NewWalkForwardServer(br *repository.BacktestRunRepository, log *zap.Logger) *WalkForwardServer {
	return &WalkForwardServer{backtestRepo: br, log: log}
}

func (s *WalkForwardServer) SetPgListen(l *pglisten.Listener) { s.pgListen = l }

// GetWalkForwardReport returns a one-shot walk-forward report for a strategy.
func (s *WalkForwardServer) GetWalkForwardReport(
	ctx context.Context,
	req *connect.Request[antv1.GetWalkForwardReportRequest],
) (*connect.Response[antv1.WalkForwardReport], error) {
	strategyID, err := uuid.Parse(req.Msg.StrategyId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	report, err := s.computeReport(ctx, strategyID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(report), nil
}

// WatchWalkForwardReport streams walk-forward report updates via SSE.
// Push-first: triggered by backtest_status pg_notify when a backtest completes.
func (s *WalkForwardServer) WatchWalkForwardReport(
	ctx context.Context,
	req *connect.Request[antv1.WatchWalkForwardReportRequest],
	stream *connect.ServerStream[antv1.WalkForwardReport],
) error {
	strategyID, err := uuid.Parse(req.Msg.StrategyId)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}

	report, err := s.computeReport(ctx, strategyID)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	if err := stream.Send(report); err != nil {
		return err
	}

	notifCh, listenCancel, _ := s.pgListen.Listen(ctx, "backtest_status")
	if listenCancel != nil {
		defer listenCancel()
	}
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-notifCh:
		case <-ticker.C:
		}

		report, err := s.computeReport(ctx, strategyID)
		if err != nil {
			s.log.Warn("WatchWalkForwardReport: compute failed", zap.Error(err))
			continue
		}
		if err := stream.Send(report); err != nil {
			return err
		}
	}
}

// computeReport builds a WalkForwardReport by fetching the latest completed
// backtest run for the given strategy and running walk-forward validation.
func (s *WalkForwardServer) computeReport(ctx context.Context, strategyID uuid.UUID) (*antv1.WalkForwardReport, error) {
	btRun, err := s.backtestRepo.GetLatestCompletedByStrategyID(ctx, strategyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &antv1.WalkForwardReport{
				StrategyId: strategyID.String(),
				Status:     antv1.WalkForwardStatus_WALK_FORWARD_STATUS_INSUFFICIENT_DATA,
				Reason:     "no completed backtest run found for this strategy",
			}, nil
		}
		return nil, err
	}

	dailyReturns := ai.EquityCurveToDailyReturns(btRun.ProtoResponse)
	if len(dailyReturns) < 60 {
		return &antv1.WalkForwardReport{
			StrategyId:    strategyID.String(),
			BacktestRunId: btRun.ID.String(),
			Status:        antv1.WalkForwardStatus_WALK_FORWARD_STATUS_INSUFFICIENT_DATA,
			Reason:        "insufficient daily returns for walk-forward validation",
		}, nil
	}

	cfg := ai.DefaultWalkForwardConfig()
	wfResult := ai.WalkForward(dailyReturns, cfg)
	cpcvSharpe := ai.CPCV(dailyReturns, 6, cfg)

	btTrades, err := s.backtestRepo.ListTradesByRunID(ctx, btRun.ID)
	if err != nil {
		return nil, err
	}

	isMetrics, oosMetrics := splitTradesMetrics(btTrades, dailyReturns)

	sharpeDeg := isMetrics.SharpeRatio - oosMetrics.SharpeRatio
	degradationRatio := 0.0
	if isMetrics.SharpeRatio > 0 {
		degradationRatio = oosMetrics.SharpeRatio / isMetrics.SharpeRatio
	}

	status := antv1.WalkForwardStatus_WALK_FORWARD_STATUS_PASS
	reason := ""
	if !wfResult.Passed {
		status = antv1.WalkForwardStatus_WALK_FORWARD_STATUS_FAIL
		reason = wfResult.Reason
	}

	folds := make([]*antv1.WalkForwardFold, len(wfResult.Folds))
	for i, f := range wfResult.Folds {
		folds[i] = &antv1.WalkForwardFold{
			FoldIndex:       int32(f.FoldIndex),
			TrainSharpe:     f.TrainSharpe,
			TestSharpe:      f.TestSharpe,
			TrainMaxDd:      f.TrainMaxDD,
			TestMaxDd:       f.TestMaxDD,
			TradeCount:      int32(f.TradeCount),
			Passed:          f.Passed,
			RejectionReason: f.RejectionReason,
		}
	}

	return &antv1.WalkForwardReport{
		StrategyId:        strategyID.String(),
		BacktestRunId:     btRun.ID.String(),
		IsMetrics:         isMetrics,
		OosMetrics:        oosMetrics,
		SharpeDegradation: sharpeDeg,
		DegradationRatio:  degradationRatio,
		Status:            status,
		Reason:            reason,
		Folds:             folds,
		CpcvMedianSharpe:  cpcvSharpe,
		ComputedAtMs:      time.Now().UnixMilli(),
	}, nil
}

// splitTradesMetrics computes IS (70%) and OOS (30%) metrics from backtest trades.
// Trades are split by chronological order — first 70% are in-sample, last 30% are out-of-sample.
func splitTradesMetrics(trades []*repository.BacktestRunTrade, dailyReturns []float64) (*antv1.WalkForwardMetrics, *antv1.WalkForwardMetrics) {
	if len(trades) == 0 {
		return &antv1.WalkForwardMetrics{}, &antv1.WalkForwardMetrics{}
	}

	splitIdx := int(float64(len(trades)) * 0.7)
	if splitIdx < 1 {
		splitIdx = 1
	}
	if splitIdx >= len(trades) {
		splitIdx = len(trades) - 1
	}

	isTrades := trades[:splitIdx]
	oosTrades := trades[splitIdx:]

	isMetrics := computeWalkForwardSegmentMetrics(isTrades, dailyReturns[:len(dailyReturns)*7/10])
	oosMetrics := computeWalkForwardSegmentMetrics(oosTrades, dailyReturns[len(dailyReturns)*7/10:])

	return isMetrics, oosMetrics
}

func computeWalkForwardSegmentMetrics(trades []*repository.BacktestRunTrade, returns []float64) *antv1.WalkForwardMetrics {
	if len(trades) == 0 {
		return &antv1.WalkForwardMetrics{}
	}

	var totalPnL, wins float64
	var minTs, maxTs int64
	for i, t := range trades {
		pnl, _ := t.PnL.Float64()
		totalPnL += pnl
		if pnl > 0 {
			wins++
		}
		if i == 0 {
			minTs = t.OpenTs
			maxTs = t.CloseTs
		} else {
			if t.OpenTs < minTs {
				minTs = t.OpenTs
			}
			if t.CloseTs > maxTs {
				maxTs = t.CloseTs
			}
		}
	}

	sharpe := 0.0
	maxDD := 0.0
	if len(returns) >= 2 {
		sharpe = computeSharpeRatio(returns)
		maxDD = computeMaxDrawdown(returns)
	}

	return &antv1.WalkForwardMetrics{
		TradeCount:    int32(len(trades)),
		NetPnl:        totalPnL,
		WinRate:       wins / float64(len(trades)) * 100.0,
		SharpeRatio:   sharpe,
		MaxDrawdown:   maxDD,
		AvgTradePnl:   totalPnL / float64(len(trades)),
		PeriodStartMs: minTs,
		PeriodEndMs:   maxTs,
	}
}

func computeSharpeRatio(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}
	var sum, sumSq float64
	for _, r := range returns {
		sum += r
		sumSq += r * r
	}
	n := float64(len(returns))
	mean := sum / n
	variance := sumSq/n - mean*mean
	if variance <= 0 {
		return 0
	}
	stdDev := math.Sqrt(variance)
	if stdDev == 0 {
		return 0
	}
	return (mean / stdDev) * math.Sqrt(252)
}

func computeMaxDrawdown(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}
	peak := 0.0
	maxDD := 0.0
	cumulative := 0.0
	for _, r := range returns {
		cumulative += r
		if cumulative > peak {
			peak = cumulative
		}
		if peak > 0 {
			dd := (peak - cumulative) / peak
			if dd > maxDD {
				maxDD = dd
			}
		}
	}
	if peak == 0 {
		return 1.0
	}
	return maxDD
}
