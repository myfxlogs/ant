# Runbook · BackendDown

> 关联告警：`BackendDown`（`up{job="alphaforge-backend"} == 0`，持续 1m，severity=critical）

## 症状

Prometheus scrape `up{job="alphaforge-backend"} == 0` 持续 1 分钟。`/healthz` 不可达。前端所有 API 调用超时或 502/524。

## 影响

**致命**——全站不可用。所有用户无法登录、查看数据、运行策略。实盘策略停止执行（无 bar 输入 + 无下单路径）。SSE 流断开。

## 诊断步骤

```bash
# 1. 确认容器状态
docker ps -a | grep alphaforge-backend

# 2. 查看崩溃日志
docker logs alphaforge-backend --tail 200

# 3. 检查三个依赖
docker exec alphaforge-postgres pg_isready
docker exec alphaforge-redis redis-cli PING
docker exec alphaforge-nats nats account info 2>/dev/null || docker logs alphaforge-nats --tail 50

# 4. 检查宿主机资源
df -h /     # 磁盘
free -m     # 内存
```

## 应急处置

1. **容器已退出** → 查看退出原因后重启：
   ```bash
   docker logs alphaforge-backend --tail 50  # 确认退出原因
   docker compose up -d backend
   ```
2. **OOM kill** → 日志中含 `OOM` 或 `signal: killed`：
   ```bash
   docker inspect alphaforge-backend | grep -i oomkilled
   ```
   临时：增大容器内存限制。根因：参考 `backend-high-memory.md` runbook 排查内存泄漏。
3. **依赖故障** → 先修依赖：
   - PG down → `docker compose restart postgres`
   - NATS down → `docker compose restart nats`
   - Redis down → `docker compose restart redis`
   依赖恢复后 backend 的 `/healthz` 会自动恢复。
4. **panic 崩溃** → 日志中含 `panic:` 或 `runtime error`：
   ```bash
   docker logs alphaforge-backend --tail 200 | grep -A5 "panic"
   ```
   收集 stack trace 报 bug，重启 backend。

## 常见根因

- **OOM kill**：容器内存超限被 kernel OOM killer 杀死。参考 `backend-high-memory.md`
- **PG 连接池耗尽**：`/healthz` 探 PG 超时，容器被标记 unhealthy 后退出。参考 `pg-pool-exhausted.md`
- **panic**：Go runtime panic（nil pointer / index out of range），未被 recover 捕获
- **依赖不可用**：PG/NATS/Redis 任一宕机，`/healthz` 返回非 200，Prometheus 标记 down
