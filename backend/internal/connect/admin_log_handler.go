package connect

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/model"
	"anttrader/internal/repository"
)

type AdminLogServer struct {
	repo *repository.AdminRepository
	log  *zap.Logger
}

var _ antv1c.AdminLogServiceHandler = (*AdminLogServer)(nil)

func NewAdminLogServer(repo *repository.AdminRepository, log *zap.Logger) *AdminLogServer {
	return &AdminLogServer{repo: repo, log: log}
}

func (s *AdminLogServer) ListLogs(ctx context.Context, req *connect.Request[antv1.ListLogsRequest]) (*connect.Response[antv1.ListLogsResponse], error) {
	params := &model.LogListParams{
		Page:       int(req.Msg.Page),
		PageSize:   int(req.Msg.PageSize),
		Module:     req.Msg.Module,
		ActionType: req.Msg.ActionType,
		StartDate:  req.Msg.StartDate,
		EndDate:    req.Msg.EndDate,
		AdminID:    req.Msg.AdminId,
	}
	logs, total, err := s.repo.ListLogs(ctx, params)
	if err != nil {
		return nil, err
	}
	items := make([]*antv1.AdminLog, len(logs))
	for i, l := range logs {
		item := &antv1.AdminLog{
			Id:            l.ID.String(),
			Module:        l.Module,
			ActionType:    l.ActionType,
			TargetType:    l.TargetType,
			TargetId:      l.TargetID,
			IpAddress:     l.IPAddress,
			UserAgent:     l.UserAgent,
			RequestMethod: l.RequestMethod,
			RequestPath:   l.RequestPath,
			Success:       l.Success,
			ErrorMessage:  l.ErrorMessage,
			CreatedAt:     timestamppb.New(l.CreatedAt),
		}
		if l.AdminID != nil {
			item.AdminId = l.AdminID.String()
		}
		items[i] = item
	}
	return connect.NewResponse(&antv1.ListLogsResponse{Logs: items, Total: total}), nil
}
