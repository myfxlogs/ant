package connect

import (
	"context"
	"encoding/json"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/interceptor"
	"anttrader/internal/model"
	"anttrader/internal/service"
)

type LogService struct {
	logSvc *service.LogService
}

func NewLogService(logSvc *service.LogService) *LogService {
	return &LogService{logSvc: logSvc}
}

func (s *LogService) GetConnectionLogs(ctx context.Context, req *connect.Request[v1.GetConnectionLogsRequest]) (*connect.Response[v1.GetConnectionLogsResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}

	params := &model.LogQueryParams{
		Page:     int(req.Msg.Page),
		PageSize: int(req.Msg.PageSize),
	}
	if req.Msg.AccountId != "" {
		params.AccountID = req.Msg.AccountId
	}
	if req.Msg.Status != "" {
		params.Status = req.Msg.Status
	}
	if req.Msg.StartDate != "" {
		params.StartDate = req.Msg.StartDate
	}
	if req.Msg.EndDate != "" {
		params.EndDate = req.Msg.EndDate
	}

	logs, total, err := s.logSvc.GetConnectionLogs(ctx, uuid.MustParse(userID), params)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	response := &v1.GetConnectionLogsResponse{
		Logs:  make([]*v1.ConnectionLog, 0, len(logs)),
		Total: int32(total),
	}

	for _, log := range logs {
		response.Logs = append(response.Logs, &v1.ConnectionLog{
			Id:                        log.ID.String(),
			AccountId:                 log.AccountID.String(),
			EventType:                 string(log.EventType),
			Status:                    string(log.Status),
			Message:                   log.Message,
			ErrorDetail:               log.ErrorDetail,
			ServerHost:                log.ServerHost,
			ServerPort:                int32(log.ServerPort),
			LoginId:                   log.LoginID,
			ConnectionDurationSeconds: log.ConnectionDurationSecs,
			CreatedAt:                 timestamppb.New(log.CreatedAt),
		})
	}

	return connect.NewResponse(response), nil
}

func (s *LogService) GetScheduleRunLogs(ctx context.Context, req *connect.Request[v1.GetScheduleRunLogsRequest]) (*connect.Response[v1.GetScheduleRunLogsResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	if req.Msg.ScheduleId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	sid, err := uuid.Parse(req.Msg.ScheduleId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	page := int(req.Msg.Page)
	pageSize := int(req.Msg.PageSize)
	rows, total, err := s.logSvc.GetScheduleRunLogs(ctx, uuid.MustParse(userID), sid, page, pageSize)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := &v1.GetScheduleRunLogsResponse{Logs: make([]*v1.ScheduleRunLog, 0, len(rows)), Total: int32(total)}
	for _, r := range rows {
		resp.Logs = append(resp.Logs, &v1.ScheduleRunLog{
			Id:           r.ID.String(),
			Kind:         r.Kind,
			Action:       r.Action,
			Status:       r.Status,
			DurationMs:   r.DurationMs,
			ErrorMessage: r.ErrorMessage,
			SignalType:   r.SignalType,
			SignalVolume: r.SignalVolume,
			CreatedAt:    timestamppb.New(r.CreatedAt),
		})
	}
	return connect.NewResponse(resp), nil
}

func (s *LogService) GetExecutionLogs(ctx context.Context, req *connect.Request[v1.GetExecutionLogsRequest]) (*connect.Response[v1.GetExecutionLogsResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}

	params := &model.LogQueryParams{
		Page:     int(req.Msg.Page),
		PageSize: int(req.Msg.PageSize),
	}
	if req.Msg.AccountId != "" {
		params.AccountID = req.Msg.AccountId
	}
	if req.Msg.ScheduleId != "" {
		params.ScheduleID = req.Msg.ScheduleId
	}
	if req.Msg.Symbol != "" {
		params.Symbol = req.Msg.Symbol
	}
	if req.Msg.Status != "" {
		params.Status = req.Msg.Status
	}
	if req.Msg.StartDate != "" {
		params.StartDate = req.Msg.StartDate
	}
	if req.Msg.EndDate != "" {
		params.EndDate = req.Msg.EndDate
	}

	logs, total, err := s.logSvc.GetExecutionLogs(ctx, uuid.MustParse(userID), params)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	response := &v1.GetExecutionLogsResponse{
		Logs:  make([]*v1.ExecutionLog, 0, len(logs)),
		Total: int32(total),
	}

	for _, log := range logs {
		el := &v1.ExecutionLog{
			Id:               log.ID.String(),
			Symbol:           log.Symbol,
			Timeframe:        log.Timeframe,
			Status:           string(log.Status),
			SignalPrice:      log.SignalPrice,
			SignalVolume:     log.SignalVolume,
			SignalStopLoss:   log.SignalStopLoss,
			SignalTakeProfit: log.SignalTakeProfit,
			ExecutedOrderId:  log.ExecutedOrderID,
			ExecutedPrice:    log.ExecutedPrice,
			ExecutedVolume:   log.ExecutedVolume,
			Profit:           log.Profit,
			ErrorMessage:     log.ErrorMessage,
			ExecutionTimeMs:  log.ExecutionTimeMs,
			CreatedAt:        timestamppb.New(log.CreatedAt),
		}
		if log.AccountID != nil {
			el.AccountId = log.AccountID.String()
		}
		if log.ScheduleID != nil {
			el.ScheduleId = log.ScheduleID.String()
		}
		if log.SignalType != "" {
			el.SignalType = string(log.SignalType)
		}
		response.Logs = append(response.Logs, el)
	}

	return connect.NewResponse(response), nil
}

func (s *LogService) GetOrderLogHistory(ctx context.Context, req *connect.Request[v1.GetOrderLogHistoryRequest]) (*connect.Response[v1.GetOrderLogHistoryResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}

	params := &model.LogQueryParams{
		Page:     int(req.Msg.Page),
		PageSize: int(req.Msg.PageSize),
	}
	if req.Msg.AccountId != "" {
		params.AccountID = req.Msg.AccountId
	}
	if req.Msg.ScheduleId != "" {
		params.ScheduleID = req.Msg.ScheduleId
	}
	if req.Msg.Symbol != "" {
		params.Symbol = req.Msg.Symbol
	}
	if req.Msg.StartDate != "" {
		params.StartDate = req.Msg.StartDate
	}
	if req.Msg.EndDate != "" {
		params.EndDate = req.Msg.EndDate
	}

	orders, total, err := s.logSvc.GetOrderHistory(ctx, uuid.MustParse(userID), params)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	response := &v1.GetOrderLogHistoryResponse{
		Orders: make([]*v1.OrderHistoryRecord, 0, len(orders)),
		Total:  int32(total),
	}

	for _, order := range orders {
		record := &v1.OrderHistoryRecord{
			Id:         order.ID.String(),
			AccountId:  order.AccountID.String(),
			Ticket:     order.Ticket,
			Symbol:     order.Symbol,
			OrderType:  string(order.OrderType),
			Lots:       order.Volume,
			OpenPrice:  order.OpenPrice,
			ClosePrice: order.ClosePrice,
			Profit:     order.Profit,
			OpenTime:   timestamppb.New(order.OpenTime),
		}
		if order.ScheduleID != uuid.Nil {
			record.ScheduleId = order.ScheduleID.String()
		}
		if order.CloseTime != nil {
			record.CloseTime = timestamppb.New(*order.CloseTime)
		}
		response.Orders = append(response.Orders, record)
	}

	return connect.NewResponse(response), nil
}

func (s *LogService) GetOperationLogs(ctx context.Context, req *connect.Request[v1.GetOperationLogsRequest]) (*connect.Response[v1.GetOperationLogsResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}

	params := &model.LogQueryParams{
		Page:     int(req.Msg.Page),
		PageSize: int(req.Msg.PageSize),
	}
	if req.Msg.Module != "" {
		params.Module = req.Msg.Module
	}
	if req.Msg.Action != "" {
		params.Action = req.Msg.Action
	}
	if req.Msg.ResourceType != "" {
		params.ResourceType = req.Msg.ResourceType
	}
	if req.Msg.ResourceId != "" {
		params.ResourceID = req.Msg.ResourceId
	}
	if req.Msg.StartDate != "" {
		params.StartDate = req.Msg.StartDate
	}
	if req.Msg.EndDate != "" {
		params.EndDate = req.Msg.EndDate
	}

	logs, total, err := s.logSvc.GetOperationLogs(ctx, uuid.MustParse(userID), params)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	response := &v1.GetOperationLogsResponse{
		Logs:  make([]*v1.OperationLog, 0, len(logs)),
		Total: int32(total),
	}

	for _, log := range logs {
		resourceID := ""
		if log.ResourceID != uuid.Nil {
			resourceID = log.ResourceID.String()
		}
		details := ""
		if log.NewValue != nil {
			if b, err := json.Marshal(log.NewValue); err == nil {
				details = string(b)
			}
		}
		if details == "" && log.ErrorMessage != "" {
			details = log.ErrorMessage
		}
		response.Logs = append(response.Logs, &v1.OperationLog{
			Id:        log.ID.String(),
			UserId:    log.UserID.String(),
			Module:    log.Module,
			Action:    log.Action,
			Details:   details,
			Ip:        log.IPAddress,
			UserAgent: log.UserAgent,
			Status:    string(log.Status),
			ErrorMessage: log.ErrorMessage,
			ResourceType: log.ResourceType,
			ResourceId:   resourceID,
			DurationMs:   log.DurationMs,
			CreatedAt: timestamppb.New(log.CreatedAt),
		})
	}

	return connect.NewResponse(response), nil
}
