# EA 完全替代 — 容器部署指南

> **⚠️ 本文档已过时。** Python strategy-service 和 strategy-worker 容器已按 ADR-0021 退役。
> EA 策略执行已全面迁移至 Go SDK（`backend/strategy/sdk/`）。本文档留存仅供历史参考，
> 部署请参考 `docker-compose.yml` 和 `AGENT.md`。

## 部署架构（历史）

```
                    ┌──────────────────┐
                    │   ant-frontend   │ :8022
                    └────────┬─────────┘
                             │
                    ┌────────▼─────────┐
                    │   ant-backend    │ :8080 (Go, 风控门)
                    │   Gate + Canary  │
                    └────────┬─────────┘
                             │ ConnectRPC
                    ┌────────▼─────────┐
                    │ strategy-service │ :8081 (Python, 回测/研究)
                    └────────┬─────────┘
                             │
                    ┌────────▼─────────┐
                    │ strategy-worker  │ (Python, 实盘 EA 执行)
                    │ seccomp+cgroup   │
                    │ non-root+断网    │
                    └──────────────────┘
```

## 快速启动

```bash
# 1. 启动基础服务 (postgres, redis, nats, clickhouse)
docker compose up -d postgres redis nats clickhouse

# 2. 等待数据库就绪
docker compose exec postgres pg_isready -U ant

# 3. 启动后端 + 策略服务 + Worker
docker compose \
  -f docker-compose.yml \
  -f deploy/docker-compose.ea.yml \
  up -d backend strategy-service strategy-worker

# 4. 验证
curl http://localhost:8080/healthz      # backend
curl http://localhost:8081/health       # strategy-service
docker compose logs strategy-worker     # worker
```

## 金丝雀灰度上线

```bash
# 1. 添加金丝雀账户
curl -X POST http://localhost:8080/api/admin/canary/accounts \
  -H "Content-Type: application/json" \
  -d '{"account_ids": ["mt4-real-1", "mt5-real-2"]}'

# 2. 激活金丝雀阶段 (0.01 手)
curl -X POST http://localhost:8080/api/admin/canary/activate

# 3. 查看状态
curl http://localhost:8080/api/admin/canary/status
# → {"stage": "canary", "allowed_lots": "0.01", "accounts": 2}

# 4. 递进手数 (每 24h + 10 笔成功交易自动递进，或手动)
curl -X POST http://localhost:8080/api/admin/canary/step-up

# 5. 全量上线
curl -X POST http://localhost:8080/api/admin/canary/promote-full
```

## Kill-Switch 应急

```bash
# 触发 (立即阻断所有实盘订单)
curl -X POST http://localhost:8080/api/admin/canary/kill-switch \
  -H "Content-Type: application/json" \
  -d '{"reason": "回测/实盘订单方向偏差"}'

# 回滚到上一阶段
curl -X POST http://localhost:8080/api/admin/canary/rollback

# 解除 kill-switch
curl -X POST http://localhost:8080/api/admin/canary/disengage

# 验证恢复
curl http://localhost:8080/api/admin/canary/status
```

## 风控门验证

```bash
# 验证 gate 已注入 (日志应有 "risk.Gate injected")
docker compose logs backend | grep "Gate"

# 验证 fail-closed (无 AccountStateProvider 时所有实盘订单被拒)
docker compose logs strategy-worker | grep "fail-closed"

# 验证 kill-switch 阻断
curl -X POST http://localhost:8080/api/admin/canary/kill-switch \
  -d '{"reason": "drill"}'
# → live_runner 日志: "order BLOCKED by risk gate ... rule=kill_switch"
```

## 沙箱逃逸验证

```bash
# 进入 worker 容器
docker compose exec strategy-worker sh

# 测试网络阻断
python3 -c "import socket; socket.socket()"  # → OSError (seccomp blocked)

# 测试文件写阻断
python3 -c "open('/tmp/test', 'w')"          # → OSError (read_only fs)

# 测试进程阻断
python3 -c "import os; os.fork()"            # → OSError (seccomp blocked)

# 但 numpy 仍然可用
python3 -c "import numpy; print(numpy.eye(3))"  # → OK
```

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `ANTRADER_RISK_GATE_ENABLED` | true | Gate 强制注入开关 (D6-A) |
| `ANTRADER_CANARY_INITIAL_LOTS` | 0.01 | 金丝雀起始手数 |
| `ANTRADER_CANARY_MAX_LOTS` | 10.0 | 最大手数 |
| `ANTRADER_CANARY_TRADES_PER_STEP` | 10 | 每阶段最少成功笔数 |
| `ANTRADER_CANARY_MIN_HOURS` | 24 | 每阶段最小持续小时 |
| `WORKER_POOL_SIZE` | 4 | Worker 进程池大小 |
| `WORKER_MAX_RSS_MB` | 512 | Worker 内存上限 |
