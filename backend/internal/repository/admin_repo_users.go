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
			&u.Role, &u.Status, &u.AccountNumber, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt, &u.MTAccountCount); err != nil {
			return nil, 0, err
		}
		users = append(users, &u)
	}
	return users, total, nil
}

func buildUserListFilters(params *model.UserListParams) (countQ, query string, args []interface{}) {
	countQ = `SELECT COUNT(*) FROM users WHERE 1=1`
	query = `SELECT u.id, u.email, u.password_hash, u.nickname, u.avatar, u.role, u.status, u.account_number, u.last_login_at, u.created_at, u.updated_at, COUNT(ma.id) as mt_account_count FROM users u LEFT JOIN mt_accounts ma ON u.id = ma.user_id WHERE 1=1`
	var conds []string
	addCond := func(col, val string) {
		if val == "" { return }
		conds = append(conds, fmt.Sprintf(" %s = $%d", col, len(args)+1))
		args = append(args, val)
	}
	if params.Search != "" {
		conds = append(conds, fmt.Sprintf(" (email ILIKE $%d OR nickname ILIKE $%d)", len(args)+1, len(args)+1))
		args = append(args, "%"+params.Search+"%")
	}
	addCond("u.status", params.Status)
	addCond("u.role", params.Role)
	for _, c := range conds { countQ += " AND" + c; query += " AND" + c }
	return
}

func (r *AdminRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user := &model.User{}
	err := r.db.QueryRow(ctx,
		`SELECT id, email, password_hash, nickname, avatar, role, status,
		        account_number, last_login_at, created_at, updated_at
		 FROM users WHERE id = $1`, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Nickname, &user.Avatar,
		&user.Role, &user.Status, &user.AccountNumber, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
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
			 SET email = $2, nickname = $3, avatar = $4, role = $5, status = $6, updated_at = CURRENT_TIMESTAMP
			 WHERE id = $1
			 RETURNING updated_at`,
			user.ID, user.Email, user.Nickname, user.Avatar, user.Role, user.Status,
		).Scan(&user.UpdatedAt)
	}

// CountAdmins returns the number of users with the "admin" role.
func (r *AdminRepository) CountAdmins(ctx context.Context) (int32, error) {
	var count int32
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role = 'admin'`).Scan(&count)
	return count, err
}

func (r *AdminRepository) DeleteUser(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// DeleteUsers deletes multiple users by ID in a single query.
// Returns the count of actually deleted rows.
func (r *AdminRepository) DeleteUsers(ctx context.Context, ids []uuid.UUID) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	ct, err := r.db.Exec(ctx, `DELETE FROM users WHERE id = ANY($1)`, ids)
	if err != nil {
		return 0, fmt.Errorf("delete users: %w", err)
	}
	return ct.RowsAffected(), nil
}

func (r *AdminRepository) SetUserStatus(ctx context.Context, id uuid.UUID, status string) error {
	result, err := r.db.Exec(ctx,
		`UPDATE users SET status = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`,
		id, status)
	if err != nil {
		return fmt.Errorf("set user status: %w", err)
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
		return fmt.Errorf("reset user password: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}
