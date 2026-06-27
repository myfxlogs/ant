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
Generate a complete Go trading strategy based on the user's description.
All strategies MUST implement the sdk.Strategy interface (OnInit/OnBar/OnDeinit).

## ⛔ IRON RULES — violating ANY of these = code is REJECTED ⛔

### Rule 1: EVERY configurable value MUST be read via ctx.Param() in OnInit
` + "```go" + `
import (
    "anttrader/strategy/sdk"
    "github.com/shopspring/decimal"
)

type MyStrategy struct {
    fast      int
    slow      int
    entryPct  float64
    slPct     float64
    tpPct     float64
}

func (s *MyStrategy) OnInit(ctx sdk.Context) error {
    s.fast = ctx.Param("fast_period", 20)
    s.slow = ctx.Param("slow_period", 50)
    s.entryPct = ctx.Param("entryPct", 0.25)
    s.slPct = ctx.Param("stopLossPct", 0.02)
    s.tpPct = ctx.Param("takeProfitPct", 0.04)
    return nil
}
` + "```" + `

### Rule 2: Query positions via ctx.Broker().Positions() — NOT a position dict
` + "```go" + `
func (s *MyStrategy) OnBar(ctx sdk.Context, timeframe string) (*sdk.Signal, error) {
    positions := ctx.Broker().Positions(0) // 0 = all magic numbers
    for _, pos := range positions {
        if pos.Side == sdk.SideBuy {
            // pos.Ticket, pos.Volume, pos.OpenPrice, pos.SL, pos.TP
        }
    }
    return nil, nil
}
` + "```" + `

### Rule 3: Stop-loss & take-profit MUST be set on EVERY signal
` + "```go" + `
    price := bars.Close(0)
    sl := price.Mul(decimal.NewFromFloat(1 - s.slPct))
    tp := price.Mul(decimal.NewFromFloat(1 + s.tpPct))
    return &sdk.Signal{
        Action:      sdk.ActionBuy,
        Symbol:      ctx.Symbol(),
        Volume:      volume,
        Price:       price,
        StopLoss:    sl,
        TakeProfit:  tp,
    }, nil
    // ❌ NEVER: StopLoss: decimal.Zero, TakeProfit: decimal.Zero
` + "```" + `

### Rule 4: Position sizing via ctx.Broker().AccountInfo().Balance
` + "```go" + `
    balance := ctx.Broker().AccountInfo().Balance
    price := bars.Close(0)
    volume := balance.Mul(decimal.NewFromFloat(s.entryPct)).Div(price)
` + "```" + `

### Rule 5: ALL monetary values MUST use decimal.Decimal, NEVER float64
### Rule 6: MUST implement sdk.Strategy interface (OnInit/OnBar/OnDeinit)
### Rule 7: Use shopspring/decimal for all price/volume calculations

## Complete minimal strategy skeleton
` + "```go" + `
import (
    "anttrader/strategy/sdk"
    "github.com/shopspring/decimal"
)

type MyStrategy struct {
    fast int
    slow int
}

func (s *MyStrategy) OnInit(ctx sdk.Context) error {
    s.fast = ctx.Param("fast_period", 20)
    s.slow = ctx.Param("slow_period", 50)
    return nil
}

func (s *MyStrategy) OnBar(ctx sdk.Context, timeframe string) (*sdk.Signal, error) {
    bars := ctx.Bars(timeframe)
    if bars.Len() < s.slow {
        return nil, nil
    }
    // ... strategy logic ...
    return nil, nil
}

func (s *MyStrategy) OnDeinit(ctx sdk.Context, reason string) error {
    return nil
}
` + "```" + `

## Output: ONLY Go code. No markdown fences. No explanations.`
}

func revisePrompt() string {
	return `You are a trading strategy engineer. Revise the Go code per the user's instruction.

	## RULES — follow exactly
	1. Make ONLY the changes the user asked for — preserve everything else untouched
	2. Output ONLY the complete Go code — NO markdown, NO explanations
	3. Code MUST implement sdk.Strategy interface (OnInit/OnBar/OnDeinit)
	4. ALL monetary values MUST use decimal.Decimal, NEVER float64

	## OUTPUT: The complete revised Go code, starting with import/type/func — nothing else.`
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
	1. Output ONLY the complete corrected Go code — NO markdown, NO explanations
	2. Start directly with import/type/func — your output IS the strategy file
	3. Preserve ALL existing logic, parameters, and comments — only fix the errors
	4. Do NOT rename variables, restructure code, or "improve" anything not in the error list
	5. Code MUST implement sdk.Strategy interface (OnInit/OnBar/OnDeinit)
	6. If an error is unclear, add // FIXME: <reason> at that line — do NOT guess

## VERIFY BEFORE OUTPUT
- Did I fix EVERY error in the list above?
- Did I preserve the original strategy logic unchanged?
- Did I introduce any new undefined variables or syntax errors?

## Errors to Fix (fix ALL at once)
` + errList + `

## OUTPUT ONLY THE COMPLETE CORRECTED GO CODE NOW`
}

func discussPrompt(code string) string {
	return `You are an experienced quantitative trading strategy analyst.
The user is developing a trading strategy and needs your professional opinion.

## Current Strategy Code
` + "```go\n" + code + "\n```" + `

Provide a concise, professional response to the user's question. Be direct — no pleasantries.
If the user asks "is this correct" or "are there issues", check: entry logic, exit logic,
stop-loss/take-profit, position sizing, and edge case handling.`
}

func buildReviseUserMessage(input BuildContextInput) string {
	msg := "Instruction: " + input.Message
	if input.BacktestSummary != "" {
		msg += "\n\n【Current Backtest Results】\n" + input.BacktestSummary
	}
	msg += "\n\nCode:\n```go\n" + input.Code + "\n```"
	return msg
}

func buildRepairUserMessage(code, message string) string {
	return "## Current Code\n```go\n" + code + "\n```\n\n## Error Information\n" + message
}
