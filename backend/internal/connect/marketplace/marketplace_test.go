package marketplace

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/marketplace"
)

// ── Stubs ──

type stubMarketplaceSvc struct {
	published         []marketplace.PublishedStrategy
	ratings           []marketplace.RatingItem
	avgRating         float64
	rateCount         int32
	comments          []marketplace.CommentItem
	commentsTotal     int32
	subs              []marketplace.SubscriptionItem
	publishID         string
	err               error
	rateErr           error
	commentErr        error
	subscribeErr      error
	unsubscribeErr    error
	purchaseErr       error
	unpublishErr      error
	publisherStatsErr error
	setPricingErr     error
	purchaseResult    *marketplace.PurchaseResult
	publisherStats    *marketplace.PublisherStats
}

func (s *stubMarketplaceSvc) Publish(_ context.Context, _ marketplace.PublishParams) (string, error) {
	return s.publishID, s.err
}
func (s *stubMarketplaceSvc) ListPublished(_ context.Context, _ string, _ int, _ int, _, _, _, _ string) ([]marketplace.PublishedStrategy, int, error) {
	return s.published, len(s.published), s.err
}
func (s *stubMarketplaceSvc) Rate(_ context.Context, _, _ string, _ int32) (float64, int32, error) {
	if s.rateErr != nil {
		return 0, 0, s.rateErr
	}
	return s.avgRating, s.rateCount, s.err
}
func (s *stubMarketplaceSvc) ListRatings(_ context.Context, _ string) ([]marketplace.RatingItem, float64, int32, error) {
	return s.ratings, s.avgRating, s.rateCount, s.err
}
func (s *stubMarketplaceSvc) Comment(_ context.Context, _, _, _ string) (string, error) {
	if s.commentErr != nil {
		return "", s.commentErr
	}
	return "comment-1", s.err
}
func (s *stubMarketplaceSvc) ListComments(_ context.Context, _ string, _, _ int32) ([]marketplace.CommentItem, int32, error) {
	total := s.commentsTotal
	if total == 0 && len(s.comments) > 0 {
		total = int32(len(s.comments))
	}
	return s.comments, total, s.err
}
func (s *stubMarketplaceSvc) Subscribe(_ context.Context, _, _, _, _ string) (string, error) {
	if s.subscribeErr != nil {
		return "", s.subscribeErr
	}
	return "sub-1", s.err
}
func (s *stubMarketplaceSvc) Unsubscribe(_ context.Context, _, _ string) error {
	if s.unsubscribeErr != nil {
		return s.unsubscribeErr
	}
	return s.err
}
func (s *stubMarketplaceSvc) PurchaseStrategy(_ context.Context, _, _, _, _ string) (*marketplace.PurchaseResult, error) {
	if s.purchaseErr != nil {
		return nil, s.purchaseErr
	}
	if s.err != nil {
		return nil, s.err
	}
	if s.purchaseResult != nil {
		return s.purchaseResult, nil
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
func (s *stubMarketplaceSvc) SetPricing(_ context.Context, _, _, _, _, _ string) error {
	if s.setPricingErr != nil {
		return s.setPricingErr
	}
	return s.err
}
func (s *stubMarketplaceSvc) Unpublish(_ context.Context, _, _ string, _ bool) error {
	if s.unpublishErr != nil {
		return s.unpublishErr
	}
	return s.err
}
func (s *stubMarketplaceSvc) GetPublisherStats(_ context.Context, _ string) (*marketplace.PublisherStats, error) {
	if s.publisherStatsErr != nil {
		return nil, s.publisherStatsErr
	}
	if s.err != nil {
		return nil, s.err
	}
	if s.publisherStats != nil {
		return s.publisherStats, nil
	}
	return &marketplace.PublisherStats{TotalPublished: 3, TotalSubscribers: 10}, nil
}
func (s *stubMarketplaceSvc) StartMarketBacktest(_ context.Context, _ marketplace.StartBacktestParams) (string, error) {
	return "run-1", s.err
}
func (s *stubMarketplaceSvc) QueryBacktestRun(_ context.Context, _ uuid.UUID) (*marketplace.BacktestRunSnapshot, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &marketplace.BacktestRunSnapshot{Status: "RUNNING"}, nil
}

func (s *stubMarketplaceSvc) GetPlatformFeeRate(_ context.Context) string {
	return "0"
}

func (s *stubMarketplaceSvc) GetLivePerformance(_ context.Context, _ string, _ int) ([]marketplace.LivePerformancePoint, *marketplace.LivePerformanceSummary, error) {
	return nil, nil, s.err
}

func (s *stubMarketplaceSvc) LinkLiveAccount(_ context.Context, _, _, _ string) error {
	return s.err
}

func (s *stubMarketplaceSvc) ValidateBacktestQuality(_ context.Context, _ []byte, _ string) ([]marketplace.QualityViolation, error) {
	return nil, nil
}

func (s *stubMarketplaceSvc) ListLeaderboard(_ context.Context, _, _, _ string, _ int) ([]marketplace.LeaderboardEntry, error) {
	return nil, s.err
}

func (s *stubMarketplaceSvc) StartTrial(_ context.Context, _, _ string) (string, time.Time, bool, error) {
	return "", time.Time{}, false, s.err
}

func (s *stubMarketplaceSvc) CompareStrategies(_ context.Context, _ []string) ([]marketplace.StrategyComparison, error) {
	return nil, s.err
}

func (s *stubMarketplaceSvc) GetStrategyPublicInfo(_ context.Context, _ string) (*antv1.GetStrategyPublicInfoResponse, error) {
	return &antv1.GetStrategyPublicInfoResponse{}, s.err
}
func (s *stubMarketplaceSvc) RequestVerification(_ context.Context, _, _, _ string) (string, string, error) {
	return "", "", s.err
}
func (s *stubMarketplaceSvc) ProcessVerification(_ context.Context, _, _ string, _ bool, _ string) error {
	return s.err
}
func (s *stubMarketplaceSvc) AdminListStrategies(_ context.Context, _, _ string, _, _ int) ([]marketplace.AdminStrategyRow, int, error) {
	return nil, 0, s.err
}
func (s *stubMarketplaceSvc) AdminFeatureStrategy(_ context.Context, _ string, _ bool, _ int32) error {
	return s.err
}
func (s *stubMarketplaceSvc) CreateRefundRequest(_ context.Context, _, _, _ string) (string, error) {
	return "", s.err
}
func (s *stubMarketplaceSvc) ListRefundRequests(_ context.Context, _ string, _, _ int) ([]marketplace.RefundRequestRow, int, error) {
	return nil, 0, s.err
}
func (s *stubMarketplaceSvc) ProcessRefundRequest(_ context.Context, _, _ string, _ bool, _ string) error {
	return s.err
}
func (s *stubMarketplaceSvc) GetMarketplaceAnalytics(_ context.Context, _ string) (*marketplace.AnalyticsResult, error) {
	return nil, s.err
}
func (s *stubMarketplaceSvc) GetTopStrategies(_ context.Context) ([]marketplace.TopItemRow, []marketplace.TopItemRow, error) {
	return nil, nil, s.err
}
func (s *stubMarketplaceSvc) GetTopProviders(_ context.Context) ([]marketplace.TopItemRow, []marketplace.TopItemRow, error) {
	return nil, nil, s.err
}
func (s *stubMarketplaceSvc) ValidateCoupon(_ context.Context, _, _, _ string) (*marketplace.CouponResult, error) {
	return nil, s.err
}
func (s *stubMarketplaceSvc) CreateCoupon(_ context.Context, _, _, _, _, _ string, _ int32, _ string, _ []string) (string, error) {
	return "", s.err
}
func (s *stubMarketplaceSvc) ListCoupons(_ context.Context, _ bool) ([]marketplace.CouponRow, error) {
	return nil, s.err
}
func (s *stubMarketplaceSvc) DisableCoupon(_ context.Context, _ string) error {
	return s.err
}
func (s *stubMarketplaceSvc) GetProviderEarnings(_ context.Context, _ string) (*marketplace.ProviderEarningsResult, error) {
	return nil, s.err
}
func (s *stubMarketplaceSvc) ListProviderTransactions(_ context.Context, _ string, _, _ int) ([]marketplace.ProviderTxRow, int, error) {
	return nil, 0, s.err
}
func (s *stubMarketplaceSvc) DetectDecay(_ context.Context, _ string) (*marketplace.DecayResult, error) {
	return nil, s.err
}
func (s *stubMarketplaceSvc) CreateOptimizationTask(_ context.Context, _, _, _ string, _ *marketplace.DecayResult) (string, error) {
	return "", s.err
}
func (s *stubMarketplaceSvc) ListOptimizationTasks(_ context.Context, _, _ string, _, _ int) ([]marketplace.OptimizationTask, int, error) {
	return nil, 0, s.err
}
func (s *stubMarketplaceSvc) GetOptimizationTask(_ context.Context, _, _ string) (*marketplace.OptimizationTask, error) {
	return nil, s.err
}
func (s *stubMarketplaceSvc) RejectOptimizationTask(_ context.Context, _, _ string) error {
	return s.err
}
func (s *stubMarketplaceSvc) PublishOptimization(_ context.Context, _, _ string) (string, error) {
	return "", s.err
}
func (s *stubMarketplaceSvc) PreviewOptimization(_ context.Context, _, _ string) (*marketplace.PreviewOptimizationResult, error) {
	return nil, s.err
}
func (s *stubMarketplaceSvc) InitiateStrategyIteration(_ context.Context, _, _ string) (string, error) {
	return "", s.err
}
func (s *stubMarketplaceSvc) CreateBundle(_ context.Context, _, _, _, _, _ string, _ []string, _ string) (string, error) {
	return "", s.err
}
func (s *stubMarketplaceSvc) ListBundles(_ context.Context, _ string, _, _ int) ([]marketplace.Bundle, int, error) {
	return nil, 0, s.err
}
func (s *stubMarketplaceSvc) GetBundle(_ context.Context, _ string) (*marketplace.Bundle, error) {
	return nil, s.err
}
func (s *stubMarketplaceSvc) PurchaseBundle(_ context.Context, _, _, _ string) (*marketplace.PurchaseResult, error) {
	return nil, s.err
}
func (s *stubMarketplaceSvc) DeleteBundle(_ context.Context, _, _ string, _ bool) error {
	return s.err
}
func (s *stubMarketplaceSvc) ListFeeTiers(_ context.Context) ([]marketplace.FeeTier, error) {
	return nil, s.err
}
func (s *stubMarketplaceSvc) UpdateFeeTier(_ context.Context, _ int32, _ string, _ int32, _ bool) error {
	return s.err
}
func (s *stubMarketplaceSvc) GetProviderFeeTierWithStats(_ context.Context, _ string) (*marketplace.ProviderFeeTierResult, error) {
	return nil, s.err
}

type stubAdminChecker struct{ isAdmin bool }

func (a *stubAdminChecker) IsAdmin(_ context.Context, _ uuid.UUID) (bool, error) {
	return a.isAdmin, nil
}

func testMarketplaceHandler(svc marketplaceSvc) *MarketplaceServer {
	return &MarketplaceServer{svc: svc, admin: &stubAdminChecker{isAdmin: true}, log: zap.NewNop()}
}

var _ marketplaceSvc = (*stubMarketplaceSvc)(nil)

// ── Tests ──

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
			{SubscriptionID: "sub-1", StrategyID: "s1", Active: true},
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
