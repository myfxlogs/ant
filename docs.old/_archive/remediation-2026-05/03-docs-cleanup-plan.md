# 旧文档清理与新文档体系规划

## 1. 清理目标

当前 `docs/` 下存在多份历史文档，其中部分内容已落后于代码实现。

本轮清理目标：

- 不让旧文档继续误导开发。
- 不直接粗暴删除有历史价值的内容。
- 建立新的权威文档体系。
- 每份旧文档都要有明确处理结果。

处理结果分为：

- **保留**：仍然准确且有价值。
- **迁移**：内容合并到新文档。
- **归档**：保留历史记录，但不作为开发依据。
- **删除**：确认无价值且已有替代。

## 2. 新文档目录建议

建议新文档长期采用以下结构：

```text
docs/
  remediation-2026-05/
    README.md
    00-governance.md
    01-master-remediation-plan.md
    02-connectrpc-unification-plan.md
    03-docs-cleanup-plan.md

  architecture/
    README.md
    target-architecture.md
    backend-modular-monolith.md
    frontend-boundary.md
    worker-model.md

  domains/
    trading-core.md
    account-connection.md
    strategy-lifecycle.md
    backtest-system.md
    ai-workflow.md
    risk-engine.md
    logging-and-audit.md

  api/
    connectrpc-style-guide.md
    proto-splitting-guide.md
    service-catalog.md

  ops/
    deployment.md
    health-checks.md
    migration-and-baseline.md
    container-restart-checklist.md

  archive/
    2026-04/
    2026-05-before-remediation/
```

## 3. 当前旧文档处理结果

| 文档 | 建议处理 | 说明 |
|---|---|---|
| `API_STYLE.md` | 删除 | 当前依据为 `docs/api/connectrpc-style-guide.md` |
| `backtest_engine.md` | 删除 | 当前依据为 `docs/domains/backtest-system.md` |
| `metatrader-gap-evaluation-2026-04-21.md` | 删除 | 2026-04 历史评估，不作为当前实施依据 |
| `metatrader-parity-assessment-2026-04-20.md` | 删除 | 2026-04 历史评估，不作为当前实施依据 |
| `myfxbook-monthly-analysis-replication.md` | 删除 | 研究性质文档，不作为当前产品实现依据 |
| `remediation-checklist-2026-04-20.md` | 删除 | 已被 `docs/remediation-2026-05/10-migration-checklist.md` 替代 |
| `worklog-next-steps.md` | 删除 | 工作日志性质，不作为权威文档 |
| `运维部署与隧道迁移手册-2026-04-30.md` | 删除 | 当前依据为 `docs/ops/deployment.md` 与 `docs/ops/container-restart-checklist.md` |
| `项目功能与逻辑链全景说明-2026-05-01.md` | 删除 | 整改前快照，当前依据为整改目录、领域文档和运维文档 |
| `plans/` | 删除 | 历史计划已被当前整改体系替代 |
| `reports/` | 删除 | 历史报告不作为当前实施依据 |
| `archive/` | 删除 | 历史材料不再作为项目阅读入口 |

## 4. 文档迁移规则

迁移旧文档时必须：

1. 先标注旧文档状态。
2. 抽取仍准确的内容。
3. 删除过期或错误描述。
4. 将“已实现”和“目标设计”分开。
5. 在新文档中注明来源。
6. 最后删除旧文档。

## 5. 不允许的做法

不允许：

- 未建立替代文档就删除仍然有效的实施依据。
- 把历史计划当成当前实现。
- 把占位接口写成已完成能力。
- 在同一文档里混写现状、规划、猜测而不标注。
- 新增与整改目标冲突的文档。

## 6. 新权威文档优先级

优先补齐顺序：

1. `docs/api/connectrpc-style-guide.md`
2. `docs/api/proto-splitting-guide.md`
3. `docs/architecture/target-architecture.md`
4. `docs/domains/trading-core.md`
5. `docs/domains/strategy-lifecycle.md`
6. `docs/domains/risk-engine.md`
7. `docs/ops/container-restart-checklist.md`

## 7. 删除旧文档前检查清单

删除旧文档前必须确认：

- 是否已有新文档替代。
- 是否仍被 README 或其他文档引用。
- 是否包含尚未迁移的重要运维命令。
- 是否包含真实环境参数或敏感信息。
- 是否会影响团队理解当前实施方式。

## 8. 最终目标

最终 `docs/` 应从“历史材料堆叠”变为“可执行工程手册”：

```text
新成员能通过 docs 理解系统
开发能通过 docs 找到边界
评审能通过 docs 检查规则
运维能通过 docs 执行部署和回滚
整改能通过 docs 跟踪进度
```
