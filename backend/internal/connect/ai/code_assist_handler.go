package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	systemai "anttrader/internal/service/systemai"
)

const codeAssistModel = "gpt-4o"
const maxCodeLen = 100 * 1024
const maxInstrLen = 4 * 1024

// CodeAssistServer implements ant.v1.CodeAssistServiceHandler.
type CodeAssistServer struct {
	systemSvc *systemai.Service
	log       *zap.Logger
}

var _ antv1c.CodeAssistServiceHandler = (*CodeAssistServer)(nil)

func NewCodeAssistServer(systemSvc *systemai.Service, log *zap.Logger) *CodeAssistServer {
	return &CodeAssistServer{systemSvc: systemSvc, log: log}
}

// protoHistoryToChat converts proto CodeChatMessage list to systemai ChatMessage list.
func protoHistoryToChat(protoMsgs []*antv1.CodeChatMessage) []systemai.ChatMessage {
	out := make([]systemai.ChatMessage, len(protoMsgs))
	for i, m := range protoMsgs {
		out[i] = systemai.ChatMessage{Role: m.Role, Content: m.Content}
	}
	return out
}

func (s *CodeAssistServer) ReviseCode(ctx context.Context, req *connect.Request[antv1.ReviseCodeRequest]) (*connect.Response[antv1.ReviseCodeResponse], error) {
	code := req.Msg.Code
	instruction := req.Msg.Instruction

	if len(code) > maxCodeLen {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("code too large: %d bytes", len(code)))
	}
	if len(instruction) > maxInstrLen {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("instruction too long: %d bytes", len(instruction)))
	}

	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	var sysPrompt, userMsg string
	if strings.TrimSpace(code) == "" {
		sysPrompt = "You are an expert quantitative trading strategy developer. " +
			"Generate a complete Python trading strategy based on the user's description. " +
			"The strategy must include: a run(ctx) function, entry/exit logic, stop-loss, " +
			"and position sizing. Return ONLY the Python code, no explanations or markdown fences."
		userMsg = fmt.Sprintf("Create a trading strategy: %s", instruction)
	} else {
		sysPrompt = "You are an expert quantitative trading strategy developer. " +
			"Revise the following Python trading strategy code according to the user's instruction. " +
			"Return ONLY the revised Python code, no explanations or markdown fences."
		userMsg = fmt.Sprintf("Instruction: %s\n\nCode:\n```python\n%s\n```", instruction, code)
	}
	messages := systemai.BuildChatMessages(sysPrompt, userMsg, protoHistoryToChat(req.Msg.History))

	revised, err := s.systemSvc.ChatCompletion(ctx, uid, messages, codeAssistModel)
	if err != nil {
		s.log.Warn("CodeAssist: ReviseCode LLM call failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("%s", systemai.FriendlyError(err)))
	}

	return connect.NewResponse(&antv1.ReviseCodeResponse{
		Text:   revised,
		Python: revised,
	}), nil
}

func (s *CodeAssistServer) ReviseCodeStream(
	ctx context.Context,
	req *connect.Request[antv1.ReviseCodeRequest],
	stream *connect.ServerStream[antv1.ReviseCodeStreamChunk],
) error {
	code := req.Msg.Code
	instruction := req.Msg.Instruction

	if len(code) > maxCodeLen {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("code too large: %d bytes", len(code)))
	}
	if len(instruction) > maxInstrLen {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("instruction too long: %d bytes", len(instruction)))
	}

	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return err
	}

	var sysPrompt, userMsg string
	if strings.TrimSpace(code) == "" {
		sysPrompt = "You are an expert quantitative trading strategy developer. " +
			"Generate a complete Python trading strategy based on the user's description. " +
			"The strategy must include: a run(ctx) function, entry/exit logic, stop-loss, " +
			"and position sizing. Return ONLY the Python code, no explanations or markdown fences."
		userMsg = fmt.Sprintf("Create a trading strategy: %s", instruction)
	} else {
		sysPrompt = "You are an expert quantitative trading strategy developer. " +
			"Revise the following Python trading strategy code according to the user's instruction. " +
			"Return ONLY the revised Python code, no explanations or markdown fences."
		userMsg = fmt.Sprintf("Instruction: %s\n\nCode:\n```python\n%s\n```", instruction, code)
	}
	messages := systemai.BuildChatMessages(sysPrompt, userMsg, protoHistoryToChat(req.Msg.History))

	var fullText strings.Builder
	err = s.systemSvc.ChatCompletionStream(ctx, uid, messages, codeAssistModel,
		func(chunk systemai.ChatStreamChunk) error {
			fullText.WriteString(chunk.Content)
			return stream.Send(&antv1.ReviseCodeStreamChunk{
				Delta: chunk.Content,
				Done:  chunk.Done,
			})
		})
	if err != nil {
		s.log.Warn("CodeAssist: ReviseCodeStream LLM call failed", zap.Error(err))
		return connect.NewError(connect.CodeInternal, fmt.Errorf("%s", systemai.FriendlyError(err)))
	}

	revised := fullText.String()
	return stream.Send(&antv1.ReviseCodeStreamChunk{Delta: "", Python: revised, Done: true})
}

func (s *CodeAssistServer) ExplainCode(ctx context.Context, req *connect.Request[antv1.ExplainCodeRequest]) (*connect.Response[antv1.ExplainCodeResponse], error) {
	code := req.Msg.Code

	if len(code) > maxCodeLen {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("code too large: %d bytes", len(code)))
	}

	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	sysPrompt := "You are an expert quantitative trading code reviewer. " +
		"Explain the following trading strategy code in clear, concise Chinese. " +
		"Cover: strategy logic, entry/exit conditions, risk management, and potential improvements. " +
		"Keep the explanation under 300 words."
	userMsg := fmt.Sprintf("Please explain this trading strategy:\n```python\n%s\n```", code)
	messages := systemai.BuildChatMessages(sysPrompt, userMsg, nil)

	explanation, err := s.systemSvc.ChatCompletion(ctx, uid, messages, codeAssistModel)
	if err != nil {
		s.log.Warn("CodeAssist: ExplainCode LLM call failed", zap.Error(err))
		explanation = "该策略代码的分析暂时不可用，请稍后重试。"
	}

	return connect.NewResponse(&antv1.ExplainCodeResponse{Explanation: explanation}), nil
}

func (s *CodeAssistServer) ValidateStrategyExtended(ctx context.Context, req *connect.Request[antv1.ValidateStrategyExtendedRequest]) (*connect.Response[antv1.ValidateStrategyExtendedResponse], error) {
	code := req.Msg.Code

	if len(code) > maxCodeLen {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("code too large: %d bytes", len(code)))
	}

	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	sysPrompt := "You are a trading strategy code validator. " +
		"Review the following Python strategy code and identify issues. " +
		"Return a JSON object with fields: valid (bool), errors (string array), warnings (string array). " +
		"Check for: missing stop-loss, missing take-profit, position sizing, error handling, " +
		"indicator usage correctness, and data boundary handling. " +
		"Respond with ONLY valid JSON, no markdown fences."
	userMsg := fmt.Sprintf("Validate this trading strategy:\n```python\n%s\n```", code)
	messages := systemai.BuildChatMessages(sysPrompt, userMsg, nil)

	result, err := s.systemSvc.ChatCompletion(ctx, uid, messages, codeAssistModel)
	if err != nil {
		s.log.Warn("CodeAssist: ValidateStrategyExtended LLM call failed", zap.Error(err))
		return connect.NewResponse(&antv1.ValidateStrategyExtendedResponse{
			Valid: false, Errors: []string{"AI 验证服务暂时不可用，请稍后重试。"}, Warnings: []string{},
		}), nil
	}

	var parsed struct {
		Valid    bool     `json:"valid"`
		Errors   []string `json:"errors"`
		Warnings []string `json:"warnings"`
	}
	cleaned := stripMarkdownFences(result)
	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		s.log.Warn("CodeAssist: ValidateStrategyExtended failed to parse LLM JSON",
			zap.Error(err), zap.String("raw", cleaned[:min(len(cleaned), 200)]))
		return connect.NewResponse(&antv1.ValidateStrategyExtendedResponse{
			Valid: false, Errors: []string{"AI 验证结果解析失败，请检查策略代码格式。"}, Warnings: []string{},
		}), nil
	}

	return connect.NewResponse(&antv1.ValidateStrategyExtendedResponse{
		Valid: parsed.Valid, Errors: parsed.Errors, Warnings: parsed.Warnings,
	}), nil
}

func stripMarkdownFences(s string) string {
	for _, fence := range []string{"```json", "```"} {
		t := strings.TrimSpace(s)
		if strings.HasPrefix(t, fence) {
			t = t[len(fence):]
			if idx := strings.LastIndex(t, "```"); idx >= 0 {
				t = t[:idx]
			}
			return strings.TrimSpace(t)
		}
	}
	return s
}
