# ADR 索引

**Format**: Architecture Decision Records，参考 [ADR GitHub Org](https://adr.github.io/) Markdown 模板（精简版）。
**编号规则**：单调递增，`NNNN-<slug>.md`。
**生命周期**：`Proposed` → `Accepted`（不可逆）→ `Superseded by NNNN`（如被替代）。
**最后更新**：2026-05-23（M0.2 全量 Accepted）

## 按状态

### Accepted（16/16 ✅）

| # | 标题 | 实施里程碑 | 关联 |
|---|---|---|---|
| 0001 | [Connect RPC + SSE 通信协议](0001-connect-rpc-sse.md) | 既定 | AGENT.md 协议规则 |
| 0002 | [三域 monorepo](0002-three-domain-monorepo.md) | 既定 | AGENT.md 三域结构 |
| 0003 | [PostgreSQL + Redis 主存方案](0003-postgres-redis.md) | 既定 | AGENT.md 数据规则 |
| 0004 | [Python 沙箱进程模型](0004-python-sandbox-process-model.md) | M3 | 安全红线 |
| 0005 | [用户中心架构（非多租户）](0005-user-centric-architecture.md) | 既定 | AlfQ 迁移 §1.2 |
| 0006 | [SQL 访问层：sqlc vs sqlx](0006-sqlc-vs-sqlx.md) | M2-M6 | 代码生成优先 |
| 0007 | [AI 策略生成 bounded tools](0007-ai-bounded-tools.md) | M4 | M4 AI 闭环 |
| 0008 | [单机 docker-compose 部署](0008-docker-compose-deployment.md) | 既定 | AGENT.md 部署形态 |
| 0009 | [运行时主版本基线锁版](0009-runtime-version-baseline.md) | 既定 | AGENT.md 版本规则 |
| 0010 | [错误码体系](0010-error-code-system.md) | M0.3 | MR-04 |
| 0011 | [可观测性：trace_id / health / metrics](0011-observability.md) | M0.3 | MR-05, MR-07 |
| 0012 | [Broker Adapter 抽象](0012-broker-adapter.md) | M1/M2 | M1+M2 前置 |
| 0013 | [因子 DSL 规范](0013-factor-dsl.md) | M7 | M7.2 因子 DSL 引擎 |
| 0014 | [ClickHouse 时序存储](0014-clickhouse-timeseries.md) | M7 | M7.4 ClickHouse 容器 + 4 表 |
| 0015 | [ONNX 推理通道](0015-onnx-inference.md) | M7 | M7.5 quantengine |
| 0016 | [沙箱降级](0016-sandbox-degradation.md) | M7 | M7.7 生产路径排除 Python |

### Proposed（0）

无待决策项。全部 12 篇 ADR 已在 M0.2 完成 Accepted。

## 按主题

| 主题 | ADR |
|---|---|
| **通信协议** | 0001 Connect RPC + SSE |
| **仓库结构** | 0002 三域 monorepo |
| **数据存储** | 0003 PostgreSQL + Redis |
| **安全沙箱** | 0004 Python 沙箱进程模型 |
| **用户模型** | 0005 用户中心架构 |
| **数据访问** | 0006 sqlc vs sqlx |
| **AI 安全** | 0007 AI bounded tools |
| **部署形态** | 0008 单机 docker-compose |
| **版本基线** | 0009 运行时主版本锁版 |
| **错误处理** | 0010 错误码体系 |
| **可观测性** | 0011 trace_id / health / metrics |
| **交易网关** | 0012 Broker Adapter |
| **因子引擎** | 0013 因子 DSL 规范 |
| **时序存储** | 0014 ClickHouse 时序存储 |
| **ML 推理** | 0015 ONNX 推理通道 |
| **安全沙箱** | 0004 Python 沙箱进程模型 · 0016 沙箱降级 |
| **策略市场** | 待立（M5 前置：D-09 计费模式） |
| **移动端** | 待立（D-04） |
| **i18n** | 待立（D-08 语种选择） |
| **a11y** | 待立（D-05 基线） |

## 编写规范

每篇 ADR 含 7 节：

1. **Status**: Accepted (yyyy-mm-dd) / Superseded by NNNN
2. **Context**: 为什么需要这个决策（≤10 行）
3. **Options**: 候选方案 A/B/C，各列利弊
4. **Decision**: 选了哪个 + 理由
5. **Consequences**: 正/负面影响、迁移路径
6. **Related**: 关联 finding / ADR / 文档
7. **History**: 状态变更记录

新决策 → 提 PR 在 `docs/adr/NNNN-<slug>.md`，编号从 `0013` 单调递增。
