package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	systemai "anttrader/internal/service/systemai"
	"anttrader/strategy/sdk"
)

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
	langTag := "go"
	if sdk.IsMQL(code) {
		langTag = "mql4"
	}
	userMsg := fmt.Sprintf("Please explain this trading strategy:\n```%s\n%s\n```", langTag, code)
	messages := systemai.BuildChatMessages(sysPrompt, userMsg, nil)

	explanation, err := s.systemSvc.ChatCompletion(ctx, uid, messages)
	if err != nil {
		s.log.Warn("CodeAssist: ExplainCode LLM call failed", zap.Error(err))
		return connect.NewResponse(&antv1.ExplainCodeResponse{
			Explanation: "Code analysis unavailable — AI service is temporarily down. Please try again later.",
		}), nil
	}

	return connect.NewResponse(&antv1.ExplainCodeResponse{Explanation: explanation}), nil
}

const maxTransformCodeLen = 65536

func (s *CodeAssistServer) TransformCode(ctx context.Context, req *connect.Request[antv1.TransformCodeRequest]) (*connect.Response[antv1.TransformCodeResponse], error) {
	code := req.Msg.SourceCode
	sourceLang := req.Msg.SourceLang

	if len(code) > maxTransformCodeLen {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("code too large: %d bytes", len(code)))
	}
	if len(code) < 20 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("code too short — please paste complete EA/indicator source"))
	}

	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	// Detect language if "auto"
	langHint := ""
	detectedLang := sourceLang
	if sourceLang == "" || sourceLang == "auto" {
		langHint = "Detect whether this is MQL4 or MQL5 code. "
	}

	sysPrompt := "You are an expert trading strategy translator. " +
		langHint +
		"Translate the following MetaTrader EA/indicator code (MQL4 or MQL5) into a " +
		"Go strategy for the AlphaForge platform.\n\n" +
		"AlphaForge uses the Go strategy SDK (package anttrader/strategy/sdk). " +
		"Generate idiomatic Go code with proper Decimal handling via shopspring/decimal.\n\n" +
		"Return ONLY the Go code inside ```go ... ``` fence."

	userMsg := fmt.Sprintf("Translate this trading EA/indicator to Go:\n```\n%s\n```", code)
	messages := systemai.BuildChatMessages(sysPrompt, userMsg, nil)

	// Retry up to 2 times for transient LLM failures (DeepSeek JSON truncation).
	var result string
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		result, lastErr = s.systemSvc.ChatCompletion(ctx, uid, messages)
		if lastErr == nil {
			break
		}
		s.log.Warn("CodeAssist: TransformCode LLM call failed, retrying",
			zap.Int("attempt", attempt+1), zap.Error(lastErr))
	}
	if lastErr != nil {
		s.log.Warn("CodeAssist: TransformCode LLM call failed after retries", zap.Error(lastErr))
		if errors.Is(lastErr, systemai.ErrInsufficientBalance) {
			return nil, systemai.WrapAIError(lastErr)
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("%s", systemai.FriendlyError(lastErr)))
	}

	// Extract Go code from response (strip markdown fences if present).
	goCode := extractCodeBlock(result)
	if goCode == "" {
		goCode = result
	}
	if goCode == "" {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("LLM returned empty response"))
	}

	return connect.NewResponse(&antv1.TransformCodeResponse{
		TargetCode:    goCode,
		Explanation:   result[:min(len(result), 200)] + "...",
		DetectedLang:  detectedLang,
	}), nil
}

func extractCodeBlock(s string) string {
	// Find ```go ... ``` or ``` ... ``` block
	start := 0
	for {
		i := -1
		for j := start; j < len(s)-6; j++ {
			if s[j] == '`' && s[j+1] == '`' && s[j+2] == '`' {
				i = j
				break
			}
		}
		if i < 0 {
			return ""
		}
		end := i + 3
		for end < len(s)-2 && !(s[end] == '`' && s[end+1] == '`' && s[end+2] == '`') {
			end++
		}
		if end+3 <= len(s) {
			code := s[i+3 : end]
			// Skip language tag if present (go, mql4, mql5, etc.)
			if nl := strings.Index(code, "\n"); nl >= 0 && nl < 20 {
				tag := strings.TrimSpace(code[:nl])
				if tag == "go" || tag == "golang" || tag == "mql4" || tag == "mql5" || tag == "mql" {
					code = code[nl+1:]
				}
			}
			return strings.TrimSpace(code)
		}
		start = end + 3
	}
}
