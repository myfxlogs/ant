# ADR 索引

**Format**: Architecture Decision Records，参考 [ADR GitHub Org](https://adr.github.io/) Markdown 模板（精简版）。
**编号规则**：单调递增，`NNNN-<slug>.md`。
**生命周期**：`Proposed` → `Accepted`（不可逆）→ `Superseded by NNNN`（如被替代）。

## 索引

| # | 标题 | 状态 | 决策人 | 关联 |
|---|---|---|---|---|
| 0001 | [Connect RPC + SSE 通信协议](0001-connect-rpc-sse.md) | Accepted | — | AGENT.md 硬性规则 |
| 0002 | [三域 monorepo（backend / strategy-service / frontend）](0002-three-domain-monorepo.md) | Accepted | — | AGENT.md 三域结构 |
| 0003 | [PostgreSQL + Redis 主存方案](0003-postgres-redis.md) | Accepted | — | AGENT.md 数据规则 |
| 0004 | [Python 沙箱进程模型](0004-python-sandbox-process-model.md) | **Proposed**（D-06） | — | SEC-4 |
| 0005 | [用户中心架构（非多租户）](0005-user-centric-architecture.md) | Accepted | — | — |
| 0006 | [SQL 访问层：sqlc vs sqlx](0006-sqlc-vs-sqlx.md) | **Proposed**（D-01） | — | CR-04 |
| 0007 | [AI 策略生成 bounded tools](0007-ai-bounded-tools.md) | **Proposed** | — | UX-CR-02 |
| 0008 | [单机 docker-compose 部署](0008-docker-compose-deployment.md) | Accepted | — | AGENT.md 部署形态 |
| 0009 | [运行时主版本基线锁版](0009-runtime-version-baseline.md) | Accepted | — | AGENT.md 版本规则 |
| 0010 | [错误码体系](0010-error-code-system.md) | **Proposed**（D-07） | — | MR-04 |
| 0011 | [可观测性：trace_id / health / metrics](0011-observability.md) | **Proposed** | — | MR-05 |
| 0012 | [Broker Adapter 抽象与多 broker 拓展](0012-broker-adapter.md) | **Proposed**（D-10） | — | M1+M2 前置 |

## 待用户拍板的决策汇总（Proposed → Accepted 前提）

| 决策点 | 默认推荐 | ADR |
|---|---|---|
| D-01 sqlc vs sqlx | A. 全面 sqlc，M2 起新模块用，存量按 M2-M5 顺序迁 | 0006 |
| D-04 移动端 | 需用户定 | （ADR 待立） |
| D-05 a11y 基线 | B. 暂不保证（短期） | （ADR 待立） |
| D-06 沙箱进程模型 | 需用户定（M3 末期前必须） | 0004 |
| D-07 错误码体系 | A. 自建 errs 包 + i18n | 0010 |
| D-08 i18n 支持语种 | 需用户定 | （ADR 待立） |
| D-09 策略市场首期 | A+B（月租 + 一次性买断） | （ADR 待立 M5） |
| D-10 broker 拓展 | 需用户定 | 0012 |

## 编写规范

每篇 ADR 含 7 节：

1. **Status**: Proposed / Accepted (yyyy-mm-dd) / Superseded by NNNN
2. **Context**: 为什么需要这个决策（≤10 行）
3. **Options**: 候选方案 A/B/C，各列利弊（如有）
4. **Decision**: 选了哪个 + 理由（Accepted 后填）
5. **Consequences**: 正/负面影响、迁移路径
6. **Related**: 关联 finding / ADR / 文档
7. **History**: 状态变更记录
