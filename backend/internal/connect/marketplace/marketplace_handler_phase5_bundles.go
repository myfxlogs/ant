package marketplace

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/marketplace"

	"github.com/google/uuid"
)

// ── Phase 5.2: Strategy Bundle Handlers ───────────────────────────────────────

func (s *MarketplaceServer) CreateBundle(
	ctx context.Context,
	req *connect.Request[antv1.CreateBundleRequest],
) (*connect.Response[antv1.CreateBundleResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	m := req.Msg
	if len(m.GetStrategyIds()) < 2 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("at least 2 strategies required"))
	}

	bundleID, err := s.svc.CreateBundle(ctx, userID, m.GetTitle(), m.GetDescription(),
		m.GetPriceModel(), m.GetPriceAmount(), m.GetStrategyIds(), m.GetPlatformFeeRate())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.CreateBundleResponse{BundleId: bundleID}), nil
}

func (s *MarketplaceServer) ListBundles(
	ctx context.Context,
	req *connect.Request[antv1.ListBundlesRequest],
) (*connect.Response[antv1.ListBundlesResponse], error) {
	m := req.Msg
	bundles, total, err := s.svc.ListBundles(ctx, m.GetPublisherId(), int(m.GetLimit()), int(m.GetOffset()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	items := make([]*antv1.BundleInfo, 0, len(bundles))
	for _, b := range bundles {
		items = append(items, bundleToProto(&b))
	}

	return connect.NewResponse(&antv1.ListBundlesResponse{
		Bundles: items,
		Total:   int32(total),
	}), nil
}

func (s *MarketplaceServer) GetBundle(
	ctx context.Context,
	req *connect.Request[antv1.GetBundleRequest],
) (*connect.Response[antv1.GetBundleResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	b, err := s.svc.GetBundle(ctx, req.Msg.GetBundleId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	return connect.NewResponse(&antv1.GetBundleResponse{
		Bundle: bundleToProto(b),
	}), nil
}

func (s *MarketplaceServer) PurchaseBundle(
	ctx context.Context,
	req *connect.Request[antv1.PurchaseBundleRequest],
) (*connect.Response[antv1.PurchaseBundleResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	result, err := s.svc.PurchaseBundle(ctx, userID, req.Msg.GetBundleId(), req.Msg.GetIdempotencyKey())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.PurchaseBundleResponse{
		SubscriptionId: result.SubscriptionID,
		TransactionId:  result.TransactionID,
		AmountCharged:  result.AmountCharged,
		BalanceAfter:   result.BalanceAfter,
	}), nil
}

func (s *MarketplaceServer) DeleteBundle(
	ctx context.Context,
	req *connect.Request[antv1.DeleteBundleRequest],
) (*connect.Response[antv1.DeleteBundleResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	isAdmin := false
	if uid, err := uuid.Parse(userID); err == nil && s.admin != nil {
		isAdmin, _ = s.admin.IsAdmin(ctx, uid)
	}
	if err := s.svc.DeleteBundle(ctx, req.Msg.GetBundleId(), userID, isAdmin); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.DeleteBundleResponse{Success: true}), nil
}

func bundleToProto(b *marketplace.Bundle) *antv1.BundleInfo {
	pb := &antv1.BundleInfo{
		Id:              b.ID,
		Title:           b.Title,
		Description:     b.Description,
		PublisherId:     b.PublisherID,
		PriceModel:      b.PriceModel,
		PriceAmount:     b.PriceAmount.String(),
		PlatformFeeRate: b.PlatformFeeRate.String(),
		TotalPurchases:  b.TotalPurchases,
		CreatedAtMs:     b.CreatedAt.UnixMilli(),
	}
	for _, item := range b.Items {
		pb.Items = append(pb.Items, &antv1.BundleItem{
			StrategyId:  item.StrategyID,
			SortOrder:   item.SortOrder,
			Title:       item.Title,
			PriceAmount: item.PriceAmount,
		})
	}
	return pb
}

// ── Phase 5.3: Tiered Fee Rate Handlers ───────────────────────────────────────

func (s *MarketplaceServer) ListFeeTiers(
	ctx context.Context,
	_ *connect.Request[emptypb.Empty],
) (*connect.Response[antv1.ListFeeTiersResponse], error) {
	tiers, err := s.svc.ListFeeTiers(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	items := make([]*antv1.FeeTierInfo, 0, len(tiers))
	for _, t := range tiers {
		items = append(items, &antv1.FeeTierInfo{
			Id:            t.ID,
			TierName:      t.TierName,
			MinSalesCount: t.MinSalesCount,
			FeeRate:       t.FeeRate.String(),
			SortOrder:     t.SortOrder,
			Enabled:       t.Enabled,
		})
	}

	return connect.NewResponse(&antv1.ListFeeTiersResponse{Tiers: items}), nil
}

func (s *MarketplaceServer) UpdateFeeTier(
	ctx context.Context,
	req *connect.Request[antv1.UpdateFeeTierRequest],
) (*connect.Response[antv1.UpdateFeeTierResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	if s.admin == nil {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("admin only"))
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid user id"))
	}
	isAdmin, _ := s.admin.IsAdmin(ctx, uid)
	if !isAdmin {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("admin only"))
	}

	m := req.Msg
	if err := s.svc.UpdateFeeTier(ctx, m.GetId(), m.GetFeeRate(), m.GetMinSalesCount(), m.GetEnabled()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.UpdateFeeTierResponse{Success: true}), nil
}

func (s *MarketplaceServer) GetProviderFeeTier(
	ctx context.Context,
	req *connect.Request[antv1.GetProviderFeeTierRequest],
) (*connect.Response[antv1.GetProviderFeeTierResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	// Only allow querying your own tier (or admin can query any).
	requestedID := req.Msg.GetPublisherId()
	if requestedID == "" {
		requestedID = userID
	}
	actorID, _ := uuid.Parse(userID)
	isAdmin := false
	if s.admin != nil {
		isAdmin, _ = s.admin.IsAdmin(ctx, actorID)
	}
	if requestedID != userID && !isAdmin {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("can only view your own fee tier"))
	}

	result, err := s.svc.GetProviderFeeTierWithStats(ctx, requestedID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.GetProviderFeeTierResponse{
		TierName:        result.Tier.TierName,
		FeeRate:         result.Tier.FeeRate.String(),
		MinSalesCount:   result.Tier.MinSalesCount,
		CurrentSales:    result.CurrentSales,
		NextTierMinSales: result.NextTierMinSales,
	}), nil
}
