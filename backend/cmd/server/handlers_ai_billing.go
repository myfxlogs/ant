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
	dailyQuota *service.DailyQuotaChecker,
) {
	// Wire wallet pre-check: block AI calls when quota exceeded AND balance insufficient.
	// Users within their monthly token quota can use AI without a wallet balance.
	// Wallet balance is only enforced when the monthly quota is exhausted.
	aiMinBalance := decimal.NewFromFloat(1.0)
	if v := os.Getenv("AI_MIN_BALANCE"); v != "" {
		if parsed, err := decimal.NewFromString(v); err == nil && !parsed.IsNegative() {
			aiMinBalance = parsed
		}
	}
	aiSvc.SetWalletChecker(func(ctx context.Context, userID uuid.UUID) error {
		// 1. Per-user daily quota (sessions + tokens).
		if dailyQuota != nil {
			if err := dailyQuota.CheckQuota(ctx, userID); err != nil {
				return err
			}
		}
		// 2. Check subscription AI token quota (monthly).
		// remaining: -1 = unlimited, 0 = exhausted, > 0 = tokens left.
		var remaining int = -1 // default unlimited if no checker
		if quotaChecker != nil {
			usedThisMonth := 0
			if summary, err := tokenUsageRepo.MonthlySummary(ctx, userID); err == nil {
				for _, v := range summary {
					usedThisMonth += v
				}
			}
			remaining = quotaChecker.CheckAITokenQuota(userID, usedThisMonth)
		}
		// 3. Wallet balance check — only enforced when monthly quota is exhausted.
		// Users within quota (or unlimited) skip this check entirely.
		if remaining == 0 {
			w, err := walletSvc.GetOrCreateWallet(ctx, userID)
			if err != nil {
				return systemai.ErrInsufficientBalance // fail-closed on DB error
			}
			bal, err := decimal.NewFromString(w.Balance)
			if err != nil {
				return systemai.ErrInsufficientBalance
			}
			if bal.LessThan(aiMinBalance) {
				return systemai.ErrInsufficientBalance
			}
		}
		return nil
	})

	// Wire post-call billing: after a successful AI call, record usage and deduct cost.
	// Users within their monthly token quota: system-paid (no wallet deduction).
	// Quota-exhausted users: wallet deduction applies.
	aiSvc.SetPostCallBiller(func(ctx context.Context, userID uuid.UUID, providerID, modelName, feature string, inputTokens, outputTokens int) error {
		cost := computeTokenCost(gatewayModelRepo, providerID, modelName, inputTokens, outputTokens)

		// Determine whether to skip wallet deduction based on monthly quota.
		skipDeduction := false
		if quotaChecker != nil {
			usedThisMonth := 0
			if summary, err := tokenUsageRepo.MonthlySummary(ctx, userID); err == nil {
				for _, v := range summary {
					usedThisMonth += v
				}
			}
			remaining := quotaChecker.CheckAITokenQuota(userID, usedThisMonth)
			// remaining: -1 = unlimited, >0 = within quota → skip deduction
			// remaining: 0 = exhausted → charge wallet
			skipDeduction = remaining != 0
		}

		if err := gatewayServer.RecordTokenUsage(ctx, userID, "system", providerID, modelName, feature, inputTokens, outputTokens, cost, skipDeduction); err != nil {
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
