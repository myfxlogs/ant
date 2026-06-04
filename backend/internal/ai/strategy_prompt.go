// strategy_prompt.go: system prompt builder for AI strategy code generation.
//
// Builds a structured system prompt that includes:
//   - Template skeleton with parameter slots
//   - User-provided parameters and preferences
//   - Python strategy contract (required functions, signals)
//   - Coding constraints (forbidden patterns, required annotations)

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
	sb.WriteString("## 策略代码规范\n\n")
	sb.WriteString("生成的代码必须包含以下两个函数：\n\n")
	sb.WriteString("```python\n")
	sb.WriteString("def initialize(state):\n")
	sb.WriteString("    # 初始化策略参数：仓位、止损、止盈等\n")
	sb.WriteString("    pass\n\n")
	sb.WriteString("def on_bar(state, bar, history):\n")
	sb.WriteString("    # 每根 K 线调用一次，实现交易逻辑\n")
	sb.WriteString("    # bar: 当前 K 线 (open, high, low, close, volume, time)\n")
	sb.WriteString("    # history: 历史 K 线列表，history[-1] 是上一个 bar\n")
	sb.WriteString("    # state: 策略状态\n")
	sb.WriteString("    #   state.position > 0: 多头持仓\n")
	sb.WriteString("    #   state.position < 0: 空头持仓\n")
	sb.WriteString("    #   state.position == 0: 空仓\n")
	sb.WriteString("    pass\n")
	sb.WriteString("```\n\n")

	// State API
	sb.WriteString("可用的 state 方法：\n")
	sb.WriteString("- `state.open_long(price, entry_pct=1.0)`: 开多仓\n")
	sb.WriteString("- `state.open_short(price, entry_pct=1.0)`: 开空仓\n")
	sb.WriteString("- `state.close_position(price)`: 平仓\n")
	sb.WriteString("- `state.close_partial(price, pct)`: 部分平仓\n\n")

	// Annotation rules
	sb.WriteString("可调参数必须使用 @param 装饰器标注：\n")
	sb.WriteString("```python\n")
	sb.WriteString("@param period 20 range=10:50:5\n")
	sb.WriteString("@param entry_pct 0.95 range=0.5:1.0:0.05\n")
	sb.WriteString("```\n\n")

	// Forbidden patterns
	sb.WriteString("## 禁止事项\n")
	sb.WriteString("1. 不要使用 eval()、exec() 或 compile()\n")
	sb.WriteString("2. 不要导入 os、subprocess、socket、pickle、marshal\n")
	sb.WriteString("3. 不要使用 open() 进行文件操作\n")
	sb.WriteString("4. 不要访问 __builtins__、globals()、locals()\n")
	sb.WriteString("5. 不要使用 atexit 或 signal 注册处理器\n")
	sb.WriteString("6. 可使用 numpy (import numpy as np)\n")
	sb.WriteString("7. 代码必须完整可执行，包含所有必要的 import\n")
	sb.WriteString("8. 只输出 Python 代码，不要包含解释文字\n\n")

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
		sb.WriteString(fmt.Sprintf("参数说明: %s\n\n", string(p.Template.ParameterSlots)))
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
