package service

import (
	"context"
	"errors"
	"strings"

	"anttrader/internal/ai"

	"github.com/google/uuid"
)

// ai_code_assist_service.go — stateless helpers around the LLM for two
// template-editor flows that don't deserve a full debate session:
//   1. ReviseCode  — "the user typed a natural-language change request,
//      rewrite the strategy code so it still passes the sandbox validator."
//   2. ExplainCode — "summarise what the strategy code does, in the user's
//      UI language (English fallback)."

// CodeChatMessage is one round of the small in-modal chat history.
type CodeChatMessage struct {
	Role    string `json:"role"`    // "user" | "assistant"
	Content string `json:"content"`
}

// AICodeAssistService wraps the configured LLM provider for code revise /
// explain operations. It does NOT persist anything — the modal in the
// frontend keeps its own ephemeral history.
type AICodeAssistService struct {
	aiCfgSvc *AIConfigService
}

func NewAICodeAssistService(aiCfgSvc *AIConfigService) *AICodeAssistService {
	return &AICodeAssistService{aiCfgSvc: aiCfgSvc}
}

func (s *AICodeAssistService) provider(ctx context.Context, userID uuid.UUID, role string) (ai.AIProvider, error) {
	if s == nil || s.aiCfgSvc == nil {
		return nil, errors.New("ai config service not available")
	}
	return s.aiCfgSvc.GetProviderByRole(ctx, userID, role)
}

// ReviseCode asks the model to rewrite `code` per `instruction`. Returns the
// raw assistant text plus the extracted python block (may be empty if the
// model misformatted).
func (s *AICodeAssistService) ReviseCode(
	ctx context.Context,
	userID uuid.UUID,
	code, instruction string,
	history []CodeChatMessage,
	locale string,
) (text, python string, err error) {
	if strings.TrimSpace(instruction) == "" {
		return "", "", errors.New("instruction is required")
	}
	prov, err := s.provider(ctx, userID, "deep")
	if err != nil {
		return "", "", err
	}

	// Reuse CodeSystemPromptV2 sandbox rules so the rewritten code passes the
	// same static validator as the debate-v2 output.
	sys := CodeSystemPromptV2(
		"(The user is editing an existing strategy template — there is no fresh debate intent.)",
		[]UpstreamSummary{},
		locale,
	)

	msgs := make([]ai.Message, 0, 4+len(history))
	msgs = append(msgs, ai.Message{Role: "system", Content: sys})
	// Anchor the conversation around the existing code so the model knows what
	// it is editing. We don't put it in the system prompt to keep that prompt
	// reusable across calls.
	msgs = append(msgs, ai.Message{
		Role: "user",
		Content: "Here is the current strategy code I want to revise:\n\n```python\n" +
			strings.TrimSpace(code) + "\n```\n\nKeep its overall intent unless I explicitly ask otherwise. " +
			"Output the FULL rewritten file, not a diff.",
	})
	for _, h := range history {
		role := strings.ToLower(strings.TrimSpace(h.Role))
		if role != "user" && role != "assistant" {
			continue
		}
		msgs = append(msgs, ai.Message{Role: role, Content: h.Content})
	}
	msgs = append(msgs, ai.Message{Role: "user", Content: instruction})

	resp, err := prov.Chat(ctx, msgs)
	if err != nil {
		return "", "", err
	}
	text = strings.TrimSpace(resp.Content)
	python = ExtractPythonBlockV2(text)
	return text, python, nil
}

// ExplainCode returns a plain-language summary of what the strategy does.
// The reply is in the UI locale (with English fallback) and intentionally
// avoids any markdown fences so the frontend can render it as plain text.
func (s *AICodeAssistService) ExplainCode(
	ctx context.Context,
	userID uuid.UUID,
	code, locale string,
) (string, error) {
	if strings.TrimSpace(code) == "" {
		return "", errors.New("code is required")
	}
	prov, err := s.provider(ctx, userID, "quick")
	if err != nil {
		// fall back to deep model if quick role isn't configured
		prov, err = s.provider(ctx, userID, "deep")
		if err != nil {
			return "", err
		}
	}

	sys := strings.Join([]string{
		"You are AntTrader's strategy code reviewer.",
		"Your job is to explain what a single strategy file does so a non-developer trader can confirm intent.",
		"",
		"[Output rules]",
		"- Plain natural language only. NO code blocks, NO ``` fences, NO markdown headings, NO YAML/JSON.",
		"- Short — at most ~200 words. Use short paragraphs or simple bullet prose.",
		"- Cover, in this order: (1) what signal the strategy emits and when; (2) which indicators / parameters it relies on; (3) how it sizes / exits trades; (4) any obvious risk or edge case.",
		"- If the code is too short / clearly broken, say so plainly instead of inventing behaviour.",
		"",
		languageHintV2(locale),
		"- If the UI locale is not English and the code contains comments in another language, still reply in the UI locale.",
		"- If you genuinely cannot reply in the UI locale, fall back to English rather than refusing.",
	}, "\n")

	usr := "Explain this strategy code:\n\n```python\n" + strings.TrimSpace(code) + "\n```"
	resp, err := prov.Chat(ctx, []ai.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: usr},
	})
	if err != nil {
		return "", err
	}
	// Defensive: strip any rogue ``` blocks the model might still emit.
	return StripCodeBlocksV2(strings.TrimSpace(resp.Content)), nil
}
