# 施工派工单：TRADE-BUILTIN-ERR-SWALLOW-1

> **状态**：设计实查完成（Devin CLI，2026-09-18），待施工 → 独立复审
> **registry**：`docs/audits/tech-debt-registry.md` 行 222
> **前置**：ORDERSEND-NILBROKER-FAILCLOSED-1 ✅done（`2739f100`）——12 站 signalMode 前移+nil-broker→fatal 已落地，本单在其上改 broker 调用点语义。

---

## 1. 缺陷实证（Devin CLI 独立核实）

### 1a. 双通道混用——根因

`sdk.Broker` 写方法的 `error` 通道**混装两类语义**，`RetCode` 通道**业务拒单但 builtin 不查**：

| 站点 | 返回形态 | 当前 builtin 后果 |
|---|---|---|
| `SimBroker.OrderSend` 保证金不足（`broker.go:143`） | `{RetNoMoney}, err("insufficient margin")` | `builtinOrderSend` err→**fatal**——业务拒单被当 infra 故障杀掉整个回测（FAILCLOSED-1 矫枉过正，真实 MQL 应返 -1+_LastError=134） |
| `SimBroker.PositionClose` 票未找到（`:214`） | `{RetRejected}, err` | err→`false,nil`——碰巧对（拒单→false），但分不出 infra |
| `SimBroker.PositionCloseBy` 校验（`:231/:234/:237`） | `{RetRejected}, err` | 同上 |
| `SimBroker.PositionModify/Price/OrderDelete` 未找到（`:351/:363/:375`） | `{RetRejected}, **nil err**` | **err==nil→`BoolVal(true)` 假成功**——拒单静默变成功 |
| `brokerImpl` executor==nil（`runner/broker.go:37/58/68/81/91`） | `{RetRejected}, err("no executor")` | infra——err 正确，但 RetRejected 是契约噪声 |
| `brokerImpl` executor errors | `{RetRejected}, err` | live OMS 路径，生产 signalMode 不可达 |

### 1b. builtin 侧缺陷（12 站）

`vm_builtin_trade_signals.go` 11 站 + `vm_builtin_trade.go` `ctradeOrder:628-631`：`err != nil → BoolVal(false), nil`——infra 失败伪装拒单。附随：`builtinOrderSend:71-77` 不查 `result.RetCode`；`builtinCloseAll:243-249` 逐仓 err→allOK=false 聚合吞没；`builtinOrderModify` `pendingPriceModifier` 分支（`:86-92`）同构。

### 1c. `_LastError` 假数据洞（同族，纳入本单）

`vm.lastError` 字段存在 + `GetLastError` 读后清 + `SetUserError` 写——**但无任何 builtin 在交易失败时写入**。策略 `OrderClose()` 返 false 后 `GetLastError()`→0=假"无错误"。拒单返 false 必须配 `_LastError` 映射才构成完整的 MQL 语义。

### 1d. 引擎日志盲区（顺带修）

`engine.go:226-281 dispatchSignal` 只用 err 打 stderr——RetCode-only 拒单（`:351/:363/:375` 类）**当前在 stderr 静默消失**。改 SimBroker 后必须补 RetCode 日志，否则观测倒退。

### 1e. 依赖面（blast radius）

- SimBroker err→nil 搬迁：**零测试断言业务拒单 err!=nil**（`engine_coverage_test.go:51/:81` 已断言 RetCode 通道）；engine stderr 消费见 §1d（本单补）。
- VM 测试：无 margin/rejection→fatal 断言（fatal 测试全是 cmd=99/volume=0 参数校验类）。
- **空 RetCode mock（自审计新实证）**：`vm_builtin_trade_signals_test.go:23-35` `testBroker` 五方法返回 `{Ticket:X}, nil` RetCode=`""`——仅 signalMode 测试用（broker 永不被调），但为契约定型须补 `RetCode: sdk.RetDone`；`vm_trade_context_redo_test.go:399` `cacheTestBroker.OrderSend` 同（该测试只调 Close/CloseBy，OrderSend 是死 mock，同补卫生）。
- 语义变更明示：回测中保证金拒单由 **fatal→-1+lastError**（MQL 忠实，EA 继续跑）——此为预期行为修正。

### 1f. `_LastError` 码表实情（自审计修正）

`interp/constants.go:377-404` MQL4 错误码表**已存在且为真实 MQL4 编号**（13=INVALID_PRICE 与 129 同码并存属实）。映射必须**用表内值**，不得用外部 4107 等记忆值。表内现有：INVALID_TRADE_VOLUME=131、NOT_ENOUGH_MONEY=134、OFF_QUOTES=136、TRADE_CONTEXT_BUSY=146、INVALID_PRICE=13；**缺 ERR_TOO_MANY_ORDERS=148、ERR_TRADE_NOT_ALLOWED=4109**（真实 MQL4 码，本单补入）。

## 2. 设计裁定

**D1 契约分裂**（`sdk/broker.go` 接口注释钉死）：

```
error  = 基础设施失败（executor 缺失/传输错误/panic）——VM 应 fatal
RetCode = 业务结果（拒单/保证金不足/无效价量）——VM 应返 false/-1 + _LastError
RetCode 在 error 路径上的取值无意义（builtin 先查 err）。
```

**D2 builtin 统一模式**（12 站写函数）——**三态分裂**（自审计修正：空 RetCode 不能当拒单）：

```go
res, err := vm.ctx.Broker().Op(...)
if err != nil {
    return BoolVal(false), fmt.Errorf("<Builtin> broker error: %w", err)  // infra → fatal
}
switch res.RetCode {
case sdk.RetDone, sdk.RetDonePartial:
    // 成功路径不变（invalidateOrderCaches + true/ticket）
case "":
    return BoolVal(false), fmt.Errorf("<Builtin>: broker returned empty RetCode") // 契约违反 → fail-closed
default:
    vm.lastError = mqlErrFromRetCode(res.RetCode)
    return BoolVal(false), nil                                            // 业务拒单 → false + _LastError
}
```

- `builtinOrderSend`：`IntVal(-1)` 替 `BoolVal(false)`（broker call `:74-78` 后加 switch，`:80` ticket 在 done 分支）
- `builtinCloseAll`（`:243-265`）：`_, err :=` 改 `res, err :=`——err→fatal；RetCode `""`→fatal；其他≠done→`allOK=false`+`vm.lastError=`（不写 fatal——聚合语义保留，部分失败=整体 false）
- `builtinOrderModify` `pendingPriceModifier` 分支（`vm_builtin_trade.go` `pm.PositionModifyPrice` 调用点）：同模式
- signalMode 分支**不动**（信号不触 broker）
- `vm.lastError` 字段 `int32`（`vm.go:40`），同包直写

**D3 `mqlErrFromRetCode` 映射**（新增 helper，vm_builtin_trade.go）——**用 §1f 表内值**：

| RetCode | 码 | 常量 |
|---|---|---|
| `RetNoMoney` | 134 | ERR_NOT_ENOUGH_MONEY |
| `RetInvalidVolume` | 131 | ERR_INVALID_TRADE_VOLUME |
| `RetOffQuotes` | 136 | ERR_OFF_QUOTES |
| `RetInvalidPrice` | **13** | ERR_INVALID_PRICE（表内值，非 4107） |
| `RetTooManyOrders` | 148 | ERR_TOO_MANY_ORDERS（**本单补入 constants.go**） |
| `RetRiskBlocked` | 4109 | ERR_TRADE_NOT_ALLOWED（**本单补入 constants.go**） |
| `RetRejected` | 146 | ERR_TRADE_CONTEXT_BUSY（通用拒单最近义） |
| default/未知非空 | 146 | 同上保守映射（非成功方向保守） |

`RetDonePartial` 计成功（部分成交仍是成交）。SimBroker `RetRejected` 用例多为票未找到——146 是通用近似码，更细粒度需扩 RetCode 枚举（边界外，记 registry 备注）。

**D4 SimBroker 通道搬迁**：5 处 err 配对业务拒单 → RetCode-only nil err（`:143/:214/:231/:234/:237`）——拒单理由由 RetCode 自描述（engine 日志改读 RetCode 补偿，见 S4）。

**D5 边界（勿动）**：
- `brokerImpl` err 路径的 `RetRejected` 噪声——err→fatal 先行，RetCode 被忽略，不动（契约注释说明）。
- `executor` 接口无 RetCode 通道——executor error→err→fatal（保守响，生产 signalMode 不可达）。
- 查询 API（Positions/Orders/HistoryOrders）无 error 通道，`brokerImpl.lastError` 旁路已存——不动。
- `builtinOrderSend` signalMode 下 `ActionNone`→-1 保留。

## 3. 施工步骤

### S1：sdk/broker.go 契约注释

在 `Broker` interface 注释块写明 D1 双通道契约（error=infra，RetCode=业务，error 路径 RetCode 无意义）。

### S2：SimBroker 拒单通道搬迁（5 处）

`strategy/backtest/broker.go`：`:143/:214/:231/:234/:237` 的 `{RetCode:X}, fmt.Errorf(...)` → `{RetCode:X}, nil`。每处保留原 message 为行尾注释说明 RetCode 语义（如 `// insufficient margin → RetNoMoney`），或直接在 RetCode 值上表达（自描述则不必）。**理由字符串不丢**：`RetNoMoney`/`RetRejected` 枚举值即理由。

### S3：engine.go dispatchSignal 补 RetCode 日志

`:248-280` 各调用点改为：

```go
if res, err := e.broker.OrderSend(req); err != nil {
    fmt.Fprintf(os.Stderr, "backtest: OrderSend error at bar %d: %v\n", e.broker.currentBar, err)
} else if res.RetCode != sdk.RetDone && res.RetCode != sdk.RetDonePartial {
    fmt.Fprintf(os.Stderr, "backtest: OrderSend rejected (%s) at bar %d\n", res.RetCode, e.broker.currentBar)
}
```

PositionClose/OrderDelete/CloseAll/CancelAll 循环内同模式（log 函数名各自替换）。

### S4：VM builtin 12 站统一改造（D2 三态分裂）

`vm_builtin_trade.go`：`builtinOrderSend`（`:74-78` err→fatal 已存，其后加三态 switch，`:80` ticket 移入 done 分支）、`ctradeOrder`（err→fatal + 三态）、新增 `mqlErrFromRetCode` helper（D3 表）。
`vm_builtin_trade_signals.go`：OrderClose/CloseBy/Modify（`:92` `pendingPriceModifier` 调用点同模式——接口声明在 trade.go:304，**调用点在本文件**）/Delete + CTrade×5（每站 `_, err :=` 改 `res, err :=` + 三态）+ CloseAll（聚合版，`res, err :=` + err→fatal + `""`→fatal + 其他→allOK=false+lastError）。
`interp/constants.go`：错误码段补 `ERR_TOO_MANY_ORDERS: IntVal(148)` + `ERR_TRADE_NOT_ALLOWED: IntVal(4109)`（真实 MQL4 码，按表序插入 147 后/65536 前——4109 值大但属真实编号，置于段尾注释说明）。
**mock 卫生**：`vm_builtin_trade_signals_test.go:23-35` 五方法 + `vm_trade_context_redo_test.go:399` OrderSend 补 `RetCode: sdk.RetDone`。

每处 err→fatal 消息格式沿用 `"<Builtin> broker error: %w"`；空 RetCode 用 `"<Builtin>: broker returned empty RetCode"`。

### S5：新测试 `vm_trade_errchannel_test.go`

```go
// mock broker 可编程返回 (OrderResult, error)，表驱动：
// (a) {RetRejected}, nil  → builtinOrderDelete → BoolVal(false)+err==nil+vm.lastError==146
// (b) {RetNoMoney}, nil   → builtinOrderSend  → IntVal(-1)+err==nil+vm.lastError==134
// (c) {}, plainErr        → builtinOrderClose → err!=nil 含 "broker error"（infra fatal）
// (d) {}, nil 空RetCode   → builtinOrderClose → err!=nil 含 "empty RetCode"（契约违反 fatal）
// (e) {RetDone}, nil      → 各 builtin → true/ticket（回归保护）
// (f) builtinCloseAll + 一仓 RetRejected → false + lastError==146
// (g) builtinOrderModify pendingPriceModifier {RetRejected},nil → false + lastError
// 另：GetLastError() 读后清语义——拒单后首读=映射码，二次读=0。
// mock ctx 参考 vm_builtin_trade_signals_test.go testContext/testBroker 先例（testBroker 嵌 sdk.Broker 兜底未实现方法）。
```

### S6：mutation 证据

1. 删任一 builtin 的 RetCode switch（恢复裸 `_, err :=`+err→false）→ (a)/(b)/(f) 用例 RED（假成功复活）
2. 任一 builtin err 恢复 `false,nil` → (c) 用例 RED（infra 静默复活）
3. 删 `vm.lastError =` 行 → (a) lastError 断言 RED
4. 删 `""` 分支（default 兜底空码）→ (d) 用例 RED（空码被当拒单而非 fatal）

## 4. 验收门

```bash
cd backend && go build ./...
go test ./tools/mql2go/ ./strategy/backtest/
go test -race -count=3 ./tools/mql2go/ ./strategy/backtest/
go vet ./...
gofmt -l <改动文件>
go run ./tools/check-file-lines --strict
git diff --check
```

## 5. 文件/规模预估

| 文件 | 改动 |
|---|---|
| `strategy/sdk/broker.go` | 契约注释 ~8 行 |
| `strategy/backtest/broker.go` | 5 处 err→nil ~5 行净 |
| `strategy/backtest/engine.go` | RetCode 日志 ~15 行 |
| `tools/mql2go/vm_builtin_trade.go` | OrderSend+ctradeOrder+helper ~35 行 |
| `tools/mql2go/vm_builtin_trade_signals.go` | 10 站+CloseAll ~45 行 |
| `tools/mql2go/interp/constants.go` | 2 新错误码常量 |
| `tools/mql2go/vm_trade_errchannel_test.go` | 新 ~160 行 |
| `vm_builtin_trade_signals_test.go` / `vm_trade_context_redo_test.go` | mock 补 RetDone 共 6 处 |

`vm_builtin_trade_signals.go` 265→~305 行、`vm_builtin_trade.go` 671→~700 行——check-lines 预警级但未破阈（450 红线=🔴仅当 strict 判 error；关注输出级别，若 🔴 则按拆分预案：新 helper 放 vm_builtin_util.go）。

## 6. 回报格式

`[施工完成:TRADE-BUILTIN-ERR-SWALLOW-1] @<commit-hash>` + 机检五件套 + mutation×4 证据。勿部署、勿 push、禁 `--no-verify`。
