package ai

import (
	"context"

	"github.com/google/uuid"

	systemai "alphaforge/internal/service/systemai"
)

// loadHistory loads recent conversation messages as context for the AgentLoop.
func (s *StrategyPlanServer) loadHistory(ctx context.Context, userID uuid.UUID, convID string, limit int) []systemai.ChatMessage {
	if convID == "" {
		return nil
	}
	cid, err := uuid.Parse(convID)
	if err != nil {
		return nil
	}
	msgs, err := s.convRepo.GetMessages(ctx, userID, cid)
	if err != nil || len(msgs) == 0 {
		return nil
	}
	if limit > 0 && len(msgs) > limit {
		msgs = msgs[len(msgs)-limit:]
	}
	out := make([]systemai.ChatMessage, len(msgs))
	for i, m := range msgs {
		role := m.Role
		if role == "assistant" {
			role = "assistant"
		}
		out[i] = systemai.ChatMessage{Role: role, Content: m.Content}
	}
	return out
}

func (s *StrategyPlanServer) persistPlan(ctx context.Context, userID uuid.UUID, convID, userMsg, plan string) {
	if convID == "" || userMsg == "" || plan == "" {
		return
	}
	cid, err := uuid.Parse(convID)
	if err != nil {
		return
	}
	_, _ = s.convRepo.AddMessage(ctx, userID, cid, "user", userMsg, nil)
	_, _ = s.convRepo.AddMessage(ctx, userID, cid, "assistant", "[PLAN]\n"+plan, nil)
	_ = s.convRepo.Touch(ctx, cid, userID)
}

func (s *StrategyPlanServer) persistExchange(ctx context.Context, userID uuid.UUID, convID, plan, code, feedback string) {
	if convID == "" {
		return
	}
	cid, err := uuid.Parse(convID)
	if err != nil {
		return
	}
	if feedback != "" {
		_, _ = s.convRepo.AddMessage(ctx, userID, cid, "user", feedback, nil)
	} else {
		_, _ = s.convRepo.AddMessage(ctx, userID, cid, "user", plan, nil)
	}
	tag := "[DISCUSSION]"
	if code != "" {
		tag = "[CODE]"
	} else if plan != "" {
		tag = "[PLAN]"
	}
	content := code
	if content == "" {
		content = plan
	}
	_, _ = s.convRepo.AddMessage(ctx, userID, cid, "assistant", tag+"\n"+content, nil)
	_ = s.convRepo.Touch(ctx, cid, userID)
}

func (s *StrategyPlanServer) persistDiagnose(ctx context.Context, userID uuid.UUID, convID, question, answer string) {
	if convID == "" {
		return
	}
	cid, err := uuid.Parse(convID)
	if err != nil {
		return
	}
	_, _ = s.convRepo.AddMessage(ctx, userID, cid, "user", question, nil)
	_, _ = s.convRepo.AddMessage(ctx, userID, cid, "assistant", "[DIAGNOSIS]\n"+answer, nil)
	_ = s.convRepo.Touch(ctx, cid, userID)
}
