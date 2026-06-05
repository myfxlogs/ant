package strategy

import (
	"context"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/service"
)

// --- Templates ---

func (s *StrategyServer) ListTemplates(ctx context.Context, req *connect.Request[antv1.ListTemplatesRequest]) (*connect.Response[antv1.ListTemplatesResponse], error) {
	rows, err := s.svc.ListTemplates(ctx, s.userID(ctx))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	templates := make([]*antv1.StrategyTemplate, len(rows))
	for i, r := range rows {
		templates[i] = templateRowToProto(&r)
	}
	return connect.NewResponse(&antv1.ListTemplatesResponse{Templates: templates}), nil
}

func (s *StrategyServer) GetTemplate(ctx context.Context, req *connect.Request[antv1.GetTemplateRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	row, err := s.svc.GetTemplate(ctx, id, s.userID(ctx))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(templateRowToProto(row)), nil
}

func (s *StrategyServer) CreateTemplate(ctx context.Context, req *connect.Request[antv1.CreateTemplateRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
	m := req.Msg
	t := service.TemplateRow{
		UserID:      s.userID(ctx),
		Name:        m.Name,
		Description: m.Description,
		Code:        m.Code,
		Status:      "published",
		Parameters:  templateParamsToProto(m.Parameters),
		IsPublic:    m.IsPublic,
		Tags:        m.Tags,
	}
	if err := s.svc.CreateTemplate(ctx, &t); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(templateRowToProto(&t)), nil
}

func (s *StrategyServer) UpdateTemplate(ctx context.Context, req *connect.Request[antv1.UpdateTemplateRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
	m := req.Msg
	id, err := uuid.Parse(m.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	existing, err := s.svc.GetTemplate(ctx, id, s.userID(ctx))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if m.Name != nil {
		existing.Name = *m.Name
	}
	if m.Description != nil {
		existing.Description = *m.Description
	}
	if m.Code != nil {
		existing.Code = *m.Code
	}
	if len(m.Parameters) > 0 {
		existing.Parameters = templateParamsToProto(m.Parameters)
	}
	if m.IsPublic != nil {
		existing.IsPublic = *m.IsPublic
	}
	if m.Tags != nil {
		existing.Tags = m.Tags
	}
	if err := s.svc.UpdateTemplate(ctx, existing); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(templateRowToProto(existing)), nil
}

func (s *StrategyServer) DeleteTemplate(ctx context.Context, req *connect.Request[antv1.DeleteTemplateRequest]) (*connect.Response[emptypb.Empty], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.svc.DeleteTemplate(ctx, id, s.userID(ctx)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *StrategyServer) CreateTemplateDraft(ctx context.Context, req *connect.Request[antv1.CreateTemplateDraftRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
	t := service.TemplateRow{
		UserID:     s.userID(ctx),
		Name:       req.Msg.Name,
		Status:     "draft",
		Parameters: []byte("[]"),
		Tags:       []string{},
	}
	if err := s.svc.CreateTemplate(ctx, &t); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(templateRowToProto(&t)), nil
}

func (s *StrategyServer) UpdateTemplateDraft(ctx context.Context, req *connect.Request[antv1.UpdateTemplateDraftRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
	m := req.Msg
	id, err := uuid.Parse(m.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	existing, err := s.svc.GetTemplate(ctx, id, s.userID(ctx))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if m.Name != nil {
		existing.Name = *m.Name
	}
	if m.Description != nil {
		existing.Description = *m.Description
	}
	if m.Code != nil {
		existing.Code = *m.Code
	}
	if len(m.Parameters) > 0 {
		existing.Parameters = templateParamsToProto(m.Parameters)
	}
	if m.Tags != nil {
		existing.Tags = m.Tags
	}
	if err := s.svc.UpdateTemplate(ctx, existing); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(templateRowToProto(existing)), nil
}

func (s *StrategyServer) PublishTemplateDraft(ctx context.Context, req *connect.Request[antv1.PublishTemplateDraftRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	existing, err := s.svc.GetTemplate(ctx, id, s.userID(ctx))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := s.svc.SetTemplateStatus(ctx, id, s.userID(ctx), "published"); err != nil {
		return nil, err
	}
	existing.Status = "published"
	return connect.NewResponse(templateRowToProto(existing)), nil
}

func (s *StrategyServer) CancelTemplateDraft(ctx context.Context, req *connect.Request[antv1.CancelTemplateDraftRequest]) (*connect.Response[emptypb.Empty], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.svc.SetTemplateStatus(ctx, id, s.userID(ctx), "canceled"); err != nil {
		return nil, err
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}
