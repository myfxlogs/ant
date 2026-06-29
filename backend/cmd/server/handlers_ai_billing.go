package main

import (
	"context"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"anttrader/internal/connect/gateway"
	"anttrader/internal/repository"
	"anttrader/internal/service"
	systemai "anttrader/internal/service/systemai"
)

// wireAIBilling configures wallet pre-check, post-call billing, and token recording
// on the AI service. All three hooks work together to ensure no free AI calls.
func wireAIBilling(
	aiSvc *systemai.Service,
	walletSvc *service.WalletService,
	gatewayServer *gateway.AIGatewayServer,
	gatewayModelRepo *repository.AIModelRepository,
	log *zap.Logger,
) {
	// Wire wallet pre-check: block AI calls when balance is insufficient.
	aiMinBalance := 1.0
	if v := os.Getenv("AI_MIN_BALANCE"); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil && parsed >= 0 {
			aiMinBalance = parsed
		}
	}
	aiSvc.SetWalletChecker(func(ctx context.Context, userID uuid.UUID) error {
		w, err := walletSvc.GetOrCreateWallet(ctx, userID)
		if err != nil {
			return nil // don't block on wallet lookup errors
		}
		bal, _ := strconv.ParseFloat(w.Balance, 64)
		if bal < aiMinBalance {
			return systemai.ErrInsufficientBalance
		}
		return nil
	})

	// Wire post-call billing: after a successful AI call, deduct cost before returning the result.
	// If deduction fails (insufficient balance), the error propagates to the user and they see a
	// friendly "insufficient balance" message. No free AI.
	aiSvc.SetPostCallBiller(func(ctx context.Context, userID uuid.UUID, providerID, modelName string, inputTokens, outputTokens int) error {
		cost := "0"
		if m, err := gatewayModelRepo.GetByProviderAndModel(ctx, providerID, modelName); err == nil && m != nil {
			ip, _ := strconv.ParseFloat(m.PricePer1MInput, 64)
			op, _ := strconv.ParseFloat(m.PricePer1MOutput, 64)
			inCost := float64(inputTokens) / 1_000_000.0 * ip
			outCost := float64(outputTokens) / 1_000_000.0 * op
			cost = strconv.FormatFloat(inCost+outCost, 'f', 8, 64)
		}
		if err := gatewayServer.RecordTokenUsage(ctx, userID, "system", providerID, modelName, "chat", inputTokens, outputTokens, cost); err != nil {
			// Map wallet insufficient balance to the AI-level sentinel so handlers
			// can detect it with errors.Is(err, systemai.ErrInsufficientBalance).
			if strings.Contains(err.Error(), "insufficient balance") {
				return systemai.ErrInsufficientBalance
			}
			return err
		}
		return nil
	})

	// Wire token billing: all ChatCompletion calls automatically record usage through this hook.
	aiSvc.SetTokenRecorder(func(ctx context.Context, r systemai.TokenRecord) {
		// Compute cost from model pricing (per-1M-token rates).
		cost := "0"
		if m, err := gatewayModelRepo.GetByProviderAndModel(ctx, r.ProviderID, r.Model); err == nil && m != nil {
			ip, _ := strconv.ParseFloat(m.PricePer1MInput, 64)
			op, _ := strconv.ParseFloat(m.PricePer1MOutput, 64)
			inCost := float64(r.InputTokens) / 1_000_000.0 * ip
			outCost := float64(r.OutputTokens) / 1_000_000.0 * op
			cost = strconv.FormatFloat(inCost+outCost, 'f', 8, 64)
		}
		// Usage tracking only (billing is handled by postCallBiller).
		_ = gatewayServer.RecordTokenUsage(ctx, r.UserID, "user", r.ProviderID, r.Model, r.Feature, r.InputTokens, r.OutputTokens, cost)
	})
}
