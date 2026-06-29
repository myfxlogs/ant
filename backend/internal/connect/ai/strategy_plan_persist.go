package ai

import (
	"context"

	"github.com/google/uuid"

	systemai "anttrader/internal/service/systemai"
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
	_, _ = s.convRepo.AddMessage(ctx, userID, cid, "user", userMsg)
	_, _ = s.convRepo.AddMessage(ctx, userID, cid, "assistant", "[PLAN]\n"+plan)
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
		_, _ = s.convRepo.AddMessage(ctx, userID, cid, "user", feedback)
	} else {
		_, _ = s.convRepo.AddMessage(ctx, userID, cid, "user", plan)
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
	_, _ = s.convRepo.AddMessage(ctx, userID, cid, "assistant", tag+"\n"+content)
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
	_, _ = s.convRepo.AddMessage(ctx, userID, cid, "user", question)
	_, _ = s.convRepo.AddMessage(ctx, userID, cid, "assistant", "[DIAGNOSIS]\n"+answer)
	_ = s.convRepo.Touch(ctx, cid, userID)
}
