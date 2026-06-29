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
	"anttrader/internal/ai"
	systemai "anttrader/internal/service/systemai"
)

const maxCodeLen = 100 * 1024
const maxInstrLen = 4 * 1024

// CodeAssistServer implements ant.v1.CodeAssistServiceHandler.
type CodeAssistServer struct {
	systemSvc *systemai.Service
	session   *ai.ConversationSession
	log       *zap.Logger
}

var _ antv1c.CodeAssistServiceHandler = (*CodeAssistServer)(nil)

func NewCodeAssistServer(systemSvc *systemai.Service, session *ai.ConversationSession, log *zap.Logger) *CodeAssistServer {
	return &CodeAssistServer{systemSvc: systemSvc, session: session, log: log}
}

// protoHistoryToChat converts proto CodeChatMessage list to systemai ChatMessage list.
func protoHistoryToChat(protoMsgs []*antv1.CodeChatMessage) []systemai.ChatMessage {
	out := make([]systemai.ChatMessage, len(protoMsgs))
	for i, m := range protoMsgs {
		out[i] = systemai.ChatMessage{Role: m.Role, Content: m.Content}
	}
	return out
}

func (s *CodeAssistServer) ValidateStrategyExtended(ctx context.Context, req *connect.Request[antv1.ValidateStrategyExtendedRequest]) (*connect.Response[antv1.ValidateStrategyExtendedResponse], error) {
	code := req.Msg.Code
	if len(code) > maxCodeLen {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("code too large: %d bytes", len(code)))
	}

	// ── Fast path: structural checks (no AI call) ──
	_, missingSigs := ai.HasRequiredSignature(code)
	structWarns := ai.StructuralWarnings(code)

	var errors []string
	var warnings []string

	// Missing signature is always an error
	for _, m := range missingSigs {
		errors = append(errors, m)
	}
	// Structural quality warnings
	warnings = append(warnings, structWarns...)

	var parametersJson string

	valid := len(errors) == 0

	return connect.NewResponse(&antv1.ValidateStrategyExtendedResponse{
		Valid:          valid,
		Errors:         errors,
		Warnings:       warnings,
		ParametersJson: parametersJson,
	}), nil
}

func (s *CodeAssistServer) TranslateParamLabels(ctx context.Context, req *connect.Request[antv1.TranslateParamLabelsRequest]) (*connect.Response[antv1.TranslateParamLabelsResponse], error) {
	names := req.Msg.ParamNames
	if len(names) == 0 {
		return connect.NewResponse(&antv1.TranslateParamLabelsResponse{}), nil
	}

	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	namesJSON, _ := json.Marshal(names)
	prompt := fmt.Sprintf(translateParamLabelsPrompt, string(namesJSON))
	messages := systemai.BuildChatMessages("You are a financial translator. Respond with ONLY valid JSON, no markdown fences.", prompt, nil)
	result, err := s.systemSvc.ChatCompletion(ctx, uid, messages)
	if err != nil {
		s.log.Warn("CodeAssist: TranslateParamLabels failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("translation failed: %w", err))
	}

	// Parse AI JSON response into translation map.
	var parsed map[string]map[string]string // locale → param_name → translation
	if err := json.Unmarshal([]byte(strings.TrimSpace(result)), &parsed); err != nil {
		s.log.Warn("CodeAssist: TranslateParamLabels JSON parse failed", zap.Error(err), zap.String("raw", result))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("translation parse failed"))
	}

	// Convert to proto type.
	translations := make(map[string]*antv1.ParamLabelMap)
	for locale, labels := range parsed {
		translations[locale] = &antv1.ParamLabelMap{Labels: labels}
	}
	return connect.NewResponse(&antv1.TranslateParamLabelsResponse{Translations: translations}), nil
}

const translateParamLabelsPrompt = `Translate these trading strategy parameter names into 5 languages.
Parameters: %s

Return a JSON object keyed by locale code. Each locale contains a map from the original parameter name to the translated label.

Supported locales: "en", "zh-cn", "zh-tw", "ja", "vi"

Rules:
- Preserve financial/quant terminology precision
- Keep translations short (1-4 words)
- "en" labels should use standard trading vocabulary (e.g. "Lot Size", not "Hand Count")
- If a name is already in the target language, keep it unchanged
- Do NOT translate proper nouns or magic numbers

Example: For parameter "翻倍", return {"en": "Multiplier", "zh-tw": "翻倍", "ja": "倍率", "vi": "Hệ số nhân"}`

func buildValidationPrompt() string {
	return "You are a trading strategy code validator. " +
		"Review the following Go strategy code and identify issues. " +
		"The code MUST use the AntTrader Go Strategy SDK (anttrader/strategy/sdk):\n" +
		"- Implements sdk.Strategy interface: OnInit/OnBar/OnDeinit.\n" +
		"- Orders via ctx.Broker().OrderSend(sdk.OrderRequest{...}).\n" +
		"- Prices via bars.Close(0) (index 0 = most recent bar).\n" +
		"- Indicators via ctx.Indicators().MA/RSI/EMA/ATR/Bands/MACD.\n" +
		"- Parameters via ctx.Param(name, default).\n" +
		"- All monetary values use decimal.Decimal, never float64.\n\n" +
		"STRICT RULES that make code INVALID:\n" +
		"- Missing OnInit, OnBar, or OnDeinit method.\n" +
		"- Using float64 for price/volume calculations.\n\n" +
		"Return a JSON object with fields: valid (bool), errors (string array), warnings (string array), " +
		"parameters (array of objects with keys: key (str), required (bool), type (str: int|float|str|bool), " +
		"default_value (str, optional), suggested_value (str, optional)). " +
		"Extract all ctx.Param() calls from OnInit() into the parameters array. " +
		"Check for: missing stop-loss, missing take-profit, position sizing, error handling, " +
		"indicator usage correctness, decimal.Decimal usage for prices, and data boundary handling. " +
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

// extractCodeFromRepair attempts to salvage Go code from an LLM response
// that may contain explanatory text (3-tier extraction).
func extractCodeFromRepair(raw string) string {
	// Tier 1: extract from ```go ... ``` fence
	if code := extractFencedCode(raw, "go"); code != "" {
		return code
	}
	// Tier 2: heuristic — find lines starting with import/package/func/type
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
		if strings.HasPrefix(trimmed, "package ") ||
			strings.HasPrefix(trimmed, "import ") ||
			strings.HasPrefix(trimmed, "import(") ||
			strings.HasPrefix(trimmed, "func ") ||
			strings.HasPrefix(trimmed, "type ") ||
			strings.HasPrefix(trimmed, "//") {
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
