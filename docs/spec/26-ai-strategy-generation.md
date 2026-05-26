# 26 · AI 策略生成架构规范

> **关联 ADR**：ADR-0017
> **关联 spec**：`docs/spec/21-backtest-replay.md`、`docs/spec/23-risk-management.md`

## 1. 使命

「让普通人都能用上量化交易系统」——非专业散户通过自然语言描述交易想法，AI 生成可在回测/仿真/实盘环境中运行的量化策略代码。

目标用户画像：
- 有交易经验但无编程能力
- 知道自己想做什么（"突破买入"、"做波段"、"趋势跟踪"）
- 但不知道如何用代码表达

产出的策略代码运行在 ant 的 DSL 引擎上，走标准回测→仿真→实盘路径。

## 2. 架构

```
用户自然语言输入
  │
  ▼
意图澄清引擎 (clarification.go)
  │  检测模糊词，追问 1-3 轮
  │
  ▼
会话记忆管理 (conversation_store.go)
  │  滑动窗口 N 轮消息 + 策略上下文
  │
  ▼
模板选择器 (template_library.go)
  │  匹配策略模板（趋势/回归/突破/网格）
  │
  ▼
AI Provider (strategy_prompt.go + LLM call)
  │  System prompt: 中文策略生成 + MQL 函数库 + DSL 语法
  │  Context: 模板骨架 + 用户参数 + 历史回测指标
  │
  ▼
策略代码生成
  │
  ▼
代码合规扫描 (code_compliance.go, 13 条规则)
  │
  ▼
自动回测 (ReplaySource + PaperExecutor, 见 spec/21/24)
  │
  ▼
回测结果注入下一轮对话 (backtest_feedback.go)
```

## 3. 意图澄清

```go
// internal/ai/clarification.go

type ClarificationRule struct {
    Keywords    []string  // 匹配的模糊词
    Questions   []string  // 追问列表
    ParamMap    map[string]string  // 映射到策略参数
}

var defaultRules = []ClarificationRule{
    {
        Keywords:  []string{"稳健", "保守", "低风险"},
        Questions: []string{
            "您能接受的最大回撤是多少？（例如：10% 以内）",
            "您偏好什么持仓周期？（日内/短线/中线/长线）",
        },
        ParamMap: map[string]string{
            "max_drawdown": "0.10",
            "min_period":   "1h",
        },
    },
    {
        Keywords:  []string{"进攻", "激进", "高风险", "高收益"},
        Questions: []string{
            "您能接受的最大回撤是多少？（例如：30%）",
            "是否允许日内高频交易？",
        },
        ParamMap: map[string]string{
            "max_drawdown": "0.30",
            "max_period":   "15m",
        },
    },
    {
        Keywords:  []string{"波段", "高抛低吸", "震荡"},
        Questions: []string{
            "您想操作的品种是什么？（BTC/ETH/外汇/指数）",
            "预计持仓时间是几小时还是几天？",
        },
    },
}
```

最多 3 轮追问。3 轮后使用默认值。

## 4. 会话记忆

```sql
CREATE TABLE ai_conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    conversation_id UUID NOT NULL,
    messages JSONB NOT NULL DEFAULT '[]',       -- [{role, content, ts}]
    strategy_context JSONB NOT NULL DEFAULT '{}', -- {dsl_code, last_backtest_metrics, parameters}
    status VARCHAR(16) DEFAULT 'active',         -- active / archived
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_ai_conv_user ON ai_conversations(user_id, conversation_id);
```

滑动窗口保留最近 20 条消息 + 策略上下文摘要。超过 20 条时，早期消息总结为一段摘要注入 prompt。

## 5. 策略模板库

```sql
CREATE TABLE platform_strategies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category VARCHAR(32) NOT NULL,       -- trend_following/mean_reversion/breakout/grid/martingale
    name VARCHAR(128) NOT NULL,
    description_zh TEXT NOT NULL,         -- 中文描述（面向用户）
    dsl_skeleton TEXT NOT NULL,           -- DSL 代码骨架，{PARAM} 为占位符
    parameter_slots JSONB NOT NULL,       -- [{name, type, default, min, max, description_zh}]
    risk_level VARCHAR(8) DEFAULT 'medium', -- low/medium/high/critical
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

模板示例：

| 类别 | 名称 | 参数 |
|------|------|------|
| trend_following | 双均线交叉 | fast_period, slow_period, symbol |
| mean_reversion | 布林带突破 | period, std_dev, symbol |
| breakout | Donchian Channel | period, symbol |
| grid | 网格交易 | grid_count, spacing_pct, symbol |
| martingale | 马丁策略 | multiplier, levels, symbol（标注 critical 风险） |

## 6. 回测反馈注入

```go
// internal/ai/backtest_feedback.go

type BacktestFeedback struct {
    Metrics    *BacktestMetrics
    Summary    string    // 一句话总结（中文）
}

type BacktestMetrics struct {
    SharpeRatio    float64
    MaxDrawdown    float64
    WinRate        float64
    ProfitFactor   float64
    TotalReturn    float64
    TotalTrades    int
}

func (f *BacktestFeedback) ToPromptContext() string {
    return fmt.Sprintf(`
【上一轮策略回测结果】
策略代码: %s
回测指标: Sharpe %.2f, 最大回撤 %.1f%%, 胜率 %.0f%%, 盈亏比 %.2f, 总收益 %.1f%%, 交易次数 %d
总结: %s
`, f.DSLCode, f.Metrics.SharpeRatio, f.Metrics.MaxDrawdown*100,
   f.Metrics.WinRate*100, f.Metrics.ProfitFactor, f.Metrics.TotalReturn*100,
   f.Metrics.TotalTrades, f.Summary)
}
```

用户说"太激进了" → AI 检查 `MaxDrawdown > 20%` → 降低 `position_size` 参数 → 重新生成 → 重新回测。

## 7. 代码生成质量门

| 门禁 | 说明 | 失败动作 |
|------|------|----------|
| code_compliance | 13 条禁止模式扫描 | 返回违规描述，要求用户修改意图 |
| DSL 语法验证 | `dsl.Parser.Parse()` 无错误 | 修复语法后重新生成 |
| 最少 bar 数 | 回测窗口 >= 1000 bars | 提示用户扩大回测时间范围 |
| 风控兼容 | 生成的策略参数不超过 risk_limits | 警告用户并建议调整 |

## 8. AI Provider 集成

复用现有 `internal/ai/` 层（12 家 provider 抽象）。策略生成使用 provider 的 chat completion API，temperature=0.3（策略代码需要确定性）。

```go
type StrategyGenerator struct {
    provider     AIProvider
    promptBuilder *StrategyPromptBuilder
    compliance   *CodeComplianceScanner
    backtester   *AutoBacktester
}

func (g *StrategyGenerator) Generate(ctx context.Context, conversation *Conversation) (*GeneratedStrategy, error) {
    prompt := g.promptBuilder.Build(ctx, conversation)
    response, err := g.provider.ChatCompletion(ctx, prompt)
    if err != nil {
        return nil, err
    }
    dslCode := extractDSLCode(response)
    if issues := g.compliance.Scan(dslCode); len(issues) > 0 {
        return nil, fmt.Errorf("code compliance: %v", issues)
    }
    metrics, err := g.backtester.Run(ctx, dslCode, conversation.StrategyContext.Parameters)
    if err != nil {
        return &GeneratedStrategy{DSLCode: dslCode, Warning: "回测失败: " + err.Error()}, nil
    }
    return &GeneratedStrategy{DSLCode: dslCode, BacktestMetrics: metrics}, nil
}
```

## 9. API

| RPC | 说明 |
|-----|------|
| `CreateConversation` | 新建 AI 策略对话 |
| `SendMessage` | 发送用户消息（stream 返回 AI 逐字输出 + 最终策略代码） |
| `GetConversation` | 获取对话历史 |
| `ListConversations` | 我的对话列表 |

## 10. 指标

| 指标 | 说明 |
|------|------|
| `ai_strategy_generation_total{result}` | 策略生成总数（success/failed/compliance_blocked） |
| `ai_clarification_rounds` | 追问轮次分布 histogram |
| `ai_backtest_trigger_total` | 自动回测触发次数 |
| `ai_conversation_duration_seconds` | 对话时长 |

## 11. 验收命令

```bash
# 1. E2E: 模糊描述 → 追问 → 策略 → 回测 → 迭代
go test -tags=e2e ./tests/e2e/ -run TestAIIterativeStrategy -v

# 2. 代码合规扫描器 13 条规则完整
grep -c "rule_" backend/internal/ai/code_compliance.go | grep -q "13"

# 3. 模板库至少 5 个模板
docker exec ant-postgres psql -U ant -t -c \
  "SELECT COUNT(*) FROM platform_strategies WHERE is_active=true" | grep -E "[5-9]|[1-9][0-9]"
```
