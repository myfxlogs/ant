package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/model"
	"alphaforge/internal/repository"
)

var (
	ErrPlanNotFound         = errors.New("subscription plan not found")
	ErrNoActiveSubscription = errors.New("no active subscription")
)

// SubscriptionService manages platform subscription lifecycle.
// Uses WalletService for billing and SubscriptionRepository for persistence.
type SubscriptionService struct {
	repo      *repository.SubscriptionRepository
	walletSvc *WalletService
	pg        *pgxpool.Pool
	log       *zap.Logger

	// Optional dependencies for usage summary (injected after construction).
	tokenUsageRepo *repository.AITokenUsageRepository
	runRepo        *repository.StrategyRunRepository
}

func NewSubscriptionService(repo *repository.SubscriptionRepository, walletSvc *WalletService, pg *pgxpool.Pool, log *zap.Logger) *SubscriptionService {
	return &SubscriptionService{repo: repo, walletSvc: walletSvc, pg: pg, log: log}
}

// SetUsageRepos injects repositories needed for GetUsageSummary.
func (s *SubscriptionService) SetUsageRepos(tokenUsageRepo *repository.AITokenUsageRepository, runRepo *repository.StrategyRunRepository) {
	s.tokenUsageRepo = tokenUsageRepo
	s.runRepo = runRepo
}

// SubscribeResult holds the outcome of a subscribe or change-plan operation.
type SubscribeResult struct {
	Subscription  *model.UserPlatformSubscription
	Plan          *model.SubscriptionPlan
	TransactionID string
	AmountCharged string
	BalanceAfter  string
}

// Subscribe creates or replaces the user's active subscription, charging the wallet for paid plans.
func (s *SubscriptionService) Subscribe(ctx context.Context, userID uuid.UUID, planName, billingCycle string, autoRenew bool) (*SubscribeResult, error) {
	plan, err := s.repo.GetPlanByName(ctx, planName)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, ErrPlanNotFound
	}

	if billingCycle != "monthly" && billingCycle != "yearly" {
		billingCycle = "monthly"
	}

	// Determine price based on billing cycle.
	priceStr := plan.PriceMonthly
	if billingCycle == "yearly" {
		priceStr = plan.PriceYearly
	}
	price, err := decimal.NewFromString(priceStr)
	if err != nil {
		return nil, fmt.Errorf("subscription: parse price: %w", err)
	}

	// Free plan: no charge, just create a subscription record.
	if price.LessThanOrEqual(decimal.Zero) {
		return s.subscribeFree(ctx, userID, plan, billingCycle)
	}

	// Paid plan: charge wallet in a transaction with subscription creation.
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("subscription: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Lock and deactivate existing subscription within the transaction.
	existing, err := s.repo.GetActiveSubscriptionTx(ctx, tx, userID)
	if err != nil {
		return nil, fmt.Errorf("subscription: check existing: %w", err)
	}
	if existing != nil {
		if err := s.repo.DeactivateSubscription(ctx, tx, existing.ID); err != nil {
			return nil, err
		}
	}

	// Charge wallet.
	walletRepo := s.walletSvc.Repo()
	wallet, err := walletRepo.GetByUserIDTx(ctx, tx, userID)
	if err != nil {
		return nil, fmt.Errorf("subscription: get wallet: %w", err)
	}
	if wallet == nil {
		return nil, ErrWalletNotFound
	}

	balance, err := decimal.NewFromString(wallet.Balance)
	if err != nil {
		return nil, fmt.Errorf("subscription: parse balance: %w", err)
	}
	if balance.LessThan(price) {
		return nil, &InsufficientBalanceError{Balance: wallet.Balance, Cost: priceStr}
	}

	subID := uuid.New()
	updated, err := walletRepo.AdjustBalanceTx(ctx, tx, wallet.ID, userID, price.Neg().String(), "purchase",
		fmt.Sprintf("Platform subscription: %s (%s)", plan.DisplayName, billingCycle), nil, "sub-"+subID.String())
	if err != nil {
		return nil, fmt.Errorf("subscription: charge wallet: %w", err)
	}

	// Create subscription record.
	now := time.Now().UTC()
	periodEnd := now.AddDate(0, 1, 0)
	if billingCycle == "yearly" {
		periodEnd = now.AddDate(1, 0, 0)
	}
	sub := &model.UserPlatformSubscription{
		ID:                 subID,
		UserID:             userID,
		PlanID:             plan.ID,
		Status:             "active",
		BillingCycle:       billingCycle,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   periodEnd,
		AutoRenew:          autoRenew,
	}
	if updated.LastTransactionID != nil {
		sub.WalletTransactionID = updated.LastTransactionID
	}
	if err := s.repo.CreateSubscription(ctx, tx, sub); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("subscription: commit: %w", err)
	}

	result := &SubscribeResult{
		Subscription:  sub,
		Plan:          plan,
		AmountCharged: priceStr,
		BalanceAfter:  updated.Balance,
	}
	if updated.LastTransactionID != nil {
		result.TransactionID = updated.LastTransactionID.String()
	}
	return result, nil
}

// subscribeFree creates or replaces a free-tier subscription without charging.
func (s *SubscriptionService) subscribeFree(ctx context.Context, userID uuid.UUID, plan *model.SubscriptionPlan, billingCycle string) (*SubscribeResult, error) {
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("subscription: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	existing, err := s.repo.GetActiveSubscriptionTx(ctx, tx, userID)
	if err != nil {
		return nil, fmt.Errorf("subscription: check existing: %w", err)
	}

	// If already on free plan, return it.
	if existing != nil {
		existingPlan, err := s.repo.GetPlanByID(ctx, existing.PlanID)
		if err != nil {
			return nil, fmt.Errorf("subscription: get existing plan: %w", err)
		}
		if existingPlan != nil && existingPlan.Name == "free" {
			return &SubscribeResult{Subscription: existing, Plan: existingPlan}, nil
		}
		// Deactivate paid subscription.
		if err := s.repo.DeactivateSubscription(ctx, tx, existing.ID); err != nil {
			return nil, err
		}
	}

	// Create free subscription.
	now := time.Now().UTC()
	sub := &model.UserPlatformSubscription{
		ID:                 uuid.New(),
		UserID:             userID,
		PlanID:             plan.ID,
		Status:             "active",
		BillingCycle:       billingCycle,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(100, 0, 0),
		AutoRenew:          false,
	}
	if err := s.repo.CreateSubscription(ctx, tx, sub); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("subscription: commit: %w", err)
	}
	return &SubscribeResult{Subscription: sub, Plan: plan}, nil
}

// CancelSubscription disables auto-renewal. The subscription remains active until period end.
func (s *SubscriptionService) CancelSubscription(ctx context.Context, userID uuid.UUID) error {
	sub, err := s.repo.GetActiveSubscription(ctx, userID)
	if err != nil {
		return err
	}
	if sub == nil {
		return ErrNoActiveSubscription
	}
	return s.repo.CancelSubscription(ctx, sub.ID)
}

// ChangePlan switches the user to a new plan with prorated billing.
func (s *SubscriptionService) ChangePlan(ctx context.Context, userID uuid.UUID, newPlanName, billingCycle string) (*SubscribeResult, error) {
	newPlan, err := s.repo.GetPlanByName(ctx, newPlanName)
	if err != nil {
		return nil, err
	}
	if newPlan == nil {
		return nil, ErrPlanNotFound
	}

	if billingCycle != "monthly" && billingCycle != "yearly" {
		billingCycle = "monthly"
	}

	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("subscription: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	existing, err := s.repo.GetActiveSubscriptionTx(ctx, tx, userID)
	if err != nil {
		return nil, fmt.Errorf("subscription: check existing: %w", err)
	}
	if existing == nil {
		_ = tx.Rollback(ctx)
		return s.Subscribe(ctx, userID, newPlanName, billingCycle, true)
	}

	oldPlan, err := s.repo.GetPlanByID(ctx, existing.PlanID)
	if err != nil {
		return nil, err
	}
	if oldPlan == nil {
		return nil, ErrPlanNotFound
	}

	// Calculate prorated credit for remaining time.
	now := time.Now().UTC()
	totalDuration := existing.CurrentPeriodEnd.Sub(existing.CurrentPeriodStart)
	remaining := existing.CurrentPeriodEnd.Sub(now)
	if remaining < 0 {
		remaining = 0
	}
	var creditAmount decimal.Decimal
	if totalDuration > 0 {
		prorationRatio := decimal.NewFromInt(remaining.Nanoseconds()).Div(decimal.NewFromInt(totalDuration.Nanoseconds()))
		oldPriceStr := oldPlan.PriceMonthly
		if existing.BillingCycle == "yearly" {
			oldPriceStr = oldPlan.PriceYearly
		}
		oldPrice, _ := decimal.NewFromString(oldPriceStr)
		creditAmount = oldPrice.Mul(prorationRatio)
	}

	// Calculate new plan charge.
	newPriceStr := newPlan.PriceMonthly
	if billingCycle == "yearly" {
		newPriceStr = newPlan.PriceYearly
	}
	newPrice, _ := decimal.NewFromString(newPriceStr)
	netCharge := newPrice.Sub(creditAmount)

	// Update subscription plan.
	periodEnd := now.AddDate(0, 1, 0)
	if billingCycle == "yearly" {
		periodEnd = now.AddDate(1, 0, 0)
	}
	if err := s.repo.UpdateSubscriptionPlan(ctx, tx, existing.ID, newPlan.ID, billingCycle, periodEnd); err != nil {
		return nil, err
	}

	var balanceAfter string
	var txID string
	if !netCharge.IsZero() {
		walletRepo := s.walletSvc.Repo()
		wallet, err := walletRepo.GetByUserIDTx(ctx, tx, userID)
		if err != nil {
			return nil, fmt.Errorf("subscription: get wallet: %w", err)
		}
		if wallet == nil {
			return nil, ErrWalletNotFound
		}

		if netCharge.GreaterThan(decimal.Zero) {
			balance, _ := decimal.NewFromString(wallet.Balance)
			if balance.LessThan(netCharge) {
				return nil, &InsufficientBalanceError{Balance: wallet.Balance, Cost: netCharge.String()}
			}
			subID := uuid.New()
			updated, err := walletRepo.AdjustBalanceTx(ctx, tx, wallet.ID, userID, netCharge.Neg().String(), "purchase",
				fmt.Sprintf("Platform subscription: %s (%s)", newPlan.DisplayName, billingCycle), nil, "sub-"+subID.String())
			if err != nil {
				return nil, fmt.Errorf("subscription: charge wallet: %w", err)
			}
			balanceAfter = updated.Balance
			if updated.LastTransactionID != nil {
				txID = updated.LastTransactionID.String()
			}
		} else {
			creditAbs := netCharge.Abs()
			refundID := uuid.New()
			updated, err := walletRepo.AdjustBalanceTx(ctx, tx, wallet.ID, userID, creditAbs.String(), "refund",
				fmt.Sprintf("Platform subscription proration credit from %s", oldPlan.DisplayName), nil, "sub-refund-"+refundID.String())
			if err != nil {
				return nil, fmt.Errorf("subscription: credit wallet: %w", err)
			}
			balanceAfter = updated.Balance
			if updated.LastTransactionID != nil {
				txID = updated.LastTransactionID.String()
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("subscription: commit: %w", err)
	}

	existing.PlanID = newPlan.ID
	existing.BillingCycle = billingCycle
	existing.CurrentPeriodEnd = periodEnd

	return &SubscribeResult{
		Subscription:  existing,
		Plan:          newPlan,
		AmountCharged: netCharge.String(),
		BalanceAfter:  balanceAfter,
		TransactionID: txID,
	}, nil
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
