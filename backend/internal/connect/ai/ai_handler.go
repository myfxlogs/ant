package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/ai"
	"anttrader/internal/repository"
	systemai "anttrader/internal/service/systemai"
)

// AIServer implements the ant.v1.AIServiceHandler interface.
type AIServer struct {
	systemSvc     *systemai.Service
	conversations *repository.AIConversationRepository
	session       *ai.ConversationSession
	agentDefRepo  *repository.AIAgentDefinitionRepository
	log           *zap.Logger
}

// NewAIServer creates an AI service ConnectRPC handler.
func NewAIServer(systemSvc *systemai.Service, conversations *repository.AIConversationRepository, session *ai.ConversationSession, log *zap.Logger) *AIServer {
	return &AIServer{systemSvc: systemSvc, conversations: conversations, session: session, log: log}
}

var _ antv1c.AIServiceHandler = (*AIServer)(nil)

// userIDFromBearer extracts the user ID from an Authorization: Bearer header.
func userIDFromBearer(r *http.Request) (uuid.UUID, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
		return uuid.Nil, fmt.Errorf("missing bearer token")
	}
	raw := strings.TrimPrefix(auth, "Bearer ")
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return uuid.Nil, fmt.Errorf("invalid jwt format")
	}
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return uuid.Nil, fmt.Errorf("decode jwt claims: %w", err)
	}
	var claims struct {
		UserID string `json:"user_id"`
		Sub    string `json:"sub"`
	}
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return uuid.Nil, fmt.Errorf("parse jwt claims: %w", err)
	}
	userID := claims.UserID
	if userID == "" {
		userID = claims.Sub
	}
	return uuid.Parse(userID)
}

// ResolveSession resolves a strategy_key to a session UUID, creating one if needed.
func (s *AIServer) ResolveSession(ctx context.Context, req *connect.Request[antv1.ResolveSessionRequest]) (*connect.Response[antv1.ResolveSessionResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if req.Msg.StrategyKey == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("strategy_key is required"))
	}
	sess, err := s.session.GetOrCreate(ctx, uid, req.Msg.StrategyKey, req.Msg.Title)
	if err != nil {
		s.log.Error("ResolveSession", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))
	}
	msgs := make([]*antv1.ConversationMessage, len(sess.Messages))
	for i, m := range sess.Messages {
		msgs[i] = &antv1.ConversationMessage{
			Id:        m.ID.String(),
			Role:      m.Role,
			Content:   m.Content,
			CreatedAt: timestamppb.New(m.CreatedAt),
		}
	}
	return connect.NewResponse(&antv1.ResolveSessionResponse{
		SessionId: sess.ID.String(),
		Messages:  msgs,
		Created:   len(sess.Messages) == 0,
	}), nil
}

// UpdateSessionStrategyKey migrates a session's strategy_key (e.g. draft:* → strategy:<id>).
func (s *AIServer) UpdateSessionStrategyKey(ctx context.Context, req *connect.Request[antv1.UpdateSessionStrategyKeyRequest]) (*connect.Response[antv1.UpdateSessionStrategyKeyResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	sid, err := uuid.Parse(req.Msg.SessionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid session_id"))
	}
	if req.Msg.StrategyKey == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("strategy_key is required"))
	}
	if err := s.session.UpdateStrategyKey(ctx, sid, uid, req.Msg.StrategyKey); err != nil {
		s.log.Warn("UpdateSessionStrategyKey failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))
	}
	return connect.NewResponse(&antv1.UpdateSessionStrategyKeyResponse{Success: true}), nil
}
