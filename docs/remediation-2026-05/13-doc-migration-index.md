# 文档保留和删除索引

## 1. 状态说明

- **保留**：仍可作为当前参考。
- **删除**：已过时、已有替代或与当前代码逻辑不一致，不再作为项目文档保留。

## 2. 根目录文档

| 文档 | 状态 | 当前依据 |
|---|---|---|
| `docs/API_STYLE.md` | 删除 | `docs/api/connectrpc-style-guide.md` |
| `docs/backtest_engine.md` | 删除 | `docs/domains/backtest-system.md` |
| `docs/metatrader-gap-evaluation-2026-04-21.md` | 删除 | 当前 MT 边界以代码、proto 和整改文档为准 |
| `docs/metatrader-parity-assessment-2026-04-20.md` | 删除 | 当前 MT 边界以代码、proto 和整改文档为准 |
| `docs/myfxbook-monthly-analysis-replication.md` | 删除 | 当前分析能力以代码和 ConnectRPC 契约为准 |
| `docs/remediation-checklist-2026-04-20.md` | 删除 | `docs/remediation-2026-05/10-migration-checklist.md` |
| `docs/worklog-next-steps.md` | 删除 | 当前任务以 `docs/remediation-2026-05/` 为准 |
| `docs/运维部署与隧道迁移手册-2026-04-30.md` | 删除 | `docs/ops/deployment.md`、`docs/ops/container-restart-checklist.md` |
| `docs/项目功能与逻辑链全景说明-2026-05-01.md` | 删除 | `docs/remediation-2026-05/04-target-architecture.md` 与领域文档 |

## 3. 子目录

| 目录 | 状态 | 处理方式 |
|---|---|---|
| `docs/archive/` | 删除 | 历史材料不再作为项目阅读入口 |
| `docs/plans/` | 删除 | 历史计划已被当前整改体系替代 |
| `docs/reports/` | 删除 | 历史报告不作为当前实施依据 |
| `docs/remediation-2026-05/` | 保留 | 当前整改权威入口 |
| `docs/api/` | 保留 | 当前 API 权威文档 |
| `docs/domains/` | 保留 | 当前领域边界文档 |
| `docs/ops/` | 保留 | 当前运维文档 |

## 4. 删除原则

文档是项目实施依据。凡是与当前代码和逻辑不符、已有权威替代、继续保留会造成误读的文档，应删除而不是归档保留。
