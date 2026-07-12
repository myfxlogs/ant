package main

import (
	"context"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"alphaforge/internal/connect/gateway"
	"alphaforge/internal/repository"
	"alphaforge/internal/service"
	systemai "alphaforge/internal/service/systemai"
)

// wireAIBilling configures wallet pre-check and post-call billing on the AI service.
// quotaChecker enforces subscription plan AI token limits before the API call.
func wireAIBilling(
	aiSvc *systemai.Service,
	walletSvc *service.WalletService,
	gatewayServer *gateway.AIGatewayServer,
	gatewayModelRepo *repository.AIModelRepository,
	quotaChecker *service.QuotaChecker,
	tokenUsageRepo *repository.AITokenUsageRepository,
) {
	// Wire wallet pre-check: block AI calls when balance is insufficient OR quota exceeded.
	aiMinBalance := decimal.NewFromFloat(1.0)
	if v := os.Getenv("AI_MIN_BALANCE"); v != "" {
		if parsed, err := decimal.NewFromString(v); err == nil && parsed.IsPositive() {
			aiMinBalance = parsed
		}
	}
	aiSvc.SetWalletChecker(func(ctx context.Context, userID uuid.UUID) error {
		// 1. Check subscription AI token quota.
		if quotaChecker != nil {
			usedThisMonth := 0
			if summary, err := tokenUsageRepo.MonthlySummary(ctx, userID); err == nil {
				for _, v := range summary {
					usedThisMonth += v
				}
			}
			remaining := quotaChecker.CheckAITokenQuota(userID, usedThisMonth)
			if remaining == 0 {
				return systemai.ErrInsufficientBalance
			}
		}
		// 2. Check wallet balance.
		w, err := walletSvc.GetOrCreateWallet(ctx, userID)
		if err != nil {
			return nil // don't block on wallet lookup errors
		}
		bal, err := decimal.NewFromString(w.Balance)
		if err != nil {
			return nil // don't block on parse errors
		}
		if bal.LessThan(aiMinBalance) {
			return systemai.ErrInsufficientBalance
		}
		return nil
	})

	// Wire post-call billing: after a successful AI call, deduct cost before returning the result.
	// If deduction fails (insufficient balance), the error propagates to the user and they see a
	// friendly "insufficient balance" message. No free AI.
	aiSvc.SetPostCallBiller(func(ctx context.Context, userID uuid.UUID, providerID, modelName, feature string, inputTokens, outputTokens int) error {
		cost := computeTokenCost(gatewayModelRepo, providerID, modelName, inputTokens, outputTokens)
		if err := gatewayServer.RecordTokenUsage(ctx, userID, "system", providerID, modelName, feature, inputTokens, outputTokens, cost); err != nil {
			if strings.Contains(err.Error(), "insufficient balance") {
				return systemai.ErrInsufficientBalance
			}
			return err
		}
		return nil
	})
}

// computeTokenCost calculates the cost of an AI call based on model pricing.
// Uses decimal arithmetic to avoid float64 precision issues.
func computeTokenCost(
	gatewayModelRepo *repository.AIModelRepository,
	providerID, modelName string,
	inputTokens, outputTokens int,
) string {
	m, err := gatewayModelRepo.GetByProviderAndModel(context.Background(), providerID, modelName)
	if err != nil || m == nil {
		return "0"
	}
	ip, err := decimal.NewFromString(m.PricePer1MInput)
	if err != nil {
		return "0"
	}
	op, err := decimal.NewFromString(m.PricePer1MOutput)
	if err != nil {
		return "0"
	}
	million := decimal.NewFromInt(1_000_000)
	inCost := decimal.NewFromInt(int64(inputTokens)).Div(million).Mul(ip)
	outCost := decimal.NewFromInt(int64(outputTokens)).Div(million).Mul(op)
	return inCost.Add(outCost).StringFixed(8)
}
