package connect

import (
	"context"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"

	v1 "anttrader/gen/proto"
	"anttrader/internal/repository"
	"anttrader/internal/service/systemai"
)

// SystemAIService is the connect handler for the SystemAIService RPC. It owns
// the systemai service plus a list of optional UI metadata (docs/apply links)
// that are served alongside each config row but never persisted.
type SystemAIService struct {
	svc *systemai.Service
}

func NewSystemAIService(svc *systemai.Service) *SystemAIService {
	return &SystemAIService{svc: svc}
}

// providerLinks supplies docs/apply URLs that the frontend renders verbatim.
// They are display-only metadata that does not belong in the DB.
var providerLinks = map[string]struct{ docs, apply string }{
	"openai":            {"https://platform.openai.com/docs/api-reference", "https://platform.openai.com/api-keys"},
	"anthropic":         {"https://docs.anthropic.com/en/api/getting-started", "https://console.anthropic.com/settings/keys"},
	"deepseek":          {"https://api-docs.deepseek.com/", "https://platform.deepseek.com/api_keys"},
	"qwen":              {"https://help.aliyun.com/zh/dashscope/", "https://bailian.console.aliyun.com/?apiKey=1"},
	"moonshot":          {"https://platform.moonshot.cn/docs", "https://platform.moonshot.cn/console/api-keys"},
	"zhipu":             {"https://open.bigmodel.cn/dev/api", "https://open.bigmodel.cn/usercenter/apikeys"},
	"openai_compatible": {"https://platform.openai.com/docs/api-reference/introduction", ""},
}

func toSystemAIProto(row *repository.SystemAIConfigRow) *v1.SystemAIConfig {
	links := providerLinks[row.ProviderID]
	return &v1.SystemAIConfig{
		ProviderId:     row.ProviderID,
		Name:           row.Name,
		BaseUrl:        row.BaseURL,
		Organization:   row.Organization,
		Models:         row.Models,
		DefaultModel:   row.DefaultModel,
		Temperature:    row.Temperature,
		TimeoutSeconds: int32(row.TimeoutSeconds),
		MaxTokens:      int32(row.MaxTokens),
		Purposes:       row.Purposes,
		PrimaryFor:     row.PrimaryFor,
		Enabled:        row.Enabled,
		HasSecret:      row.HasSecret,
		CreatedAt:      row.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      row.UpdatedAt.Format(time.RFC3339),
		UpdatedBy:      row.UpdatedBy,
		DocsUrl:        links.docs,
		ApplyUrl:       links.apply,
	}
}

func currentUserTag(ctx context.Context) string {
	uid, err := getUserIDFromContext(ctx)
	if err != nil {
		return ""
	}
	return uid.String()
}

func (s *SystemAIService) ListSystemAIConfigs(ctx context.Context, _ *connect.Request[v1.ListSystemAIConfigsRequest]) (*connect.Response[v1.ListSystemAIConfigsResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.svc.List(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*v1.SystemAIConfig, 0, len(rows))
	for _, r := range rows {
		out = append(out, toSystemAIProto(r))
	}
	return connect.NewResponse(&v1.ListSystemAIConfigsResponse{Items: out}), nil
}

func (s *SystemAIService) GetSystemAIConfig(ctx context.Context, req *connect.Request[v1.GetSystemAIConfigRequest]) (*connect.Response[v1.GetSystemAIConfigResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	row, err := s.svc.Get(ctx, userID, strings.TrimSpace(req.Msg.ProviderId))
	if err != nil {
		if errors.Is(err, repository.ErrSystemAIConfigNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.GetSystemAIConfigResponse{Item: toSystemAIProto(row)}), nil
}

func (s *SystemAIService) UpdateSystemAIConfig(ctx context.Context, req *connect.Request[v1.UpdateSystemAIConfigRequest]) (*connect.Response[v1.UpdateSystemAIConfigResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	pid := strings.TrimSpace(req.Msg.ProviderId)
	if pid == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("provider_id is required"))
	}
	row := &repository.SystemAIConfigRow{
		UserID:         userID,
		ProviderID:     pid,
		Name:           req.Msg.Name,
		BaseURL:        strings.TrimSpace(req.Msg.BaseUrl),
		Organization:   req.Msg.Organization,
		Models:         req.Msg.Models,
		DefaultModel:   strings.TrimSpace(req.Msg.DefaultModel),
		Temperature:    req.Msg.Temperature,
		TimeoutSeconds: int(req.Msg.TimeoutSeconds),
		MaxTokens:      int(req.Msg.MaxTokens),
		Purposes:       req.Msg.Purposes,
		PrimaryFor:     req.Msg.PrimaryFor,
		Enabled:        req.Msg.Enabled,
	}
	if err := s.svc.UpdateConfig(ctx, row, currentUserTag(ctx)); err != nil {
		if errors.Is(err, repository.ErrSystemAIConfigNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.UpdateSystemAIConfigResponse{ProviderId: pid}), nil
}

func (s *SystemAIService) UpdateSystemAISecret(ctx context.Context, req *connect.Request[v1.UpdateSystemAISecretRequest]) (*connect.Response[v1.UpdateSystemAISecretResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	pid := strings.TrimSpace(req.Msg.ProviderId)
	if pid == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("provider_id is required"))
	}
	if err := s.svc.UpdateSecret(ctx, userID, pid, req.Msg.Secret, currentUserTag(ctx)); err != nil {
		if errors.Is(err, repository.ErrSystemAIConfigNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.UpdateSystemAISecretResponse{
		ProviderId:    pid,
		SecretUpdated: strings.TrimSpace(req.Msg.Secret) != "",
	}), nil
}

func (s *SystemAIService) DiscoverSystemAIModels(ctx context.Context, req *connect.Request[v1.DiscoverSystemAIModelsRequest]) (*connect.Response[v1.DiscoverSystemAIModelsResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	pid := strings.TrimSpace(req.Msg.ProviderId)
	models, err := s.svc.DiscoverModels(ctx, userID, pid)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New(systemai.FriendlyError(err)))
	}
	if len(models) == 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("no models discovered"))
	}
	// `row.Models` 是用户在 /ai/settings 显式策划的「已启用模型」清单（会出现
	// 在 /ai/agents 下拉里），不应被 /v1/models 全量结果覆盖；否则用户每次清理
	// 都会被下一次自动发现重新灌满。这里只在 default_model 缺省时回填一个，
	// 发现到的完整列表通过 RPC 响应 Models 字段返回给前端，由前端用作建议项。
	if strings.TrimSpace(models[0]) != "" {
		if row, gerr := s.svc.Get(ctx, userID, pid); gerr == nil && strings.TrimSpace(row.DefaultModel) == "" {
			row.DefaultModel = models[0]
			_ = s.svc.UpdateConfig(ctx, row, currentUserTag(ctx))
		}
	}
	return connect.NewResponse(&v1.DiscoverSystemAIModelsResponse{
		ProviderId:   pid,
		Models:       models,
		DefaultModel: models[0],
	}), nil
}

func (s *SystemAIService) ValidateSystemAIConnection(ctx context.Context, req *connect.Request[v1.ValidateSystemAIConnectionRequest]) (*connect.Response[v1.ValidateSystemAIConnectionResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	pid := strings.TrimSpace(req.Msg.ProviderId)
	models, err := s.svc.DiscoverModels(ctx, userID, pid)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New(systemai.FriendlyError(err)))
	}
	return connect.NewResponse(&v1.ValidateSystemAIConnectionResponse{
		ProviderId: pid,
		Ok:         true,
		ModelCount: int32(len(models)),
	}), nil
}
