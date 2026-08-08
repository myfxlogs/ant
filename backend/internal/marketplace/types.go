package marketplace

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/notification"
	"alphaforge/internal/repository"
)

// DemandRecorder records demand signals for unsupported builtins (K3).
// Implemented by knowledgebase.Service.
type DemandRecorder interface {
	RecordDemandSignal(ctx context.Context, builtinName string, userID uuid.UUID) error
}

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
	optimizer         codeOptimizer
	creditSvc         CreditService
	demandRecorder    DemandRecorder
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

// SetOptimizer injects the AI strategy generator for optimization.
func (s *Service) SetOptimizer(o codeOptimizer) {
	s.optimizer = o
}

// SetCreditService injects the credit service for author-initiated AI iteration billing.
func (s *Service) SetCreditService(cs CreditService) {
	s.creditSvc = cs
}

// SetDemandRecorder injects the K3 demand signal recorder (knowledgebase.Service).
func (s *Service) SetDemandRecorder(dr DemandRecorder) {
	s.demandRecorder = dr
}

// codeOptimizer is the AI strategy generator used for optimization.
type codeOptimizer interface {
	Generate(ctx context.Context, userID uuid.UUID, msg *antv1.AgentGenerateStrategyRequest, stream func(*antv1.AgentGenerateStrategyChunk) error) error
}

// CreditService is the interface for AI credit billing (PreHold + Settle).
// Used for author-initiated AI iteration: agent generation bills the author's credits.
type CreditService interface {
	PreHold(ctx context.Context, userID uuid.UUID, sessionID string, providerID, modelName string) error
	Settle(ctx context.Context, userID uuid.UUID, sessionID, providerID, modelName string, inputTokens, outputTokens int) error
	ReleaseHold(ctx context.Context, userID uuid.UUID, sessionID string) error
	CheckBalance(ctx context.Context, userID uuid.UUID, minCredits decimal.Decimal) error
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

// ── Settlement status constants ──────────────────────────────────────────────

const (
	SettlementStatusFrozen   = "frozen"
	SettlementStatusSettled  = "settled"
	SettlementStatusRefunded = "refunded"
)

const DefaultRefundWindowDays = 7

// ── Wallet transaction type constants ────────────────────────────────────────

const (
	TxTypePurchase       = "purchase"
	TxTypeRefund         = "refund"
	TxTypeRefundReversal = "refund_reversal"
	TxTypeSettlement     = "settlement"
	TxTypeFeeSettlement  = "fee_settlement"
)

// Legacy tx_type values (pre-5.4, no longer created but may exist in DB).
const (
	TxTypeSale        = "sale"
	TxTypePlatformFee = "platform_fee"
)

// ── Idempotency key prefix constants (SSOT) ──────────────────────────────────
// Active:  mkt-buy-{idemKey}, mkt-refund-{subID}, mkt-rev-{subID}, mkt-fee-rev-{subID}
//
//	mkt-renew-{subID}, mkt-settle-{settlementID}, mkt-fee-settle-{settlementID}
//
// Legacy:  mkt-sale-*, mkt-fee-*, mkt-renew-sale-*, mkt-renew-fee-* (pre-5.4, kept for subJoinOnClause)
const (
	IdemKeyBuy       = "mkt-buy-"
	IdemKeySale      = "mkt-sale-"
	IdemKeyFee       = "mkt-fee-"
	IdemKeyRenewBuy  = "mkt-renew-"
	IdemKeyRenewSale = "mkt-renew-sale-"
	IdemKeyRenewFee  = "mkt-renew-fee-"
	IdemKeyRefund    = "mkt-refund-"
	IdemKeyRev       = "mkt-rev-"
	IdemKeyFeeRev    = "mkt-fee-rev-"
	IdemKeySettle    = "mkt-settle-"
	IdemKeyFeeSettle = "mkt-fee-settle-"
)

// subJoinOnClause is the shared ON clause for joining wallet_transactions to
// user_subscriptions via idem_key prefix matching. Covers:
// - initial sales (mkt-sale-{idemKey} → us.idempotency_key) — legacy, pre-5.4
// - renewal sales (mkt-renew-sale-{subID} → us.id) — legacy, pre-5.4
// - refund reversals (mkt-rev-{subID} → us.id)
// - settlements (mkt-settle-{settlementID} → marketplace_settlements.purchase_id → us.id)
const subJoinOnClause = ` ON (
  us.idempotency_key = REPLACE(wt.idem_key, 'mkt-sale-', '')
  OR us.id::text = REPLACE(wt.idem_key, 'mkt-renew-sale-', '')
  OR us.id::text = REPLACE(wt.idem_key, 'mkt-rev-', '')
  OR us.id = (
    SELECT ms.purchase_id FROM marketplace_settlements ms
    WHERE ms.id::text = REPLACE(wt.idem_key, 'mkt-settle-', '')
  )
)`

// ── Request / Response types ──────────────────────────────────────────────────

// PublishParams carries the full strategy metadata for publishing.
type PublishParams struct {
	UserID                string
	StrategyID            string
	Title                 string
	Description           string
	PriceModel            string
	PriceAmount           string // decimal string
	AssetClass            string
	Symbols               []string
	Timeframe             string
	RiskLevel             string
	Tags                  []string
	CodeSnippet           string // optional public code preview set by publisher
	BacktestSnapshotProto []byte // optional proto-serialized BacktestSnapshot (nil → SQL NULL)
	PlatformFeeRate       string // decimal string, platform commission rate (0.0–1.0)
	Disclaimer            string // optional risk disclaimer
	TrialDays             int    // publisher-configurable trial period (default 7)
	RefundWindowDays      int    // publisher-configurable refund window in days (default 7)
}

// PublishedStrategy represents a strategy listed in the marketplace
// with full metadata from marketplace_strategies (M12-B1).
type PublishedStrategy struct {
	PublishID             string
	StrategyID            string
	StrategyName          string
	PublisherUserID       string
	PublishedAt           time.Time
	Title                 string
	Description           string
	PriceModel            string
	PriceAmount           *string // decimal string
	AssetClass            string
	Symbols               []string
	Timeframe             *string
	RiskLevel             string
	Tags                  []string
	TotalSubscribers      int
	WinRate               *decimal.Decimal
	TotalPnL              *decimal.Decimal
	AvgRating             float64
	RatingCount           int32
	CodeSnippet           string                  // publisher-provided code preview
	BacktestSnapshotProto *antv1.BacktestSnapshot // optional backtest snapshot (proto)
	ProviderVerified      bool                    // provider identity verified
	ProviderType          string                  // human | ai | hybrid
	Disclaimer            string                  // risk disclaimer
	DecayStatus           string                  // none | decaying | decayed
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
	TotalRevenue     string // decimal string — settled revenue only
	MonthlyRevenue   string // decimal string (last 30 days) — settled revenue only
	AvgRating        float64
	TopStrategyID    string
	TopStrategyTitle string
	TopStrategySubs  int32
	// Phase 2.4: Enhanced analytics
	RevenueTrend      []RevenueTrendPoint
	SubscriberTrend   []SubscriberTrendPoint
	StrategyBreakdown []StrategyBreakdown
	// Phase 5.4: Settlement balance breakdown
	PendingSettlement  string // decimal string — frozen provider_amount sum
	NextSettlementDate string // ISO 8601 — earliest settles_at among frozen rows
}

type RevenueTrendPoint struct {
	DateMs              int64
	SaleRevenue         string // decimal string
	SubscriptionRevenue string // decimal string
}

type SubscriberTrendPoint struct {
	DateMs         int64
	NewSubscribers int32
	Churned        int32
	Active         int32
}

type StrategyBreakdown struct {
	StrategyID       string
	Title            string
	TotalSubscribers int32
	Revenue          string // decimal string
	AvgRating        float64
	RatingCount      int32
	PriceModel       string
	PriceAmount      string
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

// isUniqueViolation checks whether err is a PostgreSQL unique constraint violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
