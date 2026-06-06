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
	ParamMap    map[string]string // from clarification engine
	History     string            // previous conversation summary
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

	// Parameter annotations in skeleton
	if p.ParamMap != nil && len(p.ParamMap) > 0 {
		sb.WriteString("根据用户偏好调整以下参数方向：\n")
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

	sb.WriteString(fmt.Sprintf("【用户需求】\n%s\n\n", p.Message))
	sb.WriteString("请生成策略代码：")

	return sb.String()
}
