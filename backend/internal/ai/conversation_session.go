// Package ai — conversation_session.go
// Lightweight strategy-scoped session management.
// Thin wrapper over AIConversationRepository — append + read only,
// no sliding window, no summaries.

package ai

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"alphaforge/internal/repository"
)

// ConversationSession manages AI chat sessions bound to strategies.
type ConversationSession struct {
	repo *repository.AIConversationRepository
}

// NewConversationSession creates a ConversationSession backed by the repo.
func NewConversationSession(repo *repository.AIConversationRepository) *ConversationSession {
	return &ConversationSession{repo: repo}
}

// Session represents a resolved strategy session with its message history.
type Session struct {
	ID          uuid.UUID
	StrategyKey string
	Title       string
	Messages    []repository.AIMessage
}

// GetOrCreate finds an existing session for strategyKey, or creates one.
func (s *ConversationSession) GetOrCreate(ctx context.Context, userID uuid.UUID, strategyKey, title string) (*Session, error) {
	conv, err := s.repo.GetByStrategyKey(ctx, userID, strategyKey)
	if err == nil {
		msgs, _ := s.repo.GetMessages(ctx, userID, conv.ID)
	// Cap at 200 messages to prevent unbounded context growth.
	if len(msgs) > 200 {
		msgs = msgs[len(msgs)-200:]
	}
		return &Session{ID: conv.ID, StrategyKey: strategyKey, Title: conv.Title, Messages: msgs}, nil
	}
	// Create new
	if title == "" {
		title = strategyKey
	}
	conv, err = s.repo.CreateWithStrategyKey(ctx, userID, title, strategyKey)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return &Session{ID: conv.ID, StrategyKey: strategyKey, Title: title}, nil
}

// AppendExchange persists a user→assistant message pair atomically.
// Non-fatal on failure — callers should log warning and continue.
func (s *ConversationSession) AppendExchange(ctx context.Context, sessionID, userID uuid.UUID, userMsg, assistantMsg string) error {
	if _, err := s.repo.AddMessage(ctx, userID, sessionID, "user", userMsg, nil); err != nil {
		return err
	}
	if _, err := s.repo.AddMessage(ctx, userID, sessionID, "assistant", assistantMsg, nil); err != nil {
		return err
	}
	return s.repo.Touch(ctx, sessionID, userID)
}

// GetMessages returns all messages for a session, ordered by creation time.
func (s *ConversationSession) GetMessages(ctx context.Context, sessionID, userID uuid.UUID) ([]repository.AIMessage, error) {
	return s.repo.GetMessages(ctx, userID, sessionID)
}

// UpdateStrategyKey migrates a session to a new strategy_key (e.g. draft:* → strategy:<id>).
func (s *ConversationSession) UpdateStrategyKey(ctx context.Context, sessionID, userID uuid.UUID, newKey string) error {
	return s.repo.UpdateStrategyKey(ctx, userID, sessionID, newKey)
}
