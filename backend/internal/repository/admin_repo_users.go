package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"anttrader/internal/model"
)

// BeginTx starts a new database transaction.
func (r *AdminRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}

type UserWithAccounts struct {
	model.User
	MTAccountCount int `json:"mt_account_count" db:"mt_account_count"`
}

func (r *AdminRepository) ListUsers(ctx context.Context, params *model.UserListParams) ([]*UserWithAccounts, int64, error) {
	page, pageSize := normalizePage(params.Page, params.PageSize)
	offset := (page - 1) * pageSize

	countQ, query, args := buildUserListFilters(params)
	var total int64
	if err := r.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil { return nil, 0, err }

	i := len(args) + 1
	query += fmt.Sprintf(" GROUP BY u.id ORDER BY u.created_at DESC LIMIT $%d OFFSET $%d", i, i+1)
	args = append(args, pageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil { return nil, 0, err }
	defer rows.Close()
	var users []*UserWithAccounts
	for rows.Next() {
		var u UserWithAccounts
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Nickname, &u.Avatar,
			&u.Role, &u.Status, &u.AccountNumber, &u.LastLoginAt, &u.DeletedAt, &u.CreatedAt, &u.UpdatedAt, &u.MTAccountCount); err != nil {
			return nil, 0, err
		}
		users = append(users, &u)
	}
	return users, total, nil
}

func buildUserListFilters(params *model.UserListParams) (countQ, query string, args []interface{}) {
	countQ = `SELECT COUNT(*) FROM users u WHERE 1=1`
	query = `SELECT u.id, u.email, u.password_hash, u.nickname, u.avatar, u.role, u.status, u.account_number, u.last_login_at, u.deleted_at, u.created_at, u.updated_at, COUNT(ma.id) as mt_account_count FROM users u LEFT JOIN mt_accounts ma ON u.id = ma.user_id WHERE 1=1`
	var conds []string
	addCond := func(col, val string) {
		if val == "" { return }
		conds = append(conds, fmt.Sprintf(" %s = $%d", col, len(args)+1))
		args = append(args, val)
	}
	if params.Search != "" {
		conds = append(conds, fmt.Sprintf(" (u.email ILIKE $%d OR u.nickname ILIKE $%d OR u.account_number ILIKE $%d)", len(args)+1, len(args)+1, len(args)+1))
		args = append(args, "%"+params.Search+"%")
	}
	addCond("u.status", params.Status)
	addCond("u.role", params.Role)
	// Three-state deleted filter: "" (default) = active only, "deleted" = deleted only, "all" = both.
	switch params.DeletedFilter {
	case "deleted":
		conds = append(conds, " u.deleted_at IS NOT NULL")
	case "all":
		// no filter — show both active and deleted
	default:
		conds = append(conds, " u.deleted_at IS NULL")
	}
	for _, c := range conds { countQ += " AND" + c; query += " AND" + c }
	return
}

func (r *AdminRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user := &model.User{}
	err := r.db.QueryRow(ctx,
		`SELECT id, email, password_hash, nickname, avatar, role, status,
		        account_number, last_login_at, deleted_at, created_at, updated_at
		 FROM users WHERE id = $1 AND deleted_at IS NULL`, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Nickname, &user.Avatar,
		&user.Role, &user.Status, &user.AccountNumber, &user.LastLoginAt, &user.DeletedAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (r *AdminRepository) CreateUser(ctx context.Context, user *model.User) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, nickname, avatar, role, status, account_number)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, created_at, updated_at`,
		user.Email, user.PasswordHash, user.Nickname, user.Avatar, user.Role, user.Status, user.AccountNumber,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *AdminRepository) UpdateUser(ctx context.Context, user *model.User) error {
		return r.db.QueryRow(ctx,
			`UPDATE users
			 SET email = $2, nickname = $3, avatar = $4, role = $5, status = $6,
			     account_number = $7, updated_at = CURRENT_TIMESTAMP
			 WHERE id = $1
			 RETURNING updated_at`,
			user.ID, user.Email, user.Nickname, user.Avatar, user.Role, user.Status, user.AccountNumber,
		).Scan(&user.UpdatedAt)
	}

// CountAdmins returns the number of users with the "admin" role (excluding soft-deleted).
func (r *AdminRepository) CountAdmins(ctx context.Context) (int32, error) {
	var count int32
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role = 'admin' AND deleted_at IS NULL`).Scan(&count)
	return count, err
}

