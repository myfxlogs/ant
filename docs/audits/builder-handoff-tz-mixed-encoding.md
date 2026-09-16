# Builder Handoff — TZ-MIXED-ENCODING-1（同列混合时区编码根因修复）

> **任务 ID**: `TZ-MIXED-ENCODING-1`
> **立项背景**: RECONCILE-TZ-WINDOW-1 sweep §3 发现同列 CST/UTC 混合写入；2026-09-16 Devin CLI 生产库 spike 实测：`trade_records` 17780 行中 **10100 行（57%）close_time 为 CST 编码**（签名 `close_time - created_at ≈ +8.00h`，avg 精确 8.00h），open_time 同型 8892 行。live 流写入源 `pipeline_callbacks.go:162` `time.Unix(...)`（Go Local=CST）**每日仍在产生新 CST 行——持续出血**。后果：`live_performance.go:247` UTC 日界桶对 CST 行日界偏 -8h → 跨日 PnL 错归（实盘战绩公开 = 核心业务失真）；analytics UTC 参数对 CST 行窗口偏 8h。
> **设计 SSOT**: 本文件即唯一真相源（决策已定，见 registry `TZ-MIXED-ENCODING-1` 定案四条）。
> **约束与目标**: ①止血——所有 Go 写入端 `.UTC()`；②回填——签名行 UPDATE 减 8h（天然幂等）；③零 schema 变更（不改 timestamptz）。
> **边界/不做**: 不改列类型；不动查询侧（analytics/reconcile 已修）；`marketplace_settlements` 7 行 settled_at 不可分辨不回填（已裁定 negligible）；wallet_transactions 若发现 Go time.Now() 写入端一并修，但存量行不可分辨不回填（记入报告）；**不回填 open_time 无签名行**——open_time 以 close_time 签名归属同行写源。

## S1 — 写入端止血（Go `.UTC()` 全枚举）

**目标**: 所有流向 `timestamp` 列 INSERT/UPDATE 参数的 Go `time.Now()`/`time.Unix()` 统一 `.UTC()`。

**已知坐标**（必须全修，非穷举——见枚举要求）：

| 文件:行 | 现状 | 修法 |
|---|---|---|
| `cmd/server/pipeline_callbacks.go:162` | `OpenTime: time.Unix(o.UpdateOpenTime,0)` / `CloseTime: time.Unix(o.UpdateCloseTime,0)` | 各加 `.UTC()` |
| `internal/marketplace/purchase.go:100` | `exp := time.Now().Add(30*24*time.Hour)` | `time.Now().UTC().Add(...)` |
| `internal/marketplace/refund.go:184` | `settlementID, time.Now(),` 参数 | `time.Now().UTC()` |
| `internal/marketplace/refund.go:242` | `settlementID, time.Now(), reversalNote` | 同上 |
| `internal/marketplace/refund.go:247` | `settlementID, time.Now(),` | 同上 |
| `internal/marketplace/settlement.go:52` | `pid, time.Now(),` 参数 | `time.Now().UTC()` |
| `internal/marketplace/settlement.go:305` | `p.id, time.Now()` | 同上 |
| `internal/marketplace/settlement.go:267` | `now := time.Now()` | **先追用途**：流入 SQL 参数→`.UTC()`；纯 Go 比较/TTL→不改并在报告注明 |

**枚举要求（S1 验收门槛）**: `grep -rn "time.Now()\|time.Unix" backend/`（排除 `_test.go`），逐站判定：流入 pgx/SQL 参数且目标列类型为 `timestamp`（非 timestamptz）→ 修；目标列 timestamptz → 不修（参数带 zone 语义正确）；纯 Go 用途（锁 TTL/metrics/比较）→ 不修。**报告须附完整枚举表**（站数、判定、修/不修理由）。漏站 = 验收不过。

**注意**: 不得全局盲替换 `time.Now()`——纯 Go 语义站点（比较、TTL）改了虽无害但污染 diff。

## S2 — 存量回填 migration

**目标**: trade_records 的 ~10100 CST 行回正为 UTC wall clock。

新建 migration（遵循 `backend/migrations/NNN_*.up.sql`/`*.down.sql` 既有命名）：

```sql
-- up
UPDATE trade_records
SET close_time = close_time - interval '8 hours',
    open_time  = open_time  - interval '8 hours'
WHERE close_time - created_at BETWEEN interval '7 hours' AND interval '9 hours';
```

- 签名 `close_time-created_at ∈ [7h,9h]`：live 行插入滞后 ≈秒级，CST 写入=真实 UTC+8h → delta≈8h 精确命中；UTC import 行 delta≪0 不命中；**无假阳性**（真实 close 不可能在 insert 后 ~8h）。
- `open_time` 同行同写源同减（open 久远的行自身无签名，按 close_time 归属）。
- **幂等**：回填后 delta≈0 不再命中签名，重复执行安全。
- down migration：`+ interval '8 hours'` 同签名反转（注明为近似恢复）。

**部署序（报告注明）**: 本 commit 内 writer 修复 + migration 同批——migration 在 backend 启动时跑，先于新流量写行 → 顺序天然安全；新 UTC 行 delta≈0 永不被回填命中。

## S3 — pin 测试 + 对抗证明

**S3a**: `pipeline_callbacks` TradeRecord 构造路径 pin 测试——构造 stream 事件→断言产物 `OpenTime.Location()==time.UTC && CloseTime.Location()==time.UTC`（找到该构造函数/方法的可测入口；若函数不可导出/依赖重，测试落同包内）。

**S3b**: 枚举复核——`grep -rn "time.Unix(.*)\s*[,}]"` 确认无残留非 UTC epoch 转换写入 DB。

**Mutation（验收强制）**: M1 删 `pipeline_callbacks.go` 一处 `.UTC()` → S3a 测试 RED → restore → GREEN。S1 其余站点属同类机制（`time.Now().UTC()` vs `time.Now()`），逐站单测不经济——枚举表 + M1 机制证明 + gofmt/vet 编译通过为验收依据（沿用 TZ-SWEEP-AFFECTED-1 的 M2 等价说明先例）。

## 机检五件套 + 门禁

`cd backend && go build ./...` / `go vet` / `gofmt -l` / `go test` 涉及包 / `go test -race -count=3` 涉及包 / `go run ./tools/check-file-lines --strict` 0 errors。migration 文件命名与格式符合现有约定（down 文件成对）。

## 报告格式

`[施工完成:TZ-MIXED-ENCODING-1] @<hash>` + S1 枚举表（全站清单+判定）+ migration 文件名 + S3 测试名 + mutation RED→GREEN 证据 + 机检结果 + 遗留声明。

## 尾部

串行，勿部署，commit 用 ANT_ROLE=builder 前缀，完成报证据等 Devin CLI 复审。
