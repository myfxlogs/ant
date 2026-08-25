package gateway

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/pkg/secretbox"
	"alphaforge/internal/repository"
	"alphaforge/internal/service"
	"alphaforge/internal/service/systemai"
)

// AIGatewayServer implements AIGatewayServiceHandler.
type AIGatewayServer struct {
	providerRepo   *repository.SystemAIProviderRepository
	modelRepo      *repository.AIModelRepository
	tokenUsageRepo *repository.AITokenUsageRepository
	walletSvc      *service.WalletService
	box            *secretbox.Box
	log            *zap.Logger
}

var _ antv1c.AIGatewayServiceHandler = (*AIGatewayServer)(nil)

func NewAIGatewayServer(
	providerRepo *repository.SystemAIProviderRepository,
	modelRepo *repository.AIModelRepository,
	tokenUsageRepo *repository.AITokenUsageRepository,
	walletSvc *service.WalletService,
	box *secretbox.Box,
	log *zap.Logger,
) *AIGatewayServer {
	return &AIGatewayServer{
		providerRepo:   providerRepo,
		modelRepo:      modelRepo,
		tokenUsageRepo: tokenUsageRepo,
		walletSvc:      walletSvc,
		box:            box,
		log:            log,
	}
}

// ── User-facing ──

func (s *AIGatewayServer) ListSystemModels(
	ctx context.Context,
	req *connect.Request[antv1.ListSystemModelsRequest],
) (*connect.Response[antv1.ListSystemModelsResponse], error) {
	models, err := s.modelRepo.ListEnabled(ctx)
	if err != nil {
		s.log.Error("ListSystemModels", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*antv1.SystemModel, 0, len(models))
	for _, m := range models {
		out = append(out, &antv1.SystemModel{
			Id:                m.ID.String(),
			ProviderId:        m.ProviderID.String(),
			ModelName:         m.ModelName,
			DisplayName:       m.DisplayName,
			PricePer_1MInput:  m.PricePer1MInput,
			PricePer_1MOutput: m.PricePer1MOutput,
		})
	}
	return connect.NewResponse(&antv1.ListSystemModelsResponse{Models: out}), nil
}

func (s *AIGatewayServer) GetTokenUsage(
	ctx context.Context,
	req *connect.Request[antv1.GetTokenUsageRequest],
) (*connect.Response[antv1.GetTokenUsageResponse], error) {
	userID, err := uuid.Parse(interceptor.GetUserID(ctx))
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid user"))
	}
	summary, err := s.tokenUsageRepo.MonthlySummary(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	monthlyCost, err := s.tokenUsageRepo.MonthlyCost(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	records, err := s.tokenUsageRepo.ListByUser(ctx, userID, 30)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	pbRecords := make([]*antv1.TokenUsageRecord, 0, len(records))
	for _, r := range records {
		pbRecords = append(pbRecords, &antv1.TokenUsageRecord{
			Id: r.ID.String(), PaidBy: r.PaidBy, ProviderId: r.ProviderID,
			ModelName: r.ModelName, Feature: r.Feature,
			InputTokens: int32(r.InputTokens), OutputTokens: int32(r.OutputTokens),
			Cost: r.Cost, CreatedAtTsMs: r.CreatedAt.UnixMilli(),
		})
	}
	ft := make(map[string]int32, len(summary))
	for k, v := range summary {
		ft[k] = int32(v)
	}
	return connect.NewResponse(&antv1.GetTokenUsageResponse{
		FeatureTokens: ft, Records: pbRecords, MonthlyCost: monthlyCost,
	}), nil
}

// ── Admin ──

func (s *AIGatewayServer) ListProviders(
	ctx context.Context,
	req *connect.Request[antv1.ListProvidersRequest],
) (*connect.Response[antv1.ListProvidersResponse], error) {
	providers, err := s.providerRepo.ListAll(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*antv1.AIProvider, 0, len(providers))
	for _, p := range providers {
		out = append(out, &antv1.AIProvider{
			Id:         p.ID.String(),
			ProviderId: p.ProviderID,
			Name:       p.Name,
			BaseUrl:    p.BaseURL,
			Enabled:    p.Enabled,
			HasApiKey:  len(p.APIKeyEncrypted) > 0,
		})
	}
	return connect.NewResponse(&antv1.ListProvidersResponse{Providers: out}), nil
}

func (s *AIGatewayServer) CreateProvider(
	ctx context.Context,
	req *connect.Request[antv1.CreateProviderRequest],
) (*connect.Response[antv1.CreateProviderResponse], error) {
	r := req.Msg
	if strings.TrimSpace(r.ProviderId) == "" || strings.TrimSpace(r.Name) == "" || strings.TrimSpace(r.BaseUrl) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("provider_id, name, and base_url are required"))
	}
	if err := systemai.ValidateBaseURL(strings.TrimSpace(r.BaseUrl)); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid base_url: %w", err))
	}
	if strings.TrimSpace(r.ApiKey) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("api_key is required"))
	}
	if s.box == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encryption not initialized"))
	}
	enc, err := repository.SealAPIKey(strings.TrimSpace(r.ApiKey), s.box)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encrypt api key: %w", err))
	}
	enabled := true
	if r.Enabled != nil {
		enabled = *r.Enabled
	}
	p := &repository.SystemAIProvider{
		ProviderID:      strings.TrimSpace(r.ProviderId),
		Name:            strings.TrimSpace(r.Name),
		BaseURL:         strings.TrimSpace(r.BaseUrl),
		APIKeyEncrypted: enc,
		Enabled:         enabled,
	}
	if err := s.providerRepo.Create(ctx, p); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create provider: %w", err))
	}
	return connect.NewResponse(&antv1.CreateProviderResponse{Id: p.ID.String()}), nil
}

func (s *AIGatewayServer) UpdateProvider(
	ctx context.Context,
	req *connect.Request[antv1.UpdateProviderRequest],
) (*connect.Response[antv1.UpdateProviderResponse], error) {
	r := req.Msg
	id, err := uuid.Parse(r.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid id"))
	}
	p, err := s.providerRepo.GetByID(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("provider not found"))
	}
	if r.Name != nil {
		p.Name = *r.Name
	}
	if r.BaseUrl != nil {
		if err := systemai.ValidateBaseURL(strings.TrimSpace(*r.BaseUrl)); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid base_url: %w", err))
		}
		p.BaseURL = strings.TrimSpace(*r.BaseUrl)
	}
	if r.Enabled != nil {
		p.Enabled = *r.Enabled
	}
	if r.ApiKey != nil && strings.TrimSpace(*r.ApiKey) != "" {
		if s.box == nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encryption not initialized"))
		}
		enc, err := repository.SealAPIKey(strings.TrimSpace(*r.ApiKey), s.box)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encrypt api key: %w", err))
		}
		p.APIKeyEncrypted = enc
	}
	if err := s.providerRepo.Update(ctx, p); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.UpdateProviderResponse{}), nil
}
