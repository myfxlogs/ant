package strategy

import (
	"context"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/service"
)

func (s *StrategyServer) RunBacktest(
	ctx context.Context, req *connect.Request[antv1.RunBacktestRequest],
) (*connect.Response[antv1.RunBacktestResponse], error) {
	if req.Msg.TemplateId == "" {
		return connect.NewResponse(&antv1.RunBacktestResponse{
			Success:   false,
			RiskLevel: "unknown",
		}), nil
	}
	// Synchronous backtest is deprecated — use StartBacktestRun for async execution
	// via the Go-native backtest engine.
	return connect.NewResponse(&antv1.RunBacktestResponse{
		Success:   false,
		RiskLevel: "unknown",
	}), nil
}

// --- Signals ---

func (s *StrategyServer) ListSignals(
	ctx context.Context, req *connect.Request[antv1.ListSignalsRequest],
) (*connect.Response[antv1.ListSignalsResponse], error) {
	m := req.Msg
	accountID, _ := uuid.Parse(m.AccountId)
	uid := s.userID(ctx)
	rows, err := s.svc.ListSignals(ctx, uid, accountID, m.Status)
	if err != nil {
		return nil, err
	}
	signals := make([]*antv1.StrategySignal, len(rows))
	for i, r := range rows {
		signals[i] = signalRowToProto(&r)
	}
	return connect.NewResponse(&antv1.ListSignalsResponse{Signals: signals}), nil
}

func (s *StrategyServer) ExecuteSignal(
	ctx context.Context, req *connect.Request[antv1.ExecuteSignalRequest],
) (*connect.Response[antv1.ExecuteSignalResponse], error) {
	id, err := uuid.Parse(req.Msg.SignalId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	row, err := s.svc.ExecuteSignal(ctx, id, s.userID(ctx))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &antv1.ExecuteSignalResponse{
		Ticket: row.Ticket, Symbol: row.Symbol,
		Type: row.SignalType, Volume: row.Volume.String(), Price: row.Price.String(),
	}
	if row.ExecutedAt != nil {
		resp.ExecutedAt = timestamppb.New(*row.ExecutedAt)
	}
	return connect.NewResponse(resp), nil
}

func (s *StrategyServer) ConfirmSignal(
	ctx context.Context, req *connect.Request[antv1.ConfirmSignalRequest],
) (*connect.Response[emptypb.Empty], error) {
	id, err := uuid.Parse(req.Msg.SignalId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.svc.ConfirmSignal(ctx, id, s.userID(ctx)); err != nil {
		return nil, err
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *StrategyServer) CancelSignal(
	ctx context.Context, req *connect.Request[antv1.CancelSignalRequest],
) (*connect.Response[emptypb.Empty], error) {
	id, err := uuid.Parse(req.Msg.SignalId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.svc.CancelSignal(ctx, id, s.userID(ctx)); err != nil {
		return nil, err
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// signalRowToProto converts a service SignalRow to a proto StrategySignal.
func signalRowToProto(r *service.SignalRow) *antv1.StrategySignal {
	if r == nil {
		return nil
	}
	pb := &antv1.StrategySignal{
		Id:             r.ID.String(),
		AccountId:      r.AccountID.String(),
		Symbol:         r.Symbol,
		SignalType:     r.SignalType,
		Volume:         r.Volume.String(),
		Price:          r.Price.String(),
		StopLoss:       r.StopLoss.String(),
		TakeProfit:     r.TakeProfit.String(),
		Reason:         r.Reason,
		Status:         r.Status,
		ExecutedTicket: r.Ticket,
		CreatedAt:      timestamppb.New(r.CreatedAt),
	}
	if r.ExecutedAt != nil {
		pb.ExecutedAt = timestamppb.New(*r.ExecutedAt)
	}
	return pb
}
