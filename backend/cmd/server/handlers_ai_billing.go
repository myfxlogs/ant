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
	aiMinBalance := parseAIMinBalance()

	aiSvc.SetWalletChecker(func(ctx context.Context, userID uuid.UUID) error {
		if dailyQuota != nil {
			if err := dailyQuota.CheckQuota(ctx, userID); err != nil {
				return err
			}
		}
		remaining := monthlyTokenRemaining(quotaChecker, tokenUsageRepo, userID)
		if remaining == 0 {
			return checkWalletBalance(walletSvc, userID, aiMinBalance)
		}
		return nil
	})

	aiSvc.SetPostCallBiller(func(ctx context.Context, userID uuid.UUID, providerID, modelName, feature string, inputTokens, outputTokens int) error {
		cost := computeTokenCost(gatewayModelRepo, providerID, modelName, inputTokens, outputTokens)
		skipDeduction := monthlyTokenRemaining(quotaChecker, tokenUsageRepo, userID) != 0

		if err := gatewayServer.RecordTokenUsage(ctx, userID, "system", providerID, modelName, feature, inputTokens, outputTokens, cost, skipDeduction); err != nil {
			if strings.Contains(err.Error(), "insufficient balance") {
				return systemai.ErrInsufficientBalance
			}
			return err
		}
		return nil
	})
}

func parseAIMinBalance() decimal.Decimal {
	aiMinBalance := decimal.NewFromFloat(1.0)
	if v := os.Getenv("AI_MIN_BALANCE"); v != "" {
		if parsed, err := decimal.NewFromString(v); err == nil && !parsed.IsNegative() {
			aiMinBalance = parsed
		}
	}
	return aiMinBalance
}

func monthlyTokenRemaining(quotaChecker *service.QuotaChecker, tokenUsageRepo *repository.AITokenUsageRepository, userID uuid.UUID) int {
	if quotaChecker == nil {
		return -1
	}
	usedThisMonth := 0
	if summary, err := tokenUsageRepo.MonthlySummary(context.Background(), userID); err == nil {
		for _, v := range summary {
			usedThisMonth += v
		}
	}
	return quotaChecker.CheckAITokenQuota(userID, usedThisMonth)
}

func checkWalletBalance(walletSvc *service.WalletService, userID uuid.UUID, minBalance decimal.Decimal) error {
	w, err := walletSvc.GetOrCreateWallet(context.Background(), userID)
	if err != nil {
		return systemai.ErrInsufficientBalance
	}
	bal, err := decimal.NewFromString(w.Balance)
	if err != nil {
		return systemai.ErrInsufficientBalance
	}
	if bal.LessThan(minBalance) {
		return systemai.ErrInsufficientBalance
	}
	return nil
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
