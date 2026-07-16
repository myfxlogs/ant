package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

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

// DeleteUserTx is the transaction variant of DeleteUser.
func (r *AdminRepository) DeleteUserTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	result, err := tx.Exec(ctx, `UPDATE users SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
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

// DeleteUsersTx is the transaction variant of DeleteUsers.
func (r *AdminRepository) DeleteUsersTx(ctx context.Context, tx pgx.Tx, ids []uuid.UUID) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	ct, err := tx.Exec(ctx, `UPDATE users SET deleted_at = NOW() WHERE id = ANY($1) AND deleted_at IS NULL`, ids)
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

// RestoreUserTx is the transaction variant of RestoreUser.
func (r *AdminRepository) RestoreUserTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	result, err := tx.Exec(ctx, `UPDATE users SET deleted_at = NULL WHERE id = $1 AND deleted_at IS NOT NULL`, id)
	if err != nil {
		return fmt.Errorf("restore user: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// userFKDependency describes a child table with FK columns referencing users(id).
type userFKDependency struct {
	Table   string
	Columns []string // all FK columns in this table referencing users(id)
}

// discoverUserFKs queries information_schema to find every table with a
// foreign key to users(id). New tables are picked up automatically —
// no hardcoded list to maintain.
func (r *AdminRepository) discoverUserFKs(ctx context.Context) ([]userFKDependency, error) {
	rows, err := r.db.Query(ctx, `
		SELECT tc.table_name,
		       array_agg(DISTINCT kcu.column_name ORDER BY kcu.column_name) AS columns
		FROM information_schema.table_constraints tc
		JOIN information_schema.referential_constraints rc
		  ON tc.constraint_name = rc.constraint_name
		 AND tc.constraint_schema = rc.constraint_schema
		JOIN information_schema.key_column_usage kcu
		  ON tc.constraint_name = kcu.constraint_name
		 AND tc.constraint_schema = kcu.constraint_schema
		JOIN information_schema.constraint_column_usage ccu
		  ON rc.unique_constraint_name = ccu.constraint_name
		 AND rc.unique_constraint_schema = ccu.constraint_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
		  AND ccu.table_name = 'users'
		  AND ccu.column_name = 'id'
		  AND tc.table_schema = 'public'
		GROUP BY tc.table_name
		ORDER BY tc.table_name
	`)
	if err != nil {
		return nil, fmt.Errorf("discover user FKs: %w", err)
	}
	defer rows.Close()

	var deps []userFKDependency
	for rows.Next() {
		var dep userFKDependency
		if err := rows.Scan(&dep.Table, &dep.Columns); err != nil {
			return nil, fmt.Errorf("scan FK dependency: %w", err)
		}
		deps = append(deps, dep)
	}
	return deps, rows.Err()
}

// buildAffectedCountQuery builds a UNION ALL query that counts rows in each
// discovered child table for a given user.
func buildAffectedCountQuery(deps []userFKDependency) string {
	if len(deps) == 0 {
		return `SELECT '', 0 WHERE FALSE`
	}
	parts := make([]string, 0, len(deps))
	for _, dep := range deps {
		label := fmt.Sprintf("'%s'", dep.Table)
		if len(dep.Columns) == 1 {
			parts = append(parts, fmt.Sprintf(
				`SELECT %s, COUNT(*) FROM %s WHERE %s = $1`,
				label, dep.Table, dep.Columns[0],
			))
		} else {
			ors := make([]string, len(dep.Columns))
			for i, col := range dep.Columns {
				ors[i] = fmt.Sprintf("%s = $1", col)
			}
			parts = append(parts, fmt.Sprintf(
				`SELECT %s, COUNT(*) FROM %s WHERE %s`,
				label, dep.Table, strings.Join(ors, " OR "),
			))
		}
	}
	return strings.Join(parts, " UNION ALL ")
}

// GetAffectedTableCounts returns the number of rows in each child table that
// reference the given user ID. FK dependencies are discovered dynamically from
// information_schema — no hardcoded table list.
func (r *AdminRepository) GetAffectedTableCounts(ctx context.Context, userID uuid.UUID) (map[string]int64, error) {
	deps, err := r.discoverUserFKs(ctx)
	if err != nil {
		return nil, err
	}

	query := buildAffectedCountQuery(deps)

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("count affected tables: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int64, len(deps))
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
	ct, err := r.db.Exec(ctx,
		`DELETE FROM users WHERE deleted_at IS NOT NULL AND deleted_at < NOW() - make_interval(days => $1)`,
		retentionDays,
	)
	if err != nil {
		return 0, fmt.Errorf("hard delete expired users: %w", err)
	}
	return ct.RowsAffected(), nil
}

// InsertAuditLog records an admin action in the audit log.
func (r *AdminRepository) InsertAuditLog(ctx context.Context, actorID uuid.UUID, action, targetID, targetEmail string, affectedData map[string]int64) error {
	return insertAuditLog(ctx, r.db, actorID, action, targetID, targetEmail, affectedData)
}

// InsertAuditLogTx is the transaction variant of InsertAuditLog.
func (r *AdminRepository) InsertAuditLogTx(ctx context.Context, tx pgx.Tx, actorID uuid.UUID, action, targetID, targetEmail string, affectedData map[string]int64) error {
	return insertAuditLog(ctx, tx, actorID, action, targetID, targetEmail, affectedData)
}

type execer interface {
	Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
}

func insertAuditLog(ctx context.Context, exec execer, actorID uuid.UUID, action, targetID, targetEmail string, affectedData map[string]int64) error {
	_, err := exec.Exec(ctx,
		`INSERT INTO admin_audit_log (actor_id, action, target_id, target_email, affected_data)
		 VALUES ($1, $2, $3, $4, $5)`,
		actorID, action, targetID, targetEmail, affectedData)
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
