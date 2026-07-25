package marketplace

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"
)

func (s *MarketplaceServer) RequestVerification(
	ctx context.Context,
	req *connect.Request[antv1.RequestVerificationRequest],
) (*connect.Response[antv1.RequestVerificationResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	requestID, status, err := s.svc.RequestVerification(ctx, userID, req.Msg.ProviderType, req.Msg.Note)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	return connect.NewResponse(&antv1.RequestVerificationResponse{
		RequestId: requestID,
		Status:    status,
	}), nil
}

func (s *MarketplaceServer) AdminProcessVerification(
	ctx context.Context,
	req *connect.Request[antv1.AdminProcessVerificationRequest],
) (*connect.Response[antv1.AdminProcessVerificationResponse], error) {
	adminUID, err := s.checkAdmin(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.svc.ProcessVerification(ctx, adminUID.String(), req.Msg.RequestId, req.Msg.Approve, req.Msg.Note); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.AdminProcessVerificationResponse{Success: true}), nil
}
