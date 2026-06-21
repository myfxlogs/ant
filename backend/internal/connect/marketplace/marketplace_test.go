package marketplace

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/marketplace"
)

// ── Stubs ──

type stubMarketplaceSvc struct {
	published []marketplace.PublishedStrategy
	ratings   []marketplace.RatingItem
	avgRating float64
	rateCount int32
	comments  []marketplace.CommentItem
	subs      []marketplace.SubscriptionItem
	publishID string
	err       error
}

func (s *stubMarketplaceSvc) Publish(_ context.Context, _ marketplace.PublishParams) (string, error) {
	return s.publishID, s.err
}
func (s *stubMarketplaceSvc) ListPublished(_ context.Context, _ string, _ int, _, _, _ string) ([]marketplace.PublishedStrategy, error) {
	return s.published, s.err
}
func (s *stubMarketplaceSvc) Rate(_ context.Context, _, _ string, _ int32) (float64, int32, error) {
	return s.avgRating, s.rateCount, s.err
}
func (s *stubMarketplaceSvc) ListRatings(_ context.Context, _ string) ([]marketplace.RatingItem, float64, int32, error) {
	return s.ratings, s.avgRating, s.rateCount, s.err
}
func (s *stubMarketplaceSvc) Comment(_ context.Context, _, _, _ string) (string, error) {
	return "comment-1", s.err
}
func (s *stubMarketplaceSvc) ListComments(_ context.Context, _ string, _, _ int32) ([]marketplace.CommentItem, int32, error) {
	return s.comments, int32(len(s.comments)), s.err
}
func (s *stubMarketplaceSvc) Subscribe(_ context.Context, _, _, _, _ string) (string, error) {
	return "sub-1", s.err
}
func (s *stubMarketplaceSvc) Unsubscribe(_ context.Context, _, _ string) error { return s.err }
func (s *stubMarketplaceSvc) PurchaseStrategy(_ context.Context, _, _, _ string) (*marketplace.PurchaseResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &marketplace.PurchaseResult{
		SubscriptionID: "sub-1",
		TransactionID:  "tx-1",
		AmountCharged:  "49.99",
		BalanceAfter:   "50.01",
	}, nil
}
func (s *stubMarketplaceSvc) ListSubscriptions(_ context.Context, _ string) ([]marketplace.SubscriptionItem, error) {
	return s.subs, s.err
}
func (s *stubMarketplaceSvc) SetPricing(_ context.Context, _, _ string, _ float64) error { return s.err }
func (s *stubMarketplaceSvc) CanAccessCode(_ context.Context, _, _ string) (bool, error) {
	if s.err != nil { return false, s.err }
	return true, nil
}
func (s *stubMarketplaceSvc) StartMarketBacktest(_ context.Context, _ marketplace.StartBacktestParams) (string, error) {
	return "run-1", s.err
}
func (s *stubMarketplaceSvc) QueryBacktestRun(_ context.Context, _ uuid.UUID) (*marketplace.BacktestRunSnapshot, error) {
	if s.err != nil { return nil, s.err }
	return &marketplace.BacktestRunSnapshot{Status: "RUNNING"}, nil
}

type stubAdminChecker struct{ isAdmin bool }

func (a *stubAdminChecker) IsAdmin(_ context.Context, _ uuid.UUID) (bool, error) { return a.isAdmin, nil }

func testMarketplaceHandler(svc marketplaceSvc) *MarketplaceServer {
	return &MarketplaceServer{svc: svc, admin: &stubAdminChecker{isAdmin: true}, log: zap.NewNop()}
}

var _ marketplaceSvc = (*stubMarketplaceSvc)(nil)

// ── Tests ──

func TestMarketplace_PublishStrategy_Success(t *testing.T) {
	t.Parallel()
	svc := &stubMarketplaceSvc{publishID: "pub-1"}
	h := testMarketplaceHandler(svc)

	resp, err := h.PublishStrategy(context.Background(), connect.NewRequest(&antv1.PublishStrategyRequest{
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

	_, err := h.PublishStrategy(context.Background(), connect.NewRequest(&antv1.PublishStrategyRequest{}))
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

	resp, err := h.RateStrategy(context.Background(), connect.NewRequest(&antv1.RateStrategyRequest{
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

	resp, err := h.Subscribe(context.Background(), connect.NewRequest(&antv1.SubscribeRequest{
		UserId: "u1", StrategyId: "s1",
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
			{SubscriptionID: "sub-1", StrategyID: "s1", Active: true},
			{SubscriptionID: "sub-2", StrategyID: "s2", Active: false},
		},
	}
	h := testMarketplaceHandler(svc)

	resp, err := h.ListSubscriptions(context.Background(), connect.NewRequest(&antv1.ListSubscriptionsRequest{UserId: "u1"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.Subscriptions) != 2 {
		t.Fatalf("expected 2, got %d", len(resp.Msg.Subscriptions))
	}
}

func TestMarketplace_Unsubscribe_Error(t *testing.T) {
	t.Parallel()
	svc := &stubMarketplaceSvc{err: errors.New("not found")}
	h := testMarketplaceHandler(svc)

	_, err := h.Unsubscribe(context.Background(), connect.NewRequest(&antv1.UnsubscribeRequest{}))
	if err == nil {
		t.Fatal("expected error from unsubscribe")
	}
}

func TestMarketplace_CommentOnStrategy(t *testing.T) {
	t.Parallel()
	h := testMarketplaceHandler(&stubMarketplaceSvc{})

	resp, err := h.CommentOnStrategy(context.Background(), connect.NewRequest(&antv1.CommentOnStrategyRequest{
		StrategyId: "s1", Content: "Great!",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Id != "comment-1" {
		t.Errorf("expected comment-1, got %s", resp.Msg.Id)
	}
}
