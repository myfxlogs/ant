package strategy

import (
	"context"
	"errors"
	"math"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/model"
	"alphaforge/internal/pglisten"
	"alphaforge/internal/repository"
)

// DivergenceServer implements LiveBacktestDivergenceService.
// It compares backtest run results with live trade records to verify
// that backtest predictions align with live performance (FEAT-4).
type DivergenceServer struct {
	backtestRepo *repository.BacktestRunRepository
	tradeRepo    *repository.TradeRecordRepository
	pgListen     *pglisten.Listener
	log          *zap.Logger
}

var _ antv1c.LiveBacktestDivergenceServiceHandler = (*DivergenceServer)(nil)

func NewDivergenceServer(br *repository.BacktestRunRepository, tr *repository.TradeRecordRepository, log *zap.Logger) *DivergenceServer {
	return &DivergenceServer{backtestRepo: br, tradeRepo: tr, log: log}
}

func (s *DivergenceServer) SetPgListen(l *pglisten.Listener) { s.pgListen = l }

// GetDivergenceReport returns a one-shot comparison between the latest
// backtest run and live trade records for a given strategy.
func (s *DivergenceServer) GetDivergenceReport(
	ctx context.Context,
	req *connect.Request[antv1.GetDivergenceReportRequest],
) (*connect.Response[antv1.GetDivergenceReportResponse], error) {
	strategyID, err := uuid.Parse(req.Msg.StrategyId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	report, err := s.computeReport(ctx, strategyID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.GetDivergenceReportResponse{Report: report}), nil
}

// WatchDivergenceReport streams divergence updates whenever new live
// trade records are synced (push-first via pg_notify on trade_record_sync).
func (s *DivergenceServer) WatchDivergenceReport(
	ctx context.Context,
	req *connect.Request[antv1.WatchDivergenceReportRequest],
	stream *connect.ServerStream[antv1.DivergenceUpdate],
) error {
	strategyID, err := uuid.Parse(req.Msg.StrategyId)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Send initial report immediately.
	report, err := s.computeReport(ctx, strategyID)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	if err := stream.Send(&antv1.DivergenceUpdate{Report: report}); err != nil {
		return err
	}

	// Push-first: listen for trade_record_sync notifications.
	notifCh, listenCancel, _ := s.pgListen.Listen(ctx, "trade_record_sync")
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
			s.log.Warn("WatchDivergenceReport: compute failed", zap.Error(err))
			continue
		}
		if err := stream.Send(&antv1.DivergenceUpdate{Report: report}); err != nil {
			return err
		}
	}
}

// computeReport builds a DivergenceReport by fetching the latest completed
// backtest run and live trade records for the given strategy.
func (s *DivergenceServer) computeReport(ctx context.Context, strategyID uuid.UUID) (*antv1.DivergenceReport, error) {
	btRun, err := s.backtestRepo.GetLatestCompletedByStrategyID(ctx, strategyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &antv1.DivergenceReport{
				StrategyId: strategyID.String(),
				Status:     antv1.DivergenceStatus_DIVERGENCE_STATUS_INSUFFICIENT_DATA,
				Detail:     "no completed backtest run found for this strategy",
			}, nil
		}
		return nil, err
	}

	liveRecords, err := s.tradeRepo.GetByStrategyID(ctx, strategyID)
	if err != nil {
		return nil, err
	}

	btTrades, err := s.backtestRepo.ListTradesByRunID(ctx, btRun.ID)
	if err != nil {
		return nil, err
	}

	btMetrics := computeBacktestMetrics(btTrades)
	liveMetrics := computeLiveMetrics(liveRecords)

	report := &antv1.DivergenceReport{
		StrategyId:      strategyID.String(),
		BacktestRunId:   btRun.ID.String(),
		BacktestMetrics: btMetrics,
		LiveMetrics:     liveMetrics,
	}

	// Compute divergence scores.
	report.PnlDivergencePct = pctDivergence(btMetrics.NetPnl, liveMetrics.NetPnl)
	report.TradeCountDivergencePct = countDivergencePct(btMetrics.TradeCount, liveMetrics.TradeCount)
	report.WinRateDivergencePct = math.Abs(btMetrics.WinRate-liveMetrics.WinRate) * 100.0
	report.SharpeDivergence = math.Abs(btMetrics.SharpeRatio - liveMetrics.SharpeRatio)

	report.Status, report.Detail = assessDivergence(report, btMetrics, liveMetrics)
	return report, nil
}

// computeBacktestMetrics derives DivergenceMetrics from backtest_run_trades.
func computeBacktestMetrics(trades []*repository.BacktestRunTrade) *antv1.DivergenceMetrics {
	m := &antv1.DivergenceMetrics{TradeCount: int32(len(trades))}
	if len(trades) == 0 {
		return m
	}

	totalPnL := decimal.Zero
	grossProfit := decimal.Zero
	grossLoss := decimal.Zero
	wins, losses := 0, 0
	pnls := make([]float64, 0, len(trades))
	var minTs, maxTs int64

	for i, t := range trades {
		totalPnL = totalPnL.Add(t.PnL)
		if t.PnL.GreaterThan(decimal.Zero) {
			grossProfit = grossProfit.Add(t.PnL)
			wins++
		} else if t.PnL.LessThan(decimal.Zero) {
			grossLoss = grossLoss.Add(t.PnL)
			losses++
		}
		pnls = append(pnls, t.PnL.InexactFloat64())
		if i == 0 || t.OpenTs < minTs {
			minTs = t.OpenTs
		}
		if t.CloseTs > maxTs {
			maxTs = t.CloseTs
		}
	}

	m.Wins = int32(wins)
	m.Losses = int32(losses)
	m.NetPnl = totalPnL.String()
	m.GrossProfit = grossProfit.String()
	m.GrossLoss = grossLoss.String()
	m.WinRate = float64(wins) / float64(len(trades)) * 100.0
	m.AvgTradePnl = totalPnL.Div(decimal.NewFromInt(int64(len(trades)))).InexactFloat64()
	m.SharpeRatio = computeSharpe(pnls)
	m.PeriodStartMs = minTs
	m.PeriodEndMs = maxTs
	return m
}

// computeLiveMetrics derives DivergenceMetrics from trade_records.
func computeLiveMetrics(records []*model.TradeRecord) *antv1.DivergenceMetrics {
	m := &antv1.DivergenceMetrics{TradeCount: int32(len(records))}
	if len(records) == 0 {
		return m
	}

	totalPnL := decimal.Zero
	grossProfit := decimal.Zero
	grossLoss := decimal.Zero
	wins, losses := 0, 0
	pnls := make([]float64, 0, len(records))
	var minMs, maxMs int64

	for i, r := range records {
		totalPnL = totalPnL.Add(r.Profit)
		if r.Profit.GreaterThan(decimal.Zero) {
			grossProfit = grossProfit.Add(r.Profit)
			wins++
		} else if r.Profit.LessThan(decimal.Zero) {
			grossLoss = grossLoss.Add(r.Profit)
			losses++
		}
		pnls = append(pnls, r.Profit.InexactFloat64())
		ms := r.OpenTime.UnixMilli()
		if i == 0 || ms < minMs {
			minMs = ms
		}
		closeMs := r.CloseTime.UnixMilli()
		if closeMs > maxMs {
			maxMs = closeMs
		}
	}

	m.Wins = int32(wins)
	m.Losses = int32(losses)
	m.NetPnl = totalPnL.String()
	m.GrossProfit = grossProfit.String()
	m.GrossLoss = grossLoss.String()
	m.WinRate = float64(wins) / float64(len(records)) * 100.0
	m.AvgTradePnl = totalPnL.Div(decimal.NewFromInt(int64(len(records)))).InexactFloat64()
	m.SharpeRatio = computeSharpe(pnls)
	m.PeriodStartMs = minMs
	m.PeriodEndMs = maxMs
	return m
}

// computeSharpe calculates a simple Sharpe-like ratio from a slice of per-trade PnLs.
// Returns 0 if insufficient data or zero standard deviation.
func computeSharpe(pnls []float64) float64 {
	n := len(pnls)
	if n < 2 {
		return 0
	}
	var sum float64
	for _, p := range pnls {
		sum += p
	}
	mean := sum / float64(n)
	var sqSum float64
	for _, p := range pnls {
		d := p - mean
		sqSum += d * d
	}
	stdDev := math.Sqrt(sqSum / float64(n))
	if stdDev == 0 {
		return 0
	}
	return mean / stdDev * math.Sqrt(float64(n))
}

// pctDivergence computes the percentage divergence between two decimal string values.
func pctDivergence(a, b string) float64 {
	dA, errA := decimal.NewFromString(a)
	dB, errB := decimal.NewFromString(b)
	if errA != nil || errB != nil {
		return 0
	}
	if dA.IsZero() {
		if dB.IsZero() {
			return 0
		}
		return 100.0
	}
	diff := dA.Sub(dB).Abs()
	return diff.Div(dA.Abs()).InexactFloat64() * 100.0
}

// countDivergencePct computes the percentage divergence between two trade counts.
func countDivergencePct(btCount, liveCount int32) float64 {
	if btCount == 0 {
		if liveCount == 0 {
			return 0
		}
		return 100.0
	}
	diff := math.Abs(float64(btCount - liveCount))
	return diff / float64(btCount) * 100.0
}

// assessDivergence determines the overall divergence status based on divergence scores.
func assessDivergence(report *antv1.DivergenceReport, bt, live *antv1.DivergenceMetrics) (antv1.DivergenceStatus, string) {
	if bt.TradeCount == 0 || live.TradeCount == 0 {
		return antv1.DivergenceStatus_DIVERGENCE_STATUS_INSUFFICIENT_DATA,
			"insufficient data: backtest or live has zero trades"
	}

	// Thresholds: <10% = consistent, 10-30% = minor, >30% = major.
	maxDivergence := report.PnlDivergencePct
	if report.TradeCountDivergencePct > maxDivergence {
		maxDivergence = report.TradeCountDivergencePct
	}
	if report.WinRateDivergencePct > maxDivergence {
		maxDivergence = report.WinRateDivergencePct
	}

	switch {
	case maxDivergence < 10:
		return antv1.DivergenceStatus_DIVERGENCE_STATUS_CONSISTENT,
			"backtest and live performance are consistent"
	case maxDivergence < 30:
		return antv1.DivergenceStatus_DIVERGENCE_STATUS_MINOR_DIVERGENCE,
			"minor divergence detected between backtest and live"
	default:
		return antv1.DivergenceStatus_DIVERGENCE_STATUS_MAJOR_DIVERGENCE,
			"major divergence detected — backtest may not predict live performance"
	}
}

// userID extracts the authenticated user ID from context (for future auth checks).
func (s *DivergenceServer) userID(ctx context.Context) uuid.UUID {
	raw := interceptor.GetUserID(ctx)
	if raw == "" {
		return uuid.Nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil
	}
	return id
}
