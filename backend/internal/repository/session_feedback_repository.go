package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SessionFeedback records a user's explicit good/bad rating for an AI generation session.
type SessionFeedback struct {
	ID        uuid.UUID `db:"id"`
	SessionID uuid.UUID `db:"session_id"`
	UserID    uuid.UUID `db:"user_id"`
	Rating    string    `db:"rating"` // "good" or "bad"
	Reason    string    `db:"reason"`
	CreatedAt time.Time `db:"created_at"`
}

// SessionFeedbackRepository persists user feedback for AI generation sessions.
type SessionFeedbackRepository struct {
	db *pgxpool.Pool
}

func NewSessionFeedbackRepository(db *pgxpool.Pool) *SessionFeedbackRepository {
	return &SessionFeedbackRepository{db: db}
}

// Upsert inserts or updates a user's feedback for a session (one per session per user).
func (r *SessionFeedbackRepository) Upsert(ctx context.Context, sessionID, userID uuid.UUID, rating, reason string) error {
	if rating != "good" && rating != "bad" {
		return fmt.Errorf("invalid rating %q: must be good or bad", rating)
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO ai_session_feedback (session_id, user_id, rating, reason)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (session_id, user_id) DO UPDATE SET rating = $3, reason = $4, created_at = NOW()`,
		sessionID, userID, rating, reason)
	if err != nil {
		return fmt.Errorf("upsert session feedback: %w", err)
	}
	return nil
}

// GetBySession returns the user's feedback for a session, or nil if not found.
func (r *SessionFeedbackRepository) GetBySession(ctx context.Context, sessionID, userID uuid.UUID) (*SessionFeedback, error) {
	var f SessionFeedback
	err := r.db.QueryRow(ctx,
		`SELECT id, session_id, user_id, rating, reason, created_at
		 FROM ai_session_feedback WHERE session_id = $1 AND user_id = $2`,
		sessionID, userID).Scan(&f.ID, &f.SessionID, &f.UserID, &f.Rating, &f.Reason, &f.CreatedAt)
	if err != nil {
		return nil, nil // not found = no feedback yet
	}
	return &f, nil
}
