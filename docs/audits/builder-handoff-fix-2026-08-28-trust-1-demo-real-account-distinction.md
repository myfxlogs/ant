# 施工提示词 — FIX-2026-08-28-TRUST-1-DEMO-REAL-ACCOUNT-DISTINCTION

> **设计 SSOT**: `docs/spec/fix-2026-08-28-trust-1-demo-real-account-distinction.md`
> **base commit**: `c0f53813`
> **scope**: 单 commit，改 adapter（mt4+mt5 FetchAccountInfo+FetchBrokerInfo）+ mdtick（MTAccountInfo+BrokerInfo）+ service（AccountInfoUpdate+UpdateAccountInfoTx+UpdateAccountType）+ connect/user（CreateAccount）+ marketplace（LinkLiveAccount+UpsertDailyPerformance+recomputePerformanceSummary+leaderboard）+ 新 migration + 新测试文件
> **不做**: 不改 broker RPC 调用（只读 Type 字段）、不改 `account_type` 列 DEFAULT、不改 ADR-0013、不部署

## 立项背景

**触发**: registry TRUST-1 — demo 账户（虚拟金）与真实金账户战绩混展无标注 → 信任护城河风险。

**证据链**:
- `mt_accounts.account_type` 列（migration 011，DEFAULT 'unknown'）无任何代码路径写入——`UpdateAccountInfo`/`CreateAccountTx`/`UpdateAccountMetrics` 均不写 account_type（实测 12 账户全 'unknown'）
- broker `AccountSummary.Type` 字段（MT4 enum Real=0/Contest=1/Demo=2，MT5 string "real"/"demo"）是权威数据源，但 `FetchAccountInfo`（mt4 `connection_account.go:48` + mt5 `connection_account.go:44`）**丢弃 `s.GetType()`**
- `marketplace_live_performance` + `marketplace_live_performance_summary` 表无 account_type 列，`LinkLiveAccount` 不校验 account_type，`LeaderboardEntry` 无 demo/real 字段

**决策（Devin CLI 定稿）**: Q1=A（real-only）/Q2=A（broker RPC 权威）/Q3=A+B（重连自动回填+脚本加速）

**约束**:
- `OnBrokerInfo`（`pipeline.go:241`）调的是 `FetchBrokerInfo`（不是 `FetchAccountInfo`），`BrokerInfo` 也需加 `AccountType` 字段
- `UpdateAccountMetrics` 是 sqlc 生成（`accounts.sql.go:196`），不改签名——新增 `UpdateAccountType` 方法单独写 account_type
- `LivePerformanceCollector.cache` 是 `map[string]string`（accountID→strategyID），扩展为 `map[string]cacheEntry{strategyID, accountType}`

## S1: adapter 层读取 broker Type 字段（FetchAccountInfo + FetchBrokerInfo）

**文件**: `backend/internal/mdgateway/adapter/mt4/connection_account.go:48-58` + `backend/internal/mdgateway/adapter/mt5/connection_account.go:44-54` + `backend/internal/mdgateway/adapter/mt4/connection.go` FetchBrokerInfo + `backend/internal/mdgateway/adapter/mt5/connection.go:311-358` FetchBrokerInfo

**S1a: MT4 FetchAccountInfo（:48-58）**
```go
s := resp.GetResult()
return &mdtick.MTAccountInfo{
    // ...existing fields...
    AccountType: mt4AccountTypeToString(s.GetType()),  // NEW
}, nil
```

**S1b: MT5 FetchAccountInfo（:44-54）**
```go
s := resp.GetResult()
return &mdtick.MTAccountInfo{
    // ...existing fields...
    AccountType: normalizeAccountType(s.GetType()),  // NEW
}, nil
```

**S1c: MT4 FetchBrokerInfo**（`connection.go` 内，查 `func (g *Gateway) FetchBrokerInfo`）
```go
s := resp.GetResult()
return &mdtick.BrokerInfo{
    // ...existing fields...
    AccountType: mt4AccountTypeToString(s.GetType()),  // NEW
}, nil
```

**S1d: MT5 FetchBrokerInfo（:311-358）**
```go
s := resp.GetResult()
return &mdtick.BrokerInfo{
    // ...existing fields...
    AccountType: normalizeAccountType(s.GetType()),  // NEW
}, nil
```

**新增 helper**（放 `mdtick/mdtick.go`，adapter 共用）:
```go
// mt4AccountTypeToString maps MT4 AccountType enum to normalized string.
// pb.AccountType: Real=0, Contest=1, Demo=2.
func Mt4AccountTypeToString(t int32) string {
    switch t {
    case 0:
        return "real"
    case 1:
        return "contest"
    case 2:
        return "demo"
    }
    return "unknown"
}

// NormalizeAccountType normalizes MT5 AccountSummary.Type string.
// MT5 returns "real"/"demo"/etc.; unknown values → "unknown".
func NormalizeAccountType(s string) string {
    switch strings.ToLower(strings.TrimSpace(s)) {
    case "real":
        return "real"
    case "demo":
        return "demo"
    case "contest":
        return "contest"
    }
    return "unknown"
}
```

**注意**: helper 放 `mdtick` 包（大写导出），MT4/MT5 adapter 都 import `mdtick`。MT4 `s.GetType()` 返回 `pb.AccountType`（int32 enum），需转 int32。MT5 `s.GetType()` 返回 string。

## S2: MTAccountInfo + BrokerInfo 加 AccountType 字段

**文件**: `backend/internal/mdgateway/adapter/mdtick/mdtick.go:81-91`（MTAccountInfo）+ `:40-57`（BrokerInfo）

```go
type MTAccountInfo struct {
    // ...existing fields...
    AccountType string  // NEW: "real"/"contest"/"demo"/"unknown" from broker
}

type BrokerInfo struct {
    // ...existing fields...
    AccountType string  // NEW: "real"/"contest"/"demo"/"unknown" from broker
}
```

## S3: AccountInfoUpdate 加 AccountType + 新增 UpdateAccountType 方法

**文件**: `backend/internal/service/account_lifecycle.go:38-50`（AccountInfoUpdate）+ `:54-66`（UpdateAccountInfoTx）+ `:68-81`（UpdateAccountInfo）+ 新增 `UpdateAccountType` 方法 + `backend/cmd/server/pipeline.go:275`（OnBrokerInfo 调用点）

**S3a: AccountInfoUpdate 加字段（:38-50）**
```go
type AccountInfoUpdate struct {
    // ...existing fields...
    AccountType string  // NEW
}
```

**S3b: UpdateAccountInfoTx 写 account_type（:54-66）**
```go
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

**S3c: UpdateAccountInfo 也加 account_type（:68-81）**——同 S3b 模式。

**S3d: 新增 UpdateAccountType 方法**（OnBrokerInfo 路径，不改 UpdateAccountMetrics 签名）
```go
// UpdateAccountType updates only account_type from broker AccountSummary.Type.
// Called from OnBrokerInfo on connect/reconnect. Does NOT touch metrics
// (UpdateAccountMetrics handles those separately).
func (s *AccountService) UpdateAccountType(ctx context.Context, id, accountType string) error {
    if accountType == "" {
        return nil // skip empty (fail-closed: don't overwrite with empty)
    }
    _, err := s.db.Exec(ctx,
        `UPDATE mt_accounts SET account_type = $2, updated_at = CURRENT_TIMESTAMP
         WHERE id = $1::uuid AND deleted_at IS NULL`,
        id, accountType)
    if err != nil {
        return fmt.Errorf("service: update account type: %w", err)
    }
    return nil
}
```

**S3e: pipeline.go:275 OnBrokerInfo 调用点**（在 `UpdateAccountMetrics` 后加）
```go
_ = accountSvc.UpdateAccountMetrics(mctx, uid, accountID,
    info.Balance, info.Equity, info.Credit,
    info.Margin, info.FreeMargin, info.MarginLevel)
if info.AccountType != "" {
    _ = accountSvc.UpdateAccountType(mctx, accountID, info.AccountType)  // NEW
}
```

## S4: CreateAccount 传 AccountType

**文件**: `backend/internal/connect/user/account_crud.go:79-86`

```go
if err := s.svc.UpdateAccountInfoTx(ctx, service.AccountInfoUpdate{
    Tx: tx, UserID: userID, ID: id, Balance: info.Balance, Equity: info.Equity,
    Credit: info.Credit, Margin: info.Margin, FreeMargin: info.FreeMargin,
    Leverage: int64(info.Leverage), Currency: info.Currency, IsInvestor: info.IsInvestor,
    AccountType: info.AccountType,  // NEW
}); err != nil {
```

## S5: LinkLiveAccount 校验 account_type（Q1=A real-only）

**文件**: `backend/internal/marketplace/live_performance.go:42-88`

在 ownership check（:52-64）后、alreadyLinked check（:66-73）前，加 account_type 校验：
```go
// NEW: Q1=A real-only — only real accounts can link to published strategies.
var accountType string
err = s.pg.QueryRow(ctx,
    `SELECT COALESCE(account_type, 'unknown') FROM mt_accounts WHERE id = $1::uuid AND deleted_at IS NULL`,
    aid).Scan(&accountType)
if err != nil {
    return fmt.Errorf("marketplace: account not found: %w", err)
}
if accountType != "real" {
    return fmt.Errorf("marketplace: only real accounts can link to published strategies (got %q)", accountType)
}
```

## S6: marketplace_live_performance + summary 加 account_type 列 + 过滤

**S6a: 新 migration**
**文件**: `backend/migrations/276_marketplace_live_performance_account_type.up.sql`
```sql
-- daily 表加列
ALTER TABLE marketplace_live_performance ADD COLUMN IF NOT EXISTS account_type VARCHAR(20) NOT NULL DEFAULT 'unknown';
CREATE INDEX IF NOT EXISTS idx_marketplace_live_performance_account_type ON marketplace_live_performance(account_type);

-- summary 表加列（leaderboard 查的是 summary 表）
ALTER TABLE marketplace_live_performance_summary ADD COLUMN IF NOT EXISTS account_type VARCHAR(20) NOT NULL DEFAULT 'unknown';
```

**文件**: `backend/migrations/276_marketplace_live_performance_account_type.down.sql`
```sql
ALTER TABLE marketplace_live_performance_summary DROP COLUMN IF EXISTS account_type;
ALTER TABLE marketplace_live_performance DROP COLUMN IF EXISTS account_type;
```

**S6b: LivePerformanceCollector.cache 扩展**
**文件**: `backend/internal/marketplace/live_performance.go:341-346`（struct）+ `:354-375`（loadCache）+ `:377-396`（OnProfitUpdate）

```go
type livePerfCacheEntry struct {
    StrategyID  string
    AccountType string
}

type LivePerformanceCollector struct {
    svc   *Service
    log   *zap.Logger
    cache map[string]livePerfCacheEntry  // accountID → {strategyID, accountType}
    mu    sync.RWMutex
}
```

`loadCache`（:358-374）改为查 `mt_accounts.account_type`：
```go
rows, err := c.svc.pg.Query(ctx,
    `SELECT ms.linked_account_id::text, ms.strategy_id::text, COALESCE(ma.account_type, 'unknown')
     FROM marketplace_strategies ms
     LEFT JOIN mt_accounts ma ON ma.id = ms.linked_account_id
     WHERE ms.linked_account_id IS NOT NULL AND ms.status = 'published'`)
// ...
for rows.Next() {
    var aid, sid, at string
    if err := rows.Scan(&aid, &sid, &at); err == nil {
        c.cache[aid] = livePerfCacheEntry{StrategyID: sid, AccountType: at}
    }
}
```

`OnProfitUpdate`（:379-396）改为：
```go
func (c *LivePerformanceCollector) OnProfitUpdate(accountID string, equity, balance decimal.Decimal) {
    c.mu.RLock()
    entry, ok := c.cache[accountID]
    c.mu.RUnlock()
    if !ok {
        return
    }
    // Q1=A real-only: skip demo/contest/unknown accounts
    if entry.AccountType != "real" {
        return
    }
    // ...existing UpsertDailyPerformance call with entry.StrategyID...
}
```

**S6c: UpsertDailyPerformance 写 account_type**
**文件**: `backend/internal/marketplace/live_performance.go:240-245`

```go
_, err = tx.Exec(ctx,
    `INSERT INTO marketplace_live_performance (strategy_id, account_id, date, daily_pnl, daily_return, equity, drawdown, total_trades, winning_trades, account_type)
     VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
     ON CONFLICT (strategy_id, account_id, date)
     DO UPDATE SET daily_pnl = $4, daily_return = $5, equity = $6, drawdown = $7, total_trades = $8, winning_trades = $9, account_type = $10`,
    sid, aid, today, dailyPnL, dailyReturn, equity, drawdown, totalTrades, winningTrades, accountType)
```

**注意**: `UpsertDailyPerformance` 需要加 `accountType string` 参数，`OnProfitUpdate` 传入 `entry.AccountType`。

**S6d: recomputePerformanceSummary 写 account_type**
**文件**: `backend/internal/marketplace/live_performance.go:313-324`

```go
_, err = tx.Exec(ctx,
    `INSERT INTO marketplace_live_performance_summary
           (strategy_id, account_id, total_return, annual_return, max_drawdown, sharpe_ratio, win_rate,
            total_trades, tracking_since, last_updated, updated_at, account_type)
     VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now(), $11)
     ON CONFLICT (strategy_id) DO UPDATE SET
       total_return = $3, annual_return = $4, max_drawdown = $5, sharpe_ratio = $6, win_rate = $7,
       total_trades = $8, tracking_since = $9, last_updated = $10, updated_at = now(), account_type = $11`,
    sid, aid, totalReturn, nullDec(annualReturn), maxDrawdown,
    nullDec(sharpeRatio), nullDec(winRate),
    allTrades, firstDate, today, accountType)
```

**注意**: `recomputePerformanceSummary` 需要加 `accountType string` 参数，从 daily 表取最新或从调用方传入。

**S6e: leaderboard 过滤 real-only**
**文件**: `backend/internal/marketplace/leaderboard.go:91`

```go
// before
extraWhere = ` AND lps.strategy_id IS NOT NULL`

// after
extraWhere = ` AND lps.strategy_id IS NOT NULL AND lps.account_type = 'real'`
```

## S7: 前端标注（Q1=A 决策下不需要）

Q1=A（real-only）决策下，demo 战绩不出现，前端无需改动。

## S8: 对抗证明

**新建文件**: `backend/internal/marketplace/live_performance_trust_test.go`（marketplace 包）
**新建文件**: `backend/internal/mdgateway/adapter/mdtick/account_type_test.go`（mdtick 包）
**新建文件**: `backend/internal/service/account_type_test.go`（service 包）

**T1: `TestMt4AccountTypeToString`**（S1 helper）
```go
func TestMt4AccountTypeToString(t *testing.T) {
    if mdtick.Mt4AccountTypeToString(0) != "real" { t.Fatal("0 must be real") }
    if mdtick.Mt4AccountTypeToString(1) != "contest" { t.Fatal("1 must be contest") }
    if mdtick.Mt4AccountTypeToString(2) != "demo" { t.Fatal("2 must be demo") }
    if mdtick.Mt4AccountTypeToString(99) != "unknown" { t.Fatal("99 must be unknown") }
}
```

**T2: `TestNormalizeAccountType`**（S1 helper）
```go
func TestNormalizeAccountType(t *testing.T) {
    if mdtick.NormalizeAccountType("real") != "real" { t.Fatal() }
    if mdtick.NormalizeAccountType("DEMO") != "demo" { t.Fatal() }
    if mdtick.NormalizeAccountType("Contest") != "contest" { t.Fatal() }
    if mdtick.NormalizeAccountType("") != "unknown" { t.Fatal() }
    if mdtick.NormalizeAccountType("foo") != "unknown" { t.Fatal() }
}
```

**T3: `TestMTAccountInfo_HasAccountTypeField`**（S2 守卫）
```go
func TestMTAccountInfo_HasAccountTypeField(t *testing.T) {
    info := mdtick.MTAccountInfo{AccountType: "demo"}
    if info.AccountType != "demo" { t.Fatal("MTAccountInfo must have AccountType field (S2)") }
}
```

**T4: `TestUpdateAccountInfoTx_WritesAccountType`**（S3 守卫）
```go
func TestUpdateAccountInfoTx_WritesAccountType(t *testing.T) {
    content := mustReadFile(t, "internal/service/account_lifecycle.go")
    idx := strings.Index(content, "func (s *AccountService) UpdateAccountInfoTx")
    if idx < 0 { t.Fatal("UpdateAccountInfoTx not found") }
    body := content[idx:]
    if !strings.Contains(body, "account_type = $") {
        t.Fatal("UpdateAccountInfoTx must write account_type (S3b)")
    }
}
```

**T5: `TestLinkLiveAccount_RejectsDemo`**（S5 守卫）
```go
func TestLinkLiveAccount_RejectsDemo(t *testing.T) {
    content := mustReadFile(t, "internal/marketplace/live_performance.go")
    idx := strings.Index(content, "func (s *Service) LinkLiveAccount")
    if idx < 0 { t.Fatal("LinkLiveAccount not found") }
    body := content[idx:]
    if !strings.Contains(body, "only real accounts can link") {
        t.Fatal("LinkLiveAccount must reject non-real accounts (S5)")
    }
}
```

**T6: `TestLeaderboard_FiltersRealOnly`**（S6e 守卫）
```go
func TestLeaderboard_FiltersRealOnly(t *testing.T) {
    content := mustReadFile(t, "internal/marketplace/leaderboard.go")
    if !strings.Contains(content, "lps.account_type = 'real'") {
        t.Fatal("leaderboard return query must filter account_type = 'real' (S6e)")
    }
}
```

**T7: `TestOnProfitUpdate_SkipsDemo`**（S6b 守卫）
```go
func TestOnProfitUpdate_SkipsDemo(t *testing.T) {
    content := mustReadFile(t, "internal/marketplace/live_performance.go")
    idx := strings.Index(content, "func (c *LivePerformanceCollector) OnProfitUpdate")
    if idx < 0 { t.Fatal("OnProfitUpdate not found") }
    body := content[idx:]
    if !strings.Contains(body, "AccountType") || !strings.Contains(body, `"real"`) {
        t.Fatal("OnProfitUpdate must check AccountType == 'real' (S6b)")
    }
}
```

**helper（每个包内联定义，调整 wd 层数）**:
```go
func mustReadFile(t *testing.T, relPath string) string {
    t.Helper()
    wd, err := os.Getwd()
    if err != nil { t.Fatalf("getwd: %v", err) }
    // marketplace: .../backend/internal/marketplace → ../..
    // service:     .../backend/internal/service → ../..
    // mdtick:      .../backend/internal/mdgateway/adapter/mdtick → ../../../..
    backendRoot := filepath.Join(wd, "..", "..")  // 调整层数
    data, err := os.ReadFile(filepath.Join(backendRoot, relPath))
    if err != nil { t.Fatalf("read %s: %v", relPath, err) }
    return string(data)
}
```

## 对抗证明 RED→restore→GREEN

**RED S1**: 删 `mt4AccountTypeToString(s.GetType())` 调用 → T1 仍 GREEN（helper 还在），但 adapter 不读 Type → 需 T1 加强为断言 adapter 调用 helper。或用 T3 间接验证（MTAccountInfo 字段存在）。
**RED S3**: 将 `account_type = $11` 从 SQL 删除 → T4 RED。
**RED S5**: 删 `only real accounts can link` 校验 → T5 RED。
**RED S6e**: 删 `lps.account_type = 'real'` → T6 RED。
**RED S6b**: 删 `AccountType != "real"` 检查 → T7 RED。
**restore**: 逐项恢复 → 全测试 GREEN。

## 验收门禁

```bash
cd backend
go build ./...
go vet ./...
go test ./internal/mdgateway/adapter/mdtick/... ./internal/service/... ./internal/marketplace/...
go test -race -count=3 ./internal/marketplace/...
go run ./tools/check-file-lines --strict   # 0 errors
git diff --check                             # clean
# migration 验证
docker compose exec postgres psql -U alphaforge -c "\d marketplace_live_performance" | grep account_type
docker compose exec postgres psql -U alphaforge -c "\d marketplace_live_performance_summary" | grep account_type
```

## 红队自审 A-F

- **A 架构**: broker RPC 权威（AGENTS.md 红线"服务器唯一真相"）；`UpdateAccountType` 单独方法不改 sqlc 生成签名；`LivePerformanceCollector.cache` 扩展复用现有结构
- **B 实现**: S1 helper 归一化是最简方案；S3d `UpdateAccountType` 避开 sqlc 重新生成；S6b cache 扩展比子查询高效
- **C 洁净**: check-lines 0 errors / 无死代码 / 无 TODO / 无调试残留
- **D 正确性**: T1-T7 对抗证明；Q1=A real-only 在 LinkLiveAccount + OnProfitUpdate + leaderboard 三层过滤；空 account_type fail-closed（UpdateAccountType 跳过空值）
- **E 合规**: 符合 AGENTS.md §1"实盘战绩公开" + "服务器唯一真相"红线
- **F 文档**: registry/STATE/handover 同步

## 收工

更新 `docs/audits/tech-debt-registry.md`（`TRUST-1` + `TRUST-1-DESIGN-AUDIT` 条目状态）+ `docs/handoff/STATE.md` + `docs/audits/handover-audit-plan.md`（追加变更日志）。

**勿部署，停手等 Devin CLI 复审。**
