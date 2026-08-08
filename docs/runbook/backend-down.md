# Runbook · BackendDown

> 占位（post-launch 补全）。关联告警：`BackendDown`（`/healthz` 连续失败）。

## 含义
后端 `/healthz`（探 PG+NATS+Redis）不可达。

## 影响
**致命**——全站不可用。

## 初步处置
1. `docker ps` 看容器状态；`docker logs alphaforge-backend --tail 100` 看崩溃原因。
2. 三依赖逐个查：PG（`docker exec alphaforge-postgres pg_isready`）、NATS、Redis。
3. 容器崩 → `docker compose up -d backend` 重启；依赖崩 → 先修依赖。

## TODO（post-launch）
崩溃分类（OOM/panic/依赖）、回滚流程、值班响应 SLA。
