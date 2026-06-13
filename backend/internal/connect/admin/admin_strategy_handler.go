package admin

import (
	"context"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/service"
)

var _ antv1c.AdminStrategyServiceHandler = (*AdminStrategyServer)(nil)

type AdminStrategyServer struct {
	svc *service.StrategySvc
	log *zap.Logger
}

func NewAdminStrategyServer(svc *service.StrategySvc, log *zap.Logger) *AdminStrategyServer {
	return &AdminStrategyServer{svc: svc, log: log}
}

// ── System strategy CRUD ──

func (s *AdminStrategyServer) ListSystemStrategies(ctx context.Context, req *connect.Request[antv1.ListSystemStrategiesRequest]) (*connect.Response[antv1.ListSystemStrategiesResponse], error) {
	rows, err := s.svc.ListSystemStrategies(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	strategies := make([]*antv1.SystemStrategy, len(rows))
	for i, r := range rows {
		strategies[i] = &antv1.SystemStrategy{
			Id:          r.ID.String(),
			Name:        r.Name,
			Description: r.Description,
			Code:        r.Code,
			IsActive:    true,
			UseCount:    r.UseCount,
			Tags:        r.Tags,
			CreatedAt:   timestamppb.New(r.CreatedAt),
			UpdatedAt:   timestamppb.New(r.UpdatedAt),
		}
	}
	return connect.NewResponse(&antv1.ListSystemStrategiesResponse{Strategies: strategies}), nil
}

func (s *AdminStrategyServer) CreateSystemStrategy(ctx context.Context, req *connect.Request[antv1.CreateSystemStrategyRequest]) (*connect.Response[antv1.SystemStrategy], error) {
	m := req.Msg
	row, err := s.svc.CreateSystemStrategy(ctx, m.Name, m.Description, m.Code, m.Tags)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.SystemStrategy{
		Id:          row.ID.String(),
		Name:        row.Name,
		Description: row.Description,
		Code:        row.Code,
		IsActive:    true,
		Tags:        row.Tags,
		CreatedAt:   timestamppb.New(row.CreatedAt),
		UpdatedAt:   timestamppb.New(row.UpdatedAt),
	}), nil
}

func (s *AdminStrategyServer) UpdateSystemStrategy(ctx context.Context, req *connect.Request[antv1.UpdateSystemStrategyRequest]) (*connect.Response[antv1.SystemStrategy], error) {
	m := req.Msg
	id, err := uuid.Parse(m.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	row, err := s.svc.UpdateSystemStrategy(ctx, id, m.Name, m.Description, m.Code, m.Tags)
	if err != nil {
		if err == service.ErrTemplateNotFound {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.SystemStrategy{
		Id:          row.ID.String(),
		Name:        row.Name,
		Description: row.Description,
		Code:        row.Code,
		IsActive:    true,
		UseCount:    row.UseCount,
		Tags:        row.Tags,
		CreatedAt:   timestamppb.New(row.CreatedAt),
		UpdatedAt:   timestamppb.New(row.UpdatedAt),
	}), nil
}

func (s *AdminStrategyServer) DeleteSystemStrategy(ctx context.Context, req *connect.Request[antv1.DeleteSystemStrategyRequest]) (*connect.Response[antv1.DeleteSystemStrategyResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.svc.DeleteSystemStrategy(ctx, id); err != nil {
		if err == service.ErrTemplateNotFound {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.DeleteSystemStrategyResponse{}), nil
}

// ── All strategies oversight ──

func (s *AdminStrategyServer) ListAllStrategies(ctx context.Context, req *connect.Request[antv1.ListAllStrategiesRequest]) (*connect.Response[antv1.ListAllStrategiesResponse], error) {
	m := req.Msg
	params := service.ListAllStrategiesParams{
		Page:     m.Page,
		PageSize: m.PageSize,
	}
	if m.Search != nil {
		params.Search = m.Search
	}
	if m.UserId != nil && *m.UserId != "" {
		uid, err := uuid.Parse(*m.UserId)
		if err == nil {
			params.UserID = &uid
		}
	}
	if m.Flag != nil {
		params.Flag = m.Flag
	}

	rows, total, err := s.svc.ListAllStrategies(ctx, params)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	strategies := make([]*antv1.StrategySummary, len(rows))
	for i, r := range rows {
		var userID, userEmail string
		if r.UserID != nil {
			userID = r.UserID.String()
		}
		userEmail = r.UserEmail
		strategies[i] = &antv1.StrategySummary{
			Id:            r.ID.String(),
			Name:          r.Name,
			UserId:        userID,
			UserEmail:     userEmail,
			Status:        r.Status,
			IsSystem:      r.IsSystem,
			IsPublic:      r.IsPublic,
			Flag:          r.Flag,
			FlagReason:    r.FlagReason,
			ScheduleCount: r.ScheduleCount,
			UseCount:      r.UseCount,
			Tags:          r.Tags,
			CreatedAt:     timestamppb.New(r.CreatedAt),
			UpdatedAt:     timestamppb.New(r.UpdatedAt),
		}
		if r.FlaggedBy != nil {
			strategies[i].FlaggedBy = r.FlaggedBy.String()
		}
	}
	return connect.NewResponse(&antv1.ListAllStrategiesResponse{Strategies: strategies, Total: total}), nil
}

func (s *AdminStrategyServer) GetStrategyDetail(ctx context.Context, req *connect.Request[antv1.GetStrategyDetailRequest]) (*connect.Response[antv1.StrategyDetail], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	row, userEmail, err := s.svc.GetTemplateDetail(ctx, id)
	if err != nil {
		if err == service.ErrTemplateNotFound {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	var userID string
	if row.UserID != nil {
		userID = row.UserID.String()
	}
	var flaggedBy string
	if row.FlaggedBy != nil {
		flaggedBy = row.FlaggedBy.String()
	}
	var flaggedAt *timestamppb.Timestamp
	if row.FlaggedAt != nil {
		flaggedAt = timestamppb.New(*row.FlaggedAt)
	}

	return connect.NewResponse(&antv1.StrategyDetail{
		Id:          row.ID.String(),
		Name:        row.Name,
		Description: row.Description,
		Code:        row.Code,
		UserId:      userID,
		UserEmail:   userEmail,
		Status:      row.Status,
		IsSystem:    row.IsSystem,
		IsPublic:    row.IsPublic,
		Flag:        row.Flag,
		FlagReason:  row.FlagReason,
		FlaggedBy:   flaggedBy,
		FlaggedAt:   flaggedAt,
		UseCount:    row.UseCount,
		Tags:        row.Tags,
		CreatedAt:   timestamppb.New(row.CreatedAt),
		UpdatedAt:   timestamppb.New(row.UpdatedAt),
	}), nil
}

// ── Compliance actions ──

func (s *AdminStrategyServer) FlagStrategy(ctx context.Context, req *connect.Request[antv1.FlagStrategyRequest]) (*connect.Response[antv1.FlagStrategyResponse], error) {
	m := req.Msg
	id, err := uuid.Parse(m.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	adminID := getActorID(ctx)
	if err := s.svc.FlagTemplate(ctx, id, m.Reason, adminID); err != nil {
		if err == service.ErrTemplateNotFound {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	s.log.Info("admin: flagged strategy", zap.String("strategy_id", m.Id), zap.String("reason", m.Reason), zap.String("actor", adminID.String()))
	return connect.NewResponse(&antv1.FlagStrategyResponse{}), nil
}

func (s *AdminStrategyServer) UnflagStrategy(ctx context.Context, req *connect.Request[antv1.UnflagStrategyRequest]) (*connect.Response[antv1.UnflagStrategyResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	adminID := getActorID(ctx)
	if err := s.svc.UnflagTemplate(ctx, id); err != nil {
		if err == service.ErrTemplateNotFound {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	s.log.Info("admin: unflagged strategy", zap.String("strategy_id", req.Msg.Id), zap.String("actor", adminID.String()))
	return connect.NewResponse(&antv1.UnflagStrategyResponse{}), nil
}

func (s *AdminStrategyServer) UnpublishStrategy(ctx context.Context, req *connect.Request[antv1.UnpublishStrategyRequest]) (*connect.Response[antv1.UnpublishStrategyResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.svc.UnpublishTemplate(ctx, id); err != nil {
		if err == service.ErrTemplateNotFound {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	s.log.Info("admin: unpublished strategy", zap.String("strategy_id", req.Msg.Id), zap.String("actor", getActorID(ctx).String()))
	return connect.NewResponse(&antv1.UnpublishStrategyResponse{}), nil
}

func (s *AdminStrategyServer) PublishStrategy(ctx context.Context, req *connect.Request[antv1.AdminPublishStrategyRequest]) (*connect.Response[antv1.AdminPublishStrategyResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.svc.PublishTemplate(ctx, id); err != nil {
		if err == service.ErrTemplateNotFound {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	s.log.Info("admin: published strategy", zap.String("strategy_id", req.Msg.Id), zap.String("actor", getActorID(ctx).String()))
	return connect.NewResponse(&antv1.AdminPublishStrategyResponse{}), nil
}

func (s *AdminStrategyServer) DisableStrategy(ctx context.Context, req *connect.Request[antv1.DisableStrategyRequest]) (*connect.Response[antv1.DisableStrategyResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.svc.DisableTemplate(ctx, id); err != nil {
		if err == service.ErrTemplateNotFound {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	s.log.Info("admin: disabled strategy", zap.String("strategy_id", req.Msg.Id), zap.String("actor", getActorID(ctx).String()))
	return connect.NewResponse(&antv1.DisableStrategyResponse{}), nil
}

func (s *AdminStrategyServer) EnableStrategy(ctx context.Context, req *connect.Request[antv1.EnableStrategyRequest]) (*connect.Response[antv1.EnableStrategyResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.svc.EnableTemplate(ctx, id); err != nil {
		if err == service.ErrTemplateNotFound {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	s.log.Info("admin: enabled strategy", zap.String("strategy_id", req.Msg.Id), zap.String("actor", getActorID(ctx).String()))
	return connect.NewResponse(&antv1.EnableStrategyResponse{}), nil
}

func (s *AdminStrategyServer) ArchiveStrategy(ctx context.Context, req *connect.Request[antv1.ArchiveStrategyRequest]) (*connect.Response[antv1.ArchiveStrategyResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.svc.ArchiveTemplate(ctx, id); err != nil {
		if err == service.ErrTemplateNotFound {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	s.log.Info("admin: archived strategy", zap.String("strategy_id", req.Msg.Id), zap.String("actor", getActorID(ctx).String()))
	return connect.NewResponse(&antv1.ArchiveStrategyResponse{}), nil
}
