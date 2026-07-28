// strategy_prompt.go: system prompt builder for AI strategy code generation.
//
// Builds a structured system prompt that includes:
//   - Template skeleton with parameter slots
//   - User-provided parameters and preferences
//   - Go strategy contract (sdk.Strategy interface)
//   - Coding constraints (forbidden patterns, required annotations)
//   - Phase 3: feedback prompt with backtest metrics + iteration context

package ai

import (
	"fmt"
	"strings"

	"alphaforge/internal/repository"
)

// PromptParams holds all inputs for building the generation prompt.
type PromptParams struct {
	Template     *repository.AIStrategyTemplate
	Message      string // user's natural language description
	Symbol       string
	Timeframe    string
	ParamMap     map[string]string // merged: LLM intent + keyword fallback
	History      string            // previous conversation summary
	Intent       *IntentResult     // LLM-extracted intent (nil for legacy callers)
	StrategyType string            // reserved for future use; Go SDK has a single strategy mode
}

// FeedbackPromptParams holds inputs for building the feedback iteration prompt (Phase 3).
type FeedbackPromptParams struct {
	PreviousCode       string           // previous strategy code
	Metrics            *FeedbackMetrics // last backtest metrics
	FeedbackMessage    string           // user's feedback text
	FeedbackHints      string           // hints from feedback_router
	StrategyType       string           // detected from previous code: "run_dataframe" or "run_context"
	GateFailureReason  string           // e.g. "lookahead", "deflated_sharpe" — empty if N/A
	GateFailureDetails string           // human-readable reason from gate pipeline
	Locale             string           // user locale for prompt language selection (default: zh)
}

// StrategyPromptBuilder constructs system + user prompts for strategy generation.
type StrategyPromptBuilder struct{}

// NewStrategyPromptBuilder creates a new prompt builder.
func NewStrategyPromptBuilder() *StrategyPromptBuilder {
	return &StrategyPromptBuilder{}
}

// BuildSystemPrompt returns the system prompt that instructs the LLM.
func (b *StrategyPromptBuilder) BuildSystemPrompt(p *PromptParams) string {
	var sb strings.Builder

	// Role definition — Claude Code style: plan first, then implement.
	sb.WriteString("你是一位专业的量化交易策略工程师。")
	sb.WriteString("你的任务是根据用户的自然语言描述，生成符合规范的 Go 策略代码。\n\n")
	sb.WriteString("## 工作方式（Claude Code 风格）\n")
	sb.WriteString("1. 先用 1-2 句话简要说明你理解的策略逻辑和执行计划\n")
	sb.WriteString("2. 然后直接生成完整的 Go 代码\n")
	sb.WriteString("3. 不要在代码中留 TODO 或占位符 — 所有参数都给具体值\n")
	sb.WriteString("4. 对于用户未明确的参数，使用合理的默认值，不要询问\n\n")

	// Strategy contract — Go SDK contract.
	sb.WriteString(contractText(p.StrategyType))
	sb.WriteString("\n")

	// LLM-extracted intent context + plan
	if p.Intent != nil && !p.Intent.NeedsClarification {
		if p.Intent.Plan != "" {
			sb.WriteString("## AI 分析的计划\n")
			sb.WriteString(p.Intent.Plan + "\n\n")
		}
		if p.Intent.StrategyFamily != "" && p.Intent.StrategyFamily != "unknown" {
			fmt.Fprintf(&sb, "- 策略类型: %s\n", p.Intent.StrategyFamily)
		}
		if p.Intent.RiskLevel != "" && p.Intent.RiskLevel != "unknown" {
			fmt.Fprintf(&sb, "- 风险偏好: %s\n", p.Intent.RiskLevel)
		}
		if p.Intent.TradeDirection != "" && p.Intent.TradeDirection != "unknown" {
			fmt.Fprintf(&sb, "- 交易方向: %s\n", p.Intent.TradeDirection)
		}
		if p.Intent.HoldingPeriod != "" && p.Intent.HoldingPeriod != "unknown" {
			fmt.Fprintf(&sb, "- 持仓周期: %s\n", p.Intent.HoldingPeriod)
		}
		if len(p.Intent.EntrySignals) > 0 {
			fmt.Fprintf(&sb, "- 入场信号: %s\n", strings.Join(p.Intent.EntrySignals, ", "))
		}
		if len(p.Intent.ExitSignals) > 0 {
			fmt.Fprintf(&sb, "- 离场信号: %s\n", strings.Join(p.Intent.ExitSignals, ", "))
		}
		sb.WriteString("\n")
	}

	// Parameter hints from clarification/fallback
	if len(p.ParamMap) > 0 {
		sb.WriteString("参数偏好：\n")
		for k, v := range p.ParamMap {
			fmt.Fprintf(&sb, "- %s: %s\n", k, v)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// BuildFeedbackPrompt returns system + user prompts for feedback iteration mode (Phase 3).
// Injects previous code, backtest metrics, feedback message, and routing hints into the prompt.
// Auto-detects strategy type from previous code (run_dataframe vs run_context).
func (b *StrategyPromptBuilder) BuildFeedbackPrompt(p *FeedbackPromptParams) (string, string) {
	metricsCtx := ""
	if p.Metrics != nil {
		metricsCtx = p.Metrics.FormatPromptContext()
	}
	hints := p.FeedbackHints
	if hints == "" {
		if NormalizeLocale(p.Locale) == "en" {
			hints = "The user is not satisfied with the backtest results. Please optimize the strategy based on the feedback."
		} else {
			hints = "用户对回测结果不满意，请根据反馈优化策略"
		}
	}
	// Detect strategy type from previous code if not explicitly set.
	st := p.StrategyType
	if st == "" {
		st = DetectCodeStrategyType(p.PreviousCode)
	}
	tmpl := feedbackSystemTemplateZH
	if NormalizeLocale(p.Locale) == "en" {
		tmpl = feedbackSystemTemplateEN
	}
	system := fmt.Sprintf(tmpl,
		contractText(st),
		p.PreviousCode,
		metricsCtx,
		hints,
	)

	// Inject gate evaluation failure context when available.
	if p.GateFailureReason != "" {
		details := p.GateFailureDetails
		if details == "" {
			details = "no additional details"
		}
		if NormalizeLocale(p.Locale) == "en" {
			system += fmt.Sprintf("\n\n## Gate Evaluation Failure\n"+
				"The previous strategy failed at Gate '%s': %s\n"+
				"Please address this issue when fixing the code to ensure the strategy passes this Gate check.",
				p.GateFailureReason, details)
		} else {
			system += fmt.Sprintf("\n\n## Gate 评估失败信息\n"+
				"上一次策略在 Gate '%s' 失败: %s\n"+
				"请在修复代码时针对此问题进行改进，确保策略能通过此次 Gate 检查。",
				p.GateFailureReason, details)
		}
	}

	userMsg := fmt.Sprintf("【用户反馈】%s\n\n请分析回测结果，给出建议，并生成优化后的代码。", p.FeedbackMessage)
	if NormalizeLocale(p.Locale) == "en" {
		userMsg = fmt.Sprintf("[User Feedback] %s\n\nPlease analyze the backtest results, provide suggestions, and generate optimized code.", p.FeedbackMessage)
	}
	return system, userMsg
}

// DetectCodeStrategyType heuristically detects the strategy type from code.
// Go SDK has a single strategy mode; this is kept for API compatibility.
func DetectCodeStrategyType(code string) string {
	return "go_strategy"
}

// BuildUserPrompt returns the user message with template context injected.
func (b *StrategyPromptBuilder) BuildUserPrompt(p *PromptParams) string {
	var sb strings.Builder

	if p.History != "" {
		fmt.Fprintf(&sb, "【对话摘要】%s\n\n", p.History)
	}

	if p.Template != nil {
		fmt.Fprintf(&sb, "【策略模板参考 (%s)】\n", p.Template.Name)
		fmt.Fprintf(&sb, "类别: %s\n", p.Template.Category)
		fmt.Fprintf(&sb, "描述: %s\n", p.Template.DescriptionZh)
		fmt.Fprintf(&sb, "参数说明: %s\n\n", p.Template.ParameterSlotsString())
	}

	if p.Symbol != "" || p.Timeframe != "" {
		sb.WriteString("【交易配置】\n")
		if p.Symbol != "" {
			fmt.Fprintf(&sb, "品种: %s\n", p.Symbol)
		}
		if p.Timeframe != "" {
			fmt.Fprintf(&sb, "周期: %s\n", p.Timeframe)
		}
		sb.WriteString("\n")
	}

	// Intent-driven guidance
	if p.Intent != nil && !p.Intent.NeedsClarification {
		switch p.Intent.RiskLevel {
		case "high":
			sb.WriteString("用户风险偏好较高，可接受更大回撤以换取更高收益。\n")
		case "low":
			sb.WriteString("用户偏好低风险策略，请注重回撤控制和稳健收益。\n")
		}
		switch p.Intent.TradeDirection {
		case "long":
			sb.WriteString("用户只想做多，不要生成做空信号。\n")
		case "short":
			sb.WriteString("用户只想做空，不要生成做多信号。\n")
		}
		sb.WriteString("\n")
	}

	fmt.Fprintf(&sb, "【用户需求】\n%s\n\n", p.Message)
	sb.WriteString("请生成策略代码：")

	return sb.String()
}
