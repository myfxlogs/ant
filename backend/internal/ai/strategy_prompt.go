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
)

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
