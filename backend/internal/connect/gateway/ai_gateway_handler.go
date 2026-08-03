package gateway

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/pkg/secretbox"
	"alphaforge/internal/repository"
	"alphaforge/internal/service"
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
			Id:              m.ID.String(),
			ProviderId:      m.ProviderID.String(),
			ModelName:       m.ModelName,
			DisplayName:     m.DisplayName,
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
	for k, v := range summary { ft[k] = int32(v) }
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
		p.BaseURL = *r.BaseUrl
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

func (s *AIGatewayServer) ListModels(
	ctx context.Context,
	req *connect.Request[antv1.ListModelsRequest],
) (*connect.Response[antv1.ListModelsResponse], error) {
	pid, err := uuid.Parse(req.Msg.ProviderId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid provider_id"))
	}
	models, err := s.modelRepo.ListByProvider(ctx, pid)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*antv1.AIModelConfig, 0, len(models))
	for _, m := range models {
		out = append(out, &antv1.AIModelConfig{
			Id: m.ID.String(), ModelName: m.ModelName, DisplayName: m.DisplayName,
			PricePer_1MInput: m.PricePer1MInput, PricePer_1MOutput: m.PricePer1MOutput,
			Enabled: m.Enabled, SortOrder: int32(m.SortOrder),
		})
	}
	return connect.NewResponse(&antv1.ListModelsResponse{Models: out}), nil
}

func (s *AIGatewayServer) UpsertModel(
	ctx context.Context,
	req *connect.Request[antv1.UpsertModelRequest],
) (*connect.Response[antv1.UpsertModelResponse], error) {
	r := req.Msg
	pid, err := uuid.Parse(r.ProviderId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid provider_id"))
	}
	m := &repository.AIModel{
		ProviderID: pid, ModelName: r.ModelName,
		PricePer1MInput: r.PricePer_1MInput, PricePer1MOutput: r.PricePer_1MOutput,
		Enabled: true,
	}
	if r.Id != nil {
		id, err := uuid.Parse(*r.Id)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid id"))
		}
		m.ID = id
	}
	if r.DisplayName != nil {
		m.DisplayName = *r.DisplayName
	}
	if r.Enabled != nil {
		m.Enabled = *r.Enabled
	}
	if r.SortOrder != nil {
		m.SortOrder = int(*r.SortOrder)
	}
	newID, err := s.modelRepo.Upsert(ctx, m)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.UpsertModelResponse{Id: newID.String()}), nil
}

func (s *AIGatewayServer) DeleteProvider(
	ctx context.Context,
	req *connect.Request[antv1.DeleteProviderRequest],
) (*connect.Response[antv1.DeleteProviderResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid id"))
	}
	if err := s.providerRepo.Delete(ctx, id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.DeleteProviderResponse{}), nil
}

func (s *AIGatewayServer) DeleteModel(
	ctx context.Context,
	req *connect.Request[antv1.DeleteModelRequest],
) (*connect.Response[antv1.DeleteModelResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid id"))
	}
	if err := s.modelRepo.Delete(ctx, id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.DeleteModelResponse{}), nil
}

// ── Token billing helper (called by chat pipeline) ──

// RecordTokenUsage records a token usage event and deducts wallet if paid_by=system.
// When skipDeduction is true, the usage record is inserted without charging the wallet
// (used for within-quota calls where tokens are included in the user's plan).
func (s *AIGatewayServer) RecordTokenUsage(
	ctx context.Context,
	userID uuid.UUID,
	paidBy, providerID, modelName, feature string,
	inputTokens, outputTokens int,
	cost string,
	skipDeduction bool,
) error {
	rec := &repository.AITokenUsage{
		UserID: userID, PaidBy: paidBy, ProviderID: providerID,
		ModelName: modelName, Feature: feature,
		InputTokens: inputTokens, OutputTokens: outputTokens, Cost: cost,
	}
	if paidBy == "system" && cost != "" && s.walletSvc != nil && !skipDeduction {
		w, err := s.walletSvc.GetOrCreateWallet(ctx, userID)
		if err != nil {
			return fmt.Errorf("get wallet: %w", err)
		}
		bal, err := decimal.NewFromString(w.Balance)
		if err != nil {
			return fmt.Errorf("parse wallet balance: %w", err)
		}
		costD, err := decimal.NewFromString(cost)
		if err != nil {
			return fmt.Errorf("parse cost: %w", err)
		}
		if bal.LessThan(costD) {
			return &service.InsufficientBalanceError{Balance: w.Balance, Cost: cost}
		}
		// Insert usage record BEFORE deducting wallet — if Insert fails,
		// we haven't charged the user.
		if err := s.tokenUsageRepo.Insert(ctx, rec); err != nil {
			return fmt.Errorf("insert token usage: %w", err)
		}
		desc := fmt.Sprintf("AI %s (%s): %d+%d tokens", feature, modelName, inputTokens, outputTokens)
		if _, err := s.walletSvc.AdjustBalance(ctx, userID, "-"+cost, "ai_usage", desc, nil, "ai-"+rec.ID.String()); err != nil {
			return fmt.Errorf("deduct wallet: %w", err)
		}
		return nil
	}
	return s.tokenUsageRepo.Insert(ctx, rec)
}
