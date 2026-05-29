package strategy

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/repository"
)

// BacktestTradesServer implements ant.v1.BacktestTradesServiceHandler.
type BacktestTradesServer struct {
	backtestRepo *repository.BacktestRunRepository
	log          *zap.Logger
}

var _ antv1c.BacktestTradesServiceHandler = (*BacktestTradesServer)(nil)

func NewBacktestTradesServer(backtestRepo *repository.BacktestRunRepository, log *zap.Logger) *BacktestTradesServer {
	return &BacktestTradesServer{backtestRepo: backtestRepo, log: log}
}

// backtestMetricsJSON mirrors the Python Metrics dataclass fields we care about.
type backtestMetricsJSON struct {
	TotalTrades   int     `json:"total_trades"`
	WinningTrades int     `json:"winning_trades"`
	LosingTrades  int     `json:"losing_trades"`
	TotalReturn   float64 `json:"total_return"`
	ProfitFactor  float64 `json:"profit_factor"`
	NetPnL        float64 `json:"net_pnl"`
}

func (s *BacktestTradesServer) ListBacktestRunTrades(ctx context.Context, req *connect.Request[antv1.ListBacktestRunTradesRequest]) (*connect.Response[antv1.ListBacktestRunTradesResponse], error) {
	runID, err := uuid.Parse(req.Msg.RunId)
	if err != nil {
		return connect.NewResponse(&antv1.ListBacktestRunTradesResponse{
			Trades:  []*antv1.BacktestTrade{},
			Summary: &antv1.BacktestTradeSummary{},
		}), nil
	}

	uid, _ := uuid.Parse(interceptor.GetUserID(ctx))
	run, err := s.backtestRepo.GetByID(ctx, uid, runID)
	if err != nil {
		s.log.Warn("BacktestTrades: get run", zap.Error(err), zap.String("run_id", req.Msg.RunId))
		return connect.NewResponse(&antv1.ListBacktestRunTradesResponse{
			Trades:  []*antv1.BacktestTrade{},
			Summary: &antv1.BacktestTradeSummary{},
		}), nil
	}

	summary := &antv1.BacktestTradeSummary{}
	if len(run.Metrics) > 0 {
		var m backtestMetricsJSON
		if err := json.Unmarshal(run.Metrics, &m); err == nil {
			summary.Count = int32(m.TotalTrades)
			summary.Wins = int32(m.WinningTrades)
			summary.Losses = int32(m.LosingTrades)
			summary.NetPnl = m.NetPnL
		}
	}

	// Read per-trade data from backtest_run_trades table (if any).
	dbTrades, err := s.backtestRepo.ListTradesByRunID(ctx, runID)
	if err != nil {
		s.log.Warn("BacktestTrades: list trades", zap.Error(err), zap.String("run_id", req.Msg.RunId))
	}

	trades := make([]*antv1.BacktestTrade, 0, len(dbTrades))
	for _, t := range dbTrades {
		trades = append(trades, &antv1.BacktestTrade{
			Ticket:     t.Ticket,
			Side:       t.Side,
			Volume:     t.Volume,
			OpenTs:     t.OpenTs,
			OpenPrice:  t.OpenPrice,
			CloseTs:    t.CloseTs,
			ClosePrice: t.ClosePrice,
			Pnl:        t.PnL,
			Commission: t.Commission,
			Reason:     t.Reason,
		})
	}

	return connect.NewResponse(&antv1.ListBacktestRunTradesResponse{
		Trades:  trades,
		Summary: summary,
	}), nil
}
