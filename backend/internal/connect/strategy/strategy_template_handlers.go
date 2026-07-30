package strategy

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/service"
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
		I18N:        parseI18nBytes(r.I18n),
		CreatedAt:   timestamppb.New(r.CreatedAt),
		UpdatedAt:   timestamppb.New(r.UpdatedAt),
	}
	if r.UserID != nil {
		pb.UserId = r.UserID.String()
	}
	if r.StrategyID != nil {
		pb.StrategyId = r.StrategyID.String()
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
	pb := templateRowToProto(row)
	// Strip code for all non-owners. Subscribers can deploy/backtest via
	// marketplace APIs, but source code is the author's IP and must not be exposed.
	if row.UserID != nil && row.UserID.String() != uid.String() && !row.IsSystem {
		pb.Code = ""
	}
	return connect.NewResponse(pb), nil
}

func (s *StrategyServer) ListStrategyCards(ctx context.Context, req *connect.Request[antv1.ListStrategyCardsRequest]) (*connect.Response[antv1.ListStrategyCardsResponse], error) {
	uid := s.userID(ctx)
	rows, total, err := s.svc.ListStrategyCards(ctx, uid, service.ListStrategyCardsParams{
		Filter: req.Msg.GetFilter(),
		Sort:   req.Msg.GetSort(),
		Search: req.Msg.GetSearch(),
		Limit:  int(req.Msg.GetLimit()),
		Offset: int(req.Msg.GetOffset()),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	cards := make([]*antv1.StrategyCard, len(rows))
	for i, r := range rows {
		cards[i] = &antv1.StrategyCard{
			Id:                    r.ID.String(),
			UserId:                r.UserID.String(),
			Name:                  r.Name,
			Description:           r.Description,
			Tags:                  r.Tags,
			IsSystem:              r.IsSystem,
			IsPublic:              r.IsPublic,
			UseCount:              r.UseCount,
			CreatedAt:             timestamppb.New(r.CreatedAt),
			Sparkline:             r.Sparkline,
			WinRate:               r.WinRate,
			MaxDrawdown:           r.MaxDrawdown,
			ProfitFactor:          r.ProfitFactor,
			SharpeRatio:           r.SharpeRatio,
			RunningSchedules:      r.RunningSchedules,
			IsMarketplacePublished: r.IsMarketplacePublished,
		}
		if r.BacktestRunID != nil {
			cards[i].BacktestRunId = r.BacktestRunID.String()
		}
	}
	return connect.NewResponse(&antv1.ListStrategyCardsResponse{Cards: cards, Total: int32(total)}), nil
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
		I18n:        i18nToBytes(req.Msg.GetI18N()),
	}
	if sid := req.Msg.GetStrategyId(); sid != "" {
		if parsed, err := uuid.Parse(sid); err == nil {
			row.StrategyID = &parsed
		}
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
	if existing.UserID != nil && existing.UserID.String() != uid.String() && !existing.IsSystem {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("not the template owner"))
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
		existing.I18n = i18nToBytes(v)
	}
	if v := req.Msg.StrategyId; v != nil && *v != "" {
		if parsed, err := uuid.Parse(*v); err == nil {
			existing.StrategyID = &parsed
		}
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
		UserID: &uid,
		Name:   req.Msg.GetName(),
		Status: "draft",
		Tags:   []string{},
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
	if existing.UserID != nil && existing.UserID.String() != uid.String() && !existing.IsSystem {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("not the template owner"))
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
	if v := req.Msg.StrategyId; v != nil && *v != "" {
		if parsed, err := uuid.Parse(*v); err == nil {
			existing.StrategyID = &parsed
		}
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

// parseI18nBytes parses JSON bytes from DB JSONB into a TemplateI18n proto.
func parseI18nBytes(b []byte) *antv1.TemplateI18N {
	if len(b) == 0 {
		return nil
	}
	var i18n antv1.TemplateI18N
	if err := protojson.Unmarshal(b, &i18n); err != nil {
		return nil
	}
	return &i18n
}

// i18nToBytes converts a TemplateI18n proto to JSON bytes for DB storage.
func i18nToBytes(i18n *antv1.TemplateI18N) []byte {
	if i18n == nil {
		return nil
	}
	b, _ := protojson.Marshal(i18n)
	return b
}
