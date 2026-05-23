# ADR-0002 三域 monorepo（backend / strategy-service / frontend）

## Status
Accepted (2026-05-23)

## Context
ant 涉及 Go 业务后端、Python 策略沙箱、React 前端三种语言/运行时。候选：① 三仓分治；② 单 monorepo + 子目录；③ workspace 工具（nx/turborepo）。

## Options
- **A. 单 monorepo，按域分子目录**（已选）：`backend/` + `strategy-service/` + `frontend/`，proto 单源 `proto/`，统一 `Makefile` 与 `docker-compose.yml`
- B. 三仓分治：releases 难协调；proto schema 同步困难
- C. nx/turborepo workspace：增加工具栈复杂度，与 Go/Python 集成弱

## Decision
**A**。子目录边界严格：跨域调用一律走 Connect RPC（不走文件 import）；proto 改动同时影响三域时，PR 必须包含三域代码生成产物的更新（CI 校验）。

## Consequences
- **+** schema/版本一致性强；本地 `docker compose up` 一键起全栈
- **+** 与 AGENT.md 部署形态（单机 compose）天然契合
- **−** 仓库体积大；clone 慢（CR-02 缓解中）
- **−** CI 需要按 path filter 选择性 build（M0.1 卡片）

## Related
- AGENT.md 三域结构
- DESIGN-REVIEW CR-02（仓库体积）、MN-03（proto 路径嵌套）

## History
- 2026-05-23 Accepted
