# ADR-0002 · ClickHouse 作为时序存储

- **状态**：Accepted
- **日期**：2026-05-23
- **关联 spec**：`docs/spec/13-clickhouse-schema.md`

## 1. 背景

ant v1 把所有数据放 PostgreSQL：
- `kline_data` 表用 TIMESTAMP + UNIQUE 约束
- 每账户 5K bars/天，10 账户 1 年 ≈ 1800 万行
- 1 broker × 100 symbol × 1m bar = 14 万 bars/天（单账户）

PG 在百万行 OHLCV 查询上仍可接受（毫秒级），但：
- **完全不能存 tick**（百万倍数据量级）
- 100 账户/年 ≈ 18 亿行 OHLCV，PG 索引膨胀严重
- 因子值（factor_values）按用户 × symbol × period × factor × ts 维度，量级再 ×5

需要专用时序存储。

## 2. 决策

**ClickHouse 24 作为时序专用库**，与 PostgreSQL 并存：

| 数据类别 | 存储 |
|---|---|
| Tick / Bar / Factor / Signal | ClickHouse `ant` database |
| User / Account / Order / Risk / AI / Market | PostgreSQL `ant` database |

**PG → CH 边界规则**：
- 用户**写入**且需事务保证 → PG
- 系统**写入**且仅追加 → CH
- 关系查询（JOIN > 2 表）→ PG
- 时间窗聚合（GROUP BY toMonth(ts)）→ CH

## 3. 备选方案

| 方案 | 优点 | 缺点 | 否决理由 |
|---|---|---|---|
| **CH（采纳）** | 列存 / 高压缩 / TTL 原生 | 不支持事务 / 没 JOIN 优势 | 时序场景最优 |
| TimescaleDB | PG 扩展，单库 | 写入压缩比比 CH 差 5-10 倍 | 容量不够 |
| InfluxDB 2.x | 时序专长 | Flux 语法另学，集群复杂 | 心智成本高 |
| 继续 PG | 单存储 | 无法存 tick；亿行性能崩溃 | 已 v1 验证不行 |
| QuestDB | SQL 兼容 | 社区小、功能不全 | 风险大 |

## 4. 后果

### 正面
- Tick 90 天全留只占 ~150 GB（100 账户）
- bar 查询 1000 行 < 100ms（实测 alfq）
- TTL 自动清理 90 天前 tick 与 2 年前 factor
- LowCardinality(String) 把 broker/canonical 压缩到字典

### 负面
- 多一个数据库要运维（备份 / 监控 / 升级）
- 没事务（写入失败重试要靠 spill_replay 幂等）
- 不支持外键（业务约束靠应用层）
- docker-compose 多一个容器（~500MB）

### 中性
- migrations 工具用自研 chmigrate（go embed sql）
- 客户端用 ClickHouse-go v2

## 5. 实施约束

### 5.1 容器配置

`docker-compose.yml`:

```yaml
ant-clickhouse:
  image: clickhouse/clickhouse-server:24-alpine
  container_name: ant-clickhouse
  environment:
    CLICKHOUSE_DB: ant
    CLICKHOUSE_USER: ${CH_USER:-default}
    CLICKHOUSE_PASSWORD: ${CH_PASSWORD:-clickhouse}
    CLICKHOUSE_DEFAULT_ACCESS_MANAGEMENT: 1
  volumes:
    - clickhouse_data:/var/lib/clickhouse
    - ./deploy/clickhouse/config.d:/etc/clickhouse-server/config.d:ro
  ulimits:
    nofile: { soft: 262144, hard: 262144 }
  healthcheck:
    test: ["CMD-SHELL", "clickhouse-client --user ${CH_USER:-default} --password ${CH_PASSWORD:-clickhouse} --query 'SELECT 1'"]
    interval: 10s
    timeout: 3s
    retries: 5
  networks: [ant-network]

volumes:
  clickhouse_data:
    name: ant_clickhouse_data
```

### 5.2 `.env.example`

```
CH_HOST=ant-clickhouse
CH_PORT=9000
CH_USER=default
CH_PASSWORD=clickhouse
CH_DATABASE=ant
CH_POOL_SIZE=10
```

### 5.3 schema

详见 `docs/spec/13-clickhouse-schema.md` §"完整 DDL"。

### 5.4 不变量

- 业务代码**禁止**直接 `clickhouse-client` 客户端调用，必须经 `internal/storage/clickhouse/` 包
- 业务代码**禁止**写入 CH（仅 mdgateway/factorsvc/quantengine 三处生产者写）
- ConnectRPC handler 读 CH 必须有 timeout（默认 5s）

## 6. 验证方式

```bash
# 容器健康
docker inspect -f '{{.State.Health.Status}}' ant-clickhouse | grep -q healthy

# 表存在
docker exec ant-clickhouse clickhouse-client --query \
  "SELECT count() FROM system.tables WHERE database='ant'" | grep -q "^[5-9]"  # ≥5 表

# TTL 生效
docker exec ant-clickhouse clickhouse-client --query \
  "SELECT engine, ttl FROM system.tables WHERE database='ant' AND name='md_ticks'" \
  | grep -q "INTERVAL 90 DAY"

# 写入路径只在 mdgateway/factorsvc/quantengine
ALLOWED='backend/internal/(mdgateway|factorsvc|quantengine|storage/clickhouse)'
git grep -l 'INSERT INTO md_ticks\|INSERT INTO md_bars\|INSERT INTO factor_values\|INSERT INTO signals' backend/ \
  | grep -vE "$ALLOWED" \
  && { echo "FAIL: unauthorized CH writer"; exit 1; } || true
```
