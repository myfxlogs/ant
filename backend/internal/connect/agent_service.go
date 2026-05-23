package connect

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/repository"
)

type AgentService struct {
	repo *repository.AgentRepository
}

func NewAgentService(repo *repository.AgentRepository) *AgentService {
	return &AgentService{repo: repo}
}

func (s *AgentService) IssueAgentToken(ctx context.Context, req *connect.Request[v1.IssueAgentTokenRequest]) (*connect.Response[v1.IssueAgentTokenResponse], error) {
	if s == nil || s.repo == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("agent service not available"))
	}
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Msg.GetName())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name required"))
	}
	plain, prefix, hash, err := repository.NewAgentPlaintextToken()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	var expiresAtPtr = req.Msg.GetExpiresAt()
	var expiresAt = timeFromProto(expiresAtPtr)
	token := &repository.AgentToken{
		UserID:             userID,
		Name:               name,
		TokenPrefix:        prefix,
		TokenHash:          hash,
		Scopes:             repository.NormalizeAgentScopes(req.Msg.GetScopes()),
		AccountAllowlist:   req.Msg.GetAccountAllowlist(),
		SymbolAllowlist:    req.Msg.GetSymbolAllowlist(),
		PaperOnly:          req.Msg.GetPaperOnly(),
		RateLimitPerMinute: int(req.Msg.GetRateLimitPerMin()),
		ExpiresAt:          expiresAt,
		Status:             "active",
	}
	if len(token.Scopes) == 0 {
		token.Scopes = []string{"R"}
	}
	if err := s.repo.CreateToken(ctx, token); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.IssueAgentTokenResponse{Token: agentTokenToProto(token), PlaintextToken: plain}), nil
}

func (s *AgentService) ListAgentTokens(ctx context.Context, req *connect.Request[v1.ListAgentTokensRequest]) (*connect.Response[v1.ListAgentTokensResponse], error) {
	if s == nil || s.repo == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("agent service not available"))
	}
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	tokens, err := s.repo.ListTokens(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*v1.AgentToken, 0, len(tokens))
	for i := range tokens {
		out = append(out, agentTokenToProto(&tokens[i]))
	}
	return connect.NewResponse(&v1.ListAgentTokensResponse{Tokens: out}), nil
}

func (s *AgentService) RevokeAgentToken(ctx context.Context, req *connect.Request[v1.RevokeAgentTokenRequest]) (*connect.Response[v1.AgentToken], error) {
	if s == nil || s.repo == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("agent service not available"))
	}
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	tokenID, err := uuid.Parse(strings.TrimSpace(req.Msg.GetTokenId()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	token, err := s.repo.RevokeToken(ctx, userID, tokenID)
	if err != nil {
		if errors.Is(err, repository.ErrAgentTokenNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(agentTokenToProto(token)), nil
}

func (s *AgentService) ListAgentAudit(ctx context.Context, req *connect.Request[v1.ListAgentAuditRequest]) (*connect.Response[v1.ListAgentAuditResponse], error) {
	if s == nil || s.repo == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("agent service not available"))
	}
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	var tokenID *uuid.UUID
	if strings.TrimSpace(req.Msg.GetTokenId()) != "" {
		id, parseErr := uuid.Parse(strings.TrimSpace(req.Msg.GetTokenId()))
		if parseErr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, parseErr)
		}
		tokenID = &id
	}
	rows, err := s.repo.ListAudit(ctx, userID, tokenID, int(req.Msg.GetLimit()), int(req.Msg.GetOffset()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*v1.AgentAuditEntry, 0, len(rows))
	for i := range rows {
		out = append(out, agentAuditToProto(&rows[i]))
	}
	return connect.NewResponse(&v1.ListAgentAuditResponse{Entries: out}), nil
}

func (s *AgentService) GetAgentCapabilities(ctx context.Context, req *connect.Request[v1.GetAgentCapabilitiesRequest]) (*connect.Response[v1.AgentCapabilities], error) {
	if err := requireAuth(ctx); err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.AgentCapabilities{Scopes: []string{"R", "W", "B", "AI", "T_PAPER"}, LiveTradingEnabled: false, AvailableTools: []string{"whoami", "list_accounts", "list_symbols", "get_quotes", "list_templates", "get_template", "get_schedule_health"}}), nil
}

func agentTokenToProto(token *repository.AgentToken) *v1.AgentToken {
	if token == nil {
		return nil
	}
	out := &v1.AgentToken{Id: token.ID.String(), UserId: token.UserID.String(), Name: token.Name, TokenPrefix: token.TokenPrefix, Scopes: token.Scopes, AccountAllowlist: token.AccountAllowlist, SymbolAllowlist: token.SymbolAllowlist, PaperOnly: token.PaperOnly, RateLimitPerMin: int32(token.RateLimitPerMinute), Status: token.Status, CreatedAt: timestamppb.New(token.CreatedAt.UTC()), UpdatedAt: timestamppb.New(token.UpdatedAt.UTC())}
	if token.ExpiresAt != nil {
		out.ExpiresAt = timestamppb.New(token.ExpiresAt.UTC())
	}
	if token.LastUsedAt != nil {
		out.LastUsedAt = timestamppb.New(token.LastUsedAt.UTC())
	}
	return out
}

func agentAuditToProto(row *repository.AgentAuditLog) *v1.AgentAuditEntry {
	if row == nil {
		return nil
	}
	return &v1.AgentAuditEntry{Id: row.ID.String(), UserId: row.UserID.String(), AgentTokenId: row.AgentTokenID.String(), AgentName: row.AgentName, RpcService: row.RPCService, RpcMethod: row.RPCMethod, Scope: row.Scope, StatusCode: row.StatusCode, IdempotencyKey: row.IdempotencyKey, RiskDecision: row.RiskDecision, RequestSummary: row.RequestSummary, ResponseSummary: row.ResponseSummary, DurationMs: row.DurationMS, CreatedAt: timestamppb.New(row.CreatedAt.UTC())}
}
