package ai

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/repository"
	systemai "anttrader/internal/service/systemai"
)

// SystemAIServer implements ant.v1.SystemAIServiceHandler.
type SystemAIServer struct {
	systemSvc *systemai.Service
	log       *zap.Logger
}

var _ antv1c.SystemAIServiceHandler = (*SystemAIServer)(nil)

func NewSystemAIServer(systemSvc *systemai.Service, log *zap.Logger) *SystemAIServer {
	return &SystemAIServer{systemSvc: systemSvc, log: log}
}

func (s *SystemAIServer) ListSystemAIConfigs(ctx context.Context, req *connect.Request[antv1.ListSystemAIConfigsRequest]) (*connect.Response[antv1.ListSystemAIConfigsResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.systemSvc.List(ctx, uid)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("%s", systemai.FriendlyError(err)))
	}
	items := make([]*antv1.SystemAIConfig, len(rows))
	for i, r := range rows {
		items[i] = systemai.RowToProto(r)
	}
	return connect.NewResponse(&antv1.ListSystemAIConfigsResponse{Items: items}), nil
}

func (s *SystemAIServer) GetSystemAIConfig(ctx context.Context, req *connect.Request[antv1.GetSystemAIConfigRequest]) (*connect.Response[antv1.GetSystemAIConfigResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if req.Msg.ProviderId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("provider_id is required"))
	}
	row, err := s.systemSvc.Get(ctx, uid, req.Msg.ProviderId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("%s", systemai.FriendlyError(err)))
	}
	return connect.NewResponse(&antv1.GetSystemAIConfigResponse{Item: systemai.RowToProto(row)}), nil
}

func (s *SystemAIServer) UpdateSystemAIConfig(ctx context.Context, req *connect.Request[antv1.UpdateSystemAIConfigRequest]) (*connect.Response[antv1.UpdateSystemAIConfigResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if req.Msg.ProviderId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("provider_id is required"))
	}
	// Validate inputs.
	if len(req.Msg.Name) > 100 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name must be 100 characters or fewer"))
	}
	if req.Msg.BaseUrl != "" {
		if _, err := url.Parse(req.Msg.BaseUrl); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid base_url: %w", err))
		}
	}
	row := &repository.SystemAIConfigRow{
		UserID:         uid,
		ProviderID:     req.Msg.ProviderId,
		Name:           req.Msg.Name,
		BaseURL:        req.Msg.BaseUrl,
		Organization:   req.Msg.Organization,
		Models:         req.Msg.Models,
		DefaultModel:   req.Msg.DefaultModel,
		Temperature:    req.Msg.Temperature,
		TimeoutSeconds: int(req.Msg.TimeoutSeconds),
		MaxTokens:      int(req.Msg.MaxTokens),
		Purposes:       req.Msg.Purposes,
		PrimaryFor:     req.Msg.PrimaryFor,
		Enabled:        req.Msg.Enabled,
	}
	if err := s.systemSvc.UpdateConfig(ctx, row, uid.String()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("%s", systemai.FriendlyError(err)))
	}
	return connect.NewResponse(&antv1.UpdateSystemAIConfigResponse{ProviderId: req.Msg.ProviderId}), nil
}

func (s *SystemAIServer) UpdateSystemAISecret(ctx context.Context, req *connect.Request[antv1.UpdateSystemAISecretRequest]) (*connect.Response[antv1.UpdateSystemAISecretResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if req.Msg.ProviderId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("provider_id is required"))
	}
	maskedSecret := "***"
	n := len(req.Msg.Secret)
	if n > 0 {
		// At least 50% masked, minimum 4 masked chars.
		masked := n / 2
		if masked < 4 {
			masked = 4
		}
		if masked > n {
			masked = n
		}
		visible := n - masked
		prefixLen := visible / 2
		suffixLen := visible - prefixLen
		maskedSecret = req.Msg.Secret[:prefixLen] + strings.Repeat("*", masked) + req.Msg.Secret[n-suffixLen:]
	}
	s.log.Info("UpdateSystemAISecret", zap.String("provider_id", req.Msg.ProviderId), zap.String("secret", maskedSecret))
	if err := s.systemSvc.UpdateSecret(ctx, uid, req.Msg.ProviderId, req.Msg.Secret, uid.String()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("%s", systemai.FriendlyError(err)))
	}
	return connect.NewResponse(&antv1.UpdateSystemAISecretResponse{
		ProviderId:    req.Msg.ProviderId,
		SecretUpdated: true,
	}), nil
}

func (s *SystemAIServer) DiscoverSystemAIModels(ctx context.Context, req *connect.Request[antv1.DiscoverSystemAIModelsRequest]) (*connect.Response[antv1.DiscoverSystemAIModelsResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if req.Msg.ProviderId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("provider_id is required"))
	}
	models, err := s.systemSvc.DiscoverModels(ctx, uid, req.Msg.ProviderId)
	if err != nil {
		s.log.Warn("discover models failed",
			zap.String("provider", req.Msg.ProviderId),
			zap.String("raw_error", err.Error()))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("%s", systemai.FriendlyError(err)))
	}
	def := ""
	if len(models) > 0 {
		def = models[0]
	}
	return connect.NewResponse(&antv1.DiscoverSystemAIModelsResponse{
		ProviderId:   req.Msg.ProviderId,
		Models:       models,
		DefaultModel: def,
	}), nil
}

func (s *SystemAIServer) ValidateSystemAIConnection(ctx context.Context, req *connect.Request[antv1.ValidateSystemAIConnectionRequest]) (*connect.Response[antv1.ValidateSystemAIConnectionResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if req.Msg.ProviderId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("provider_id is required"))
	}
	models, err := s.systemSvc.DiscoverModels(ctx, uid, req.Msg.ProviderId)
	if err != nil {
		s.log.Warn("validate connection failed",
			zap.String("provider", req.Msg.ProviderId),
			zap.String("raw_error", err.Error()))
		return connect.NewResponse(&antv1.ValidateSystemAIConnectionResponse{
			ProviderId: req.Msg.ProviderId,
			Ok:         false,
		}), nil
	}
	return connect.NewResponse(&antv1.ValidateSystemAIConnectionResponse{
		ProviderId: req.Msg.ProviderId,
		Ok:         true,
		ModelCount: int32(len(models)),
	}), nil
}
