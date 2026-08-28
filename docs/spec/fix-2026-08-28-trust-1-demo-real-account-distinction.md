# FIX-2026-08-28-TRUST-1 · Demo/Real 账户区分方案

> **Status**: 🟦open（设计 SSOT，待业主决策 3 个业务问题后施工）
> **Priority**: P1（信任护城河——demo 账户虚拟金战绩与真实金混展无标注）
> **Author**: Devin CLI
> **Date**: 2026-08-28
> **关联**: AGENTS.md §1（"实盘战绩公开"是核心差异点）

## 1. 问题陈述

### 1.1 症状

Demo 账户（虚拟金）与真实金账户的战绩在市场页公开展示，**无任何标注区分**。用户无法判断一个策略的"实盘战绩"是来自真实金还是虚拟金。

### 1.2 根因（3 层）

**根因 A：`account_type` 列无写入路径**

`mt_accounts.account_type` 列（migration 011）DEFAULT `'unknown'`，但**无任何代码路径写入**：

- `UpdateAccountInfo` / `UpdateAccountInfoTx`（`account_lifecycle.go:54-81`）更新 balance/equity/margin/leverage/currency/is_investor，**不含 account_type**
- `UpdateAccountMetrics`（`account_lifecycle.go:84`）更新 runtime 指标，**不含 account_type**
- `CreateAccountTx`（`account_crud.go:68`）创建账户时**不传 account_type**
- `FetchAccountInfo`（`mt4/connection_account.go:18` / `mt5/connection_account.go:17`）读取 broker `AccountSummary` 但**丢弃 `Type` 字段**

实测：12 个 connected 账户 `account_type` 全为 `'unknown'`。

**根因 B：broker AccountSummary 的 Type 字段未被读取**

| 平台 | proto 字段 | 类型 | 值 | 当前处理 |
|------|-----------|------|-----|---------|
| MT4 | `AccountSummary.Type` | enum `AccountType` | Real=0 / Contest=1 / Demo=2 | **丢弃**（`FetchAccountInfo` 不读 `s.GetType()`） |
| MT5 | `AccountSummary.Type` | string | "real"/"demo"/... | **丢弃**（`FetchAccountInfo` 不读 `s.GetType()`） |

broker 是 demo/real 的**权威数据源**，但数据在 adapter 层被丢弃，从未到达 DB。

**根因 C：市场战绩展示无 demo/real 标注**

`marketplace_live_performance` 表（migration 214）有 `account_id` 但**无 `account_type` 列**。`LeaderboardEntry`（`leaderboard.go:12-38`）无 demo/real 字段。前端 `LeaderboardTab.tsx` / `MarketTab.tsx` 无 demo/real 标注。

`LinkLiveAccount`（`live_performance.go:42`）链接账户到策略时**不校验 account_type**——demo 账户可以链接到已发布策略，其战绩进入 leaderboard 公开展示。

### 1.3 影响范围

| 层 | 文件 | 问题 |
|----|------|------|
| adapter | `mt4/connection_account.go:48-58` | 丢弃 `s.GetType()` |
| adapter | `mt5/connection_account.go:44-54` | 丢弃 `s.GetType()` |
| model | `mdtick/mdtick.go:81-91` `MTAccountInfo` | 无 `AccountType` 字段 |
| model | `mdtick/mdtick.go:40-57` `BrokerInfo` | 无 `AccountType` 字段 |
| service | `account_lifecycle.go:38-50` `AccountInfoUpdate` | 无 `AccountType` 字段 |
| service | `account_lifecycle.go:54-81` `UpdateAccountInfo` | 不写 `account_type` |
| service | `account_crud.go:79-86` `CreateAccount` | 不传 `account_type` |
| service | `live_performance.go:42-88` `LinkLiveAccount` | 不校验 `account_type` |
| marketplace | `leaderboard.go:12-38` `LeaderboardEntry` | 无 demo/real 字段 |
| marketplace | `live_performance.go:172` `UpsertDailyPerformance` | 不记录 `account_type` |
| DB | `marketplace_live_performance` | 无 `account_type` 列 |
| 前端 | `LeaderboardTab.tsx` / `MarketTab.tsx` | 无 demo/real 标注 |

## 2. 需业主决策的 3 个业务问题

### Q1：demo 账户战绩如何处理？

**方案 A（推荐）**：**real-only**——市场 leaderboard 只展示真实金账户战绩。demo 账户不可 `LinkLiveAccount`，已链接的 demo 账户战绩从 leaderboard 排除。

- ✅ 信任护城河最强（用户看到的战绩 100% 真实金）
- ✅ 符合 AGENTS.md §1"实盘战绩公开"定位
- ⚠️ demo 账户用户无法展示战绩（但 demo 本就是测试用途）
- ⚠️ 需要清理已链接的 demo 账户

**方案 B**：**标注**——demo 账户战绩可展示但明确标注"模拟账户"。leaderboard 默认只显示 real，用户可切换查看 demo。

- ✅ 不丢失 demo 数据
- ⚠️ 标注可能被用户忽略
- ⚠️ 前端需增加切换 UI

**方案 C**：**允许**——demo 与 real 混展无标注（现状）。

- ❌ 信任风险（虚拟金战绩可能误导用户）

**Devin CLI 建议**：**方案 A**（real-only）。AGENTS.md §1 明确"实盘战绩公开"是核心差异点，demo 战绩不应出现在公开市场。

### Q2：account_type 数据源是 broker RPC 还是用户声明？

**方案 A（推荐）**：**broker RPC 权威**——连接时从 `AccountSummary.Type` 读取，写入 `mt_accounts.account_type`。每次连接/重连更新。

- ✅ 服务器唯一真相（AGENTS.md 红线）
- ✅ 无法伪造（用户不能把 demo 标成 real）
- ⚠️ broker RPC 可能返回空/异常值，需 fail-closed

**方案 B**：**用户声明**——用户创建账户时选择 demo/real，写入 DB。

- ❌ 可伪造（用户可把 demo 标成 real）
- ❌ 违反"服务器唯一真相"红线

**方案 C**：**broker 权威 + 用户可覆盖**——默认从 broker 读取，用户可申请覆盖（需审核）。

- ⚠️ 复杂，且覆盖路径可被滥用

**Devin CLI 建议**：**方案 A**（broker RPC 权威）。符合 AGENTS.md "服务器有的数据一律以服务器为唯一真相"红线。

### Q3：已存在的 12 个 `account_type='unknown'` 账户如何回填？

**方案 A（推荐）**：**重连时自动回填**——修复 `FetchAccountInfo` 读取 `Type` 字段后，下次连接/重连时 `UpdateAccountInfo` 自动写入 `account_type`。12 个 unknown 账户在下次重连时自动修正。

- ✅ 无需手动操作
- ✅ 幂等（每次连接都更新）
- ⚠️ 需要等待账户重连（可能延迟）

**方案 B**：**一次性 SQL 回填**——写脚本对 12 个账户调 `AccountSummary` RPC，批量更新 `account_type`。

- ✅ 立即生效
- ⚠️ 需要单独脚本 + 运维操作

**方案 C**：**不回填**——只修复新账户，旧账户保持 unknown。

- ❌ 旧账户继续无标注

**Devin CLI 建议**：**方案 A**（重连时自动回填）+ **方案 B**（部署后立即跑一次脚本加速回填）。

## 3. 修复方案（待业主决策后定稿）

### 3.1 前置条件

业主需对 Q1/Q2/Q3 做出决策。以下方案假设 **Q1=A / Q2=A / Q3=A+B**（Devin CLI 建议）。

### 3.2 S1：adapter 层读取 broker Type 字段

**文件**: `mt4/connection_account.go:48-58` + `mt5/connection_account.go:44-54`

```go
// MT4: AccountSummary.Type 是 enum (Real=0/Contest=1/Demo=2)
s := resp.GetResult()
return &mdtick.MTAccountInfo{
    // ...existing fields...
    AccountType: mt4AccountTypeToString(s.GetType()),  // NEW
}, nil
```

```go
// MT5: AccountSummary.Type 是 string ("real"/"demo"/...)
s := resp.GetResult()
return &mdtick.MTAccountInfo{
    // ...existing fields...
    AccountType: normalizeAccountType(s.GetType()),  // NEW
}, nil
```

**新增 helper**:
- `mt4AccountTypeToString(pb.AccountType) string`——enum → "real"/"contest"/"demo"
- `normalizeAccountType(string) string`——MT5 string 归一化为 "real"/"contest"/"demo"/"unknown"

### 3.3 S2：MTAccountInfo 加 AccountType 字段

**文件**: `mdtick/mdtick.go:81-91`

```go
type MTAccountInfo struct {
    // ...existing fields...
    AccountType string  // NEW: "real"/"contest"/"demo"/"unknown" from broker
}
```

### 3.4 S3：AccountInfoUpdate 加 AccountType 字段 + UpdateAccountInfo 写入

**文件**: `account_lifecycle.go:38-50` + `account_lifecycle.go:54-81`

```go
type AccountInfoUpdate struct {
    // ...existing fields...
    AccountType string  // NEW
}

func (s *AccountService) UpdateAccountInfoTx(ctx context.Context, p AccountInfoUpdate) error {
    _, err := p.Tx.Exec(ctx, `
        UPDATE mt_accounts SET
            balance = $3, equity = $4, credit = $5, margin = $6,
            free_margin = $7, leverage = $8, currency = $9,
            is_investor = $10, account_type = $11, updated_at = CURRENT_TIMESTAMP
        WHERE id = $1::uuid AND user_id = $2 AND deleted_at IS NULL
    `, p.ID, p.UserID, p.Balance, p.Equity, p.Credit, p.Margin, p.FreeMargin, p.Leverage, p.Currency, p.IsInvestor, p.AccountType)
    // ...
}
```

### 3.5 S4：CreateAccount 传 AccountType

**文件**: `connect/user/account_crud.go:79-86`

```go
if err := s.svc.UpdateAccountInfoTx(ctx, service.AccountInfoUpdate{
    // ...existing fields...
    AccountType: info.AccountType,  // NEW
}); err != nil {
```

### 3.6 S5：LinkLiveAccount 校验 account_type（Q1=A 时）

**文件**: `live_performance.go:42-88`

```go
func (s *Service) LinkLiveAccount(ctx context.Context, strategyID, accountID, userID string) error {
    // ...existing ownership check...

    // NEW: 校验 account_type
    var accountType string
    err = s.pg.QueryRow(ctx,
        `SELECT COALESCE(account_type,'unknown') FROM mt_accounts WHERE id = $1::uuid AND deleted_at IS NULL`,
        aid).Scan(&accountType)
    if err != nil {
        return fmt.Errorf("marketplace: account not found: %w", err)
    }
    if accountType != "real" {
        return fmt.Errorf("marketplace: only real accounts can link to published strategies (got %s)", accountType)
    }

    // ...existing link logic...
}
```

### 3.7 S6：marketplace_live_performance 加 account_type 列 + 过滤

**文件**: `migrations/XXX_marketplace_live_performance_account_type.up.sql`

```sql
ALTER TABLE marketplace_live_performance ADD COLUMN IF NOT EXISTS account_type VARCHAR(20) NOT NULL DEFAULT 'unknown';
CREATE INDEX IF NOT EXISTS idx_marketplace_live_performance_account_type ON marketplace_live_performance(account_type);
```

**文件**: `live_performance.go:172` `UpsertDailyPerformance`——INSERT 时写入 `account_type`（从 `mt_accounts` 查询）。

**文件**: `leaderboard.go:73-124` `buildLeaderboardQuery`——`return` 类型查询加 `AND lps.account_type = 'real'`（Q1=A 时）。

### 3.8 S7：前端标注（Q1=B 时，Q1=A 不需要）

如果业主选 Q1=B（标注），前端需：
- `LeaderboardTab.tsx` 加 demo/real 标签
- i18n 加 `marketplace.account_type_demo` / `marketplace.account_type_real` key
- leaderboard 查询返回 `account_type` 字段

如果 Q1=A（real-only），前端无需改动（demo 战绩不出现）。

### 3.9 S8：对抗证明

**T1**: `TestFetchAccountInfo_ReadsType_MT4`——mock MT4 AccountSummary 返回 Type=Demo → 断言 `MTAccountInfo.AccountType == "demo"`。
**T2**: `TestFetchAccountInfo_ReadsType_MT5`——mock MT5 AccountSummary 返回 Type="real" → 断言 `MTAccountInfo.AccountType == "real"`。
**T3**: `TestUpdateAccountInfo_WritesAccountType`——断言 SQL 含 `account_type = $`。
**T4**: `TestLinkLiveAccount_RejectsDemo`——account_type="demo" → 断言返回 error（Q1=A 时）。
**T5**: `TestLeaderboard_FiltersDemoAccounts`——断言 SQL 含 `account_type = 'real'`（Q1=A 时）。

### 3.10 S9：一次性回填脚本（Q3=B）

部署后执行脚本：对所有 `account_type='unknown'` 的 connected 账户调 `FetchAccountInfo`，更新 `account_type`。

## 4. 风险

| 风险 | 严重度 | 缓解 |
|------|--------|------|
| broker Type 字段返回空/异常 | 🟡 中 | `normalizeAccountType` 归一化，空值 → "unknown"，fail-closed |
| 已链接的 demo 账户需清理 | 🟡 中 | Q1=A 时需 unlink 已链接的 demo 账户（一次性 SQL） |
| MT4 enum 值与 MT5 string 不一致 | 🟢 低 | `mt4AccountTypeToString` + `normalizeAccountType` 统一归一化 |
| 前端改动范围 | 🟢 低 | Q1=A 时前端无需改动 |

## 5. 不做

- 不改 broker RPC 调用（只读 `Type` 字段，不改 `AccountSummary` 请求）
- 不改 `account_type` 列的 DEFAULT 值（保持 'unknown'，由 broker 数据覆盖）
- 不改 ADR-0013
- 不部署（施工完成后停手等 Devin CLI 复审）

## 6. 验收标准

- 机检五件套全绿
- 对抗证明 T1-T5 RED→restore→GREEN
- 部署后实测：12 个 unknown 账户全部回填为 real/demo/contest
- leaderboard 只展示 real 账户战绩（Q1=A 时）

## 7. 待业主决策清单

| # | 问题 | Devin CLI 建议 | 业主决策 |
|---|------|---------------|----------|
| Q1 | demo 账户战绩如何处理？ | A（real-only） | ⬜ |
| Q2 | account_type 数据源？ | A（broker RPC 权威） | ⬜ |
| Q3 | 12 个 unknown 账户如何回填？ | A+B（重连自动 + 脚本加速） | ⬜ |
