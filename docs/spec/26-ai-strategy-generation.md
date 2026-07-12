# 26 · AI 策略生成架构规范

> **关联 ADR**：ADR-0017
> **关联 spec**：`docs/spec/21-backtest-replay.md`、`docs/spec/23-risk-management.md`
> **参考项目**：QuantDinger（参数优化引擎设计借鉴）
> **最后更新**：2026-06-04（合并方案：ant NL→DSL + QuantDinger 参数优化）

## 1. 使命

「让普通人都能用上量化交易系统」——非专业散户通过自然语言描述交易想法，AI 生成可在回测/仿真/实盘环境中运行的量化策略代码。

目标用户画像：
- 有交易经验但无编程能力
- 知道自己想做什么（"突破买入"、"做波段"、"趋势跟踪"）
- 但不知道如何用代码表达

产出的策略代码运行在 ant 的 DSL 引擎上，走标准回测→仿真→实盘路径。

## 2. 四阶段流水线

```
用户自然语言输入
  │
  ▼
┌──────────────────────────────────────────────────────┐
│ Phase 1: 策略创作 (NL → DSL Code)                     │
│                                                      │
│  意图澄清引擎 (clarification.go)                       │
│    │  检测模糊词，追问 1-3 轮                           │
│    ▼                                                 │
│  会话记忆管理 (conversation_store.go)                   │
│    │  滑动窗口 20 条消息 + 策略上下文摘要                │
│    ▼                                                 │
│  模板选择器 (template_library.go)                      │
│    │  匹配策略模板（趋势/回归/突破/网格）                 │
│    ▼                                                 │
│  AI 生成 DSL 代码 (strategy_prompt.go + LLM)           │
│    │  System prompt: 中文策略生成 + DSL 语法             │
│    │  Context: 模板骨架 + 用户参数                       │
│    ▼                                                 │
│  代码合规扫描 (code_compliance.go, 13 条规则)            │
│    │                                                 │
│    ▼                                                 │
│  初步回测验证（快速 sanity check）                      │
│                                                      │
└──────────────────────┬───────────────────────────────┘
                       ▼
┌──────────────────────────────────────────────────────┐
│ Phase 2: 参数优化 (Smart Tuning)  ← 借鉴 QuantDinger   │
│                                                      │
│  市场环境检测 (regime.go)                               │
│    │  bull_trend / bear_trend / high_volatility       │
│    │  / range_compression / transition                │
│    ▼                                                 │
│  参数空间提取                                          │
│    │  从 @param 注解提取可调参数 + 范围                  │
│    ▼                                                 │
│  优化引擎 (optimizers.go)                              │
│    │  Grid Search / Random / DE / TPE                 │
│    │  多轮 AI 参数提议（温度递增 0.7 → 0.8）            │
│    ▼                                                 │
│  批量回测 + Regime-aware 评分 (scoring.go)              │
│    │  7 维加权评分，权重随 regime 调整                   │
│    ▼                                                 │
│  OOS 验证 (70/30 split)                               │
│    │  过拟合检测：OOS 退化 > 40% → 标记 overfit         │
│    ▼                                                 │
│  最优参数组合输出                                      │
│                                                      │
└──────────────────────┬───────────────────────────────┘
                       ▼
┌──────────────────────────────────────────────────────┐
│ Phase 3: 反馈迭代 (Human-in-the-loop)                  │
│                                                      │
│  回测结果注入对话 (backtest_feedback.go)                 │
│    │  用户: "太激进了，能不能保守一点"                    │
│    │  AI 检查 MaxDrawdown > 20%                        │
│    │  AI 调整 position_size → 重新生成                  │
│    ▼                                                 │
│  循环回 Phase 1 或 Phase 2（视反馈类型）                 │
│                                                      │
│  • 策略逻辑不满 → 回到 Phase 1（重新生成 DSL）           │
│  • 回测指标不满 → 回到 Phase 2（重新优化参数）            │
│                                                      │
└──────────────────────┬───────────────────────────────┘
                       ▼
┌──────────────────────────────────────────────────────┐
│ Phase 4: 质量门禁 (6-Gate) ✅ 已实现                    │
│                                                      │
│  Compliance → LookAhead → WalkForward                 │
│  → DeflatedSharpe → Paper → Correlation               │
│                                                      │
│  全部通过 → Go Live                                   │
└──────────────────────────────────────────────────────┘
```

## 3. Phase 1: 策略创作

### 3.1 意图澄清

```go
// internal/ai/clarification.go

type ClarificationRule struct {
    Keywords    []string
    Questions   []string
    ParamMap    map[string]string
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

### 3.2 会话记忆

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

### 3.3 策略模板库

```sql
CREATE TABLE platform_strategies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category VARCHAR(32) NOT NULL,       -- trend_following/mean_reversion/breakout/grid/martingale
    name VARCHAR(128) NOT NULL,
    description_zh TEXT NOT NULL,
    dsl_skeleton TEXT NOT NULL,           -- DSL 代码骨架，{PARAM} 为占位符
    parameter_slots JSONB NOT NULL,       -- [{name, type, default, min, max, description_zh}]
    risk_level VARCHAR(8) DEFAULT 'medium',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

AI 生成流程：选择模板 → 从用户描述填充参数 → 定制条件 → 代码合规扫描 → 回测验证。

### 3.4 代码生成质量门

| 门禁 | 说明 | 失败动作 |
|------|------|----------|
| code_compliance | 13 条禁止模式扫描 | 返回违规描述，要求用户修改意图 |
| DSL 语法验证 | `dsl.Parser.Parse()` 无错误 | 修复语法后重新生成 |
| 最少 bar 数 | 回测窗口 >= 1000 bars | 提示用户扩大回测时间范围 |
| 风控兼容 | 生成的策略参数不超过 risk_limits | 警告用户并建议调整 |

## 4. Phase 2: 参数优化（Smart Tuning）

> 设计借鉴 QuantDinger `backend_api_python/app/services/experiment/`，
> 调整为 ant 的 Go 后端 + DSL 回测引擎。

### 4.1 市场环境检测 (regime.go)

```go
// internal/ai/regime.go

type MarketRegime int

const (
    RegimeBullTrend       MarketRegime = iota  // EMA gap >= 1%, efficiency >= 0.55
    RegimeBearTrend                            // Same, negative price change
    RegimeHighVolatility                       // Realized vol >= 4.5% or ATR% >= 3.5%
    RegimeRangeCompression                     // EMA gap <= 0.45%, efficiency <= 0.38%
    RegimeTransition                           // Everything else
)

type RegimeResult struct {
    Regime       MarketRegime
    Confidence   float64
    Features     map[string]float64  // EMA gap, efficiency, vol, ATR%
    Recommended  []string            // e.g. ["trend_following", "breakout"]
}

type MarketRegimeService struct {
    symbolRepo   SymbolRepository
    barFetcher   BarFetcher
}

func (s *MarketRegimeService) Detect(ctx context.Context, symbol string) (*RegimeResult, error) {
    // Rule-based classification using recent price bars.
    // 5 regime types mapped to recommended strategy families.
}
```

### 4.2 参数空间提取

从 DSL 代码的 `@param` 注解中提取可调参数：

```
@param fast_period 10    range=5:50:5
@param slow_period 30    range=20:100:10
@param stop_loss 0.02    range=0.01:0.05:0.005
@strategy position_size 0.1
```

```go
// internal/ai/param_extractor.go

type TunableParam struct {
    Name    string
    Default float64
    Min     float64
    Max     float64
    Step    float64
    Type    string  // "int" | "float" | "choice"
    Choices []string
}

type RiskParams struct {
    PositionSize  float64
    StopLoss      float64
    TakeProfit    float64
    Leverage      float64
    TrailingStop  float64
}
```

### 4.3 优化引擎 (optimizers.go)

四合一优化器，依赖-free（纯 stdlib）：

| 引擎 | 说明 | 适用场景 |
|------|------|----------|
| **Grid Search** | 笛卡尔积 + shuffle | 参数少（≤4 维） |
| **Random Search** | 随机采样，固定种子(0xC0DE) | 参数多（>4 维） |
| **Differential Evolution** | rand/1/bin，离散参数 index 编码 | 中等预算（20-80 次） |
| **TPE** | 轻量 Tree-structured Parzen Estimator | 同上，更智能 |

```go
// internal/ai/optimizers.go

type Optimizer interface {
    Propose(params map[string]TunableParam, history []*OptimizationResult, budget int) []map[string]float64
}

type OptimizationResult struct {
    Params    map[string]float64
    Score     float64
    Sharpen   float64
    MaxDD     float64
    WinRate   float64
    Trades    int
    Grade     string  // A ~ E
}
```

### 4.4 AI 多轮参数提议（借鉴 QuantDinger）

在 Grid/Random 之外，AI 也可参与参数提议：

```go
// internal/ai/param_proposer.go

type AIRoundProposer struct {
    provider  AIProvider
    prompt    *ParamPromptBuilder
}

// ProposeParams generates N diverse parameter sets for one round.
// Temperature increases with round number to balance exploit/explore.
func (p *AIRoundProposer) ProposeParams(ctx context.Context, req *ProposeRequest) (*ProposeResponse, error) {
    // Build prompt with:
    //  - Indicator code (truncated to 4000 chars)
    //  - Tunable parameter definitions
    //  - Risk/position parameters
    //  - Market regime context
    //  - Previous round results (rounds 2+)
    // Temperature: 0.7 + round_num * 0.05
}
```

### 4.5 Regime-Aware 评分 (scoring.go)

```go
// internal/ai/scoring.go

type ScoringService struct {
    regimeWeights map[MarketRegime]*ScoreWeights
}

type ScoreWeights struct {
    Return       float64  // 默认 0.22
    AnnualReturn float64  // 默认 0.12
    Sharpe       float64  // 默认 0.18
    ProfitFactor float64  // 默认 0.14
    WinRate      float64  // 默认 0.09
    Drawdown     float64  // 默认 0.15
    Stability    float64  // 默认 0.10
}

// Regime-adjusted weights:
//   bull_trend:     Return↑(0.28), Sharpe↑(0.22), Drawdown↓(0.10)
//   bear_trend:     Drawdown↑(0.25), Stability↑(0.15)
//   high_vol:       Drawdown↑(0.30), WinRate↓(0.05)
//   range_compress: WinRate↑(0.18), Stability↑(0.15)
//   transition:     Default weights
```

评分等级：A (≥85)、B (≥72)、C (≥60)、D (≥45)、E (<45)

惩罚项：交易 < 5 次扣 12 分，< 12 次扣 5 分。

### 4.6 OOS 验证

```
In-Sample (70%)          Out-of-Sample (30%)
├──────────────────────┼────────────────────┤
  优化参数               验证泛化能力
                        退化 > 40% → overfit
```

```go
// internal/ai/oos_validator.go

type OOSValidator struct {
    splitRatio float64  // 0.7
    maxDegradation float64  // 0.4 — OOS score 不能比 IS 低超过 40%
}

func (v *OOSValidator) Validate(isScore, oosScore float64) *OOSResult {
    degradation := (isScore - oosScore) / isScore
    return &OOSResult{
        ISScore:      isScore,
        OOSScore:     oosScore,
        Degradation:  degradation,
        IsOverfit:    degradation > v.maxDegradation,
    }
}
```

## 5. Phase 3: 反馈迭代

### 5.1 回测反馈注入

```go
// internal/ai/backtest_feedback.go

type BacktestFeedback struct {
    Metrics *BacktestMetrics
    Summary string
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

### 5.2 反馈路由

```
用户输入 → 分类
  ├── 策略逻辑相关（"改成做多"、"加上止损"）
  │     → 回到 Phase 1（重新生成 DSL）
  │
  └── 回测指标相关（"太激进了"、"收益太低"）
        → 回到 Phase 2（重新优化参数）
```

```go
// internal/ai/feedback_router.go

type FeedbackRouter struct{}

func (r *FeedbackRouter) Route(userMessage string, lastMetrics *BacktestMetrics) FeedbackTarget {
    // Classify user intent → Phase1 | Phase2
    // "太激进/风险太大/回撤太多/降低仓位" → Phase2 (re-optimize)
    // "改成/加上/去除/换品种" → Phase1 (regenerate DSL)
}
```

## 6. Phase 4: 质量门禁 ✅ 已实现

> 已实现，见 `backend/internal/ai/gate_pipeline.go` 及相关文件。

| 关 | 名称 | 检查内容 |
|----|------|----------|
| 1 | Compliance | DSL 合规检查 |
| 2 | LookAhead | 前视偏差扫描 |
| 3 | WalkForward | 步进前向分析 + CPCV |
| 4 | DeflatedSharpe | 多重检验校正后的 DSR |
| 5 | Paper | 仿真交易 ≥ 14 天，Net P&L > 0 |
| 6 | Correlation | 与现有实盘策略的相关性 < 0.7 |

全部通过 → Go Live。

## 7. AI Provider 集成

复用现有 `internal/ai/` 层（12 家 provider 抽象）。

| 阶段 | Temperature | 原因 |
|------|-------------|------|
| Phase 1 (DSL 生成) | 0.3 | 代码需要确定性 |
| Phase 2 (参数提议) | 0.7 + round*0.05 | 平衡探索与利用 |
| Phase 3 (反馈理解) | 0.7 | 自然语言理解 |

```go
type StrategyGenerator struct {
    provider      AIProvider
    promptBuilder *StrategyPromptBuilder
    compliance    *CodeComplianceScanner
    backtester    *AutoBacktester
    tuner         *SmartTuningEngine
    feedback      *FeedbackRouter
}

func (g *StrategyGenerator) Generate(ctx context.Context, conversation *Conversation) (*GeneratedStrategy, error) {
    // Phase 1: NL → DSL
    dslCode, err := g.generateDSL(ctx, conversation)
    if err != nil {
        return nil, err
    }

    // Phase 2: Smart Tuning
    optResult, err := g.tuner.Optimize(ctx, dslCode, conversation.StrategyContext)
    if err != nil {
        // Non-fatal: return strategy with default params
        return &GeneratedStrategy{DSLCode: dslCode, Warning: "optimization failed: " + err.Error()}, nil
    }

    return &GeneratedStrategy{
        DSLCode:         dslCode,
        OptimalParams:   optResult.Params,
        BacktestMetrics: optResult.Metrics,
        OOSResult:       optResult.OOS,
        Regime:          optResult.Regime,
    }, nil
}
```

## 8. API

| RPC | 说明 | 阶段 |
|-----|------|------|
| `CreateConversation` | 新建 AI 策略对话 | Phase 1 |
| `SendMessage` | 发送用户消息（SSE stream：逐字输出 + 最终策略代码） | Phase 1 |
| `GetConversation` | 获取对话历史 | Phase 1 |
| `ListConversations` | 我的对话列表 | Phase 1 |
| `RunSmartTuning` | 对指定策略代码运行参数优化（SSE stream：每轮进度） | Phase 2 |
| `DetectRegime` | 检测当前市场环境 | Phase 2 |
| `RunGateEvaluation` | 运行 6-Gate 质量门禁 ✅ 已实现 | Phase 4 |

### SSE 事件流（Smart Tuning）

```
event: round_start    → {round: 1, candidates: 5}
event: candidate_done → {round: 1, candidate: 3, score: 78, grade: "B"}
event: candidate_done → {round: 1, candidate: 5, score: 84, grade: "A"}
event: round_summary  → {round: 1, best_score: 84, avg_score: 72}
event: oos_start      → {candidates: 3}
event: oos_done       → {degradation: 0.12, is_overfit: false}
event: complete       → {best_params: {...}, is_score: 84, oos_score: 74}
```

## 9. 数据库

```sql
-- Phase 1
CREATE TABLE ai_conversations (...);  -- 见 §3.2
CREATE TABLE platform_strategies (...);  -- 见 §3.3
CREATE TABLE ai_clarification_rules (...);  -- ADR-0017

-- Phase 2
CREATE TABLE optimization_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id UUID REFERENCES platform_strategies(id),
    dsl_code TEXT NOT NULL,
    regime VARCHAR(32),
    in_sample_start TIMESTAMPTZ NOT NULL,
    in_sample_end TIMESTAMPTZ NOT NULL,
    oos_start TIMESTAMPTZ NOT NULL,
    oos_end TIMESTAMPTZ NOT NULL,
    rounds JSONB,         -- [{round, candidates: [{params, score, grade}], best}]
    best_params JSONB,
    best_is_score DECIMAL(5,1),
    best_oos_score DECIMAL(5,1),
    degradation DECIMAL(4,3),
    is_overfit BOOLEAN,
    status VARCHAR(16) DEFAULT 'running',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    finished_at TIMESTAMPTZ
);

-- Phase 4 ✅ 已有 qd_analysis_memory / qd_ai_calibration（如需要）
```

## 10. 指标

| 指标 | 说明 |
|------|------|
| `ai_strategy_generation_total{result}` | Phase 1 策略生成总数 |
| `ai_clarification_rounds` | 追问轮次分布 histogram |
| `ai_tuning_rounds_total` | Phase 2 优化轮次 |
| `ai_tuning_candidates_total` | Phase 2 候选参数组数 |
| `ai_tuning_oos_degradation` | OOS 退化率分布 |
| `ai_backtest_trigger_total` | 回测触发次数 |
| `ai_gate_pass_rate` | Phase 4 通过率 |
| `ai_conversation_duration_seconds` | 对话时长 |

## 11. 验收命令

```bash
# 1. E2E: 模糊描述 → 追问 → 策略 → 优化 → 反馈 → 迭代
go test -tags=e2e ./tests/e2e/ -run TestAIIterativeStrategy -v

# 2. Smart Tuning: 参数优化 + OOS 验证
go test -tags=e2e ./tests/e2e/ -run TestSmartTuning -v

# 3. 代码合规扫描器 13 条规则完整
grep -c "rule_" backend/internal/ai/code_compliance.go | grep -q "13"

# 4. 模板库至少 5 个模板
docker exec alphaforge-postgres psql -U ant -t -c \
  "SELECT COUNT(*) FROM platform_strategies WHERE is_active=true" | grep -E "[5-9]|[1-9][0-9]"

# 5. 优化引擎 4 种模式
grep -c "Grid\|Random\|DE\|TPE" backend/internal/ai/optimizers.go | grep -q "4"
```

## 12. 实施优先级

| 优先级 | 组件 | 依赖 |
|--------|------|------|
| P0 | Phase 1: clarification.go + conversation_store.go + template_library.go | 无（基础设施已就绪） |
| P0 | Phase 1: strategy_prompt.go + GenerateStrategy RPC | SystemAI（已实现） |
| P1 | Phase 2: regime.go + scoring.go | 回测引擎（已实现） |
| P1 | Phase 2: optimizers.go + param_extractor.go | Phase 1 产出 |
| P1 | Phase 2: Smart Tuning SSE API | Phase 2 核心 |
| P2 | Phase 3: backtest_feedback.go + feedback_router.go | Phase 1 + 2 |
| P3 | Phase 4: ✅ 已实现 | — |

## 8. 实现状态 (2026-06-05)

### 已完成

**架构统一（全链路 proto binary）**：
| 层 | 格式 | 状态 |
|---|------|------|
| 传输 | ConnectRPC proto binary | ✅ |
| 存储 | BYTEA proto binary (proto_response, StrategyParams, ParameterSpace 等) | ✅ |
| NATS | TickPayload, TradeEventPayload, AccountEventPayload proto | ✅ |
| Redis | OrderCacheEntry proto | ✅ |

**价格精度（全栈 decimal.Decimal）**：
- 模型层 9 文件 112 字段 float64→decimal.Decimal
- 仓库层 + mthub + risksvc + costsvc + backtest 全链路适配
- 边界转换: decimal→proto float64 (InexactFloat64)

**Push-first（零轮询）**：
- WatchBacktestRun / WatchExperiment / WatchSchedules: SSE streaming
- 服务端 LISTEN/NOTIFY 事件驱动（延迟 <1ms，DB 查询减少 98%）
- 前端: SmartTuningPanel + SchedulePage 全部 SSE 订阅

**Phase 1: NL → Go 策略代码**：
- 澄清引擎 (clarification.go)
- 模板库 (template_library.go, 5 个种子模板)
- AI prompt 构建 (strategy_prompt.go)
- 合规扫描 13 条规则 (code_compliance.go)
- 策略生成 ConnectRPC handler (strategy_gen_handler.go)

**Phase 2: Smart Tuning 参数优化**：
- 参数提取 regex (param_extractor.go)
- Grid Search (组合索引 O(candidates×dims))
- Random Search (固定种子 0xC0DE)
- DE 优化器 (rand/1/bin, math.Round, popSize=10*dims capped 50)
- AnnealedGaussianSearch (高斯抖动+退火)
- AI 多轮参数提议 (param_proposer.go, AIProposer 接口)
- Regime 检测 (自适应阈值, Spearman 秩相关)
- 7 组件 regime-aware 评分 (scoring.go)
- OOS 验证 70/30 时间分割

**Phase 3: 反馈闭环**：
- Backtest 结果注入 AI prompt (backtest_feedback.go)
- NL 反馈路由 (feedback_router.go)

**Phase 4: Gate 评估**：
- 过拟合检测
- Walk-forward 验证
- 权益曲线分析

### 优化记录

| 日期 | 优化 | 效果 |
|------|------|------|
| 2026-06-05 | Grid Search 递归→组合索引 | O(product)→O(candidates×dims) |
| 2026-06-05 | SSE 2s ticker→LISTEN/NOTIFY | 延迟 2s→<1ms |
| 2026-06-05 | DE popSize cap 12→50 | 高维空间进化效率提升 |
| 2026-06-05 | Stability Pearson r→Spearman ρ | 权益单调性度量最优解 |
| 2026-06-05 | TPE→AnnealedGaussianSearch | 算法正名, 移除误导 |
| 2026-06-04 | JSON→proto 全线重构 | 消除 16 个 JSON 使用点 |
| 2026-06-04 | failRun uuid.Nil 修复 | SQL UPDATE 不再静默失败 |
| 2026-06-04 | ClaimNextForWork 缺 parameter_overrides | DE 分数从相同→有区分度 |
| 2026-06-06 | LLM Client 实现 + AIProposer 注入 | AI 管线全链路打通: NL→LLM→代码→回测→优化 |

### 待做

- strategy_schedules/templates Go 代码 proto 适配 (部分完成)
- Backtest legacy JSONB 列清理
