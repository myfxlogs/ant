package autotrading

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"connectrpc.com/connect"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/model"
)

// GetAutoTradingStatus returns a summary of the user's auto-trading state.
func (s *AutoTradingServer) GetAutoTradingStatus(
	ctx context.Context,
	_ *connect.Request[antv1.GetAutoTradingStatusRequest],
) (*connect.Response[antv1.AutoTradingStatus], error) {
	uid := s.userID(ctx)
	if uid == uuid.Nil {
		return nil, connect.NewError(connect.CodeUnauthenticated,
			fmt.Errorf("authentication required"))
	}
	gs, err := s.autoRepo.GetGlobalSettingsByUserID(ctx, uid)
	globalEnabled := false
	if err == nil && gs != nil {
		globalEnabled = gs.AutoTradeEnabled
	}

	// Resolve live metrics from existing tables; degrade to 0 on any query failure.
	activeStrategies := 0
	if n, err := s.autoRepo.CountActiveSchedules(ctx, uid); err == nil {
		activeStrategies = n
	}
	pendingSignals := 0
	if n, err := s.autoRepo.CountPendingExecutions(ctx, uid); err == nil {
		pendingSignals = n
	}
	todayExecutions := 0
	if n, err := s.autoRepo.CountTodayExecutionsByUser(ctx, uid); err == nil {
		todayExecutions = n
	}
	todayProfit := 0.0
	if p, err := s.autoRepo.GetTodayProfitByUser(ctx, uid); err == nil {
		todayProfit = p
	}

	return connect.NewResponse(&antv1.AutoTradingStatus{
		GlobalEnabled:    globalEnabled,
		ActiveStrategies: int32(activeStrategies),
		PendingSignals:   int32(pendingSignals),
		TodayExecutions:  int32(todayExecutions),
		TodayProfit:      strconv.FormatFloat(todayProfit, 'f', -1, 64),
	}), nil
}

// GetTradingLogs returns paginated trading logs for the current user.
func (s *AutoTradingServer) GetTradingLogs(
	ctx context.Context,
	req *connect.Request[antv1.GetTradingLogsRequest],
) (*connect.Response[antv1.GetTradingLogsResponse], error) {
	uid := s.userID(ctx)
	if uid == uuid.Nil {
		return nil, connect.NewError(connect.CodeUnauthenticated,
			fmt.Errorf("authentication required"))
	}
	params := &model.LogListParams{
		Page:      int(req.Msg.Page),
		PageSize:  int(req.Msg.PageSize),
		Module:    req.Msg.LogType,
		StartDate: req.Msg.StartDate,
		EndDate:   req.Msg.EndDate,
	}
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	logs, total, err := s.autoRepo.GetTradingLogs(ctx, uid, params)
	if err != nil {
		s.log.Error("get trading logs failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*antv1.TradingLog, len(logs))
	for i, l := range logs {
		out[i] = tradingLogToProto(l)
	}
	return connect.NewResponse(&antv1.GetTradingLogsResponse{
		Logs: out, Total: int32(total),
	}), nil
}

// GetRecentTradingLogs returns the most recent trading logs.
func (s *AutoTradingServer) GetRecentTradingLogs(
	ctx context.Context,
	req *connect.Request[antv1.GetRecentTradingLogsRequest],
) (*connect.Response[antv1.GetRecentTradingLogsResponse], error) {
	uid := s.userID(ctx)
	if uid == uuid.Nil {
		return nil, connect.NewError(connect.CodeUnauthenticated,
			fmt.Errorf("authentication required"))
	}
	limit := int(req.Msg.Limit)
	if limit <= 0 {
		limit = 20
	}
	logs, err := s.autoRepo.GetRecentTradingLogs(ctx, uid, limit)
	if err != nil {
		s.log.Error("get recent trading logs failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*antv1.TradingLog, len(logs))
	for i, l := range logs {
		out[i] = tradingLogToProto(l)
	}
	return connect.NewResponse(&antv1.GetRecentTradingLogsResponse{Logs: out}), nil
}
