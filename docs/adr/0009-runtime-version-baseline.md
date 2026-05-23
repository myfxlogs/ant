# ADR-0009 运行时主版本基线锁版

## Status
Accepted (2026-05-23)

## Context
AGENT.md「版本」规则：默认追随官网最新稳定版。但运行时主版本（语言/数据库/前端框架）频繁升级风险高，需锁版理由可追溯。

## Options
- **A. 主版本通过 ADR 锁版，次/补丁版本跟最新**（已选）
- B. 全部跟最新：风险高
- C. 全部锁定：长期落后

## Decision
**A**。当前基线（2026-05-23）：

| 组件 | 版本 | 锁版理由 |
|---|---|---|
| Go | 1.26 | 当前最新稳定，2026-02 GA |
| Python | 3.14 | 当前最新稳定，2025-10 GA |
| Node | 24 LTS | Active LTS（2025-10 起） |
| TypeScript | 5.9 | 5.x 系列最新；6.0 未稳定 |
| PostgreSQL | 18 | 当前最新稳定 |
| Redis | 8 | 当前最新稳定 |

升级流程：① 提 ADR superseding 本文件；② 给出 breaking changes 评估 + 兼容性测试结果；③ 至少先在 `develop` 分支跑满 1 周；④ 升级 commit 必须包含 docker-compose / Dockerfile / `go.mod` / `requirements.txt` / `package.json` 同步更新。

## Consequences
- **+** 稳定性可控；跨人协作版本一致
- **+** CI 锁版 + Dockerfile 锁版
- **−** 落后官网最新版数月（可接受）

## Related
- AGENT.md「版本」规则
- 实际配置：`@/opt/ant/backend/Dockerfile` `@/opt/ant/strategy-service/Dockerfile` `@/opt/ant/frontend/Dockerfile` `@/opt/ant/docker-compose.yml`

## History
- 2026-05-23 Accepted（初版基线）
