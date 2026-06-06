package ai

import (
	"context"

	"connectrpc.com/connect"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/repository"
)

var defaultAgents = []*antv1.AIAgentDefinition{
	{AgentKey: "strategist", Type: "style", Name: "Strategy Analyst", Identity: "Senior quantitative strategy analyst", Enabled: true, Position: 1},
	{AgentKey: "risk_manager", Type: "risk", Name: "Risk Manager", Identity: "Strict risk control expert", Enabled: true, Position: 2},
	{AgentKey: "executor", Type: "execution", Name: "Execution Advisor", Identity: "Trade execution optimization expert", Enabled: false, Position: 3},
	{AgentKey: "researcher", Type: "macro", Name: "Market Researcher", Identity: "Macroeconomic and industry researcher", Enabled: false, Position: 4},
}

func (s *AIServer) ListAgents(ctx context.Context, req *connect.Request[antv1.ListAgentsRequest]) (*connect.Response[antv1.ListAgentsResponse], error) {
	uid, err := userIDFromCtx(ctx)
	// Start with system defaults.
	agentMap := make(map[string]*antv1.AIAgentDefinition, len(defaultAgents))
	for _, a := range defaultAgents {
		agentMap[a.AgentKey] = a
	}
	// Overlay user-saved definitions from DB.
	if err == nil && s.agentDefRepo != nil {
		rows, listErr := s.agentDefRepo.ListByUser(ctx, uid)
		if listErr == nil {
			for _, row := range rows {
				agentMap[row.AgentKey] = &antv1.AIAgentDefinition{
					AgentKey: row.AgentKey, Type: row.Type,
					Name: row.Name, Identity: row.Identity,
					InputHint: row.InputHint, Enabled: row.Enabled,
					Position: row.Position, ProviderId: row.ProviderID,
					ModelOverride: row.ModelOverride,
				}
			}
		}
	}
	out := make([]*antv1.AIAgentDefinition, 0, len(agentMap))
	for _, a := range agentMap {
		out = append(out, a)
	}
	return connect.NewResponse(&antv1.ListAgentsResponse{Agents: out}), nil
}

// SetAgentDefRepo injects the AI agent definition repository.
func (s *AIServer) SetAgentDefRepo(repo *repository.AIAgentDefinitionRepository) {
	s.agentDefRepo = repo
}

// BatchSetAgents persists agent definitions via ConnectRPC (proto binary).
func (s *AIServer) BatchSetAgents(ctx context.Context, req *connect.Request[antv1.BatchSetAgentsRequest]) (*connect.Response[antv1.BatchSetAgentsResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	entries := req.Msg.GetAgents()
	rows := make([]*repository.AIAgentDefinitionRow, 0, len(entries))
	for _, a := range entries {
		rows = append(rows, &repository.AIAgentDefinitionRow{
			UserID:        uid,
			AgentKey:      a.GetAgentKey(),
			Type:          a.GetType(),
			Name:          a.GetName(),
			Identity:      a.GetIdentity(),
			InputHint:     a.GetInputHint(),
			Enabled:       a.GetEnabled(),
			Position:      a.GetPosition(),
			ProviderID:    a.GetProviderId(),
			ModelOverride: a.GetModelOverride(),
		})
	}
	if err := s.agentDefRepo.ReplaceByUser(ctx, uid, rows); err != nil {
		s.log.Error("BatchSetAgents: replace failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.BatchSetAgentsResponse{Success: true}), nil
}

// ListAgentDefs returns saved agent definitions via ConnectRPC (proto binary).
func (s *AIServer) ListAgentDefs(ctx context.Context, req *connect.Request[antv1.ListAgentDefsRequest]) (*connect.Response[antv1.ListAgentDefsResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.agentDefRepo.ListByUser(ctx, uid)
	if err != nil {
		s.log.Error("ListAgentDefs: list failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	agents := make([]*antv1.AIAgentDefinition, 0, len(rows))
	for _, r := range rows {
		agents = append(agents, &antv1.AIAgentDefinition{
			AgentKey:      r.AgentKey,
			Type:          r.Type,
			Name:          r.Name,
			Identity:      r.Identity,
			InputHint:     r.InputHint,
			Enabled:       r.Enabled,
			Position:      r.Position,
			ProviderId:    r.ProviderID,
			ModelOverride: r.ModelOverride,
		})
	}
	return connect.NewResponse(&antv1.ListAgentDefsResponse{Agents: agents}), nil
}
