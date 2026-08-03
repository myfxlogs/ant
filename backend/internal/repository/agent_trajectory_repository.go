package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AgentTrajectoryEvent records a single step in an agent execution loop.
// Used for data collection, replay, quality analysis, and cost tracking.
type AgentTrajectoryEvent struct {
	ID           uuid.UUID  `db:"id"`
	SessionID    uuid.UUID  `db:"session_id"`
	UserID       uuid.UUID  `db:"user_id"`
	EventSeq     int        `db:"event_seq"`
	EventType    string     `db:"event_type"`
	Content      string     `db:"content"`
	Metadata     []byte     `db:"metadata"`
	TokenInput   int        `db:"token_input"`
	TokenOutput  int        `db:"token_output"`
	Cost         string     `db:"cost"`
	DurationMs   int        `db:"duration_ms"`
	CreatedAt    time.Time  `db:"created_at"`
}

// AgentTrajectoryRepository persists agent execution events.
type AgentTrajectoryRepository struct {
	db *pgxpool.Pool
}

func NewAgentTrajectoryRepository(db *pgxpool.Pool) *AgentTrajectoryRepository {
	return &AgentTrajectoryRepository{db: db}
}

// Insert records a single trajectory event.
func (r *AgentTrajectoryRepository) Insert(ctx context.Context, e *AgentTrajectoryEvent) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO agent_trajectory_events
		   (id, session_id, user_id, event_seq, event_type, content, metadata, token_input, token_output, cost, duration_ms)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		e.ID, e.SessionID, e.UserID, e.EventSeq, e.EventType, e.Content, e.Metadata,
		e.TokenInput, e.TokenOutput, e.Cost, e.DurationMs)
	if err != nil {
		return fmt.Errorf("insert trajectory event: %w", err)
	}
	return nil
}

// ListBySession returns all events for a session ordered by sequence.
func (r *AgentTrajectoryRepository) ListBySession(ctx context.Context, sessionID uuid.UUID) ([]*AgentTrajectoryEvent, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, session_id, user_id, event_seq, event_type, content, metadata, token_input, token_output, cost, duration_ms, created_at
		 FROM agent_trajectory_events
		 WHERE session_id = $1
		 ORDER BY event_seq ASC`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list trajectory events: %w", err)
	}
	defer rows.Close()

	var events []*AgentTrajectoryEvent
	for rows.Next() {
		e := &AgentTrajectoryEvent{}
		if err := rows.Scan(&e.ID, &e.SessionID, &e.UserID, &e.EventSeq, &e.EventType,
			&e.Content, &e.Metadata, &e.TokenInput, &e.TokenOutput, &e.Cost, &e.DurationMs, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// NextEventSeq returns the next sequence number for a session.
func (r *AgentTrajectoryRepository) NextEventSeq(ctx context.Context, sessionID uuid.UUID) (int, error) {
	var seq int
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(MAX(event_seq), 0) + 1 FROM agent_trajectory_events WHERE session_id = $1`,
		sessionID).Scan(&seq)
	return seq, err
}

// SessionCost returns the total cost of all events in a session.
func (r *AgentTrajectoryRepository) SessionCost(ctx context.Context, sessionID uuid.UUID) (string, error) {
	var cost string
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(cost::numeric), 0)::text FROM agent_trajectory_events WHERE session_id = $1`,
		sessionID).Scan(&cost)
	return cost, err
}

// UserDailyCost returns total trajectory cost for a user today.
func (r *AgentTrajectoryRepository) UserDailyCost(ctx context.Context, userID uuid.UUID) (string, error) {
	var cost string
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(cost::numeric), 0)::text
		 FROM agent_trajectory_events
		 WHERE user_id = $1 AND created_at >= date_trunc('day', NOW())`,
		userID).Scan(&cost)
	return cost, err
}
