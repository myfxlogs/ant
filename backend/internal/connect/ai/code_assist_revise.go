package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/ai"
	systemai "anttrader/internal/service/systemai"
)

func (s *CodeAssistServer) ReviseCode(ctx context.Context, req *connect.Request[antv1.ReviseCodeRequest]) (*connect.Response[antv1.ReviseCodeResponse], error) {
	code := req.Msg.Code
	instruction := req.Msg.Instruction
	if err := validateCodeAssistLimits(code, instruction); err != nil {
		return nil, err
	}
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	pc := ai.BuildContext(ai.BuildContextInput{Code: code, Message: instruction, Locale: req.Msg.Locale, IsMQL: isMQLCode(code)})
	messages := systemai.BuildChatMessages(pc.SystemPrompt, pc.UserMessage, protoHistoryToChat(req.Msg.History))
	revised, err := s.systemSvc.ChatCompletion(ctx, uid, messages)
	if err != nil {
		s.log.Warn("CodeAssist: ReviseCode LLM call failed", zap.Error(err))
		if errors.Is(err, systemai.ErrInsufficientBalance) {
		return nil, systemai.WrapAIError(err)
	}
	return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("%s", systemai.FriendlyError(err)))
	}

	result := revised
	if pc.Mode == ai.ModeRepair {
		if code := extractCodeFromRepair(revised, isMQLCode(code)); code != "" {
			result = code
		}
	}
	return connect.NewResponse(&antv1.ReviseCodeResponse{Text: result, Python: result}), nil
}

func (s *CodeAssistServer) ReviseCodeStream(
	ctx context.Context,
	req *connect.Request[antv1.ReviseCodeRequest],
	stream *connect.ServerStream[antv1.ReviseCodeStreamChunk],
) error {
	code := req.Msg.Code
	instruction := req.Msg.Instruction
	if err := validateCodeAssistLimits(code, instruction); err != nil {
		return err
	}
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return err
	}

	pc := ai.BuildContext(ai.BuildContextInput{Code: code, Message: instruction, Locale: req.Msg.Locale, IsMQL: isMQLCode(code)})
	messages := systemai.BuildChatMessages(pc.SystemPrompt, pc.UserMessage, protoHistoryToChat(req.Msg.History))
	var fullText strings.Builder
	err = s.systemSvc.ChatCompletionStream(ctx, uid, messages,
		func(chunk systemai.ChatStreamChunk) error {
			fullText.WriteString(chunk.Content)
			return stream.Send(&antv1.ReviseCodeStreamChunk{Delta: chunk.Content, Done: chunk.Done})
		})
	if err != nil {
		s.log.Warn("CodeAssist: ReviseCodeStream LLM call failed", zap.Error(err))
		if errors.Is(err, systemai.ErrInsufficientBalance) {
	return systemai.WrapAIError(err)
	}
	return connect.NewError(connect.CodeInternal, fmt.Errorf("%s", systemai.FriendlyError(err)))
	}

	// Repair mode post-processing
	result := fullText.String()
	if pc.Mode == ai.ModeRepair {
		if code := extractCodeFromRepair(result, isMQLCode(code)); code != "" {
			result = code
		}
	}

	// Auto-persist to session
	if req.Msg.SessionId != "" {
		sid, parseErr := uuid.Parse(req.Msg.SessionId)
		if parseErr == nil {
			if err := s.session.AppendExchange(ctx, sid, uid, instruction, result); err != nil {
				s.log.Warn("session append failed", zap.Error(err))
			}
		}
	}

	return stream.Send(&antv1.ReviseCodeStreamChunk{Delta: "", Python: result, Done: true})
}

func validateCodeAssistLimits(code, instruction string) error {
	if len(code) > maxCodeLen {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("code too large: %d bytes", len(code)))
	}
	if len(instruction) > maxInstrLen {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("instruction too long: %d bytes", len(instruction)))
	}
	return nil
}
