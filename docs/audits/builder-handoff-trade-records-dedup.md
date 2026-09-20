# Builder Handoff — TRADE-RECORDS-DUP-1 去重施工

> 设计 SSOT：`docs/audits/design-trade-records-dedup.md`（2026-09-19 Devin CLI 设计实查）。
> 施工只执行、不决策。完成报证据等 Devin CLI 独立复审。**勿部署、勿实盘单、勿触容器/生产 DB。**

## 0. 立项一句话

trade_records 同族三伤：879 对 CST/UTC 重复平仓行 + **3,024 行 epoch-零幻影行（仍在出血：昨日 +555）** + 160 BALANCE 出入金混入。去重删行归档 + 写入方止血 + VerifyChain dedup_log 豁免。

## 施工步骤

### S1 — migration 281：`trade_record_dedup_log` 表 + 两类删除（`backend/migrations/281_trade_records_dedup.{up,down}.sql`）

**up.sql** 结构（全部包在已有 migration 事务内）：

```sql
SET LOCAL session_replication_role = 'replica';  -- 旁路 prevent_trade_delete（先例 278）

CREATE TABLE trade_record_dedup_log (
    log_id        BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    removed_id    UUID NOT NULL,        -- 原行 id
    kept_id       UUID,                 -- D1 类保留的 UTC 行 id；D2 类 NULL
    dup_class     TEXT NOT NULL,        -- 'cst_utc_8h' / 'epoch_zero'
    removed_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    removed_by    TEXT NOT NULL DEFAULT 'migration_281',
    -- 原行全列快照（含链材料）——列序/类型必须与 trade_records 物理序一致（INSERT..SELECT t.* 直插）
    id UUID, account_id UUID, ticket BIGINT,
    symbol VARCHAR(20), order_type VARCHAR(30),
    volume NUMERIC(10,4), open_price NUMERIC(18,8), close_price NUMERIC(18,8),
    profit NUMERIC(18,4), swap NUMERIC(18,4), commission NUMERIC(18,4),
    open_time TIMESTAMP, close_time TIMESTAMP,
    stop_loss NUMERIC(18,8), take_profit NUMERIC(18,8),
    order_comment VARCHAR(200), magic_number BIGINT, platform VARCHAR(10),
    created_at TIMESTAMP, updated_at TIMESTAMP,
    schedule_id UUID, user_id UUID,
    seq BIGINT, prev_hash BYTEA, entry_hash BYTEA
);
```

**D1 类（879 对，删 CST 侧 = close_time/open_time 均 +8h 的行）**——判据必须逐字采用已验证的严格签名：

```sql
WITH doomed AS (
  SELECT b.id, a.id AS kept_id
  FROM trade_records a JOIN trade_records b
    ON b.account_id = a.account_id AND b.ticket = a.ticket
   AND b.close_time = a.close_time + interval '8 hours'
   AND b.open_time  = a.open_time  + interval '8 hours'
   AND b.volume = a.volume AND b.open_price = a.open_price
   AND b.close_price = a.close_price AND b.profit = a.profit
), ins AS (
  INSERT INTO trade_record_dedup_log (removed_id, kept_id, dup_class,
    id, account_id, ticket, symbol, order_type, volume, open_price, close_price,
    profit, swap, commission, open_time, close_time, stop_loss, take_profit,
    order_comment, magic_number, platform, created_at, updated_at,
    schedule_id, user_id, seq, prev_hash, entry_hash)
  SELECT d.id, d.kept_id, 'cst_utc_8h', t.* FROM doomed d JOIN trade_records t ON t.id = d.id
  RETURNING removed_id
)
DELETE FROM trade_records t USING ins WHERE t.id = ins.removed_id;
```

**D2 类（3,024 行 epoch 幻影）**——同构 CTE，`dup_class='epoch_zero'`、`kept_id=NULL`、判据 `close_time='1970-01-01'`。

> 实测断言（迁移后核对）：D1 删除=879、D2 删除=3,024、`dedup_log` 总行=3,903。
> 幂等：二次执行两 CTE 各删 0 行。

**down.sql**：`SET LOCAL session_replication_role='replica'` → `INSERT INTO trade_records (<全列>) SELECT <快照列> FROM trade_record_dedup_log`——**seq 为 GENERATED ALWAYS AS IDENTITY，须 `OVERRIDING SYSTEM VALUE`** → `DROP TABLE trade_record_dedup_log`。

### S2 — 写入方止血（`internal/service/account_sync_service.go` + `internal/connect/system/mthub_service_orders.go`）

两处同型 `orderRecordToTradeRecord`（`:161` / `:107`）在**调用方循环**加跳过守卫（保持转换器纯映射）：

- `account_sync_service.go:99` 循环内：`if r.State != mthub.OrderStateClosed || r.CloseTime.IsZero() { continue }`；`ot := r.OrderType; if ot == mthub.OrderBalance || ot == mthub.OrderCredit { continue }`
- `mthub_service_orders.go:91` 循环同型守卫。

> 枚举坐标：`OrderStateClosed`/`OrderBalance`/`OrderCredit` 定义于 `internal/mthub/order_types.go`（adapter 已产出这些值，见 mt4 `order_history.go:55,127-135`、mt5 `order_history.go:114-116`）。

### S3 — VerifyChain dedup_log 豁免（`internal/repository/trade_record_repository.go:256-310`）

在 `prev_hash` 不匹配 `expectedPrevHash` 的 break 分支前加豁免：

```go
// deleted-link exemption: prev_hash pointing at a dedup-logged row is a
// documented removed chain segment, not tampering.
if !bytesEqual(prevHash, expectedPrevHash) {
    if isDedupLoggedHash(ctx, r.db, prevHash) { /* skip break, log info-level */ }
    else { breaks = append(...) }
}
```

`isDedupLoggedHash`：`SELECT EXISTS(SELECT 1 FROM trade_record_dedup_log WHERE entry_hash = $1)`——调用前 `WHERE entry_hash IS NOT NULL` 注意 nil 安全（`prevHash` 为 nil 时短路 false 不查库）。加载一次 per VerifyChain 调用装入 `map[string]struct{}` 亦可（3,903 行量级一次 SELECT 即可，勿逐行查询）。

### T1 — 写入守卫测试（两站点各一）

- 构造 `OrderRecord{State: OrderStateOpen, CloseTime: 零值}` 经同步路径 → 断言未写入 trade_records（fake repo 捕获或 BatchCreate 调用参数断言）。
- 构造 `State: Closed, CloseTime: 真实` → 断言写入。
- 构造 `State: Closed, CloseTime: 零值`（双层守卫判别位）→ 断言未写入。
- 构造 `OrderType: OrderBalance` → 断言未写入。

### T2 — VerifyChain 豁免测试

- 集成测试：插入链行 → 手动 DELETE 中间行（测试 tx 内 `SET LOCAL session_replication_role='replica'`）+ 写入 dedup_log 快照 → `VerifyChain` 断言无 `chain_break`（`deleted_link` 豁免生效）；删 dedup_log 记录后重跑 → 断言 `chain_break` 出现（豁免判别性）。

### T3 — 迁移幂等测试（本地 DB）

- up 执行后：两删除 CTE 计数断言；重跑 up 内 CTE → 0 行；down 执行 → `trade_records` 行数复原 + dedup_log 删除行逐字段回插（抽查 seq/prev_hash/entry_hash 一致）。

## 对抗证明规格（验收必演）

- **M1** 删 `State != OrderStateClosed` 分支 → T1 Open 态写入 → RED。
- **M2** 删 `CloseTime.IsZero()` 冗余判 → T1 零值 CloseTime 写入 → RED。
- **M3** 删豁免分支 → T2 chain_break 复现 → RED。
- **M4** D1 CTE 判据去掉 `b.open_time = a.open_time + 8h` → 生产副本重数应 **>879**（巧合对混入）→ 计数断言 RED。
- restore → 全 GREEN，工作区逐字节一致。

## 边界（不做）

- 不删 BALANCE/挂单类行；不改消费方查询；不改 VerifyChain 签名；不改 adapter；不动 `orders`/`positions`。
- 不动 hash-covered 字段（prev/seq/entry_hash 不可变是设计意图，除 dedup_log 快照与 down 复原）。
- 不加新依赖、不新建服务型文件（守卫内联在既有循环）。

## 门禁

`go build ./...` / `go vet ./...` / `go test` 影响包（`internal/mthub`、`internal/service`、`internal/connect/system`、`internal/repository`）/ `go test -race` ×3 / `go run ./tools/check-file-lines --strict` 0 errors / `git diff --check` / gofmt 触碰文件净 / migration up+down 本地实测。**勿部署，停手等 Devin CLI 复审。**
