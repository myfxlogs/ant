# 设计 SSOT — TRADE-RECORDS-DUP-1 去重专项

> registry `TRADE-RECORDS-DUP-1`（行 33，🟦open）。2026-09-19 Devin CLI 设计实查。
> 结论：三类幻影/重复行同族归并清理 + 写入方止血（仍在出血）+ VerifyChain dedup_log 豁免。

## 1. 立项证据链

- 2026-09-19 部署期 migration 278 实锤：879 对 live-CST/history-UTC 重复平仓行撞 `uk_trade_record_ticket`，`NOT EXISTS` 守卫跳过后留残登记。
- 本设计实查在同族数据面上另实证**两类更大幻影**（见 §3），与 879 对同根（两写入方 + verbatim 映射）。

## 2. 链机制事实（代码实证）

| 机制 | 事实 | 坐标 |
|---|---|---|
| 写入 | `insertWithHashChain`：`prev_hash` = **全局尾** `ORDER BY seq DESC LIMIT 1`（跨账户）；`ON CONFLICT (account_id,ticket,close_time) DO NOTHING`；`entry_hash=SHA256(prev‖seq‖account‖ticket‖symbol‖volume‖open‖close‖profit‖open_ms‖close_ms)` | `internal/repository/trade_record_repository.go:187-251` |
| 验证 | `VerifyChain(user,account)` 按**同账户** seq 扫，查 `prev_hash == 前一同账户行 entry_hash` + 逐行重算 | 同文件 `:256-310` |
| 写/验错位实证 | 14,392 已 hash 行中 **14,391 prev_hash=全局前行**、仅 12,279 = 同账户前行 → 同账户校验自带基线噪音 2,113 行 | 生产实测 |
| 不可变 | `protect_trade_hash`（BEFORE UPDATE：entry_hash 一次性 NULL→非 NULL、prev/seq 全不可变）+ **`prevent_trade_delete`（BEFORE DELETE 无条件拒绝）** | migrations 263/275 |
| 调用面 | VerifyChain **生产零调用**（仅 integration test）；消费方 21 文件 ~60 查询点，BALANCE/CREDIT 已有 `order_type NOT IN` 防御过滤 | grep 实证 |

## 3. 幻影/重复行分类实测（生产，2026-09-19）

| 类 | 量 | 签名 | 字段事实 | 处置 |
|---|---|---|---|---|
| **D1 +8h 重复对** | 879 对 | 同 account+ticket，close_time 恰差 8h（CST 行 +8h） | 金额/价格/profit/品种**零分歧**；order_type 仅大小写差（CST `buy` vs UTC `BUY`）；UTC 侧 magic 更全（496 对有差）+open_time 同 +8h 编码。分类：778 双 NULL-hash / **101 双已 hash** | **删 CST 侧**（UTC 全维度更优）；实测新断点：A 类 0 / C 类 **20** |
| **D2 epoch-零幻影行** | **3,024** | `close_time='1970-01-01'`（开仓时历史记录含未平仓项被 verbatim 写入） | 全已 hash；8 账户；**仍在出血**（09-18 +555 / 09-19 +55）；2,999 已有同 ticket 真实平仓行，25 仍开仓 | 全部删 + 写入方先止血（否则同步重生）；实测新断点 **2,024** |
| **D3 BALANCE 行** | 160 | `order_type='BALANCE'`，close_time=真实操作时间，close_price=0，profit=±出入金 | 非交易但为真实资金事件；消费方已全站过滤 | **保留**（审计性+消费方免疫），登记为已知 wart；写入方今后不再写 |
| 挂单类行 | 36 | BUY_LIMIT/SELL_LIMIT/STOP | 全部真实 close_time+close_price>0——成交挂单平仓=**合法交易** | 保留 |
| 三元组 | 28 ticket | epoch+CST+UTC 三形态叠加 | 是 D1∪D2 的交集，非第四类 | 同上 |
| 合法多行 | 3,873 组 | 同 ticket 多 distinct close_time | 部分平仓等真实多事件 | 不动（非重复签名） |

## 4. 根因链（写入方止血 = 删行前提）

- **R-live（已修）**：`buildClosedTradeRecord`（`cmd/server/pipeline_callbacks.go:135`）`UpdateType=="close" && UpdateCloseTime>0` 守卫 + `.UTC()` ——现行干净。
- **R-sync1（出血源）**：`SyncAccountHistory`（`internal/service/account_sync_service.go:87-103`，pipeline.go:243/399 周期+连接触发）→ `orderRecordToTradeRecord`（`:161`）**verbatim 映射 `r.CloseTime`，无 State/IsZero 过滤**。
- **R-sync2（同病）**：`SyncOrderHistory` RPC（`internal/connect/system/mthub_service_orders.go:81-99`）→ 同名 verbatim 函数（`:107`）——**两处重复实现同缺守卫**。
- **R-adapter（如实标注但未防）**：mt4 `FetchOrderHistory:133` 与 mt5 `order_history.go:115` 均检测 `close_time=0 → State=Open` 但仍带 `CloseTime=epoch`——上游语义正确，责任在消费方过滤。
- **R-balance**：同两 verbatim 路径把 `OrderBalance/OrderCredit`（出入金）写进交易台账（现存 160 行）。

## 5. 方案（D1-D7）

- **D1 `trade_record_dedup_log` 表**（migration 281）：全列快照（含 `id/seq/prev_hash/entry_hash` 原值）+ `kept_id`（保留行 id，D1 类=UTC 行）+ `dup_class` + `removed_at` + `removed_by='migration_281'`。**删除即无损**——被删行全字段+链材料归档，可精确重建（down 迁移据此恢复）。
- **D2 D1 类删 CST 侧 879 行**：判据 `b.close_time = a.close_time + interval '8 hours' AND b.created_at <> a.created_at`（与 278 的 NOT EXISTS 同型，确定性）。无 merge——UTC 侧字段已全维度 ≥CST 侧（唯一例外 comment 7 对差异，dedup_log 全量保真）。
- **D3 D2 类删 epoch 行 3,024 行**：判据 `close_time='1970-01-01'`。含 25 仍开仓行——开仓态归 positions 表，closed-trade 台账不应承载。
- **D4 触发器旁路**：`SET LOCAL session_replication_role='replica'`（migration 事务内，superuser `ant` 可用；先例 278）。DELETE 不受 `protect_trade_hash`（UPDATE-only）但受 `prevent_trade_delete`——必须旁路。
- **D5 VerifyChain dedup_log 豁免**：`prev_hash ∉ {expectedPrevHash}` 时若 `prev_hash ∈ (SELECT entry_hash FROM trade_record_dedup_log)` → 记 `deleted_link` 信息级条目（非 break）——被删链位可证伪可审计，豁免边界=仅指向已知删除行。**hash_mismatch 自洽校验不动**。基线噪音（写全局/验同账户错位 2,113 行）为既有独立债另立 `VERIFY-CHAIN-SEMANTIC-1`，不在本批。
- **D6 写入方止血**（本批核心防护）：两 `orderRecordToTradeRecord` 站点统一守卫——`r.State != mthub.OrderStateClosed || r.CloseTime.IsZero() → skip`；`ot ∈ {OrderBalance, OrderCredit} → skip`（出入金不入交易台账）。
- **D7 down 迁移**：`INSERT INTO trade_records SELECT <原列> FROM trade_record_dedup_log`（id/seq/prev_hash/entry_hash 原值回插 → 链精确复原）+ `DROP TABLE`。触发器旁路同上。

## 6. 边界（不做）

- 不动 21 个消费方查询（删行后自动正确；BALANCE 行保留故过滤逻辑不变）。
- 不改 VerifyChain 每账户签名/语义（仅加豁免分支；全局语义修正另立项）。
- 不删 BALANCE 行（真实资金事件+消费方免疫）。
- 不动 `orders`/`positions` 表、不动 mtapi adapter（上游标注语义正确）。
- 不重算/回填任何已 hash 行字段（hash-covered 字段不可变是设计意图）。

## 7. 对抗证明规格（验收必演）

- **M1** 删 D6 守卫 `State != Closed` 分支 → 测试构造 Open 态记录被写入 → RED。
- **M2** 删 `CloseTime.IsZero()` 冗余判 → 构造 State=Closed 但 CloseTime 零值记录写入 → RED（双层守卫判别）。
- **M3** 删 VerifyChain 豁免分支 → 测试制造 dedup_log 记录+悬空 prev_hash → RED（chain_break 复现）。
- **M4** 迁移重跑幂等：同一 CTE 二次执行 0 行删除（幂等性钉住）。
- 测试内删除走 `session_replication_role` 旁路（与生产迁移同径），禁直接改触发器。

## 8. 门禁

`go build`/`go vet`/`go test` 影响包/`race`/`check-file-lines 0 errors`/`git diff --check`；migration up/down 本地实测（up 后行数=预期删除数、dedup_log 行数=删除数、down 后全复原+VerifyChain 行为一致）。**勿部署，停手等 Devin CLI 复审。**
