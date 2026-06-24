// Package ai — prompt_context.go
// PromptContext builds mode-specific system prompts for AI code interactions.
// Pure functions, no side effects, no state. Replaces the ad-hoc
// buildCodeAssistPrompt() in code_assist_handler.go.

package ai

import "strings"

// InteractionMode classifies user intent for AI code assistance.
type InteractionMode int

const (
	ModeGenerate InteractionMode = iota // no code exists, create from scratch
	ModeRevise                          // modify existing code
	ModeRepair                          // fix validation/runtime errors
	ModeDiscuss                         // ask questions about the code
)

// PromptContext holds all context needed to build mode-specific prompts.
type PromptContext struct {
	Mode             InteractionMode
	SystemPrompt     string
	UserMessage      string
	Code             string
	Symbol           string
	Timeframe        string
	BacktestSummary  string
	ValidationErrors []string
}

// BuildContextInput is the parameter object for BuildContext.
// Struct encapsulation keeps the function signature within the 5-parameter limit.
type BuildContextInput struct {
	Code             string
	Message          string
	Symbol           string
	Timeframe        string
	BacktestSummary  string
	ValidationErrors []string
	Locale           string // user UI language (e.g. en, zh, zh-TW, ja, vi); blank = no directive
}

// localeDirective returns a language instruction for AI prose responses.
// Delegates to the centralized ai.LocaleDirective.
func localeDirective(locale string) string {
	return LocaleDirective(locale)
}

// BuildContext analyzes code + message and returns the appropriate PromptContext.
func BuildContext(input BuildContextInput) *PromptContext {
	mode := classifyIntent(input.Code, input.Message)

	pc := &PromptContext{
		Mode:             mode,
		Code:             input.Code,
		Symbol:           input.Symbol,
		Timeframe:        input.Timeframe,
		BacktestSummary:  input.BacktestSummary,
		ValidationErrors: input.ValidationErrors,
	}

	switch mode {
	case ModeGenerate:
		pc.SystemPrompt = generatePrompt()
		pc.UserMessage = input.Message
	case ModeRevise:
		pc.SystemPrompt = revisePrompt()
		pc.UserMessage = buildReviseUserMessage(input)
	case ModeRepair:
		pc.SystemPrompt = repairPrompt(input.ValidationErrors)
		pc.UserMessage = buildRepairUserMessage(input.Code, input.Message)
	case ModeDiscuss:
		pc.SystemPrompt = discussPrompt(input.Code) + localeDirective(input.Locale)
		pc.UserMessage = input.Message
	}

	return pc
}

// classifyIntent determines the interaction mode from code + message.
func classifyIntent(code, message string) InteractionMode {
	if strings.TrimSpace(code) == "" {
		return ModeGenerate
	}
	lower := strings.ToLower(message)

	// Repair: error-related keywords (highest priority)
	repairKw := []string{
		"报错", "error", "错误", "traceback", "缺少参数", "missing",
		"验证失败", "syntax error", "syntaxerror", "undefined", "未定义",
		"缺少 required", "参数不足", "attributeerror", "typeerror",
	}
	for _, kw := range repairKw {
		if strings.Contains(lower, kw) {
			return ModeRepair
		}
	}

	// Discuss: question/analysis keywords
	discussKw := []string{
		"为什么", "什么意思", "怎么样", "对吗", "分析",
		"解释", "what", "why", "how", "explain", "对不对",
	}
	for _, kw := range discussKw {
		if strings.Contains(lower, kw) {
			return ModeDiscuss
		}
	}

	// Default: revise
	return ModeRevise
}

func generatePrompt() string {
	return `You are a professional quantitative trading strategy engineer.
Generate a complete Python trading strategy based on the user's description.

## ⛔ IRON RULES — violating ANY of these = code is REJECTED ⛔

### Rule 1: Every @param and @strategy MUST read from context.get()
` + "```python" + `
# @param fast_period 20 range=10:50:10
# @param slow_period 50 range=30:80:10
# @strategy entryPct 0.25
# @strategy stopLossPct 0.02
# @strategy takeProfitPct 0.04

def run(context):
    fast_period = int(context.get('fast_period', 20))    # ✅ MUST read from context
    slow_period = int(context.get('slow_period', 50))    # ✅ MUST read from context
    entry_pct   = float(context.get('entryPct', 0.25))   # ✅ MUST read from context
    sl_pct      = float(context.get('stopLossPct', 0.02))
    tp_pct      = float(context.get('takeProfitPct', 0.04))

    # ❌ NEVER do this: ema20 = calc_ema(close, 20)  ← hardcoded 20 instead of fast_period
    # ❌ NEVER do this: sl_pct = 0.02                ← hardcoded instead of context.get()
` + "```" + `

### Rule 2: Use CORRECT position dict keys — NEVER invent fake field names
The engine injects position as a dict with these EXACT keys:
` + "```python" + `
position = context.get('position')  # None means no position
if position is not None:
    side   = position['side']         # ✅ 'buy'=long, 'sell'=short
    volume = position['volume']       # ✅ lot size
    price  = position['open_price']   # ✅ entry price
    sl     = position.get('sl', 0)    # ✅ current stop-loss
    tp     = position.get('tp', 0)    # ✅ current take-profit
    # ❌ NEVER use: position['direction'], position['type'], position['price']
` + "```" + `

### Rule 3: Stop-loss & take-profit MUST be set on EVERY bar when holding a position
` + "```python" + `
    if position is not None:
        entry_price = position['open_price']
        if position['side'] == 'buy':
            stop_loss   = entry_price * (1 - sl_pct)    # ✅ every bar
            take_profit = entry_price * (1 + tp_pct)    # ✅ every bar
        else:
            stop_loss   = entry_price * (1 + sl_pct)
            take_profit = entry_price * (1 - tp_pct)
    # ❌ NEVER: stop_loss=0.0, take_profit=0.0 when position exists
` + "```" + `

### Rule 4: Position sizing MUST use initial_balance, NOT current balance
` + "```python" + `
    initial_balance = float(context.get('initial_balance', 10000.0))
    volume_ordered = (initial_balance * entry_pct) / close[-1]   # ✅
    # ❌ NEVER: volume = (context.get('balance') * entry_pct) / close[-1]
` + "```" + `

### Rule 5: ALL variables must be defined at function start, before any return
### Rule 6: Use descriptive method names — underscore-prefixed helpers (_count_orders etc) are REJECTED
### Rule 7: np and math are pre-injected — do NOT import them
### Rule 8: MUST use SDK class format. Inherit from StrategyBase. At least one lifecycle hook (on_init/on_bar/on_tick etc). def run(context) is REJECTED.

## Output: ONLY Python code. No markdown fences. No explanations.`
}

func revisePrompt() string {
	return `You are a trading strategy engineer. Revise the Python code per the user's instruction.

	## RULES — follow exactly
	1. Make ONLY the changes the user asked for — preserve everything else untouched
	2. Output ONLY the complete Python code — NO markdown, NO explanations
	3. Code MUST use SDK class format (class X(StrategyBase)) — NOT def run(context)
	4. Use descriptive method names — avoid underscore-prefixed helpers (_helper)
	5. ALL monetary values MUST use Decimal(str(x)), NEVER float

	## OUTPUT: The complete revised code, starting with import/class/# — nothing else.`
	}

func repairPrompt(errors []string) string {
	errList := ""
	for _, e := range errors {
		errList += "- " + e + "\n"
	}
	if errList == "" {
		errList = "- (errors provided in user message)\n"
	}
	return `You are a trading strategy CODE REPAIR EXPERT. Fix ONLY the listed errors — do NOT change anything else.

## ⛔ CRITICAL RULES
	1. Output ONLY the complete corrected Python code — NO markdown, NO explanations
	2. Start directly with import/class/# — your output IS the strategy file
	3. Preserve ALL existing logic, parameters, and comments — only fix the errors
	4. Do NOT rename variables, restructure code, or "improve" anything not in the error list
	5. Code MUST use SDK class format (class X(StrategyBase)) — NOT def run(context)
	6. Avoid underscore-prefixed method names (_helper) — use descriptive names instead
	7. If an error is unclear, add # FIXME: <reason> at that line — do NOT guess

## VERIFY BEFORE OUTPUT
- Does the fix address the exact error listed?
- Did I preserve the original strategy logic unchanged?
- Did I introduce any new undefined variables or syntax errors?

## Errors to Fix (one by one)
` + errList + `

## OUTPUT ONLY THE COMPLETE CORRECTED CODE NOW`
}

func discussPrompt(code string) string {
	return `You are an experienced quantitative trading strategy analyst.
The user is developing a trading strategy and needs your professional opinion.

## Current Strategy Code
` + "```python\n" + code + "\n```" + `

Provide a concise, professional response to the user's question. Be direct — no pleasantries.
If the user asks "is this correct" or "are there issues", check: entry logic, exit logic,
stop-loss/take-profit, position sizing, and edge case handling.`
}

func buildReviseUserMessage(input BuildContextInput) string {
	msg := "Instruction: " + input.Message
	if input.BacktestSummary != "" {
		msg += "\n\n【Current Backtest Results】\n" + input.BacktestSummary
	}
	msg += "\n\nCode:\n```python\n" + input.Code + "\n```"
	return msg
}

func buildRepairUserMessage(code, message string) string {
	return "## Current Code\n```python\n" + code + "\n```\n\n## Error Information\n" + message
}
