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
	"alphaforge/internal/repository"
	"alphaforge/internal/service"
)

// subscriptionSvc is the local interface for the subscription service.
type subscriptionSvc interface {
	Subscribe(ctx context.Context, userID uuid.UUID, planName, billingCycle string, autoRenew bool) (*service.SubscribeResult, error)
	CancelSubscription(ctx context.Context, userID uuid.UUID) error
	ChangePlan(ctx context.Context, userID uuid.UUID, newPlanName, billingCycle string) (*service.SubscribeResult, error)
}

type boundAccountSvc interface {
	ListBoundAccounts(ctx context.Context, userID uuid.UUID) ([]repository.BoundAccountRow, error)
	UnbindAccount(ctx context.Context, userID, accountID uuid.UUID) error
	GetAccountLimit(ctx context.Context, userID uuid.UUID) (int, error)
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
	boundSvc   boundAccountSvc
	log        *zap.Logger
}

var _ antv1c.SubscriptionServiceHandler = (*Server)(nil)

func NewServer(svc subscriptionSvc, planReader planReader, boundSvc boundAccountSvc, log *zap.Logger) *Server {
	return &Server{svc: svc, planReader: planReader, boundSvc: boundSvc, log: log}
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

func (s *Server) ListBoundAccounts(ctx context.Context, _ *connect.Request[antv1.ListBoundAccountsRequest]) (*connect.Response[antv1.ListBoundAccountsResponse], error) {
	uid, err := parseUserID(ctx)
	if err != nil {
		return nil, err
	}
	accounts, err := s.boundSvc.ListBoundAccounts(ctx, uid)
	if err != nil {
		s.log.Error("ListBoundAccounts", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	limit, err := s.boundSvc.GetAccountLimit(ctx, uid)
	if err != nil {
		s.log.Error("ListBoundAccounts: get limit", zap.Error(err))
		limit = 0
	}
	items := make([]*antv1.BoundAccount, len(accounts))
	for i, a := range accounts {
		items[i] = &antv1.BoundAccount{
			MtAccountId:   a.MTAccountID.String(),
			Login:         a.Login,
			Broker:        a.Broker,
			Server:        a.Server,
			MtType:        a.MTType,
			AccountStatus: a.AccountStatus,
			BoundAt:       a.BoundAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	return connect.NewResponse(&antv1.ListBoundAccountsResponse{Accounts: items, MaxAccounts: int32(limit)}), nil
}

func (s *Server) UnbindAccount(ctx context.Context, req *connect.Request[antv1.UnbindAccountRequest]) (*connect.Response[antv1.UnbindAccountResponse], error) {
	uid, err := parseUserID(ctx)
	if err != nil {
		return nil, err
	}
	accountID, err := uuid.Parse(req.Msg.MtAccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid mt_account_id"))
	}
	if err := s.boundSvc.UnbindAccount(ctx, uid, accountID); err != nil {
		s.log.Error("UnbindAccount", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.UnbindAccountResponse{}), nil
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
