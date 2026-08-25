// subscription_service_billing.go — Billing helpers and charge processing extracted from subscription_service.go.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"alphaforge/internal/model"
)

func computeProrationCredit(existing *model.UserPlatformSubscription, oldPlan *model.SubscriptionPlan) decimal.Decimal {
	now := time.Now().UTC()
	totalDuration := existing.CurrentPeriodEnd.Sub(existing.CurrentPeriodStart)
	remaining := existing.CurrentPeriodEnd.Sub(now)
	if remaining < 0 {
		remaining = 0
	}
	if totalDuration <= 0 {
		return decimal.Zero
	}
	prorationRatio := decimal.NewFromInt(remaining.Nanoseconds()).Div(decimal.NewFromInt(totalDuration.Nanoseconds()))
	oldPriceStr := oldPlan.PriceMonthly
	if existing.BillingCycle == billingCycleYearly {
		oldPriceStr = oldPlan.PriceYearly
	}
	oldPrice, err := decimal.NewFromString(oldPriceStr)
	if err != nil {
		return decimal.Zero
	}
	return oldPrice.Mul(prorationRatio)
}

func planPrice(plan *model.SubscriptionPlan, cycle string) decimal.Decimal {
	priceStr := plan.PriceMonthly
	if cycle == billingCycleYearly {
		priceStr = plan.PriceYearly
	}
	price, _ := decimal.NewFromString(priceStr)
	return price
}

func (s *SubscriptionService) processPlanCharge(ctx context.Context, tx pgx.Tx, userID uuid.UUID, netCharge decimal.Decimal, newPlan *model.SubscriptionPlan, oldPlan *model.SubscriptionPlan) (string, string) {
	if netCharge.IsZero() {
		return "", ""
	}
	walletRepo := s.walletSvc.Repo()
	wallet, err := walletRepo.GetByUserIDTx(ctx, tx, userID)
	if err != nil || wallet == nil {
		return "", ""
	}
	if netCharge.GreaterThan(decimal.Zero) {
		balance, err := decimal.NewFromString(wallet.Balance)
		if err != nil || balance.LessThan(netCharge) {
			return "", ""
		}
		subID := uuid.New()
		updated, err := walletRepo.AdjustBalanceTx(ctx, tx, wallet.ID, userID, netCharge.Neg().String(), "purchase",
			fmt.Sprintf("Platform subscription: %s (%s)", newPlan.DisplayName, ""), nil, "sub-"+subID.String())
		if err != nil {
			return "", ""
		}
		txID := ""
		if updated.LastTransactionID != nil {
			txID = updated.LastTransactionID.String()
		}
		return updated.Balance, txID
	}
	creditAbs := netCharge.Abs()
	refundID := uuid.New()
	updated, err := walletRepo.AdjustBalanceTx(ctx, tx, wallet.ID, userID, creditAbs.String(), "refund",
		fmt.Sprintf("Platform subscription proration credit from %s", oldPlan.DisplayName), nil, "sub-refund-"+refundID.String())
	if err != nil {
		return "", ""
	}
	txID := ""
	if updated.LastTransactionID != nil {
		txID = updated.LastTransactionID.String()
	}
	return updated.Balance, txID
}

// GetMySubscription returns the user's active subscription and resolved plan.
// If no active subscription exists, returns the free plan as default.
func (s *SubscriptionService) GetMySubscription(ctx context.Context, userID uuid.UUID) (*model.UserPlatformSubscription, *model.SubscriptionPlan, error) {
	sub, err := s.repo.GetActiveSubscription(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	if sub != nil {
		plan, err := s.repo.GetPlanByID(ctx, sub.PlanID)
		if err != nil {
			return nil, nil, err
		}
		return sub, plan, nil
	}
	// No active subscription — return free plan as default.
	plan, err := s.repo.GetPlanByName(ctx, "free")
	if err != nil {
		return nil, nil, err
	}
	return nil, plan, nil
}
