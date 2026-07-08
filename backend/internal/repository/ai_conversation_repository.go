package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AIConversation struct {
	ID           uuid.UUID `db:"id"`
	UserID       uuid.UUID `db:"user_id"`
	Title        string    `db:"title"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
	MessageCount int       `db:"message_count"`
}

type AIMessage struct {
	ID             uuid.UUID `db:"id"`
	ConversationID uuid.UUID `db:"conversation_id"`
	Role           string    `db:"role"`
	Content        string    `db:"content"`
	CreatedAt      time.Time `db:"created_at"`
	TurnData       []byte    `db:"turn_data"`
}

type AIConversationRepository struct {
	db *pgxpool.Pool
}

func NewAIConversationRepository(db *pgxpool.Pool) *AIConversationRepository {
	return &AIConversationRepository{db: db}
}

func (r *AIConversationRepository) Create(ctx context.Context, userID uuid.UUID, title string) (*AIConversation, error) {
	return r.CreateWithID(ctx, userID, uuid.New(), title)
}

// CreateWithID creates a conversation with an explicit ID (caller-provided UUID).
// Use this when the frontend generates the conversation ID and the backend must
// persist it as-is so subsequent requests with the same ID can load history.
func (r *AIConversationRepository) CreateWithID(ctx context.Context, userID, convID uuid.UUID, title string) (*AIConversation, error) {
	conv := &AIConversation{
		ID:        convID,
		UserID:    userID,
		Title:     title,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO ai_conversations (id, user_id, title, created_at, updated_at)
		 VALUES ($1, $2, $3, NOW(), NOW())`,
		conv.ID, conv.UserID, conv.Title,
	)
	if err != nil {
		return nil, err
	}
	return conv, nil
}

func (r *AIConversationRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]AIConversation, error) {
	// Capped at 100 rows — sufficient for chat history display.
	rows, err := r.db.Query(ctx,
		`SELECT c.id, c.user_id, c.title, c.created_at, c.updated_at,
		        COALESCE(m.cnt, 0) AS message_count
		 FROM ai_conversations c
		 LEFT JOIN (SELECT conversation_id, COUNT(*) AS cnt FROM ai_messages GROUP BY conversation_id) m
		   ON m.conversation_id = c.id
		 WHERE c.user_id = $1
		 ORDER BY c.updated_at DESC
		 LIMIT 100`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var convs []AIConversation
	for rows.Next() {
		var c AIConversation
		if err := rows.Scan(&c.ID, &c.UserID, &c.Title, &c.CreatedAt, &c.UpdatedAt, &c.MessageCount); err != nil {
			return nil, err
		}
		convs = append(convs, c)
	}
	return convs, rows.Err()
}

func (r *AIConversationRepository) GetByID(ctx context.Context, id, userID uuid.UUID) (*AIConversation, error) {
	var conv AIConversation
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, title, created_at, updated_at FROM ai_conversations
		 WHERE id = $1 AND user_id = $2`,
		id, userID,
	).Scan(&conv.ID, &conv.UserID, &conv.Title, &conv.CreatedAt, &conv.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *AIConversationRepository) UpdateTitle(ctx context.Context, id, userID uuid.UUID, title string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE ai_conversations SET title = $1, updated_at = $2 WHERE id = $3 AND user_id = $4`,
		title, time.Now(), id, userID,
	)
	if err != nil {
		return fmt.Errorf("update conversation title: %w", err)
	}
	return nil
}

func (r *AIConversationRepository) Touch(ctx context.Context, id, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE ai_conversations SET updated_at = $1 WHERE id = $2 AND user_id = $3`,
		time.Now(), id, userID,
	)
	if err != nil {
		return fmt.Errorf("touch conversation: %w", err)
	}
	return nil
}

func (r *AIConversationRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM ai_conversations WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	if err != nil {
		return fmt.Errorf("delete conversation: %w", err)
	}
	return nil
}

func (r *AIConversationRepository) AddMessage(ctx context.Context, userID, conversationID uuid.UUID, role, content string, turnData []byte) (*AIMessage, error) {
	msg := &AIMessage{
		ID:             uuid.New(),
		ConversationID: conversationID,
		Role:           role,
		Content:        content,
		CreatedAt:      time.Now(),
	}
	ct, err := r.db.Exec(ctx,
		`INSERT INTO ai_messages (id, conversation_id, role, content, created_at, turn_data)
		 SELECT $1, $2, $3, $4, $5, $6
		 FROM ai_conversations
		 WHERE id = $2 AND user_id = $7`,
		msg.ID, msg.ConversationID, msg.Role, msg.Content, msg.CreatedAt, turnData, userID,
	)
	if err != nil {
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, fmt.Errorf("conversation not found or not owned")
	}
	return msg, nil
}

func (r *AIConversationRepository) GetMessages(ctx context.Context, userID, conversationID uuid.UUID) ([]AIMessage, error) {
	// Single JOIN query atomically verifies ownership AND fetches messages.
	// Eliminates the TOCTOU window between the old two-step approach.
	rows, err := r.db.Query(ctx,
		`SELECT m.id, m.conversation_id, m.role, m.content, m.created_at, COALESCE(m.turn_data, NULL) AS turn_data
		 FROM ai_messages m
		 JOIN ai_conversations c ON c.id = m.conversation_id
		 WHERE m.conversation_id = $1 AND c.user_id = $2
		 ORDER BY m.created_at ASC`,
		conversationID, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []AIMessage
	for rows.Next() {
		var m AIMessage
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.CreatedAt, &m.TurnData); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// DeleteMessagesByConversation removes all messages for a conversation.
// Ownership is verified atomically via the EXISTS subquery.
func (r *AIConversationRepository) DeleteMessagesByConversation(ctx context.Context, userID, conversationID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM ai_messages
		 WHERE conversation_id = $1
		   AND EXISTS (SELECT 1 FROM ai_conversations WHERE id = $1 AND user_id = $2)`,
		conversationID, userID,
	)
	return err
}

// GetByStrategyKey finds a conversation by (user_id, strategy_key).
// Returns pgx.ErrNoRows if not found.
func (r *AIConversationRepository) GetByStrategyKey(ctx context.Context, userID uuid.UUID, strategyKey string) (*AIConversation, error) {
	var conv AIConversation
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, title, created_at, updated_at FROM ai_conversations
		 WHERE user_id = $1 AND strategy_key = $2`,
		userID, strategyKey,
	).Scan(&conv.ID, &conv.UserID, &conv.Title, &conv.CreatedAt, &conv.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

// CreateWithStrategyKey creates a conversation with a strategy_key binding.
func (r *AIConversationRepository) CreateWithStrategyKey(ctx context.Context, userID uuid.UUID, title, strategyKey string) (*AIConversation, error) {
	conv := &AIConversation{
		ID:        uuid.New(),
		UserID:    userID,
		Title:     title,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO ai_conversations (id, user_id, title, strategy_key, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, NOW(), NOW())`,
		conv.ID, conv.UserID, conv.Title, strategyKey,
	)
	if err != nil {
		return nil, err
	}
	return conv, nil
}

// UpdateStrategyKey migrates a conversation's strategy_key (e.g. draft:* → strategy:<id>).
func (r *AIConversationRepository) UpdateStrategyKey(ctx context.Context, userID, sessionID uuid.UUID, newKey string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE ai_conversations SET strategy_key = $1, updated_at = NOW()
		 WHERE id = $2 AND user_id = $3`,
		newKey, sessionID, userID,
	)
	return err
}
