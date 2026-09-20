# 派工单：VERIFY-CHAIN-SEMANTIC-1 — VerifyChain 全局链语义修正

> 日期：2026-09-19 ｜ 派工：Devin CLI ｜ 设计 SSOT：`docs/audits/design-verify-chain-semantic.md`
> 基线：HEAD 含 `91ba089d`（DUP-1 部署终验）｜ 边界：勿部署、勿实盘单、勿触容器、禁 `--no-verify`

## 约束与目标

修验不修写。链是全局 append-only（写路径 `ORDER BY seq DESC LIMIT 1` 全局尾正确，**禁动**）；VerifyChain 的 `WHERE account_id` 是错误语义假设，产 2,113 基线假 break。改全局 union 视角（trade_records∪dedup_log），产库实测 union 序 14,409 行 **0 linkage break**。VerifyChain 生产零调用（无 RPC），签名改造安全。

## S1 — model.ChainBreak 增 AccountID

`backend/internal/model/trade.go` `ChainBreak` struct 增 `AccountID uuid.UUID `json:"account_id"``（import uuid 已有）。Type 注释补 'hash_mismatch/chain_break'（deleted_link 不再产出——union 视角归档行是链成员，注释保留历史说明）。

## S2 — VerifyChain 改全局 union（trade_record_repository.go）

1. 新 `VerifyGlobalChain(ctx) ([]model.ChainBreak, error)`：union 查询——

```sql
SELECT seq, ticket, prev_hash, entry_hash, account_id, symbol, volume::text,
       open_price::text, close_price::text, profit::text, open_time, close_time, 'live' AS src
FROM trade_records WHERE entry_hash IS NOT NULL
UNION ALL
SELECT seq, ticket, prev_hash, entry_hash, account_id, symbol, volume::text,
       open_price::text, close_price::text, profit::text, open_time, close_time, 'archived'
FROM trade_record_dedup_log WHERE entry_hash IS NOT NULL
ORDER BY seq ASC
```

走法同现行：expectedPrevHash=上一 union 行 entry_hash；不等→`chain_break`（Detail 附 src 与 seq）；每行 `computeTradeEntryHash` 重算≠entry_hash→`hash_mismatch`。**两侧列序/类型必须逐字一致**（log 快照列=trade_records 物理序，类型已一致）。dedup_log 不存在（42P01）→union 仅 live 侧（`loadDedupLoggedHashes` 的 42P01 处理逻辑可复用于判断表存在性：先 `to_regclass` 探一次再拼 SQL，或 TryQuery 捕获 42P01——择一，禁吞其他错）。

2. `VerifyChain(ctx, userID, accountID)` 保留签名：内部调 VerifyGlobalChain→`breaks` 按 `b.AccountID==accountID` 过滤返回；userID 形参不再参与查询（注释如实标注「account_id 全局唯一，userID 保留为签名稳定」）。

3. **删** `loadDedupLoggedHashes` + deleted_link 豁免分支 + `pgconn` import（若仅为其所用）——union 正确性取代豁免。

## S3 — integration test 改写/新增（`trade_record_dedup_verify_integration_test.go` 同目录新文件 `trade_record_global_verify_integration_test.go`，`//go:build integration`）

- **T1 产数据级自净**：对含真实链数据的库（或无种子的测试库）跑 `VerifyGlobalChain`→断言 **`chain_break` 类 0**；`hash_mismatch` 逐条 Logf 披露不硬断言（可能是真实篡改史，审计报告而非测试失败）
- **T2 篡改判别**：种子 A→B 链；UPDATE B.profit 不改 hash→hash_mismatch；改 B.prev_hash→chain_break
- **T3 未归档删除**：删 B 不入 log→C 报 chain_break
- **T4 归档删除自净**：删 B+插 dedup_log→**0 break**（豁免机制已被正确性取代——原 T3 deleted_link 断言改写为「0 entries」）。**⚠️harness 必须建全列 dedup_log**——`trade_record_dedup_verify_integration_test.go` 的子集列 harness 不满足 union 全列查询；复用 migration 281 的 CREATE TABLE 原文（执行 up.sql 内嵌 DDL 或逐字复制）
- **T5 过滤语义**：acctA/acctB 各断一点→VerifyChain(acctA) 只报 acctA 断点（AccountID 过滤实证）
- **T6 42P01**：DROP dedup_log→VerifyGlobalChain 不报错
- 原 `trade_record_dedup_verify_integration_test.go` T3/T4 与新语义冲突的改写（deleted_link 类型不再产出→断言更新为 chain_break 语义或归并入 T4）

## 验收

build/vet/`go test ./internal/repository ./internal/mthub ./internal/service -count=1`、integration tag（TEST_PG_DSN 指向克隆或测试库）、check-lines 0 errors、diff-check。完成后报证据等 Devin CLI 复审——M1-M3 mutation 由复审方执行。
