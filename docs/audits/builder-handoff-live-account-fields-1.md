# Builder Handoff — LIVE-ACCOUNT-FIELDS-1

> live runner `Account()` 未填充 Leverage/Currency/Company/Mode —— live 管线补全。
> 设计 SSOT：Devin CLI。施工只执行、不决策。完成报证据等独立复审。

## 0. 立项证据链（设计实查已核实）

**缺陷面**：`backend/strategy/runner/broker.go:203-219` harness 模式（`b.executor==nil`）`Account()` 只填 Balance/Equity/Margin/FreeMargin/Login/IsDemo/IsConnected/IsTradeAllowed——`Leverage→0`、`Currency/Company/Mode→零值`。回测路径不受影响（`backtest/broker.go:653-664` SimBroker 已填 Leverage=配置、Currency="USD" venue 常量）。

**VM 消费端实况**（`tools/mql2go/`）：

| 消费点 | 字段 | 缺失时行为 |
|---|---|---|
| `vm_builtin_account.go:29-31` `AccountLeverage()` | Leverage | **静默返回 0**（本债主要失真） |
| `vm_builtin_mql5_info.go:106-108` `AccountInfoInteger(ACCOUNT_LEVERAGE=2)` | Leverage | 静默 0 |
| `vm_builtin_account.go:78-82` `AccountFreeMarginCheck` / `:270-273` `MarketInfo(MODE_MARGININIT/REQUIRED)` | Leverage | 0→error（已 fail-closed） |
| `vm_builtin_account.go:39-45` `AccountCurrency()` / `vm_builtin_mql5_info.go:165-169` `AccountInfoString(ACCOUNT_CURRENCY)` | Currency | ""→error（已 fail-closed） |
| `vm_builtin_account.go:47-53` `AccountCompany()` | Company | ""→error（已 fail-closed） |
| `vm_builtin_mql5_info.go:116-133` `ACCOUNT_MARGIN_MODE=7`/`ACCOUNT_HEDGE_ALLOWED=10` | Mode | ""→error（已 fail-closed） |

**权威源核实**：

| 字段 | 权威源 | 现状 |
|---|---|---|
| Leverage | `mt_accounts.leverage`（`service/account_lifecycle.go:72` `UpdateAccountInfo` 从 mtapi AccountSummary 同步；`PositionSnapshot.Leverage` 已流） | proto 无字段 |
| Currency | `mt_accounts.currency`（同上同步） | proto 无字段 |
| Company | `mt_accounts.broker_company` | **proto `company=27` 已存在且 `live_context.go:273` 已填充**——只差 runner 侧接线 |
| Mode(hedging/netting) | MT4：平台语义恒 hedging（MT4 无 netting）；MT5：mtapi `AccountSummary.Method`(AccMethod enum) **存在但 adapter 未映射**（`mdtick.BrokerInfo/MTAccountInfo/ProfitUpdate` 三结构全无该字段，`mt5/connection.go:345-358` 只映射 9 字段） | proto 无字段；`mt_accounts.account_method`="master"\|"investor" 是 investor 标志**非** margin mode，不可用 |

**关键语义边界**：proto `LiveStrategyContext.mode=13` 是执行模式 `"live"|"paper"`（`vm_live_dispatch.go:92` 用于 `runner.Config.Mode`），与 `sdk.AccountInfo.Mode`（`"hedging"|"netting"`，sdk/types.go）**完全不同**——禁止直接映射。

**管线拓扑**（全部实核）：服务端 `live_context.go:injectAccountTruth:256-302`（每字段单 lookup，`lookupBool` helper，live 全 lookup 必须成功 fail-closed / paper 容忍）→ proto `LiveStrategyContext` → 三处 init/OnBar 接线点（`vm_live_dispatch.go:106-109`、`vm_live_session.go:118-121`、`vm_live_handlers.go:30-34`——恰好是现有 `SetLogin`+`SetAccountStatus` 同点位）→ `runner.Set*` → `ctx.live*` → `brokerImpl.Account()`。tick/trade/timer 路径（`vm_live_handlers.go:140/170/251`）只调 `UpdateLiveState` 更新金融量，**不设身份字段**——本债同样不接。

**REUSE/NEW**：`bash scripts/cap.sh Leverage/AccountInfo` 已跑——risk.LeverageCap（订单意图风控，语义不同）、UpdateAccountInfo（写方向）均不可复用。新增 = NEW：`accountIdentityLookup`（单行三列一次查询，替代 3 个单列 lookup——减少 2 次 DB roundtrip，签名仍是可 stub 的 `func(ctx,accountID)(T,error)` 惯例）、`SetAccountIdentity`（身份 setter 聚合，与 SetLogin/SetAccountStatus 同层）。复用 = REUSE：`brokerCompanyLookup`（已在调）、`lookupBool` 惯例、`injectAccountTruth` 注入点。

**边界外登记（不动）**：
- MT5 `AccMethod` adapter 映射（`mdtick.BrokerInfo`+mt5 adapter+snapshot 流向）→ 另债，本债 MT5 `account_mode` 如实置 `""` 保持 VM fail-closed。
- `AccountInfoDouble(ACCOUNT_MARGIN_LEVEL=6)` `vm_builtin_mql5_info.go:60-64` 返回 `Equity/Margin` 比率——真实 MQL5 报**百分比**（疑 100× 偏差），另债复核。
- `mt_accounts.account_method` 语义是 investor 标志（`mdgateway/runner_gateway.go:175-178` 写入 "master"/"investor"），勿当 margin mode 用。

## S1 — proto 契约扩展（`proto/ant/v1/strategy_runtime.proto`）

在 `LiveStrategyContext` 末尾（`is_trade_allowed = 30` 后，`strategy_runtime.proto:264-267` 区域）追加：

```proto
  // LIVE-ACCOUNT-FIELDS-1: account identity from mt_accounts (server-side
  // authoritative, same trust class as login/company/is_demo).
  // leverage: account leverage (e.g. 100 = 1:100); 0 = not populated.
  // currency: deposit currency (e.g. "USD"); "" = not populated.
  // account_mode: margin mode "hedging"|"netting"; "" = unknown — distinct
  // from mode (field 13) which is execution mode "live"|"paper". MT5 margin
  // mode pending AccMethod adapter wiring; VM consumers fail-closed on "".
  int32 leverage = 31;
  string currency = 32;
  string account_mode = 33;
```

然后 `make proto` 重新生成（buf generate → `backend/gen/proto/ant/v1/` + `frontend/src/gen/ant/v1/`）。生成物 diff 一并提交。**字段号 31/32/33 已核实未占用**。

## S2 — 服务端注入（`internal/connect/strategy/`）

**S2a** `strategy_execution_handler.go`（`accountIsInvestorLookup` 声明 `:124` 之后）：

```go
// AccountIdentity is the leverage/currency/platform triple read from one
// mt_accounts row. LIVE-ACCOUNT-FIELDS-1.
type AccountIdentity struct {
    Leverage int32
    Currency string
    MTType   string // "mt4" | "mt5"
}
// accountIdentityLookup resolves leverage/currency/mt_type in one query.
accountIdentityLookup func(ctx context.Context, accountID string) (*AccountIdentity, error)
```

并在 `SetAccountIsInvestorLookup`（`:210-212`）后加 `SetAccountIdentityLookup` setter。

**S2b** `live_context.go injectAccountTruth`（investor gating 块 `:297-299` 之前）：

```go
// LIVE-ACCOUNT-FIELDS-1: leverage/currency/account-mode from one mt_accounts row.
if s.accountIdentityLookup != nil {
    ident, err := s.accountIdentityLookup(ctx, cfg.AccountID)
    if err != nil {
        if cfg.Mode == modeLive {
            return fmt.Errorf("account identity lookup failed: %w", err)
        }
    } else if ident != nil {
        lctx.Leverage = ident.Leverage
        lctx.Currency = ident.Currency
        lctx.AccountMode = accountModeForMTType(ident.MTType)
    }
}
// Live identity completeness: leverage<=0 / empty currency means the
// authoritative row was never synced — fail closed instead of serving
// fabricated zeros. account_mode "" is tolerated (MT5 unknown → VM
// consumers fail-closed); company "" tolerated (VM fail-closed too).
if cfg.Mode == modeLive {
    if lctx.Leverage <= 0 {
        return fmt.Errorf("account leverage missing or non-positive: %d", lctx.Leverage)
    }
    if lctx.Currency == "" {
        return fmt.Errorf("account currency missing")
    }
}
```

helper（`live_context.go` 文件尾）：

```go
// accountModeForMTType derives margin mode from platform semantics.
// MT4 is hedging-only (platform truth, not a lookup). MT5 margin mode is
// per-account broker config — mtapi exposes it via AccountSummary.Method
// but the adapter does not surface it yet; "" = unknown → VM fail-closed.
func accountModeForMTType(mtType string) string {
    if mtType == "mt4" {
        return "hedging"
    }
    return ""
}
```

## S3 — lookup 实现（`cmd/server/handlers_strategy.go`）

`SetAccountIsInvestorLookup`（`:206-215`）之后：

```go
srv.SetAccountIdentityLookup(func(ctx context.Context, accountID string) (*strategy.AccountIdentity, error) {
    var ident strategy.AccountIdentity
    err := pool.QueryRow(ctx,
        `SELECT COALESCE(leverage,0), COALESCE(currency,''), COALESCE(mt_type,'') FROM mt_accounts WHERE id = $1::uuid AND deleted_at IS NULL`,
        accountID).Scan(&ident.Leverage, &ident.Currency, &ident.MTType)
    if err != nil {
        return nil, fmt.Errorf("identity lookup: %w", err)
    }
    return &ident, nil
})
```

（import 别名按文件现有 `strategy` 包引用方式；若该文件对 connect/strategy 包的引用名不同，沿用文件内既有名。）

## S4 — runner 侧接线（`strategy/runner/`）

**S4a** `context.go` `liveIsTradeAllowed` 字段（`:40`）后、`// Symbol info` 注释块（`:42`）前加：

```go
liveLeverage    int32  // LIVE-ACCOUNT-FIELDS-1
liveCurrency    string // LIVE-ACCOUNT-FIELDS-1
liveCompany     string // LIVE-ACCOUNT-FIELDS-1
liveAccountMode string // "hedging"|"netting"|""(unknown→VM fail-closed)
```

**S4b** `runner.go` `SetAccountStatus`（`:106-117`）后加：

```go
// SetAccountIdentity sets leverage/currency/company/account-mode for
// harness mode. Called alongside SetLogin/SetAccountStatus at init and
// OnBar dispatch. LIVE-ACCOUNT-FIELDS-1.
func (r *Runner) SetAccountIdentity(leverage int32, currency, company, accountMode string) {
    r.ctx.mu.Lock()
    defer r.ctx.mu.Unlock()
    r.ctx.liveLeverage = leverage
    r.ctx.liveCurrency = currency
    r.ctx.liveCompany = company
    r.ctx.liveAccountMode = accountMode
}
```

**S4c** `broker.go Account()`（`:208-217` 字面量内）追加字段 + 白名单映射 helper：

```go
Leverage:       b.runner.ctx.liveLeverage,
Currency:       b.runner.ctx.liveCurrency,
Company:        b.runner.ctx.liveCompany,
Mode:           sdkAccountMode(b.runner.ctx.liveAccountMode),
```

```go
// sdkAccountMode whitelists the proto string into sdk.AccountMode —
// unrecognized values ("" / garbage) map to "" so VM consumers fail-closed.
func sdkAccountMode(s string) sdk.AccountMode {
    switch sdk.AccountMode(s) {
    case sdk.ModeHedging, sdk.ModeNetting:
        return sdk.AccountMode(s)
    }
    return ""
}
```

**S4d** 三处接线（与 SetLogin/SetAccountStatus 同行点位）：
- `vm_live_handlers.go` OnBar 块 `r.SetAccountStatus(lctx.IsDemo, ...)`（`:34`）后：`r.SetAccountIdentity(lctx.Leverage, lctx.Currency, lctx.Company, lctx.AccountMode)`
- `vm_live_dispatch.go` `r.SetAccountStatus(bctx.IsDemo, ...)`（`:109`）后：`r.SetAccountIdentity(bctx.Leverage, bctx.Currency, bctx.Company, bctx.AccountMode)`
- `vm_live_session.go` `r.SetAccountStatus(bctx.IsDemo, ...)`（`:121`）后：同上

tick/trade/timer（`vm_live_handlers.go:140/170/251`）**不动**——身份字段只在 init+OnBar 设定（现有 SetLogin/SetAccountStatus 惯例）。

## S5 — 测试

测试文件落点：runner 侧 `strategy/runner/` 现有 harness 测试惯例；服务端 `internal/connect/strategy/`（`vm_trade_context6_batch2_test.go` 同族）。

**T1 runner 端到端**：harness 模式 `SetAccountIdentity(200,"USD","BrokerCo","hedging")` 后跑编译执行断言 VM 层值——`AccountLeverage()==200`、`AccountCurrency()=="USD"`、`AccountCompany()=="BrokerCo"`、`AccountInfoInteger(ACCOUNT_MARGIN_MODE)==2`、`AccountInfoInteger(ACCOUNT_HEDGE_ALLOWED)==1`。（用现有 harness 测试的编译+执行通道，断言经 VM 执行而非 Go 直读。）

**T2 runner Mode 白名单**：`SetAccountIdentity(...,accountMode="bogus")` → `AccountInfoInteger(ACCOUNT_MARGIN_MODE)` 返回 error（fail-closed 不误判）。

**T3 服务端 injectAccountTruth**：stub `accountIdentityLookup` 返回 `{200,"USD","mt4"}` + 既有 stub 家族 → live 模式 lctx.Leverage==200/Currency=="USD"/AccountMode=="hedging"；stub MTType="mt5" → AccountMode==""（不误判）。

**T4 live fail-closed**：stub 返回 `{0,"","mt4"}` → injectAccountTruth 返回 error（leverage 0 + currency "" 都触发——可分两个子断言）；stub 返回 error → live 返回 error。paper 模式（`cfg.Mode != modeLive`）同输入不返回 error。

**T5 paper 容忍**：lookup error 时 paper 不炸且字段保持零值。

## 验收门

1. `cd backend && go build ./...`
2. 目标测试全绿（新增 T1-T5 + `go test ./strategy/runner/ ./internal/connect/strategy/ ./tools/mql2go/`）
3. race×3：上述三包 `-race -count=3`
4. `go vet ./...` / `gofmt -l` 改动文件净 / `git diff --check`
5. `go run ./tools/check-file-lines --strict`（🔴 阻断；🟡🟢 过）
6. **先红后绿对抗证明**（逐项）：
   - M1：摘除 `broker.go Account()` 的 Leverage/Currency/Company/Mode 四行 → T1 应 RED（leverage 回 0、currency/company 回 error、margin_mode 回 error）
   - M2：`sdkAccountMode` 还原为直转 `sdk.AccountMode(s)` → T2 应 RED（bogus 不再 error）
   - M3：摘除 injectAccountTruth 的 live 完备性检查块 → T4 应 RED
   - M4：`accountModeForMTType` 还原恒 `""` → T3 的 mt4 断言应 RED（"hedging"→""）
   - restore → GREEN，报告每个 mutation 的实际失败信息
7. **禁止**：碰 `mode=13` 执行模式字段、tick/trade/timer UpdateLiveState 路径、`AccountFreeMarginCheck`/`MarketInfo` 的 leverage-zero error 语义、`mt_accounts.account_method` 任何用法、ACCOUNT_MARGIN_LEVEL 计算（另债）。

## 范围边界

只改：`strategy_runtime.proto` + 生成物、`strategy_execution_handler.go`、`live_context.go`、`cmd/server/handlers_strategy.go`、`runner/context.go`、`runner/runner.go`、`runner/broker.go`、`vm_live_handlers.go`、`vm_live_dispatch.go`、`vm_live_session.go` + 测试文件。registry/STATE.md 状态更新到 🟦open（施工完成待复审）后停手，勿部署勿 push。
