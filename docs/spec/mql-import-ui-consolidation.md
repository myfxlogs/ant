# Spec: MQL Import UI 统一管线

- **状态**：Proposed
- **日期**：2026-07-24
- **关联 ADR**：ADR-0023（MQL 源码为唯一真实来源）、ADR-0024（Agent-Native 策略平台）

## 1. 问题陈述

### 1.1 当前状态

MQL Import UI（`ImportEAPanel.tsx`）当前暴露 2 个按钮 + 1 个条件按钮：

| 按钮 | 后端 RPC | 做什么 | 持久化? |
|------|---------|--------|---------|
| 分析策略结构 | `AnalyzeImportCode` | 静态分析：覆盖率 + 盲区 + 参数 | ❌ |
| 确认导入 (覆盖率≥40%) | `ImportStrategy` | 编译 IR + 持久化到 DB + 返回原始 MQL | ✅ |
| 盲区桥接 (覆盖率<40% 或主动) | `SubmitStrategy` | 编译 + 回测 + LLM 画像 + LLM 分析 + LLM 桥接 | ❌ |

### 1.2 核心矛盾

**两条后端管线语义重叠但能力互补**：

- `ImportStrategy`（`StrategyExecutionServer`）：
  - 做了：编译 IR、持久化到 `imported_strategies`、创建版本快照
  - 没做：回测、LLM 分析、盲区桥接
  - 返回 `goCode = source`（原始 MQL 原样返回，ADR-0023 D1: MQL 是唯一真实来源）
  - **"应用到编辑器"是把用户刚粘贴的代码原样设回编辑器 — 空操作**

- `SubmitStrategy`（`GatewayServer`）：
  - 做了：编译 + 回测 + LLM 策略画像 + LLM 回测分析 + LLM 盲区桥接
  - 没做：持久化到 `imported_strategies`（`strategyId = uuid.New()` 随机生成，不入库）
  - 返回 `bridgedPythonSource`（LLM 翻译的 Python 子集代码）

**用户被迫理解内部架构**才能选择正确的按钮，但"确认导入"和"盲区桥接"的区别是实现细节，不是用户关心点。

### 1.3 额外问题

1. `ImportStrategy` 的 `goCode` 字段返回原始 MQL，前端 `onApplyCode(res.goCode)` 把用户刚粘贴的代码原样写回编辑器 — **空操作**（用户代码已在 textarea 里）
2. `GenerateImportCode` handler 已是 `Unimplemented` stub，proto 定义仍存在
3. `SubmitStrategy` 不存库 → 桥接成功的策略丢失，用户无法后续管理

## 2. 设计目标

1. **单入口**：用户只看到一个动作 — "导入策略"
2. **零空操作**：每个后端调用都有不可替代的价值
3. **自动分流**：高覆盖率直接导入，低覆盖率自动桥接 — 用户不需要选择
4. **全链持久化**：导入结果存库，用户可后续管理
5. **符合 ADR-0023/0024**：MQL 是唯一真实来源，Agent 桥接是 ADR-0024 规定的盲区处理路径

## 3. 方案

### 3.1 UI 交互流程

```
用户粘贴 MQL 代码
  ↓
[导入策略] ← 唯一按钮
  ↓
后端 SubmitStrategy:
  Step 1: 编译 + 覆盖率分析
  Step 2: 回测 (VM + SimBroker)
  Step 3: LLM 策略画像
  Step 4: LLM 回测分析
  Step 5: 盲区桥接 (覆盖率 < 100% 时自动触发)
  Step 6: 持久化到 imported_strategies ← 新增
  ↓
前端展示:
  ├─ 分析报告 (覆盖率 + 盲区 + 参数)
  ├─ 回测结果 (收益/回撤/夏普)
  ├─ 桥接 diff (如有)
  └─ strategyId (已存库)
```

### 3.2 后端变更

#### 3.2.1 `SubmitStrategy` 增加持久化 (Step 7)

在 `gateway.go` 的 `SubmitStrategy` handler 中，Step 6（盲区桥接）之后新增：

```go
// ── Step 7: Persist to imported_strategies ──
if s.importedRepo != nil {
    row := &repository.ImportedStrategy{
        UserID:        userID,
        Name:          strategyName,
        SourceLang:    language,
        SourceCode:    sourceCode,
        Params:        interp.SerializeParams(extractParams(coverage)),
        CoverageScore: coverage.Score,
    }
    if err := s.importedRepo.Create(ctx, row); err != nil {
        s.log.Warn("SubmitStrategy: persist failed", zap.Error(err))
    } else {
        strategyID = row.ID.String()
        // Create version snapshot
        if s.versionRepo != nil {
            s.versionRepo.CreateVersion(ctx, row.ID, uid, sourceCode, language, "Agent import")
        }
    }
}
```

**需要**：`GatewayServer` 新增 `importedRepo` 和 `versionRepo` 字段，在 `NewGatewayServer` 中注入。

#### 3.2.2 `ImportStrategy` handler 标记废弃

`ImportStrategy` 的唯一独有价值是持久化。一旦 `SubmitStrategy` 也持久化，`ImportStrategy` 完全冗余。

处置：保留 handler 但标记废弃注释，前端不再调用。后续 proto 清理时移除。

#### 3.2.3 `GenerateImportCode` proto 定义移除

`GenerateImportCode` handler 已是 `Unimplemented` stub。移除 proto 定义和生成代码。

### 3.3 前端变更

#### 3.3.1 `ImportEAPanel.tsx` 简化

**删除**：
- `importing` / `applied` 状态
- `handleConfirmImport` 函数
- `strategyImportApi.importStrategy` 调用
- "确认导入"按钮
- "盲区桥接"按钮 + 参数面板
- `bridgeSymbol` / `bridgeTimeframe` / `showBridgeParams` 状态

**保留**：
- "分析策略结构"按钮 → 改为自动触发或保留为预览
- `AnalyzeImportCode` 调用（轻量预览，不回测不桥接）

**新增**：
- "导入策略"按钮 → 调用 `submitStrategy`
- 回测结果展示区域
- 桥接参数（symbol/timeframe）移到导入按钮旁，作为可选配置

#### 3.3.2 `strategy.ts` 清理

删除 `importStrategy` 方法（前端不再调用）。

### 3.4 数据流对比

```
Before (3 按钮, 2 后端管线):
  用户 → AnalyzeImportCode → 展示报告
  用户 → ImportStrategy → 空操作(goCode=source) + 持久化
  用户 → SubmitStrategy → 回测+LLM+桥接 → 不持久化

After (1 按钮, 1 后端管线):
  用户 → AnalyzeImportCode → 展示报告 (轻量预览, 可选)
  用户 → SubmitStrategy → 回测+LLM+桥接+持久化 → 展示完整结果
```

## 4. 方案验证：是否为最优解？

### 4.1 对照 ADR-0023

| ADR-0023 决策 | 本方案 |
|---|---|
| D1: MQL 源码是唯一真实来源 | ✅ `imported_strategies.source_code` 存 MQL 原文 |
| D4: 去掉 Go 代码生成 | ✅ `ImportStrategy.goCode = source` 是 D4 的遗留产物，本方案消除 |
| D5: Agent 架构 | ✅ `SubmitStrategy` 是 ADR-0024 的 Agent Gateway |

### 4.2 对照 ADR-0024

| ADR-0024 决策 | 本方案 |
|---|---|
| D5: Agent 层架构 | ✅ `SubmitStrategy` 是 Agent Gateway 的核心 RPC |
| D6: ConnectRPC Agent Gateway | ✅ 统一入口为 `SubmitStrategy` |
| §5.3 六个 LLM 注入点 | ✅ 注入点 [1]-[6] 都在 `SubmitStrategy` 内 |

### 4.3 备选方案分析

| 方案 | 描述 | 优点 | 缺点 | 结论 |
|------|------|------|------|------|
| **A: 统一到 SubmitStrategy** (本方案) | SubmitStrategy 增加持久化，前端只调一个 RPC | 单入口、零空操作、全链持久化 | GatewayServer 需注入 repo | ✅ 最优 |
| B: 统一到 ImportStrategy | ImportStrategy 增加回测+LLM | 不改 Gateway | ImportStrategy 在 StrategyExecutionServer，没有 LLM/回测能力，需大改 | ❌ 违反服务边界 |
| C: 保持双管线，修复空操作 | ImportStrategy 不再返回 goCode，只返回 strategyId | 改动最小 | 用户仍需选按钮，SubmitStrategy 仍不持久化 | ❌ 治标不治本 |
| D: 前端编排两个 RPC | 前端先调 ImportStrategy 拿 strategyId，再调 SubmitStrategy 做回测 | 后端不改 | 两次 RPC 往返、strategyId 关联复杂、用户仍看到两个按钮 | ❌ 前端复杂度增加 |

**方案 A 是最优解**：
1. 改动集中在 `GatewayServer`（加 2 个 repo 字段 + 1 个 Step），不跨服务边界
2. `SubmitStrategy` 已经做了全部有价值的计算（编译+回测+LLM+桥接），只差持久化
3. 前端从 3 按钮 + 条件分支 → 1 按钮，大幅简化
4. 消除 `ImportStrategy.goCode = source` 空操作

### 4.4 风险

| 风险 | 影响 | 缓解 |
|------|------|------|
| SubmitStrategy 耗时较长 (LLM 调用 5-30s) | 用户等待体验 | 前端显示分步进度（编译→回测→分析→桥接）；后续可改 SSE 流式 |
| SubmitStrategy 失败时无 strategyId | 用户无法重试 | 失败时仍返回覆盖率报告，用户可修改代码后重试 |
| ImportStrategy 废弃后旧数据无影响 | 已导入的策略仍在 DB | 不迁移数据，只停止新增 |

## 5. 实施计划

1. **后端**：`GatewayServer` 注入 `importedRepo` + `versionRepo`，`SubmitStrategy` 增加 Step 7 持久化
2. **前端**：`ImportEAPanel.tsx` 重构为单按钮，`strategy.ts` 删除 `importStrategy`
3. **清理**：移除 `GenerateImportCode` proto 定义 + 生成代码
4. **验证**：tsc + go build + 前端 build + 部署

## 6. 不做的事

- 不改 `AnalyzeImportCode`（轻量预览有价值，保留）
- 不改 `SubmitStrategy` 的 LLM/回测/桥接逻辑（已验证可用）
- 不迁移 `ImportStrategy` 的历史数据
- 不改 proto 的 `ImportStrategy` 定义（handler 保留 stub，后续清理）
