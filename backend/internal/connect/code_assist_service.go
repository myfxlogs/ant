package connect

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	v1 "anttrader/gen/proto"
	"anttrader/internal/service"
)

type CodeAssistService struct {
	assist *service.AICodeAssistService
	python *service.PythonStrategyService
}

func NewCodeAssistService(assist *service.AICodeAssistService, python *service.PythonStrategyService) *CodeAssistService {
	return &CodeAssistService{assist: assist, python: python}
}

func (s *CodeAssistService) ReviseCode(ctx context.Context, req *connect.Request[v1.ReviseCodeRequest]) (*connect.Response[v1.ReviseCodeResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	history := make([]service.CodeChatMessage, 0, len(req.Msg.History))
	for _, h := range req.Msg.History {
		if h != nil {
			history = append(history, service.CodeChatMessage{Role: h.Role, Content: h.Content})
		}
	}
	text, python, err := s.assist.ReviseCode(ctx, userID, req.Msg.Code, req.Msg.Instruction, history, req.Msg.Locale)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&v1.ReviseCodeResponse{Text: text, Python: python}), nil
}

func (s *CodeAssistService) ExplainCode(ctx context.Context, req *connect.Request[v1.ExplainCodeRequest]) (*connect.Response[v1.ExplainCodeResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	text, err := s.assist.ExplainCode(ctx, userID, req.Msg.Code, req.Msg.Locale)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&v1.ExplainCodeResponse{Explanation: text}), nil
}

func (s *CodeAssistService) ValidateStrategyExtended(ctx context.Context, req *connect.Request[v1.ValidateStrategyExtendedRequest]) (*connect.Response[v1.ValidateStrategyExtendedResponse], error) {
	if _, err := getUserIDFromContext(ctx); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Msg.Code) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("code is required"))
	}
	resp, err := s.python.ValidateStrategy(ctx, req.Msg.Code)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	out := &v1.ValidateStrategyExtendedResponse{Valid: resp.Valid, Errors: resp.Errors, Warnings: resp.Warnings}
	for _, p := range resp.Parameters {
		out.Parameters = append(out.Parameters, toRequiredParamSpec(p))
	}
	return connect.NewResponse(out), nil
}

func toRequiredParamSpec(p map[string]interface{}) *v1.RequiredParamSpec {
	return &v1.RequiredParamSpec{
		Key:            fmt.Sprint(p["key"]),
		Required:       asBool(p["required"]),
		DefaultValue:   valueString(p["default"]),
		Type:           fmt.Sprint(p["type"]),
		SuggestedValue: valueString(p["suggested"]),
	}
}

func asBool(v interface{}) bool {
	b, _ := v.(bool)
	return b
}

func valueString(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}
