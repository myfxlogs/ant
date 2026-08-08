package ai

import (
	"context"
	"testing"

	systemai "alphaforge/internal/service/systemai"
)

// stubPlanTracker implements PlanTracker for testing.
type stubPlanTracker struct {
	hasActive   bool
	currentStep string
	completed   int
	total       int
}

func (s *stubPlanTracker) HasActivePlan() bool { return s.hasActive }
func (s *stubPlanTracker) CurrentStep() string { return s.currentStep }
func (s *stubPlanTracker) CompletedSteps() int { return s.completed }
func (s *stubPlanTracker) TotalSteps() int     { return s.total }

// Adversarial proof: SetPlanTracker stores the tracker on the AgentLoop.
// Remove the field and the plan-driven guidance won't fire.
func TestAgentLoop_SetPlanTracker(t *testing.T) {
	reg := NewEmptyToolRegistry()
	loop := NewAgentLoop(reg,
		func(_ context.Context, _ []systemai.ChatMessage, _ []systemai.ToolDefinition, _ func(systemai.ChatStreamChunk) error) error {
			return nil
		},
		nil, nil, nil,
	)

	pt := &stubPlanTracker{hasActive: true, currentStep: "test step", completed: 1, total: 3}
	loop.SetPlanTracker(pt)

	if loop.planTracker == nil {
		t.Fatal("SetPlanTracker should store the tracker")
	}

	if !loop.planTracker.HasActivePlan() {
		t.Error("HasActivePlan should return true")
	}
	if loop.planTracker.CurrentStep() != "test step" {
		t.Errorf("CurrentStep should return 'test step', got %q", loop.planTracker.CurrentStep())
	}
	if loop.planTracker.CompletedSteps() != 1 {
		t.Errorf("CompletedSteps should return 1, got %d", loop.planTracker.CompletedSteps())
	}
	if loop.planTracker.TotalSteps() != 3 {
		t.Errorf("TotalSteps should return 3, got %d", loop.planTracker.TotalSteps())
	}
}

// Adversarial proof: nil planTracker means no plan-driven guidance.
func TestAgentLoop_NilPlanTracker(t *testing.T) {
	reg := NewEmptyToolRegistry()
	loop := NewAgentLoop(reg,
		func(_ context.Context, _ []systemai.ChatMessage, _ []systemai.ToolDefinition, _ func(systemai.ChatStreamChunk) error) error {
			return nil
		},
		nil, nil, nil,
	)

	if loop.planTracker != nil {
		t.Fatal("planTracker should be nil by default")
	}
}

// Adversarial proof: PlanTracker interface contract — all methods present.
func TestPlanTracker_Interface(t *testing.T) {
	var _ PlanTracker = &stubPlanTracker{}
}

// Adversarial proof: compressContext preserves system + user messages.
// Remove the keep[0] = messages[0] line and system prompt is lost.
func TestCompressContext_PreservesSystemAndUser(t *testing.T) {
	messages := make([]systemai.ChatMessage, 30)
	messages[0] = systemai.ChatMessage{Role: "system", Content: "system prompt"}
	messages[1] = systemai.ChatMessage{Role: "user", Content: "user prompt"}
	for i := 2; i < 30; i++ {
		messages[i] = systemai.ChatMessage{Role: "assistant", Content: "filler"}
	}

	compressed := compressContext(messages)

	if compressed[0].Role != "system" || compressed[0].Content != "system prompt" {
		t.Errorf("system message not preserved: %+v", compressed[0])
	}
	if compressed[1].Role != "user" || compressed[1].Content != "user prompt" {
		t.Errorf("user message not preserved: %+v", compressed[1])
	}
}

// Adversarial proof: compressContext is a no-op for short conversations.
func TestCompressContext_ShortConversation(t *testing.T) {
	messages := []systemai.ChatMessage{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "usr"},
		{Role: "assistant", Content: "asst"},
	}

	result := compressContext(messages)
	if len(result) != 3 {
		t.Errorf("expected 3 messages (no-op), got %d", len(result))
	}
}
