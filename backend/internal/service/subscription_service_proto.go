package service

import (
	"context"
	"fmt"

	protopkg "google.golang.org/protobuf/encoding/protojson"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/model"
)

// UserSubscriptionInfo holds the resolved subscription + plan for proto response.
type UserSubscriptionInfo struct {
	Subscription *antv1.UserSubscription
	Plan         *antv1.Plan
}

// ListPlansProto returns all active plans as proto messages.
func (s *SubscriptionService) ListPlansProto(ctx context.Context) ([]*antv1.Plan, error) {
	plans, err := s.repo.ListPlans(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*antv1.Plan, len(plans))
	for i, p := range plans {
		out[i] = planToProto(p)
	}
	return out, nil
}

// GetMySubscriptionProto returns the user's active subscription and plan as proto messages.
func (s *SubscriptionService) GetMySubscriptionProto(ctx context.Context, userID uuid.UUID) (*UserSubscriptionInfo, error) {
	sub, plan, err := s.GetMySubscription(ctx, userID)
	if err != nil {
		return nil, err
	}
	info := &UserSubscriptionInfo{
		Plan: planToProto(plan),
	}
	if sub != nil {
		info.Subscription = SubToProto(sub, plan)
	}
	return info, nil
}

func planToProto(p *model.SubscriptionPlan) *antv1.Plan {
	if p == nil {
		return nil
	}
	return &antv1.Plan{
		Id:                    p.ID.String(),
		Name:                  p.Name,
		DisplayName:           p.DisplayName,
		PriceMonthly:          p.PriceMonthly,
		PriceYearly:           p.PriceYearly,
		MaxAiTokensMonthly:    int32(p.MaxAITokensMonthly),
		MaxStrategies:         int32(p.MaxStrategies),
		MaxBacktestsDaily:     int32(p.MaxBacktestsDaily),
		MaxLiveStrategies:     int32(p.MaxLiveStrategies),
		MaxSymbolsPerStrategy: int32(p.MaxSymbolsPerStrategy),
		CapabilityTier:        int32(p.CapabilityTier),
		Features:              parseFeaturesToStruct(p.Features),
		SortOrder:             int32(p.SortOrder),
		MaxMtAccounts:         int32(p.MaxMTAccounts),
	}
}

func SubToProto(s *model.UserPlatformSubscription, plan *model.SubscriptionPlan) *antv1.UserSubscription {
	if s == nil {
		return nil
	}
	sub := &antv1.UserSubscription{
		Id:           s.ID.String(),
		PlanId:       s.PlanID.String(),
		Status:       s.Status,
		BillingCycle: s.BillingCycle,
		AutoRenew:    s.AutoRenew,
	}
	if plan != nil {
		sub.PlanName = plan.Name
		sub.PlanDisplayName = plan.DisplayName
	}
	if !s.CurrentPeriodStart.IsZero() {
		sub.CurrentPeriodStart = timestamppb.New(s.CurrentPeriodStart)
	}
	if !s.CurrentPeriodEnd.IsZero() {
		sub.CurrentPeriodEnd = timestamppb.New(s.CurrentPeriodEnd)
	}
	if s.CancelledAt != nil && !s.CancelledAt.IsZero() {
		sub.CancelledAt = timestamppb.New(*s.CancelledAt)
	}
	if !s.CreatedAt.IsZero() {
		sub.CreatedAt = timestamppb.New(s.CreatedAt)
	}
	return sub
}

// EnsureFreeSubscription creates a free-tier subscription for users who don't have one.
// Called during registration to ensure all users have a plan record.
func (s *SubscriptionService) EnsureFreeSubscription(ctx context.Context, userID uuid.UUID) error {
	freePlan, err := s.repo.GetPlanByName(ctx, "free")
	if err != nil {
		return fmt.Errorf("ensure free subscription: get free plan: %w", err)
	}
	if freePlan == nil {
		return nil
	}
	_, err = s.subscribeFree(ctx, userID, freePlan, billingCycleMonthly)
	return err
}

// GetUsageSummaryProto returns aggregated usage data for the current month.
func (s *SubscriptionService) GetUsageSummaryProto(ctx context.Context, userID uuid.UUID) (*antv1.UsageSummary, *antv1.Plan, error) {
	_, plan, err := s.GetMySubscription(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	pbPlan := planToProto(plan)

	summary := &antv1.UsageSummary{
		AiTokensLimit:     int32(plan.MaxAITokensMonthly),
		MaxStrategies:     int32(plan.MaxStrategies),
		MaxLiveStrategies: int32(plan.MaxLiveStrategies),
	}

	// AI token usage.
	if s.tokenUsageRepo != nil {
		featureTokens, err := s.tokenUsageRepo.MonthlySummary(ctx, userID)
		if err != nil {
			s.log.Warn("GetUsageSummary: token summary failed", zap.Error(err))
		} else {
			total := 0
			ft := make(map[string]int32, len(featureTokens))
			for k, v := range featureTokens {
				ft[k] = int32(v)
				total += v
			}
			summary.AiTokensUsed = int32(total)
			summary.TokensByFeature = ft
		}
		monthlyCost, err := s.tokenUsageRepo.MonthlyCost(ctx, userID)
		if err != nil {
			s.log.Warn("GetUsageSummary: monthly cost failed", zap.Error(err))
		} else {
			summary.AiCost = monthlyCost
		}
	}

	// Strategy runtime.
	if s.runRepo != nil {
		minutes, err := s.runRepo.MonthlyRuntimeMinutes(ctx, userID)
		if err != nil {
			s.log.Warn("GetUsageSummary: runtime minutes failed", zap.Error(err))
		} else {
			summary.StrategyRuntimeMinutes = int32(minutes)
		}
		active, err := s.runRepo.CountActiveByUser(ctx, userID)
		if err != nil {
			s.log.Warn("GetUsageSummary: active count failed", zap.Error(err))
		} else {
			summary.ActiveStrategies = int32(active)
		}
	}

	// Wallet balance.
	wallet, err := s.walletSvc.GetOrCreateWallet(ctx, userID)
	if err != nil {
		s.log.Warn("GetUsageSummary: wallet lookup failed", zap.Error(err))
	} else {
		summary.WalletBalance = wallet.Balance
	}

	return summary, pbPlan, nil
}

// parseFeaturesToStruct converts a JSON string (from DB JSONB) to *structpb.Struct.
func parseFeaturesToStruct(s string) *structpb.Struct {
	if s == "" || s == "{}" {
		st, _ := structpb.NewStruct(map[string]any{})
		return st
	}
	st := &structpb.Struct{}
	if err := protopkg.Unmarshal([]byte(s), st); err != nil {
		st, _ := structpb.NewStruct(map[string]any{})
		return st
	}
	return st
}
