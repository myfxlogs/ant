package marketplace

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/marketplace"
	"anttrader/internal/pglisten"
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
	PurchaseStrategy(ctx context.Context, userID, strategyID, idempotencyKey string) (*marketplace.PurchaseResult, error)
	ListSubscriptions(ctx context.Context, userID string) ([]marketplace.SubscriptionItem, error)
	SetPricing(ctx context.Context, strategyID, priceModel string, priceAmount, platformFeeRate float64) error
	Unpublish(ctx context.Context, strategyID, userID string, isAdmin bool) error
	GetPublisherStats(ctx context.Context, userID string) (*marketplace.PublisherStats, error)
	CanAccessCode(ctx context.Context, userID, strategyID string) (bool, error)
	StartMarketBacktest(ctx context.Context, params marketplace.StartBacktestParams) (string, error)
	QueryBacktestRun(ctx context.Context, runID uuid.UUID) (*marketplace.BacktestRunSnapshot, error)
}

// MarketplaceServer implements ant.v1.MarketplaceServiceHandler.
type MarketplaceServer struct {
	svc      marketplaceSvc
	admin    interceptor.AdminChecker
	log      *zap.Logger
	pgListen *pglisten.Listener // Push-First: PG LISTEN for backtest status updates
}

var _ antv1c.MarketplaceServiceHandler = (*MarketplaceServer)(nil)

func NewMarketplaceServer(svc marketplaceSvc, admin interceptor.AdminChecker, log *zap.Logger) *MarketplaceServer {
	return &MarketplaceServer{svc: svc, admin: admin, log: log}
}

// SetPgListen injects the PG listener for push-first SSE streaming.
func (s *MarketplaceServer) SetPgListen(l *pglisten.Listener) { s.pgListen = l }
