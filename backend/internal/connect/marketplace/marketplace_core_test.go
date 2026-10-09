package marketplace

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/marketplace"
)

func TestMarketplace_PublishStrategy_Success(t *testing.T) {
	t.Parallel()
	svc := &stubMarketplaceSvc{publishID: "pub-1"}
	h := testMarketplaceHandler(svc)
	ctx := context.WithValue(context.Background(), interceptor.UserIDKey, "00000000-0000-0000-0000-000000000001")

	resp, err := h.PublishStrategy(ctx, connect.NewRequest(&antv1.PublishStrategyRequest{
		StrategyId: "s1", Title: "My Strategy",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.PublishId != "pub-1" {
		t.Errorf("expected pub-1, got %s", resp.Msg.PublishId)
	}
}

func TestMarketplace_PublishStrategy_Error(t *testing.T) {
	t.Parallel()
	svc := &stubMarketplaceSvc{err: errors.New("db down")}
	h := testMarketplaceHandler(svc)
	ctx := context.WithValue(context.Background(), interceptor.UserIDKey, "00000000-0000-0000-0000-000000000001")

	_, err := h.PublishStrategy(ctx, connect.NewRequest(&antv1.PublishStrategyRequest{}))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMarketplace_ListPublished_Empty(t *testing.T) {
	t.Parallel()
	h := testMarketplaceHandler(&stubMarketplaceSvc{})

	resp, err := h.ListPublished(context.Background(), connect.NewRequest(&antv1.ListPublishedRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.Strategies) != 0 {
		t.Fatalf("expected 0 strategies, got %d", len(resp.Msg.Strategies))
	}
}

func TestMarketplace_ListPublished_WithItems(t *testing.T) {
	t.Parallel()
	now := time.Now()
	svc := &stubMarketplaceSvc{
		published: []marketplace.PublishedStrategy{{
			PublishID: "p1", StrategyID: "s1", StrategyName: "Test Strategy",
			PublisherUserID: "u1", PublishedAt: now, Title: "Best", Description: "Desc",
			PriceModel: "free", AssetClass: "forex",
		}},
	}
	h := testMarketplaceHandler(svc)

	resp, err := h.ListPublished(context.Background(), connect.NewRequest(&antv1.ListPublishedRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.Strategies) != 1 {
		t.Fatalf("expected 1, got %d", len(resp.Msg.Strategies))
	}
	s := resp.Msg.Strategies[0]
	if s.PublishId != "p1" || s.StrategyName != "Test Strategy" {
		t.Errorf("unexpected strategy: %+v", s)
	}
}

func TestMarketplace_RateStrategy(t *testing.T) {
	t.Parallel()
	svc := &stubMarketplaceSvc{avgRating: 4.5, rateCount: 10}
	h := testMarketplaceHandler(svc)
	ctx := context.WithValue(context.Background(), interceptor.UserIDKey, "00000000-0000-0000-0000-000000000001")

	resp, err := h.RateStrategy(ctx, connect.NewRequest(&antv1.RateStrategyRequest{
		StrategyId: "s1", Rating: 5,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.AvgRating != 4.5 || resp.Msg.RatingCount != 10 {
		t.Errorf("expected 4.5/10, got %.1f/%d", resp.Msg.AvgRating, resp.Msg.RatingCount)
	}
}

func TestMarketplace_Subscribe_Success(t *testing.T) {
	t.Parallel()
	h := testMarketplaceHandler(&stubMarketplaceSvc{})
	ctx := context.WithValue(context.Background(), interceptor.UserIDKey, "00000000-0000-0000-0000-000000000001")

	resp, err := h.Subscribe(ctx, connect.NewRequest(&antv1.SubscribeRequest{
		PublisherUserId: "u2", StrategyId: "s1",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.SubscriptionId != "sub-1" {
		t.Errorf("expected sub-1, got %s", resp.Msg.SubscriptionId)
	}
}

func TestMarketplace_SetPricing_AdminRequired(t *testing.T) {
	t.Parallel()
	svc := &stubMarketplaceSvc{}
	h := &MarketplaceServer{svc: svc, admin: &stubAdminChecker{isAdmin: false}, log: zap.NewNop()}

	_, err := h.SetStrategyPricing(context.Background(), connect.NewRequest(&antv1.SetStrategyPricingRequest{}))
	if err == nil {
		t.Fatal("expected permission denied for non-admin")
	}
}

func TestMarketplace_ListSubscriptions(t *testing.T) {
	t.Parallel()
	svc := &stubMarketplaceSvc{
		subs: []marketplace.SubscriptionItem{
			{SubscriptionID: "sub-1", StrategyID: "s1", StrategyTitle: "Golden Cross", Active: true},
			{SubscriptionID: "sub-2", StrategyID: "s2", Active: false},
		},
	}
	h := testMarketplaceHandler(svc)
	ctx := context.WithValue(context.Background(), interceptor.UserIDKey, "00000000-0000-0000-0000-000000000001")

	resp, err := h.ListSubscriptions(ctx, connect.NewRequest(&antv1.ListSubscriptionsRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.Subscriptions) != 2 {
		t.Fatalf("expected 2, got %d", len(resp.Msg.Subscriptions))
	}
	if resp.Msg.Subscriptions[0].StrategyTitle != "Golden Cross" {
		t.Errorf("expected StrategyTitle 'Golden Cross', got %q", resp.Msg.Subscriptions[0].StrategyTitle)
	}
}

func TestMarketplace_Unsubscribe_Error(t *testing.T) {
	t.Parallel()
	svc := &stubMarketplaceSvc{err: errors.New("not found")}
	h := testMarketplaceHandler(svc)
	ctx := context.WithValue(context.Background(), interceptor.UserIDKey, "00000000-0000-0000-0000-000000000001")

	_, err := h.Unsubscribe(ctx, connect.NewRequest(&antv1.UnsubscribeRequest{}))
	if err == nil {
		t.Fatal("expected error from unsubscribe")
	}
}

func TestMarketplace_CommentOnStrategy(t *testing.T) {
	t.Parallel()
	h := testMarketplaceHandler(&stubMarketplaceSvc{})
	ctx := context.WithValue(context.Background(), interceptor.UserIDKey, "00000000-0000-0000-0000-000000000001")

	resp, err := h.CommentOnStrategy(ctx, connect.NewRequest(&antv1.CommentOnStrategyRequest{
		StrategyId: "s1", Content: "Great!",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Id != "comment-1" {
		t.Errorf("expected comment-1, got %s", resp.Msg.Id)
	}
}

// ── Auth validation tests ──
