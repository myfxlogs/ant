package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NotificationRow mirrors the notifications table.
type NotificationRow struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Type      string
	Title     string
	Message   string
	DataJSON  string
	IsRead    bool
	CreatedAt time.Time
}

// NotificationRepository manages the notifications table.
type NotificationRepository struct {
	pool *pgxpool.Pool
}

func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{pool: pool}
}

// ListByUser returns paginated notifications with total unread count.
func (r *NotificationRepository) ListByUser(
	ctx context.Context, userID uuid.UUID,
	limit, offset int32, unreadOnly bool,
) ([]NotificationRow, int32, error) {
	if limit <= 0 {
		limit = 50
	}
	where := "WHERE user_id = $1"
	args := []interface{}{userID}
	if unreadOnly {
		where += " AND is_read = false"
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, type, title, message, data_json, is_read, created_at
		 FROM notifications `+where+` ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		args[0], limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []NotificationRow
	for rows.Next() {
		var n NotificationRow
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title,
			&n.Message, &n.DataJSON, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, n)
	}

	var unread int32
	err = r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false`,
		userID,
	).Scan(&unread)
	if err != nil {
		unread = 0
	}
	return out, unread, nil
}

// MarkRead sets is_read = true for a single notification.
func (r *NotificationRepository) MarkRead(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE notifications SET is_read = true WHERE id = $1`, id)
	return err
}

// MarkAllRead sets is_read = true for all notifications belonging to a user.
func (r *NotificationRepository) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE notifications SET is_read = true WHERE user_id = $1 AND is_read = false`, userID)
	return err
}
