package ai

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/repository"
	systemai "anttrader/internal/service/systemai"
)

// AIServer implements the ant.v1.AIServiceHandler interface.
type AIServer struct {
	systemSvc    *systemai.Service
	conversations *repository.AIConversationRepository
	log          *zap.Logger
}

// NewAIServer creates an AI service ConnectRPC handler.
func NewAIServer(systemSvc *systemai.Service, conversations *repository.AIConversationRepository, log *zap.Logger) *AIServer {
	return &AIServer{systemSvc: systemSvc, conversations: conversations, log: log}
}

var _ antv1c.AIServiceHandler = (*AIServer)(nil)

// ── Chat ──

func (s *AIServer) Chat(ctx context.Context, req *connect.Request[antv1.ChatRequest]) (*connect.Response[antv1.ChatResponse], error) {
	m := req.Msg
	if len(m.Message) > 10000 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("message too long: %d bytes", len(m.Message)))
	}

	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	systemPrompt := "You are a helpful quantitative trading assistant."
	reply, err := s.systemSvc.ChatCompletion(ctx, uid, systemPrompt, m.Message, "")
	if err != nil {
		s.log.Error("Chat: ChatCompletion failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("AI service temporarily unavailable"))
	}

	if m.ConversationId != "" {
		cid, parseErr := uuid.Parse(m.ConversationId)
		if parseErr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid conversation_id: %w", parseErr))
		}
			if _, lookupErr := s.conversations.GetByID(ctx, cid, uid); lookupErr == nil {
				if _, aErr := s.conversations.AddMessage(ctx, uid, cid, "user", m.Message); aErr != nil {
					s.log.Warn("Chat: AddMessage user failed", zap.Error(aErr))
				}
				if _, aErr := s.conversations.AddMessage(ctx, uid, cid, "assistant", reply); aErr != nil {
					s.log.Warn("Chat: AddMessage assistant failed", zap.Error(aErr))
				}
				_ = s.conversations.Touch(ctx, cid, uid)
		}
	}

	return connect.NewResponse(&antv1.ChatResponse{
		Message:     reply,
		Suggestions: []string{"分析当前持仓风险", "推荐交易策略", "解释技术指标"},
	}), nil
}

func (s *AIServer) ChatStream(ctx context.Context, req *connect.Request[antv1.ChatRequest], stream *connect.ServerStream[antv1.ChatStreamChunk]) error {
	m := req.Msg
	if len(m.Message) > 10000 {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("message too long: %d bytes", len(m.Message)))
	}

	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return connect.NewError(connect.CodeUnauthenticated, err)
	}

	systemPrompt := "You are a helpful quantitative trading assistant."
	reply, err := s.systemSvc.ChatCompletion(ctx, uid, systemPrompt, m.Message, "")
	if err != nil {
		s.log.Error("ChatStream: ChatCompletion failed", zap.Error(err))
		return connect.NewError(connect.CodeInternal, fmt.Errorf("AI service temporarily unavailable"))
	}

	runes := []rune(reply)
	for i, c := range runes {
		chunk := &antv1.ChatStreamChunk{Delta: string(c), Done: false}
		if i == len(runes)-1 {
			chunk.Done = true
			chunk.PromptTokens = 10
			chunk.CompletionTokens = int32(len(runes) / 4)
		}
		if sendErr := stream.Send(chunk); sendErr != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("send chat stream chunk: %w", sendErr))
		}
	}

	// Persist messages to conversation after streaming completes.
	if m.ConversationId != "" {
		cid, parseErr := uuid.Parse(m.ConversationId)
		if parseErr != nil {
			return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid conversation_id: %w", parseErr))
		}
			if _, lookupErr := s.conversations.GetByID(ctx, cid, uid); lookupErr == nil {
				if _, aErr := s.conversations.AddMessage(ctx, uid, cid, "user", m.Message); aErr != nil {
					s.log.Warn("ChatStream: AddMessage user failed", zap.Error(aErr))
				}
				if _, aErr := s.conversations.AddMessage(ctx, uid, cid, "assistant", reply); aErr != nil {
					s.log.Warn("ChatStream: AddMessage assistant failed", zap.Error(aErr))
				}
				_ = s.conversations.Touch(ctx, cid, uid)
		}
	}

	return nil
}

// ── Conversations ──

func (s *AIServer) ListConversations(ctx context.Context, req *connect.Request[antv1.ListConversationsRequest]) (*connect.Response[antv1.ListConversationsResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	convs, err := s.conversations.ListByUser(ctx, uid)
	if err != nil {
		s.log.Error("ListConversations", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))
	}
	var out []*antv1.ConversationSummary
	for _, c := range convs {
		out = append(out, &antv1.ConversationSummary{
			Id:           c.ID.String(),
			Title:        c.Title,
			MessageCount: int32(c.MessageCount),
			CreatedAt:    timestamppb.New(c.CreatedAt),
			UpdatedAt:    timestamppb.New(c.UpdatedAt),
		})
	}
	return connect.NewResponse(&antv1.ListConversationsResponse{Conversations: out}), nil
}

func (s *AIServer) GetConversation(ctx context.Context, req *connect.Request[antv1.GetConversationRequest]) (*connect.Response[antv1.GetConversationResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	cid, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	conv, err := s.conversations.GetByID(ctx, cid, uid)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("conversation not found"))
	}
	summary := &antv1.ConversationSummary{
		Id:        conv.ID.String(),
		Title:     conv.Title,
		CreatedAt: timestamppb.New(conv.CreatedAt),
		UpdatedAt: timestamppb.New(conv.UpdatedAt),
	}
	msgs, err := s.conversations.GetMessages(ctx, uid, cid)
	if err != nil {
		s.log.Error("GetConversation messages", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))
	}
	var messages []*antv1.ConversationMessage
	for _, m := range msgs {
		messages = append(messages, &antv1.ConversationMessage{
			Id:        m.ID.String(),
			Role:      m.Role,
			Content:   m.Content,
			CreatedAt: timestamppb.New(m.CreatedAt),
		})
	}
	return connect.NewResponse(&antv1.GetConversationResponse{
		Conversation: summary,
		Messages:     messages,
	}), nil
}

func (s *AIServer) CreateConversation(ctx context.Context, req *connect.Request[antv1.CreateConversationRequest]) (*connect.Response[antv1.CreateConversationResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	title := req.Msg.Title
	if title == "" {
		title = "新对话"
	}
	if len(title) > 200 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("title too long: %d characters", len(title)))
	}
	conv, err := s.conversations.Create(ctx, uid, title)
	if err != nil {
		s.log.Error("CreateConversation", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))
	}
	return connect.NewResponse(&antv1.CreateConversationResponse{
		Conversation: &antv1.ConversationSummary{
			Id:        conv.ID.String(),
			Title:     conv.Title,
			CreatedAt: timestamppb.New(conv.CreatedAt),
			UpdatedAt: timestamppb.New(conv.UpdatedAt),
		},
	}), nil
}

func (s *AIServer) DeleteConversation(ctx context.Context, req *connect.Request[antv1.DeleteConversationRequest]) (*connect.Response[antv1.DeleteConversationResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	cid, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	// Verify ownership before deleting messages or the conversation itself.
	if _, err := s.conversations.GetByID(ctx, cid, uid); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("conversation not found"))
	}
	if err := s.conversations.DeleteMessagesByConversation(ctx, uid, cid); err != nil {
		s.log.Error("DeleteConversation: failed to delete messages", zap.String("cid", cid.String()), zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))
	}
	if err := s.conversations.Delete(ctx, cid, uid); err != nil {
		s.log.Error("DeleteConversation", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))
	}
	return connect.NewResponse(&antv1.DeleteConversationResponse{}), nil
}

func (s *AIServer) UpdateConversationTitle(ctx context.Context, req *connect.Request[antv1.UpdateConversationTitleRequest]) (*connect.Response[antv1.UpdateConversationTitleResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if len(req.Msg.Title) > 200 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("title too long: %d characters", len(req.Msg.Title)))
	}
	cid, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.conversations.UpdateTitle(ctx, cid, uid, req.Msg.Title); err != nil {
		s.log.Error("UpdateConversationTitle", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))
	}
	return connect.NewResponse(&antv1.UpdateConversationTitleResponse{}), nil
}

// ── Agents ──

var defaultAgents = []*antv1.AIAgentDefinition{
	{AgentKey: "strategist", Type: "primary", Name: "策略分析师", Identity: "资深量化交易策略师", Enabled: true, Position: 1},
	{AgentKey: "risk_manager", Type: "secondary", Name: "风控经理", Identity: "严格的风险控制专家", Enabled: true, Position: 2},
	{AgentKey: "executor", Type: "secondary", Name: "执行顾问", Identity: "交易执行优化专家", Enabled: false, Position: 3},
	{AgentKey: "researcher", Type: "secondary", Name: "市场研究员", Identity: "宏观经济和行业研究员", Enabled: false, Position: 4},
}

func (s *AIServer) ListAgents(ctx context.Context, req *connect.Request[antv1.ListAgentsRequest]) (*connect.Response[antv1.ListAgentsResponse], error) {
	return connect.NewResponse(&antv1.ListAgentsResponse{Agents: defaultAgents}), nil
}
