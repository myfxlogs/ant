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

	"anttrader/internal/repository"
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
			sb.WriteString(fmt.Sprintf("- 策略类型: %s\n", p.Intent.StrategyFamily))
		}
		if p.Intent.RiskLevel != "" && p.Intent.RiskLevel != "unknown" {
			sb.WriteString(fmt.Sprintf("- 风险偏好: %s\n", p.Intent.RiskLevel))
		}
		if p.Intent.TradeDirection != "" && p.Intent.TradeDirection != "unknown" {
			sb.WriteString(fmt.Sprintf("- 交易方向: %s\n", p.Intent.TradeDirection))
		}
		if p.Intent.HoldingPeriod != "" && p.Intent.HoldingPeriod != "unknown" {
			sb.WriteString(fmt.Sprintf("- 持仓周期: %s\n", p.Intent.HoldingPeriod))
		}
		if len(p.Intent.EntrySignals) > 0 {
			sb.WriteString(fmt.Sprintf("- 入场信号: %s\n", strings.Join(p.Intent.EntrySignals, ", ")))
		}
		if len(p.Intent.ExitSignals) > 0 {
			sb.WriteString(fmt.Sprintf("- 离场信号: %s\n", strings.Join(p.Intent.ExitSignals, ", ")))
		}
		sb.WriteString("\n")
	}

	// Parameter hints from clarification/fallback
	if p.ParamMap != nil && len(p.ParamMap) > 0 {
		sb.WriteString("参数偏好：\n")
		for k, v := range p.ParamMap {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
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
		hints = "用户对回测结果不满意，请根据反馈优化策略"
	}
	// Detect strategy type from previous code if not explicitly set.
	st := p.StrategyType
	if st == "" {
		st = DetectCodeStrategyType(p.PreviousCode)
	}
	system := fmt.Sprintf(feedbackSystemTemplate,
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
		system += fmt.Sprintf("\n\n## Gate 评估失败信息\n"+
			"上一次策略在 Gate '%s' 失败: %s\n"+
			"请在修复代码时针对此问题进行改进，确保策略能通过此次 Gate 检查。",
			p.GateFailureReason, details)
	}

	user := fmt.Sprintf("【用户反馈】%s\n\n请分析回测结果，给出建议，并生成优化后的代码。", p.FeedbackMessage)
	return system, user
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
		sb.WriteString(fmt.Sprintf("【对话摘要】%s\n\n", p.History))
	}

	if p.Template != nil {
		sb.WriteString(fmt.Sprintf("【策略模板参考 (%s)】\n", p.Template.Name))
		sb.WriteString(fmt.Sprintf("类别: %s\n", p.Template.Category))
		sb.WriteString(fmt.Sprintf("描述: %s\n", p.Template.DescriptionZh))
		sb.WriteString(fmt.Sprintf("参数说明: %s\n\n", p.Template.ParameterSlotsString()))
	}

	if p.Symbol != "" || p.Timeframe != "" {
		sb.WriteString("【交易配置】\n")
		if p.Symbol != "" {
			sb.WriteString(fmt.Sprintf("品种: %s\n", p.Symbol))
		}
		if p.Timeframe != "" {
			sb.WriteString(fmt.Sprintf("周期: %s\n", p.Timeframe))
		}
		sb.WriteString("\n")
	}

	// Intent-driven guidance
	if p.Intent != nil && !p.Intent.NeedsClarification {
		if p.Intent.RiskLevel == "high" {
			sb.WriteString("用户风险偏好较高，可接受更大回撤以换取更高收益。\n")
		} else if p.Intent.RiskLevel == "low" {
			sb.WriteString("用户偏好低风险策略，请注重回撤控制和稳健收益。\n")
		}
		if p.Intent.TradeDirection == "long" {
			sb.WriteString("用户只想做多，不要生成做空信号。\n")
		} else if p.Intent.TradeDirection == "short" {
			sb.WriteString("用户只想做空，不要生成做多信号。\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("【用户需求】\n%s\n\n", p.Message))
	sb.WriteString("请生成策略代码：")

	return sb.String()
}
