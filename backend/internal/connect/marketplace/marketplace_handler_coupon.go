package marketplace

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	antv1 "alphaforge/gen/proto/ant/v1"
)

func (s *MarketplaceServer) ValidateCoupon(
	ctx context.Context,
	req *connect.Request[antv1.ValidateCouponRequest],
) (*connect.Response[antv1.ValidateCouponResponse], error) {
	m := req.Msg
	if m.Code == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("code is required"))
	}

	result, err := s.svc.ValidateCoupon(ctx, m.Code, m.StrategyId, m.PurchaseAmount)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.ValidateCouponResponse{
		Valid:          result.Valid,
		DiscountType:   result.DiscountType,
		DiscountAmount: result.DiscountAmount.String(),
		FinalAmount:    result.FinalAmount.String(),
		ErrorMessage:   result.ErrorMessage,
	}), nil
}

func (s *MarketplaceServer) CreateCoupon(
	ctx context.Context,
	req *connect.Request[antv1.CreateCouponRequest],
) (*connect.Response[antv1.CreateCouponResponse], error) {
	adminUID, err := s.checkAdmin(ctx)
	if err != nil {
		return nil, err
	}
	adminID := adminUID.String()

	m := req.Msg
	if m.Code == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("code is required"))
	}

	id, err := s.svc.CreateCoupon(ctx, adminID, m.Code, m.DiscountType, m.DiscountValue, m.MinPurchaseAmount, m.MaxUses, m.ExpiresAt, m.ApplicableStrategyIds)
	if err != nil {
		return connect.NewResponse(&antv1.CreateCouponResponse{
			Success: false,
			Error:   err.Error(),
		}), nil
	}

	return connect.NewResponse(&antv1.CreateCouponResponse{
		Id:     id,
		Success: true,
	}), nil
}

func (s *MarketplaceServer) ListCoupons(
	ctx context.Context,
	req *connect.Request[antv1.ListCouponsRequest],
) (*connect.Response[antv1.ListCouponsResponse], error) {
	if _, err := s.checkAdmin(ctx); err != nil {
		return nil, err
	}

	rows, err := s.svc.ListCoupons(ctx, req.Msg.EnabledOnly)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	coupons := make([]*antv1.CouponInfo, 0, len(rows))
	for _, r := range rows {
		c := &antv1.CouponInfo{
			Id:                   r.ID,
			Code:                 r.Code,
			DiscountType:         r.DiscountType,
			DiscountValue:        r.DiscountValue.String(),
			MinPurchaseAmount:    r.MinPurchase.String(),
			MaxUses:              r.MaxUses,
			UsedCount:            r.UsedCount,
			Enabled:              r.Enabled,
			ApplicableStrategyIds: r.ApplicableStrategyIDs,
		}
		if r.ExpiresAt != nil {
			c.ExpiresAt = r.ExpiresAt.Format("2006-01-02T15:04:05Z")
		}
		coupons = append(coupons, c)
	}

	return connect.NewResponse(&antv1.ListCouponsResponse{
		Coupons: coupons,
	}), nil
}

func (s *MarketplaceServer) DisableCoupon(
	ctx context.Context,
	req *connect.Request[antv1.DisableCouponRequest],
) (*connect.Response[antv1.DisableCouponResponse], error) {
	if _, err := s.checkAdmin(ctx); err != nil {
		return nil, err
	}

	if err := s.svc.DisableCoupon(ctx, req.Msg.CouponId); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.DisableCouponResponse{
		Success: true,
	}), nil
}
