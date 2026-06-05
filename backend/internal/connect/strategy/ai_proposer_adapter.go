package strategy

import (
	"context"

	"github.com/google/uuid"

	"anttrader/internal/ai"
	systemai "anttrader/internal/service/systemai"
)

var _ ai.AIProposer = (*systemAIAdapter)(nil)

// systemAIAdapter adapts systemai.Service to ai.AIProposer for use in
// the experiment worker. Each experiment gets a fresh adapter scoped to
// its user so API key resolution works correctly.
type systemAIAdapter struct {
	svc    *systemai.Service
	userID uuid.UUID
}

func (a *systemAIAdapter) ChatCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	msgs := []systemai.ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
	return a.svc.ChatCompletion(ctx, a.userID, msgs, "")
}
