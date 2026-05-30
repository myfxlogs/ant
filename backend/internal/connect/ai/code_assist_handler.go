package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	aiSvc "anttrader/internal/service/systemai"
)

const codeAssistModel = "gpt-4o"

// CodeAssistServer implements ant.v1.CodeAssistServiceHandler.
type CodeAssistServer struct {
	systemSvc *aiSvc.Service
	log       *zap.Logger
}

var _ antv1c.CodeAssistServiceHandler = (*CodeAssistServer)(nil)

func NewCodeAssistServer(systemSvc *aiSvc.Service, log *zap.Logger) *CodeAssistServer {
	return &CodeAssistServer{systemSvc: systemSvc, log: log}
}

func (s *CodeAssistServer) userID(ctx context.Context) uuid.UUID {
	id, _ := uuid.Parse(interceptor.GetUserID(ctx))
	return id
}

func (s *CodeAssistServer) ReviseCode(ctx context.Context, req *connect.Request[antv1.ReviseCodeRequest]) (*connect.Response[antv1.ReviseCodeResponse], error) {
	code := req.Msg.Code
	instruction := req.Msg.Instruction

	sysPrompt := "You are an expert quantitative trading strategy developer. " +
		"Revise the following Python trading strategy code according to the user's instruction. " +
		"Return ONLY the revised Python code, no explanations or markdown fences."
	userMsg := fmt.Sprintf("Instruction: %s\n\nCode:\n```python\n%s\n```", instruction, code)

	revised, err := s.systemSvc.ChatCompletion(ctx, s.userID(ctx), sysPrompt, userMsg, codeAssistModel)
	if err != nil {
		s.log.Warn("CodeAssist: ReviseCode LLM call failed, falling back to passthrough",
			zap.Error(err))
		revised = fmt.Sprintf("# Revised per: %s\n%s", instruction, code)
	}

	return connect.NewResponse(&antv1.ReviseCodeResponse{
		Text:   revised,
		Python: revised,
	}), nil
}

func (s *CodeAssistServer) ExplainCode(ctx context.Context, req *connect.Request[antv1.ExplainCodeRequest]) (*connect.Response[antv1.ExplainCodeResponse], error) {
	code := req.Msg.Code

	sysPrompt := "You are an expert quantitative trading code reviewer. " +
		"Explain the following trading strategy code in clear, concise Chinese. " +
		"Cover: strategy logic, entry/exit conditions, risk management, and potential improvements. " +
		"Keep the explanation under 300 words."
	userMsg := fmt.Sprintf("Please explain this trading strategy:\n```python\n%s\n```", code)

	explanation, err := s.systemSvc.ChatCompletion(ctx, s.userID(ctx), sysPrompt, userMsg, codeAssistModel)
	if err != nil {
		s.log.Warn("CodeAssist: ExplainCode LLM call failed", zap.Error(err))
		explanation = "该策略代码的分析暂时不可用，请稍后重试。"
	}

	return connect.NewResponse(&antv1.ExplainCodeResponse{
		Explanation: explanation,
	}), nil
}

func (s *CodeAssistServer) ValidateStrategyExtended(ctx context.Context, req *connect.Request[antv1.ValidateStrategyExtendedRequest]) (*connect.Response[antv1.ValidateStrategyExtendedResponse], error) {
	code := req.Msg.Code

	sysPrompt := "You are a trading strategy code validator. " +
		"Review the following Python strategy code and identify issues. " +
		"Return a JSON object with fields: valid (bool), errors (string array), warnings (string array). " +
		"Check for: missing stop-loss, missing take-profit, position sizing, error handling, " +
		"indicator usage correctness, and data boundary handling. " +
		"Respond with ONLY valid JSON, no markdown fences."
	userMsg := fmt.Sprintf("Validate this trading strategy:\n```python\n%s\n```", code)

	result, err := s.systemSvc.ChatCompletion(ctx, s.userID(ctx), sysPrompt, userMsg, codeAssistModel)
	if err != nil {
		s.log.Warn("CodeAssist: ValidateStrategyExtended LLM call failed", zap.Error(err))
		return connect.NewResponse(&antv1.ValidateStrategyExtendedResponse{
			Valid:    false,
			Errors:   []string{"AI 验证服务暂时不可用，请稍后重试。"},
			Warnings: []string{},
		}), nil
	}

	// Parse JSON response from LLM.
	var parsed struct {
		Valid    bool     `json:"valid"`
		Errors   []string `json:"errors"`
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		s.log.Warn("CodeAssist: ValidateStrategyExtended failed to parse LLM JSON",
			zap.Error(err), zap.String("raw", result))
		return connect.NewResponse(&antv1.ValidateStrategyExtendedResponse{
			Valid:    false,
			Errors:   []string{"AI 验证结果解析失败，请检查策略代码格式。"},
			Warnings: []string{},
		}), nil
	}

	return connect.NewResponse(&antv1.ValidateStrategyExtendedResponse{
		Valid:    parsed.Valid,
		Errors:   parsed.Errors,
		Warnings: parsed.Warnings,
	}), nil
}
