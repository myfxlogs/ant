package system

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/model"
	"anttrader/internal/service"
)

// LogServiceServer implements ant.v1.LogServiceHandler.
type LogServiceServer struct {
	logSvc *service.LogService
	log    *zap.Logger
}

var _ antv1c.LogServiceHandler = (*LogServiceServer)(nil)

func NewLogServiceServer(logSvc *service.LogService, log *zap.Logger) *LogServiceServer {
	return &LogServiceServer{logSvc: logSvc, log: log}
}

func (s *LogServiceServer) userID(ctx context.Context) uuid.UUID {
	id, _ := uuid.Parse(interceptor.GetUserID(ctx))
	return id
}

func connectionLogToProto(l *model.AccountConnectionLog) *antv1.ConnectionLog {
	return &antv1.ConnectionLog{
		Id:                        l.ID.String(),
		AccountId:                 l.AccountID.String(),
		EventType:                 string(l.EventType),
		Status:                    string(l.Status),
		Message:                   l.Message,
		ErrorDetail:               l.ErrorDetail,
		ServerHost:                l.ServerHost,
		ServerPort:                int32(l.ServerPort),
		LoginId:                   l.LoginID,
		ConnectionDurationSeconds: l.ConnectionDurationSecs,
		CreatedAt:                 timestamppb.New(l.CreatedAt),
	}
}

func executionLogToProto(l *model.StrategyExecutionLog) *antv1.ExecutionLog {
	e := &antv1.ExecutionLog{
		Id:              l.ID.String(),
		Symbol:          l.Symbol,
		Timeframe:       l.Timeframe,
		Status:          string(l.Status),
		SignalType:      string(l.SignalType),
		SignalPrice:     l.SignalPrice,
		SignalVolume:    l.SignalVolume,
		SignalStopLoss:  l.SignalStopLoss,
		SignalTakeProfit: l.SignalTakeProfit,
		ExecutedOrderId: l.ExecutedOrderID,
		ExecutedPrice:   l.ExecutedPrice,
		ExecutedVolume:  l.ExecutedVolume,
		Profit:          l.Profit,
		ErrorMessage:    l.ErrorMessage,
		ExecutionTimeMs: l.ExecutionTimeMs,
		CreatedAt:       timestamppb.New(l.CreatedAt),
	}
	if l.AccountID != nil {
		e.AccountId = l.AccountID.String()
	}
	if l.ScheduleID != nil {
		e.ScheduleId = l.ScheduleID.String()
	}
	return e
}

func orderHistoryToProto(o *model.OrderHistory) *antv1.OrderHistoryRecord {
	r := &antv1.OrderHistoryRecord{
		Id:         o.ID.String(),
		AccountId:  o.AccountID.String(),
		ScheduleId: o.ScheduleID.String(),
		Ticket:     o.Ticket,
		Symbol:     o.Symbol,
		OrderType:  string(o.OrderType),
		Lots:       o.Volume,
		OpenPrice:  o.OpenPrice,
		ClosePrice: o.ClosePrice,
		Profit:     o.Profit,
		OpenTime:   timestamppb.New(o.OpenTime),
	}
	if o.CloseTime != nil {
		r.CloseTime = timestamppb.New(*o.CloseTime)
	}
	return r
}

func operationLogToProto(l *model.SystemOperationLog) *antv1.OperationLog {
	details, _ := json.Marshal(l.NewValue)
	return &antv1.OperationLog{
		Id:           l.ID.String(),
		UserId:       l.UserID.String(),
		Module:       l.Module,
		Action:       l.Action,
		Details:      string(details),
		Ip:           l.IPAddress,
		UserAgent:    l.UserAgent,
		Status:       string(l.Status),
		ErrorMessage: l.ErrorMessage,
		ResourceType: l.ResourceType,
		ResourceId:   l.ResourceID.String(),
		DurationMs:   l.DurationMs,
		CreatedAt:    timestamppb.New(l.CreatedAt),
	}
}

func (s *LogServiceServer) GetConnectionLogs(ctx context.Context, req *connect.Request[antv1.GetConnectionLogsRequest]) (*connect.Response[antv1.GetConnectionLogsResponse], error) {
	params := &model.LogQueryParams{
		Page:      int(req.Msg.Page),
		PageSize:  int(req.Msg.PageSize),
		AccountID: req.Msg.AccountId,
		Status:    req.Msg.Status,
		StartDate: req.Msg.StartDate,
		EndDate:   req.Msg.EndDate,
	}
	logs, total, err := s.logSvc.GetConnectionLogs(ctx, s.userID(ctx), params)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	items := make([]*antv1.ConnectionLog, len(logs))
	for i, l := range logs {
		items[i] = connectionLogToProto(l)
	}
	return connect.NewResponse(&antv1.GetConnectionLogsResponse{Logs: items, Total: int32(total)}), nil
}

func (s *LogServiceServer) GetExecutionLogs(ctx context.Context, req *connect.Request[antv1.GetExecutionLogsRequest]) (*connect.Response[antv1.GetExecutionLogsResponse], error) {
	params := &model.LogQueryParams{
		Page:       int(req.Msg.Page),
		PageSize:   int(req.Msg.PageSize),
		AccountID:  req.Msg.AccountId,
		ScheduleID: req.Msg.ScheduleId,
		Symbol:     req.Msg.Symbol,
		Status:     req.Msg.Status,
		StartDate:  req.Msg.StartDate,
		EndDate:    req.Msg.EndDate,
	}
	logs, total, err := s.logSvc.GetExecutionLogs(ctx, s.userID(ctx), params)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	items := make([]*antv1.ExecutionLog, len(logs))
	for i, l := range logs {
		items[i] = executionLogToProto(l)
	}
	return connect.NewResponse(&antv1.GetExecutionLogsResponse{Logs: items, Total: int32(total)}), nil
}

func (s *LogServiceServer) GetOrderLogHistory(ctx context.Context, req *connect.Request[antv1.GetOrderLogHistoryRequest]) (*connect.Response[antv1.GetOrderLogHistoryResponse], error) {
	params := &model.LogQueryParams{
		Page:       int(req.Msg.Page),
		PageSize:   int(req.Msg.PageSize),
		AccountID:  req.Msg.AccountId,
		ScheduleID: req.Msg.ScheduleId,
		Symbol:     req.Msg.Symbol,
		StartDate:  req.Msg.StartDate,
		EndDate:    req.Msg.EndDate,
	}
	orders, total, err := s.logSvc.GetOrderHistory(ctx, s.userID(ctx), params)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	items := make([]*antv1.OrderHistoryRecord, len(orders))
	for i, o := range orders {
		items[i] = orderHistoryToProto(o)
	}
	return connect.NewResponse(&antv1.GetOrderLogHistoryResponse{Orders: items, Total: int32(total)}), nil
}

func (s *LogServiceServer) GetOperationLogs(ctx context.Context, req *connect.Request[antv1.GetOperationLogsRequest]) (*connect.Response[antv1.GetOperationLogsResponse], error) {
	params := &model.LogQueryParams{
		Page:         int(req.Msg.Page),
		PageSize:     int(req.Msg.PageSize),
		Module:       req.Msg.Module,
		Action:       req.Msg.Action,
		ResourceType: req.Msg.ResourceType,
		ResourceID:   req.Msg.ResourceId,
		StartDate:    req.Msg.StartDate,
		EndDate:      req.Msg.EndDate,
	}
	logs, total, err := s.logSvc.GetOperationLogs(ctx, s.userID(ctx), params)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	items := make([]*antv1.OperationLog, len(logs))
	for i, l := range logs {
		items[i] = operationLogToProto(l)
	}
	return connect.NewResponse(&antv1.GetOperationLogsResponse{Logs: items, Total: int32(total)}), nil
}

func (s *LogServiceServer) GetScheduleRunLogs(ctx context.Context, req *connect.Request[antv1.GetScheduleRunLogsRequest]) (*connect.Response[antv1.GetScheduleRunLogsResponse], error) {
	uid := s.userID(ctx)
	scheduleID, err := uuid.Parse(req.Msg.ScheduleId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	logs, total, err := s.logSvc.GetScheduleRunLogs(ctx, uid, scheduleID, int(req.Msg.Page), int(req.Msg.PageSize))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	items := make([]*antv1.ScheduleRunLog, len(logs))
	for i, l := range logs {
		items[i] = &antv1.ScheduleRunLog{
			Id:           l.ID.String(),
			Kind:         l.Kind,
			Action:       l.Action,
			Status:       l.Status,
			DurationMs:   l.DurationMs,
			ErrorMessage: l.ErrorMessage,
			SignalType:   l.SignalType,
			SignalVolume: l.SignalVolume,
			CreatedAt:    timestamppb.New(l.CreatedAt),
		}
	}
	return connect.NewResponse(&antv1.GetScheduleRunLogsResponse{Logs: items, Total: int32(total)}), nil
}
