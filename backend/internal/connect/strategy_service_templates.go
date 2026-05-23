package connect

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/model"
	"anttrader/internal/repository"
	"anttrader/internal/service"
)

func (s *StrategyService) validateTemplateCodeStrict(ctx context.Context, code string) error {
	if s.pythonSvc == nil {
		return errors.New("python strategy service not available")
	}
	resp, err := s.pythonSvc.ValidateStrategy(ctx, code)
	if err != nil {
		return err
	}
	if !resp.Valid {
		if len(resp.Errors) > 0 {
			return errors.New(resp.Errors[0])
		}
		return errors.New("策略代码验证未通过")
	}
	if len(resp.Errors) > 0 {
		return errors.New(resp.Errors[0])
	}
	if len(resp.Warnings) > 0 {
		return errors.New(strings.Join(resp.Warnings, "; "))
	}
	return nil
}

func (s *StrategyService) ListTemplates(ctx context.Context, req *connect.Request[v1.ListTemplatesRequest]) (*connect.Response[v1.ListTemplatesResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if s.templateSvc == nil {
		return connect.NewResponse(&v1.ListTemplatesResponse{Templates: []*v1.StrategyTemplate{}}), nil
	}
	items, err := s.templateSvc.GetTemplatesByUser(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	locale := pickPreferredLocale(req)
	out := make([]*v1.StrategyTemplate, 0, len(items))
	for _, it := range items {
		out = append(out, convertTemplateToPBWithLocale(it, locale))
	}
	return connect.NewResponse(&v1.ListTemplatesResponse{Templates: out}), nil
}

func (s *StrategyService) GetTemplate(ctx context.Context, req *connect.Request[v1.GetTemplateRequest]) (*connect.Response[v1.StrategyTemplate], error) {
	if s.templateSvc == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("template service not available"))
	}
	tplID, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	tpl, err := s.templateSvc.GetTemplate(ctx, tplID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	locale := pickPreferredLocale(req)
	return connect.NewResponse(convertTemplateToPBWithLocale(tpl, locale)), nil
}

func (s *StrategyService) CreateTemplate(ctx context.Context, req *connect.Request[v1.CreateTemplateRequest]) (*connect.Response[v1.StrategyTemplate], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if s.templateSvc == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("template service not available"))
	}
	if err := s.validateTemplateCodeStrict(ctx, req.Msg.Code); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	params := make([]model.TemplateParameter, 0, len(req.Msg.Parameters))
	for _, p := range req.Msg.Parameters {
		if p == nil {
			continue
		}
		params = append(params, model.TemplateParameter{
			Name:        p.Name,
			Type:        p.Type,
			Default:     p.Default,
			Min:         p.Min,
			Max:         p.Max,
			Step:        p.Step,
			Label:       p.Label,
			Description: p.Description,
			Options:     p.Options,
		})
	}
	tpl, err := s.templateSvc.CreateTemplate(ctx, userID, req.Msg.Name, req.Msg.Description, req.Msg.Code, params, req.Msg.IsPublic, req.Msg.Tags)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(convertTemplateToPB(tpl)), nil
}

func (s *StrategyService) CreateTemplateDraft(ctx context.Context, req *connect.Request[v1.CreateTemplateDraftRequest]) (*connect.Response[v1.StrategyTemplate], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if s.templateSvc == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("template service not available"))
	}
	tpl, err := s.templateSvc.CreateDraft(ctx, userID, req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(convertTemplateToPB(tpl)), nil
}

func (s *StrategyService) UpdateTemplateDraft(ctx context.Context, req *connect.Request[v1.UpdateTemplateDraftRequest]) (*connect.Response[v1.StrategyTemplate], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if s.templateSvc == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("template service not available"))
	}
	tplID, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	tpl, err := s.templateSvc.GetTemplate(ctx, tplID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if req.Msg.Name != nil {
		tpl.Name = *req.Msg.Name
	}
	if req.Msg.Description != nil {
		tpl.Description = *req.Msg.Description
	}
	if req.Msg.Code != nil {
		if err := s.validateTemplateCodeStrict(ctx, *req.Msg.Code); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		tpl.Code = *req.Msg.Code
	}
	if req.Msg.Tags != nil {
		tpl.Tags = req.Msg.Tags
	}
	if req.Msg.Parameters != nil {
		params := make([]model.TemplateParameter, 0, len(req.Msg.Parameters))
		for _, p := range req.Msg.Parameters {
			if p == nil {
				continue
			}
			params = append(params, model.TemplateParameter{
				Name:        p.Name,
				Type:        p.Type,
				Default:     p.Default,
				Min:         p.Min,
				Max:         p.Max,
				Step:        p.Step,
				Label:       p.Label,
				Description: p.Description,
				Options:     p.Options,
			})
		}
		_ = tpl.SetParameters(params)
	}
	if err := s.templateSvc.UpdateDraft(ctx, userID, tpl); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(convertTemplateToPB(tpl)), nil
}

func (s *StrategyService) PublishTemplateDraft(ctx context.Context, req *connect.Request[v1.PublishTemplateDraftRequest]) (*connect.Response[v1.StrategyTemplate], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if s.templateSvc == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("template service not available"))
	}
	tplID, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	candidate, err := s.templateSvc.GetTemplate(ctx, tplID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if candidate.UserID != userID {
		return nil, connect.NewError(connect.CodePermissionDenied, service.ErrTemplateUnauthorized)
	}
	if err := s.validateTemplateCodeStrict(ctx, candidate.Code); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	tpl, err := s.templateSvc.PublishDraft(ctx, userID, tplID)
	if err != nil {
		if errors.Is(err, service.ErrTemplateNotDraft) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		if errors.Is(err, service.ErrTemplateCodeEmpty) || errors.Is(err, service.ErrTemplateNameEmpty) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		if errors.Is(err, service.ErrTemplateUnauthorized) {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(convertTemplateToPB(tpl)), nil
}

func (s *StrategyService) CancelTemplateDraft(ctx context.Context, req *connect.Request[v1.CancelTemplateDraftRequest]) (*connect.Response[emptypb.Empty], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if s.templateSvc == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("template service not available"))
	}
	tplID, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.templateSvc.CancelDraft(ctx, userID, tplID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *StrategyService) UpdateTemplate(ctx context.Context, req *connect.Request[v1.UpdateTemplateRequest]) (*connect.Response[v1.StrategyTemplate], error) {
	if s.templateSvc == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("template service not available"))
	}
	tplID, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	tpl, err := s.templateSvc.GetTemplate(ctx, tplID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if req.Msg.Name != nil {
		tpl.Name = *req.Msg.Name
	}
	if req.Msg.Description != nil {
		tpl.Description = *req.Msg.Description
	}
	if req.Msg.Code != nil {
		if err := s.validateTemplateCodeStrict(ctx, *req.Msg.Code); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		tpl.Code = *req.Msg.Code
	}
	if req.Msg.IsPublic != nil {
		tpl.IsPublic = *req.Msg.IsPublic
	}
	if req.Msg.Tags != nil {
		tpl.Tags = req.Msg.Tags
	}
	if req.Msg.Parameters != nil {
		params := make([]model.TemplateParameter, 0, len(req.Msg.Parameters))
		for _, p := range req.Msg.Parameters {
			if p == nil {
				continue
			}
			params = append(params, model.TemplateParameter{
				Name:        p.Name,
				Type:        p.Type,
				Default:     p.Default,
				Min:         p.Min,
				Max:         p.Max,
				Step:        p.Step,
				Label:       p.Label,
				Description: p.Description,
				Options:     p.Options,
			})
		}
		_ = tpl.SetParameters(params)
	}
	if err := s.templateSvc.UpdateTemplate(ctx, tpl); err != nil {
		if errors.Is(err, repository.ErrTemplateIsSystem) {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(convertTemplateToPB(tpl)), nil
}

func (s *StrategyService) DeleteTemplate(ctx context.Context, req *connect.Request[v1.DeleteTemplateRequest]) (*connect.Response[emptypb.Empty], error) {
	if s.templateSvc == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("template service not available"))
	}
	tplID, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.templateSvc.DeleteTemplate(ctx, tplID); err != nil {
		if errors.Is(err, repository.ErrTemplateIsSystem) {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
		if errors.Is(err, repository.ErrTemplateNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func convertTemplateToPB(t *model.StrategyTemplate) *v1.StrategyTemplate {
	if t == nil {
		return &v1.StrategyTemplate{}
	}
	paramsOut := []*v1.TemplateParameter{}
	if ps, err := t.GetParameters(); err == nil {
		for _, p := range ps {
			p0 := p
			paramsOut = append(paramsOut, &v1.TemplateParameter{
				Name:        p0.Name,
				Type:        p0.Type,
				Default:     p0.Default,
				Min:         p0.Min,
				Max:         p0.Max,
				Step:        p0.Step,
				Label:       p0.Label,
				Description: p0.Description,
				Options:     p0.Options,
			})
		}
	}
	return &v1.StrategyTemplate{
		Id:          t.ID.String(),
		UserId:      t.UserID.String(),
		Name:        t.Name,
		Description: t.Description,
		Code:        t.Code,
		Parameters:  paramsOut,
		IsPublic:    t.IsPublic,
		IsSystem:    t.IsSystem,
		Tags:        t.Tags,
		UseCount:    int32(t.UseCount),
		Status:      t.Status,
		CreatedAt:   timestamppb.New(t.CreatedAt),
		UpdatedAt:   timestamppb.New(t.UpdatedAt),
	}
}

// convertTemplateToPBWithLocale applies i18n overrides for name/description when available.
func convertTemplateToPBWithLocale(t *model.StrategyTemplate, locale string) *v1.StrategyTemplate {
	pb := convertTemplateToPB(t)
	if t == nil {
		return pb
	}
	if i18n, _ := t.GetI18n(); i18n != nil {
		// Prefer exact match, then language-only, then en, then zh-CN, then original
		if i18n.Name != nil {
			if v := pickI18n(i18n.Name, locale); v != "" {
				pb.Name = v
			}
		}
		if i18n.Description != nil {
			if v := pickI18n(i18n.Description, locale); v != "" {
				pb.Description = v
			}
		}
		if ps, err := t.GetParameters(); err == nil && len(ps) == len(pb.Parameters) && i18n.Params != nil {
			// Patch parameter labels/descriptions when present
			for i, p := range ps {
				if pI18n, ok := i18n.Params[p.Name]; ok {
					if pI18n.Label != nil {
						if v := pickI18n(pI18n.Label, locale); v != "" {
							pb.Parameters[i].Label = v
						}
					}
					if pI18n.Description != nil {
						if v := pickI18n(pI18n.Description, locale); v != "" {
							pb.Parameters[i].Description = v
						}
					}
				}
			}
		}
	}
	return pb
}

// pickPreferredLocale extracts the preferred locale from the request headers.
func pickPreferredLocale[T any](req *connect.Request[T]) string {
	h := req.Header().Get("Accept-Language")
	if h == "" {
		return ""
	}
	// take first token before comma, trim spaces
	token := h
	if idx := strings.IndexRune(h, ','); idx >= 0 {
		token = h[:idx]
	}
	token = strings.TrimSpace(strings.ToLower(token))
	switch token {
	case "zh-cn", "zh-hans", "zh":
		return "zh-CN"
	case "zh-tw", "zh-hant":
		return "zh-TW"
	case "en", "en-us", "en-gb":
		return "en"
	case "ja", "ja-jp":
		return "ja"
	case "vi", "vi-vn":
		return "vi"
	default:
		// language-only fallback, e.g. fr -> fr
		if len(token) >= 2 {
			return strings.ToLower(token[:2])
		}
		return ""
	}
}

// pickI18n returns localized text with sensible fallbacks.
func pickI18n(m map[string]string, locale string) string {
	if m == nil {
		return ""
	}
	// exact locale
	if v := m[locale]; v != "" {
		return v
	}
	// language-only
	if len(locale) >= 2 {
		if v := m[strings.ToLower(locale[:2])]; v != "" {
			return v
		}
	}
	if v := m["en"]; v != "" {
		return v
	}
	if v := m["zh-CN"]; v != "" {
		return v
	}
	// any value
	for _, v := range m {
		if v != "" {
			return v
		}
	}
	return ""
}
