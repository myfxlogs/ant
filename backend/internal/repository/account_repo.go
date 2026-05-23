package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"anttrader/internal/model"
)

var (
	ErrAccountNotFound      = errors.New("account not found")
	ErrAccountAlreadyExists = errors.New("account already exists")
)

type AccountRepository struct {
	db *sqlx.DB
}

func NewAccountRepository(db *sqlx.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (r *AccountRepository) Create(ctx context.Context, account *model.MTAccount) error {
	query := `
		INSERT INTO mt_accounts (
			id, user_id, mt_type, broker_company, broker_server, broker_host,
			login, password, alias, is_disabled, balance, credit, equity,
			margin, free_margin, margin_level, leverage, currency,
			account_method, account_type, is_investor, account_status, stream_status,
			mt_token, last_error, last_connected_at, last_checked_at,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29
		)`

	now := time.Now()
	if account.ID == uuid.Nil {
		account.ID = uuid.New()
	}
	account.CreatedAt = now
	account.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, query,
		account.ID, account.UserID, account.MTType, account.BrokerCompany,
		account.BrokerServer, account.BrokerHost, account.Login, account.Password,
		account.Alias, account.IsDisabled, account.Balance, account.Credit,
		account.Equity, account.Margin, account.FreeMargin, account.MarginLevel,
		account.Leverage, account.Currency, account.AccountMethod, account.AccountType,
		account.IsInvestor, account.AccountStatus, account.StreamStatus, account.MTToken,
		account.LastError, account.LastConnectedAt, account.LastCheckedAt,
		account.CreatedAt, account.UpdatedAt,
	)

	return err
}

func (r *AccountRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.MTAccount, error) {
	query := `SELECT * FROM mt_accounts WHERE id = $1`
	var account model.MTAccount
	err := r.db.GetContext(ctx, &account, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}
	return &account, nil
}

func (r *AccountRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.MTAccount, error) {
	query := `SELECT * FROM mt_accounts WHERE user_id = $1 ORDER BY is_disabled ASC, created_at DESC`
	var accounts []*model.MTAccount
	err := r.db.SelectContext(ctx, &accounts, query, userID)
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

func (r *AccountRepository) GetByLoginAndHost(ctx context.Context, login, brokerHost string) (*model.MTAccount, error) {
	query := `SELECT * FROM mt_accounts WHERE login = $1 AND broker_host = $2`
	var account model.MTAccount
	err := r.db.GetContext(ctx, &account, query, login, brokerHost)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}
	return &account, nil
}

func (r *AccountRepository) Update(ctx context.Context, account *model.MTAccount) error {
	query := `
		UPDATE mt_accounts SET
			balance = $2, credit = $3, equity = $4, margin = $5,
			free_margin = $6, margin_level = $7, leverage = $8,
			currency = $9, account_type = $10, is_investor = $11,
			updated_at = $12
		WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query,
		account.ID, account.Balance, account.Credit, account.Equity,
		account.Margin, account.FreeMargin, account.MarginLevel,
		account.Leverage, account.Currency, account.AccountType,
		account.IsInvestor, time.Now())
	return err
}

func (r *AccountRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, lastError string) error {
	query := `
		UPDATE mt_accounts SET
			account_status = $2, last_error = $3, updated_at = $4
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id, status, lastError, time.Now())
	return err
}

func (r *AccountRepository) UpdateDisabled(ctx context.Context, id uuid.UUID, isDisabled bool) error {
	query := `
		UPDATE mt_accounts SET
			is_disabled = $2, updated_at = $3
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id, isDisabled, time.Now())
	return err
}

func (r *AccountRepository) UpdateConnectedAt(ctx context.Context, id uuid.UUID, connectedAt time.Time) error {
	query := `
		UPDATE mt_accounts SET
			last_connected_at = $2, updated_at = $3
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id, connectedAt, time.Now())
	return err
}

func (r *AccountRepository) UpdateToken(ctx context.Context, id uuid.UUID, token string) error {
	query := `
		UPDATE mt_accounts SET
			mt_token = $2, account_status = 'connected',
			last_connected_at = $3, updated_at = $4
		WHERE id = $1`

	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, id, token, now, now)
	return err
}

func (r *AccountRepository) UpdateAccountInfo(ctx context.Context, id uuid.UUID, balance, credit, equity, margin, freeMargin, marginLevel float64, leverage int, currency string) error {
	query := `
		UPDATE mt_accounts SET
			balance = $2, credit = $3, equity = $4, margin = $5,
			free_margin = $6, margin_level = $7, leverage = $8, currency = $9,
			updated_at = $10
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id, balance, credit, equity, margin, freeMargin, marginLevel, leverage, currency, time.Now())
	return err
}

func (r *AccountRepository) UpdateAccountFullInfo(ctx context.Context, id uuid.UUID, balance, credit, equity, margin, freeMargin, marginLevel float64, leverage int, currency, accountType string, isInvestor bool) error {
	query := `
		UPDATE mt_accounts SET
			balance = $2, credit = $3, equity = $4, margin = $5,
			free_margin = $6, margin_level = $7, leverage = $8, currency = $9,
			account_type = $10, is_investor = $11,
			updated_at = $12
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id, balance, credit, equity, margin, freeMargin, marginLevel, leverage, currency, accountType, isInvestor, time.Now())
	return err
}

func (r *AccountRepository) UpdateAccountMethod(ctx context.Context, id uuid.UUID, accountMethod string) error {
	query := `UPDATE mt_accounts SET account_method = $2, updated_at = $3 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, accountMethod, time.Now())
	return err
}

func (r *AccountRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `DELETE FROM account_connection_logs WHERE account_id = $1`, id)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM trade_records WHERE account_id = $1`, id)
	if err != nil {
		return err
	}

	result, err := tx.ExecContext(ctx, `DELETE FROM mt_accounts WHERE id = $1`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrAccountNotFound
	}

	return tx.Commit()
}

func (r *AccountRepository) SetDisabled(ctx context.Context, id uuid.UUID, disabled bool) error {
	query := `UPDATE mt_accounts SET is_disabled = $2, updated_at = $3 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, disabled, time.Now())
	return err
}

func (r *AccountRepository) GetAll(ctx context.Context) ([]*model.MTAccount, error) {
	query := `
		SELECT 
			id, user_id, mt_type, broker_company, broker_server, broker_host,
			login, password, alias, is_disabled, balance, credit, equity,
			margin, free_margin, margin_level, leverage, currency,
			account_method, account_type, is_investor, account_status, stream_status,
			mt_token, last_error, last_connected_at, last_checked_at,
			created_at, updated_at
		FROM mt_accounts 
		ORDER BY created_at DESC
	`

	var accounts []*model.MTAccount
	err := r.db.SelectContext(ctx, &accounts, query)
	return accounts, err
}

func (r *AccountRepository) GetAllActive(ctx context.Context) ([]*model.MTAccount, error) {
	query := `
		SELECT 
			id, user_id, mt_type, broker_company, broker_server, broker_host,
			login, password, alias, is_disabled, balance, credit, equity,
			margin, free_margin, margin_level, leverage, currency,
			account_method, account_type, is_investor, account_status, stream_status,
			mt_token, last_error, last_connected_at, last_checked_at,
			created_at, updated_at
		FROM mt_accounts 
		WHERE is_disabled = false
		ORDER BY created_at DESC
	`

	var accounts []*model.MTAccount
	err := r.db.SelectContext(ctx, &accounts, query)
	return accounts, err
}

func (r *AccountRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM mt_accounts WHERE user_id = $1`
	var count int
	err := r.db.GetContext(ctx, &count, query, userID)
	return count, err
}

func (r *AccountRepository) UpdateAccountType(ctx context.Context, id uuid.UUID, accountType string, isInvestor bool) error {
	query := `
		UPDATE mt_accounts SET
			account_type = $2, is_investor = $3, updated_at = $4
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id, accountType, isInvestor, time.Now())
	return err
}

// UpdatePassword 只更新存储的 MT 登录密码。调用前请务必在上层先用新密码做一次
// Connect 测试，确认密码可用后再落库。
func (r *AccountRepository) UpdatePassword(ctx context.Context, id uuid.UUID, password string) error {
	query := `UPDATE mt_accounts SET password = $2, updated_at = $3 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, password, time.Now())
	return err
}

// UpdateIsInvestor 仅更新 is_investor 字段，供 VerifyTradePermission 之类的轻量验证使用。
func (r *AccountRepository) UpdateIsInvestor(ctx context.Context, id uuid.UUID, isInvestor bool) error {
	query := `UPDATE mt_accounts SET is_investor = $2, updated_at = $3 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, isInvestor, time.Now())
	return err
}
