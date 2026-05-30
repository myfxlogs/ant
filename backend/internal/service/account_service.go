package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"anttrader/internal/repository"
)

// ErrAccountAlreadyBound is returned when an MT account is already bound to another user.
var ErrAccountAlreadyBound = errors.New("this trading account is already bound to another user")

// ErrAccountPasswordMismatch is returned when the old password does not match.
var ErrAccountPasswordMismatch = errors.New("old password does not match")

// AccountService provides account CRUD and lifecycle operations.
type AccountService struct {
	db      *pgxpool.Pool
	queries *repository.Queries
	log     *zap.Logger
}

// NewAccountService creates an account service backed by the given pool.
func NewAccountService(db *pgxpool.Pool) *AccountService {
	return &AccountService{
		db:      db,
		queries: repository.New(db),
	}
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
	pgID, err := stringToPgUUID(id)
	if err != nil {
		return fmt.Errorf("service: update account metrics: invalid account id: %w", err)
	}
	return s.queries.UpdateAccountMetrics(ctx, repository.UpdateAccountMetricsParams{
		ID:          pgID,
		UserID:      uuidToPgUUID(userID),
		Balance:     float64ToPgNumeric(balance),
		Equity:      float64ToPgNumeric(equity),
		Credit:      float64ToPgNumeric(credit),
		Margin:      float64ToPgNumeric(margin),
		FreeMargin:  float64ToPgNumeric(freeMargin),
		MarginLevel: float64ToPgNumeric(marginLevel),
	})
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
	pgID, err := stringToPgUUID(userID)
	if err != nil {
		return nil, fmt.Errorf("service: get account snapshots: invalid user id: %w", err)
	}
	rows, err := s.queries.GetAccountSnapshots(ctx, pgID)
	if err != nil {
		return nil, err
	}
	out := make([]AccountSnapshot, len(rows))
	for i, r := range rows {
		out[i] = AccountSnapshot{
			ID:          pgUUIDToString(r.ID),
			Status:      r.AccountStatus,
			Balance:     pgNumericToFloat64(r.Balance),
			Equity:      pgNumericToFloat64(r.Equity),
			Credit:      pgNumericToFloat64(r.Credit),
			Margin:      pgNumericToFloat64(r.Margin),
			FreeMargin:  pgNumericToFloat64(r.FreeMargin),
			MarginLevel: pgNumericToFloat64(r.MarginLevel),
		}
	}
	return out, nil
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

// --- type conversion helpers (sqlc pgtype <-> Go native) ---

func uuidToPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func stringToPgUUID(s string) (pgtype.UUID, error) {
	uid, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: uid, Valid: true}, nil
}

func pgUUIDToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}

func float64ToPgNumeric(v float64) pgtype.Numeric {
	// pgtype.Numeric implements sql.Scanner and accepts float64 directly.
	var n pgtype.Numeric
	_ = n.Scan(v)
	return n
}

func pgNumericToFloat64(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f8, err := n.Float64Value()
	if err != nil || !f8.Valid {
		return 0
	}
	return f8.Float64
}

func pgInt4ToInt32(i pgtype.Int4) int32 {
	if !i.Valid {
		return 0
	}
	return i.Int32
}

func pgTextToString(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

func pgTimestampToString(t pgtype.Timestamp) string {
	if !t.Valid {
		return ""
	}
	return t.Time.UTC().Format(time.RFC3339)
}

func pgBoolToBool(b pgtype.Bool) bool {
	return b.Valid && b.Bool
}

// mtAccountToDTO maps a sqlc-generated MtAccount to the service DTO.
func mtAccountToDTO(a repository.MtAccount) AccountDTO {
	return AccountDTO{
		ID:              pgUUIDToString(a.ID),
		UserID:          pgUUIDToString(a.UserID),
		Platform:        a.MtType,
		Broker:          pgTextToString(a.BrokerCompany),
		Login:           a.Login,
		Server:          pgTextToString(a.BrokerServer),
		IsDisabled:      pgBoolToBool(a.IsDisabled),
		Status:          a.AccountStatus,
		Balance:         pgNumericToFloat64(a.Balance),
		Equity:          pgNumericToFloat64(a.Equity),
		Credit:          pgNumericToFloat64(a.Credit),
		Margin:          pgNumericToFloat64(a.Margin),
		FreeMargin:      pgNumericToFloat64(a.FreeMargin),
		MarginLevel:     pgNumericToFloat64(a.MarginLevel),
		Leverage:        pgInt4ToInt32(a.Leverage),
		Currency:        pgTextToString(a.Currency),
		LastError:       pgTextToString(a.LastError),
		LastConnectedAt: pgTimestampToString(a.LastConnectedAt),
	}
}
