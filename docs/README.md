# ant 文档（v2 · MT 基础重写版）

> **重置说明**：v1 文档已归档至 `docs.old/`。v2 围绕「**MetaTrader 作为量化数据基础**」重新设计，参照 alfq 架构并基于 ant 已有积累（CircuitBreaker、Spill 旋转、mtapi 暗坑修复）做精简化重构。
>
> **关键决策**：彻底重写 MT 层（mt4client/mt5client → 新 mdgateway+adapter+mthub），目标 ~600 行替代 ~4500 行；保留 ant 已有的故障恢复增量；切断与老 kline_service 的双数据源。

---

## 阅读路径

| 角色 | 推荐阅读顺序 |
|---|---|
| **新工程师 onboarding** | `architecture/01-vision.md` → `architecture/02-overview.md` → `architecture/03-data-flow.md` |
| **MT 适配实现者** | `spec/10-mt-adapter.md` → `spec/16-mtapi-quirks-register.md` → `adr/0003` |
| **行情网关实现者** | `spec/11-mdgateway.md` → `spec/13-clickhouse-schema.md` → `adr/0004` `0005` |
| **会话/下单实现者** | `spec/12-mthub.md` → `spec/14-rpc-contracts.md` |
| **运维/SRE** | `spec/15-observability.md` → `spec/20-slo.md` → `runbook/mt-incidents.md` |
| **数据基础对账/排错** | `spec/19-md-doctor.md` → `spec/18-backfiller.md` |
| **安全/密钥** | `spec/17-secrets-and-errors.md` → `adr/0011` |
| **数据归属（C2C 多租户）** | `adr/0006-platform-shared-vs-user-private.md` |
| **M10 数据基础硬化实施者** | `adr/0008` `0009` `0010` `0011` → `spec/18` `19` `20` → `plan/ROADMAP.md` §M10 |
| **决策审计** | `adr/` 0001–0011 全部按编号读 |
| **执行计划** | `plan/ROADMAP.md` + `plan/M10-DEEPSEEK-PROMPT.md` + `plan/M10-CLARIFICATIONS.md` |
| **AI Agent（DeepSeek）开工** | `AGENT.md` §12 阅读清单 → `plan/M10-DEEPSEEK-PROMPT.md` |

---

## 目录结构

```
docs/
├── README.md                    本文件
├── architecture/
│   ├── 01-vision.md             设计哲学：MT = 地基
│   ├── 02-overview.md           整体架构图 + 7 层职责划分
│   └── 03-data-flow.md          tick/bar/factor/signal 流转时序
├── spec/
│   ├── 10-mt-adapter.md         mtapi gRPC → Gateway 接口契约
│   ├── 11-mdgateway.md          网关内部：normalizer/quality/aggregator/publisher/writer/spill/circuit（含 M10 强化叠加段）
│   ├── 12-mthub.md              会话注册中心 + OrderEventBroker
│   ├── 13-clickhouse-schema.md  CH 表设计、TTL、分区、查询模式（§2.6 DLQ + §2.7 Buffer + §2.8 v2）
│   ├── 14-rpc-contracts.md      ConnectRPC proto 契约（ant.v1）
│   ├── 15-observability.md      Prometheus 指标、健康检查、日志规范（§6.x M10 alert + §6.y M10 metric）
│   ├── 16-mtapi-quirks-register.md  mtapi 暗坑清单
│   ├── 17-secrets-and-errors.md     Vault envelope encryption + 错误处理规范
│   ├── 18-backfiller.md             历史回填器（mtapi GetPriceHistory）  ← M10
│   ├── 19-md-doctor.md              端到端对账 CLI                       ← M10
│   └── 20-slo.md                    SLO + Error Budget 框架              ← M10
├── adr/
│   ├── README.md                ADR 索引
│   ├── 0001-mt-foundation-full-rewrite.md
│   ├── 0002-clickhouse-as-timeseries.md
│   ├── 0003-direct-mtapi-no-wrapping.md
│   ├── 0004-tick-dedup-and-quality.md
│   ├── 0005-circuit-breaker-with-spill.md
│   ├── 0006-platform-shared-vs-user-private.md   C2C 数据归属
│   ├── 0007-post-m7-retrospective.md             M7 实际等价路线 A 复盘
│   ├── 0008-storage-dedup-and-time-axis.md       ← M10
│   ├── 0009-replay-dual-write-and-bar-finality.md ← M10
│   ├── 0010-slo-alert-dlq-trace.md               ← M10
│   └── 0011-capacity-vault-cache-hardening.md    ← M10
├── plan/
│   ├── ROADMAP.md                  里程碑与卡片（M7-M9 ☑；M10 18 卡片）
│   ├── BACKLOG.md                  待办与已知缺陷
│   ├── M10-DEEPSEEK-PROMPT.md      M10 启动指引（喂给 DeepSeek 的 prompt）
│   └── M10-CLARIFICATIONS.md       M10 文档审核需澄清项
├── handover/
│   ├── verify-M7.*.log             M7 各卡片验收日志
│   ├── M7-closure.md               M7 关闭报告
│   └── M10-*.{md,json}             M10 关闭后产出（md-doctor / slo-report）
└── runbook/
    └── mt-incidents.md             常见故障应急手册
```

---

## 设计哲学一句话

> **MT 是 ant 的地基；地基的稳定性 = 项目天花板。**
> 所以我们：(1) 用最少的代码实现最完整的语义；(2) 把可观测性与故障恢复内建到每一层；(3) 把暗坑作为知识沉淀而非散落代码。

---

## 与旧文档的差异

| 维度 | docs.old (v1) | docs (v2) |
|---|---|---|
| MT 层定位 | "K 线源 + 下单通道" | "量化数据基础设施" |
| 代码量目标 | 现状 ~4500 行 | 目标 ~600 行 |
| 数据存储 | PG.kline_data 单源 | ClickHouse 4 表 + PG 业务库 |
| 文档粒度 | 12 篇综合 | 7 篇规范 + 5 篇 ADR + 1 篇 quirks |
| ADR 编号 | 0001-0016（混合） | 0001-0011（MT 重写 + C2C 归属 + M10 硬化） |
