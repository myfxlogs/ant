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

	"alphaforge/internal/repository"
)

// ErrAccountAlreadyBound is returned when an MT account is already bound to another user.
var ErrAccountAlreadyBound = errors.New("this trading account is already bound to another user")

// ErrAccountPasswordMismatch is returned when the old password does not match.
var ErrAccountPasswordMismatch = errors.New("old password does not match")

// AccountService provides account CRUD and lifecycle operations.
type AccountService struct {
	db     *pgxpool.Pool
	queries *repository.Queries
	log    *zap.Logger
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
func NewAccountService(db *pgxpool.Pool) *AccountService {
	return &AccountService{
		db:                db,
		queries:           repository.New(db),
		summaryCache:      make(map[string]*userSummaryCacheEntry),
		snapshotThrottle:  make(map[string]time.Time),
	}
}

// SetLogger injects a logger into the service.
func (s *AccountService) SetLogger(log *zap.Logger) { s.log = log }

// ── DTO types ──

// AccountDTO is a lightweight account view for the frontend.
type AccountDTO struct {
	ID, UserID, Platform, Broker, Login, Server, BrokerHost string
	Status                                                   string
	Balance, Equity, Credit, Margin, FreeMargin              decimal.Decimal
	MarginLevel                                              decimal.Decimal
	Leverage                                                 int32
	Currency                                                 string
	LastError                                                string
	LastConnectedAt                                          string
	IsInvestor                                               bool
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

// ── CRUD ──

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

// BeginTx starts a new database transaction.
func (s *AccountService) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return s.db.Begin(ctx)
}

// CreateAccountTx inserts a new MT account row within a transaction.
func (s *AccountService) CreateAccountTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, login, password, mtType, brokerCompany, brokerServer, brokerHost string) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO mt_accounts (user_id, login, password, mt_type, broker_company, broker_server, broker_host, account_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id::text
	`, userID, login, password, mtType, brokerCompany, brokerServer, brokerHost, string(StatusConnecting)).Scan(&id)
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

// UpdateAccount updates broker fields for an account.
func (s *AccountService) UpdateAccount(ctx context.Context, userID uuid.UUID, id, brokerCompany, brokerServer, brokerHost string) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE mt_accounts SET
			broker_company = COALESCE(NULLIF($3, ''), broker_company),
			broker_server  = COALESCE(NULLIF($4, ''), broker_server),
			broker_host    = COALESCE(NULLIF($5, ''), broker_host),
			updated_at     = CURRENT_TIMESTAMP
		WHERE id = $1::uuid AND user_id = $2 AND deleted_at IS NULL
	`, id, userID, brokerCompany, brokerServer, brokerHost)
	if err != nil {
		return fmt.Errorf("service: update account: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// DeleteAccount soft-deletes an MT account by setting deleted_at.
func (s *AccountService) DeleteAccount(ctx context.Context, userID uuid.UUID, id string) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE mt_accounts SET deleted_at = NOW(), account_status = 'disconnected', password = '' WHERE id = $1::uuid AND user_id = $2 AND deleted_at IS NULL`,
		id, userID)
	if err != nil {
		return fmt.Errorf("service: delete account: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	s.InvalidateSummaryCache(userID.String())
	return nil
}


// ── Account Info ──

// GetAccountCredentials returns the credentials needed for MT password verification.
func (s *AccountService) GetAccountCredentials(ctx context.Context, userID uuid.UUID, id string) (*AccountCredentials, error) {
	pgID, err := stringToPgUUID(id)
	if err != nil {
		return nil, fmt.Errorf("service: get account credentials: invalid account id: %w", err)
	}
	row, err := s.queries.GetAccountCredentials(ctx, repository.GetAccountCredentialsParams{ID: pgID, UserID: uuidToPgUUID(userID)})
	if err != nil {
		return nil, fmt.Errorf("service: get account credentials: %w", err)
	}
	return &AccountCredentials{
		Login:      row.Login,
		Platform:   row.MtType,
		BrokerHost: row.BrokerHost,
	}, nil
}

// UpdateAccountInfoTx updates balance/equity/margin/leverage/currency within a transaction.
// Does NOT touch account_status — that is owned by the gateway lifecycle.
func (s *AccountService) UpdateAccountInfoTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, id string, balance, equity, credit, margin, freeMargin decimal.Decimal, leverage int64, currency string, isInvestor bool) error {
	_, err := tx.Exec(ctx, `
		UPDATE mt_accounts SET
			balance = $3, equity = $4, credit = $5, margin = $6,
			free_margin = $7, leverage = $8, currency = $9,
			is_investor = $10, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1::uuid AND user_id = $2 AND deleted_at IS NULL
	`, id, userID, balance, equity, credit, margin, freeMargin, leverage, currency, isInvestor)
	if err != nil {
		return fmt.Errorf("service: update account info: %w", err)
	}
	return nil
}

// UpdateAccountInfo updates balance/equity/margin/leverage/currency after MT verification.
func (s *AccountService) UpdateAccountInfo(ctx context.Context, userID uuid.UUID, id string, balance, equity, credit, margin, freeMargin decimal.Decimal, leverage int64, currency string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE mt_accounts SET
			balance = $3, equity = $4, credit = $5, margin = $6,
			free_margin = $7, leverage = $8, currency = $9,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1::uuid AND user_id = $2 AND deleted_at IS NULL
	`, id, userID, balance, equity, credit, margin, freeMargin, leverage, currency)
	if err != nil {
		return fmt.Errorf("service: update account info: %w", err)
	}
	return nil
}

// UpdateAccountMetrics updates runtime balance/equity/margin metrics from broker callbacks.
func (s *AccountService) UpdateAccountMetrics(ctx context.Context, userID uuid.UUID, id string, balance, equity, credit, margin, freeMargin, marginLevel decimal.Decimal) error {
	pgID, err := stringToPgUUID(id)
	if err != nil {
		return fmt.Errorf("service: update account metrics: invalid account id: %w", err)
	}
	balN, err := decimalToPgNumeric(balance)
	if err != nil {
		return fmt.Errorf("service: update account metrics: balance: %w", err)
	}
	eqN, err := decimalToPgNumeric(equity)
	if err != nil {
		return fmt.Errorf("service: update account metrics: equity: %w", err)
	}
	crN, err := decimalToPgNumeric(credit)
	if err != nil {
		return fmt.Errorf("service: update account metrics: credit: %w", err)
	}
	marginN, err := decimalToPgNumeric(margin)
	if err != nil {
		return fmt.Errorf("service: update account metrics: margin: %w", err)
	}
	fmN, err := decimalToPgNumeric(freeMargin)
	if err != nil {
		return fmt.Errorf("service: update account metrics: free_margin: %w", err)
	}
	mlN, err := decimalToPgNumeric(marginLevel)
	if err != nil {
		return fmt.Errorf("service: update account metrics: margin_level: %w", err)
	}
	return s.queries.UpdateAccountMetrics(ctx, repository.UpdateAccountMetricsParams{
		ID:          pgID,
		UserID:      uuidToPgUUID(userID),
		Balance:     balN,
		Equity:      eqN,
		Credit:      crN,
		Margin:      marginN,
		FreeMargin:  fmN,
		MarginLevel: mlN,
	})
}

// UpdateBrokerThresholds updates broker margin_call/stop_out thresholds on an account.
// This is a system callback (called from the gateway pipeline where only accountID
// is available). Do not expose via ConnectRPC without adding a user_id check.
func (s *AccountService) UpdateBrokerThresholds(ctx context.Context, id string, marginCallPct, stopOutPct float64) error {
	_, err := s.db.Exec(ctx,
		`UPDATE mt_accounts SET broker_margin_call_pct=$1, broker_stop_out_pct=$2,
		 updated_at=CURRENT_TIMESTAMP WHERE id=$3 AND deleted_at IS NULL`, marginCallPct, stopOutPct, id)
	if err != nil {
		return fmt.Errorf("service: update broker thresholds: %w", err)
	}
	return nil
}

// LogAudit inserts an account audit event. Non-blocking — failures are logged but not returned.
func (s *AccountService) LogAudit(ctx context.Context, accountID, userID uuid.UUID, action, detail string) {
	if _, err := s.db.Exec(ctx,
		`INSERT INTO account_audit_log (account_id, user_id, action, detail) VALUES ($1, $2, $3, $4)`,
		accountID, userID, action, detail); err != nil && s.log != nil {
		s.log.Warn("LogAudit: insert failed", zap.String("action", action), zap.Error(err))
	}
}

// UserOwnsAccount checks if an account belongs to the given user.
func (s *AccountService) UserOwnsAccount(ctx context.Context, userID, accountID string) (bool, error) {
	pgUserID, err := stringToPgUUID(userID)
	if err != nil {
		return false, fmt.Errorf("service: user owns account: invalid user id: %w", err)
	}
	pgAccountID, err := stringToPgUUID(accountID)
	if err != nil {
		return false, fmt.Errorf("service: user owns account: invalid account id: %w", err)
	}
	return s.queries.UserOwnsAccount(ctx, repository.UserOwnsAccountParams{ID: pgAccountID, UserID: pgUserID})
}

// GetUserAccountIDs returns all account IDs belonging to a user.
func (s *AccountService) GetUserAccountIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.db.Query(ctx, "SELECT id FROM mt_accounts WHERE user_id = $1 AND deleted_at IS NULL", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
