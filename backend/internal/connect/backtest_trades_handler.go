package connect

import (
	"context"

	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
)

// BacktestTradesServer implements ant.v1.BacktestTradesServiceHandler.
type BacktestTradesServer struct{ log *zap.Logger }

var _ antv1c.BacktestTradesServiceHandler = (*BacktestTradesServer)(nil)

func NewBacktestTradesServer(log *zap.Logger) *BacktestTradesServer {
	return &BacktestTradesServer{log: log}
}

func (s *BacktestTradesServer) ListBacktestRunTrades(ctx context.Context, req *connect.Request[antv1.ListBacktestRunTradesRequest]) (*connect.Response[antv1.ListBacktestRunTradesResponse], error) {
	return connect.NewResponse(&antv1.ListBacktestRunTradesResponse{
		Trades:  []*antv1.BacktestTrade{},
		Summary: &antv1.BacktestTradeSummary{Count: 0, Wins: 0, Losses: 0},
	}), nil
}
