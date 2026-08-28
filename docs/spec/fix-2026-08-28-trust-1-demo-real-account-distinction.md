# FIX-2026-08-28-TRUST-1 · Demo/Real 账户区分方案

> **Status**: ✅定稿（设计 SSOT，Devin CLI 决策 Q1=A/Q2=A/Q3=A+B，待施工）
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

## 2. 业务决策（Devin CLI 定稿）

### Q1：demo 账户战绩如何处理？→ **决策 A（real-only）**

**现状**：demo 与 real 混展无标注。

**决策**：**real-only**——市场 leaderboard 只展示真实金账户战绩。demo 账户不可 `LinkLiveAccount`，已链接的 demo 账户战绩从 leaderboard 排除。

**理由**：AGENTS.md §1 明确"实盘战绩公开"是核心差异点，demo 战绩不应出现在公开市场。信任护城河最强（用户看到的战绩 100% 真实金）。demo 本就是测试用途，无需展示战绩。已链接的 demo 账户需一次性 unlink（S5 部署后 SQL 清理）。

### Q2：account_type 数据源？→ **决策 A（broker RPC 权威）**

**现状**：`account_type` 列无写入路径，12 账户全 'unknown'。

**决策**：**broker RPC 权威**——连接时从 `AccountSummary.Type` 读取，写入 `mt_accounts.account_type`。每次连接/重连更新。

**理由**：AGENTS.md 红线"服务器有的数据一律以服务器为唯一真相"。broker 是 demo/real 的权威数据源（MT4 enum Real=0/Contest=1/Demo=2，MT5 string "real"/"demo"），无法伪造。空/异常值归一化为 'unknown'，fail-closed。

### Q3：12 个 unknown 账户如何回填？→ **决策 A+B（重连自动 + 脚本加速）**

**现状**：12 个 connected 账户 `account_type='unknown'`。

**决策**：**重连时自动回填**（S1-S4 修复后下次连接/重连时 `UpdateAccountInfo` 自动写入）+ **一次性脚本加速回填**（部署后立即跑脚本对 12 个账户调 `FetchAccountInfo`，批量更新）。

**理由**：重连自动回填无需手动操作且幂等；脚本加速回填立即生效，不必等待账户重连。

## 3. 修复方案（已定稿，Q1=A / Q2=A / Q3=A+B）

### 3.1 前置条件

决策已由 Devin CLI 定稿（§2）。

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

### 3.7 S6：marketplace_live_performance + summary 加 account_type 列 + 过滤

**文件**: `migrations/XXX_marketplace_live_performance_account_type.up.sql`

```sql
-- daily 表加列
ALTER TABLE marketplace_live_performance ADD COLUMN IF NOT EXISTS account_type VARCHAR(20) NOT NULL DEFAULT 'unknown';
CREATE INDEX IF NOT EXISTS idx_marketplace_live_performance_account_type ON marketplace_live_performance(account_type);

-- summary 表加列（审计 finding #1 修复：leaderboard 查的是 summary 表，必须加列才能过滤）
ALTER TABLE marketplace_live_performance_summary ADD COLUMN IF NOT EXISTS account_type VARCHAR(20) NOT NULL DEFAULT 'unknown';
```

**文件**: `live_performance.go:172` `UpsertDailyPerformance`——INSERT 时写入 `account_type`。

**审计 finding #1 修复**：`UpsertDailyPerformance` 当前不查 `mt_accounts`，需要先查 account_type 再传入 INSERT。两种方式：
- 方式 A（推荐）：`OnProfitUpdate`（`live_performance.go:379`）从 `LivePerformanceCollector.cache` 已有 accountID → strategyID 映射，扩展 cache 存 account_type，`OnProfitUpdate` 传入 account_type。
- 方式 B：`UpsertDailyPerformance` 内部子查询 `(SELECT COALESCE(account_type,'unknown') FROM mt_accounts WHERE id = $2)`。

**方式 A 更优**：cache 已存在，扩展一个字段比每次 INSERT 加子查询更高效。

**文件**: `live_performance.go:314` `recomputePerformanceSummary`——重算 summary 时写入 account_type（从 daily 表取最新）。

**文件**: `leaderboard.go:73-124` `buildLeaderboardQuery`——`return` 类型查询加 `AND lps.account_type = 'real'`（Q1=A 时）。

### 3.8 S7：前端标注（Q1=A 决策下不需要）

Q1=A（real-only）决策下，demo 战绩不出现，前端无需改动。leaderboard 查询过滤 `account_type = 'real'` 即可。

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

## 7. 决策清单（Devin CLI 定稿）

| # | 问题 | 决策 | 理由 |
|---|------|------|------|
| Q1 | demo 账户战绩如何处理？ | A（real-only） | AGENTS.md §1"实盘战绩公开"，信任护城河最强 |
| Q2 | account_type 数据源？ | A（broker RPC 权威） | AGENTS.md 红线"服务器唯一真相"，无法伪造 |
| Q3 | 12 个 unknown 账户如何回填？ | A+B（重连自动+脚本加速） | 重连自动幂等，脚本立即生效 |
