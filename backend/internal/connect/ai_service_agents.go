package connect

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	v1 "anttrader/gen/proto"
	"anttrader/internal/service"
)

// ai_service_agents.go — AI Agent 定义 RPC 入口（自 060 起 user-scoped）。
// 模型绑定通过 (provider_id, model_override) 直接指向 system_ai_configs 行，
// 不再依赖 ai_config_profiles。

// agentToProto 将 service 层 AIAgentDefinition 转 proto。
func agentToProto(d *service.AIAgentDefinition) *v1.AIAgentDefinition {
	if d == nil {
		return &v1.AIAgentDefinition{}
	}
	return &v1.AIAgentDefinition{
		Id:            d.ID.String(),
		AgentKey:      d.AgentKey,
		Type:          d.Type,
		Name:          d.Name,
		Identity:      d.Identity,
		InputHint:     d.InputHint,
		Enabled:       d.Enabled,
		Position:      d.Position,
		ProviderId:    d.ProviderID,
		ModelOverride: d.ModelOverride,
	}
}

// agentFromProto 解析 proto 入参。Position 由调用方 fallback。
func agentFromProto(in *v1.AIAgentDefinition) *service.AIAgentDefinition {
	if in == nil {
		return nil
	}
	id, _ := uuid.Parse(strings.TrimSpace(in.Id))
	return &service.AIAgentDefinition{
		ID:            id,
		AgentKey:      strings.TrimSpace(in.AgentKey),
		Type:          strings.TrimSpace(in.Type),
		Name:          in.Name,
		Identity:      in.Identity,
		InputHint:     in.InputHint,
		Enabled:       in.Enabled,
		Position:      in.Position,
		ProviderID:    strings.TrimSpace(in.ProviderId),
		ModelOverride: strings.TrimSpace(in.ModelOverride),
	}
}

// ListAgents 返回当前用户的 Agent 列表。库里为空时由 service 层 seed 默认值。
func (s *AIService) ListAgents(ctx context.Context, req *connect.Request[v1.ListAgentsRequest]) (*connect.Response[v1.ListAgentsResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if s.aiAgentSvc == nil {
		return connect.NewResponse(&v1.ListAgentsResponse{Agents: []*v1.AIAgentDefinition{}}), nil
	}
	locale := pickPreferredLocale(req)
	defs, err := s.aiAgentSvc.ListAgents(ctx, userID, locale)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*v1.AIAgentDefinition, 0, len(defs))
	for _, d := range defs {
		out = append(out, agentToProto(d))
	}
	return connect.NewResponse(&v1.ListAgentsResponse{Agents: out}), nil
}

// SetAgents 整体替换当前用户的 Agent 列表。
func (s *AIService) SetAgents(ctx context.Context, req *connect.Request[v1.SetAgentsRequest]) (*connect.Response[v1.SetAgentsResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if s.aiAgentSvc == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("agent service not configured"))
	}
	in := req.Msg.GetAgents()
	defs := make([]*service.AIAgentDefinition, 0, len(in))
	for _, p := range in {
		if d := agentFromProto(p); d != nil {
			defs = append(defs, d)
		}
	}
	saved, err := s.aiAgentSvc.SetAgents(ctx, userID, defs)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	out := make([]*v1.AIAgentDefinition, 0, len(saved))
	for _, d := range saved {
		out = append(out, agentToProto(d))
	}
	return connect.NewResponse(&v1.SetAgentsResponse{Agents: out}), nil
}
