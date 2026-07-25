package marketplace

import (
	"context"
	"fmt"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/marketplace"
	"alphaforge/internal/pglisten"
)

// marketplaceSvc is the local interface for marketplace business logic.
// Defined on the consumer side — marketplace.Service package need not know about it.
type marketplaceSvc interface {
	Publish(ctx context.Context, params marketplace.PublishParams) (string, error)
	ListPublished(ctx context.Context, userID string, limit int, offset int, assetClass, keyword, sortBy string) ([]marketplace.PublishedStrategy, error)
	Rate(ctx context.Context, userID, strategyID string, rating int32) (float64, int32, error)
	ListRatings(ctx context.Context, strategyID string) ([]marketplace.RatingItem, float64, int32, error)
	Comment(ctx context.Context, userID, strategyID, content string) (string, error)
	ListComments(ctx context.Context, strategyID string, limit, offset int32) ([]marketplace.CommentItem, int32, error)
	Subscribe(ctx context.Context, userID, publisherUserID, strategyID, kind string) (string, error)
	Unsubscribe(ctx context.Context, userID, subscriptionID string) error
	PurchaseStrategy(ctx context.Context, userID, strategyID, couponCode, idempotencyKey string) (*marketplace.PurchaseResult, error)
	ListSubscriptions(ctx context.Context, userID string) ([]marketplace.SubscriptionItem, error)
	SetPricing(ctx context.Context, userID, strategyID, priceModel, priceAmount, platformFeeRate string) error
	Unpublish(ctx context.Context, strategyID, userID string, isAdmin bool) error
	GetPublisherStats(ctx context.Context, userID string) (*marketplace.PublisherStats, error)
	StartMarketBacktest(ctx context.Context, params marketplace.StartBacktestParams) (string, error)
	QueryBacktestRun(ctx context.Context, runID uuid.UUID) (*marketplace.BacktestRunSnapshot, error)
	GetPlatformFeeRate(ctx context.Context) string
	GetLivePerformance(ctx context.Context, strategyID string, limit int) ([]marketplace.LivePerformancePoint, *marketplace.LivePerformanceSummary, error)
	LinkLiveAccount(ctx context.Context, strategyID, accountID, userID string) error
	ValidateBacktestQuality(ctx context.Context, snapshotProto []byte, strategyID string) ([]marketplace.QualityViolation, error)
	ListLeaderboard(ctx context.Context, lbType, period, assetClass string, limit int) ([]marketplace.LeaderboardEntry, error)
	StartTrial(ctx context.Context, userID, strategyID string) (trialID string, expiresAt time.Time, alreadyTried bool, err error)
	CompareStrategies(ctx context.Context, strategyIDs []string) ([]marketplace.StrategyComparison, error)
	GetStrategyPublicInfo(ctx context.Context, strategyID string) (*antv1.GetStrategyPublicInfoResponse, error)
	RequestVerification(ctx context.Context, userID, providerType, note string) (string, string, error)
	ProcessVerification(ctx context.Context, adminID, requestID string, approve bool, note string) error
	AdminListStrategies(ctx context.Context, status, keyword string, limit, offset int) ([]marketplace.AdminStrategyRow, int, error)
	AdminFeatureStrategy(ctx context.Context, strategyID string, featured bool, priority int32) error
	CreateRefundRequest(ctx context.Context, userID, subscriptionID, reason string) (string, error)
	ListRefundRequests(ctx context.Context, status string, limit, offset int) ([]marketplace.RefundRequestRow, int, error)
	ProcessRefundRequest(ctx context.Context, adminID, refundID string, approve bool, reviewNote string) error
	GetMarketplaceAnalytics(ctx context.Context, period string) (*marketplace.AnalyticsResult, error)
	GetTopStrategies(ctx context.Context) ([]marketplace.TopItemRow, []marketplace.TopItemRow, error)
	GetTopProviders(ctx context.Context) ([]marketplace.TopItemRow, []marketplace.TopItemRow, error)
	ValidateCoupon(ctx context.Context, code, strategyID, amount string) (*marketplace.CouponResult, error)
	CreateCoupon(ctx context.Context, adminID, code, discountType, discountValue, minPurchase string, maxUses int32, expiresAt string, applicableStrategyIDs []string) (string, error)
	ListCoupons(ctx context.Context, enabledOnly bool) ([]marketplace.CouponRow, error)
	DisableCoupon(ctx context.Context, couponID string) error
	GetProviderEarnings(ctx context.Context, userID string) (*marketplace.ProviderEarningsResult, error)
	ListProviderTransactions(ctx context.Context, userID string, limit, offset int) ([]marketplace.ProviderTxRow, int, error)
	DetectDecay(ctx context.Context, strategyID string) (*marketplace.DecayResult, error)
	CreateOptimizationTask(ctx context.Context, strategyID, publisherID, triggerReason string, decayResult *marketplace.DecayResult) (string, error)
	ListOptimizationTasks(ctx context.Context, publisherID, status string, limit, offset int) ([]marketplace.OptimizationTask, int, error)
	GetOptimizationTask(ctx context.Context, taskID, publisherID string) (*marketplace.OptimizationTask, error)
	RejectOptimizationTask(ctx context.Context, taskID, publisherID string) error
	PublishOptimization(ctx context.Context, taskID, publisherID string) (string, error)
	PreviewOptimization(ctx context.Context, taskID, publisherID string) (*marketplace.PreviewOptimizationResult, error)
	CreateBundle(ctx context.Context, publisherID, title, description, priceModel, priceAmount string, strategyIDs []string, platformFeeRate string) (string, error)
	ListBundles(ctx context.Context, publisherID string, limit, offset int) ([]marketplace.Bundle, int, error)
	GetBundle(ctx context.Context, bundleID string) (*marketplace.Bundle, error)
	PurchaseBundle(ctx context.Context, userID, bundleID, idempotencyKey string) (*marketplace.PurchaseResult, error)
	DeleteBundle(ctx context.Context, bundleID, userID string, isAdmin bool) error
	ListFeeTiers(ctx context.Context) ([]marketplace.FeeTier, error)
	UpdateFeeTier(ctx context.Context, tierID int32, feeRate string, minSalesCount int32, enabled bool) error
	GetProviderFeeTierWithStats(ctx context.Context, publisherID string) (*marketplace.ProviderFeeTierResult, error)
}

// agentGenerator is the interface for the AI strategy generator (agent.Generator).
type agentGenerator interface {
	Generate(ctx context.Context, userID uuid.UUID, msg *antv1.AgentGenerateStrategyRequest, stream func(*antv1.AgentGenerateStrategyChunk) error) error
}

// MarketplaceServer implements ant.v1.MarketplaceServiceHandler.
type MarketplaceServer struct {
	svc         marketplaceSvc
	admin       interceptor.AdminChecker
	log         *zap.Logger
	pgListen    *pglisten.Listener // Push-First: PG LISTEN for backtest status updates
	gen         agentGenerator     // Phase 2: AI strategy generator
	autoLimiter *autoGenerateLimiter
	limiterOnce sync.Once
	batch       *marketplace.BatchGenerator // Phase 2.2: batch generation queue
	pgPool      *pgxpool.Pool               // Phase 2.3: template queries
}

var _ antv1c.MarketplaceServiceHandler = (*MarketplaceServer)(nil)

func NewMarketplaceServer(svc marketplaceSvc, admin interceptor.AdminChecker, log *zap.Logger) *MarketplaceServer {
	return &MarketplaceServer{svc: svc, admin: admin, log: log}
}

// checkAdmin safely parses the user ID from context and verifies admin status.
// Returns the parsed UUID and nil error if admin, or nil UUID and a ConnectRPC error otherwise.
// M12: Replaces uuid.MustParse which can panic on invalid/empty user IDs.
func (s *MarketplaceServer) checkAdmin(ctx context.Context) (uuid.UUID, error) {
	uid, err := uuid.Parse(interceptor.GetUserID(ctx))
	if err != nil {
		return uuid.Nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	isAdmin, err := s.admin.IsAdmin(ctx, uid)
	if err != nil || !isAdmin {
		return uuid.Nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("admin only"))
	}
	return uid, nil
}

// SetPgListen injects the PG listener for push-first SSE streaming.
func (s *MarketplaceServer) SetPgListen(l *pglisten.Listener) { s.pgListen = l }

// SetGenerator injects the AI strategy generator for Phase 2 GenerateAndPublish.
func (s *MarketplaceServer) SetGenerator(g agentGenerator) { s.gen = g }

// SetBatchGenerator injects the batch generator for Phase 2.2 admin operations.
func (s *MarketplaceServer) SetBatchGenerator(b *marketplace.BatchGenerator) { s.batch = b }

// SetPgPool injects the pool for Phase 2.3 template queries.
func (s *MarketplaceServer) SetPgPool(p *pgxpool.Pool) { s.pgPool = p }
