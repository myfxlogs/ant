# 设计 SSOT：VERIFY-CHAIN-SEMANTIC-1 — VerifyChain 全局链语义修正

> 日期：2026-09-19 ｜ 作者：Devin CLI ｜ 状态：设计冻结（自审通过）
> 关联：TRADE-RECORDS-DUP-1（dedup_log 使本设计可行）、VERIFY-CHAIN-SEMANTIC-1 registry 行 34

## 1. 证据链（实测定论，非推断）

| 事实 | 证据 |
|---|---|
| 写路径取**全局尾** | `trade_record_repository.go:197` `SELECT entry_hash FROM trade_records ORDER BY seq DESC LIMIT 1`（无 account/user 过滤），在 `pg_advisory_xact_lock(20827)` 下串行追加——链是**全局 append-only** |
| 验链按**同账户**走 | `VerifyChain(userID, accountID)` `WHERE user_id=$1 AND account_id=$2 ORDER BY seq`，期望 `prev_hash == 同账户前一行 entry_hash` |
| 错位规模 | 产库实测：14,392 已 hash 行中 **2,113 行 prev_hash 指向异账户行**（跟随全局尾写入=正确，被误判 chain_break） |
| **union 视角零断点** | `trade_records ∪ trade_record_dedup_log`（entry_hash NOT NULL）按 seq 走 lag：14,409 行，prev_hash=NULL 仅 1（链创世行），**prev_but_no_leader=0、global_breaks=0**——含被删 3,920 行的完整原链逐节吻合 |
| VerifyChain 调用面 | 生产代码**零调用**（无 ConnectRPC/proto 端点，仅 repo 方法+integration test）——改签名/语义安全 |
| dedup_log 快照完备 | 快照含 seq/prev_hash/entry_hash+全部 entry_hash 重算字段（account_id/ticket/symbol/volume/三价格/profit/双时间）——可参与链重建**且可重算验篡改** |

## 2. 根因

验链查询 `WHERE account_id` 是**错误的语义假设**（以为链是 per-account），写路径实际实现的是**全局链**（prev_hash=全局 seq 尾）。写侧正确——全局链才是 append-only 台账应有的抗篡改形态（跨账户插入也无法伪造）；验侧错位导致 2,113 基线假 break，审计能力被噪音淹没。TRADE-RECORDS-DUP-1 的 dedup_log 归档使本修正**完整可行**：被删链节可从快照重算，union 视角恢复"删除前原链"。

## 3. 设计决策

**D1 验链改全局 union 视角（主修正）**
`VerifyChain` 查询改为：

```sql
SELECT seq, ticket, prev_hash, entry_hash, account_id, symbol, volume::text,
       open_price::text, close_price::text, profit::text, open_time, close_time,
       'live' AS src FROM trade_records WHERE entry_hash IS NOT NULL
UNION ALL
SELECT seq, ticket, prev_hash, entry_hash, account_id, symbol, volume::text,
       open_price::text, close_price::text, profit::text, open_time, close_time,
       'archived' FROM trade_record_dedup_log WHERE entry_hash IS NOT NULL
ORDER BY seq ASC
```

- 每行 `prev_hash` 必须等于**union 序上一行** entry_hash（NULL==NULL 创世行合法）
- 每行 `entry_hash` 重算比对（字段篡改检测——archived 行同样重算，日志表本身防篡改也受验）
- 42P01（dedup_log 不存在=281 未跑/down 后）→ union 仅 live 侧，语义=「无归档依据则断链按实报」

**D2 签名与报告面**
- `VerifyGlobalChain(ctx) ([]ChainBreak, error)`：全局验链主 API（零调用面=自由签名）
- `VerifyChain(ctx, userID, accountID)` 保留签名但**改为**：跑全局验链→按 `account_id` 过滤返回（「该账户涉及的断点」语义——如实而非假装 per-account 链存在）
- `model.ChainBreak` 增 `AccountID uuid.UUID` 字段（过滤所需；Detail 含 src='live'/'archived' 标注断点侧）
- **删 `loadDedupLoggedHashes` + deleted_link 分支**——union 视角下归档行本就是链成员，豁免机制被正确性替代（deleted_link 类型保留于 model 注释为历史，不再产出）

**D3 NULL-hash 行处理（第二类噪音一并消除）**
union 仅含 `entry_hash IS NOT NULL` 行（链前时代行在链外=如实不可验）。**现行 VerifyChain 无此过滤**——NULL-hash 行 `entryHash=nil` vs 重算值必不等→每行报 `hash_mismatch`；产库实测 NULL-hash 行 5,209 行=第二类基线噪音（2,113 chain_break 之外；live hashed=11,347、log hashed=3,062、log NULL=858）。`IS NOT NULL` 过滤两类噪音同消。夹在链中的 NULL-hash 行不影响 linkage：已 hash 后继行的 prev_hash 指向写时全局尾（可能 NULL）——验时与 union「上一有 hash 行」比对，不一致按实报 chain_break（真实历史断裂非误报）。

**D4 不做**
- 不改写路径（全局尾正确）
- 不给 dedup_log 加 FK/触发器（审计表最小面）
- 不接 RPC 端点（VerifyChain 暴露面另行设计——本批只修语义）

## 4. 范围边界

改：`trade_record_repository.go`（VerifyChain 查询体+新方法+删豁免）、`model/trade.go`（ChainBreak+AccountID）、对应 integration test 改写。
不动：写路径、insertWithHashChain、dedup_log 表结构、任何 proto。

## 5. 测试计划（integration，TEST_PG_DSN 产数据克隆或自含种子）

- **T1 全局验链过产数据克隆**：union 走 → **0 break**（自净实证——2,113 基线噪音消失）
- **T2 篡改判别**：种子链改一行 profit（不改 hash）→ hash_mismatch 报出；改 prev_hash → chain_break
- **T3 未归档删除判别**：删链中一行（不入 dedup_log）→ follower chain_break（真篡改检出）
- **T4 归档删除自净**：删行+入 log → 0 break（union 重建原链，取代 deleted_link 豁免）
- **T5 per-account 过滤**：造异账户断点 → VerifyChain(acctA) 只见 acctA 断点
- **T6 42P01**：drop dedup_log → VerifyChain 不报错、断链按实报

## 6. Mutation（独立复审执行）

- **M1** union 改回 `WHERE account_id`（复辟错位）→ T1 产数据 RED（2,113 假 break 回归）
- **M2** 删 archived 行 entry_hash 重算 → T4 变体（log 行字段被改）RED
- **M3** 删 union 仅查 live → T4 RED（归档行不再在链→断点复活）

## 7. 验收要点

S1 union 查询+D2 签名改造+D3 删豁免+model 字段；T1-T6；M1-M3 RED→GREEN；build/vet/影响包/check-lines/diff-check；不部署（复审后统一部署）。
