package connect

import (
	"context"
	"encoding/json"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/model"
)

func (s *AdminService) ListLogs(ctx context.Context, req *connect.Request[v1.ListLogsRequest]) (*connect.Response[v1.ListLogsResponse], error) {
	_, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	params := &model.LogListParams{
		Page:       int(req.Msg.Page),
		PageSize:   int(req.Msg.PageSize),
		Module:     req.Msg.Module,
		ActionType: req.Msg.ActionType,
		StartDate:  req.Msg.StartDate,
		EndDate:    req.Msg.EndDate,
		AdminID:    req.Msg.UserId,
	}
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	}

	result, err := s.adminSvc.ListLogs(ctx, params)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	logs := make([]*v1.AdminLog, len(result.Data.([]*model.AdminLog)))
	for i, l := range result.Data.([]*model.AdminLog) {
		logs[i] = convertAdminLogToProto(l)
	}

	return connect.NewResponse(&v1.ListLogsResponse{
		Logs:  logs,
		Total: int32(result.Total),
	}), nil
}

func (s *AdminService) ExportLogs(ctx context.Context, req *connect.Request[v1.ExportLogsRequest]) (*connect.Response[v1.ExportLogsResponse], error) {
	_, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	params := &model.LogListParams{
		Page:       1,
		PageSize:   10000,
		ActionType: req.Msg.Action,
		StartDate:  req.Msg.StartDate,
		EndDate:    req.Msg.EndDate,
		AdminID:    req.Msg.UserId,
	}

	result, err := s.adminSvc.ListLogs(ctx, params)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	logs := make([]map[string]interface{}, len(result.Data.([]*model.AdminLog)))
	for i, l := range result.Data.([]*model.AdminLog) {
		logs[i] = map[string]interface{}{
			"id":         l.ID.String(),
			"user_id":    l.AdminID,
			"action":     l.ActionType,
			"module":     l.Module,
			"target":     l.TargetType + ":" + l.TargetID,
			"details":    l.Details,
			"success":    l.Success,
			"error":      l.ErrorMessage,
			"created_at": l.CreatedAt,
		}
	}

	data, err := json.Marshal(logs)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.ExportLogsResponse{
		Data:     data,
		Filename: "admin_logs.json",
	}), nil
}

func convertAdminLogToProto(l *model.AdminLog) *v1.AdminLog {
	var adminID string
	if l.AdminID != nil {
		adminID = l.AdminID.String()
	}
	detailsBytes, _ := json.Marshal(l.Details)
	return &v1.AdminLog{
		Id:          l.ID.String(),
		UserId:      adminID,
		Module:      l.Module,
		ActionType:  l.ActionType,
		TargetType:  l.TargetType,
		TargetId:    l.TargetID,
		Success:     l.Success,
		ErrorMessage: l.ErrorMessage,
		Action:      l.ActionType,
		Details:     string(detailsBytes),
		Ip:          l.IPAddress,
		CreatedAt:   timestamppb.New(l.CreatedAt),
	}
}
