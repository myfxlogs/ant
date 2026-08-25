// ai_gateway_handler_usage.go — Token usage and model discovery extracted from ai_gateway_handler.go.
package gateway

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/repository"
	"alphaforge/internal/service"
	"alphaforge/internal/service/systemai"
)

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

func (s *AIGatewayServer) DiscoverGatewayModels(
	ctx context.Context,
	req *connect.Request[antv1.DiscoverGatewayModelsRequest],
) (*connect.Response[antv1.DiscoverGatewayModelsResponse], error) {
	id, err := uuid.Parse(req.Msg.ProviderId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid provider_id"))
	}
	p, err := s.providerRepo.GetByID(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("provider not found"))
	}
	if len(p.APIKeyEncrypted) == 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("provider has no API key configured"))
	}
	secret, err := repository.OpenAPIKey(p.APIKeyEncrypted, s.box)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("decrypt api key: %w", err))
	}
	models, err := systemai.DiscoverModelsByConfig(ctx, p.ProviderID, p.BaseURL, secret)
	if err != nil {
		s.log.Warn("discover gateway models failed",
			zap.String("provider", p.ProviderID),
			zap.String("base_url", p.BaseURL),
			zap.Error(err))
		return connect.NewResponse(&antv1.DiscoverGatewayModelsResponse{}), nil
	}
	return connect.NewResponse(&antv1.DiscoverGatewayModelsResponse{Models: models}), nil
}
