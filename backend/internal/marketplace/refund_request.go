package marketplace

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// RefundRequestRow represents a row in marketplace_refund_requests.
type RefundRequestRow struct {
	ID             string
	UserID         string
	UserName       string
	SubscriptionID string
	StrategyTitle  string
	Amount         string
	Reason         string
	Status         string
	CreatedAt      time.Time
	ReviewedBy     string
	ReviewNote     string
}

// CreateRefundRequest creates a pending refund request for a subscription.
// Validates that the subscription belongs to the user and is within the
// publisher-configured refund window (default 7 days, read from settlement row).
// The entire operation runs in a single transaction with FOR UPDATE on the subscription
// row to prevent races between the active-status check and the insert.
func (s *Service) CreateRefundRequest(ctx context.Context, userID, subscriptionID, reason string) (string, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return "", fmt.Errorf("marketplace: invalid user_id: %w", err)
	}
	sid, err := uuid.Parse(subscriptionID)
	if err != nil {
		return "", fmt.Errorf("marketplace: invalid subscription_id: %w", err)
	}

	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("marketplace: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Lock subscription row for the duration of this operation.
	var active bool
	var createdAt time.Time
	err = tx.QueryRow(ctx,
		`SELECT active, created_at FROM user_subscriptions
		 WHERE id = $1 AND subscriber_user_id = $2 FOR UPDATE`,
		sid, uid,
	).Scan(&active, &createdAt)
	if err != nil {
		return "", fmt.Errorf("marketplace: subscription not found: %w", err)
	}
	if !active {
		return "", fmt.Errorf("marketplace: subscription is not active")
	}

	// Read refund_window_days from the settlement row (publisher-configurable).
	// Falls back to DefaultRefundWindowDays if no settlement exists.
	refundWindowDays := DefaultRefundWindowDays
	var dbRefundWindowDays int
	err = tx.QueryRow(ctx,
		`SELECT COALESCE(refund_window_days, $2) FROM marketplace_settlements
		 WHERE purchase_id = $1 LIMIT 1`,
		sid, DefaultRefundWindowDays,
	).Scan(&dbRefundWindowDays)
	if err == nil && dbRefundWindowDays > 0 {
		refundWindowDays = dbRefundWindowDays
	}

	// Check refund window.
	if time.Since(createdAt) > time.Duration(refundWindowDays)*24*time.Hour {
		return "", fmt.Errorf("marketplace: refund window (%d days) has expired", refundWindowDays)
	}

	// Atomic check-then-insert: prevents concurrent duplicate pending requests.
	var id string
	err = tx.QueryRow(ctx,
		`INSERT INTO marketplace_refund_requests (user_id, subscription_id, reason, status)
		 SELECT $1, $2, $3, 'pending'
		 WHERE NOT EXISTS (
		   SELECT 1 FROM marketplace_refund_requests
		   WHERE subscription_id = $2 AND status = 'pending'
		 )
		 RETURNING id::text`,
		uid, sid, reason,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("marketplace: a pending refund request already exists for this subscription")
		}
		return "", fmt.Errorf("marketplace: create refund request: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("marketplace: commit refund request: %w", err)
	}

	return id, nil
}

// ListRefundRequests lists refund requests with optional status filter (admin).
func (s *Service) ListRefundRequests(ctx context.Context, status string, limit, offset int) ([]RefundRequestRow, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	var conditions []string
	var args []interface{}
	argIdx := 1

	if status != "" {
		conditions = append(conditions, fmt.Sprintf("rr.status = $%d", argIdx))
		args = append(args, status)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	var total int32
	countQuery := "SELECT COUNT(*) FROM marketplace_refund_requests rr" + whereClause
	err := s.pg.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("marketplace: count refund requests: %w", err)
	}

	query := `SELECT rr.id::text, rr.user_id::text,
	        COALESCE(u.email, u.nickname, rr.user_id::text),
	        rr.subscription_id::text,
	        COALESCE(ms.title, ''),
	        COALESCE(ms.price_amount::text, '0'),
	        rr.reason, rr.status, rr.created_at,
	        COALESCE(rr.reviewed_by::text, ''), COALESCE(rr.review_note, '')
		 FROM marketplace_refund_requests rr
		 LEFT JOIN users u ON u.id = rr.user_id
		 LEFT JOIN user_subscriptions us ON us.id = rr.subscription_id
		 LEFT JOIN marketplace_strategies ms ON ms.strategy_id = us.target_strategy_id`
	query += whereClause
	query += fmt.Sprintf(" ORDER BY rr.created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := s.pg.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("marketplace: list refund requests: %w", err)
	}
	defer rows.Close()

	var result []RefundRequestRow
	for rows.Next() {
		var r RefundRequestRow
		var reviewedBy pgtype.Text
		if err := rows.Scan(
			&r.ID, &r.UserID, &r.UserName,
			&r.SubscriptionID, &r.StrategyTitle, &r.Amount,
			&r.Reason, &r.Status, &r.CreatedAt,
			&reviewedBy, &r.ReviewNote,
		); err != nil {
			return nil, 0, fmt.Errorf("marketplace: scan refund request: %w", err)
		}
		if reviewedBy.Valid {
			r.ReviewedBy = reviewedBy.String
		}
		result = append(result, r)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return result, int(total), nil
}

// ProcessRefundRequest approves or rejects a refund request in a single DB transaction.
func (s *Service) ProcessRefundRequest(ctx context.Context, adminID, refundID string, approve bool, reviewNote string) error {
	aid, err := uuid.Parse(adminID)
	if err != nil {
		return fmt.Errorf("marketplace: invalid admin_id: %w", err)
	}
	rid, err := uuid.Parse(refundID)
	if err != nil {
		return fmt.Errorf("marketplace: invalid refund_id: %w", err)
	}

	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return fmt.Errorf("marketplace: process refund begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Get refund request details.
	var userID, subscriptionID, currentStatus string
	err = tx.QueryRow(ctx,
		`SELECT user_id::text, subscription_id::text, status
		 FROM marketplace_refund_requests WHERE id = $1 FOR UPDATE`,
		rid,
	).Scan(&userID, &subscriptionID, &currentStatus)
	if err != nil {
		return fmt.Errorf("marketplace: refund request not found: %w", err)
	}

	if currentStatus != "pending" {
		return fmt.Errorf("marketplace: refund request already processed (status=%s)", currentStatus)
	}

	status := "rejected"
	if approve {
		uid, err := uuid.Parse(userID)
		if err != nil {
			return fmt.Errorf("marketplace: invalid user_id: %w", err)
		}
		sid, err := uuid.Parse(subscriptionID)
		if err != nil {
			return fmt.Errorf("marketplace: invalid subscription_id: %w", err)
		}
		if _, err := s.refundPurchaseTx(ctx, tx, uid, sid); err != nil {
			return fmt.Errorf("marketplace: execute refund: %w", err)
		}
		status = "approved"
	}

	_, err = tx.Exec(ctx,
		`UPDATE marketplace_refund_requests
		 SET status = $1, reviewed_by = $2, review_note = $3, reviewed_at = now()
		 WHERE id = $4`,
		status, aid, reviewNote, rid)
	if err != nil {
		return fmt.Errorf("marketplace: update refund request status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("marketplace: commit process refund: %w", err)
	}
	s.pubCache.clear()
	return nil
}
