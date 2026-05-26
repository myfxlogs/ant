# ADR-0017 · AI 会话记忆 + 意图澄清 + 回测反馈

- **状态**：Accepted
- **日期**：2026-05-24
- **决策者**：架构组
- **关联 spec**：`docs/spec/26-ai-strategy-generation.md`、`docs/spec/21-backtest-replay.md`
- **关联 ADR**：ADR-0012

## 1. 背景

项目使命：「让普通人都能用上量化交易系统」——非专业散户通过自然语言描述交易想法，AI 生成专业量化策略代码。

当前 AI 层（评估 B+ / 72%）已有：
- 12 家 AI provider 抽象层
- `strategy_prompt.go`（中文 system prompt + MQL 风格函数库）
- `symbol_extractor.go`（~50 个中英品种映射）
- `code_compliance.go`（13 条禁止模式检测）

但缺少四个关键能力：
1. **会话记忆**：每次对话独立，无上下文延续
2. **意图澄清**：用户输入模糊（"做一个稳健的策略"）时无追问机制
3. **回测反馈**：策略生成是一次性的，不学习之前生成的质量
4. **策略模板库**：每次都从零开始生成

## 2. 决策

### 2.1 会话记忆

使用滑动窗口 + 策略上下文注入 system prompt（不做模型微调）：

```
[System Prompt: Chinese strategy generation + MQL function reference]
[Context: 策略模板库索引]
[Session Memory (sliding window N messages)]
[Strategy Context: 当前策略 DSL 代码 + 最近一次回测关键指标]
[User Message]
```

存储：PG `ai_conversations` 表（`user_id, conversation_id, messages JSONB, strategy_context JSONB, created_at`）。

### 2.2 意图澄清

模糊词触发追问的规则表：

| 模糊词 | 澄清问题 | 映射参数 |
|--------|----------|----------|
| "稳健"/"保守" | "最大回撤容忍度？持仓周期偏好？" | max_drawdown ≤ 10%, period ≥ 1h |
| "进攻型"/"激进" | "可接受的最大回撤？是否允许日内交易？" | max_drawdown ≤ 30%, period ≤ 15m |
| "做波段" | "波段周期？持仓时间？单品种还是多品种？" | holding_period, symbol_scope |

最多 3 轮追问，之后使用默认值进入生成。规则存储在 `ai_clarification_rules` 配置表。

### 2.3 回测反馈

策略生成后自动运行回测（ReplaySource + PaperExecutor，见 ADR-0012/0015），回测结果注入下一轮对话：

```
用户: "太激进了，能不能保守一点"
  → AI 查看最近一次回测指标: Sharpe 1.2, max_drawdown 35%
  → AI 理解"太激进" = max_drawdown 太高
  → 调整参数: 降低仓位 → 重新生成 → 重新回测
  → 呈现新指标: Sharpe 0.9, max_drawdown 12%
```

### 2.4 策略模板库

`platform_strategies` 表（平台共享，无 `user_id`）：

| 类别 | 模板示例 | 参数槽位 |
|------|----------|----------|
| 趋势跟踪 | 双均线交叉 | fast_period, slow_period, symbol |
| 均值回归 | 布林带突破 | period, std_dev, symbol |
| 突破 | Donchian Channel | period, symbol |
| 网格 | Grid Trading | grid_count, spacing, symbol |
| 马丁 | Martingale | multiplier, levels, symbol（标注高风险） |

AI 生成流程：选择模板 → 从用户描述填充参数 → 定制条件 → 代码合规扫描 → 回测验证。

## 3. 备选方案

| 方案 | 否决理由 |
|------|----------|
| 微调量化专用模型 | 成本高、迭代慢、供应商锁定；当前阶段过度投资 |
| 纯规则模板选择（无 AI）| 不满足"自然语言→策略"核心使命 |
| 无限轮次对话 | token 成本失控；3 轮上限平衡用户体验和成本 |

## 4. 后果

- **正面**：策略生成质量通过迭代显著提升；用户可以"对话式"优化策略
- **负面**：token 消耗增加（长 system prompt + 滑动窗口 + 回测指标）
- **中性**：`ai_conversations` 表增长需定期清理（TTL 90 天）

## 5. 实施约束

1. `internal/ai/conversation_store.go`：PG 会话存储
2. `internal/ai/clarification.go`：模糊词检测 + 追问规则引擎
3. `internal/ai/backtest_feedback.go`：回测结果 → prompt 注入
4. `internal/ai/template_library.go`：模板选择 + 参数填充
5. PG migration：`ai_conversations` + `ai_clarification_rules` 表
6. Seed：`platform_strategies` 初始模板数据（5-8 个）
7. spec/26 详细规范

## 6. 验证方式

```bash
# E2E: 模糊描述 → 2 轮追问 → 策略生成 → 自动回测 → "太激进" → 降低仓位 → 回测改善
go test -tags=integration ./tests/e2e/ -run TestAIIterativeStrategy -v
```
