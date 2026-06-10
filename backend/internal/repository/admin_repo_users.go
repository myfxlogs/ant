package repository

import (
	"context"
	"encoding/json"
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
		conds = append(conds, fmt.Sprintf(" (email ILIKE $%d OR nickname ILIKE $%d)", len(args)+1, len(args)+1))
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
			 SET email = $2, nickname = $3, avatar = $4, role = $5, status = $6, updated_at = CURRENT_TIMESTAMP
			 WHERE id = $1
			 RETURNING updated_at`,
			user.ID, user.Email, user.Nickname, user.Avatar, user.Role, user.Status,
		).Scan(&user.UpdatedAt)
	}

// CountAdmins returns the number of users with the "admin" role (excluding soft-deleted).
func (r *AdminRepository) CountAdmins(ctx context.Context) (int32, error) {
	var count int32
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role = 'admin' AND deleted_at IS NULL`).Scan(&count)
	return count, err
}

// DeleteUser soft-deletes a user by setting deleted_at = NOW().
// Returns ErrUserNotFound if the user doesn't exist or is already deleted.
func (r *AdminRepository) DeleteUser(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, `UPDATE users SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// DeleteUsers batch soft-deletes multiple users by ID.
// Returns the count of users that were actually soft-deleted (not already deleted).
func (r *AdminRepository) DeleteUsers(ctx context.Context, ids []uuid.UUID) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	ct, err := r.db.Exec(ctx, `UPDATE users SET deleted_at = NOW() WHERE id = ANY($1) AND deleted_at IS NULL`, ids)
	if err != nil {
		return 0, fmt.Errorf("delete users: %w", err)
	}
	return ct.RowsAffected(), nil
}

// RestoreUser clears the soft-delete marker for a previously deleted user.
func (r *AdminRepository) RestoreUser(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, `UPDATE users SET deleted_at = NULL WHERE id = $1 AND deleted_at IS NOT NULL`, id)
	if err != nil {
		return fmt.Errorf("restore user: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// GetAffectedTableCounts returns the number of rows in each child table that
// reference the given user ID. Uses a single UNION ALL query (one DB round-trip)
// instead of N individual queries.
func (r *AdminRepository) GetAffectedTableCounts(ctx context.Context, userID uuid.UUID) (map[string]int64, error) {
	const query = `
		SELECT 'account_connection_logs', COUNT(*) FROM account_connection_logs WHERE user_id = $1
		UNION ALL
		SELECT 'admins', COUNT(*) FROM admins WHERE user_id = $1
		UNION ALL
		SELECT 'api_keys', COUNT(*) FROM api_keys WHERE user_id = $1
		UNION ALL
		SELECT 'order_history', COUNT(*) FROM order_history WHERE user_id = $1
		UNION ALL
		SELECT 'strategy_execution_logs', COUNT(*) FROM strategy_execution_logs WHERE user_id = $1
		UNION ALL
		SELECT 'system_operation_logs', COUNT(*) FROM system_operation_logs WHERE user_id = $1
		UNION ALL
		SELECT 'user_ai_agents', COUNT(*) FROM user_ai_agents WHERE user_id = $1
		UNION ALL
		SELECT 'user_strategy_publishes', COUNT(*) FROM user_strategy_publishes WHERE user_id = $1
		UNION ALL
		SELECT 'user_subscriptions', COUNT(*) FROM user_subscriptions WHERE subscriber_user_id = $1 OR target_user_id = $1
		UNION ALL
		SELECT 'wallet_transactions', COUNT(*) FROM wallet_transactions WHERE user_id = $1
		UNION ALL
		SELECT 'marketplace_strategies', COUNT(*) FROM marketplace_strategies WHERE publisher_id = $1
		UNION ALL
		SELECT 'platform_strategies', COUNT(*) FROM platform_strategies WHERE published_by = $1
		UNION ALL
		SELECT 'sanctioned_countries', COUNT(*) FROM sanctioned_countries WHERE added_by = $1
		UNION ALL
		SELECT 'user_jurisdiction', COUNT(*) FROM user_jurisdiction WHERE kyc_verified_by = $1
		UNION ALL
		SELECT 'mt_accounts', COUNT(*) FROM mt_accounts WHERE user_id = $1
		UNION ALL
		SELECT 'user_wallets', COUNT(*) FROM user_wallets WHERE user_id = $1
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("count affected tables: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var name string
		var count int64
		if err := rows.Scan(&name, &count); err != nil {
			return nil, fmt.Errorf("scan affected count: %w", err)
		}
		if count > 0 {
			result[name] = count
		}
	}
	return result, rows.Err()
}

// HardDeleteExpiredUsers physically deletes users soft-deleted more than
// retentionDays ago. At this point, CASCADE/SET NULL FKs from migrations
// 149/150 take effect, cleaning up all dependent data.
func (r *AdminRepository) HardDeleteExpiredUsers(ctx context.Context, retentionDays int) (int64, error) {
	query := fmt.Sprintf(`DELETE FROM users WHERE deleted_at IS NOT NULL AND deleted_at < NOW() - INTERVAL '%d days'`, retentionDays)
	ct, err := r.db.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("hard delete expired users: %w", err)
	}
	return ct.RowsAffected(), nil
}

// InsertAuditLog records an admin action in the audit log.
func (r *AdminRepository) InsertAuditLog(ctx context.Context, actorID uuid.UUID, action, targetID, targetEmail string, affectedData map[string]int64) error {
	dataJSON, err := json.Marshal(affectedData)
	if err != nil {
		return fmt.Errorf("marshal affected data: %w", err)
	}
	_, err = r.db.Exec(ctx,
		`INSERT INTO admin_audit_log (actor_id, action, target_id, target_email, affected_data)
		 VALUES ($1, $2, $3, $4, $5)`,
		actorID, action, targetID, targetEmail, dataJSON)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
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
