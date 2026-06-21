// strategy_contracts.go: Python strategy contract texts and template constants.
//
// Extracted from strategy_prompt.go to keep the builder file focused on
// prompt assembly logic while contract texts stay in their own file.
// These are pure data — no business logic.

package ai

import "strings"

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

	sb.WriteString("## 强制性安全规则（违反将导致代码被拒绝）\n\n")

	sb.WriteString("### 1. @param 与变量一致性\n")
	sb.WriteString("每个 @param 注释都必须有对应的变量定义，且必须从 context 读取：\n")
	sb.WriteString("```python\n")
	sb.WriteString("# @param fast_period 20 range=10:50:10\n")
	sb.WriteString("fast_period = int(context.get('fast_period', 20))  # ✅ 从 context 读取\n")
	sb.WriteString("```\n")
	sb.WriteString("**禁止** 在 @param 声明后使用硬编码值：\n")
	sb.WriteString("```python\n")
	sb.WriteString("# @param fast_period 20\n")
	sb.WriteString("ema20 = calc_ema(close, 20)  # ❌ 硬编码 20，必须用 fast_period 变量\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### 2. @strategy 参数读取\n")
	sb.WriteString("@strategy 声明的参数必须从 context 读取：\n")
	sb.WriteString("```python\n")
	sb.WriteString("# @strategy entryPct 0.25\n")
	sb.WriteString("# @strategy stopLossPct 0.02\n")
	sb.WriteString("# @strategy takeProfitPct 0.04\n")
	sb.WriteString("entry_pct = float(context.get('entryPct', 0.25))       # ✅ 必须定义\n")
	sb.WriteString("sl_pct   = float(context.get('stopLossPct', 0.02))\n")
	sb.WriteString("tp_pct   = float(context.get('takeProfitPct', 0.04))\n")
	sb.WriteString("```\n")
	sb.WriteString("**禁止** 硬编码：`sl_pct = 0.02` 在有 @strategy 声明时 ❌\n\n")

	sb.WriteString("### 3. 止损止盈必须持续返回\n")
	sb.WriteString("持有仓位时，stop_loss 和 take_profit **必须每根 bar 都返回**，不能只在新开仓时设置：\n")
	sb.WriteString("```python\n")
	sb.WriteString("if position is not None:  # 持有仓位\n")
	sb.WriteString("    # ✅ 必须返回止损止盈，保护现有仓位\n")
	sb.WriteString("    entry_price = position.get('open_price', close[-1])\n")
	sb.WriteString("    if position.get('type') == 'long':\n")
	sb.WriteString("        stop_loss   = entry_price * (1 - sl_pct)\n")
	sb.WriteString("        take_profit = entry_price * (1 + tp_pct)\n")
	sb.WriteString("    else:\n")
	sb.WriteString("        stop_loss   = entry_price * (1 + sl_pct)\n")
	sb.WriteString("        take_profit = entry_price * (1 - tp_pct)\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### 4. 仓位大小计算\n")
	sb.WriteString("使用 entryPct 计算手数时，**必须用初始余额**，不能用当前余额（避免重复计算）：\n")
	sb.WriteString("```python\n")
	sb.WriteString("initial_balance = float(context.get('initial_balance', 10000.0))\n")
	sb.WriteString("volume_ordered = (initial_balance * entry_pct) / close[-1]\n")
	sb.WriteString("```\n")
	sb.WriteString("禁止使用 `context.get('balance')` 计算仓位大小（每根 bar 都会变）❌\n\n")

	sb.WriteString("### 5. 变量必须先定义后使用\n")
	sb.WriteString("所有变量（fast_period, entry_pct, sl_pct 等）必须在 run() 函数开头定义，\n")
	sb.WriteString("确保在任何 return 分支之前都已初始化，避免 NameError。\n\n")

	sb.WriteString("### 6. 沙箱约束\n")
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
