# Builder Handoff — MT5-ACCMETHOD-ADAPTER-1

> mtapi MT5 `AccountSummary.Method`(AccMethod) 存在但 adapter 未映射 → MT5 账户 margin mode 无权威源。
> 设计 SSOT：Devin CLI。施工只执行、不决策。完成报证据等独立复审。
> **依赖**：本债建立在 LIVE-ACCOUNT-FIELDS-1 已验收基础上（`AccountIdentity` 结构、`lctx.account_mode` 字段、`accountIdentityLookup` 均已存在）——前债未验收前勿开工。

## 0. 立项证据链（设计实查已核实）

**缺口**：`reference/grpc/mt5.proto` `AccountSummary.Method`（f10，`AccMethod` enum：`AccMethod_Default=0`/`AccMethod_Netting=1`/`AccMethod_Hedging=2`，生成枚举 `mt5/mt5.pb.go:83-95` `pb.AccMethod_AccMethod_*`）是 MT5 账户 hedging/netting **唯一权威源**，但 adapter 面丢弃——`mdtick.MTAccountInfo`（`mdtick.go:85-98`）、`BrokerInfo`（`:41-61`）、`ProfitUpdate`（`:101-117`）三结构全无该字段；`mt5/connection_account.go:44-56` 只映射 9 字段、`mt5/connection.go:345-358` BrokerInfo 同缺。**MT4 proto AccountSummary 无 Method 字段**（MT4 恒 hedging 平台语义——无需也不可映射）。

**语义陷阱已排除**：`mt_accounts.account_method` 存 `"master"|"investor"`（`mdgateway/runner_gateway.go:175-178` investor 标志写入），**非** margin mode——禁止复用该列。

**写入通道实证**：`mdtick.MTAccountInfo` 唯一消费链 = `verifyMTCredentials`（`account_crud.go:41-47` 绑定时 mtapi 验证）→ `service.AccountInfoUpdate` → `UpdateAccountInfoTx`（`account_lifecycle.go:57-69`，绑定事务内写 balance/equity/credit/margin/free_margin/leverage/currency/is_investor/account_type）。leverage/currency 同路——margin_mode 跟随同一通道即全栈一致。connect 路径 `updateAccountStatusOnConnect`（`runner_gateway.go:150+`）只写 status/is_investor/account_method，**不写** summary 字段——margin_mode 不入该路（绑定即足，broker 侧静态配置）。

**前债接点**（LIVE-ACCOUNT-FIELDS-1 落地后存在）：`AccountIdentity{Leverage,Currency,MTType}` 结构 + `accountIdentityLookup`（`SELECT COALESCE(leverage,0),COALESCE(currency,''),mt_type`）+ `lctx.account_mode` + `accountModeForMTType`（mt4→hedging 兜底）。本债把 account_mode 从"平台推导"升级为"DB 权威读"。

**REUSE/NEW**：REUSE——`MTAccountInfo`/`AccountInfoUpdate`/`UpdateAccountInfoTx`/`accountIdentityLookup` 既有通道全复用（margin_mode 与 leverage/currency 同生命周期：绑定写入、DB 镜像、lookup 读）；NEW——无新函数（`accMethodToString` 映射 helper 唯一新函数，mt5 包内私有）。

## S1 — adapter 映射（2 文件）

**S1a** `internal/mdgateway/adapter/mdtick/mdtick.go` `MTAccountInfo`（`:97` `AccountType` 后）：

```go
// MarginMode is the broker account margin mode: "hedging"|"netting"|"".
// Sourced from mtapi MT5 AccountSummary.Method (AccMethod); "" = unknown.
// MT4 has no Method field — adapter fills "hedging" (platform semantics:
// MT4 is hedging-only). MT5-ACCMETHOD-ADAPTER-1.
MarginMode string
```

**S1b** `mt5/connection_account.go`（`:56` `AccountType` 行后）：

```go
MarginMode:  accMethodToString(s.GetMethod()),
```

helper（同文件尾）：

```go
// accMethodToString maps mtapi AccMethod to margin mode string.
// Default/unknown → "" (fail-closed — do not guess).
func accMethodToString(m pb.AccMethod) string {
    switch m {
    case pb.AccMethod_AccMethod_Netting:
        return "netting"
    case pb.AccMethod_AccMethod_Hedging:
        return "hedging"
    }
    return ""
}
```

**S1c** `mt4/connection_account.go`（`:55` 区域）：`MarginMode: "hedging",`（注释：`// MT4 is hedging-only — platform semantics, not a lookup`）

**不动** `BrokerInfo`/`ProfitUpdate`——snapshot 链路无需 margin_mode（DB 路径已够，静态配置）。

## S2 — DB 列 + 写入通道

**S2a** `backend/migrations/280_mt_accounts_margin_mode.up.sql`：

```sql
-- 280_mt_accounts_margin_mode.up.sql
-- MT5-ACCMETHOD-ADAPTER-1: broker account margin mode (hedging/netting)
-- from mtapi AccountSummary.Method, written at account bind/verify.
-- NULL = unknown (MT5 Default / pre-migration rows); distinct from
-- account_method which stores the investor flag ("master"/"investor").
ALTER TABLE mt_accounts ADD COLUMN IF NOT EXISTS margin_mode VARCHAR(16);
COMMENT ON COLUMN mt_accounts.margin_mode IS 'broker margin mode: hedging|netting|NULL(unknown); from mtapi AccMethod — NOT the investor-flag account_method';
```

`280_mt_accounts_margin_mode.down.sql`：`ALTER TABLE mt_accounts DROP COLUMN IF EXISTS margin_mode;`

**S2b** `model/mt_account.go`：`MarginMode *string`（`db:"margin_mode"`，NULL 语义保留——"" 与 NULL 区分"写过空值"vs"从未同步"；若文件惯例用 string+COALESCE 也可，以 NULL-able 为准）

**S2c** `service/account_lifecycle.go`：`AccountInfoUpdate` 加 `MarginMode string`；`UpdateAccountInfoTx`/`UpdateAccountInfo` 两 UPDATE 各加 `margin_mode = $N`（参数序位顺延）——`""`→`NULL` 转换：`NULLIF($N,'')`（保持"未同步=NULL"语义）。

**S2d** `account_crud.go:82`：`MarginMode: info.MarginMode` 入 AccountInfoUpdate。

**S2e** `sqlc generate`（沿用 DATA-TRUTH-3 装的 v1.31.1）——`SELECT *` 展开自动带列；`sqlc_models.go` 得 `MarginMode pgtype.Text`。**波及面警告**：`accounts.sql.go` 的 SELECT * 展开会多出 margin_mode 列——属预期；其他 drift 阻断上报。`admin_repo_accounts.go` 显式 SELECT 列表**不加** margin_mode（admin 面无需该列，范围纪律）。

## S3 — strategy 侧升级 account_mode 为 DB 权威读

**S3a** `strategy_execution_handler.go` `AccountIdentity` 加 `MarginMode string` 字段。

**S3b** `cmd/server/handlers_strategy.go` lookup SQL：`SELECT COALESCE(leverage,0),COALESCE(currency,''),COALESCE(mt_type,''),COALESCE(margin_mode,'')` + scan 第四列。

**S3c** `live_context.go`：`lctx.AccountMode` 改直读——

```go
mode := ident.MarginMode
if mode == "" {
    mode = accountModeForMTType(ident.MTType) // legacy rows: MT4 NULL → hedging
}
lctx.AccountMode = mode
```

`accountModeForMTType` 保留为 MT4 legacy 兜底（绑定早于本列的存量行）；doc 注释更新为"MT4 platform-semantics fallback for pre-margin_mode rows"。

**S3d** live fail-closed 边界**不变**：`margin_mode==""` 对 MT5 仍是"诚实未知"→VM fail-closed（不加 live 完备性要求——MT5 Default(0) 是合法 broker 响应，不是缺失）。

## S4 — 测试

**T1 adapter 映射**（mt5 包测试）：`accMethodToString(Netting)=="netting"`/`(Hedging)=="hedging"`/`(Default)==""`/未知枚举值 `AccMethod(99)`→`""`。

**T2 写入链**（service 层测试若存在先例）：`UpdateAccountInfoTx` margin_mode 写入+`""`→NULL 转换（无 DB 测试环境则如实报告跳过，以代码审查+SQL 语法验证代替）。

**T3 服务端**（`internal/connect/strategy/` 同族 stub 惯例）：stub `accountIdentityLookup` 返回 `{200,"USD","mt5","netting"}` → `lctx.AccountMode=="netting"`；`{"mt5",""}`→`""`；`{"mt4",""}`→`"hedging"`（legacy 兜底）；`{"mt4","hedging"}`→`"hedging"`（DB 值优先）。

**T4 VM 端到端**（runner harness 测试）：`SetAccountIdentity(...,accountMode="netting")` → `AccountInfoInteger(ACCOUNT_MARGIN_MODE)==0`、`ACCOUNT_HEDGE_ALLOWED==0`——netting 值走通 VM 层（前债 T1 已覆 hedging=2/1，本债补 netting=0/0 对侧）。

## 验收门

1. `cd backend && go build ./...`
2. 目标测试全绿 + `go test ./internal/connect/strategy/ ./internal/mdgateway/adapter/mt5/ ./internal/mdgateway/adapter/mt4/ ./internal/service/ ./strategy/runner/`
3. 上述包 `-race -count=3`
4. `go vet ./...` / `gofmt -l` 改动文件净 / `git diff --check` / `check-file-lines --strict`
5. **先红后绿**：
   - M1：`accMethodToString` 恒 `""` → T1 应 RED + T3 netting 断言应 RED
   - M2：S3c 摘除 `ident.MarginMode` 直读（恢复仅 mt_type 推导）→ T3 netting/mt5 断言应 RED（mt4 兜底断言不受影响仍过——边界独立）
   - M3：mt4 adapter `MarginMode:"hedging"` 改 `""` → 若 T1b/adapter 测试覆盖则 RED（若无直测，经 T3 链路间接——如实报告覆盖路径）
   - restore→GREEN
6. **禁止**：碰 `account_method`（investor 标志列）任何用法、`BrokerInfo`/`ProfitUpdate`/PositionSnapshot、proto `mode=13`、wiring.go 视图查询、`ACCOUNT_MARGIN_LEVEL` 计算（另债）；不跑 DB 写、不部署。

## 范围边界

只改：`mdtick.go`、`mt5/connection_account.go`、`mt4/connection_account.go`、`migrations/280_*` 新文件对、`model/mt_account.go`、`service/account_lifecycle.go`、`account_crud.go`、sqlc 生成物、`strategy_execution_handler.go`、`live_context.go`、`cmd/server/handlers_strategy.go` + 测试文件。registry/STATE 更新到 🟦open（施工完成待复审）后停手，勿部署勿 push。
