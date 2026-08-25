// ai_gateway_handler_models.go — Model CRUD operations extracted from ai_gateway_handler.go.
package gateway

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/repository"
)

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
		s.log.Error("upsert model failed",
			zap.String("provider_id", r.ProviderId),
			zap.String("model_name", r.ModelName),
			zap.String("price_in", r.PricePer_1MInput),
			zap.String("price_out", r.PricePer_1MOutput),
			zap.Error(err))
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
