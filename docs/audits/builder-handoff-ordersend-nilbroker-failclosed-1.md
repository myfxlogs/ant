# 施工派工单：ORDERSEND-NILBROKER-FAILCLOSED-1

> **状态**：设计实查完成（Devin CLI，2026-09-18），待施工 → 独立复审
> **registry**：`docs/audits/tech-debt-registry.md` 行 215
> **范围声明**：registry 原条目只点名 `builtinOrderSend`。设计实查发现**同构缺陷遍布同族 12 个交易写函数**——按"同一文件/同一缺陷类别/同一修复形态"扩族裁定（先例：VM-GLOBAL-ARRAY-DECL-1 由 2 缺陷扩为 4 错标点），本单覆盖全部 12 站点。边界外项见 §4。

---

## 1. 缺陷实证（Devin CLI 独立核实）

### 1a. 家族全貌 — 两类缺陷叠加

| 函数 | 文件:行 | 缺陷 A（nil broker→假值，在 signalMode 之前） | 缺陷 B（broker error 吞没→false） |
|---|---|---|---|
| `builtinOrderSend` | `vm_builtin_trade.go:15` | `IntVal(-1), nil` | —（FAILCLOSED-1 已修 error 传播） |
| `ctradeOrder` | `vm_builtin_trade.go:581` | `BoolVal(false), nil` | :628-631 `err→false,nil` |
| `builtinOrderClose` | `vm_builtin_trade_signals.go:18` | `BoolVal(false), nil` | :32-35 |
| `builtinOrderCloseBy` | `:41` | 同 | :55-58 |
| `builtinOrderModify` | `:64` | 同 | :82-91（含 pendingPriceModifier 分支） |
| `builtinOrderDelete` | `:98` | 同 | :110-113 |
| `builtinCTradePositionClose` | `:119` | 同 | :132-135 |
| `builtinCTradePositionClosePartial` | `:141` | 同 | :155-158 |
| `builtinCTradePositionCloseBy` | `:164` | 同 | :178-181 |
| `builtinCTradePositionModify` | `:187` | 同 | :203-206 |
| `builtinCTradeOrderDelete` | `:212` | 同 | :224-227 |
| `builtinCloseAll` | `:233` | 同 | :243-249（逐仓 err→allOK=false 聚合吞没） |

### 1b. 缺陷 A 的双重谎言（排序缺陷）

所有 12 站点的结构均为：

```go
if vm.ctx.Broker() == nil { return <假值>, nil }   // ← 在 signalMode 检查之前
...
if vm.signalMode { vm.signal = ...; return <真值>, nil }
_, err := vm.ctx.Broker().<op>(...)
```

`signalMode` 下信号由服务端 OMS 派发（`vm.go:42-44` 注释明言 "server-side dispatch"），**本来就不调用 broker**。nil broker 在信号分支前拦截造成双重谎言：①信号被静默丢弃 ②返回 -1/false 假"拒单"。

生产可达性：live 恒 `runner.New`→`brokerImpl` 非 nil（`runner.go:49`），回测恒 `backtestContext`→`SimBroker` 非 nil（`engine_context.go:79`）——nil broker 仅见于裸 `NewVM`/自定义 ctx。**属防御纵深+排序正确性修复**，非现网活缺陷，但 fail-closed 契约要求修正。

### 1c. OrderSend 附加发现

- `builtinOrderSend` 校验顺序当前为 broker-nil→parse→validate→signalMode。signalMode 下 `cmd>5`→`orderCmdToSignalAction`→`ActionNone`→`IntVal(-1),nil`（`:52-54`）——**又一处静默 -1**。validate 前置可同批消灭。
- `result.RetCode` 未检查（`:71-77`）——`brokerImpl` 所有拒单均 RetRejected+err 配对（`broker.go:37-94`），故对 brokerImpl 无害；其他 sdk.Broker 实现若 RetCode-only 拒单会返假票号。**记边界**（§4 新债），本单不动。

### 1d. 依赖面（blast radius）= 2 处断言

- `vm_context_test.go:51-55`：`builtinOrderSend(vm, nil)` 断言 `Int==-1 && err==nil`（注释自标 "the fail-closed gap is separately ticketed"——占位断言，本单推翻）
- `vm_context_test.go:100-102`：`builtinOrderClose(vm, nil)` 断言 `Bool==false`
- `vm_context_test.go:103-105`：`builtinOrdersTotal`→0 —— **不动**（查询类，§4 边界）
- 其余调 trade builtin 的测试 ctx 均带非 nil broker（`testContext{testBroker}`、`cacheTestContext`、`accountTestContext`），e2e 走 runner/backtest 非 nil broker——零波及。
- `argS/argI/argD` 对 nil args 边界安全（`vm_builtin_util.go:16-37` 返回零值）——`builtinOrderSend(vm, nil)` 在新顺序下 parse 得 cmd=0/volume=0→"invalid volume" error，仍为 error 语义。

## 2. 设计裁定

**D1 修复形态（12 站点统一）**：`signalMode` 分支前移 → nil-broker 检查只守卫真实 broker 调用 → nil broker → `fmt.Errorf`（fatal，同 FAILCLOSED-1 broker error 先例）。信号路径对 broker 存在性完全免疫。

**D2 OrderSend 指令顺序**（新）：parse args → validate(cmd∈[0,5], volume>0) → `if signalMode` emit → `if Broker()==nil` error → broker.OrderSend → propagate。validate 在 signalMode 前——顺带消灭 `cmd>5`→ActionNone→静默 -1 子路径。

**D3 错误语义**：nil broker = 环境配置缺失（非拒单），`fmt.Errorf("<Builtin>: no broker in the VM")` → callBuiltin 置 fatalError → VM 停。返回值约定保留签名形态（OrderSend `IntVal(-1), err`；bool 族 `BoolVal(false), err`）——err 非 nil 时值不可达。

**D4 边界（本单不做，另立新债）**：
- **缺陷 B**（11 站 broker error→false 吞没 + CloseAll allOK 聚合 + OrderSend RetCode 未查）→ 新债 `TRADE-BUILTIN-ERR-SWALLOW-1`：infra 失败（"no executor configured"）伪装成"拒单"，且真 MQL 拒单语义（false+GetLastError）vs fatal 是**独立设计决策**，需 RetCode/error 语义审计后定。
- **查询 API**（OrdersTotal/OrdersHistoryTotal/PositionsTotal→0、OrderSelect/PositionGetTicket/PositionSelectByTicket→false）：读路径空态语义不同类，本单不动。

## 3. 施工步骤

### S1：vm_builtin_trade.go — builtinOrderSend 重排

新顺序（对照 :14-78）：

```go
func builtinOrderSend(vm *VM, args []interp.Value) (interp.Value, error) {
    // parse（原 :19-27 不变）
    symbol := argS(args, 0); cmd := argI(args, 1); volume := argD(args, 2)
    price := argD(args, 3); deviation := argI(args, 4); sl := argD(args, 5)
    tp := argD(args, 6); comment := argS(args, 7); magic := argI(args, 8)

    // validate（原 :31-36 不变，含 VM-RUNTIME-FAILCLOSED-1 注释保留）
    if cmd < 0 || cmd > 5 { return IntVal(-1), fmt.Errorf("OrderSend: invalid cmd %d (must be 0-5)", cmd) }
    if volume.LessThanOrEqual(decimal.Zero) { return ..., fmt.Errorf("OrderSend: invalid volume %s (must be > 0)", volume.String()) }

    req := sdk.OrderRequest{...}; mapOrderCmd(cmd, &req)   // 原 :38-48 不变

    if vm.signalMode {                                    // 原 :50-69 整段不变
        action := orderCmdToSignalAction(cmd)
        if action == sdk.ActionNone { return interp.IntVal(-1), nil }
        vm.signal = &sdk.Signal{...}
        vm.invalidateOrderCaches()
        return interp.IntVal(1), nil
    }

    // ORDERSEND-NILBROKER-FAILCLOSED-1：无 broker = 环境缺失，fail-closed
    // （原在函数顶部、先于 signalMode——会拦信号路径造成信号静默丢弃）
    if vm.ctx.Broker() == nil {
        return interp.IntVal(-1), fmt.Errorf("OrderSend: no broker in the VM")
    }
    result, err := vm.ctx.Broker().OrderSend(req)         // 原 :71-77 不变
    if err != nil { return interp.IntVal(-1), fmt.Errorf("OrderSend broker error: %w", err) }
    vm.invalidateOrderCaches()
    return interp.IntVal(int32(result.Ticket)), nil
}
```

注意：`signalMode` 下 `action==ActionNone` 的 `IntVal(-1),nil` 分支**保留不动**（cmd 已经 validate∈[0,5]，该分支理论不可达；改动它超范围）。

### S2：vm_builtin_trade.go — ctradeOrder 重排

原 :581-583 的 nil-broker 检查**移到 signalMode 分支之后**（:626 后、:628 broker 调用前）：

```go
func ctradeOrder(...) (interp.Value, error) {
    // parse（原 :584-606 不变：volume/symbol/price/sl/tp/comment + req 构造）
    ...
    if vm.signalMode {              // 原 :608-626 整段不变
        ...emit signal; return BoolVal(true), nil
    }
    if vm.ctx.Broker() == nil {
        return interp.BoolVal(false), fmt.Errorf("CTrade order: no broker in the VM")
    }
    _, err := vm.ctx.Broker().OrderSend(req)   // 原 :628-633 不变（err→false 吞没属缺陷 B，另债）
    ...
}
```

### S3：vm_builtin_trade_signals.go — 10 站点同构重排

对下列 10 个函数执行同一变换：**将 `if vm.ctx.Broker()==nil → false` 块从函数顶移到 `if vm.signalMode {...}` 块之后、broker 调用之前**，返回值改 `fmt.Errorf`：

| 函数 | 错误消息 |
|---|---|
| `builtinOrderClose` | `"OrderClose: no broker in the VM"` |
| `builtinOrderCloseBy` | `"OrderCloseBy: no broker in the VM"` |
| `builtinOrderModify` | `"OrderModify: no broker in the VM"` |
| `builtinOrderDelete` | `"OrderDelete: no broker in the VM"` |
| `builtinCTradePositionClose` | `"CTrade.PositionClose: no broker in the VM"` |
| `builtinCTradePositionClosePartial` | `"CTrade.PositionClosePartial: no broker in the VM"` |
| `builtinCTradePositionCloseBy` | `"CTrade.PositionCloseBy: no broker in the VM"` |
| `builtinCTradePositionModify` | `"CTrade.PositionModify: no broker in the VM"` |
| `builtinCTradeOrderDelete` | `"CTrade.OrderDelete: no broker in the VM"` |
| `builtinCloseAll` | `"CloseAll: no broker in the VM"` |

各函数其余行**逐字保留**（ticket/volume parse 在原位置——parse 不依赖 broker，移到 signalMode 前后的相对顺序保持原样；signalMode 块、broker 调用、err→false、invalidateOrderCaches 全不动）。

示例形态（builtinOrderClose :17-38）：

```go
func builtinOrderClose(vm *VM, args []interp.Value) (interp.Value, error) {
    ticket := int64(argI(args, 0))
    volume := argD(args, 1)
    if vm.signalMode {                    // 原 :23-31 不变
        vm.signal = &sdk.Signal{...}
        vm.invalidateOrderCaches()
        return interp.BoolVal(true), nil
    }
    // ORDERSEND-NILBROKER-FAILCLOSED-1: no broker is an environment defect,
    // not a rejection — fail closed (was BoolVal(false), nil before the
    // signalMode check, which also dropped live signals).
    if vm.ctx.Broker() == nil {
        return interp.BoolVal(false), fmt.Errorf("OrderClose: no broker in the VM")
    }
    _, err := vm.ctx.Broker().PositionClose(ticket, volume)
    if err != nil { return interp.BoolVal(false), nil }   // 缺陷 B 保留（另债）
    vm.invalidateOrderCaches()
    return interp.BoolVal(true), nil
}
```

每函数加一行 `ORDERSEND-NILBROKER-FAILCLOSED-1` 注释（一行即可，勿复制整段）。

### S4：vm_context_test.go 断言翻转（2 处）

- `:51-55`：改断言 `builtinOrderSend(vm, nil)` → `err != nil` 且含 `"no broker"`（注意 nil args 会先撞 volume 校验——若实现按 S1 顺序，err 实为 "invalid volume"；**断言应写 `err != nil`，不断言具体消息**，或显式传 `volume>0` 的 args 后断言 "no broker"——二选一，推荐后者精确定位）。
- `:100-102`：`builtinOrderClose(vm, nil)` → `err != nil` 含 `"no broker"`。
- 删除/更新 :51-52 的占位注释（gap 已 ticketed 并已修）。
- `:93` 注释 "Broker()==nil checks still fire" 语义已变——更新为 fail-closed 语义描述。
- `:103-105` OrdersTotal→0 **不动**。

### S5：新测试 `vm_nilbroker_failclosed_test.go`

```go
// 12 站点 × 两场景，表驱动：
// (a) 非 signal + nil broker → err != nil && strings.Contains(err, "no broker")
// (b) signalMode + nil broker → err == nil && vm.signal != nil && 返回值成功
//     （信号 emitted = 排序修复的直接证据；变异掉重排即 RED）
//
// 用 NewVM(&Bytecode{Builtins: make(map[string]BuiltinID)})（noopContext → Broker()=nil，
// 同 vm_context_test.go 先例）+ vm.signalMode=true 直调各 builtin，传合法 args
// （OrderSend: cmd=0 volume=1；Close/Delete: ticket=1 volume=1；Modify: ticket=1 price/sl/tp；
// CloseBy: t1,t2；CloseAll: 无参）。
// 断言 signal.Action 正确（ActionBuy/ActionClose/ActionModify/ActionCancel/ActionCloseAll）。
// 注：ctradeOrder 为内部 helper，经 builtinCTradeBuy(vm, args)（volume/symbol/price/sl/tp/comment）
// 或 builtinCTradeSell 等包装入口触达。
```

### S6：mutation 证据（施工方先跑，复审独立重跑）

1. `builtinOrderSend` 恢复 broker-nil-first → (b) 用例 RED（signal nil+返回 -1）
2. 任一 sibling（如 OrderClose）恢复 nil-broker→`false,nil` → (a) 用例 RED（err==nil）
3. 任一 sibling 删 signalMode 前移 → (b) 用例 RED（信号丢）

## 4. 验收门

```bash
cd backend && go build ./...
go test ./tools/mql2go/
go test -race -count=3 ./tools/mql2go/
go vet ./...
gofmt -l <改动文件>
go run ./tools/check-file-lines --strict
git diff --check
```

## 5. 边界（勿动）

- 缺陷 B（broker error→false 吞没 ×11 + RetCode 未查 + CloseAll allOK）→ `TRADE-BUILTIN-ERR-SWALLOW-1` 另债，**本单不修**（修了就混语义决策）。
- 查询 API nil-broker→0/false（OrdersTotal/OrdersHistoryTotal/PositionsTotal/OrderSelect/PositionGetTicket/PositionSelectByTicket）——读路径不同类，不动。
- `signalMode` 下 `action==ActionNone`→-1 分支保留。
- `invalidateOrderCaches` 调用点不动。
- `vm_builtin_trade_signals.go` 当前 253 行，S3 净增 ~10 行——远低于 450 红线。

## 6. 回报格式

`[施工完成:ORDERSEND-NILBROKER-FAILCLOSED-1] @<commit-hash>` + 机检五件套 + mutation×3 证据。勿部署、勿 push、禁 `--no-verify`。
