package marketplace

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"
)

func (s *MarketplaceServer) RequestRefund(
	ctx context.Context,
	req *connect.Request[antv1.RequestRefundRequest],
) (*connect.Response[antv1.RequestRefundResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	m := req.Msg
	if m.SubscriptionId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("subscription_id is required"))
	}
	if m.Reason == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("reason is required"))
	}

	refundID, err := s.svc.CreateRefundRequest(ctx, userID, m.SubscriptionId, m.Reason)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	return connect.NewResponse(&antv1.RequestRefundResponse{
		RefundId: refundID,
		Status:   "pending_review",
	}), nil
}

func (s *MarketplaceServer) AdminListRefundRequests(
	ctx context.Context,
	req *connect.Request[antv1.AdminListRefundRequestsRequest],
) (*connect.Response[antv1.AdminListRefundRequestsResponse], error) {
	adminUID, err := s.checkAdmin(ctx)
	if err != nil {
		return nil, err
	}
	_ = adminUID

	m := req.Msg
	rows, total, err := s.svc.ListRefundRequests(ctx, m.Status, int(m.Limit), int(m.Offset))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	items := make([]*antv1.RefundRequestItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, &antv1.RefundRequestItem{
			RefundId:       r.ID,
			UserId:         r.UserID,
			UserName:       r.UserName,
			SubscriptionId: r.SubscriptionID,
			StrategyTitle:  r.StrategyTitle,
			Amount:         r.Amount,
			Reason:         r.Reason,
			Status:         r.Status,
			CreatedAt:      r.CreatedAt.Format("2006-01-02T15:04:05Z"),
			ReviewedBy:     r.ReviewedBy,
			ReviewNote:     r.ReviewNote,
		})
	}

	return connect.NewResponse(&antv1.AdminListRefundRequestsResponse{
		Requests: items,
		Total:    int32(total),
	}), nil
}

func (s *MarketplaceServer) AdminProcessRefund(
	ctx context.Context,
	req *connect.Request[antv1.AdminProcessRefundRequest],
) (*connect.Response[antv1.AdminProcessRefundResponse], error) {
	adminUID, err := s.checkAdmin(ctx)
	if err != nil {
		return nil, err
	}
	adminID := adminUID.String()

	m := req.Msg
	if m.RefundId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("refund_id is required"))
	}

	if err := s.svc.ProcessRefundRequest(ctx, adminID, m.RefundId, m.Approve, m.ReviewNote); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.AdminProcessRefundResponse{
		Success: true,
	}), nil
}
