package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
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
}

type AIConversationRepository struct {
	db *sqlx.DB
}

func NewAIConversationRepository(db *sqlx.DB) *AIConversationRepository {
	return &AIConversationRepository{db: db}
}

func (r *AIConversationRepository) Create(ctx context.Context, userID uuid.UUID, title string) (*AIConversation, error) {
	conv := &AIConversation{
		ID:        uuid.New(),
		UserID:    userID,
		Title:     title,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO ai_conversations (id, user_id, title, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		conv.ID, conv.UserID, conv.Title, conv.CreatedAt, conv.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return conv, nil
}

func (r *AIConversationRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]AIConversation, error) {
	var convs []AIConversation
	err := r.db.SelectContext(ctx, &convs,
		`SELECT c.id, c.user_id, c.title, c.created_at, c.updated_at,
		        COALESCE(m.cnt, 0) AS message_count
		 FROM ai_conversations c
		 LEFT JOIN (SELECT conversation_id, COUNT(*) AS cnt FROM ai_messages GROUP BY conversation_id) m
		   ON m.conversation_id = c.id
		 WHERE c.user_id = $1
		 ORDER BY c.updated_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	return convs, nil
}

func (r *AIConversationRepository) GetByID(ctx context.Context, id, userID uuid.UUID) (*AIConversation, error) {
	var conv AIConversation
	err := r.db.GetContext(ctx, &conv,
		`SELECT id, user_id, title, created_at, updated_at FROM ai_conversations
		 WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *AIConversationRepository) UpdateTitle(ctx context.Context, id, userID uuid.UUID, title string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE ai_conversations SET title = $1, updated_at = $2 WHERE id = $3 AND user_id = $4`,
		title, time.Now(), id, userID,
	)
	return err
}

func (r *AIConversationRepository) Touch(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE ai_conversations SET updated_at = $1 WHERE id = $2`,
		time.Now(), id,
	)
	return err
}

func (r *AIConversationRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM ai_conversations WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	return err
}

func (r *AIConversationRepository) AddMessage(ctx context.Context, conversationID uuid.UUID, role, content string) (*AIMessage, error) {
	msg := &AIMessage{
		ID:             uuid.New(),
		ConversationID: conversationID,
		Role:           role,
		Content:        content,
		CreatedAt:      time.Now(),
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO ai_messages (id, conversation_id, role, content, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		msg.ID, msg.ConversationID, msg.Role, msg.Content, msg.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (r *AIConversationRepository) GetMessages(ctx context.Context, conversationID uuid.UUID) ([]AIMessage, error) {
	var msgs []AIMessage
	err := r.db.SelectContext(ctx, &msgs,
		`SELECT id, conversation_id, role, content, created_at
		 FROM ai_messages WHERE conversation_id = $1 ORDER BY created_at ASC`,
		conversationID,
	)
	if err != nil {
		return nil, err
	}
	return msgs, nil
}
