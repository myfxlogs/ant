package marketplace

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/notification"
	"alphaforge/internal/repository"
)

// Service implements the C2C marketplace (strategy publish + subscribe).
// M12-B1: unified model — Publish writes to both user_strategy_publishes
// and marketplace_strategies; ListPublished JOINs both for rich metadata.
type Service struct {
	pg                *pgxpool.Pool
	walletRepo        *repository.WalletRepository
	log               *zap.Logger
	livePerfCollector *LivePerformanceCollector
	pubCache          *publishedCache
	notifSender       *notification.Sender
}

// SystemUserID is the designated platform system account for fee collection.
var SystemUserID = uuid.Nil

// New creates a marketplace service.
func New(pg *pgxpool.Pool, walletRepo *repository.WalletRepository, log *zap.Logger) *Service {
	return &Service{pg: pg, walletRepo: walletRepo, log: log, pubCache: newPublishedCache()}
}

// SetLivePerfCollector binds the live performance collector after construction.
// Called after NewLivePerformanceCollector creates the collector with this Service.
func (s *Service) SetLivePerfCollector(c *LivePerformanceCollector) {
	s.livePerfCollector = c
}

// SetWalletRepo binds the wallet repository after construction.
// Used when the Service is created early (for pipeline integration) but
// the wallet repo is only available later (inside registerHandlers).
func (s *Service) SetWalletRepo(w *repository.WalletRepository) {
	s.walletRepo = w
}

// SetNotificationSender injects the notification sender for marketplace events.
func (s *Service) SetNotificationSender(ns *notification.Sender) {
	s.notifSender = ns
}

// GetPlatformFeeRate reads the marketplace platform fee rate from system_config.
// Returns "0" if not configured or disabled.
func (s *Service) GetPlatformFeeRate(ctx context.Context) string {
	var rate string
	err := s.pg.QueryRow(ctx,
		`SELECT value FROM system_config WHERE key = 'marketplace.platform_fee_rate' AND enabled = true`,
	).Scan(&rate)
	if err != nil || rate == "" {
		return "0"
	}
	return rate
}

// ── Price model constants ────────────────────────────────────────────────────

const (
	PriceModelFree         = "free"
	PriceModelOnce         = "once"
	PriceModelSubscription = "subscription"
)

// ── Subscription kind constants ──────────────────────────────────────────────

const (
	SubKindPurchase     = "purchase"
	SubKindSubscription = "subscription"
)

// ── Wallet transaction type constants ────────────────────────────────────────

const (
	TxTypePurchase       = "purchase"
	TxTypeSale           = "sale"
	TxTypeRefund         = "refund"
	TxTypeRefundReversal = "refund_reversal"
	TxTypePlatformFee    = "platform_fee"
)

// ── Request / Response types ──────────────────────────────────────────────────

// PublishParams carries the full strategy metadata for publishing.
type PublishParams struct {
	UserID               string
	StrategyID           string
	Title                string
	Description          string
	PriceModel           string
	PriceAmount          string // decimal string
	AssetClass           string
	Symbols              []string
	Timeframe            string
	RiskLevel            string
	Tags                 []string
	CodeSnippet          string  // optional public code preview set by publisher
	BacktestSnapshotProto []byte  // optional proto-serialized BacktestSnapshot (nil → SQL NULL)
	PlatformFeeRate      string  // decimal string, platform commission rate (0.0–1.0)
	Disclaimer           string  // optional risk disclaimer
}

// PublishedStrategy represents a strategy listed in the marketplace
// with full metadata from marketplace_strategies (M12-B1).
type PublishedStrategy struct {
	PublishID        string
	StrategyID       string
	StrategyName     string
	PublisherUserID  string
	PublishedAt      time.Time
	Title            string
	Description      string
	PriceModel       string
	PriceAmount      *string // decimal string
	AssetClass       string
	Symbols          []string
	Timeframe        *string
	RiskLevel        string
	Tags             []string
	TotalSubscribers int
	WinRate          *decimal.Decimal
	TotalPnL         *decimal.Decimal
	AvgRating        float64
	RatingCount      int32
	CodeSnippet          string              // publisher-provided code preview
	BacktestSnapshotProto *antv1.BacktestSnapshot // optional backtest snapshot (proto)
	ProviderVerified     bool                // provider identity verified
	ProviderType         string              // human | ai | hybrid
	Disclaimer           string              // risk disclaimer
}

// BacktestRunSnapshot is a lightweight read of a single backtest_runs row.
type BacktestRunSnapshot struct {
	Status        string
	Error         string
	Symbol        string
	Timeframe     string
	ProtoResponse []byte
	StartedAt     *time.Time
	FinishedAt    *time.Time
	TemplateID    *uuid.UUID
}

// StartBacktestParams carries the parameters for marketplace backtest execution.
type StartBacktestParams struct {
	UserID         string
	StrategyID     string
	Symbol         string
	Timeframe      string
	StartDateMs    int64
	EndDateMs      int64
	InitialCapital decimal.Decimal
	Commission     decimal.Decimal
	Slippage       decimal.Decimal
	Leverage       decimal.Decimal
	TradeDirection string
}

// PublisherStats holds aggregated dashboard statistics for a strategy publisher.
type PublisherStats struct {
	TotalPublished   int32
	TotalSubscribers int32
	TotalRevenue     string // decimal string
	MonthlyRevenue   string // decimal string (last 30 days)
	AvgRating        float64
	TopStrategyID    string
	TopStrategyTitle string
	TopStrategySubs  int32
	// Phase 2.4: Enhanced analytics
	RevenueTrend       []RevenueTrendPoint
	SubscriberTrend    []SubscriberTrendPoint
	StrategyBreakdown  []StrategyBreakdown
}

type RevenueTrendPoint struct {
	DateMs              int64
	SaleRevenue         string // decimal string
	SubscriptionRevenue string // decimal string
}

type SubscriberTrendPoint struct {
	DateMs          int64
	NewSubscribers  int32
	Churned         int32
	Active          int32
}

type StrategyBreakdown struct {
	StrategyID      string
	Title           string
	TotalSubscribers int32
	Revenue         string // decimal string
	AvgRating       float64
	RatingCount     int32
	PriceModel      string
	PriceAmount     string
}

// PurchaseResult holds the outcome of a paid strategy purchase.
type PurchaseResult struct {
	SubscriptionID string
	TransactionID  string
	AmountCharged  string
	BalanceAfter   string
}

// RatingItem represents a single user rating for a strategy.
type RatingItem struct {
	ID        string
	UserID    string
	Rating    int32
	CreatedAt time.Time
}

// CommentItem represents a single user comment on a strategy.
type CommentItem struct {
	ID        string
	UserID    string
	UserName  string
	Content   string
	CreatedAt time.Time
}

// isUniqueViolation checks whether err is a PostgreSQL unique constraint violation.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique")
}
