package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"anttrader/internal/model"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// DebateSession represents a persisted multi-agent debate conversation.
type DebateSession struct {
	ID                     uuid.UUID      `db:"id" json:"id"`
	UserID                 uuid.UUID      `db:"user_id" json:"userId"`
	Title                  string         `db:"title" json:"title"`
	Status                 string         `db:"status" json:"status"`
	Agents                 model.StringArray `db:"agents" json:"agents"`
	CurrentIntentTurnID    uuid.NullUUID  `db:"current_intent_turn_id" json:"currentIntentTurnId,omitempty"`
	CurrentConsensusTurnID uuid.NullUUID  `db:"current_consensus_turn_id" json:"currentConsensusTurnId,omitempty"`
	CurrentCodeTurnID      uuid.NullUUID  `db:"current_code_turn_id" json:"currentCodeTurnId,omitempty"`
	TemplateID             sql.NullString `db:"template_id" json:"templateId,omitempty"`
	ParamSchema            model.JSONB    `db:"param_schema" json:"paramSchema,omitempty"`
	CreatedAt              time.Time      `db:"created_at" json:"createdAt"`
	UpdatedAt              time.Time      `db:"updated_at" json:"updatedAt"`
}

// DebateTurn represents a single message / artifact within a debate session.
//
// Note: we keep ContentJSON as a raw []byte so that database/sql can scan
// NULL values into it without errors (json.RawMessage as a value type does not
// implement sql.Scanner and cannot accept NULL directly). The service layer
// converts this into json.RawMessage for API responses.
type DebateTurn struct {
	ID           uuid.UUID     `db:"id" json:"id"`
	SessionID    uuid.UUID     `db:"session_id" json:"sessionId"`
	ParentTurnID uuid.NullUUID `db:"parent_turn_id" json:"parentTurnId,omitempty"`
	Type         string        `db:"type" json:"type"`
	Role         string        `db:"role" json:"role"`
	Status       string        `db:"status" json:"status"`
	ContentText  string        `db:"content_text" json:"contentText"`
	ContentJSON  []byte        `db:"content_json" json:"-"`
	CreatedAt    time.Time     `db:"created_at" json:"createdAt"`
}

type DebateRepository struct {
	db *sqlx.DB
}

func NewDebateRepository(db *sqlx.DB) *DebateRepository {
	return &DebateRepository{db: db}
}

// CreateSession inserts a new session owned by the given user.
func (r *DebateRepository) CreateSession(ctx context.Context, userID uuid.UUID, title string, agents []string) (*DebateSession, error) {
	if title == "" {
		title = "Debate"
	}
	s := &DebateSession{
		ID:        uuid.New(),
		UserID:    userID,
		Title:     title,
		Status:    "idle",
		Agents:    model.StringArray(agents),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO debate_sessions (id, user_id, title, status, agents, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		s.ID, s.UserID, s.Title, s.Status, s.Agents, s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// ListSessions returns sessions belonging to the user ordered by updated_at desc.
func (r *DebateRepository) ListSessions(ctx context.Context, userID uuid.UUID, limit int) ([]DebateSession, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var rows []DebateSession
	err := r.db.SelectContext(ctx, &rows,
		`SELECT id, user_id, title, status, agents,
		        current_intent_turn_id, current_consensus_turn_id, current_code_turn_id,
		        template_id, param_schema, created_at, updated_at
		 FROM debate_sessions
		 WHERE user_id = $1 AND status <> 'archived'
		 ORDER BY updated_at DESC
		 LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// GetSession returns a single session scoped to the owner.
func (r *DebateRepository) GetSession(ctx context.Context, id, userID uuid.UUID) (*DebateSession, error) {
	var s DebateSession
	err := r.db.GetContext(ctx, &s,
		`SELECT id, user_id, title, status, agents,
		        current_intent_turn_id, current_consensus_turn_id, current_code_turn_id,
		        template_id, param_schema, created_at, updated_at
		 FROM debate_sessions
		 WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

// SessionPatch collects optional mutable fields.
type SessionPatch struct {
	Title                  *string
	Status                 *string
	Agents                 []string
	AgentsSet              bool
	CurrentIntentTurnID    *uuid.UUID
	CurrentConsensusTurnID *uuid.UUID
	CurrentCodeTurnID      *uuid.UUID
	TemplateID             *string
	ParamSchema            *model.JSONB
}

// UpdateSession applies a patch to a session. Only non-nil fields are written.
func (r *DebateRepository) UpdateSession(ctx context.Context, id, userID uuid.UUID, patch *SessionPatch) error {
	if patch == nil {
		return nil
	}
	sets := make([]string, 0, 8)
	args := make([]interface{}, 0, 8)
	idx := 1
	if patch.Title != nil {
		sets = append(sets, "title = $"+itoa(idx))
		args = append(args, *patch.Title)
		idx++
	}
	if patch.Status != nil {
		sets = append(sets, "status = $"+itoa(idx))
		args = append(args, *patch.Status)
		idx++
	}
	if patch.AgentsSet {
		sets = append(sets, "agents = $"+itoa(idx))
		args = append(args, model.StringArray(patch.Agents))
		idx++
	}
	if patch.CurrentIntentTurnID != nil {
		sets = append(sets, "current_intent_turn_id = $"+itoa(idx))
		args = append(args, *patch.CurrentIntentTurnID)
		idx++
	}
	if patch.CurrentConsensusTurnID != nil {
		sets = append(sets, "current_consensus_turn_id = $"+itoa(idx))
		args = append(args, *patch.CurrentConsensusTurnID)
		idx++
	}
	if patch.CurrentCodeTurnID != nil {
		sets = append(sets, "current_code_turn_id = $"+itoa(idx))
		args = append(args, *patch.CurrentCodeTurnID)
		idx++
	}
	if patch.TemplateID != nil {
		sets = append(sets, "template_id = $"+itoa(idx))
		args = append(args, *patch.TemplateID)
		idx++
	}
	if patch.ParamSchema != nil {
		sets = append(sets, "param_schema = $"+itoa(idx))
		args = append(args, *patch.ParamSchema)
		idx++
	}
	if len(sets) == 0 {
		return nil
	}
	sets = append(sets, "updated_at = $"+itoa(idx))
	args = append(args, time.Now())
	idx++
	args = append(args, id, userID)
	query := "UPDATE debate_sessions SET " + strings.Join(sets, ", ") +
		" WHERE id = $" + itoa(idx) + " AND user_id = $" + itoa(idx+1)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// DeleteSession removes a session (and its turns via ON DELETE CASCADE).
func (r *DebateRepository) DeleteSession(ctx context.Context, id, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM debate_sessions WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	return err
}

// AddTurn inserts a turn and bumps session updated_at.
func (r *DebateRepository) AddTurn(ctx context.Context, t *DebateTurn) (*DebateTurn, error) {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}
	if t.Status == "" {
		t.Status = "approved"
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO debate_turns (id, session_id, parent_turn_id, type, role, status, content_text, content_json, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		t.ID, t.SessionID, t.ParentTurnID, t.Type, t.Role, t.Status, t.ContentText, nullableJSON(t.ContentJSON), t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	_, _ = r.db.ExecContext(ctx,
		`UPDATE debate_sessions SET updated_at = $1 WHERE id = $2`,
		time.Now(), t.SessionID,
	)
	return t, nil
}

// ListTurns returns all turns of a session in chronological order.
func (r *DebateRepository) ListTurns(ctx context.Context, sessionID uuid.UUID) ([]DebateTurn, error) {
	var rows []DebateTurn
	err := r.db.SelectContext(ctx, &rows,
		`SELECT id, session_id, parent_turn_id, type, role, status, content_text, content_json, created_at
		 FROM debate_turns
		 WHERE session_id = $1
		 ORDER BY created_at ASC, id ASC`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// UpdateTurnStatus changes a single turn's status (approve/reject/supersede).
func (r *DebateRepository) UpdateTurnStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE debate_turns SET status = $1 WHERE id = $2`,
		status, id,
	)
	return err
}

// -- helpers --

func nullableJSON(raw []byte) interface{} {
	if len(raw) == 0 {
		return nil
	}
	return raw
}
