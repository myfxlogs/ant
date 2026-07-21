package marketplace

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"
)

// StartTrial starts a 7-day free trial for a strategy (Phase 3.2).
func (s *MarketplaceServer) StartTrial(
	ctx context.Context,
	req *connect.Request[antv1.StartTrialRequest],
) (*connect.Response[antv1.StartTrialResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}

	if req.Msg.StrategyId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("strategy_id is required"))
	}

	trialID, expiresAt, alreadyTried, err := s.svc.StartTrial(ctx, userID, req.Msg.StrategyId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("start trial: %w", err))
	}

	return connect.NewResponse(&antv1.StartTrialResponse{
		TrialId:      trialID,
		ExpiresAtMs:  expiresAt.UnixMilli(),
		AlreadyTried: alreadyTried,
	}), nil
}
