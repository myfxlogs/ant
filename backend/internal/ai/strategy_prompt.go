// strategy_prompt.go: system prompt builder for AI strategy code generation.
//
// Builds a structured system prompt that includes:
//   - Template skeleton with parameter slots
//   - User-provided parameters and preferences
//   - Python strategy contract (required functions, signals)
//   - Coding constraints (forbidden patterns, required annotations)
//   - Phase 3: feedback prompt with backtest metrics + iteration context
//   - Vectorized (run_dataframe) and event-driven (run_context) dual-mode support

package ai

import (
	"fmt"
	"strings"

	"anttrader/internal/repository"
	"anttrader/internal/service"
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
	StrategyType string            // "run_dataframe" (vectorized) or "run_context" (event-driven); empty = default to run_context
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

// contractText returns the appropriate Python strategy contract based on strategy type.
func contractText(strategyType string) string {
	if strategyType == "run_dataframe" {
		return dataframeContractText()
	}
	return contextContractText()
}

// contextContractText returns the event-driven run(context) contract + forbidden patterns.
func contextContractText() string {
	var sb strings.Builder
	sb.WriteString("## 策略代码规范（事件驱动模式）\n\n")
	sb.WriteString("你必须定义一个 run(context) 函数，返回交易信号字典：\n\n")
	sb.WriteString("```python\n")
	sb.WriteString("def run(context):\n")
	sb.WriteString("    # context 是字典，包含以下键：\n")
	sb.WriteString("    #   context['open']/['high']/['low']/['close']: 价格列表\n")
	sb.WriteString("    #   context['volume']: 成交量列表\n")
	sb.WriteString("    #   context.get('position'): 当前持仓 或 None（用 .get 访问）\n")
	sb.WriteString("    #   context.get('balance'): 当前余额（用 .get 访问）\n\n")
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
	sb.WriteString("**重要：@param 注释只用于引擎提取，代码中必须定义同名变量：**\n")
	sb.WriteString("```python\n")
	sb.WriteString("fast_period = 10  # 必须定义变量，否则 NameError\n")
	sb.WriteString("slow_period = 30\n")
	sb.WriteString("```\n\n")

	sb.WriteString("## 沙箱约束\n")
	sb.WriteString("- np, math 已预注入，禁止 import 语句\n")
	sb.WriteString("- 禁止 eval/exec/open/globals/locals/dunder/global\n\n")
	return sb.String()
}

// dataframeContractText returns the vectorized run_dataframe(df, params) contract.
// This is the recommended mode for indicator-based strategies: cleaner signal
// logic, better chart integration, and efficient vectorized backtesting.
func dataframeContractText() string {
	var sb strings.Builder
	sb.WriteString("## 策略代码规范（矢量模式 / IndicatorStrategy）\n\n")
	sb.WriteString("你必须定义一个 `run_dataframe(df, params)` 函数，返回信号 DataFrame：\n\n")

	sb.WriteString("### 函数签名\n")
	sb.WriteString("```python\n")
	sb.WriteString("def run_dataframe(df, params):\n")
	sb.WriteString("    # df: 包含完整 OHLC 的 DataFrame\n")
	sb.WriteString("    #   列: 'open', 'high', 'low', 'close', 'volume'\n")
	sb.WriteString("    #   'time' 列可能存在，但不要假设其类型\n")
	sb.WriteString("    # params: 参数字典，通过 params.get('key', default) 读取\n")
	sb.WriteString("    #\n")
	sb.WriteString("    # 返回: DataFrame，与输入 df 长度一致，包含信号列\n")
	sb.WriteString("    #   两路信号（简单趋势策略）: df['buy'], df['sell']\n")
	sb.WriteString("    #   四路信号（多空分离，推荐）: df['open_long'], df['close_long'],\n")
	sb.WriteString("    #                                df['open_short'], df['close_short']\n")
	sb.WriteString("    return df\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### 信号列规范\n")
	sb.WriteString("**两路信号** (buy/sell)：\n")
	sb.WriteString("- `df['buy']` / `df['sell']`: bool 列，长度必须等于 len(df)\n")
	sb.WriteString("- 配合 `tradeDirection` 声明使用：\n")
	sb.WriteString("  - `long`: buy=开多, sell=平多\n")
	sb.WriteString("  - `short`: buy=平空, sell=开空\n")
	sb.WriteString("  - `both`: buy=开多(持空则先平空), sell=开空(持多则先平多)\n\n")
	sb.WriteString("**四路信号** (推荐，语义更精确)：\n")
	sb.WriteString("- `df['open_long']` / `df['close_long']`: 多仓开/平\n")
	sb.WriteString("- `df['open_short']` / `df['close_short']`: 空仓开/平\n")
	sb.WriteString("- 四列必须同时存在且为 bool\n")
	sb.WriteString("- 同 bar 优先级: close_* 先于 open_*（引擎自动处理先平后开）\n\n")

	sb.WriteString("### 关键约束\n")
	sb.WriteString("- **MUST** 首行写 `df = df.copy()`\n")
	sb.WriteString("- **MUST** 信号列做边缘触发（只在条件刚成立时触发一次）：\n")
	sb.WriteString("```python\n")
	sb.WriteString("def edge(s):\n")
	sb.WriteString("    s = s.fillna(False).astype(bool)\n")
	sb.WriteString("    return s & ~s.shift(1).fillna(False)\n")
	sb.WriteString("\n")
	sb.WriteString("df['buy'] = edge(raw_buy_condition)\n")
	sb.WriteString("df['sell'] = edge(raw_sell_condition)\n")
	sb.WriteString("```\n")
	sb.WriteString("- **MUST NOT** 使用 `shift(-1)` 或任何未来数据\n")
	sb.WriteString("- **MUST** 所有信号列 `fillna(False).astype(bool)`\n")
	sb.WriteString("- **MUST** 信号 DataFrame 长度与输入 df 完全一致\n\n")

	sb.WriteString("### 元数据声明\n")
	sb.WriteString("脚本开头声明名称和描述：\n")
	sb.WriteString("```python\n")
	sb.WriteString("my_indicator_name = \"策略名称\"\n")
	sb.WriteString("my_indicator_description = \"策略描述\"\n")
	sb.WriteString("```\n\n")
	sb.WriteString("可调参数（引擎和 UI 均可识别）：\n")
	sb.WriteString("```python\n")
	sb.WriteString("# @param fast_len int 20 Fast EMA length\n")
	sb.WriteString("# @param slow_len int 50 Slow EMA length\n")
	sb.WriteString("# @param rsi_floor float 45.0 Minimum RSI\n")
	sb.WriteString("```\n")
	sb.WriteString("代码中用 `params.get()` 读取：\n")
	sb.WriteString("```python\n")
	sb.WriteString("fast_len = int(params.get('fast_len', 20))\n")
	sb.WriteString("```\n\n")
	sb.WriteString("默认风控配置（引擎负责执行，不要在 df 里重复造止损列）：\n")
	sb.WriteString("```python\n")
	sb.WriteString("# @strategy stopLossPct 0.03       # 3% 止损\n")
	sb.WriteString("# @strategy takeProfitPct 0.06     # 6% 止盈\n")
	sb.WriteString("# @strategy entryPct 0.25          # 25% 资金开仓\n")
	sb.WriteString("# @strategy tradeDirection both    # long | short | both\n")
	sb.WriteString("```\n")
	sb.WriteString("**数值单位**: 0-1 小数比例（0.03 = 3%）。**禁止**把 `leverage` 写进 @strategy。\n\n")

	sb.WriteString("### 图表输出（可选，用于前端渲染指标线）\n")
	sb.WriteString("```python\n")
	sb.WriteString("output = {\n")
	sb.WriteString("    \"name\": my_indicator_name,\n")
	sb.WriteString("    \"plots\": [\n")
	sb.WriteString("        {\"name\": \"EMA Fast\", \"data\": ema_fast.fillna(0).tolist(),\n")
	sb.WriteString("         \"color\": \"#1890ff\", \"overlay\": True},\n")
	sb.WriteString("    ],\n")
	sb.WriteString("    \"signals\": [\n")
	sb.WriteString("        {\"type\": \"buy\", \"text\": \"B\",\n")
	sb.WriteString("         \"data\": [df['low'].iloc[i]*0.995 if df['buy'].iloc[i] else None for i in range(len(df))],\n")
	sb.WriteString("         \"color\": \"#00E676\"},\n")
	sb.WriteString("    ]\n")
	sb.WriteString("}\n")
	sb.WriteString("```\n")
	sb.WriteString("- 所有 plot['data'] 和 signal['data'] 长度必须等于 len(df)\n")
	sb.WriteString("- signal 无信号位置用 None\n\n")

	sb.WriteString("### 沙箱约束\n")
	sb.WriteString("- 允许 import: numpy, pandas, math, json, datetime, time, collections, functools, itertools, statistics, decimal, fractions, copy\n")
	sb.WriteString("- pd, np, params 已预注入，一般无需再 import\n")
	sb.WriteString("- 禁止: 网络请求、文件读写、子进程、eval/exec/open/__import__/getattr/setattr\n\n")
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

	// Strategy contract — contractText defaults empty to "run_context".
	st := p.StrategyType
	sb.WriteString(contractText(st))

	// Indicator catalog — only for vectorized strategies so LLM knows available helpers.
	if st == "run_dataframe" {
		sb.WriteString(service.BuildIndicatorCatalogPromptBlockCompact())
		sb.WriteString("\n")
	}

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

// DetectCodeStrategyType heuristically detects whether code uses run_dataframe or run_context.
func DetectCodeStrategyType(code string) string {
	if strings.Contains(code, "def run_dataframe(") {
		return "run_dataframe"
	}
	return "run_context"
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
