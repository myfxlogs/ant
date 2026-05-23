package service

// debate_v2_dto.go
// 自 debate_v2_service.go 拆出：DTO 类型定义、Session→DTO 装配、resolveAgent。
// 主流程仍在 debate_v2_service.go；这里只是把「读 + 转视图」相关聚到一起，
// 让主文件保持在 800 行以内（见 rules）。

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/model"
	"anttrader/internal/repository"
)

// --------------------------------------------------------------------------
// DTOs
// --------------------------------------------------------------------------

// V2MessageDTO is a single chat message inside a step.
type V2MessageDTO struct {
	ID      uuid.UUID `json:"id"`
	Role    string    `json:"role"` // "user" | "assistant"
	Content string    `json:"content"`
	Kind    string    `json:"kind,omitempty"` // optional: "kickoff"
}

// V2StepDTO bundles the chat for one step plus a stable label source.
type V2StepDTO struct {
	StepKey   string         `json:"stepKey"` // "intent" | "agent:<key>" | "code"
	AgentKey  string         `json:"agentKey,omitempty"`
	AgentName string         `json:"agentName,omitempty"`
	Messages  []V2MessageDTO `json:"messages"`
}

// V2CodeDTO carries the final code proposal (if any).
type V2CodeDTO struct {
	Text   string `json:"text"`
	Python string `json:"python"`
}

// V2UsageDTO accumulates token usage across all assistant turns of a session.
type V2UsageDTO struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

// V2SessionDTO is returned by every v2 endpoint.
type V2SessionDTO struct {
	ID          uuid.UUID                 `json:"id"`
	Title       string                    `json:"title"`
	Status      string                    `json:"status"`
	CurrentStep string                    `json:"currentStep"`
	Agents      []string                  `json:"agents"`
	Steps       []V2StepDTO               `json:"steps"`
	ParamSchema []model.TemplateParameter `json:"paramSchema,omitempty"`
	Code        *V2CodeDTO                `json:"code,omitempty"`
	// Provider + model that the *current* step would use when the user sends
	// the next message. Lets the UI show "current model: deepseek-chat".
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
	// Cumulative token usage across every assistant turn in the session.
	Usage     V2UsageDTO `json:"usage"`
	CreatedAt string     `json:"createdAt"`
	UpdatedAt string     `json:"updatedAt"`
}

// --------------------------------------------------------------------------
// Session → DTO
// --------------------------------------------------------------------------

func (s *DebateV2Service) fetchDTO(ctx context.Context, userID, sessionID uuid.UUID, locale string) (*V2SessionDTO, error) {
	row, err := s.repo.GetSession(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, errors.New("session not found")
	}
	turns, err := s.repo.ListTurns(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return s.buildDTO(ctx, row, turns, locale)
}

func (s *DebateV2Service) buildDTO(ctx context.Context, row *repository.DebateSession, turns []repository.DebateTurn, locale string) (*V2SessionDTO, error) {
	dto := &V2SessionDTO{
		ID:          row.ID,
		Title:       row.Title,
		Status:      row.Status,
		CurrentStep: stepKeyFromStatus(row.Status),
		Agents:      append([]string{}, []string(row.Agents)...),
		Steps:       []V2StepDTO{},
		CreatedAt:   row.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:   row.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}

	if len(row.ParamSchema) > 0 {
		var params []model.TemplateParameter
		if err := json.Unmarshal(row.ParamSchema, &params); err == nil {
			dto.ParamSchema = params
		}
	}

	stepOrder := []string{v2StepIntent}
	for _, a := range dto.Agents {
		stepOrder = append(stepOrder, v2StepAgentPrefix+a)
	}
	byStep := make(map[string][]V2MessageDTO, len(stepOrder))
	for _, t := range turns {
		if t.Status != "approved" {
			continue
		}
		if t.Type != "v2_user" && t.Type != "v2_assistant" {
			continue
		}
		meta := unmarshalV2Meta(t.ContentJSON)
		if meta.StepKey == "" {
			continue
		}
		role := "assistant"
		if t.Type == "v2_user" {
			role = "user"
		}
		byStep[meta.StepKey] = append(byStep[meta.StepKey], V2MessageDTO{
			ID:      t.ID,
			Role:    role,
			Content: t.ContentText,
			Kind:    meta.Kind,
		})
	}

	for _, key := range stepOrder {
		step := V2StepDTO{StepKey: key, Messages: byStep[key]}
		if strings.HasPrefix(key, v2StepAgentPrefix) {
			agentKey := strings.TrimPrefix(key, v2StepAgentPrefix)
			step.AgentKey = agentKey
			step.AgentName = s.resolveAgent(ctx, row.UserID, agentKey, locale).Name
		}
		dto.Steps = append(dto.Steps, step)
	}

	for _, t := range turns {
		if t.Type != "v2_code" || t.Status != "approved" {
			continue
		}
		meta := unmarshalV2Meta(t.ContentJSON)
		dto.Code = &V2CodeDTO{Text: t.ContentText, Python: meta.Python}
	}

	for _, t := range turns {
		if t.Status != "approved" {
			continue
		}
		if t.Type != "v2_assistant" && t.Type != "v2_code" {
			continue
		}
		meta := unmarshalV2Meta(t.ContentJSON)
		dto.Usage.PromptTokens += meta.PromptTokens
		dto.Usage.CompletionTokens += meta.CompletionTokens
		dto.Usage.TotalTokens += meta.TotalTokens
	}

	if p, err := s.getProviderForStep(ctx, row.UserID, dto.CurrentStep); err == nil && p != nil {
		dto.Provider = p.GetProviderName()
		dto.Model = p.GetModelName()
	}

	return dto, nil
}

// resolveAgent 找出某个 agentKey 对应的展示名 + identity。
// 内置类型一律强制使用 agent_i18n 中的本地化文本，避免 LLM 因用户保存的
// 自定义文案而切换语言。
func (s *DebateV2Service) resolveAgent(ctx context.Context, userID uuid.UUID, key, locale string) AgentPromptInput {
	var found AgentPromptInput
	var resolvedType string
	if s.aiAgentSvc != nil {
		if defs, err := s.aiAgentSvc.ListAgents(ctx, userID, locale); err == nil {
			for _, d := range defs {
				if d == nil {
					continue
				}
				if d.AgentKey == key || d.Type == key {
					name := d.Name
					if name == "" {
						name = d.Type
					}
					found = AgentPromptInput{Name: name, Type: d.Type, Identity: d.Identity}
					resolvedType = d.Type
					break
				}
			}
		}
	}
	if resolvedType == "" {
		resolvedType = key
		found = AgentPromptInput{Name: key, Type: key}
	}
	if entry, ok := localizedAgentFor(resolvedType, locale); ok {
		found.Name = entry.Name
		found.Identity = entry.Identity
		found.Type = resolvedType
	}
	return found
}
