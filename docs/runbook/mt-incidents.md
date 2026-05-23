# Runbook · MT 行情链路常见故障

> 范围：mdgateway / mthub / ClickHouse / NATS 相关故障
> 适用：值班 SRE + AI 自检

## 1. 故障速查表

| 现象 | 可能原因 | 跳到 |
|---|---|---|
| `/readyz` 返回 503 | PG/CH/Redis/NATS 任一不通 | §2 |
| Prometheus `md_circuit_state == 1` | 单 broker 不可达 | §3 |
| `md_ch_write_errors_total` rate 高 | CH 写入故障 | §4 |
| `md_spill_dir_bytes` 持续增长 | spill 在累积，CH 没恢复 | §4 |
| `md_tick_total` 单账户为 0 持续 | 该 broker 订阅失败 | §5 |
| `md_tick_dropped_total{reason="bid_gt_ask"}` 飙升 | broker 数据脏 | §6 |
| 前端 SSE OrderEvent 卡住 | mthub event broker 满 | §7 |
| 启动后 spill 目录有 jsonl 但 CH 计数没涨 | replay 失败 | §8 |

---

## 2. /readyz 返回 503

### 2.1 检查每个依赖

```bash
curl -s http://localhost:8080/readyz | jq '.'
```

定位失败的 check，按下表处理：

| check 失败 | 命令 | 修复 |
|---|---|---|
| `postgres` | `docker exec ant-postgres pg_isready` | 重启 ant-postgres；查 disk |
| `clickhouse` | `docker exec ant-clickhouse clickhouse-client --query 'SELECT 1'` | 见 §4 |
| `redis` | `docker exec ant-redis redis-cli PING` | 重启 ant-redis |
| `nats` | `docker exec ant-nats nats account info` | 重启 ant-nats |
| `mdgateway.ratio < 0.5` | 见 §5 | |

---

## 3. CircuitBreaker 打开

### 3.1 找出受影响账户

```bash
curl -s http://localhost:8080/metrics \
  | grep '^md_circuit_state{' \
  | awk -F'"' '$0 ~ /1$/ {print $2, $4}'
```

输出 `account_id, broker` 列表。

### 3.2 单账户 livez 状态

```bash
curl -s http://localhost:8080/livez/account/$ACCOUNT_ID | jq '.'
```

`state` 通常是 `degraded`，看 `last_tick_at`。

### 3.3 触发原因

- 账户 password 错误（broker 拒登）→ `docker logs ant-backend | grep $ACCOUNT_ID | grep -i 'login\|auth'`
- mtapi 网关不可达 → `docker logs ant-backend | grep $ACCOUNT_ID | grep -i 'dial\|connection refused'`
- broker 服务器升级中 → 30s 后自动 half_open 尝试

### 3.4 手动修复

```bash
# 重启该账户的 gateway（不影响其他）
curl -X POST http://localhost:8080/admin/v1/restart-account \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d "{\"account_id\": \"$ACCOUNT_ID\"}"
```

---

## 4. ClickHouse 写入故障

### 4.1 验证 CH 状态

```bash
docker exec ant-clickhouse clickhouse-client --query "SELECT 1"
docker exec ant-clickhouse df -h /var/lib/clickhouse
docker logs ant-clickhouse --tail 200
```

### 4.2 常见原因

| 症状 | 修复 |
|---|---|
| `Table is in readonly mode` | `docker exec ant-clickhouse clickhouse-client --query "SYSTEM ENABLE TABLE WRITES"` |
| `Disk full` | 清理 `clickhouse_data` 卷；缩 TTL；扩盘 |
| `Connection refused` | `docker compose restart ant-clickhouse` |
| `Memory limit exceeded` | 调整 `max_memory_usage` 配置 |

### 4.3 spill 验证

```bash
# spill 文件数
docker exec ant-backend ls /var/lib/ant/spill/*.jsonl 2>/dev/null | wc -l

# 待 replay 大小
docker exec ant-backend du -sh /var/lib/ant/spill/

# 等 CH 恢复后强制 replay
curl -X POST http://localhost:8080/admin/v1/spill-replay \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

---

## 5. 单账户 tick rate 0

### 5.1 排除链路

```bash
ACCOUNT_ID=...
curl -s http://localhost:8080/livez/account/$ACCOUNT_ID | jq '.'
```

- `state=disconnected` → §3
- `state=connected` 但 `tick_rate_1m=0` → §5.2
- `subscribed_symbols=[]` → 检查 `mt_accounts.canonical_subscribed_symbols`

### 5.2 broker 推送停滞

```bash
# 看最近 10 条 broker 推送日志
docker logs ant-backend --tail 1000 \
  | jq -c "select(.account_id == \"$ACCOUNT_ID\")" \
  | tail -10
```

可能：
- 周末 / 假期（FX 不交易）
- broker 该 symbol 进入维护
- mtapi.io 网关单边断开（mtapi 本身没 close stream）

### 5.3 强制重订阅

```bash
curl -X POST http://localhost:8080/admin/v1/restart-account \
  -d "{\"account_id\": \"$ACCOUNT_ID\"}"
```

---

## 6. bid > ask drop 飙升

### 6.1 定位 broker / symbol

```promql
topk(10, rate(md_tick_dropped_total{reason="bid_gt_ask"}[5m]))
```

### 6.2 处理策略

- 单 symbol 持续 → broker 数据脏；联系 broker 或 disable 该 symbol
- 全 broker 多 symbol → 网络/proxy 损坏；查 mtapi.io 状态
- 某账户突发 → 与 broker 维护期同步；30 分钟后自愈

---

## 7. SSE OrderEvent 卡住

### 7.1 检查 broker 容量

```promql
mthub_event_subscriber_count{user_id="..."}
mthub_event_dropped_total{reason="chan_full"}
```

### 7.2 处理

- 个别 user 卡住 → 让 user 重连（前端 reload）
- 全局事件丢弃多 → 增大 `mthub.events.subscriberChanSize` 配置

---

## 8. spill replay 失败

### 8.1 查 failed 目录

```bash
docker exec ant-backend ls -la /var/lib/ant/spill/failed/
docker exec ant-backend cat /var/lib/ant/spill/failed/*/*.errors.jsonl | head
```

### 8.2 常见错误

| 错误 | 处理 |
|---|---|
| `cannot parse Decimal` | 数据格式问题；手动 fix jsonl 后移回主目录 |
| `unknown column` | schema 漂移；先跑 chmigrate |
| `connection refused` | CH 未起；等 CH 起来后 |

### 8.3 手动重放

```bash
# 把 failed 文件移回主目录
docker exec ant-backend sh -c "
  mv /var/lib/ant/spill/failed/*/*.jsonl /var/lib/ant/spill/
"

# 触发 replay
curl -X POST http://localhost:8080/admin/v1/spill-replay
```

---

## 9. 应急切换：旧 PG 路径

> **极端情况**：CH 长时间不可恢复（>1h），且 spill 已 100GB 警戒线
> **临时**：业务读路径 fallback PG（M7-rewrite 期间保留的能力）

```bash
# 切换 read 路径到 PG
curl -X PATCH http://localhost:8080/admin/v1/config \
  -d '{"market_read_source": "pg_legacy"}'
```

**注意**：这只是 K 线读路径，tick / factor / signal 仍在 spill 等待 CH 恢复。

事后回滚：

```bash
curl -X PATCH http://localhost:8080/admin/v1/config \
  -d '{"market_read_source": "clickhouse"}'
```

---

## 10. 升级期间快速回滚

```bash
# 1. 停 backend
docker compose stop ant-backend

# 2. 切回上一镜像（前提：用 tag）
export ANT_BACKEND_IMAGE=ant-backend:v1.X.Y-1
docker compose up -d ant-backend

# 3. 验证
sleep 30
curl -sf http://localhost:8080/readyz | jq '.status'
```

---

## 附录：日志聚合查询

```bash
# 某账户最近 1h 所有日志
docker logs ant-backend --since 1h \
  | jq -c "select(.account_id == \"$ACCOUNT_ID\")" \
  | head -200

# 某 trace_id 全链路
TRACE=...
docker logs ant-backend \
  | jq -c "select(.trace_id == \"$TRACE\")"
```
