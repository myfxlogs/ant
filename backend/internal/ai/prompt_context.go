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
		pc.SystemPrompt = discussPrompt(input.Code)
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

## Strategy Code Specification
- Must define a run(context) function
- Return a trade signal dict: {'signal': 'buy'|'sell'|'hold', 'volume': 1.0, ...}
- Tunable parameters must use # @param annotations

## Prohibited
- Do NOT use eval(), exec(), compile()
- Do NOT import os, subprocess, socket
- Do NOT use open() for file operations
- Output ONLY Python code — no explanations or markdown fences`
}

func revisePrompt() string {
	return `You are a professional quantitative trading strategy engineer.
Revise the following Python strategy code according to the user's instruction.

## Revision Rules
- Keep the existing code structure and style
- Only modify what the instruction asks for
- Preserve all existing # @param annotations

## Output Rules
- Output the COMPLETE revised code
- Do NOT include explanations or markdown fences
- The first character must be import, def, class, or #`
}

func repairPrompt(errors []string) string {
	errList := ""
	for _, e := range errors {
		errList += "- " + e + "\n"
	}
	if errList == "" {
		errList = "- (errors provided in user message)\n"
	}
	return `You are a trading strategy CODE REPAIR TOOL. Your ONLY job is to fix errors.

## STRICT OUTPUT RULES — VIOLATION WILL BREAK THE PIPELINE
1. Output ONLY the complete, corrected Python code
2. Do NOT include ANY explanatory text
3. Do NOT say "here is the fixed code" or similar
4. Do NOT wrap code in markdown fences (` + "```python ```" + `)
5. Do NOT analyze the error causes
6. Do NOT give suggestions or tips
7. If you cannot fix, output the original code with # FIXME: <reason> comments

## Errors to Fix
` + errList + `

## CRITICAL REMINDER
Your response will be written directly to a strategy file and executed.
If it contains non-code text, the pipeline will FAIL.
Output MUST start with import, def, class, #, or a blank line.`
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
