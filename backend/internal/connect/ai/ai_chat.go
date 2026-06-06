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

	systemPrompt := "You are a helpful quantitative trading assistant."
	messages := systemai.BuildChatMessages(systemPrompt, m.Message, nil)
	reply, err := s.systemSvc.ChatCompletion(ctx, uid, messages, "")
	if err != nil {
		s.log.Error("Chat: ChatCompletion failed", zap.String("user_id", uid.String()))
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

	systemPrompt := "You are a helpful quantitative trading assistant."
	var fullReply strings.Builder

	err = s.systemSvc.ChatCompletionStream(ctx, uid, systemai.BuildChatMessages(systemPrompt, m.Message, nil), "", func(chunk systemai.ChatStreamChunk) error {
		fullReply.WriteString(chunk.Content)
		sendChunk := &antv1.ChatStreamChunk{Delta: chunk.Content, Done: chunk.Done}
		return stream.Send(sendChunk)
	})
	if err != nil {
		s.log.Error("ChatStream: streaming failed",
			zap.String("user_id", uid.String()),
		)
		return connect.NewError(connect.CodeInternal, fmt.Errorf("AI streaming temporarily unavailable"))
	}

	// Persist messages to conversation after streaming completes.
	if m.ConversationId != "" {
		cid, parseErr := uuid.Parse(m.ConversationId)
		if parseErr != nil {
			return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid conversation_id: %w", parseErr))
		}
		reply := fullReply.String()
		if _, lookupErr := s.conversations.GetByID(ctx, cid, uid); lookupErr == nil {
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
