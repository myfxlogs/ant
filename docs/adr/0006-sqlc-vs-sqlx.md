# ADR-0006 SQL 访问层：sqlc vs sqlx

## Status
**Accepted** (2026-05-23) — 按推荐方向 A (全面 sqlc)；M2 起新模块强制用 sqlc，存量按 M2-M5 顺序迁

## Context
AGENT.md 写明「sqlc 优先，不引入 ORM」，但仓库现状（`@/opt/ant/backend/internal/repository/*.go` 36 文件）**全部手写 sqlx**。规则与现实严重背离（DESIGN-REVIEW CR-04）。需决策走向。

## Options
- **A. 全面 sqlc**（推荐）：M0.3 引入 `sqlc.yaml` + 生成层；M2 起新模块强制用 sqlc；存量按 M2-M5 顺序迁；M6 末完成全量
  - **+** 类型安全；schema 改动编译期暴露；与「代码生成优先」纪律一致
  - **−** 短期工作量大（36 仓库文件迁移）；sqlc 复杂查询写法学习成本
- B. 接受 sqlx 现状：删除 AGENT.md 关于 sqlc 的承诺；改写为「使用 sqlx + strong testing」
  - **+** 零额外迁移成本
  - **−** 类型安全弱；schema 漂移检测靠人工
- C. 混合：新模块 sqlc，存量 sqlx；长期共存
  - **+** 渐进
  - **−** 双栈维护；新人学习成本翻倍

## Decision
**A (全面 sqlc)**：M2 起新模块强制用 sqlc；存量按 M2-M5 顺序迁移；M6 前全量切换。

## Consequences（按 A 推演）
- **+** 与 AGENT.md 一致；与 AlfQ OMS / risksvc 移植同步引入 sqlc
- **+** Schema 演进时 CI 强制编译期校验
- **−** 迁移期 sqlx + sqlc 共存，需明确边界（M2 起新代码强制 sqlc）

## Related
- DESIGN-REVIEW CR-04
- AGENT.md 工程纪律「代码生成优先」
- AlfQ 迁移计划 §M2 OMS 持久化

## History
- 2026-05-23 Proposed

- 2026-05-23 Accepted（按推荐方向落地，详见 Decision）
