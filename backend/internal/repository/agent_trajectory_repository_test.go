package repository

import (
	"testing"

	"github.com/google/uuid"
)

func TestAgentTrajectoryEvent_Struct(t *testing.T) {
	e := &AgentTrajectoryEvent{
		ID:         uuid.New(),
		SessionID:  uuid.New(),
		UserID:     uuid.New(),
		EventSeq:   1,
		EventType:  "reasoning",
		Content:    "Analyzing user request",
		TokenInput: 100,
		TokenOutput: 50,
		Cost:       "0.00200000",
		DurationMs: 250,
	}
	if e.EventType != "reasoning" {
		t.Errorf("EventType = %q, want reasoning", e.EventType)
	}
	if e.EventSeq != 1 {
		t.Errorf("EventSeq = %d, want 1", e.EventSeq)
	}
}

func TestNewAgentTrajectoryRepository(t *testing.T) {
	repo := NewAgentTrajectoryRepository(nil)
	if repo == nil {
		t.Fatal("NewAgentTrajectoryRepository returned nil")
	}
}
