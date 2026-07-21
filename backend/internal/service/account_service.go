package service

import (
	"errors"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/repository"
	"alphaforge/internal/secrets"
)

// ErrAccountAlreadyBound is returned when an MT account is already bound to another user.
var ErrAccountAlreadyBound = errors.New("this trading account is already bound to another user")

// ErrAccountPasswordMismatch is returned when the old password does not match.
var ErrAccountPasswordMismatch = errors.New("old password does not match")

// AccountService provides account CRUD and lifecycle operations.
type AccountService struct {
	db      *pgxpool.Pool
	queries *repository.Queries
	sec     secrets.Client
	log     *zap.Logger
	// Snapshot/cache fields (used by lifecycle methods, wired by NewAccountService).
	summaryMu          sync.RWMutex
	summaryCache       map[string]*userSummaryCacheEntry
	snapshotThrottleMu sync.Mutex
	snapshotThrottle   map[string]time.Time
}

// userSummaryCacheEntry holds per-account data for a user.
type userSummaryCacheEntry struct {
	accounts map[string]accountSummaryItem
	summary  UserAccountsSummary
}

type accountSummaryItem struct {
	balance decimal.Decimal
	equity  decimal.Decimal
	status  string
}

// NewAccountService creates an account service backed by the given pool.
func NewAccountService(db *pgxpool.Pool, sec secrets.Client) *AccountService {
	return &AccountService{
		db:               db,
		queries:          repository.New(db),
		sec:              sec,
		summaryCache:     make(map[string]*userSummaryCacheEntry),
		snapshotThrottle: make(map[string]time.Time),
	}
}

// SetLogger injects a logger into the service.
func (s *AccountService) SetLogger(log *zap.Logger) { s.log = log }

// ── DTO types ──

// AccountDTO is a lightweight account view for the frontend.
type AccountDTO struct {
	ID, UserID, Platform, Broker, Login, Server, BrokerHost string
	Status                                                  string
	Balance, Equity, Credit, Margin, FreeMargin             decimal.Decimal
	MarginLevel                                             decimal.Decimal
	Leverage                                                int32
	Currency                                                string
	LastError                                               string
	LastConnectedAt                                         string
	IsInvestor                                              bool
}

// AccountCredentials holds the fields needed to verify an MT account password.
type AccountCredentials struct {
	Login      string
	Platform   string
	BrokerHost string
}

// AccountSnapshot holds the current state of an MT account from the database.
type AccountSnapshot struct {
	ID          string
	Status      string
	Balance     decimal.Decimal
	Equity      decimal.Decimal
	Credit      decimal.Decimal
	Margin      decimal.Decimal
	FreeMargin  decimal.Decimal
	MarginLevel decimal.Decimal
}

// UserAccountsSummary holds the aggregated account summary for a user.
type UserAccountsSummary struct {
	TotalBalance   decimal.Decimal
	TotalEquity    decimal.Decimal
	TotalProfit    decimal.Decimal
	AccountCount   int32
	ConnectedCount int32
}
