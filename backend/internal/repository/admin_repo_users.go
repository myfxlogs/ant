package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"anttrader/internal/model"
)

type UserWithAccounts struct {
	model.User
	MTAccountCount int `json:"mt_account_count" db:"mt_account_count"`
}

func (r *AdminRepository) ListUsers(ctx context.Context, params *model.UserListParams) ([]*UserWithAccounts, int64, error) {
	page, pageSize := normalizePage(params.Page, params.PageSize)
	offset := (page - 1) * pageSize

	countQuery := `SELECT COUNT(*) FROM users WHERE 1=1`
	query := `SELECT u.id, u.email, u.password_hash, u.nickname, u.avatar,
		       u.role, u.status, u.last_login_at, u.created_at, u.updated_at,
		       COUNT(ma.id) as mt_account_count
		FROM users u
		LEFT JOIN mt_accounts ma ON u.id = ma.user_id
		WHERE 1=1`

	var conds []string
	var args []interface{}

	addCond := func(field, col string, val string) {
		if val == "" {
			return
		}
		i := len(args) + 1
		conds = append(conds, fmt.Sprintf(" %s = $%d", col, i))
		args = append(args, val)
	}

	if params.Search != "" {
		i := len(args) + 1
		conds = append(conds, fmt.Sprintf(" (email ILIKE $%d OR nickname ILIKE $%d)", i, i))
		args = append(args, "%"+params.Search+"%")
	}
	addCond("status", "u.status", params.Status)
	addCond("role", "u.role", params.Role)

	for _, c := range conds {
		countQuery += " AND" + c
		query += " AND" + c
	}

	var total int64
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	i := len(args) + 1
	query += fmt.Sprintf(" GROUP BY u.id ORDER BY u.created_at DESC LIMIT $%d OFFSET $%d", i, i+1)
	args = append(args, pageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*UserWithAccounts
	for rows.Next() {
		var u UserWithAccounts
		if err := rows.Scan(
			&u.ID, &u.Email, &u.PasswordHash, &u.Nickname, &u.Avatar,
			&u.Role, &u.Status, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
			&u.MTAccountCount,
		); err != nil {
			return nil, 0, err
		}
		users = append(users, &u)
	}
	return users, total, nil
}

func (r *AdminRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user := &model.User{}
	err := r.sqlxDB.GetContext(ctx, user,
		`SELECT id, email, password_hash, nickname, avatar, role, status,
		        last_login_at, created_at, updated_at
		 FROM users WHERE id = $1`, id)
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
		`INSERT INTO users (email, password_hash, nickname, avatar, role, status)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at, updated_at`,
		user.Email, user.PasswordHash, user.Nickname, user.Avatar, user.Role, user.Status,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *AdminRepository) UpdateUser(ctx context.Context, user *model.User) error {
	return r.db.QueryRow(ctx,
		`UPDATE users
		 SET nickname = $2, avatar = $3, role = $4, status = $5, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $1
		 RETURNING updated_at`,
		user.ID, user.Nickname, user.Avatar, user.Role, user.Status,
	).Scan(&user.UpdatedAt)
}

func (r *AdminRepository) DeleteUser(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *AdminRepository) SetUserStatus(ctx context.Context, id uuid.UUID, status string) error {
	result, err := r.db.Exec(ctx,
		`UPDATE users SET status = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`,
		id, status)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *AdminRepository) ResetUserPassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	result, err := r.db.Exec(ctx,
		`UPDATE users SET password_hash = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`,
		id, passwordHash)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}
