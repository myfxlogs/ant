package strategy

import (
	"context"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/interceptor"
	"anttrader/internal/strategysvc"
)

func (s *StrategyServer) RunBacktest(ctx context.Context, req *connect.Request[antv1.RunBacktestRequest]) (*connect.Response[antv1.RunBacktestResponse], error) {
	m := req.Msg
	if s.client != nil && m.TemplateId != "" {
		tid, err := uuid.Parse(m.TemplateId)
		if err == nil {
			userID, _ := uuid.Parse(interceptor.GetUserID(ctx))
			tmpl, err := s.svc.GetTemplate(ctx, tid, userID)
			if err != nil {
				s.log.Warn("RunBacktest: get template failed, falling back", zap.Error(err))
			} else if tmpl.Code != "" {
				balance := m.InitialCapital
				if balance <= 0 {
					balance = 10000
				}
				result, err := s.client.Backtest(ctx, &strategysvc.BacktestRequest{
					Code:      tmpl.Code,
					Symbol:    m.Symbol,
					Timeframe: m.Timeframe,
					Balance:   balance,
				})
				if err != nil {
					s.log.Warn("RunBacktest: python backtest failed, falling back", zap.Error(err))
				} else if result.Success {
					riskLevel := result.RiskLevel
					if riskLevel == "" {
						riskLevel = "medium"
					}
					reliable := result.IsReliable
					if !reliable && result.TradeCount >= 10 {
						reliable = true
					}
					return connect.NewResponse(&antv1.RunBacktestResponse{
						Success: true,
						Metrics: &antv1.BacktestMetrics{
							TotalReturn:   result.TotalReturn,
							AnnualReturn:  result.AnnualReturn,
							MaxDrawdown:   result.MaxDrawdown,
							SharpeRatio:   result.SharpeRatio,
							WinRate:       result.WinRate,
							ProfitFactor:  result.ProfitFactor,
							TotalTrades:   result.TotalTrades,
							WinningTrades: result.WinningTrades,
							LosingTrades:  result.LosingTrades,
							AverageProfit: result.AverageProfit,
							AverageLoss:   result.AverageLoss,
						},
						RiskScore:    result.RiskScore,
						RiskLevel:    riskLevel,
						RiskReasons:  result.RiskReasons,
						RiskWarnings: result.RiskWarnings,
						IsReliable:   reliable,
					}), nil
				}
			}
		}
	}
	return connect.NewResponse(&antv1.RunBacktestResponse{
		Success:    true,
		RiskLevel:  "unknown",
		IsReliable: true,
	}), nil
}

// --- Signals ---

func (s *StrategyServer) ListSignals(ctx context.Context, req *connect.Request[antv1.ListSignalsRequest]) (*connect.Response[antv1.ListSignalsResponse], error) {
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

func (s *StrategyServer) ExecuteSignal(ctx context.Context, req *connect.Request[antv1.ExecuteSignalRequest]) (*connect.Response[antv1.ExecuteSignalResponse], error) {
	id, err := uuid.Parse(req.Msg.SignalId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	row, err := s.svc.ExecuteSignal(ctx, id, s.userID(ctx))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &antv1.ExecuteSignalResponse{
		Ticket: row.Ticket,
		Symbol: row.Symbol,
		Type:   row.SignalType,
		Volume: row.Volume,
		Price:  row.Price,
	}
	if row.ExecutedAt != nil {
		resp.ExecutedAt = timestamppb.New(*row.ExecutedAt)
	}
	return connect.NewResponse(resp), nil
}

func (s *StrategyServer) ConfirmSignal(ctx context.Context, req *connect.Request[antv1.ConfirmSignalRequest]) (*connect.Response[emptypb.Empty], error) {
	id, err := uuid.Parse(req.Msg.SignalId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.svc.ConfirmSignal(ctx, id, s.userID(ctx)); err != nil {
		return nil, err
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *StrategyServer) CancelSignal(ctx context.Context, req *connect.Request[antv1.CancelSignalRequest]) (*connect.Response[emptypb.Empty], error) {
	id, err := uuid.Parse(req.Msg.SignalId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.svc.CancelSignal(ctx, id, s.userID(ctx)); err != nil {
		return nil, err
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}
