package ai

import (
	"context"
	"encoding/json"
	"errors"
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

	pc := ai.BuildContext(ai.BuildContextInput{Code: code, Message: instruction, Locale: req.Msg.Locale})
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

	pc := ai.BuildContext(ai.BuildContextInput{Code: code, Message: instruction, Locale: req.Msg.Locale})
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
		"Python strategy for the AntTrader platform.\n\n" +
		"AntTrader Strategy SDK API (the ONLY valid API — use EXACTLY these signatures):\n\n" +
		"## Lifecycle (inherit from StrategyBase)\n" +
		"- `def on_init(self) -> None:` — replaces OnInit(). Register params, set timer.\n" +
		"- `def on_tick(self) -> None:` — replaces OnTick(). Primary entry for tick-driven EAs.\n" +
		"- `def on_bar(self, timeframe: str) -> None:` — replaces OnCalculate(). New bar closed.\n" +
		"- `def on_timer(self) -> None:` — replaces OnTimer(). Requires ctx.set_timer(seconds).\n" +
		"- `def on_trade(self) -> None:` — replaces OnTrade(). After any trade event.\n" +
		"- `def on_deinit(self, reason: str) -> None:` — replaces OnDeinit(). Cleanup.\n\n" +
		"## Order Entry (via self.broker)\n" +
		"- `self.broker.order_send(OrderRequest(symbol=..., type=OrderType.BUY, volume=Decimal(str(lot)), ...))`\n" +
		"  OrderType values: BUY, SELL, BUY_LIMIT, SELL_LIMIT, BUY_STOP, SELL_STOP,\n" +
		"  BUY_STOP_LIMIT, SELL_STOP_LIMIT.\n" +
		"  Optional fields: price (Decimal, omit for market orders), sl, tp (Decimal or None),\n" +
		"  deviation (int, slippage in points), magic (int), comment (str),\n" +
		"  type_filling (TypeFilling.FOK/IOC/RETURN), stop_limit_price (Decimal, only for *_STOP_LIMIT).\n" +
		"  Returns OrderResult with retcode, ticket, price, volume.\n" +
		"- `self.broker.position_close(ticket, volume=None)` — close position (None=full, Decimal=partial).\n" +
		"- `self.broker.position_modify(ticket, sl=None, tp=None)` — modify SL/TP (Decimal or None).\n" +
		"- `self.broker.order_delete(ticket)` — cancel a pending order.\n\n" +
		"## Position & Order Query (via self.broker)\n" +
		"- `self.broker.positions(symbol=None, magic=None) -> list[Position]` — open positions.\n" +
		"  Position fields: ticket, symbol, side (PositionSide.BUY/SELL), volume (Decimal),\n" +
		"  open_price (Decimal), sl, tp, profit, swap, magic, comment, open_time_ms.\n" +
		"- `self.broker.orders(symbol=None, magic=None) -> list[PendingOrder]` — pending orders.\n" +
		"  PendingOrder fields: ticket, symbol, type (OrderType), volume, price, sl, tp, magic.\n" +
		"- `self.broker.account() -> AccountInfo` — balance, equity, margin, free_margin, margin_level,\n" +
		"  leverage, currency, mode (AccountMode.NETTING/HEDGING). All amounts are Decimal.\n" +
		"- `self.broker.symbol_info(symbol) -> SymbolInfo` — digits, point, tick_size, tick_value,\n" +
		"  contract_size, volume_min/max/step, stops_level, freeze_level, swap_long/short, margin_rate.\n" +
		"- `self.broker.server_time() -> int` — unix_ms.\n\n" +
		"## Price Data (via self.ctx, MQL reverse indexing: [0]=current, [1]=previous)\n" +
		"- `bars = self.ctx.bars(timeframe=None)` — returns Bars. None = primary timeframe.\n" +
		"- `bars.close[0]`, `bars.open[0]`, `bars.high[0]`, `bars.low[0]`, `bars.volume[0]`, `bars.time[0]`.\n" +
		"- `bars.total()` — number of available bars.\n\n" +
		"## Indicators (via self.indicators, shift=0 = current bar, all return float)\n" +
		"- `self.indicators.ma(period=14, shift=0, method='sma')` — methods: sma/ema/smma/lwma.\n" +
		"- `self.indicators.ema(period=14, shift=0)`\n" +
		"- `self.indicators.rsi(period=14, shift=0)`\n" +
		"- `self.indicators.bands(period=20, deviation=2.0, shift=0) -> (upper, middle, lower)`\n" +
		"- `self.indicators.macd(fast=12, slow=26, signal=9, shift=0) -> (macd, signal, histogram)`\n" +
		"- `self.indicators.atr(period=14, shift=0)`\n" +
		"- `self.indicators.stochastic(k_period=5, d_period=3, shift=0) -> (k, d)`\n" +
		"- `self.indicators.cci(period=14, shift=0)`\n" +
		"- `self.indicators.i_custom(name, params=[], buffer=0, shift=0)` — custom indicator.\n\n" +
		"## Parameters & Timer (via self.ctx)\n" +
		"- `self.ctx.param(name, default=None)` — read extern/input parameter (type: object, cast as needed).\n" +
		"- `self.ctx.set_timer(seconds)` — enable periodic on_timer callback (min 1s).\n" +
		"- `self.ctx.kill_timer()` — disable timer.\n\n" +
		"## Critical Rules\n" +
		"1. ALL monetary values (prices, volumes, balances) MUST use Decimal(str(x)), NEVER float.\n" +
		"2. Import from app.sdk: StrategyBase, OrderRequest, OrderType, OrderResult, Position,\n" +
		"   PendingOrder, PositionSide, Retcode, AccountMode, TypeFilling, Decimal.\n" +
		"3. Replace extern/input with self.ctx.param() calls in on_init().\n" +
		"4. MQL OrderSelect loop → `for order in self.broker.orders():` or `for pos in self.broker.positions():`.\n" +
		"5. MQL Close[i] → bars.close[i]; MQL iMA() → self.indicators.ma().\n" +
		"6. NEVER use self.buy(), self.sell(), self.close_all(), self.sma() — these DO NOT EXIST.\n" +
		"7. Return ONLY the Python code inside ```python ... ``` fence.\n" +
		"8. Mark untranslatable MQL (DLL, WebRequest, GUI, FileIO) with `# TRANSPILER-GAP: <reason>`.\n" +
			"9. Use descriptive method names — underscore-prefixed private helpers (_count_orders,\n" +
			"   _send_order) are REJECTED. Use count_orders, send_order instead.\n\n" +
		"## Few-Shot Example\n" +
		"MQL: `int OnInit() { EventSetTimer(60); return INIT_SUCCEEDED; }`\n" +
		"SDK:\n```python\n" +
		"def on_init(self) -> None:\n" +
		"    self.ctx.set_timer(60)\n" +
		"```\n\n" +
		"MQL: `OrderSend(Symbol(), OP_BUY, 0.1, Ask, 3, 0, 0, \"entry\", 12345, 0, clrNONE);`\n" +
		"SDK:\n```python\n" +
		"req = OrderRequest(symbol=self.ctx.symbol, type=OrderType.BUY,\n" +
		"                   volume=Decimal('0.10'), magic=12345, comment='entry')\n" +
		"result = self.broker.order_send(req)\n" +
		"```"

	userMsg := fmt.Sprintf("Translate this trading EA/indicator to Python:\n```\n%s\n```", code)
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

	// Extract Python code from response (strip markdown fences if present).
	python := extractCodeBlock(result)
	if python == "" {
		python = result
	}
	// If still empty, return an error instead of an empty response.
	if python == "" {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("LLM returned empty response"))
	}

	return connect.NewResponse(&antv1.TransformCodeResponse{
		TargetCode:    python,
		Explanation:   result[:min(len(result), 200)] + "...",
		DetectedLang:  detectedLang,
	}), nil
}

func extractCodeBlock(s string) string {
	// Find ```python ... ``` block
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
			// Skip language tag if present
			for len(code) > 0 && (code[0] == 'p' || code[0] == 'P') {
				nl := 0
				for nl < len(code) && code[nl] != '\n' {
					nl++
				}
				if nl < len(code) && nl < 20 {
					code = code[nl+1:]
				} else {
					break
				}
			}
			return strings.TrimSpace(code)
		}
		start = end + 3
	}
}

func (s *CodeAssistServer) ValidateStrategyExtended(ctx context.Context, req *connect.Request[antv1.ValidateStrategyExtendedRequest]) (*connect.Response[antv1.ValidateStrategyExtendedResponse], error) {
	code := req.Msg.Code
	if len(code) > maxCodeLen {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("code too large: %d bytes", len(code)))
	}

	// ── Fast path: compliance scan + Python AST (no AI call) ──
	// Catches sandbox-blocking errors deterministically in <1s.
	scanner := ai.NewCodeComplianceScanner()
	blocks, warns := scanner.Scan(code)
	_, missingSigs := scanner.HasRequiredSignature(code)
	structWarns := scanner.StructuralWarnings(code)

	var errors []string
	var warnings []string

	// Blocking issues → errors
	for _, b := range blocks {
		errors = append(errors, fmt.Sprintf("[%s] line %d: %s", b.RuleName, b.Line, b.Message))
	}
	// Missing signature is always an error
	for _, m := range missingSigs {
		errors = append(errors, m)
	}
	// Non-blocking rule warnings
	for _, w := range warns {
		warnings = append(warnings, fmt.Sprintf("[%s] line %d: %s", w.RuleName, w.Line, w.Message))
	}
	// Structural warnings
	warnings = append(warnings, structWarns...)

	// Run Python ast.parse() for authoritative syntax errors
	if s.pythonStrategyClient != nil {
		pyResp, pyErr := s.pythonStrategyClient.Validate(ctx, connect.NewRequest(&antv1.ValidateStrategyRequest{Code: code}))
		if pyErr == nil && pyResp != nil && pyResp.Msg != nil {
			if len(pyResp.Msg.Errors) > 0 {
				errors = append(pyResp.Msg.Errors, errors...)
			}
			if len(pyResp.Msg.Warnings) > 0 {
				warnings = append(warnings, pyResp.Msg.Warnings...)
			}
		}
	}

	valid := len(errors) == 0

	return connect.NewResponse(&antv1.ValidateStrategyExtendedResponse{
		Valid:    valid,
		Errors:   errors,
		Warnings: warnings,
	}), nil
}

func buildValidationPrompt() string {
	return "You are a trading strategy code validator. " +
		"Review the following Python strategy code and identify issues. " +
		"The code MUST use the AntTrader Strategy SDK (docs/spec/30-strategy-sdk.md):\n" +
		"- Class inherits from StrategyBase with on_init/on_tick/on_bar/on_deinit hooks.\n" +
		"- Orders via self.broker.order_send(OrderRequest(...)).\n" +
		"- Prices via self.ctx.bars().close[0] (MQL reverse indexing).\n" +
		"- Indicators via self.indicators.ma/rsi/ema/atr/bands/macd.\n" +
		"- Parameters via self.ctx.param(name, default).\n" +
		"- All monetary values use Decimal(str(x)), never float.\n\n" +
		"STRICT RULES that make code INVALID:\n" +
		"- Non-SDK format (def run(context), signal={...}) — MUST be class-based.\n" +
		"- Underscore-prefixed method names (_helper, _count_orders) — REJECTED.\n" +
		"- Missing lifecycle hook (on_init/on_bar/on_tick etc).\n\n" +
		"Return a JSON object with fields: valid (bool), errors (string array), warnings (string array), " +
		"parameters (array of objects with keys: key (str), required (bool), type (str: int|float|str|bool), " +
		"default_value (str, optional), suggested_value (str, optional)). " +
		"Extract all self.ctx.param() calls from on_init() into the parameters array. " +
		"Check for: non-SDK format, underscore-prefixed helpers, missing stop-loss, missing take-profit, " +
		"position sizing, error handling, indicator usage correctness, " +
		"Decimal usage for prices, and data boundary handling. " +
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
