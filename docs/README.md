# ant 文档

> v2 文档围绕「MetaTrader 作为量化数据基础」设计。策略执行架构由 ADR-0023 定义（MQL → AST → Bytecode VM 进程内执行）。

---

## 阅读路径

| 角色 | 推荐阅读顺序 |
|---|---|
| **新工程师 onboarding** | `architecture/01-vision.md` → `architecture/02-overview.md` → `architecture/03-data-flow.md` |
| **策略管线理解** | `go-native-strategy-pipeline.md` → `adr/0023` → `adr/0022` → `adr/0021` |
| **MT 适配实现者** | `spec/10-mt-adapter.md` → `spec/16-mtapi-quirks-register.md` → `adr/0003` |
| **行情网关实现者** | `spec/11-mdgateway.md` → `spec/13-clickhouse-schema.md` → `adr/0004` `0005` |
| **会话/下单实现者** | `spec/12-mthub.md` → `spec/14-rpc-contracts.md` |
| **运维/SRE** | `spec/15-observability.md` → `spec/20-slo.md` → `runbook/mt-incidents.md` |
| **数据归属（C2C 多租户）** | `adr/0006-platform-shared-vs-user-private.md` |
| **决策审计** | `adr/` 全部按编号读 |

---

## 目录结构

```
docs/
├── README.md                    本文件
├── CAPABILITIES.md              能力索引（自动生成）
├── go-native-strategy-pipeline.md   策略管线全景（MQL → Bytecode VM → 实盘执行）
├── architecture/
│   ├── 01-vision.md             设计哲学：MT = 地基
│   ├── 02-overview.md           整体架构图 + 7 层职责划分
│   └── 03-data-flow.md          tick/bar/factor/signal 流转时序
├── spec/
│   ├── 00-architecture-overview.md
│   ├── 09-postgres-schema-catalog.md
│   ├── 10-mt-adapter.md         mtapi gRPC → Gateway 接口契约
│   ├── 11-mdgateway.md          网关内部设计
│   ├── 12-mthub.md              会话注册中心 + OrderEventBroker
│   ├── 13-clickhouse-schema.md  CH 表设计、TTL、分区
│   ├── 14-rpc-contracts.md      ConnectRPC proto 契约
│   ├── 15-observability.md      Prometheus 指标、健康检查
│   ├── 16-mtapi-quirks-register.md  mtapi 暗坑清单
│   ├── 17-secrets-and-errors.md     Vault + 错误处理规范
│   ├── 18-backfiller.md             历史回填器
│   ├── 19-md-doctor.md              端到端对账 CLI
│   ├── 20-slo.md                    SLO + Error Budget 框架
│   ├── 21-backtest-replay.md
│   ├── 22-order-state-machine.md
│   ├── 23-risk-management.md
│   ├── 24-paper-trading.md
│   ├── 25-bar-revision-cascade.md
│   ├── 26-ai-strategy-generation.md
│   ├── 27-signal-execution-slo.md
│   ├── 31-risk-gate.md
│   └── 31-saas-foundation.md
├── adr/
│   ├── README.md                ADR 索引
│   └── 0001–0023                架构决策记录
├── plans/
│   └── 2026-06-29-mql-coverage-completion.md   MQL 覆盖率补齐计划
├── roadmaps/
│   └── live-strategy-user-facing.md            实盘策略用户功能路线图
├── superpowers/
├── runbook/
│   └── mt-incidents.md          常见故障应急手册
└── spec/                        技术规范
```

---

## 关键 ADR

| ADR | 标题 | 状态 |
|-----|------|------|
| 0012 | 统一回测/实盘代码路径 | Accepted |
| 0020 | EA 完全替代：统一 Strategy SDK + 双实现 Broker | Superseded by 0021 |
| 0021 | 策略运行时从 Python 迁移到 Go | Partially superseded by 0023 |
| 0022 | MQL 盲区架构 — 静态分析 + 运行时追踪 + 致命阻断 | Accepted |
| 0023 | AST 树遍历解释器 + MQL 源码为唯一真实来源 | Accepted |
