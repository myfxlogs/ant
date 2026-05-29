// Package service provides the ant v2 business service layer.
//
// Architecture (ADR-0006): platform-shared tables ∪ user-private tables.
// List methods merge both sources; detail methods route by ID to the correct table.
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrAccountAlreadyBound is returned when an MT account is already bound to another user.
var ErrAccountAlreadyBound = errors.New("this trading account is already bound to another user")

// PlatformService provides read access to platform + user strategy/factor/ai data.
type PlatformService struct {
	pg *pgxpool.Pool
}

// NewPlatformService creates a platform service backed by the given pool.
func NewPlatformService(pg *pgxpool.Pool) *PlatformService {
	return &PlatformService{pg: pg}
}

// Strategy represents a strategy visible to a user (platform ∪ user-private).
type Strategy struct {
	ID          string
	Name        string
	Description string
	Expression  string
	Category    string
	Source      string // "platform" or "user"
}

// ListStrategies returns all official platform strategies plus the user's own.
func (s *PlatformService) ListStrategies(ctx context.Context, userID string) ([]Strategy, error) {
	rows, err := s.pg.Query(ctx, `
		SELECT ps.id, ps.name, ps.description, ps.expression, ps.category, 'platform' as source
		FROM platform_strategies ps
		WHERE ps.is_official = true
		UNION ALL
		SELECT usp.strategy_id, st.name, st.description, st.expression, st.category, 'user' as source
		FROM user_strategy_publishes usp
		JOIN strategy_templates st ON st.id::text = usp.strategy_id
		WHERE usp.publisher_user_id = $1
		ORDER BY name
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("service: list strategies: %w", err)
	}
	defer rows.Close()

	var out []Strategy
	for rows.Next() {
		var s Strategy
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.Expression, &s.Category, &s.Source); err != nil {
			return nil, fmt.Errorf("service: list strategies: scan: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// UserSubscription represents a user's subscription to a platform strategy.
type UserSubscription struct {
	ID               string
	TargetUserID     string
	TargetStrategyID string
	Kind             string // "copy_trade" | "signal" | "follow"
}

// ListSubscriptions returns all active subscriptions for a user.
func (s *PlatformService) ListSubscriptions(ctx context.Context, userID string) ([]UserSubscription, error) {
	rows, err := s.pg.Query(ctx, `
		SELECT id, target_user_id, target_strategy_id, kind
		FROM user_subscriptions
		WHERE subscriber_user_id = $1 AND active = true
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("service: list subscriptions: %w", err)
	}
	defer rows.Close()

	var out []UserSubscription
	for rows.Next() {
		var sub UserSubscription
		if err := rows.Scan(&sub.ID, &sub.TargetUserID, &sub.TargetStrategyID, &sub.Kind); err != nil {
			return nil, fmt.Errorf("service: list subscriptions: scan: %w", err)
		}
		out = append(out, sub)
	}
	return out, rows.Err()
}

// Admin represents a platform administrator.
type Admin struct {
	UserID string
	Scope  string // "platform:admin"
}

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

// ListAccounts returns all accounts belonging to the given user.
func (s *PlatformService) ListAccounts(ctx context.Context, userID uuid.UUID) ([]AccountDTO, error) {
	rows, err := s.pg.Query(ctx, `
		SELECT id, user_id, mt_type, broker_company, login, broker_server, is_disabled,
		       account_status, balance, equity, COALESCE(credit, 0), margin, free_margin, margin_level, leverage, currency, COALESCE(last_error, ''),
		       COALESCE(last_connected_at::text, '')
		FROM mt_accounts WHERE user_id = $1 ORDER BY mt_type, login
	`, userID)
	if err != nil { return nil, err }
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

// ConnectAccount marks an account as ready for connection.
func (s *PlatformService) ConnectAccount(ctx context.Context, userID uuid.UUID, accountID string) error {
	var exists bool
	err := s.pg.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM mt_accounts WHERE id = $1::uuid AND user_id = $2)", accountID, userID).Scan(&exists)
	if err != nil { return err }
	if !exists { return fmt.Errorf("account not found: %s", accountID) }
	return nil
}

// GetAccount returns a single account by ID.
func (s *PlatformService) GetAccount(ctx context.Context, userID uuid.UUID, accountID string) (*AccountDTO, error) {
	var a AccountDTO
	err := s.pg.QueryRow(ctx, `
		SELECT id, user_id, mt_type, broker_company, login, broker_server, is_disabled,
		       account_status, balance, equity, COALESCE(credit, 0), margin, free_margin, margin_level, leverage, currency, COALESCE(last_error, ''),
		       COALESCE(last_connected_at::text, '')
		FROM mt_accounts WHERE id = $1::uuid AND user_id = $2
	`, accountID, userID).Scan(&a.ID, &a.UserID, &a.Platform, &a.Broker, &a.Login, &a.Server, &a.IsDisabled,
		&a.Status, &a.Balance, &a.Equity, &a.Credit, &a.Margin, &a.FreeMargin, &a.MarginLevel, &a.Leverage, &a.Currency, &a.LastError,
		&a.LastConnectedAt)
	if err != nil { return nil, err }
	return &a, nil
}

// IsAdmin checks if a user has admin privileges.
// CreateAccount inserts a new MT account row and returns the generated ID.
func (s *PlatformService) CreateAccount(ctx context.Context, userID uuid.UUID, login, password, mtType, brokerCompany, brokerServer, brokerHost string) (string, error) {
	var id string
	err := s.pg.QueryRow(ctx, `
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

// UpdateAccount updates broker fields and disabled status for an account.
func (s *PlatformService) UpdateAccount(ctx context.Context, userID uuid.UUID, id, brokerCompany, brokerServer, brokerHost string, isDisabled *bool) error {
	_, err := s.pg.Exec(ctx, `
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

// UpdateAccountInfo updates balance/equity/margin/leverage/currency + status after MT verification.
func (s *PlatformService) UpdateAccountInfo(ctx context.Context, userID uuid.UUID, id string, balance, equity, credit, margin, freeMargin float64, leverage int64, currency string) error {
	_, err := s.pg.Exec(ctx, `
		UPDATE mt_accounts SET
			balance = COALESCE(NULLIF($3::float8, 0), balance),
			equity  = COALESCE(NULLIF($4::float8, 0), equity),
			credit  = COALESCE(NULLIF($5::float8, 0), credit),
			margin  = COALESCE(NULLIF($6::float8, 0), margin),
			free_margin = COALESCE(NULLIF($7::float8, 0), free_margin),
			leverage = COALESCE(NULLIF($8::int4, 0), leverage),
			currency = COALESCE(NULLIF($9, ''), currency),
			account_status = 'connected',
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1::uuid AND user_id = $2
	`, id, userID, balance, equity, credit, margin, freeMargin, leverage, currency)
	if err != nil {
		return fmt.Errorf("service: update account info: %w", err)
	}
	return nil
}

// AccountCredentials holds the fields needed to verify an MT account password.
type AccountCredentials struct {
	Login      string
	Platform   string
	BrokerHost string
}

// GetAccountCredentials returns the credentials needed for MT password verification.
func (s *PlatformService) GetAccountCredentials(ctx context.Context, userID uuid.UUID, id string) (*AccountCredentials, error) {
	var c AccountCredentials
	err := s.pg.QueryRow(ctx,
		`SELECT login, mt_type, broker_host FROM mt_accounts WHERE id = $1::uuid AND user_id = $2`,
		id, userID).Scan(&c.Login, &c.Platform, &c.BrokerHost)
	if err != nil {
		return nil, fmt.Errorf("service: get account credentials: %w", err)
	}
	return &c, nil
}

// DeleteAccount removes an MT account and all its related data.
// Related tables without ON DELETE CASCADE are cleaned up first.
func (s *PlatformService) DeleteAccount(ctx context.Context, userID uuid.UUID, id string) error {
	// Delete related rows from tables that lack ON DELETE CASCADE.
	related := []string{
		`DELETE FROM account_balance_history WHERE account_id = $1::uuid`,
		`DELETE FROM account_connection_logs WHERE account_id = $1::uuid`,
		`DELETE FROM strategy_execution_logs WHERE account_id = $1::uuid`,
		`DELETE FROM order_history WHERE account_id = $1::uuid`,
	}
	for _, q := range related {
		if _, err := s.pg.Exec(ctx, q, id); err != nil {
			return fmt.Errorf("service: delete account: cleanup: %w", err)
		}
	}

	_, err := s.pg.Exec(ctx, `DELETE FROM mt_accounts WHERE id = $1::uuid AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("service: delete account: %w", err)
	}
	return nil
}

// DisconnectAccount sets the account status to disconnected.
func (s *PlatformService) DisconnectAccount(ctx context.Context, userID uuid.UUID, id string) error {
	_, err := s.pg.Exec(ctx, `
		UPDATE mt_accounts SET account_status = 'disconnected', updated_at = CURRENT_TIMESTAMP WHERE id = $1::uuid AND user_id = $2
	`, id, userID)
	if err != nil {
		return fmt.Errorf("service: disconnect account: %w", err)
	}
	return nil
}

// ReconnectAccount sets the account status to connecting for re-connection.
func (s *PlatformService) ReconnectAccount(ctx context.Context, userID uuid.UUID, id string) error {
	_, err := s.pg.Exec(ctx, `
		UPDATE mt_accounts SET account_status = 'connecting', updated_at = CURRENT_TIMESTAMP WHERE id = $1::uuid AND user_id = $2
	`, id, userID)
	if err != nil {
		return fmt.Errorf("service: reconnect account: %w", err)
	}
	return nil
}

// UpdateTradingPassword updates the trading password for an account.
func (s *PlatformService) UpdateTradingPassword(ctx context.Context, userID uuid.UUID, id, newPassword string) error {
	_, err := s.pg.Exec(ctx, `
		UPDATE mt_accounts SET password = $3, updated_at = CURRENT_TIMESTAMP WHERE id = $1::uuid AND user_id = $2
	`, id, userID, newPassword)
	if err != nil {
		return fmt.Errorf("service: update trading password: %w", err)
	}
	return nil
}

func (s *PlatformService) IsAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	var count int
	err := s.pg.QueryRow(ctx, "SELECT count(*) FROM admins WHERE user_id = $1", userID).Scan(&count)
	return count > 0, err
}

// UserOwnsAccount checks if an account belongs to the given user.
func (s *PlatformService) UserOwnsAccount(ctx context.Context, userID, accountID string) (bool, error) {
	var exists bool
	err := s.pg.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM mt_accounts WHERE id = $1 AND user_id = $2)",
		accountID, userID,
	).Scan(&exists)
	return exists, err
}

// GetUserAccountIDs returns all account IDs belonging to a user.
func (s *PlatformService) GetUserAccountIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.pg.Query(ctx, "SELECT id FROM mt_accounts WHERE user_id = $1", userID)
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

// AccountSnapshot holds the current state of an MT account from the database.
type AccountSnapshot struct {
	ID           string
	Status       string
	Balance      float64
	Equity       float64
	Credit       float64
	Margin       float64
	FreeMargin   float64
	MarginLevel  float64
}

// GetUserAccountSnapshots returns current state for all of a user's MT accounts.
func (s *PlatformService) GetUserAccountSnapshots(ctx context.Context, userID string) ([]AccountSnapshot, error) {
	rows, err := s.pg.Query(ctx,
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

// UserAccountsSummary holds the aggregated account summary for a user.
type UserAccountsSummary struct {
	TotalBalance   float64
	TotalEquity    float64
	TotalProfit    float64
	AccountCount   int32
	ConnectedCount int32
}

// GetUserAccountsSummary computes an aggregated summary of a user's accounts.
func (s *PlatformService) GetUserAccountsSummary(ctx context.Context, userID string) (*UserAccountsSummary, error) {
	rows, err := s.pg.Query(ctx,
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
