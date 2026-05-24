# ADR-0011 · 容量调优 + Vault 轮换 + Normalizer 缓存失效

- **状态**：Accepted
- **日期**：2026-05-24
- **决策者**：架构组
- **关联 spec**：`docs/spec/11-mdgateway.md` §4 §9 §13.6 / `docs/spec/17-secrets-and-errors.md`
- **关联 ADR**：ADR-0005

## 1. 背景

数据基础在 100 账户峰值场景下三个潜在故障点：

### 1.1 CHWriter 容量配置过保守（M-3）

当前默认：
```
QueueSize: 5000
MaxBatchSize: 1000
FlushInterval: 1s
```

100 活跃账户 × 5 品种 × 50 tick/s（欧盘 EURUSD / NFP 高峰）= 25k tick/s。`QueueSize=5000` 在 200ms 内打满 → 全部走 spill。spill 是 jsonl 顺序 fsync，单盘 IOPS 上限 ~5k；雪崩可能再触发"spill 失败 → 熔断"（ADR-0005 §spill-failed），错误传播放大。

### 1.2 Vault master key 单点（L-5）

`ANT_MASTER_KEY` 单一 env 变量保护所有 mt_account 加密字段。一旦泄漏：
- 全部账户密码/token 解密，无法 revoke
- 无 key rotation 流程

### 1.3 Normalizer cache 失效不及时（L-4）

`normalizer.go` algorithmic fallback 缓存 1h TTL。运营在 PG `broker_symbols` 加新映射后，最长 1h 才生效。期间该 broker 该 symbol 全部走 fallback，写错 canonical → md_ticks/md_bars 数据"分裂"。

## 2. 决策

### 2.1 CHWriter 调优 + Buffer engine 中间层

**配置层**（默认调整，可由 env 覆盖）：

| 参数 | 旧 | 新 | 说明 |
|---|---|---|---|
| `QueueSize` | 5000 | 50000 | 25k tick/s 下 2s 缓冲 |
| `MaxBatchSize` | 1000 | 10000 | 单次 INSERT 大批量降低 CH merge 压力 |
| `FlushInterval` | 1s | 500ms | 队列大但延迟降一半 |

**架构层**：CH 端引入 `Buffer` engine 表 `md_ticks_buffer`（spec/13 §2.7 新建），CHWriter INSERT 写 buffer 表，CH 内部异步 flush 到 `md_ticks`。优势：
- 应用层 batch 与 CH 内部 batch 解耦
- CH 进程内合并小批，磁盘 IOPS 压力降至 1/10
- buffer 满时自动 flush，无需应用感知

风险：CH 进程 OOM 时 buffer 数据丢。但与现有"CH down → spill"路径不冲突（CH 进程崩了 INSERT 直接报错，应用走 spill）。

### 2.2 Envelope encryption + Key rotation

`internal/secrets/` 升级为双层架构：

```
Master Key (KEK)              ← 来自 KMS / age / 文件系统（生产强烈建议 KMS）
   │
   └─ AES-Wrap encrypts
        │
        ▼
Data Key (DEK)                ← 每个加密字段一份，随密文存储
   │
   └─ AES-256-GCM encrypts
        │
        ▼
Plaintext (password / token)
```

存储格式（PG `mt_accounts.password_encrypted bytea`）：
```
[1B version][16B dek_kid][12B nonce][N B wrapped_dek][M B ciphertext][16B tag]
```

接口：
```go
type Vault interface {
    Encrypt(ctx context.Context, plaintext []byte) ([]byte, error)
    Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error)
    Rotate(ctx context.Context) error  // 生成新 master key 版本，逐步重加密
}
```

Master key 来源（按优先级）：
1. `ANT_KMS_KEY_ID` env → AWS KMS / GCP KMS / Azure Key Vault
2. `ANT_MASTER_KEY_FILE` env → 文件系统（age 加密文件，启动时人工解锁）
3. `ANT_MASTER_KEY` env（仅开发/测试，启动 log warn）

Rotation 流程：
- `cmd/ant-vault rotate` CLI：生成新 master key version → 后台逐行 re-encrypt → 标记旧 version 为 retired
- 30 天后删除旧 version

### 2.3 Normalizer cache 主动失效

新增 `internal/mdgateway/normalizer_invalidator.go`：
- PG 触发器在 `broker_symbols` INSERT/UPDATE/DELETE 时 `NOTIFY broker_symbols_changed`（payload = JSON 含 broker, symbol_raw）
- normalizer 启动时建立 `pgx.Conn.LISTEN broker_symbols_changed`
- 收到通知 → `cache.Remove(broker:symbol_raw)`，下次查询走 PG

降级：LISTEN 连接断开时回退到 30s ticker 轮询 `MAX(updated_at)` 兜底。

## 3. 备选方案

| 子项 | 备选 | 否决 |
|---|---|---|
| Buffer engine | 应用层加更多内存队列 | 复杂度上升、监控难；CH 原生 Buffer engine 久经考验 |
| Buffer engine | Kafka/Redpanda 中间层 | 引入运维负担；当前规模过度设计 |
| Envelope encryption | 升级 ANT_MASTER_KEY 为 32B + log 警告 | 不解决泄漏后无法 revoke 问题 |
| Envelope encryption | HashiCorp Vault Server | 单机部署阶段引入额外服务过重；接口预留即可 |
| Cache 失效 | 缩短 TTL 到 30s | 永远 30s 延迟；PG 查询频率 ×120 |
| Cache 失效 | 重启 mdgateway | 运维痛点；不可接受 |

## 4. 后果

- **正面**：100 账户峰值不丢数据；密钥可轮换；canonical 配置秒级生效
- **负面**：
  - Buffer engine 引入额外表，CH 资源占用 +5%
  - Vault 接口变化，已加密数据需 re-encrypt 迁移（一次性脚本，约 10 分钟）
  - PG LISTEN 长连接占用 1 个 conn slot
- **中性**：CHWriter 默认配置变化，env 覆盖优先级保持

## 5. 实施约束

1. `internal/storage/clickhouse/buffer.go` 新增 `EnsureBufferTable(name)`，启动时 idempotent 创建 `md_ticks_buffer`
2. `clickhouse_writer.go` INSERT target 改为 `md_ticks_buffer`（barwriter 同理 → `md_bars_buffer`）
3. `internal/secrets/vault.go` 重构为 envelope；旧实现保留为 `vault_legacy.go`，启动时检测格式自动迁移
4. 一次性 CLI `cmd/ant-vault-migrate/main.go`：扫 `mt_accounts.password_encrypted` 旧格式 → 解密 → envelope encrypt 重写
5. `migrations/102_broker_symbols_notify.up.sql`：CREATE FUNCTION + TRIGGER（`pg_notify`）
6. `normalizer.go` 接受可选 `cancel <-chan struct{}` 用于 listener 注入
7. spec/11 §4 §9 §13.6 同步更新
8. spec/17 §1 envelope 设计图

## 6. 验证方式

```bash
# 1. CHWriter 默认配置
grep -E 'QueueSize.*50000|MaxBatchSize.*10000|FlushInterval.*500' \
  backend/internal/mdgateway/clickhouse_writer.go

# 2. Buffer engine 表存在
docker exec ant-clickhouse clickhouse-client --query \
  "SELECT engine FROM system.tables WHERE database='ant' AND name='md_ticks_buffer'" \
  | grep -q '^Buffer$'

# 3. 100 账户负载测试（mock broker）：5 分钟无 spill
go test -tags=loadtest -timeout 10m ./tests/loadtest/ -run Test100AccountsNoSpill
# 验证：md_spill_writes_total 全程 == 0

# 4. Envelope encryption 格式
go test -race -cover ./internal/secrets/... | awk '/coverage:/{gsub("%",""); if ($2<90) exit 1}'
# 单测：encrypt → decrypt 还原；旧格式自动识别迁移

# 5. Vault rotation
go run ./cmd/ant-vault rotate --dry-run | grep -q 'rows_to_rewrite'
go run ./cmd/ant-vault rotate
# 验证：所有 mt_accounts 字段 dek_kid 更新为新 version

# 6. PG NOTIFY 触发 cache 失效
go test -tags=integration ./internal/mdgateway/ -run TestNormalizerInvalidation -v
# 测试：UPDATE broker_symbols → 100ms 内 normalizer.cache miss

# 7. LISTEN 断线降级到 ticker
go test -tags=integration ./internal/mdgateway/ -run TestNormalizerListenerFallback -v
```
