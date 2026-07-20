# 策略市场 · 施工入口

> **目标读者**：GLM（执行端）
> **角色**：你负责按本文档指明的顺序和约束，逐模块完成落地实现。
> **架构决策由本文档 + 各 Phase 实施文档共同给出，你不需要自行设计。**

---

## 文档结构（三层）

```
docs/roadmaps/strategy-marketplace.md          ← 📐 设计文档（架构、DB、API、决策、指标）
docs/plan/marketplace/
  ├── README.md                                ← 🚪 施工入口（执行顺序、约束、启动检查清单）
  ├── phase-1-trust-infrastructure.md          ← 📋 实施细则: 5 模块 · 48 任务
  ├── phase-2-ai-strategy-supply.md            ← 📋 实施细则: 4 模块 · 28 任务
  ├── phase-3-growth-engine.md                 ← 📋 实施细则: 5 模块 · 26 任务
  ├── phase-4-platform-ops.md                  ← 📋 实施细则: 6 模块 · 28 任务
  └── phase-5-moat.md                          ← 📋 实施细则: 4 模块 · 16 任务
```

**三层关系**：
| 层级 | 文档 | 角色 | 读它做什么 |
|------|------|------|-----------|
| 📐 设计 | `docs/roadmaps/strategy-marketplace.md` | 架构总纲 | 理解市场定位、DB 表结构、Phase 间关系、指标目标 |
| 🚪 入口 | `docs/plan/marketplace/README.md` | 施工指引 | 知道执行顺序、全局约束、启动条件 |
| 📋 细则 | `docs/plan/marketplace/phase-*.md` | 落地清单 | 逐条执行任务，每个任务有文件路径+代码+验收标准 |

---

## 执行顺序（唯一，不可并行 Phase）

```
Phase 1 → Phase 2 → Phase 3 → [Phase 4 ‖ Phase 5]
```

- **Phase 1 必须先完成**：信任基础设施是所有后续 Phase 的前提（没有实盘跟踪就没有排行榜，没有质量门槛不能 AI 批量生成）。
- **Phase 2 依赖 Phase 1.2（质量门槛）和 1.5（价格模型修复）**。
- **Phase 3 依赖 Phase 1（实盘数据）和 Phase 2（策略供给）**。
- **Phase 4 和 Phase 5 可并行**：互不依赖。

---

## 每个 Phase 的施工流程

1. **读 Phase 文档开头的「目标与边界」「依赖图」**——理解本 Phase 要达成什么、模块间先后关系。
2. **按模块顺序执行**：每个模块标注了 `前置依赖`，按照依赖图中的顺序施工。
3. **每个模块做完 → 跑 Gate 检验**：Gate 命令写在每个模块末尾。
4. **整个 Phase 做完 → 跑「Phase 完成检验」**：命令写在 Phase 文档末尾。
5. **用 git 管理进度**：每完成一个模块 commit 一次，标题格式 `feat(marketplace): [Phase X] 模块名`。

---

## 施工约束（全 Phase 通用）

### 架构约束
- 🔴 **禁用 timer/ticker/polling/cron**：全部用事件驱动（NATS/PG NOTIFY）或惰性求值。如果某个需求看起来"需要定时任务"，停下来——一定是设计有问题。
- 🔴 **Proto only**：所有跨进程数据交换用 protobuf，禁止 JSON。
- 🔴 **Decimal only**：所有价格/金额用 `decimal.Decimal`，禁止 float64。
- 🔴 **ConnectRPC + SSE**：所有 API 用 ConnectRPC，实时推送用 server-stream SSE。
- 🔴 **无 REST endpoint**：除了 healthz/readyz/livez/metrics。

### 代码约束
- Go 文件 ≤ 300 行（软性）/ ≤ 450 行（硬性红线）
- TS 文件 ≤ 250 行（软性）/ ≤ 375 行（硬性红线）
- 函数 ≤ 50 行
- 禁止 `//nolint`、`# noqa`、`// @ts-ignore`

### 复用约束
- 动工任何新 file/function 前，跑 `bash scripts/cap.sh <动词>` 确认无现成能力。
- 每个 PR 描述里给出 `REUSE:` 或 `NEW:` 结论。

---

## 启动检查清单

施工前逐条确认：

- [ ] `go build ./...` 通过（当前 main 分支）
- [ ] `cd backend && go run ./tools/check-file-lines --strict` 通过
- [ ] `bash scripts/gen_capability_map.sh` 已刷新
- [ ] 数据库 migration 序号确认（选下一个未用的序号）
- [ ] 理解「push-first」约束的含义

---

## 施工开始

→ **现在打开 `phase-1-trust-infrastructure.md`，从模块 1.5（价格模型 BUG 修复）开始。**
