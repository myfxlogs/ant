// strategy_contracts.go: Go strategy contract texts and template constants.
//
// Extracted from strategy_prompt.go to keep the builder file focused on
// prompt assembly logic while contract texts stay in their own file.
// These are pure data — no business logic.

package ai

import "strings"

// goStrategyCodeExample is the canonical Go SDK strategy code example shared
// by all agent prompts and the contract text. Single source of truth.
const goStrategyCodeExample = "```go\n" +
	"import (\n" +
	"    \"alphaforge/strategy/sdk\"\n" +
	"    \"github.com/shopspring/decimal\"\n" +
	")\n\n" +
	"type MyStrategy struct {\n" +
	"    fastPeriod int\n" +
	"    slPct      float64\n" +
	"    tpPct      float64\n" +
	"    entryPct   float64\n" +
	"}\n\n" +
	"func (s *MyStrategy) OnInit(ctx sdk.Context) error {\n" +
	"    s.fastPeriod = ctx.Param(\"fast_period\", 20)\n" +
	"    s.slPct = ctx.Param(\"stopLossPct\", 0.02)\n" +
	"    s.tpPct = ctx.Param(\"takeProfitPct\", 0.04)\n" +
	"    s.entryPct = ctx.Param(\"entryPct\", 0.25)\n" +
	"    return nil\n" +
	"}\n\n" +
	"func (s *MyStrategy) OnBar(ctx sdk.Context, timeframe string) (*sdk.Signal, error) {\n" +
	"    bars := ctx.Bars(timeframe)\n" +
	"    if bars.Len() < s.fastPeriod+1 {\n" +
	"        return &sdk.Signal{Action: sdk.ActionHold}, nil\n" +
	"    }\n" +
	"    price := bars.Close(0)\n" +
	"    balance := ctx.Broker().AccountInfo().Balance\n" +
	"    volume := balance.Mul(decimal.NewFromFloat(s.entryPct)).Div(price)\n" +
	"    return &sdk.Signal{\n" +
	"        Action:     sdk.ActionBuy,\n" +
	"        Symbol:     ctx.Symbol(),\n" +
	"        Volume:     volume,\n" +
	"        Price:      price,\n" +
	"        StopLoss:   price.Mul(decimal.NewFromFloat(1 - s.slPct)),\n" +
	"        TakeProfit: price.Mul(decimal.NewFromFloat(1 + s.tpPct)),\n" +
	"    }, nil\n" +
	"}\n\n" +
	"func (s *MyStrategy) OnDeinit(ctx sdk.Context, reason string) error { return nil }\n" +
	"```\n"

// goStrategyConstraints is the canonical constraint list shared by agent prompts.
const goStrategyConstraints = `Code constraints (violations cause code rejection):
- Must implement sdk.Strategy interface: OnInit/OnBar/OnDeinit
- All monetary values use decimal.Decimal, never float64
- Indicators via ctx.Indicators() (MA/EMA/RSI/ATR/Bands/MACD etc.)
- Positions via ctx.Broker().Positions()
- Forbidden: eval, exec, file I/O, network, subprocess

Common anti-patterns to avoid (these will be flagged by the deep check):
- Every parameter must be read via ctx.Param() in OnInit — never hardcode
- StopLoss/TakeProfit must not be decimal.Zero when holding a position
- Volume must be calculated from AccountInfo().Balance — never hardcode
- bars.Len() must exceed indicator period — otherwise return ActionHold
- RSI must use Wilder's smoothing (alpha=1/period), not SMA smoothing
- ATR needs period+1 bars — check bars.Len() >= atr_period + 1

Parameter declaration (auto-detected by optimizer):
` + "```go\n" +
	"// Read parameters in OnInit via ctx.Param(), second arg is default\n" +
	"s.fastPeriod = ctx.Param(\"fast_period\", 20)      // range=5:50:5\n" +
	"s.slowPeriod = ctx.Param(\"slow_period\", 50)      // range=20:100:10\n" +
	"s.slPct = ctx.Param(\"stopLossPct\", 0.02)          // strategy-level param\n" +
	"s.tpPct = ctx.Param(\"takeProfitPct\", 0.04)        // strategy-level param\n" +
	"s.entryPct = ctx.Param(\"entryPct\", 0.25)          // capital allocation per trade\n" +
	"```\n"

// goStrategyConstraintsZH is the Chinese version of the constraint list.
const goStrategyConstraintsZH = `代码约束（违反会导致代码被拒绝）：
- 必须实现 sdk.Strategy 接口：OnInit/OnBar/OnDeinit
- 所有金额使用 decimal.Decimal，禁止 float64
- 指标通过 ctx.Indicators() 获取（MA/EMA/RSI/ATR/Bands/MACD 等）
- 持仓通过 ctx.Broker().Positions() 查询
- 禁止使用 eval、exec、文件读写、网络请求、子进程

生成代码时必须规避以下常见错误（这些在深度检测中会被标记）：
- 每个参数必须在 OnInit 中通过 ctx.Param() 读取，禁止硬编码
- 持仓时必须每根 bar 返回止损止盈，不能设为 decimal.Zero
- 仓位大小用 AccountInfo().Balance 计算，不要硬编码 volume
- bars.Len() 必须大于指标周期，否则返回 ActionHold
- RSI 计算必须用 Wilder's 平滑（alpha=1/period），不要用 SMA 平滑
- ATR 计算需要 period+1 根 bar 的数据

参数声明（引擎自动识别，用于优化器扫描）：
` + "```go\n" +
	"// 在 OnInit 中通过 ctx.Param() 读取参数，第二个参数为默认值\n" +
	"s.fastPeriod = ctx.Param(\"fast_period\", 20)      // range=5:50:5\n" +
	"s.slowPeriod = ctx.Param(\"slow_period\", 50)      // range=20:100:10\n" +
	"s.slPct = ctx.Param(\"stopLossPct\", 0.02)          // 策略级参数\n" +
	"s.tpPct = ctx.Param(\"takeProfitPct\", 0.04)        // 策略级参数\n" +
	"s.entryPct = ctx.Param(\"entryPct\", 0.25)          // 单次开仓资金比例\n" +
	"```\n"

// contractText returns the Go strategy contract based on strategy type.
func contractText(strategyType string) string {
	return goStrategyContractText()
}

// goStrategyContractText returns the Go SDK strategy contract + MANDATORY safety rules.
func goStrategyContractText() string {
	var sb strings.Builder

	sb.WriteString("## ⛔ 代码生成铁律 — 违反任何一条输出都会被拒绝 ⛔\n\n")

	sb.WriteString(goStrategyRule1Text)

	sb.WriteString("### 铁律 2：持仓查询通过 ctx.Broker().Positions() — 返回 []sdk.Position\n")
	sb.WriteString("```go\n")
	sb.WriteString("func (s *MyStrategy) OnBar(ctx sdk.Context, timeframe string) (*sdk.Signal, error) {\n")
	sb.WriteString("    positions := ctx.Broker().Positions(0) // 0 = all magic numbers\n")
	sb.WriteString("    for _, pos := range positions {\n")
	sb.WriteString("        // pos.Ticket (int64), pos.Side (sdk.SideBuy/SideSell)\n")
	sb.WriteString("        // pos.Volume (decimal.Decimal), pos.OpenPrice (decimal.Decimal)\n")
	sb.WriteString("        // pos.SL, pos.TP (decimal.Decimal)\n")
	sb.WriteString("    }\n")
	sb.WriteString("    return nil, nil\n")
	sb.WriteString("}\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### 铁律 3：持仓时必须每根 bar 返回止损止盈，不能设为 0\n")
	sb.WriteString("```go\n")
	sb.WriteString("    price := bars.Close(0)\n")
	sb.WriteString("    if pos.Side == sdk.SideBuy {\n")
	sb.WriteString("        stopLoss = price.Mul(decimal.NewFromFloat(1 - s.slPct))\n")
	sb.WriteString("        takeProfit = price.Mul(decimal.NewFromFloat(1 + s.tpPct))\n")
	sb.WriteString("    } else {\n")
	sb.WriteString("        stopLoss = price.Mul(decimal.NewFromFloat(1 + s.slPct))\n")
	sb.WriteString("        takeProfit = price.Mul(decimal.NewFromFloat(1 - s.tpPct))\n")
	sb.WriteString("    }\n")
	sb.WriteString("    // ❌ 绝对禁止：StopLoss: decimal.Zero, TakeProfit: decimal.Zero\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### 铁律 4：仓位大小用 AccountInfo().Balance\n")
	sb.WriteString("```go\n")
	sb.WriteString("    balance := ctx.Broker().AccountInfo().Balance\n")
	sb.WriteString("    price := bars.Close(0)\n")
	sb.WriteString("    volume := balance.Mul(decimal.NewFromFloat(s.entryPct)).Div(price)\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### 铁律 5：函数签名与返回值\n")
	sb.WriteString("```go\n")
	sb.WriteString("func (s *MyStrategy) OnBar(ctx sdk.Context, timeframe string) (*sdk.Signal, error) {\n")
	sb.WriteString("    bars := ctx.Bars(timeframe)\n")
	sb.WriteString("    // bars.Close(0) = current bar close (index 0 = most recent)\n")
	sb.WriteString("    // bars.Len() = number of bars available\n")
	sb.WriteString("    return &sdk.Signal{\n")
	sb.WriteString("        Action:      sdk.ActionBuy,   // ActionBuy | ActionSell | ActionHold\n")
	sb.WriteString("        Symbol:      ctx.Symbol(),\n")
	sb.WriteString("        Volume:      volume,          // decimal.Decimal\n")
	sb.WriteString("        Price:       price,           // decimal.Decimal\n")
	sb.WriteString("        StopLoss:    stopLoss,        // decimal.Decimal (持仓时禁止为 0)\n")
	sb.WriteString("        TakeProfit:  takeProfit,      // decimal.Decimal (持仓时禁止为 0)\n")
	sb.WriteString("    }, nil\n")
	sb.WriteString("}\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### 铁律 6：必须实现 sdk.Strategy 接口\n")
	sb.WriteString("- OnInit(ctx sdk.Context) error\n")
	sb.WriteString("- OnBar(ctx sdk.Context, timeframe string) (*sdk.Signal, error)\n")
	sb.WriteString("- OnDeinit(ctx sdk.Context, reason string) error\n")
	sb.WriteString("- 所有金额使用 decimal.Decimal，禁止 float64\n")
	sb.WriteString("- 指标通过 ctx.Indicators() 获取 (MA/EMA/RSI/ATR/Bands/MACD/Stochastic 等)\n\n")

	return sb.String()
}

// feedbackSystemTemplateZH is the Chinese system prompt template for feedback iteration mode.

const goStrategyRule1Text = `### 铁律 1：每个参数必须在 OnInit 中通过 ctx.Param() 读取，禁止硬编码
` + "```go\n" + `import (
    "alphaforge/strategy/sdk"
    "github.com/shopspring/decimal"
)

type MyStrategy struct {
    fastPeriod   int
    slowPeriod   int
    entryPct     float64
    slPct        float64
    tpPct        float64
}

func (s *MyStrategy) OnInit(ctx sdk.Context) error {
    s.fastPeriod = ctx.Param("fast_period", 20)
    s.slowPeriod = ctx.Param("slow_period", 50)
    s.entryPct = ctx.Param("entryPct", 0.25)
    s.slPct = ctx.Param("stopLossPct", 0.02)
    s.tpPct = ctx.Param("takeProfitPct", 0.04)
    return nil
}
` + "```\n\n"

// feedbackSystemTemplateZH is the Chinese system prompt template for feedback iteration mode.
const feedbackSystemTemplateZH = `你是量化策略迭代助手。用户已查看回测结果并给出反馈，你需要：

1. 分析回测结果的问题（1-2 句，中文）
2. 给出具体优化建议（1-2 条）
3. 生成优化后的完整 Go 策略代码

## 输出格式（严格遵守）
用 <section> 标签分隔三个部分：

<section type="analysis">
简要分析回测结果，指出问题。例如："Sharpe 0.45 偏低，最大回撤 28%% 超过风控线..."
</section>

<section type="advice">
具体优化建议。例如："建议将 fast_period 从 5 调整到 10 以减少过度交易"
</section>

<section type="code">
` + "```go" + `
// 完整的优化后 Go 策略代码（实现 sdk.Strategy 接口）
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

// feedbackSystemTemplateEN is the English system prompt template for feedback iteration mode.
const feedbackSystemTemplateEN = `You are a quantitative strategy iteration assistant. The user has reviewed backtest results and provided feedback. You need to:

1. Analyze the backtest results and identify problems (1-2 sentences)
2. Provide specific optimization suggestions (1-2 items)
3. Generate the complete optimized Go strategy code

## Output Format (strictly follow)
Use <section> tags to separate three parts:

<section type="analysis">
Briefly analyze the backtest results and identify issues. Example: "Sharpe 0.45 is low, max drawdown 28%% exceeds risk limits..."
</section>

<section type="advice">
Specific optimization suggestions. Example: "Consider increasing fast_period from 5 to 10 to reduce overtrading"
</section>

<section type="code">
` + "```go" + `
// Complete optimized Go strategy code (implements sdk.Strategy interface)
` + "```" + `
</section>

## Code Standards
%s

## Current Strategy Code
%s

## Backtest Results
%s

## Optimization Hints
%s`
