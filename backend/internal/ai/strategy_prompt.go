// strategy_prompt.go: system prompt builder for AI strategy code generation.
//
// Builds a structured system prompt that includes:
//   - Template skeleton with parameter slots
//   - User-provided parameters and preferences
//   - Python strategy contract (required functions, signals)
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
	Template    *repository.AIStrategyTemplate
	Message     string // user's natural language description
	Symbol      string
	Timeframe   string
	ParamMap    map[string]string // merged: LLM intent + keyword fallback
	History     string            // previous conversation summary
	Intent      *IntentResult     // LLM-extracted intent (nil for legacy callers)
}

// FeedbackPromptParams holds inputs for building the feedback iteration prompt (Phase 3).
type FeedbackPromptParams struct {
	PreviousCode    string           // previous strategy code
	Metrics         *FeedbackMetrics // last backtest metrics
	FeedbackMessage string           // user's feedback text
	FeedbackHints   string           // hints from feedback_router
}

// strategyContractText returns the shared Python strategy contract + forbidden patterns.
func strategyContractText() string {
	var sb strings.Builder
	sb.WriteString("## 策略代码规范\n\n")
	sb.WriteString("你必须定义一个 run(context) 函数，返回交易信号字典：\n\n")
	sb.WriteString("```python\n")
	sb.WriteString("def run(context):\n")
	sb.WriteString("    # context 是字典，包含以下键：\n")
	sb.WriteString("    #   context['open']/['high']/['low']/['close']: 价格列表\n")
	sb.WriteString("    #   context['volume']: 成交量列表\n")
	sb.WriteString("    #   context['position']: 当前持仓 dict {'side':'buy'/'sell',...} 或 None\n")
	sb.WriteString("    #   context['balance']: 当前余额\n\n")
	sb.WriteString("    # 返回信号字典：\n")
	sb.WriteString("    return {\n")
	sb.WriteString("        'signal': 'buy',     # 'buy','sell','hold'\n")
	sb.WriteString("        'volume': 1.0,      # 交易手数\n")
	sb.WriteString("        'stop_loss': 0.0,   # 止损价格(可选)\n")
	sb.WriteString("        'take_profit': 0.0, # 止盈价格(可选)\n")
	sb.WriteString("    }\n")
	sb.WriteString("```\n\n")

	sb.WriteString("可调参数使用 Python 注释标注（引擎会从注释中提取）：\n")
	sb.WriteString("```python\n")
	sb.WriteString("# @param fast_period 10 range=5:50:5\n")
	sb.WriteString("# @param slow_period 30 range=20:100:10\n")
	sb.WriteString("```\n\n")

	sb.WriteString("## 禁止事项\n")
	sb.WriteString("1. 不要使用 eval()、exec() 或 compile()\n")
	sb.WriteString("2. 不要导入 os、subprocess、socket、pickle、marshal\n")
	sb.WriteString("3. 不要使用 open() 进行文件操作\n")
	sb.WriteString("4. 不要访问 __builtins__、globals()、locals()\n")
	sb.WriteString("5. 不要使用 atexit 或 signal 注册处理器\n")
	sb.WriteString("6. 可使用 numpy (import numpy as np)\n")
	sb.WriteString("7. 代码必须完整可执行，包含所有必要的 import\n")
	sb.WriteString("8. 只输出 Python 代码，不要包含解释文字\n\n")
	return sb.String()
}

// feedbackSystemTemplate is the system prompt template for feedback iteration mode.
// It instructs the LLM to analyze backtest results and output structured sections.
const feedbackSystemTemplate = `你是量化策略迭代助手。用户已查看回测结果并给出反馈，你需要：

1. 分析回测结果的问题（1-2 句，中文）
2. 给出具体优化建议（1-2 条）
3. 生成优化后的完整 Python 策略代码

## 输出格式（严格遵守）
用 <section> 标签分隔三个部分：

<section type="analysis">
简要分析回测结果，指出问题。例如："Sharpe 0.45 偏低，最大回撤 28%% 超过风控线..."
</section>

<section type="advice">
具体优化建议。例如："建议将 fast_period 从 5 调整到 10 以减少过度交易"
</section>

<section type="code">
` + "```python" + `
# 完整的优化后策略代码（包含所有必要的 import 和 @param 注解）
` + "```" + `
</section>

## 代码规范
%s

## 当前策略代码
%s

## 回测结果
%s

## 优化方向提示
%s`

// StrategyPromptBuilder constructs system + user prompts for strategy generation.
type StrategyPromptBuilder struct{}

// NewStrategyPromptBuilder creates a new prompt builder.
func NewStrategyPromptBuilder() *StrategyPromptBuilder {
	return &StrategyPromptBuilder{}
}

// BuildSystemPrompt returns the system prompt that instructs the LLM.
func (b *StrategyPromptBuilder) BuildSystemPrompt(p *PromptParams) string {
	var sb strings.Builder

	// Role definition
	sb.WriteString("你是一位专业的量化交易策略工程师。")
	sb.WriteString("你的任务是根据用户的自然语言描述，生成符合规范的 Python 策略代码。\n\n")

	// Strategy contract
	sb.WriteString(strategyContractText())

	// LLM-extracted intent context
	if p.Intent != nil && !p.Intent.NeedsClarification {
		sb.WriteString("根据分析，用户的策略偏好如下：\n")
		if p.Intent.StrategyFamily != "" && p.Intent.StrategyFamily != "unknown" {
			sb.WriteString(fmt.Sprintf("- 策略类型: %s\n", p.Intent.StrategyFamily))
		}
		if p.Intent.RiskLevel != "" && p.Intent.RiskLevel != "unknown" {
			sb.WriteString(fmt.Sprintf("- 风险偏好: %s\n", p.Intent.RiskLevel))
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
func (b *StrategyPromptBuilder) BuildFeedbackPrompt(p *FeedbackPromptParams) (string, string) {
	metricsCtx := ""
	if p.Metrics != nil {
		metricsCtx = p.Metrics.FormatPromptContext()
	}
	hints := p.FeedbackHints
	if hints == "" {
		hints = "用户对回测结果不满意，请根据反馈优化策略"
	}
	system := fmt.Sprintf(feedbackSystemTemplate,
		strategyContractText(),
		p.PreviousCode,
		metricsCtx,
		hints,
	)
	user := fmt.Sprintf("【用户反馈】%s\n\n请分析回测结果，给出建议，并生成优化后的代码。", p.FeedbackMessage)
	return system, user
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
