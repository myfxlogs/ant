package strategy

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/service"
)

// templateRowToProto converts a service.TemplateRow to a proto StrategyTemplate.
func templateRowToProto(r *service.TemplateRow) *antv1.StrategyTemplate {
	if r == nil {
		return nil
	}
	pb := &antv1.StrategyTemplate{
		Id:          r.ID.String(),
		Name:        r.Name,
		Description: r.Description,
		Code:        r.Code,
		IsPublic:    r.IsPublic,
		IsSystem:    r.IsSystem,
		Tags:        r.Tags,
		UseCount:    r.UseCount,
		Status:      r.Status,
		I18N:        string(r.I18n),
		CreatedAt:   timestamppb.New(r.CreatedAt),
		UpdatedAt:   timestamppb.New(r.UpdatedAt),
	}
	if r.UserID != nil {
		pb.UserId = r.UserID.String()
	}
	if len(r.Parameters) > 0 {
		var params antv1.StrategyTemplate
		if err := proto.Unmarshal(r.Parameters, &params); err == nil {
			pb.Parameters = params.Parameters
		}
	}
	return pb
}

func (s *StrategyServer) ListTemplates(ctx context.Context, req *connect.Request[antv1.ListTemplatesRequest]) (*connect.Response[antv1.ListTemplatesResponse], error) {
	uid := s.userID(ctx)
	rows, err := s.svc.ListTemplates(ctx, uid)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	templates := make([]*antv1.StrategyTemplate, len(rows))
	for i, row := range rows {
		templates[i] = templateRowToProto(&row)
	}
	return connect.NewResponse(&antv1.ListTemplatesResponse{Templates: templates}), nil
}

func (s *StrategyServer) GetTemplate(ctx context.Context, req *connect.Request[antv1.GetTemplateRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
	uid := s.userID(ctx)
	id, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid template id"))
	}
	row, err := s.svc.GetTemplate(ctx, id, uid)
	if err != nil {
		if errors.Is(err, service.ErrTemplateNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("template not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(templateRowToProto(row)), nil
}

func (s *StrategyServer) CreateTemplate(ctx context.Context, req *connect.Request[antv1.CreateTemplateRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
	uid := s.userID(ctx)
	row := &service.TemplateRow{
		UserID:      &uid,
		Name:        req.Msg.GetName(),
		Description: req.Msg.GetDescription(),
		Code:        req.Msg.GetCode(),
		Status:      "published",
		IsPublic:    req.Msg.GetIsPublic(),
		Tags:        req.Msg.GetTags(),
		I18n:        []byte(req.Msg.GetI18N()),
	}
	if req.Msg.Parameters != nil {
		wrapper := &antv1.StrategyTemplate{Parameters: req.Msg.Parameters}
		row.Parameters, _ = proto.Marshal(wrapper)
	}
	if err := s.svc.CreateTemplate(ctx, row); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(templateRowToProto(row)), nil
}

func (s *StrategyServer) UpdateTemplate(ctx context.Context, req *connect.Request[antv1.UpdateTemplateRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
	uid := s.userID(ctx)
	id, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid template id"))
	}
	existing, err := s.svc.GetTemplate(ctx, id, uid)
	if err != nil {
		if errors.Is(err, service.ErrTemplateNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("template not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if v := req.Msg.Name; v != nil {
		existing.Name = *v
	}
	if v := req.Msg.Description; v != nil {
		existing.Description = *v
	}
	if v := req.Msg.Code; v != nil {
		existing.Code = *v
	}
	if v := req.Msg.IsPublic; v != nil {
		existing.IsPublic = *v
	}
	if req.Msg.Tags != nil {
		existing.Tags = req.Msg.Tags
	}
	if v := req.Msg.I18N; v != nil {
		existing.I18n = []byte(*v)
	}
	if req.Msg.Parameters != nil {
		wrapper := &antv1.StrategyTemplate{Parameters: req.Msg.Parameters}
		existing.Parameters, _ = proto.Marshal(wrapper)
	}
	if err := s.svc.UpdateTemplate(ctx, existing); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(templateRowToProto(existing)), nil
}

func (s *StrategyServer) DeleteTemplate(ctx context.Context, req *connect.Request[antv1.DeleteTemplateRequest]) (*connect.Response[emptypb.Empty], error) {
	uid := s.userID(ctx)
	id, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid template id"))
	}
	if err := s.svc.DeleteTemplate(ctx, id, uid); err != nil {
		if errors.Is(err, service.ErrTemplateNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("template not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *StrategyServer) CreateTemplateDraft(ctx context.Context, req *connect.Request[antv1.CreateTemplateDraftRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
	uid := s.userID(ctx)
	row := &service.TemplateRow{
		UserID:   &uid,
		Name:     req.Msg.GetName(),
		Status:   "draft",
		Tags:     []string{},
	}
	if err := s.svc.CreateTemplate(ctx, row); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(templateRowToProto(row)), nil
}

func (s *StrategyServer) UpdateTemplateDraft(ctx context.Context, req *connect.Request[antv1.UpdateTemplateDraftRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
	uid := s.userID(ctx)
	id, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid template id"))
	}
	existing, err := s.svc.GetTemplate(ctx, id, uid)
	if err != nil {
		if errors.Is(err, service.ErrTemplateNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("template not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if v := req.Msg.Name; v != nil {
		existing.Name = *v
	}
	if v := req.Msg.Description; v != nil {
		existing.Description = *v
	}
	if v := req.Msg.Code; v != nil {
		existing.Code = *v
	}
	if req.Msg.Tags != nil {
		existing.Tags = req.Msg.Tags
	}
	if req.Msg.Parameters != nil {
		wrapper := &antv1.StrategyTemplate{Parameters: req.Msg.Parameters}
		existing.Parameters, _ = proto.Marshal(wrapper)
	}
	if err := s.svc.UpdateTemplate(ctx, existing); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(templateRowToProto(existing)), nil
}

func (s *StrategyServer) PublishTemplateDraft(ctx context.Context, req *connect.Request[antv1.PublishTemplateDraftRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
	uid := s.userID(ctx)
	id, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid template id"))
	}
	if err := s.svc.SetTemplateStatus(ctx, id, uid, "published"); err != nil {
		if errors.Is(err, service.ErrTemplateNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("template not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	row, err := s.svc.GetTemplate(ctx, id, uid)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(templateRowToProto(row)), nil
}
