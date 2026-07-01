package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"anttrader/internal/repository"
)

// ErrAccountAlreadyBound is returned when an MT account is already bound to another user.
var ErrAccountAlreadyBound = errors.New("this trading account is already bound to another user")

// ErrAccountPasswordMismatch is returned when the old password does not match.
var ErrAccountPasswordMismatch = errors.New("old password does not match")

// AccountService provides account CRUD and lifecycle operations.
type AccountService struct {
	db           *pgxpool.Pool
	queries      *repository.Queries
	log          *zap.Logger
	summaryMu          sync.RWMutex
	summaryCache       map[string]*userSummaryCacheEntry // userID -> cached per-account data
	snapshotThrottleMu sync.Mutex
	snapshotThrottle   map[string]time.Time
}

// userSummaryCacheEntry holds per-account data for a user, enabling incremental
// aggregate updates from profit events without a full DB scan.
type userSummaryCacheEntry struct {
	accounts map[string]accountSummaryItem // accountID -> latest metrics
	summary  UserAccountsSummary           // pre-computed aggregate
}

type accountSummaryItem struct {
	balance decimal.Decimal
	equity  decimal.Decimal
	status  string
}

// NewAccountService creates an account service backed by the given pool.
func NewAccountService(db *pgxpool.Pool) *AccountService {
	return &AccountService{
		db:           db,
		queries:      repository.New(db),
		summaryCache:     make(map[string]*userSummaryCacheEntry),
		snapshotThrottle: make(map[string]time.Time),
	}
}

// SetLogger injects a logger into the service.
func (s *AccountService) SetLogger(log *zap.Logger) { s.log = log }

// AccountDTO is a lightweight account view for the frontend.
type AccountDTO struct {
	ID, UserID, Platform, Broker, Login, Server string
	IsDisabled                                   bool
	Status                                       string
	Balance, Equity, Credit, Margin, FreeMargin  decimal.Decimal
	MarginLevel                                  decimal.Decimal
	Leverage                                     int32
	Currency                                     string
	LastError                                    string
	LastConnectedAt                              string
	IsInvestor                                   bool
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

// ListAccounts returns all accounts belonging to the given user.
func (s *AccountService) ListAccounts(ctx context.Context, userID uuid.UUID) ([]AccountDTO, error) {
	rows, err := s.queries.ListAccounts(ctx, uuidToPgUUID(userID))
	if err != nil {
		return nil, err
	}
	out := make([]AccountDTO, len(rows))
	for i, r := range rows {
		out[i] = mtAccountToDTO(r)
	}
	return out, nil
}

// GetAccount returns a single account by ID.
func (s *AccountService) GetAccount(ctx context.Context, userID uuid.UUID, accountID string) (*AccountDTO, error) {
	pgID, err := stringToPgUUID(accountID)
	if err != nil {
		return nil, fmt.Errorf("service: get account: invalid account id: %w", err)
	}
	row, err := s.queries.GetAccount(ctx, repository.GetAccountParams{ID: pgID, UserID: uuidToPgUUID(userID)})
	if err != nil {
		return nil, err
	}
	a := mtAccountToDTO(row)
	return &a, nil
}

// BeginTx starts a new database transaction (for #2, #28).
func (s *AccountService) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return s.db.Begin(ctx)
}

// CreateAccountTx inserts a new MT account row within a transaction (#2).
func (s *AccountService) CreateAccountTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, login, password, mtType, brokerCompany, brokerServer, brokerHost string) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO mt_accounts (user_id, login, password, mt_type, broker_company, broker_server, broker_host, account_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'connecting')
		RETURNING id::text
	`, userID, login, password, mtType, brokerCompany, brokerServer, brokerHost).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return "", ErrAccountAlreadyBound
		}
		return "", fmt.Errorf("service: create account: %w", err)
	}
	return id, nil
}

// CreateAccount inserts a new MT account row and returns the generated ID.
func (s *AccountService) CreateAccount(ctx context.Context, userID uuid.UUID, login, password, mtType, brokerCompany, brokerServer, brokerHost string) (string, error) {
	tx, err := s.BeginTx(ctx)
	if err != nil {
		return "", fmt.Errorf("service: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)
	id, err := s.CreateAccountTx(ctx, tx, userID, login, password, mtType, brokerCompany, brokerServer, brokerHost)
	if err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("service: commit tx: %w", err)
	}
	s.InvalidateSummaryCache(userID.String())
	return id, nil
}

// UpdateAccount updates broker fields and disabled status for an account.
func (s *AccountService) UpdateAccount(ctx context.Context, userID uuid.UUID, id, brokerCompany, brokerServer, brokerHost string, isDisabled *bool) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE mt_accounts SET
			broker_company = COALESCE(NULLIF($3, ''), broker_company),
			broker_server  = COALESCE(NULLIF($4, ''), broker_server),
			broker_host    = COALESCE(NULLIF($5, ''), broker_host),
			is_disabled    = COALESCE($6, is_disabled),
			updated_at     = CURRENT_TIMESTAMP
		WHERE id = $1::uuid AND user_id = $2
	`, id, userID, brokerCompany, brokerServer, brokerHost, isDisabled)
	if err != nil {
		return fmt.Errorf("service: update account: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	// Invalidate summary cache when disabled status may have changed (disabled
	// accounts stop sending profit events, so incremental updates can't fix it).
	if isDisabled != nil {
		s.InvalidateSummaryCache(userID.String())
	}
	return nil
}

// DeleteAccount removes an MT account and all its related data within a transaction (#28).
// Related tables without ON DELETE CASCADE are cleaned up first.
func (s *AccountService) DeleteAccount(ctx context.Context, userID uuid.UUID, id string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("service: delete account: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Delete related rows from tables that lack ON DELETE CASCADE.
	related := []string{
		`DELETE FROM account_balance_history WHERE account_id = $1::uuid`,
		`DELETE FROM account_connection_logs WHERE account_id = $1::uuid`,
		`DELETE FROM strategy_execution_logs WHERE account_id = $1::uuid`,
		`DELETE FROM order_history WHERE account_id = $1::uuid`,
	}
	for _, q := range related {
		if _, err := tx.Exec(ctx, q, id); err != nil {
			return fmt.Errorf("service: delete account: cleanup: %w", err)
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM mt_accounts WHERE id = $1::uuid AND user_id = $2`, id, userID); err != nil {
		return fmt.Errorf("service: delete account: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("service: delete account: commit tx: %w", err)
	}
	s.InvalidateSummaryCache(userID.String())
	return nil
}
