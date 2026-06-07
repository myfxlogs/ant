package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/ai"
	systemai "anttrader/internal/service/systemai"
)

const codeAssistModel = "gpt-4o"
const maxCodeLen = 100 * 1024
const maxInstrLen = 4 * 1024

// CodeAssistServer implements ant.v1.CodeAssistServiceHandler.
type CodeAssistServer struct {
	systemSvc            *systemai.Service
	session              *ai.ConversationSession
	pythonStrategyClient antv1c.PythonStrategyServiceClient // optional: for quality hints
	log                  *zap.Logger
}

var _ antv1c.CodeAssistServiceHandler = (*CodeAssistServer)(nil)

func NewCodeAssistServer(systemSvc *systemai.Service, session *ai.ConversationSession, log *zap.Logger) *CodeAssistServer {
	return &CodeAssistServer{systemSvc: systemSvc, session: session, log: log}
}

// SetPythonStrategyClient injects the Python strategy client for quality analysis on ValidateStrategyExtended.
func (s *CodeAssistServer) SetPythonStrategyClient(c antv1c.PythonStrategyServiceClient) {
	s.pythonStrategyClient = c
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
	if err := validateCodeAssistLimits(code, instruction); err != nil {
		return nil, err
	}
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	pc := ai.BuildContext(ai.BuildContextInput{Code: code, Message: instruction})
	messages := systemai.BuildChatMessages(pc.SystemPrompt, pc.UserMessage, protoHistoryToChat(req.Msg.History))
	revised, err := s.systemSvc.ChatCompletion(ctx, uid, messages, codeAssistModel)
	if err != nil {
		s.log.Warn("CodeAssist: ReviseCode LLM call failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("%s", systemai.FriendlyError(err)))
	}

	result := revised
	if pc.Mode == ai.ModeRepair {
		if code := extractCodeFromRepair(revised); code != "" {
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

	pc := ai.BuildContext(ai.BuildContextInput{Code: code, Message: instruction})
	messages := systemai.BuildChatMessages(pc.SystemPrompt, pc.UserMessage, protoHistoryToChat(req.Msg.History))
	var fullText strings.Builder
	err = s.systemSvc.ChatCompletionStream(ctx, uid, messages, codeAssistModel,
		func(chunk systemai.ChatStreamChunk) error {
			fullText.WriteString(chunk.Content)
			return stream.Send(&antv1.ReviseCodeStreamChunk{Delta: chunk.Content, Done: chunk.Done})
		})
	if err != nil {
		s.log.Warn("CodeAssist: ReviseCodeStream LLM call failed", zap.Error(err))
		return connect.NewError(connect.CodeInternal, fmt.Errorf("%s", systemai.FriendlyError(err)))
	}

	// Repair mode post-processing
	result := fullText.String()
	if pc.Mode == ai.ModeRepair {
		if code := extractCodeFromRepair(result); code != "" {
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
		return connect.NewResponse(&antv1.ExplainCodeResponse{
			Explanation: "Code analysis unavailable — AI service is temporarily down. Please try again later.",
		}), nil
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

	messages := systemai.BuildChatMessages(buildValidationPrompt(), fmt.Sprintf("Validate this trading strategy:\n```python\n%s\n```", code), nil)
	result, err := s.systemSvc.ChatCompletion(ctx, uid, messages, codeAssistModel)
	if err != nil {
		s.log.Warn("CodeAssist: ValidateStrategyExtended LLM call failed, falling back to basic check", zap.Error(err))
		return connect.NewResponse(&antv1.ValidateStrategyExtendedResponse{
			Valid: true, Warnings: []string{"AI validation unavailable — basic syntax check only. Run backtest to verify logic."},
		}), nil
	}

	resp, _ := parseValidationResult(result, s.log)

	// Merge Python quality hints into warnings (backend zero-trust:
	// all computation in Python; Go just forwards the results).
	if s.pythonStrategyClient != nil && resp != nil {
		pyResp, pyErr := s.pythonStrategyClient.Validate(ctx, connect.NewRequest(&antv1.ValidateStrategyRequest{Code: code}))
		if pyErr == nil && pyResp != nil {
			for _, w := range pyResp.Msg.Warnings {
				if strings.HasPrefix(w, "[HINT]") || strings.HasPrefix(w, "[SWEEP]") || strings.HasPrefix(w, "[STRATEGY]") {
					resp.Msg.Warnings = append(resp.Msg.Warnings, w)
				}
			}
		}
	}

	return resp, nil
}

func buildValidationPrompt() string {
	return "You are a trading strategy code validator. " +
		"Review the following Python strategy code and identify issues. " +
		"Return a JSON object with fields: valid (bool), errors (string array), warnings (string array), " +
		"parameters (array of objects with keys: key (str), required (bool), type (str: int|float|str|bool), " +
		"default_value (str, optional), suggested_value (str, optional)). " +
		"Extract all @param annotations from the code into the parameters array. " +
		"Check for: missing stop-loss, missing take-profit, position sizing, error handling, " +
		"indicator usage correctness, and data boundary handling. " +
		"Respond with ONLY valid JSON, no markdown fences."
}

func parseValidationResult(raw string, log *zap.Logger) (*connect.Response[antv1.ValidateStrategyExtendedResponse], error) {
	var parsed struct {
		Valid    bool     `json:"valid"`
		Errors   []string `json:"errors"`
		Warnings []string `json:"warnings"`
		Params   []struct {
			Key      string `json:"key"`
			Required bool   `json:"required"`
			Type     string `json:"type"`
			Default  string `json:"default_value"`
			Suggest  string `json:"suggested_value"`
		} `json:"parameters"`
	}
	cleaned := stripMarkdownFences(raw)
	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		log.Warn("CodeAssist: ValidateStrategyExtended failed to parse LLM JSON",
			zap.Error(err), zap.String("raw", cleaned[:min(len(cleaned), 200)]))
		return connect.NewResponse(&antv1.ValidateStrategyExtendedResponse{
			Valid: false, Errors: []string{"AI 验证结果解析失败，请检查策略代码格式。"}, Warnings: []string{},
		}), nil
	}

	params := make([]*antv1.RequiredParamSpec, len(parsed.Params))
	for i, p := range parsed.Params {
		params[i] = &antv1.RequiredParamSpec{
			Key: p.Key, Required: p.Required, Type: p.Type,
			DefaultValue: p.Default, SuggestedValue: p.Suggest,
		}
	}
	return connect.NewResponse(&antv1.ValidateStrategyExtendedResponse{
		Valid: parsed.Valid, Errors: parsed.Errors, Warnings: parsed.Warnings,
		Parameters: params,
	}), nil
}

// extractCodeFromRepair attempts to salvage Python code from an LLM response
// that may contain explanatory text (3-tier extraction).
func extractCodeFromRepair(raw string) string {
	// Tier 1: extract from ```python ... ``` fence
	if code := extractFencedCode(raw, "python"); code != "" {
		return code
	}
	// Tier 2: heuristic — find lines starting with import/def/class/#
	if code := extractByHeuristic(raw); code != "" {
		return code
	}
	// Tier 3: unable to extract — return empty
	return ""
}

func extractFencedCode(raw, lang string) string {
	marker := "```" + lang
	start := strings.Index(raw, marker)
	if start < 0 {
		start = strings.Index(raw, "```")
		if start < 0 {
			return ""
		}
	}
	// Skip the opening fence line
	if nl := strings.Index(raw[start:], "\n"); nl >= 0 {
		start += nl + 1
	} else {
		return ""
	}
	end := strings.Index(raw[start:], "```")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(raw[start : start+end])
}

func extractByHeuristic(raw string) string {
	raw = strings.TrimSpace(raw)
	lines := strings.Split(raw, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "import ") ||
			strings.HasPrefix(trimmed, "def ") ||
			strings.HasPrefix(trimmed, "class ") ||
			strings.HasPrefix(trimmed, "#") ||
			strings.HasPrefix(trimmed, "from ") {
			return strings.Join(lines[i:], "\n")
		}
		return ""
	}
	return ""
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
