package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// ErrAccountAlreadyBound is returned when an MT account is already bound to another user.
var ErrAccountAlreadyBound = errors.New("this trading account is already bound to another user")

// ErrAccountPasswordMismatch is returned when the old password does not match.
var ErrAccountPasswordMismatch = errors.New("old password does not match")

// AccountService provides account CRUD and lifecycle operations.
type AccountService struct {
	db  *pgxpool.Pool
	log *zap.Logger
}

// NewAccountService creates an account service backed by the given pool.
func NewAccountService(db *pgxpool.Pool) *AccountService {
	return &AccountService{db: db}
}

// SetLogger injects a logger into the service.
func (s *AccountService) SetLogger(log *zap.Logger) { s.log = log }

// AccountDTO is a lightweight account view for the frontend.
type AccountDTO struct {
	ID, UserID, Platform, Broker, Login, Server string
	IsDisabled                                   bool
	Status                                       string
	Balance, Equity, Credit, Margin, FreeMargin  float64
	MarginLevel                                  float64
	Leverage                                     int32
	Currency                                     string
	LastError                                    string
	LastConnectedAt                              string
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
	Balance     float64
	Equity      float64
	Credit      float64
	Margin      float64
	FreeMargin  float64
	MarginLevel float64
}

// UserAccountsSummary holds the aggregated account summary for a user.
type UserAccountsSummary struct {
	TotalBalance   float64
	TotalEquity    float64
	TotalProfit    float64
	AccountCount   int32
	ConnectedCount int32
}

// ListAccounts returns all accounts belonging to the given user.
func (s *AccountService) ListAccounts(ctx context.Context, userID uuid.UUID) ([]AccountDTO, error) {
	rows, err := s.db.Query(ctx, `
			SELECT id, user_id, mt_type, broker_company, login, broker_server, is_disabled,
			       account_status, balance, equity, COALESCE(credit, 0), margin, free_margin, margin_level, leverage, currency, COALESCE(last_error, ''),
			       COALESCE(last_connected_at::text, '')
			FROM mt_accounts WHERE user_id = $1 ORDER BY mt_type, login
		`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AccountDTO
	for rows.Next() {
		var a AccountDTO
		if err := rows.Scan(&a.ID, &a.UserID, &a.Platform, &a.Broker, &a.Login, &a.Server, &a.IsDisabled,
			&a.Status, &a.Balance, &a.Equity, &a.Credit, &a.Margin, &a.FreeMargin, &a.MarginLevel, &a.Leverage, &a.Currency, &a.LastError,
			&a.LastConnectedAt); err != nil {
			return nil, fmt.Errorf("service: list accounts: scan: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ConnectAccount marks an account as ready for connection (#36: now sets status to 'connecting').
func (s *AccountService) ConnectAccount(ctx context.Context, userID uuid.UUID, accountID string) error {
	tag, err := s.db.Exec(ctx,
		"UPDATE mt_accounts SET account_status = 'connecting', updated_at = CURRENT_TIMESTAMP WHERE id = $1::uuid AND user_id = $2",
		accountID, userID)
	if err != nil {
		return fmt.Errorf("service: connect account: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("account not found: %s", accountID)
	}
	return nil
}

// GetAccount returns a single account by ID.
func (s *AccountService) GetAccount(ctx context.Context, userID uuid.UUID, accountID string) (*AccountDTO, error) {
	var a AccountDTO
	err := s.db.QueryRow(ctx, `
			SELECT id, user_id, mt_type, broker_company, login, broker_server, is_disabled,
			       account_status, balance, equity, COALESCE(credit, 0), margin, free_margin, margin_level, leverage, currency, COALESCE(last_error, ''),
			       COALESCE(last_connected_at::text, '')
			FROM mt_accounts WHERE id = $1::uuid AND user_id = $2
		`, accountID, userID).Scan(&a.ID, &a.UserID, &a.Platform, &a.Broker, &a.Login, &a.Server, &a.IsDisabled,
		&a.Status, &a.Balance, &a.Equity, &a.Credit, &a.Margin, &a.FreeMargin, &a.MarginLevel, &a.Leverage, &a.Currency, &a.LastError,
		&a.LastConnectedAt)
	if err != nil {
		return nil, err
	}
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
	return id, nil
}

// UpdateAccount updates broker fields and disabled status for an account.
func (s *AccountService) UpdateAccount(ctx context.Context, userID uuid.UUID, id, brokerCompany, brokerServer, brokerHost string, isDisabled *bool) error {
	_, err := s.db.Exec(ctx, `
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
	return nil
}

// UpdateAccountInfoTx updates balance/equity/margin/leverage/currency + status within a transaction (#2, #32).
func (s *AccountService) UpdateAccountInfoTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, id string, balance, equity, credit, margin, freeMargin float64, leverage int64, currency string) error {
	_, err := tx.Exec(ctx, `
			UPDATE mt_accounts SET
				balance = $3,
				equity  = $4,
				credit  = $5,
				margin  = $6,
				free_margin = $7,
				leverage = $8,
				currency = $9,
				account_status = 'connected',
				updated_at = CURRENT_TIMESTAMP
			WHERE id = $1::uuid AND user_id = $2
		`, id, userID, balance, equity, credit, margin, freeMargin, leverage, currency)
	if err != nil {
		return fmt.Errorf("service: update account info: %w", err)
	}
	return nil
}

// UpdateAccountInfo updates balance/equity/margin/leverage/currency + status after MT verification (#32: removed COALESCE(NULLIF(0)) to allow zero-balance).
func (s *AccountService) UpdateAccountInfo(ctx context.Context, userID uuid.UUID, id string, balance, equity, credit, margin, freeMargin float64, leverage int64, currency string) error {
	_, err := s.db.Exec(ctx, `
			UPDATE mt_accounts SET
				balance = $3,
				equity  = $4,
				credit  = $5,
				margin  = $6,
				free_margin = $7,
				leverage = $8,
				currency = $9,
				account_status = 'connected',
				updated_at = CURRENT_TIMESTAMP
			WHERE id = $1::uuid AND user_id = $2
		`, id, userID, balance, equity, credit, margin, freeMargin, leverage, currency)
	if err != nil {
		return fmt.Errorf("service: update account info: %w", err)
	}
	return nil
}

// MarkAccountNeedsRebind marks an account as needing rebind instead of deleting it (#4).
func (s *AccountService) MarkAccountNeedsRebind(ctx context.Context, userID uuid.UUID, id string) error {
	_, err := s.db.Exec(ctx,
		`UPDATE mt_accounts SET account_status = 'needs_rebind', updated_at = CURRENT_TIMESTAMP WHERE id = $1::uuid AND user_id = $2`,
		id, userID)
	if err != nil {
		return fmt.Errorf("service: mark account needs_rebind: %w", err)
	}
	return nil
}

// GetAccountCredentials returns the credentials needed for MT password verification.
func (s *AccountService) GetAccountCredentials(ctx context.Context, userID uuid.UUID, id string) (*AccountCredentials, error) {
	var c AccountCredentials
	err := s.db.QueryRow(ctx,
		`SELECT login, mt_type, broker_host FROM mt_accounts WHERE id = $1::uuid AND user_id = $2`,
		id, userID).Scan(&c.Login, &c.Platform, &c.BrokerHost)
	if err != nil {
		return nil, fmt.Errorf("service: get account credentials: %w", err)
	}
	return &c, nil
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
	return nil
}

// DisconnectAccount sets the account status to disconnected.
func (s *AccountService) DisconnectAccount(ctx context.Context, userID uuid.UUID, id string) error {
	_, err := s.db.Exec(ctx, `
			UPDATE mt_accounts SET account_status = 'disconnected', updated_at = CURRENT_TIMESTAMP WHERE id = $1::uuid AND user_id = $2
		`, id, userID)
	if err != nil {
		return fmt.Errorf("service: disconnect account: %w", err)
	}
	return nil
}

// DisconnectAccountByID sets the account status to disconnected by account ID only
// (for system callbacks where userID is not available).
func (s *AccountService) DisconnectAccountByID(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, `
			UPDATE mt_accounts SET account_status = 'disconnected', updated_at = CURRENT_TIMESTAMP WHERE id = $1::uuid
		`, id)
	if err != nil {
		return fmt.Errorf("service: disconnect account by id: %w", err)
	}
	return nil
}

// UpdateAccountMetrics updates runtime balance/equity/margin metrics from broker callbacks.
// Unlike UpdateAccountInfo, this does not overwrite leverage or currency.
func (s *AccountService) UpdateAccountMetrics(ctx context.Context, userID uuid.UUID, id string, balance, equity, credit, margin, freeMargin, marginLevel float64) error {
	_, err := s.db.Exec(ctx, `
			UPDATE mt_accounts SET
				balance = $3,
				equity  = $4,
				credit  = $5,
				margin  = $6,
				free_margin = $7,
				margin_level = $8,
				account_status = 'connected',
				updated_at = CURRENT_TIMESTAMP
			WHERE id = $1::uuid AND user_id = $2
		`, id, userID, balance, equity, credit, margin, freeMargin, marginLevel)
	if err != nil {
		return fmt.Errorf("service: update account metrics: %w", err)
	}
	return nil
}

// RecordBalanceSnapshot inserts an hourly equity/balance snapshot row.
func (s *AccountService) RecordBalanceSnapshot(ctx context.Context, id string, balance, equity, margin, freeMargin float64) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO account_balance_history (account_id, balance, equity, margin, free_margin, recorded_at)
		 VALUES ($1, $2, $3, $4, $5, NOW())`,
		id, balance, equity, margin, freeMargin)
	if err != nil {
		return fmt.Errorf("service: record balance snapshot: %w", err)
	}
	return nil
}

// UpdateBrokerThresholds updates broker margin_call/stop_out thresholds on an account.
func (s *AccountService) UpdateBrokerThresholds(ctx context.Context, id string, marginCallPct, stopOutPct float64) error {
	_, err := s.db.Exec(ctx,
		`UPDATE mt_accounts SET broker_margin_call_pct=$1, broker_stop_out_pct=$2,
		 updated_at=CURRENT_TIMESTAMP WHERE id=$3`, marginCallPct, stopOutPct, id)
	if err != nil {
		return fmt.Errorf("service: update broker thresholds: %w", err)
	}
	return nil
}

// ReconnectAccount sets the account status to connecting for re-connection.
func (s *AccountService) ReconnectAccount(ctx context.Context, userID uuid.UUID, id string) error {
	_, err := s.db.Exec(ctx, `
			UPDATE mt_accounts SET account_status = 'connecting', updated_at = CURRENT_TIMESTAMP WHERE id = $1::uuid AND user_id = $2
		`, id, userID)
	if err != nil {
		return fmt.Errorf("service: reconnect account: %w", err)
	}
	return nil
}

// UpdateTradingPassword updates the trading password for an account (#38: old password verification).
func (s *AccountService) UpdateTradingPassword(ctx context.Context, userID uuid.UUID, id, oldPassword, newPassword string) error {
	tag, err := s.db.Exec(ctx, `
			UPDATE mt_accounts SET password = $4, updated_at = CURRENT_TIMESTAMP WHERE id = $1::uuid AND user_id = $2 AND password = $3
		`, id, userID, oldPassword, newPassword)
	if err != nil {
		return fmt.Errorf("service: update trading password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAccountPasswordMismatch
	}
	return nil
}

// UserOwnsAccount checks if an account belongs to the given user.
func (s *AccountService) UserOwnsAccount(ctx context.Context, userID, accountID string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM mt_accounts WHERE id = $1 AND user_id = $2)",
		accountID, userID,
	).Scan(&exists)
	return exists, err
}

// GetUserAccountIDs returns all account IDs belonging to a user.
func (s *AccountService) GetUserAccountIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.db.Query(ctx, "SELECT id FROM mt_accounts WHERE user_id = $1", userID)
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

// GetUserAccountSnapshots returns current state for all of a user's MT accounts.
func (s *AccountService) GetUserAccountSnapshots(ctx context.Context, userID string) ([]AccountSnapshot, error) {
	rows, err := s.db.Query(ctx,
		"SELECT id, account_status, balance, equity, credit, margin, free_margin, margin_level FROM mt_accounts WHERE user_id = $1",
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AccountSnapshot
	for rows.Next() {
		var a AccountSnapshot
		if err := rows.Scan(&a.ID, &a.Status, &a.Balance, &a.Equity, &a.Credit, &a.Margin, &a.FreeMargin, &a.MarginLevel); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// GetUserAccountsSummary computes an aggregated summary of a user's accounts (#30: log scan errors instead of silently skipping).
func (s *AccountService) GetUserAccountsSummary(ctx context.Context, userID string) (*UserAccountsSummary, error) {
	rows, err := s.db.Query(ctx,
		"SELECT balance, equity, account_status FROM mt_accounts WHERE user_id = $1", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sum UserAccountsSummary
	for rows.Next() {
		var balance, equity float64
		var status string
		if err := rows.Scan(&balance, &equity, &status); err != nil {
			if s.log != nil {
				s.log.Warn("GetUserAccountsSummary: scan error, skipping row", zap.Error(err))
			}
			continue
		}
		sum.TotalBalance += balance
		sum.TotalEquity += equity
		sum.TotalProfit += equity - balance
		sum.AccountCount++
		if status == "connected" {
			sum.ConnectedCount++
		}
	}
	return &sum, rows.Err()
}
