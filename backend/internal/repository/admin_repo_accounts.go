package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"anttrader/internal/model"
)

type AccountWithUser struct {
	model.MTAccount
	UserEmail    string  `json:"user_email" db:"user_email"`
	UserNickname *string `json:"user_nickname" db:"user_nickname"`
}

var accountCols = `ma.id, ma.user_id, ma.mt_type, ma.broker_company, ma.broker_server,
		   ma.broker_host, ma.login, ma.password, ma.alias, ma.is_disabled,
		   ma.balance, ma.credit, ma.equity, ma.margin, ma.free_margin,
		   ma.margin_level, ma.leverage, ma.currency, ma.account_method,
		   ma.is_investor, ma.account_status, ma.stream_status, ma.mt_token,
		   ma.last_error, ma.last_connected_at, ma.last_checked_at,
		   ma.created_at, ma.updated_at, ma.account_type`

func (r *AdminRepository) ListAccounts(ctx context.Context, params *model.AccountListParams) ([]*AccountWithUser, int64, error) {
	page, pageSize := normalizePage(params.Page, params.PageSize)
	offset := (page - 1) * pageSize

	countQuery := `SELECT COUNT(*) FROM mt_accounts ma JOIN users u ON ma.user_id = u.id WHERE 1=1`
	query := fmt.Sprintf(`SELECT %s, u.email as user_email, u.nickname as user_nickname
			      FROM mt_accounts ma
			      JOIN users u ON ma.user_id = u.id
			      WHERE 1=1`, accountCols)

	var conds []string
	var args []interface{}

	addCond := func(col, val string) {
		if val == "" {
			return
		}
		i := len(args) + 1
		conds = append(conds, fmt.Sprintf(" %s = $%d", col, i))
		args = append(args, val)
	}

	if params.Search != "" {
		i := len(args) + 1
		conds = append(conds, fmt.Sprintf(" (ma.login ILIKE $%d OR u.email ILIKE $%d OR u.nickname ILIKE $%d)", i, i, i))
		args = append(args, "%"+params.Search+"%")
	}
	addCond("ma.account_status", params.Status)
	addCond("ma.mt_type", params.MTType)
	addCond("ma.user_id", params.UserID)

	for _, c := range conds {
		countQuery += " AND" + c
		query += " AND" + c
	}

	var total int64
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	i := len(args) + 1
	query += fmt.Sprintf(" ORDER BY ma.created_at DESC LIMIT $%d OFFSET $%d", i, i+1)
	args = append(args, pageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var accounts []*AccountWithUser
	for rows.Next() {
		var a AccountWithUser
		if err := rows.Scan(
			&a.ID, &a.UserID, &a.MTType, &a.BrokerCompany, &a.BrokerServer,
			&a.BrokerHost, &a.Login, &a.Password, &a.Alias, &a.IsDisabled,
			&a.Balance, &a.Credit, &a.Equity, &a.Margin, &a.FreeMargin,
			&a.MarginLevel, &a.Leverage, &a.Currency, &a.AccountMethod,
			&a.IsInvestor, &a.AccountStatus, &a.StreamStatus, &a.MTToken,
			&a.LastError, &a.LastConnectedAt, &a.LastCheckedAt,
			&a.CreatedAt, &a.UpdatedAt, &a.AccountType,
			&a.UserEmail, &a.UserNickname,
		); err != nil {
			return nil, 0, err
		}
		accounts = append(accounts, &a)
	}
	return accounts, total, nil
}

func (r *AdminRepository) SetAccountStatus(ctx context.Context, id uuid.UUID, status string) error {
	result, err := r.db.Exec(ctx,
		`UPDATE mt_accounts SET account_status = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`,
		id, status)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrAccountNotFound
	}
	return nil
}

func (r *AdminRepository) GetAccountByID(ctx context.Context, id uuid.UUID) (*AccountWithUser, error) {
	query := fmt.Sprintf(`SELECT %s, u.email as user_email, u.nickname as user_nickname
			      FROM mt_accounts ma
			      JOIN users u ON ma.user_id = u.id
			      WHERE ma.id = $1`, accountCols)
	var a AccountWithUser
	err := r.db.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.UserID, &a.MTType, &a.BrokerCompany, &a.BrokerServer,
		&a.BrokerHost, &a.Login, &a.Password, &a.Alias, &a.IsDisabled,
		&a.Balance, &a.Credit, &a.Equity, &a.Margin, &a.FreeMargin,
		&a.MarginLevel, &a.Leverage, &a.Currency, &a.AccountMethod,
		&a.IsInvestor, &a.AccountStatus, &a.StreamStatus, &a.MTToken,
		&a.LastError, &a.LastConnectedAt, &a.LastCheckedAt,
		&a.CreatedAt, &a.UpdatedAt,
		&a.UserEmail, &a.UserNickname,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}
	return &a, nil
}
