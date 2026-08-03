package ai

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"connectrpc.com/connect"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/ai"
	"alphaforge/internal/repository"
	systemai "alphaforge/internal/service/systemai"
)

// AIServer implements the ant.v1.AIServiceHandler interface.
type AIServer struct {
	systemSvc      *systemai.Service
	conversations  *repository.AIConversationRepository
	session        *ai.ConversationSession
	agentDefRepo   *repository.AIAgentDefinitionRepository
	feedbackRepo   *repository.SessionFeedbackRepository
	log            *zap.Logger
}

// NewAIServer creates an AI service ConnectRPC handler.
func NewAIServer(systemSvc *systemai.Service, conversations *repository.AIConversationRepository, session *ai.ConversationSession, log *zap.Logger) *AIServer {
	return &AIServer{systemSvc: systemSvc, conversations: conversations, session: session, log: log}
}

// SetFeedbackRepo injects the session feedback repository.
func (s *AIServer) SetFeedbackRepo(repo *repository.SessionFeedbackRepository) {
	s.feedbackRepo = repo
}

var _ antv1c.AIServiceHandler = (*AIServer)(nil)

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

// SubmitSessionFeedback records a user's good/bad rating for a generation session.
func (s *AIServer) SubmitSessionFeedback(ctx context.Context, req *connect.Request[antv1.SubmitSessionFeedbackRequest]) (*connect.Response[antv1.SubmitSessionFeedbackResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	sid, err := uuid.Parse(req.Msg.SessionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid session_id"))
	}
	if req.Msg.Rating != "good" && req.Msg.Rating != "bad" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("rating must be good or bad"))
	}
	if s.feedbackRepo == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("feedback not configured"))
	}
	if err := s.feedbackRepo.Upsert(ctx, sid, uid, req.Msg.Rating, req.Msg.Reason); err != nil {
		s.log.Warn("SubmitSessionFeedback failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))
	}
	return connect.NewResponse(&antv1.SubmitSessionFeedbackResponse{Success: true}), nil
}

// GetSessionFeedback returns the user's feedback for a session, if any.
func (s *AIServer) GetSessionFeedback(ctx context.Context, req *connect.Request[antv1.GetSessionFeedbackRequest]) (*connect.Response[antv1.GetSessionFeedbackResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	sid, err := uuid.Parse(req.Msg.SessionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid session_id"))
	}
	if s.feedbackRepo == nil {
		return connect.NewResponse(&antv1.GetSessionFeedbackResponse{}), nil
	}
	f, err := s.feedbackRepo.GetBySession(ctx, sid, uid)
	if err != nil || f == nil {
		return connect.NewResponse(&antv1.GetSessionFeedbackResponse{}), nil
	}
	return connect.NewResponse(&antv1.GetSessionFeedbackResponse{
		Rating:       f.Rating,
		Reason:       f.Reason,
		CreatedAtTsMs: f.CreatedAt.UnixMilli(),
	}), nil
}
