package strategy

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

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

// extractTradeSummary reads trade counts from proto binary ExecuteBacktestResponse.
func extractTradeSummary(protoResp []byte) *antv1.BacktestTradeSummary {
	if len(protoResp) == 0 {
		return &antv1.BacktestTradeSummary{}
	}
	var resp antv1.ExecuteBacktestResponse
	if err := proto.Unmarshal(protoResp, &resp); err != nil {
		return &antv1.BacktestTradeSummary{}
	}
	m := resp.GetMetrics()
	trades := resp.GetTrades()
	var netPnl float64
	for _, t := range trades {
		netPnl += t.GetPnl()
	}
	return &antv1.BacktestTradeSummary{
		Count:  m.GetTotalTrades(),
		Wins:   m.GetWinningTrades(),
		Losses: m.GetLosingTrades(),
		NetPnl: netPnl,
	}
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

	summary := extractTradeSummary(run.ProtoResponse)

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
