package ai

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	systemai "anttrader/internal/service/systemai"
)

// loadConversationHistory returns the last N messages from a conversation
// as systemai.ChatMessage history. Returns nil if conversation_id is empty
// or the conversation doesn't exist.
func (s *AIServer) loadConversationHistory(ctx context.Context, uid uuid.UUID, conversationID string, maxHistory int) []systemai.ChatMessage {
	if conversationID == "" {
		return nil
	}
	cid, err := uuid.Parse(conversationID)
	if err != nil {
		return nil
	}
	msgs, err := s.conversations.GetMessages(ctx, uid, cid)
	if err != nil || len(msgs) == 0 {
		return nil
	}
	if maxHistory > 0 && len(msgs) > maxHistory {
		msgs = msgs[len(msgs)-maxHistory:]
	}
	history := make([]systemai.ChatMessage, len(msgs))
	for i, m := range msgs {
		history[i] = systemai.ChatMessage{Role: m.Role, Content: m.Content}
	}
	return history
}

// Chat handles a single-turn AI conversation request.
func (s *AIServer) Chat(ctx context.Context, req *connect.Request[antv1.ChatRequest]) (*connect.Response[antv1.ChatResponse], error) {
	m := req.Msg
	if len(m.Message) > 10000 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("message too long: %d bytes", len(m.Message)))
	}

	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	systemPrompt := LangPrompt(LangFromAccept(req.Header().Get("Accept-Language")))
	history := s.loadConversationHistory(ctx, uid, m.ConversationId, 20)
	s.log.Info("AI Chat", zap.String("lang", LangFromAccept(req.Header().Get("Accept-Language"))), zap.String("user_id", uid.String()), zap.Int("history", len(history)))
	reply, err := s.systemSvc.ChatCompletion(ctx, uid, systemai.BuildChatMessages(systemPrompt, m.Message, history))
	if err != nil {
		s.log.Error("Chat: ChatCompletion failed", zap.String("user_id", uid.String()))
		return nil, systemai.WrapAIError(err)
	}

	if m.ConversationId != "" {
		cid, parseErr := uuid.Parse(m.ConversationId)
		if parseErr == nil {
			if _, aErr := s.conversations.AddMessage(ctx, uid, cid, "user", m.Message); aErr != nil {
				s.log.Warn("Chat: AddMessage user failed", zap.Error(aErr))
			}
			if _, aErr := s.conversations.AddMessage(ctx, uid, cid, "assistant", reply); aErr != nil {
				s.log.Warn("Chat: AddMessage assistant failed", zap.Error(aErr))
			}
			if err := s.conversations.Touch(ctx, cid, uid); err != nil {
				s.log.Warn("Chat: Touch conversation failed", zap.String("cid", cid.String()), zap.Error(err))
			}
		}
	}

	return connect.NewResponse(&antv1.ChatResponse{
		Message:     reply,
		Suggestions: []string{"分析当前持仓风险", "推荐交易策略", "解释技术指标"},
	}), nil
}

// ChatStream handles a streaming AI conversation.
func (s *AIServer) ChatStream(ctx context.Context, req *connect.Request[antv1.ChatRequest], stream *connect.ServerStream[antv1.ChatStreamChunk]) error {
	m := req.Msg
	if len(m.Message) > 10000 {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("message too long: %d bytes", len(m.Message)))
	}

	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return connect.NewError(connect.CodeUnauthenticated, err)
	}

	systemPrompt := LangPrompt(LangFromAccept(req.Header().Get("Accept-Language")))
	history := s.loadConversationHistory(ctx, uid, m.ConversationId, 20)
	var fullReply strings.Builder

	err = s.systemSvc.ChatCompletionStream(ctx, uid, systemai.BuildChatMessages(systemPrompt, m.Message, history), func(chunk systemai.ChatStreamChunk) error {
		fullReply.WriteString(chunk.Content)
		sendChunk := &antv1.ChatStreamChunk{Delta: chunk.Content, Done: chunk.Done}
		return stream.Send(sendChunk)
	})
	if err != nil {
		s.log.Error("ChatStream: streaming failed", zap.String("user_id", uid.String()))
		return systemai.WrapAIError(err)
	}

	// Persist messages to conversation after streaming completes.
	if m.ConversationId != "" {
		cid, parseErr := uuid.Parse(m.ConversationId)
		if parseErr == nil {
			reply := fullReply.String()
			if _, aErr := s.conversations.AddMessage(ctx, uid, cid, "user", m.Message); aErr != nil {
				s.log.Warn("ChatStream: AddMessage user failed", zap.Error(aErr))
			}
			if _, aErr := s.conversations.AddMessage(ctx, uid, cid, "assistant", reply); aErr != nil {
				s.log.Warn("ChatStream: AddMessage assistant failed", zap.Error(aErr))
			}
			if err := s.conversations.Touch(ctx, cid, uid); err != nil {
				s.log.Warn("Chat: Touch conversation failed", zap.String("cid", cid.String()), zap.Error(err))
			}
		}
	}

	return nil
}
