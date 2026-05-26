package ai

import (
	"context"
	"time"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
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

func (s *SystemAIServer) userID(ctx context.Context) uuid.UUID {
	id, _ := uuid.Parse(interceptor.GetUserID(ctx))
	return id
}

func rowToProto(r *repository.SystemAIConfigRow) *antv1.SystemAIConfig {
	return &antv1.SystemAIConfig{
		ProviderId:     r.ProviderID,
		Name:           r.Name,
		BaseUrl:        r.BaseURL,
		Organization:   r.Organization,
		Models:         r.Models,
		DefaultModel:   r.DefaultModel,
		Temperature:    r.Temperature,
		TimeoutSeconds: int32(r.TimeoutSeconds),
		MaxTokens:      int32(r.MaxTokens),
		Purposes:       r.Purposes,
		PrimaryFor:     r.PrimaryFor,
		Enabled:        r.Enabled,
		HasSecret:      r.HasSecret,
		CreatedAt:      r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      r.UpdatedAt.Format(time.RFC3339),
		UpdatedBy:      r.UpdatedBy,
	}
}

func (s *SystemAIServer) ListSystemAIConfigs(ctx context.Context, req *connect.Request[antv1.ListSystemAIConfigsRequest]) (*connect.Response[antv1.ListSystemAIConfigsResponse], error) {
	rows, err := s.systemSvc.List(ctx, s.userID(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]*antv1.SystemAIConfig, len(rows))
	for i, r := range rows {
		items[i] = rowToProto(r)
	}
	return connect.NewResponse(&antv1.ListSystemAIConfigsResponse{Items: items}), nil
}

func (s *SystemAIServer) GetSystemAIConfig(ctx context.Context, req *connect.Request[antv1.GetSystemAIConfigRequest]) (*connect.Response[antv1.GetSystemAIConfigResponse], error) {
	row, err := s.systemSvc.Get(ctx, s.userID(ctx), req.Msg.ProviderId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&antv1.GetSystemAIConfigResponse{Item: rowToProto(row)}), nil
}

func (s *SystemAIServer) UpdateSystemAIConfig(ctx context.Context, req *connect.Request[antv1.UpdateSystemAIConfigRequest]) (*connect.Response[antv1.UpdateSystemAIConfigResponse], error) {
	uid := s.userID(ctx)
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
		return nil, err
	}
	return connect.NewResponse(&antv1.UpdateSystemAIConfigResponse{ProviderId: req.Msg.ProviderId}), nil
}

func (s *SystemAIServer) UpdateSystemAISecret(ctx context.Context, req *connect.Request[antv1.UpdateSystemAISecretRequest]) (*connect.Response[antv1.UpdateSystemAISecretResponse], error) {
	uid := s.userID(ctx)
	if err := s.systemSvc.UpdateSecret(ctx, uid, req.Msg.ProviderId, req.Msg.Secret, uid.String()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&antv1.UpdateSystemAISecretResponse{
		ProviderId:    req.Msg.ProviderId,
		SecretUpdated: true,
	}), nil
}

func (s *SystemAIServer) DiscoverSystemAIModels(ctx context.Context, req *connect.Request[antv1.DiscoverSystemAIModelsRequest]) (*connect.Response[antv1.DiscoverSystemAIModelsResponse], error) {
	models, err := s.systemSvc.DiscoverModels(ctx, s.userID(ctx), req.Msg.ProviderId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
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
	models, err := s.systemSvc.DiscoverModels(ctx, s.userID(ctx), req.Msg.ProviderId)
	if err != nil {
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
