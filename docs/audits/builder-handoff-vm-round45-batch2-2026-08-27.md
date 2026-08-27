# Builder Handoff: VM-TRADE-CONTEXT-6（Batch 2）

> **设计/验收方**：Devin CLI
> **施工方**：Devin IDE / Windsurf
> **基线 HEAD**：`036f7683`（工作树干净，Batch 1 + Batch 4 验收后开工）
> **边界**：只施工 VM-TRADE-CONTEXT-6，禁改写历史审计事实，禁扩 scope，禁 commit/push/deploy。
> **施工后状态**：`🟦open（施工完成，待独立复审）`，不得自标 ✅done。

---

## 立项背景

D-REVERT-SCOPE-DRIFT-001 回滚了 VM-TRADE-CONTEXT-6 的全部修复。当前代码状态：

1. `vmHandleBar`（`vm_live_handlers.go:14-65`）**无 OHLCV 数组长度校验**——`lctx.Open[i]`/`High[i]`/`Low[i]`/`Close[i]`/`Volume[i]` 访问不检查数组长度一致，长度不一致会 panic。
2. `parseDecimal`（`backtest_worker_helpers.go:25-38`）**invalid decimal 静默转零**——`decimal.NewFromString` 失败返回 `decimal.Zero`，不报 error。`parseInt64` 同理（`:51`）。
3. `buildLiveContext`（`live_context.go:200-286`）**不注入 Login/Company/IsDemo/IsConnected/IsTradeAllowed**——`LiveStrategyContext` proto 有这些字段（proto codegen 已修复），但 Go 代码不填充。
4. `VMLiveSession.Start()` **无 `validateFirstBarContext`**——invalid decimal 在 Init 前不拒绝。
5. nil `Positions`/`PendingOrders`/`Symbols` **不拒绝**——nil repeated message 被当空切片处理。

---

## 🔴 绝对边界（违反 = 直接判失败）

1. **只改** `internal/connect/strategy/` 下的文件 + 新建测试文件 + 文档。**禁止改** `tools/mql2go/` 生产代码（VM-API-TRUTH-3 在 Batch 3 处理）。
2. **禁止删/改** `buildLiveContext` 的 OHLCV 填充逻辑、`backfillContextStrings` 的 PositionCache 逻辑、`backfillSymbolInfo`。
3. 禁止改 proto / DB schema / 部署 / 其他功能块。
4. 禁止 commit / push / deploy。禁 `--no-verify`。
5. 收工只显式 `git add` 本任务涉及的文件，禁止 `git add -A`。

---

## 施工步骤

- **S1** `backtest_worker_helpers.go`：新增 `parseDecimalStrict(s string) (decimal.Decimal, error)` 和 `parseInt64Strict(s string) (int64, error)`——失败返回 error 不转零。**保留** `parseDecimal`/`parseInt64` 旧函数（backtest 路径仍用，不破坏现有行为）。

- **S2** `vm_live_handlers.go` `vmHandleBar`：OHLCV 数组长度校验——`len(Close)` 与 `len(Open)`/`len(High)`/`len(Low)`/`len(Volume)`/`len(BarTimesMs)` 不一致时返回 `Success: false` + error 含 "OHLCV array length mismatch"。多 symbol 同样校验（`ss.Close`/`Open`/`High`/`Low`/`Volume` 长度一致）。

- **S3** `vm_live_handlers.go` `vmHandleBar`/`vmHandleTick`/`vmHandleTrade`：改用 `parseDecimalStrict`/`parseInt64Strict` 替代 `parseDecimal`/`parseInt64`，error 时返回 `Success: false` + error 含 "invalid decimal" / "invalid int"。**保留** `parseDecimal`/`parseInt64` 在 backtest 路径的使用（backtest 容错，live 严格）。

- **S4** `vm_live_handlers.go`：nil repeated message 拒绝——`lctx.Positions == nil` 且 `cfg.Mode == "live"` 时返回 error（nil positions 在 live mode 是数据缺失不是空仓）。`lctx.PendingOrders == nil` 同理。`lctx.Symbols` nil 不拒绝（可选字段）。

- **S5** `vm_live_session.go` `Start()`：新增 `validateFirstBarContext` 在 `Init()` 前执行——校验 first bar context 的 OHLCV 长度一致 + financial 字段（Balance/Equity/Margin/FreeMargin）非空合法 decimal。invalid 时返回 error 不执行 Init。

- **S6** `live_context.go` `buildLiveContext`：注入服务端账户真值。新增 lookup 函数（从 `mt_accounts` 表查询）：
  - `accountLoginLookup(accountID) (int64, error)` — `mt_accounts.login`
  - `accountIsDemoLookup(accountID) (bool, error)` — `mt_accounts.account_type`
  - `accountConnectedLookup(accountID) (bool, error)` — `mt_accounts.account_status == 'connected'`
  - `accountTradeAllowedLookup(accountID) (bool, error)` — `mt_accounts.account_status == 'trade_allowed'`
  - `accountIsInvestorLookup(accountID) (bool, error)` — `mt_accounts.is_investor`
  - `brokerCompanyLookup(accountID) (string, error)` — 已存在则复用
  - live mode：所有 lookup 必须成功，error 时 fail-closed 返回 error。investor 账户 IsTradeAllowed=false 即使 connected。paper mode：lookup error 非致命（fail-open for simulation）。
  - 填充 `lctx.Login`/`Company`/`IsDemo`/`IsConnected`/`IsTradeAllowed`。

- **S7** `internal/connect/strategy/strategy_execution_handler.go`（`StrategyExecutionServer` struct `:31`）+ `live_context.go`：接线 lookup 函数到 `StrategyExecutionServer`——server struct 加 lookup 字段（func 类型），`buildLiveContext` 通过 server 方法调用。DB query error 传播（不混淆真实 false）。

- **S8** file-lines 拆分：如果 `live_context.go` 或 `vm_live_handlers.go` 超 450 行，按语义拆分。拆分前先 `bash scripts/cap.sh` 查重。

---

## 测试与对抗证明

- **T1** `TestVMHandleBar_ArrayLengthMismatch`：构造 Close=5 但 Open=3 的 lctx → 返回 error 含 "OHLCV array length mismatch"。
- **T2** `TestVMHandleBar_InvalidDecimalRejected`：构造 Close=["abc"] 的 lctx → 返回 error 含 "invalid decimal"。
- **T3** `TestVMHandleBar_NilPositionRejected`：live mode + lctx.Positions=nil → 返回 error。
- **T4** `TestBuildLiveContext_InjectsLoginAndCompany`：mock lookup 返回 Login=12345, Company="Exness" → lctx.Login=12345, lctx.Company="Exness"。
- **T5** `TestBuildLiveContext_LiveModeLookupFailClosed`：mock lookup 返回 error → buildLiveContext 返回 error。
- **T6** `TestBuildLiveContext_InvestorGatingTradeAllowed`：isInvestor=true + connected=true → IsTradeAllowed=false。
- **T7** `TestVMLiveSession_StartRejectsInvalidFirstBarContext`：invalid decimal in first bar → Start 返回 error，Init 不执行（g_init=0）。
- **T8** `TestVMLiveSession_EndToEndAccountNumberReadback`：端到端——mock lookup Login=12345 → VM 执行 `AccountNumber()` → g_accountNumber=12345。

### 对抗证明

- **P1**：删 OHLCV 校验 → T1 RED（panic 或不报错）→ 恢复 → GREEN。
- **P2**：删 strict parse → T2 RED（不报 error）→ 恢复 → GREEN。
- **P3**：删 nil position 检查 → T3 RED（不报 error）→ 恢复 → GREEN。
- **P4**：删 lookup fail-closed → T5 RED（不报 error）→ 恢复 → GREEN。
- **P5**：删 validateFirstBarContext → T7 RED（Init 执行 g_init=1）→ 恢复 → GREEN。
- **P6**：删 Login 注入 → T8 RED（g_accountNumber=0）→ 恢复 → GREEN。

---

## 红队自审

1. `parseDecimalStrict` 在 live 路径使用后，是否有合法的 "0" 值被误拒？（"0" 是合法 decimal，不应被拒。）
2. nil positions 拒绝是否影响 paper mode？（paper mode 不拒绝，只 live mode 拒绝。）
3. lookup fail-closed 是否导致 DB 临时故障时所有 live 策略停止？（是——这是设计意图，fail-closed 优于用错误数据执行。）
4. investor gating 是否覆盖所有 IsTradeAllowed 消费点？（builtinIsTradeAllowed 在 Batch 3 改读 context。）
5. `validateFirstBarContext` 是否与 `vmHandleBar` 的校验重复？（不重复——validateFirstBarContext 在 Start 前一次性校验，vmHandleBar 每事件校验。）

---

## 验收门禁

```
gofmt -l <改动文件>
go build ./...
go vet ./internal/connect/strategy/...
go test ./internal/connect/strategy/... -count=1
go test -race ./internal/connect/strategy/... -count=1  # 连跑 3 次
go test ./tools/mql2go/... -count=1  # 确认无回归
go run ./tools/check-file-lines --strict
git diff --check
```

---

## 回填与收尾

registry 本条回填真实实现 + REUSE/NEW 结论 + 对抗证明输出 + 红队自审；`handover-audit-plan.md` 追加一行。**状态填 `🟦open（施工完成，待独立复审）`，不得自标 ✅done。**

> **勿部署、勿 push、停手等 Devin CLI 复审。禁止 `--no-verify`。**
