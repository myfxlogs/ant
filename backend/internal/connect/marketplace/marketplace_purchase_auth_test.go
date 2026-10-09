package marketplace

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/marketplace"
)

func TestRateStrategy_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := testMarketplaceHandler(&stubMarketplaceSvc{})
	_, err := h.RateStrategy(context.Background(), connect.NewRequest(&antv1.RateStrategyRequest{
		StrategyId: "s1", Rating: 5,
	}))
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeUnauthenticated {
		t.Fatalf("expected CodeUnauthenticated, got %v", err)
	}
}

func TestRateStrategy_ServiceError(t *testing.T) {
	t.Parallel()
	svc := &stubMarketplaceSvc{rateErr: errors.New("db error")}
	h := testMarketplaceHandler(svc)
	ctx := context.WithValue(context.Background(), interceptor.UserIDKey, "u1")
	_, err := h.RateStrategy(ctx, connect.NewRequest(&antv1.RateStrategyRequest{StrategyId: "s1", Rating: 5}))
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeInternal {
		t.Fatalf("expected CodeInternal, got %v", err)
	}
}

func TestCommentOnStrategy_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := testMarketplaceHandler(&stubMarketplaceSvc{})
	_, err := h.CommentOnStrategy(context.Background(), connect.NewRequest(&antv1.CommentOnStrategyRequest{
		StrategyId: "s1", Content: "good",
	}))
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeUnauthenticated {
		t.Fatalf("expected CodeUnauthenticated, got %v", err)
	}
}

func TestSubscribe_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := testMarketplaceHandler(&stubMarketplaceSvc{})
	_, err := h.Subscribe(context.Background(), connect.NewRequest(&antv1.SubscribeRequest{
		StrategyId: "s1", Kind: "free",
	}))
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeUnauthenticated {
		t.Fatalf("expected CodeUnauthenticated, got %v", err)
	}
}

func TestSubscribe_SelfSubscribe(t *testing.T) {
	t.Parallel()
	svc := &stubMarketplaceSvc{subscribeErr: errors.New("cannot subscribe to your own strategy")}
	h := testMarketplaceHandler(svc)
	ctx := context.WithValue(context.Background(), interceptor.UserIDKey, "u1")
	_, err := h.Subscribe(ctx, connect.NewRequest(&antv1.SubscribeRequest{StrategyId: "s1", Kind: "free"}))
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", err)
	}
}

func TestUnsubscribe_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := testMarketplaceHandler(&stubMarketplaceSvc{})
	_, err := h.Unsubscribe(context.Background(), connect.NewRequest(&antv1.UnsubscribeRequest{}))
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeUnauthenticated {
		t.Fatalf("expected CodeUnauthenticated, got %v", err)
	}
}

func TestPurchaseStrategy_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := testMarketplaceHandler(&stubMarketplaceSvc{})
	_, err := h.PurchaseStrategy(context.Background(), connect.NewRequest(&antv1.PurchaseStrategyRequest{}))
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeUnauthenticated {
		t.Fatalf("expected CodeUnauthenticated, got %v", err)
	}
}

func TestPurchaseStrategy_InsufficientBalance(t *testing.T) {
	t.Parallel()
	svc := &stubMarketplaceSvc{purchaseErr: errors.New("insufficient balance")}
	h := testMarketplaceHandler(svc)
	ctx := context.WithValue(context.Background(), interceptor.UserIDKey, "u1")
	_, err := h.PurchaseStrategy(ctx, connect.NewRequest(&antv1.PurchaseStrategyRequest{StrategyId: "s1"}))
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeFailedPrecondition {
		t.Fatalf("expected CodeFailedPrecondition, got %v", err)
	}
}

func TestPurchaseStrategy_AlreadySubscribed(t *testing.T) {
	t.Parallel()
	svc := &stubMarketplaceSvc{purchaseErr: errors.New("already subscribed")}
	h := testMarketplaceHandler(svc)
	ctx := context.WithValue(context.Background(), interceptor.UserIDKey, "u1")
	_, err := h.PurchaseStrategy(ctx, connect.NewRequest(&antv1.PurchaseStrategyRequest{StrategyId: "s1"}))
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeAlreadyExists {
		t.Fatalf("expected CodeAlreadyExists, got %v", err)
	}
}

func TestPurchaseStrategy_OwnStrategy(t *testing.T) {
	t.Parallel()
	svc := &stubMarketplaceSvc{purchaseErr: errors.New("cannot purchase your own strategy")}
	h := testMarketplaceHandler(svc)
	ctx := context.WithValue(context.Background(), interceptor.UserIDKey, "u1")
	_, err := h.PurchaseStrategy(ctx, connect.NewRequest(&antv1.PurchaseStrategyRequest{StrategyId: "s1"}))
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", err)
	}
}

func TestPurchaseStrategy_Subscription_Success(t *testing.T) {
	t.Parallel()
	svc := &stubMarketplaceSvc{
		purchaseResult: &marketplace.PurchaseResult{
			SubscriptionID: "sub-monthly",
			TransactionID:  "tx-monthly",
			AmountCharged:  "9.99",
			BalanceAfter:   "90.01",
		},
	}
	h := testMarketplaceHandler(svc)
	ctx := context.WithValue(context.Background(), interceptor.UserIDKey, "u1")
	resp, err := h.PurchaseStrategy(ctx, connect.NewRequest(&antv1.PurchaseStrategyRequest{StrategyId: "s1"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.SubscriptionId != "sub-monthly" || resp.Msg.AmountCharged != "9.99" {
		t.Fatalf("unexpected response: %+v", resp.Msg)
	}
}

func TestPublishStrategy_InvalidPriceModel(t *testing.T) {
	t.Parallel()
	svc := &stubMarketplaceSvc{err: errors.New("marketplace: unsupported price_model")}
	h := testMarketplaceHandler(svc)
	ctx := context.WithValue(context.Background(), interceptor.UserIDKey, "u1")
	_, err := h.PublishStrategy(ctx, connect.NewRequest(&antv1.PublishStrategyRequest{
		StrategyId: "s1", Title: "Bad", PriceModel: "monthly",
	}))
	if err == nil {
		t.Fatal("expected error for invalid price_model")
	}
}

func TestPurchaseStrategy_InvalidPriceModel(t *testing.T) {
	t.Parallel()
	svc := &stubMarketplaceSvc{purchaseErr: errors.New("marketplace: unsupported price_model")}
	h := testMarketplaceHandler(svc)
	ctx := context.WithValue(context.Background(), interceptor.UserIDKey, "u1")
	_, err := h.PurchaseStrategy(ctx, connect.NewRequest(&antv1.PurchaseStrategyRequest{StrategyId: "s1"}))
	if err == nil {
		t.Fatal("expected error for invalid price_model")
	}
}

func TestUnpublishStrategy_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := testMarketplaceHandler(&stubMarketplaceSvc{})
	_, err := h.UnpublishStrategy(context.Background(), connect.NewRequest(&antv1.UnpublishMarketStrategyRequest{}))
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeUnauthenticated {
		t.Fatalf("expected CodeUnauthenticated, got %v", err)
	}
}

func TestUnpublishStrategy_NotPublisher(t *testing.T) {
	t.Parallel()
	svc := &stubMarketplaceSvc{unpublishErr: errors.New("only the publisher can unpublish")}
	h := testMarketplaceHandler(svc)
	ctx := context.WithValue(context.Background(), interceptor.UserIDKey, "u1")
	_, err := h.UnpublishStrategy(ctx, connect.NewRequest(&antv1.UnpublishMarketStrategyRequest{StrategyId: "s1"}))
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied, got %v", err)
	}
}

func TestUnpublishStrategy_Success(t *testing.T) {
	t.Parallel()
	h := testMarketplaceHandler(&stubMarketplaceSvc{})
	ctx := context.WithValue(context.Background(), interceptor.UserIDKey, "u1")
	resp, err := h.UnpublishStrategy(ctx, connect.NewRequest(&antv1.UnpublishMarketStrategyRequest{StrategyId: "s1"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Status != "hidden" {
		t.Errorf("expected status 'hidden', got %s", resp.Msg.Status)
	}
}

func TestSetStrategyPricing_AdminSuccess(t *testing.T) {
	t.Parallel()
	svc := &stubMarketplaceSvc{}
	h := &MarketplaceServer{svc: svc, admin: &stubAdminChecker{isAdmin: true}, log: zap.NewNop()}
	ctx := context.WithValue(context.Background(), interceptor.UserIDKey, "550e8400-e29b-41d4-a716-446655440000")
	resp, err := h.SetStrategyPricing(ctx, connect.NewRequest(&antv1.SetStrategyPricingRequest{
		StrategyId: "s1", PriceModel: "paid", PriceAmount: "100", PlatformFeeRate: 0.1,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.PriceModel != "paid" {
		t.Errorf("expected paid, got %s", resp.Msg.PriceModel)
	}
}

func TestSetStrategyPricing_InvalidUserID(t *testing.T) {
	t.Parallel()
	h := &MarketplaceServer{svc: &stubMarketplaceSvc{}, admin: &stubAdminChecker{isAdmin: true}, log: zap.NewNop()}
	_, err := h.SetStrategyPricing(context.Background(), connect.NewRequest(&antv1.SetStrategyPricingRequest{}))
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeUnauthenticated {
		t.Fatalf("expected CodeUnauthenticated, got %v", err)
	}
}

func TestGetPublisherStats_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := testMarketplaceHandler(&stubMarketplaceSvc{})
	_, err := h.GetPublisherStats(context.Background(), connect.NewRequest(&antv1.GetPublisherStatsRequest{}))
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeUnauthenticated {
		t.Fatalf("expected CodeUnauthenticated, got %v", err)
	}
}

func TestGetPublisherStats_Success(t *testing.T) {
	t.Parallel()
	svc := &stubMarketplaceSvc{
		publisherStats: &marketplace.PublisherStats{
			TotalPublished:   5,
			TotalSubscribers: 42,
			TotalRevenue:     "1000.00",
			MonthlyRevenue:   "200.00",
			AvgRating:        4.2,
			TopStrategyID:    "strat-1",
			TopStrategyTitle: "Best EA",
			TopStrategySubs:  20,
		},
	}
	h := testMarketplaceHandler(svc)
	ctx := context.WithValue(context.Background(), interceptor.UserIDKey, "u1")
	resp, err := h.GetPublisherStats(ctx, connect.NewRequest(&antv1.GetPublisherStatsRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.TotalPublished != 5 {
		t.Errorf("expected 5, got %d", resp.Msg.TotalPublished)
	}
	if resp.Msg.TopStrategyTitle != "Best EA" {
		t.Errorf("expected 'Best EA', got %s", resp.Msg.TopStrategyTitle)
	}
}

func TestListSubscriptions_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := testMarketplaceHandler(&stubMarketplaceSvc{})
	_, err := h.ListSubscriptions(context.Background(), connect.NewRequest(&antv1.ListSubscriptionsRequest{}))
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeUnauthenticated {
		t.Fatalf("expected CodeUnauthenticated, got %v", err)
	}
}

// ── parseDecimal tests ──
