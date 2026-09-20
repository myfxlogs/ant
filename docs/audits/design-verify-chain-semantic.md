# 设计 SSOT：VERIFY-CHAIN-SEMANTIC-1 — 交易台账 hash 链端到端语义修正

> 日期：2026-09-19（初版）→ 2026-09-20（审计修订 v2：浮点编码根因补入+写路径规范化纳入范围）
> 作者：Devin CLI ｜ 状态：设计冻结 v2（对抗审计后修订）
> 关联：TRADE-RECORDS-DUP-1（dedup_log 使全局验链可行）、VERIFY-CHAIN-SEMANTIC-1 registry 行 34

## 0. v2 修订说明（对抗审计抓出的设计错误）

初版设计判定「写路径正确、禁动」，仅修验侧视角。审计实测推翻该前提：**stored `entry_hash` 覆盖的是写时内存浮点表示（`NewFromFloat` 最短串，含 ulp 噪声），NUMERIC 列定标舍入后该表示不可重建**——验侧重算在产库只能「确认」不能「否认」，且**新行沿同病继续写入**。只修验侧视角=出厂一个对存量 32% 行与未来全部行都无力的验链器。故 v2 将「写侧 hash 输入规范化」纳入范围——属同一缺陷类（hash 链语义），非范围扩张。

## 1. 证据链（全部产库实测，非推断）

| 事实 | 证据 |
|---|---|
| 写路径取**全局尾** | `trade_record_repository.go:197` `SELECT entry_hash FROM trade_records ORDER BY seq DESC LIMIT 1`（无账户过滤），`pg_advisory_xact_lock(20827)` 串行追加——链是**全局 append-only** |
| 验链按**同账户**走 | `VerifyChain` `WHERE user_id AND account_id ORDER BY seq`，期望 prev=同账户前行——**语义错位**，产 2,113 基线假 chain_break |
| **union 视角零断点** | `trade_records ∪ trade_record_dedup_log`（`entry_hash IS NOT NULL`）按 seq 走查：14,409 行（live 11,347+archived 3,062），prev=NULL 仅 seq=13599 创世行，**global chain_breaks=0**——归档快照使被删链节完整可重建 |
| **重算编码根因** | 写侧 `computeTradeEntryHash(..., record.Volume.String(), ...)`——`record.X` 来自 `decimal.NewFromFloat(broker float)`，最短浮点串（可含 ulp 噪声）。实测穷举 seq=13601：`open_price="4324.1990000000005"`（+1ulp）命中 stored hash——列存 `4324.19900000`、回读 `"4324.199"`，写时表示**永久不可重建** |
| 重算可确认率 | union 14,409 行实测：`Decimal.String()` 编码 **9,804 可确认（68%）**、**4,605 不可确认**（live 3,952+archived 653）；`volume::text` 列范式编码 ~0%——现行 VerifyChain 用 `::text`，故产库全量失配 |
| NULL-hash 行 | live 5,209 + archived 858（链前时代行，链外=如实不可验）。现行验链无 `IS NOT NULL` 过滤，NULL 行必报 hash_mismatch=第二类基线噪音 |
| VerifyChain 调用面 | 生产**零调用**（无 RPC/proto 端点，仅 repo 方法+集成测试）——签名/语义改造安全 |
| dedup_log 快照完备 | 快照列含全部 hash 输入字段（seq/prev_hash/entry_hash/account_id/ticket/symbol/volume/三价格/profit/双时间），列类型与 live **逐格一致**（information_schema 实测 volume NUMERIC(10,4)、价格 NUMERIC(18,8)、profit NUMERIC(18,4)） |

## 2. 根因（三个，同属 hash 链语义缺陷类）

**根因 A（验侧错位）**：`WHERE account_id` 假设 per-account 链，写侧实现全局链——2,113 假 break。
**根因 B（写侧编码不可重建）**：hash 覆盖 `Decimal.String()` 写时浮点表示，NUMERIC 列不保留该表示——字段级篡改检测对存量 32% 行与未来全部行无效。
**根因 C（写侧尾读非 union——复审抓出，v3 补记）**：v2 漏点。`insertWithHashChain` 尾读仅 `trade_records`；dedup 归档可让 union 真尾高于 live 尾（产库实况：live 尾 seq=102029，归档含 seq 102070–102092 共 17 行）。此后每个新 append 的 prev 指向陈旧 live 尾，union 走查在归档尾后接缝处**永久 chain_break**。读写必须共享同一链宇宙。施工提交 `e5290433` 的 T2-T9 失败正是该潜伏缺陷首次浮出（复审实证）。

## 3. 设计决策

**D1 验链改全局 union 视角（根因 A 修正）**

```sql
SELECT seq, ticket, prev_hash, entry_hash, account_id, symbol, volume::text,
       open_price::text, close_price::text, profit::text, open_time, close_time,
       'live' AS src FROM trade_records WHERE entry_hash IS NOT NULL
UNION ALL
SELECT seq, ticket, prev_hash, entry_hash, account_id, symbol, volume::text,
       open_price::text, close_price::text, profit::text, open_time, close_time,
       'archived' FROM trade_record_dedup_log WHERE entry_hash IS NOT NULL
ORDER BY seq ASC, src ASC
```

- 每行 `prev_hash` == union 序上一已 hash 行 `entry_hash`（首行 NULL==nil 创世合法；后续 NULL prev→chain_break）
- archived 行同为链成员，同走 linkage+重算（日志表本身防篡改也受验）
- 42P01（log 不存在）→ union 仅 live 侧，语义=「无归档依据则断链按实报」
- `IS NOT NULL` 过滤消除第二类噪音；NULL-hash 行不静默丢弃——产出 `Type:"unhashed"` informational findings（审计如实披露链外存量，不混入 linkage）

**D2 写侧 hash 输入规范化（根因 B 修正）**
`insertWithHashChain` 改 hash 覆盖**列范式表示**：

```sql
INSERT INTO trade_records (...) VALUES (...)
ON CONFLICT (account_id, ticket, close_time) DO NOTHING
RETURNING id, seq, volume::text, open_price::text, close_price::text, profit::text,
          open_time, close_time
```

- `computeTradeEntryHash` 输入改取 RETURNING 的 `::text` 串（volume/open/close/profit）+RETURNING 的 open/close_time（列存时刻）+record 的 prevHash/seq/accountID/ticket/symbol（uuid/int/string 无表示损失，维持原取法）
- 效果：stored hash 覆盖=列持久化表示 → `::text` 重算**逐字可复现** → 新行 100% 字段级可验（篡改即失配，无浮点歧义）
- ON CONFLICT 行不返行不 hash——语义不变；持久化字段值零变化（hash 输入换源，落库数据不变）

**D2b 写侧尾读 union 化（根因 C 修正，v3 复审补）**

`insertWithHashChain` 的 prev_hash 尾读改为覆盖与验链相同的 union 全集：

```sql
SELECT entry_hash FROM (
    SELECT seq, entry_hash FROM trade_records
    UNION ALL
    SELECT seq, entry_hash FROM trade_record_dedup_log
) AS chain_tail ORDER BY seq DESC LIMIT 1
```

- 「prev = 写入时刻 union 序 max-seq 行 entry」不变量无条件成立——归档尾行不再使接缝断链
- **tx 内不能捕获 42P01**（失败的关系引用中止整个事务，25P02）；先 `to_regclass('trade_record_dedup_log') IS NOT NULL` 探测再选查询——pre-281/post-down 自动回退 live 侧，advisory lock 语义不变
- 并发 DDL 竞争窗口（probe 与查询间表被 drop）→ 42P01 上抛 fail-closed，不静默

**D3 双编码重算（存量行最大化确认）**
验侧每行重算**两个候选编码**（同一 `::text` 列值派生，零额外表列）：

1. 范式编码：`s` 本身（覆盖 D2 后的新行 + 存量中写时表示恰等于列范式的行）
2. 规格化编码：`decimal.RequireFromString(s).String()`（复现写侧 `NewFromFloat().String()` 形态——实测恢复 9,804/14,409）

- 任一命中→verified；皆不中→`hash_mismatch`
- 语义如实分级：**canonical-era 行失配=确定性篡改**；**legacy 行失配=篡改或浮点表示漂移（不可分）**——finding Detail 如实标注歧义，产库 legacy 不可确认基线=**4,605**（实测值，克隆库确定性断言可用）

**D4 签名与报告面**
- `VerifyGlobalChain(ctx) ([]ChainBreak, error)`：全局验链主 API
- `VerifyChain(ctx, userID, accountID)` 保留签名：全局验链→`AccountID` 过滤返回（「该账户涉及的断点」如实语义）
- `model.ChainBreak` 增 `AccountID uuid.UUID`；Detail 含 src/era 标注
- **删 `loadDedupLoggedHashes`+deleted_link 分支**——归档行本就是链成员，豁免被正确性取代

**D5 不做**
- 不改持久化数据/列结构/dedup_log 表结构/触发器
- 不接 RPC 端点（暴露面另行设计）
- 不回填 legacy 行 hash（不可重建是历史事实，回填会伪造「一直可验」假象；D2+D3 已使验链能力前向完整）

## 4. 范围边界

改：`trade_record_repository.go`（VerifyGlobalChain+VerifyChain 改造+删豁免+insertWithHashChain hash 输入换 RETURNING 源）、`model/trade.go`（ChainBreak+AccountID）、集成测试改写+新增。
不动：schema/dedup_log/proto/持久化字段语义/链续接语义。

## 5. 测试计划（integration，TEST_PG_DSN 产数据克隆或自含种子）

- **T1 产数据级**：`VerifyGlobalChain` → `chain_break` **硬断言 0**（union 自净实证）；`hash_mismatch` 计数披露——克隆库确定性断言 `==4,605`（legacy 不可确认基线实测值），计数漂移即回归信号
- **T2 篡改判别**：种子链（canonical 编码写入）UPDATE profit 不改 hash→hash_mismatch；改 prev_hash→chain_break
- **T3 未归档删除**：删链中行不入 log→follower chain_break
- **T4 归档删除自净**：删行+全列 dedup_log 归档→0 break。**harness 用 migration 281 原文全列 DDL**（子集列 harness 不满足 union 查询）
- **T5 per-account 过滤**：异账户断点→VerifyChain(acctA) 只见 acctA
- **T6 42P01**：drop log→验链不报错、断链按实报；**写路径同测**（log 缺席时 `Create` 经 live 侧回退成功——D2b probe 分支覆盖）
- **T7 legacy 双编码**：种子行手工写 `Decimal.String()` 编码 hash（如 volume `"0.05"`、列存 `"0.0500"`）→verified（规格化通道命中）；canonical-only 变体应失配（D3 承重判别）
- **T8 canonical 写路径**：insert 噪声十进制（`decimal.RequireFromString("0.30000000000000004")`、列存 `0.3000`）→`::text` 重算命中 verified（写侧规范化实证）
- **T9 unhashed 披露**：NULL-hash 种子行→`unhashed` informational finding 产出且不污染 linkage

## 6. Mutation（独立复审执行）

- **M1** union 改回 `WHERE account_id`→T1 RED（假 break 回归）
- **M2** 删规格化编码候选（canonical-only）→T7 RED（legacy 行不可确认回归）
- **M3** 删 union archived 侧→T4 RED（断点复活）
- **M4** 写侧回退 `record.X.String()`→T8 RED（噪声行失配回归）
- **M5** 删 `unhashed` 产出→T9 RED
- **M6** 写侧尾读回退 live-only（根因 C 回归）→T2-T5/T9 RED（归档尾后接缝断链复活）

## 7. 验收要点

S1 model 字段+S2 写侧规范化+S3 union 验链/双编码/删豁免+S4 测试 T1-T9；M1-M5 RED→GREEN；build/vet/影响包/check-lines/diff-check；不部署（复审后统一部署决策）。
