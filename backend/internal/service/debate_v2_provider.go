package service

// debate_v2_provider.go
// 自 debate_v2_service.go 拆出：模型 provider 解析 + 历史 / 摘要 helper。
// 这里只放无状态工具与 provider 选择；主流程仍在 debate_v2_service.go。

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"anttrader/internal/ai"
	"anttrader/internal/repository"
)

// getProviderByRole 按角色取 provider；自 060 起 role 实际被忽略，
// 委托给 AIConfigService 选用户当前默认的 SystemAI provider。
func (s *DebateV2Service) getProviderByRole(ctx context.Context, userID uuid.UUID, role string) (ai.AIProvider, error) {
	if s.aiCfgSvc == nil {
		return nil, errors.New("ai config service not available")
	}
	return s.aiCfgSvc.GetProviderByRole(ctx, userID, role)
}

// getProviderForStep 选当前步骤要用的 provider：
//  1. agent: 步骤 → 取该 Agent 自身 (provider_id, model_override)；
//  2. 没绑或回退失败 → 走用户的 Default Primary Model（GetProviderByRole 内部
//     已经先看 users.ai_primary_*，再 fallback 到 system_ai_configs 首行）。
//
// 失败时继续尝试用户其它可用 SystemAI provider，避免单一模型超时卡死。
func (s *DebateV2Service) getProviderForStep(ctx context.Context, userID uuid.UUID, stepKey string) (ai.AIProvider, error) {
	providers, err := s.getProvidersForStep(ctx, userID, stepKey)
	if err != nil {
		return nil, err
	}
	if len(providers) == 0 {
		return nil, errors.New("no ai provider available")
	}
	return providers[0], nil
}

func (s *DebateV2Service) getProvidersForStep(ctx context.Context, userID uuid.UUID, stepKey string) ([]ai.AIProvider, error) {
	out := make([]ai.AIProvider, 0, 4)
	seen := map[string]struct{}{}
	add := func(p ai.AIProvider) {
		if p == nil {
			return
		}
		key := providerLabel(p)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	if strings.HasPrefix(stepKey, v2StepAgentPrefix) && s.aiAgentSvc != nil {
		agentKey := strings.TrimPrefix(stepKey, v2StepAgentPrefix)
		if defs, err := s.aiAgentSvc.ListAgents(ctx, userID, ""); err == nil {
			for _, d := range defs {
				if d == nil {
					continue
				}
				if d.AgentKey == agentKey || d.Type == agentKey {
					if p, ok := s.providerForAgent(ctx, userID, d.ProviderID, d.ModelOverride); ok && p != nil {
						add(p)
					}
					break
				}
			}
		}
	}
	if p, err := s.getProviderByRole(ctx, userID, "deep"); err == nil {
		add(p)
	}
	if len(out) > 0 {
		return out, nil
	}
	for _, p := range s.systemFallbackProviders(ctx, userID) {
		add(p)
	}
	if len(out) == 0 {
		return nil, errors.New("no ai provider available")
	}
	return out, nil
}

// providerForAgent 根据 Agent 表中的 (provider_id, model_override) 直接查
// system_ai_configs 行并构造 ai.AIProvider。provider_id 为空表示「使用用户
// 默认 SystemAI provider」，返回 (nil, false) 由调用方 fallback。
func (s *DebateV2Service) providerForAgent(ctx context.Context, userID uuid.UUID, providerID, modelOverride string) (ai.AIProvider, bool) {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" || s.systemAISvc == nil || s.aiCfgSvc == nil {
		return nil, false
	}
	cfg, err := s.systemAISvc.BuildProviderConfig(ctx, userID, providerID)
	if err != nil || cfg == nil {
		return nil, false
	}
	if m := strings.TrimSpace(modelOverride); m != "" {
		cfg.Model = m
	}
	p, perr := s.aiCfgSvc.BuildProvider(cfg)
	if perr != nil || p == nil {
		return nil, false
	}
	return p, true
}

func (s *DebateV2Service) systemFallbackProviders(ctx context.Context, userID uuid.UUID) []ai.AIProvider {
	if s.systemAISvc == nil || s.aiCfgSvc == nil {
		return nil
	}
	providerIDs := []string{"deepseek", "openai", "anthropic", "qwen", "moonshot", "zhipu", "openai_compatible"}
	out := make([]ai.AIProvider, 0, len(providerIDs))
	for _, id := range providerIDs {
		cfg, err := s.systemAISvc.BuildProviderConfig(ctx, userID, id)
		if err != nil || cfg == nil || !cfg.Enabled || strings.TrimSpace(cfg.APIKey) == "" || strings.TrimSpace(cfg.Model) == "" {
			continue
		}
		p, err := s.aiCfgSvc.BuildProvider(cfg)
		if err == nil && p != nil {
			out = append(out, p)
		}
	}
	return out
}

// providerUsageMeta 从 provider 响应里提取「provider/model + token 用量」，
// 调用方再补 StepKey/Kind/Python 等业务字段。
func providerUsageMeta(p ai.AIProvider, resp *ai.Response) v2TurnMeta {
	m := v2TurnMeta{}
	if p != nil {
		m.Provider = p.GetProviderName()
		m.Model = p.GetModelName()
	}
	if resp != nil {
		m.PromptTokens = resp.Usage.PromptTokens
		m.CompletionTokens = resp.Usage.CompletionTokens
		m.TotalTokens = resp.Usage.TotalTokens
	}
	return m
}

// collectStepHistory 把已 approved 的某一步对话转成 ai.Message 链；
// 助手发言会先剥代码块（防 prompt 注入)。
func collectStepHistory(turns []repository.DebateTurn, stepKey string) []ai.Message {
	out := make([]ai.Message, 0, 8)
	for _, t := range turns {
		if t.Status != "approved" {
			continue
		}
		if t.Type != "v2_user" && t.Type != "v2_assistant" {
			continue
		}
		meta := unmarshalV2Meta(t.ContentJSON)
		if meta.StepKey != stepKey {
			continue
		}
		role := "assistant"
		if t.Type == "v2_user" {
			role = "user"
		}
		content := t.ContentText
		if role == "assistant" {
			content = StripCodeBlocksV2(content)
		}
		if strings.TrimSpace(content) == "" {
			continue
		}
		out = append(out, ai.Message{Role: role, Content: content})
	}
	return out
}

// lastAssistantReply 取某一步最后一条 assistant 回复（排除 kickoff 自述）。
func lastAssistantReply(turns []repository.DebateTurn, stepKey string) string {
	for i := len(turns) - 1; i >= 0; i-- {
		t := turns[i]
		if t.Type != "v2_assistant" || t.Status != "approved" {
			continue
		}
		meta := unmarshalV2Meta(t.ContentJSON)
		if meta.StepKey == stepKey && meta.Kind != "kickoff" {
			return strings.TrimSpace(t.ContentText)
		}
	}
	return ""
}

// collectAgentSummaries 给 code-gen 步骤准备各 agent 的最终结论摘要。
func collectAgentSummaries(turns []repository.DebateTurn, agents []string) []UpstreamSummary {
	out := make([]UpstreamSummary, 0, len(agents))
	for _, a := range agents {
		text := lastAssistantReply(turns, v2StepAgentPrefix+a)
		if strings.TrimSpace(text) == "" {
			continue
		}
		out = append(out, UpstreamSummary{Name: a, Text: text})
	}
	return out
}
