# 策略市场 · 施工入口

> **目标读者**：GLM（执行端）
> **角色**：你负责按本文档指明的顺序和约束，逐模块完成落地实现。
> **架构决策已定，你不需要自行设计。**

---

## 文档结构

```
docs/roadmaps/strategy-marketplace.md          ← 📐 设计文档（产品边界、收入模型、风险、Phase 摘要）
docs/blocks/strategy-marketplace/
  ├── README.md                                ← 块入口（代码位置、依赖、设计决策）
  └── plans/
        ├── README.md                          ← 🚪 你在这里
        ├── seo-strategy.md                    ← 🔍 SEO 策略
        ├── phase-1-trust-infrastructure.md    ← 📋 Phase 1
        ├── phase-2-ai-strategy-supply.md      ← 📋 Phase 2
        ├── phase-3-growth-engine.md           ← 📋 Phase 3
        ├── phase-4-platform-ops.md            ← 📋 Phase 4
        └── phase-5-moat.md                    ← 📋 Phase 5
```

---

## 执行顺序

```
施工前清理（下线 copytrade）→ Phase 1 → Phase 2 → Phase 3 → [Phase 4 ‖ Phase 5]
```

- **Phase 1 必须先完成**：信任基础设施 + SEO 基础（S1 Seo.tsx 补全 + S3 关键词落地页 + /brokers 页）
- **Phase 2 依赖 Phase 1.2（质量门槛）和 1.5（价格模型修复）**
- **Phase 3 依赖 Phase 1（实盘数据）和 Phase 2（策略供给）** + SEO 深化（S2 策略详情页 + S4 sitemap/JSON-LD）
- **Phase 4 和 Phase 5 可并行**：互不依赖

---

## ⚠️ 施工前清理（开工前必须先做）

按照设计文档 v4 产品边界，跟单**不在产品范围内**。以下代码/表/RPC 必须物理删除：

- [ ] **确认 copytrade 无外部依赖**：grep 所有 import `alphaforge/internal/marketplace`，确认无模块引用 `CopyTradeEngine`、`NewCopyTradeEngine`、`CopySignalEvent`
- [ ] **删除 `backend/internal/marketplace/copytrade.go`**
- [ ] **删除 copytrade 相关 DB 表**：新 migration `DROP TABLE IF EXISTS copytrade_signals, copy_trade_links CASCADE;`
- [ ] **删除 copytrade 相关 proto 定义**（如有 RPC），`buf generate`
- [ ] **清理 connect 层引用**（如有 copytrade handler）
- [ ] **Gate**：`go build ./...` + `go test ./...` 通过

> 需要时 `git checkout` 恢复。留着 = 持续编译和维护成本。

---

## 每个 Phase 的施工流程

1. **读 Phase 文档「目标与边界」「依赖图」**——理解本 Phase 要达成什么
2. **Phase 1 和 Phase 3 含 SEO 模块（S1-S4）**——嵌入在文档末尾，与其他模块并行，不要漏掉
3. **按模块顺序执行**——每个模块有 `前置依赖` 标注
4. **每个模块做完 → 跑 Gate**——Gate 命令写在模块末尾
5. **整个 Phase 做完 → 跑「Phase 完成检验」**——命令写在 Phase 文档末尾
6. **git 管理进度**：每完成一个模块 commit 一次，格式 `feat(marketplace): [Phase X] 模块名`

---

## 施工约束（全 Phase 通用）

### 架构约束
- 🔴 **禁用 timer/ticker/polling/cron**：全部事件驱动（NATS/PG NOTIFY）或惰性求值。感觉需要定时任务 → 停下来，设计有问题
- 🔴 **Proto only**：跨进程数据交换用 protobuf，禁止 JSON
- 🔴 **Decimal only**：价格/金额用 `decimal.Decimal`，禁止 float64
- 🔴 **ConnectRPC + SSE**：所有 API 用 ConnectRPC，实时推送用 server-stream SSE
- 🔴 **无 REST endpoint**：除 healthz/readyz/livez/metrics

### 代码约束
- Go 文件 ≤ 300 行（软性）/ ≤ 450 行（硬性红线）
- TS 文件 ≤ 250 行 / ≤ 375 行
- 函数 ≤ 50 行
- 禁止 `//nolint`、`# noqa`、`// @ts-ignore`

### 复用约束
- 动工任何新 file/function 前跑 `bash scripts/cap.sh <动词>`
- PR 描述给出 `REUSE:` 或 `NEW:` 结论

---

## 启动检查清单

- [ ] `go build ./...` 通过
- [ ] `cd backend && go run ./tools/check-file-lines --strict` 通过
- [ ] `bash scripts/gen_capability_map.sh` 已刷新
- [ ] DB migration 序号确认（选下一个未用的）
- [ ] 理解「push-first」约束

---

## 施工开始

→ **先读 `docs/roadmaps/strategy-marketplace.md`（设计文档）和 `docs/blocks/strategy-marketplace/README.md`（块入口），再打开 `phase-1-trust-infrastructure.md` 开始施工。**
