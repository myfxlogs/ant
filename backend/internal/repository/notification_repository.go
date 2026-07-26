package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// NotificationRow mirrors the notifications table.
type NotificationRow struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Type      string
	Title     string
	Message   string
	Data      *structpb.Struct
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
		`SELECT id, user_id, type, title, message, data_proto, is_read, created_at
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
		var rawBytes []byte
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title,
			&n.Message, &rawBytes, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, 0, err
		}
		if len(rawBytes) > 0 {
			var s structpb.Struct
			if proto.Unmarshal(rawBytes, &s) == nil {
				n.Data = &s
			}
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
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

// MarkReadForUser sets is_read = true for a single notification, scoped to the
// authenticated user. Returns an error if the notification does not belong to
// the user (no rows affected → pgx.ErrNoRows via RowsAffected check).
func (r *NotificationRepository) MarkReadForUser(ctx context.Context, id, userID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE notifications SET is_read = true WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("notification not found or not owned by user")
	}
	return nil
}

// MarkAllRead sets is_read = true for all notifications belonging to a user.
func (r *NotificationRepository) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE notifications SET is_read = true WHERE user_id = $1 AND is_read = false`, userID)
	return err
}

// Insert creates a new notification and returns the row.
func (r *NotificationRepository) Insert(
	ctx context.Context, userID uuid.UUID, typ, title, message string, data *structpb.Struct,
) (NotificationRow, error) {
	var dataBytes []byte
	if data != nil {
		dataBytes, _ = proto.Marshal(data)
	}
	var n NotificationRow
	var rawBytes []byte
	err := r.pool.QueryRow(ctx,
		`INSERT INTO notifications (user_id, type, title, message, data_proto)
		 VALUES ($1,$2,$3,$4,$5)
		 RETURNING id, user_id, type, title, message, data_proto, is_read, created_at`,
		userID, typ, title, message, dataBytes,
	).Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Message, &rawBytes, &n.IsRead, &n.CreatedAt)
	if err != nil {
		return n, err
	}
	if len(rawBytes) > 0 {
		var s structpb.Struct
		if proto.Unmarshal(rawBytes, &s) == nil {
			n.Data = &s
		}
	}
	return n, nil
}
