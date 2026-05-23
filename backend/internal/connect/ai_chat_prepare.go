package connect

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	v1 "anttrader/gen/proto"
	"anttrader/internal/ai"
	"anttrader/internal/repository"
	"anttrader/internal/service"
)

// Tuning: see docs/AI对话体验与可靠性优化.md
const (
	chatHistoryMaxMessages = 40
	chatProviderDeadline   = 3 * time.Minute
)

func trimConversationHistory(msgs []repository.AIMessage, max int) []repository.AIMessage {
	if max <= 0 || len(msgs) <= max {
		return msgs
	}
	return msgs[len(msgs)-max:]
}

// prepareChatInvocation resolves provider and builds ai.Message slice (system + history + user).
// notConfigured is true when the product should return the soft "errors.ai.not_configured" response without calling the LLM.
func (s *AIService) prepareChatInvocation(ctx context.Context, userID uuid.UUID, msg *v1.ChatRequest) (provider ai.AIProvider, aiMessages []ai.Message, convID uuid.UUID, notConfigured bool, err error) {
	cxt := msg.GetContext()
	aiRole := strings.TrimSpace(extractContextField(cxt, "AIRole"))
	if aiRole == "" {
		aiRole = "default"
	}

	if s.aiCfgSvc != nil {
		if aiRole != "default" {
			if p, perr := s.aiCfgSvc.GetProviderByRole(ctx, userID, aiRole); perr == nil {
				provider = p
			}
		}
		if provider == nil {
			cfg, ok, cfgErr := s.aiCfgSvc.GetConfig(ctx, userID)
			if cfgErr != nil {
				return nil, nil, uuid.Nil, false, connect.NewError(connect.CodeInternal, cfgErr)
			}
			if ok && cfg != nil && cfg.Enabled {
				p, perr := s.aiCfgSvc.BuildProvider(cfg)
				if perr != nil {
					return nil, nil, uuid.Nil, false, connect.NewError(connect.CodeInvalidArgument, perr)
				}
				provider = p
			}
		}
	}

	if provider == nil {
		if s.aiManager == nil {
			return nil, nil, uuid.Nil, true, nil
		}
		p, perr := s.aiManager.GetCurrentProvider()
		if perr != nil {
			return nil, nil, uuid.Nil, true, nil
		}
		provider = p
	}

	var memoryText string
	if s.pythonSvc != nil {
		symbol := extractContextField(cxt, "Symbol")
		timeframe := extractContextField(cxt, "Timeframe")
		if symbol != "" && timeframe != "" {
			memCtx, memCancel := context.WithTimeout(ctx, 5*time.Second)
			mems, memErr := s.pythonSvc.QueryMemory(memCtx, symbol, timeframe, "", 3)
			memCancel()
			if memErr == nil && len(mems) > 0 {
				memoryText = service.FormatMemoriesAsPrompt(mems)
			}
		}
	}

	systemPrompt := "你是一个专业的量化交易策略助手。要求：1) 回复要简洁，优先用要点，最多 15 行。2) 如需给出策略代码，必须放在一个 ```python 代码块``` 中，只输出一份完整可运行代码。3) 代码必须能通过本系统的‘验证代码’（语法正确、不要包含无关的外部依赖/网络请求/文件读写）。4) 除非用户要求，不要输出冗长推导。\n\n当你生成 AntTrader Python 策略代码时，必须严格遵守接口：必须定义 def run(context): 且 run 必须且只能接收一个参数，参数名必须是 context；不得使用 run(ctx) 或 run(context, xxx) 等签名。run(context) 返回 dict。"

	if b64 := extractContextField(cxt, "SystemPromptB64"); b64 != "" {
		if decoded, derr := base64.StdEncoding.DecodeString(b64); derr == nil && len(decoded) > 0 {
			systemPrompt = string(decoded)
		}
	}

	if c := strings.TrimSpace(cxt); c != "" {
		if i := strings.Index(c, "Locale:"); i >= 0 {
			line := c[i+len("Locale:"):]
			if j := strings.IndexAny(line, "\n\r"); j >= 0 {
				line = line[:j]
			}
			locale := strings.TrimSpace(line)
			switch locale {
			case "zh-CN", "zh-TW", "en", "ja", "vi":
				var langName string
				switch locale {
				case "zh-CN":
					langName = "简体中文"
				case "zh-TW":
					langName = "繁體中文"
				case "en":
					langName = "English"
				case "ja":
					langName = "日本語"
				case "vi":
					langName = "Tiếng Việt"
				}
				if langName != "" {
					systemPrompt += "\n\nIMPORTANT: Respond in " + langName + " only."
				}
			}
		}
	}

	if msg.GetConversationId() != "" {
		convID, _ = uuid.Parse(msg.GetConversationId())
	}

	if memoryText != "" {
		systemPrompt += "\n\n" + memoryText
	}

	aiMessages = []ai.Message{{Role: "system", Content: systemPrompt}}
	if convID != uuid.Nil && s.convRepo != nil {
		history, _ := s.convRepo.GetMessages(ctx, convID)
		history = trimConversationHistory(history, chatHistoryMaxMessages)
		for _, m := range history {
			if m.Role == "user" || m.Role == "assistant" {
				aiMessages = append(aiMessages, ai.Message{Role: m.Role, Content: m.Content})
			}
		}
	}
	if b64 := extractContextField(cxt, "HistoryB64"); b64 != "" {
		if decoded, derr := base64.StdEncoding.DecodeString(b64); derr == nil && len(decoded) > 0 {
			var hist []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			}
			if jerr := json.Unmarshal(decoded, &hist); jerr == nil {
				for _, m := range hist {
					if (m.Role == "user" || m.Role == "assistant") && m.Content != "" {
						aiMessages = append(aiMessages, ai.Message{Role: m.Role, Content: m.Content})
					}
				}
			}
		}
	}
	aiMessages = append(aiMessages, ai.Message{Role: "user", Content: msg.GetMessage()})
	return provider, aiMessages, convID, false, nil
}

func persistChatTurn(ctx context.Context, s *AIService, convID, userID uuid.UUID, userText, assistantText string) {
	if convID == uuid.Nil || s.convRepo == nil {
		return
	}
	_, _ = s.convRepo.AddMessage(ctx, convID, "user", userText)
	_, _ = s.convRepo.AddMessage(ctx, convID, "assistant", assistantText)
	_ = s.convRepo.Touch(ctx, convID)

	conv, cerr := s.convRepo.GetByID(ctx, convID, userID)
	if cerr == nil && conv != nil && conv.Title == "ai.conversation.defaultTitle" {
		title := userText
		if len([]rune(title)) > 20 {
			title = string([]rune(title)[:20])
		}
		_ = s.convRepo.UpdateTitle(ctx, convID, userID, title)
	}
}

func mapChatProviderError(err error) *connect.Error {
	if err == nil {
		return nil
	}
	errMsg := err.Error()
	lower := strings.ToLower(errMsg)
	isQuota := strings.Contains(lower, "allocationquota.freetieronly") || strings.Contains(lower, "free tier") || strings.Contains(lower, "free-tier") || strings.Contains(lower, "quota")
	if isQuota {
		if strings.Contains(lower, "status 403") || strings.Contains(lower, " 403") {
			return connect.NewError(connect.CodeResourceExhausted, errors.New(errMsg))
		}
	}
	if strings.Contains(lower, "status 429") || strings.Contains(lower, " 429") || strings.Contains(lower, "too many requests") {
		return connect.NewError(connect.CodeResourceExhausted, errors.New(errMsg))
	}
	return connect.NewError(connect.CodeInternal, err)
}
