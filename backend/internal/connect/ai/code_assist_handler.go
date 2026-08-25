package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/ai"
	systemai "alphaforge/internal/service/systemai"
	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go"
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

	// ── MQL path: compile via Bytecode VM, extract params from AST ──
	if sdk.IsMQL(code) {
		return s.validateMQL(ctx, code)
	}

	// ── Python subset path: compile via Bytecode VM (same pipeline) ──
	if sdk.IsPython(code) {
		return s.validatePython(ctx, code)
	}

	// ── Go path: structural checks (legacy generated Go strategy) ──
	_, missingSigs := ai.HasRequiredSignature(code)
	structWarns := ai.StructuralWarnings(code)

	var errors []string
	var warnings []string

	errors = append(errors, missingSigs...)
	warnings = append(warnings, structWarns...)

	parameterEntries := ExtractParams(code)

	valid := len(errors) == 0

	return connect.NewResponse(&antv1.ValidateStrategyExtendedResponse{
		Valid:            valid,
		Errors:           errors,
		Warnings:         warnings,
		ParameterEntries: parameterEntries,
	}), nil
}

// validateMQL compiles MQL source via the Bytecode VM pipeline and extracts parameters.
func (s *CodeAssistServer) validateMQL(_ context.Context, code string) (*connect.Response[antv1.ValidateStrategyExtendedResponse], error) {
	runner, err := mql2go.CompileMQL(code)
	if err != nil {
		return connect.NewResponse(&antv1.ValidateStrategyExtendedResponse{
			Valid:  false,
			Errors: []string{fmt.Sprintf("MQL compile failed: %v", err)},
		}), nil
	}

	paramInfos := mql2go.ExtractParamInfos(runner.Bytecode())
	params := make([]*antv1.RequiredParamSpec, 0, len(paramInfos))
	var entries []*antv1.ParameterEntry
	for _, p := range paramInfos {
		pType := mqlTypeToProtoType(p.Type)
		params = append(params, &antv1.RequiredParamSpec{
			Key:          p.Name,
			Required:     false,
			Type:         pType,
			DefaultValue: p.Default,
		})
		entries = append(entries, &antv1.ParameterEntry{Name: p.Name, Type: pType, Default: p.Default})
	}

	// Parse @param annotations for sweep dimensions and @strategy directives from comments.
	annotParams := ai.ExtractParamAnnotations(code)
	var sweepDims []*antv1.SweepDimension
	for _, ap := range annotParams {
		sweepDims = append(sweepDims, &antv1.SweepDimension{
			Key:      ap.Name,
			Type:     ai.ParamTypeString(ap.Default),
			Default:  ap.Default,
			Min:      ap.Min,
			Max:      ap.Max,
			Step:     ap.Step,
			HasRange: ap.HasRange,
		})
	}
	var strategyDirs []*antv1.StrategyDirective
	for _, d := range ai.ExtractStrategyDirectives(code) {
		strategyDirs = append(strategyDirs, &antv1.StrategyDirective{
			Key:   d.Key,
			Value: d.Value,
		})
	}

	return connect.NewResponse(&antv1.ValidateStrategyExtendedResponse{
		Valid:              true,
		Parameters:         params,
		ParameterEntries:   entries,
		SweepDimensions:    sweepDims,
		StrategyDirectives: strategyDirs,
	}), nil
}

// validatePython compiles Python subset source via the Bytecode VM pipeline and extracts parameters.
func (s *CodeAssistServer) validatePython(_ context.Context, code string) (*connect.Response[antv1.ValidateStrategyExtendedResponse], error) {
	runner, err := mql2go.CompilePython(code)
	if err != nil {
		return connect.NewResponse(&antv1.ValidateStrategyExtendedResponse{
			Valid:  false,
			Errors: []string{fmt.Sprintf("Python compile failed: %v", err)},
		}), nil
	}

	paramInfos := mql2go.ExtractParamInfos(runner.Bytecode())
	params := make([]*antv1.RequiredParamSpec, 0, len(paramInfos))
	var entries []*antv1.ParameterEntry
	for _, p := range paramInfos {
		pType := mqlTypeToProtoType(p.Type)
		params = append(params, &antv1.RequiredParamSpec{
			Key:          p.Name,
			Required:     false,
			Type:         pType,
			DefaultValue: p.Default,
		})
		entries = append(entries, &antv1.ParameterEntry{Name: p.Name, Type: pType, Default: p.Default})
	}

	annotParams := ai.ExtractParamAnnotations(code)
	var sweepDims []*antv1.SweepDimension
	for _, ap := range annotParams {
		sweepDims = append(sweepDims, &antv1.SweepDimension{
			Key:      ap.Name,
			Type:     ai.ParamTypeString(ap.Default),
			Default:  ap.Default,
			Min:      ap.Min,
			Max:      ap.Max,
			Step:     ap.Step,
			HasRange: ap.HasRange,
		})
	}
	var strategyDirs []*antv1.StrategyDirective
	for _, d := range ai.ExtractStrategyDirectives(code) {
		strategyDirs = append(strategyDirs, &antv1.StrategyDirective{
			Key:   d.Key,
			Value: d.Value,
		})
	}

	return connect.NewResponse(&antv1.ValidateStrategyExtendedResponse{
		Valid:              true,
		Parameters:         params,
		ParameterEntries:   entries,
		SweepDimensions:    sweepDims,
		StrategyDirectives: strategyDirs,
	}), nil
}

// mqlTypeToProtoType maps MQL type names to proto RequiredParamSpec type strings.
func mqlTypeToProtoType(mqlType string) string {
	switch mqlType {
	case "int", "long", "uint", "ulong", "short", "ushort", "char", "uchar":
		return "int"
	case "double", "float":
		return "float"
	case "bool":
		return "bool"
	default:
		return "str"
	}
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

// paramPattern matches ctx.Param*("name", default) calls.
// Group 1: method suffix (empty, Int, Decimal, Bool, String)
// Group 2: parameter name
// Group 3: default value expression
var paramPattern = regexp.MustCompile(`ctx\.Param(Int|Decimal|Bool|String)?\s*\(\s*"([^"]+)"\s*,\s*([^)]+)\)`)

// paramType maps Param* suffix to proto type string.
var paramType = map[string]string{
	"":        "str",
	"Int":     "int",
	"Decimal": "float",
	"Bool":    "bool",
	"String":  "str",
}

// ExtractParams statically extracts ctx.Param*() calls and returns structured entries.
