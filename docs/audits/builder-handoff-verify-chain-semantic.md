# 派工单：VERIFY-CHAIN-SEMANTIC-1 — 交易台账 hash 链端到端语义修正

> 日期：2026-09-19（v1）→ 2026-09-20（v2：浮点编码根因补入，写侧规范化纳入范围）→ 2026-09-20（v3：复审补根因 C 写侧尾读 union 化）
> 派工：Devin CLI ｜ 设计 SSOT：`docs/audits/design-verify-chain-semantic.md`（v3，先读）
> 基线：HEAD 含 `91ba089d`｜边界：勿部署、勿实盘单、勿触容器、禁 `--no-verify`

> **v3 复审补记（Devin CLI 审计侧执行）**：施工 `e5290433` 复审抓出根因 C——写侧尾读仅 `trade_records` 而 union 真尾可在 dedup_log（产库 live 尾 seq=102029、归档尾 seq=102092），新 append 产假 chain_break。修复已落：尾读 union 化+`to_regclass` 探测回退（tx 内不可捕获 42P01，25P02 毒化事务）；T6 补写路径回退覆盖；T7 种子改 union 尾追加；存量 hash 集成测试补 FK 父行。M1/M2/M4/M6 复审方已执行 RED→恢复 GREEN。

## 约束与目标

两个根因同修：①验侧 `WHERE account_id` 错位（产 2,113 假 break，写全局尾正确）→改 union 验链；②写侧 hash 覆盖 `NewFromFloat` 最短浮点串（含 ulp 噪声），NUMERIC 列不可重建——产库实测 14,409 已 hash 行中 `Decimal.String()` 编码仅 9,804 可重算、4,605 永久不可确认、`::text` 编码 ~0%。不修写侧=验链对未来全部行无效。目标：全局 union 验链 + 写侧 hash 输入规范化 + 双编码重算。

## S1 — model.ChainBreak 增 AccountID

`backend/internal/model/trade.go` `ChainBreak` 增 `AccountID uuid.UUID `json:"account_id"``。Type 注释补枚举：`chain_break`/`hash_mismatch`/`unhashed`（deleted_link 不再产出，注释保留历史说明）。

## S2 — insertWithHashChain hash 输入规范化（trade_record_repository.go）

INSERT 的 RETURNING 扩列，hash 覆盖**列范式表示**：

```sql
... ON CONFLICT (account_id, ticket, close_time) DO NOTHING
RETURNING id, seq, volume::text, open_price::text, close_price::text, profit::text,
          open_time, close_time
```

- `computeTradeEntryHash` 输入：prevHash/seq/accountID/ticket/symbol 维持 record 原取法；volume/openPrice/closePrice/profit 取 RETURNING `::text` 串；timeMs 取 RETURNING 的 open_time/close_time `.UnixMilli()`（列存时刻）
- ON CONFLICT ErrNoRows 早退逻辑不变；`record.EntryHash` 赋值逻辑不变
- 持久化数据零变化（hash 输入换源，落库值不变）
- **v3 补**：prev_hash 尾读改 union 全集（live ∪ dedup_log 按 seq DESC LIMIT 1）——归档尾行高于 live 尾时新行续在归档尾后不断链；先 `to_regclass('trade_record_dedup_log') IS NOT NULL` 探测（tx 内 42P01 会 25P02 毒化事务），缺席回退 live 侧

## S3 — VerifyChain 改全局 union + 双编码重算

1. 新 `VerifyGlobalChain(ctx) ([]model.ChainBreak, error)`：

```sql
SELECT seq, ticket, prev_hash, entry_hash, account_id, symbol, volume::text,
       open_price::text, close_price::text, profit::text, open_time, close_time, 'live' AS src
FROM trade_records WHERE entry_hash IS NOT NULL
UNION ALL
SELECT seq, ticket, prev_hash, entry_hash, account_id, symbol, volume::text,
       open_price::text, close_price::text, profit::text, open_time, close_time, 'archived'
FROM trade_record_dedup_log WHERE entry_hash IS NOT NULL
ORDER BY seq ASC, src ASC
```

   - linkage：expectedPrevHash=上一 union 行 entry_hash（首行 prev NULL==nil 创世合法；后续 NULL/不等→`chain_break`，Detail 附 src+seq）
   - 重算：**双候选编码**——①`s`（`::text` 原值）；②`decimal.RequireFromString(s).String()`（规格化）。任一命中=verified；皆不中→`hash_mismatch`（Detail 注明 src；新行失配=确定篡改，legacy 行失配如实标「篡改或浮点表示漂移」歧义）
   - NULL-hash 行（两侧 `entry_hash IS NULL`）：产出 `Type:"unhashed"` informational findings（seq/ticket/account/src），不参与 linkage（另行计数查询或 union 含 NULL 行走旁支——实现择一，禁静默丢弃）
   - dedup_log 不存在（42P01）→union 仅 live 侧（`to_regclass` 探测或捕获 42P01，禁吞其他错）

2. `VerifyChain(ctx, userID, accountID)` 保留签名：内部调 VerifyGlobalChain→`b.AccountID==accountID` 过滤；userID 形参注释如实标注「account_id 全局唯一，userID 保留签名稳定」。

3. **删** `loadDedupLoggedHashes`+deleted_link 分支+`pgconn` import（若仅为其用）——union 正确性取代豁免。

## S4 — 集成测试（`trade_record_global_verify_integration_test.go` 新文件，`//go:build integration`；原 dedup_verify 冲突用例改写）

- **T1 产数据级**：`VerifyGlobalChain`→`chain_break` **硬断言 0**；`hash_mismatch` 计数确定性断言 `==4,605`（克隆库 legacy 基线实测值——计数漂移即回归信号）
- **T2**：种子链（走 S2 canonical 写入）UPDATE profit 不改 hash→hash_mismatch；改 prev_hash→chain_break
- **T3**：删链中行不入 log→follower chain_break
- **T4**：删行+全列 dedup_log 归档→0 break。**harness 执行 migration 281 up.sql 原文 DDL**（子集列不满足 union 查询）
- **T5**：异账户断点→VerifyChain(acctA) 只见 acctA
- **T6**：DROP log→不报错、断链按实报
- **T7**：种子行手工写 `Decimal.String()` 编码 hash（volume `"0.05"`/列存 `"0.0500"`）→verified（规格化通道命中）——D3 承重判别
- **T8**：insert `decimal.RequireFromString("0.30000000000000004")` 噪声值（列存 `0.3000`）→`::text` 重算命中 verified——S2 规范化实证
- **T9**：NULL-hash 种子行→`unhashed` finding 产出且不污染 linkage
- 原 `trade_record_dedup_verify_integration_test.go` T3/T4（deleted_link 断言）改写归并入新 T4

## 验收

build/vet/`go test ./internal/repository ./internal/mthub ./internal/service -count=1`、integration tag（TEST_PG_DSN）、check-lines 0 errors、diff-check。完成后报证据等 Devin CLI 复审——M1-M5 mutation 由复审方执行。
