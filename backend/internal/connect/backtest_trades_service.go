package connect

import (
	"context"
	"encoding/json"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	v1 "anttrader/gen/proto"
	"anttrader/internal/service"
)

type BacktestTradesService struct {
	backtestRun *service.BacktestRunService
}

func NewBacktestTradesService(backtestRun *service.BacktestRunService) *BacktestTradesService {
	return &BacktestTradesService{backtestRun: backtestRun}
}

func (s *BacktestTradesService) ListBacktestRunTrades(ctx context.Context, req *connect.Request[v1.ListBacktestRunTradesRequest]) (*connect.Response[v1.ListBacktestRunTradesResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	runID, err := uuid.Parse(req.Msg.RunId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	run, err := s.backtestRun.Get(ctx, userID, runID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	var wrapper struct {
		Trades []service.BacktestTradePython `json:"trades"`
	}
	if len(run.Metrics) > 0 {
		_ = json.Unmarshal(run.Metrics, &wrapper)
	}
	out := make([]*v1.BacktestTrade, 0, len(wrapper.Trades))
	summary := &v1.BacktestTradeSummary{Count: int32(len(wrapper.Trades))}
	for _, t := range wrapper.Trades {
		out = append(out, &v1.BacktestTrade{Ticket: t.Ticket, Side: t.Side, Volume: t.Volume, OpenTs: t.OpenTs, OpenPrice: t.OpenPrice, CloseTs: t.CloseTs, ClosePrice: t.ClosePrice, Pnl: t.PnL, Commission: t.Commission, Reason: t.Reason})
		if t.PnL > 0 {
			summary.Wins++
		} else if t.PnL < 0 {
			summary.Losses++
		}
		summary.NetPnl += t.PnL
	}
	return connect.NewResponse(&v1.ListBacktestRunTradesResponse{Trades: out, Summary: summary}), nil
}
