package subscription

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/service"
)

// subscriptionSvc is the local interface for the subscription service.
type subscriptionSvc interface {
	Subscribe(ctx context.Context, userID uuid.UUID, planName, billingCycle string, autoRenew bool) (*service.SubscribeResult, error)
	CancelSubscription(ctx context.Context, userID uuid.UUID) error
	ChangePlan(ctx context.Context, userID uuid.UUID, newPlanName, billingCycle string) (*service.SubscribeResult, error)
}

// planReader is the interface for reading plans and user subscriptions as proto.
type planReader interface {
	ListPlansProto(ctx context.Context) ([]*antv1.Plan, error)
	GetMySubscriptionProto(ctx context.Context, userID uuid.UUID) (*service.UserSubscriptionInfo, error)
	GetUsageSummaryProto(ctx context.Context, userID uuid.UUID) (*antv1.UsageSummary, *antv1.Plan, error)
}

// Server implements ant.v1.SubscriptionServiceHandler.
type Server struct {
	svc        subscriptionSvc
	planReader planReader
	log        *zap.Logger
}

var _ antv1c.SubscriptionServiceHandler = (*Server)(nil)

func NewServer(svc subscriptionSvc, planReader planReader, log *zap.Logger) *Server {
	return &Server{svc: svc, planReader: planReader, log: log}
}

func (s *Server) ListPlans(ctx context.Context, req *connect.Request[antv1.ListPlansRequest]) (*connect.Response[antv1.ListPlansResponse], error) {
	plans, err := s.planReader.ListPlansProto(ctx)
	if err != nil {
		s.log.Error("ListPlans", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.ListPlansResponse{Plans: plans}), nil
}

func (s *Server) GetMySubscription(ctx context.Context, req *connect.Request[antv1.GetMySubscriptionRequest]) (*connect.Response[antv1.GetMySubscriptionResponse], error) {
	uid, err := parseUserID(ctx)
	if err != nil {
		return nil, err
	}
	info, err := s.planReader.GetMySubscriptionProto(ctx, uid)
	if err != nil {
		s.log.Error("GetMySubscription", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.GetMySubscriptionResponse{
		Subscription: info.Subscription,
		Plan:         info.Plan,
	}), nil
}

func (s *Server) Subscribe(ctx context.Context, req *connect.Request[antv1.SubscribePlanRequest]) (*connect.Response[antv1.SubscribePlanResponse], error) {
	uid, err := parseUserID(ctx)
	if err != nil {
		return nil, err
	}
	if req.Msg.PlanName == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("plan_name is required"))
	}
	result, err := s.svc.Subscribe(ctx, uid, req.Msg.PlanName, req.Msg.BillingCycle, req.Msg.AutoRenew)
	if err != nil {
		return nil, mapSubscribeError(err)
	}
	return connect.NewResponse(&antv1.SubscribePlanResponse{
		Subscription:  service.SubToProto(result.Subscription, result.Plan),
		TransactionId: result.TransactionID,
		AmountCharged: result.AmountCharged,
		BalanceAfter:  result.BalanceAfter,
	}), nil
}

func (s *Server) CancelSubscription(ctx context.Context, req *connect.Request[antv1.CancelSubscriptionRequest]) (*connect.Response[antv1.CancelSubscriptionResponse], error) {
	uid, err := parseUserID(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.svc.CancelSubscription(ctx, uid); err != nil {
		return nil, mapSubscribeError(err)
	}
	return connect.NewResponse(&antv1.CancelSubscriptionResponse{}), nil
}

func (s *Server) ChangePlan(ctx context.Context, req *connect.Request[antv1.ChangePlanRequest]) (*connect.Response[antv1.ChangePlanResponse], error) {
	uid, err := parseUserID(ctx)
	if err != nil {
		return nil, err
	}
	if req.Msg.NewPlanName == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("new_plan_name is required"))
	}
	result, err := s.svc.ChangePlan(ctx, uid, req.Msg.NewPlanName, req.Msg.BillingCycle)
	if err != nil {
		return nil, mapSubscribeError(err)
	}
	return connect.NewResponse(&antv1.ChangePlanResponse{
		Subscription:  service.SubToProto(result.Subscription, result.Plan),
		TransactionId: result.TransactionID,
		AmountCharged: result.AmountCharged,
		BalanceAfter:  result.BalanceAfter,
	}), nil
}

func (s *Server) GetUsageSummary(ctx context.Context, req *connect.Request[antv1.GetUsageSummaryRequest]) (*connect.Response[antv1.GetUsageSummaryResponse], error) {
	uid, err := parseUserID(ctx)
	if err != nil {
		return nil, err
	}
	summary, plan, err := s.planReader.GetUsageSummaryProto(ctx, uid)
	if err != nil {
		s.log.Error("GetUsageSummary", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.GetUsageSummaryResponse{
		Summary: summary,
		Plan:    plan,
	}), nil
}

// ── helpers ──

func parseUserID(ctx context.Context) (uuid.UUID, error) {
	uid := interceptor.GetUserID(ctx)
	if uid == "" {
		return uuid.Nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	id, err := uuid.Parse(uid)
	if err != nil {
		return uuid.Nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid user id"))
	}
	return id, nil
}

func mapSubscribeError(err error) error {
	switch {
	case errors.Is(err, service.ErrPlanNotFound):
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("plan not found"))
	case errors.Is(err, service.ErrNoActiveSubscription):
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("no active subscription"))
	default:
		var ibe *service.InsufficientBalanceError
		if errors.As(err, &ibe) {
			return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("insufficient balance: have %s, need %s", ibe.Balance, ibe.Cost))
		}
		return connect.NewError(connect.CodeInternal, err)
	}
}
