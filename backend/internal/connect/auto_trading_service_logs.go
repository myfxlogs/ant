package connect

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/interceptor"
	"anttrader/internal/model"
)

func (s *AutoTradingService) GetTradingLogs(ctx context.Context, req *connect.Request[v1.GetTradingLogsRequest]) (*connect.Response[v1.GetTradingLogsResponse], error) {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeTradeRead); err != nil {
		return nil, err
	}

	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("errors.not_authenticated"))
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	params := &model.LogListParams{
		Page:       int(req.Msg.Page),
		PageSize:   int(req.Msg.PageSize),
		Module:     req.Msg.LogType,
		ActionType: "",
		StartDate:  req.Msg.StartDate,
		EndDate:    req.Msg.EndDate,
	}

	logs, total, err := s.autoTradingSvc.GetTradingLogs(ctx, uid, params)
	if err != nil {
		return nil, err
	}

	response := &v1.GetTradingLogsResponse{
		Total: int32(total),
		Logs:  make([]*v1.TradingLog, len(logs)),
	}

	for i, log := range logs {
		response.Logs[i] = &v1.TradingLog{
			Id:        log.ID.String(),
			UserId:    log.UserID.String(),
			AccountId: log.AccountID.String(),
			LogType:   log.LogType,
			Action:    log.Action,
			Symbol:    log.Symbol,
			Details:   log.Message,
			Volume:    log.Volume,
			Price:     log.Price,
			Ticket:    log.Ticket,
			Profit:    log.Profit,
			CreatedAt: timestamppb.New(log.CreatedAt),
		}
	}

	return connect.NewResponse(response), nil
}

func (s *AutoTradingService) GetRecentTradingLogs(ctx context.Context, req *connect.Request[v1.GetRecentTradingLogsRequest]) (*connect.Response[v1.GetRecentTradingLogsResponse], error) {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeTradeRead); err != nil {
		return nil, err
	}

	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("errors.not_authenticated"))
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	logs, err := s.autoTradingSvc.GetRecentTradingLogs(ctx, uid, int(req.Msg.Limit))
	if err != nil {
		return nil, err
	}

	response := &v1.GetRecentTradingLogsResponse{
		Logs: make([]*v1.TradingLog, len(logs)),
	}

	for i, log := range logs {
		response.Logs[i] = &v1.TradingLog{
			Id:        log.ID.String(),
			UserId:    log.UserID.String(),
			AccountId: log.AccountID.String(),
			LogType:   log.LogType,
			Action:    log.Action,
			Symbol:    log.Symbol,
			Details:   log.Message,
			Volume:    log.Volume,
			Price:     log.Price,
			Ticket:    log.Ticket,
			Profit:    log.Profit,
			CreatedAt: timestamppb.New(log.CreatedAt),
		}
	}

	return connect.NewResponse(response), nil
}
