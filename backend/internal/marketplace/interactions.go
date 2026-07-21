package marketplace

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// ── Rating ────────────────────────────────────────────────────────────────────

// Rate inserts or updates a user's rating for a strategy and returns the new
// average and count (matching the RateStrategyResponse proto shape).
func (s *Service) Rate(ctx context.Context, userID, strategyID string, rating int32) (float64, int32, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return 0, 0, fmt.Errorf("marketplace: invalid user_id: %w", err)
	}
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return 0, 0, fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}
	_, err = s.pg.Exec(ctx,
		`INSERT INTO marketplace_ratings (strategy_id, user_id, rating)
			 VALUES ($1,$2,$3) ON CONFLICT (strategy_id, user_id) DO UPDATE SET rating=$3`,
		sid, uid, rating)
	if err != nil {
		return 0, 0, fmt.Errorf("marketplace: rate: %w", err)
	}
	var avg float64
	var count int32
	err = s.pg.QueryRow(ctx,
		`SELECT COALESCE(AVG(rating),0), COUNT(*) FROM marketplace_ratings WHERE strategy_id=$1`, sid,
	).Scan(&avg, &count)

	// Notify the strategy publisher of the new rating.
	var title string
	_ = s.pg.QueryRow(ctx, `SELECT COALESCE(title,'') FROM marketplace_strategies WHERE strategy_id=$1`, sid).Scan(&title)
	go s.notifyNewRating(context.Background(), sid, title, rating)

	s.pubCache.clear()
	return avg, count, err
}

// ListRatings returns all ratings for a strategy with average and count.
func (s *Service) ListRatings(ctx context.Context, strategyID string) ([]RatingItem, float64, int32, error) {
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}
	rows, err := s.pg.Query(ctx,
		`SELECT id, user_id, rating, created_at FROM marketplace_ratings WHERE strategy_id=$1 ORDER BY created_at DESC`, sid)
	if err != nil {
		return nil, 0, 0, err
	}
	defer rows.Close()
	var items []RatingItem
	for rows.Next() {
		var r RatingItem
		if err := rows.Scan(&r.ID, &r.UserID, &r.Rating, &r.CreatedAt); err != nil {
			return nil, 0, 0, err
		}
		items = append(items, r)
	}
	var avg float64
	var count int32
	_ = s.pg.QueryRow(ctx,
		`SELECT COALESCE(AVG(rating),0), COUNT(*) FROM marketplace_ratings WHERE strategy_id=$1`, sid,
	).Scan(&avg, &count)
	return items, avg, count, rows.Err()
}

// ── Comment ───────────────────────────────────────────────────────────────────

// Comment adds a comment to a strategy and returns the new comment ID.
func (s *Service) Comment(ctx context.Context, userID, strategyID, content string) (string, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return "", fmt.Errorf("marketplace: invalid user_id: %w", err)
	}
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return "", fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}
	var id uuid.UUID
	err = s.pg.QueryRow(ctx,
		`INSERT INTO marketplace_comments (strategy_id, user_id, content)
			 VALUES ($1,$2,$3) RETURNING id`, sid, uid, content).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("marketplace: comment: %w", err)
	}
	s.pubCache.clear()
	return id.String(), nil
}

// ListComments returns paginated comments for a strategy, with total count.
func (s *Service) ListComments(ctx context.Context, strategyID string, limit, offset int32) ([]CommentItem, int32, error) {
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return nil, 0, fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}
	var total int32
	_ = s.pg.QueryRow(ctx,
		`SELECT COUNT(*) FROM marketplace_comments WHERE strategy_id=$1`, sid).Scan(&total)
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.pg.Query(ctx,
		`SELECT c.id, c.user_id, COALESCE(u.nickname,u.email,''), c.content, c.created_at
			 FROM marketplace_comments c LEFT JOIN users u ON u.id=c.user_id
			 WHERE c.strategy_id=$1 ORDER BY c.created_at ASC LIMIT $2 OFFSET $3`,
		sid, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var items []CommentItem
	for rows.Next() {
		var c CommentItem
		if err := rows.Scan(&c.ID, &c.UserID, &c.UserName, &c.Content, &c.CreatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, c)
	}
	return items, total, rows.Err()
}
