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
All strategies MUST use the SDK class-based format (StrategyBase).

## ⛔ IRON RULES — violating ANY of these = code is REJECTED ⛔

### Rule 1: MUST use SDK class format with @param/@strategy annotations
` + "```python" + `
from app.sdk import StrategyBase, OrderRequest, OrderType
from decimal import Decimal

# @param fast_period 20 range=10:50:10
# @param slow_period 50 range=30:80:10
# @strategy entryPct 0.25
# @strategy stopLossPct 0.02
# @strategy takeProfitPct 0.04

class MyStrategy(StrategyBase):
    def on_init(self):
        self.fast = int(self.ctx.param('fast_period', 20))
        self.slow = int(self.ctx.param('slow_period', 50))
        self.entry_pct = float(self.ctx.param('entryPct', 0.25))
        self.sl_pct = float(self.ctx.param('stopLossPct', 0.02))
        self.tp_pct = float(self.ctx.param('takeProfitPct', 0.04))
        # ✅ MUST read params via self.ctx.param() in on_init
        # ❌ NEVER hardcode: fast = 20

    def on_bar(self, timeframe=None):
        bars = self.ctx.bars(timeframe)
        if bars.total() < self.slow:
            return
        # ❌ NEVER: def run(context): return {...}
` + "```" + `

### Rule 2: Query positions via self.broker.positions() — NOT position dict
` + "```python" + `
    def on_bar(self, timeframe=None):
        positions = self.broker.positions()  # returns list[Position]
        for pos in positions:
            if pos.side == PositionSide.BUY:
                # ✅ pos.ticket, pos.volume, pos.open_price, pos.sl, pos.tp
                pass
        # ❌ NEVER: position = context.get('position')
        # ❌ NEVER: position['side'], position['open_price']
` + "```" + `

### Rule 3: Stop-loss & take-profit MUST be set on EVERY order
` + "```python" + `
        price = Decimal(str(bars.close[0]))
        sl = price * Decimal(str(1 - self.sl_pct))
        tp = price * Decimal(str(1 + self.tp_pct))
        self.broker.order_send(OrderRequest(
            symbol=self.ctx.symbol, type=OrderType.BUY,
            volume=volume, sl=sl, tp=tp))
        # ❌ NEVER: sl=Decimal('0'), tp=Decimal('0')
` + "```" + `

### Rule 4: Position sizing via self.broker.account().balance
` + "```python" + `
        balance = float(self.broker.account().balance)
        price = float(bars.close[0])
        volume = Decimal(str(balance * self.entry_pct / price))
        # ✅ Use account().balance for position sizing
        # ❌ NEVER: context.get('initial_balance')
` + "```" + `

### Rule 5: ALL monetary values MUST use Decimal(str(x)), NEVER float
### Rule 6: Use descriptive method names — underscore-prefixed helpers (_count_orders etc) are REJECTED
### Rule 7: Import ONLY from app.sdk — np, math, pandas are pre-injected
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
