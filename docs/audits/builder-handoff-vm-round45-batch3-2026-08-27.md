# Builder Handoff: VM-API-TRUTH-3（Batch 3）

> **设计/验收方**：Devin CLI
> **施工方**：Devin IDE / Windsurf
> **基线 HEAD**：Batch 2 验收后开工
> **边界**：只施工 VM-API-TRUTH-3，禁改写历史审计事实，禁扩 scope，禁 commit/push/deploy。
> **施工后状态**：`🟦open（施工完成，待独立复审）`，不得自标 ✅done。

---

## 立项背景

D-REVERT-SCOPE-DRIFT-001 回滚了 VM-API-TRUTH-3 的全部修复。当前代码状态：

`vm_builtin_checkup.go:12-34` 三个 builtin 全部硬编码：
- `builtinIsConnected` → `return BoolVal(true), nil`（恒 true）
- `builtinIsDemo` → `return BoolVal(true), nil`（恒 true）
- `builtinIsTradeAllowed` → `return BoolVal(true), nil`（恒 true）

MQL 策略调用 `IsConnected()`/`IsDemo()`/`IsTradeAllowed()` 时永远拿到 true，与实际账户状态无关。live mode 下策略可能基于错误的连接/权限状态执行交易。

Batch 2 已注入 `LiveStrategyContext.Login/Company/IsDemo/IsConnected/IsTradeAllowed`，本批让 VM builtins 从 context 读取这些真值。

---

## 🔴 绝对边界

1. **只改** `tools/mql2go/vm_builtin_checkup.go` + `tools/mql2go/vm.go`（如需暴露 context）+ 新建测试文件 + 文档。**禁止改** `internal/connect/strategy/`（Batch 2 已处理 context 注入）。
2. **禁止改** 其他 builtin（`IsDllsAllowed`/`IsExpertEnabled`/`IsLibrariesAllowed`/`IsTradeContextBusy`/`IsStopped` 等）——这些在 backtest context 下固定值是正确的。
3. 禁止改 proto / DB schema / 部署。
4. 禁止 commit / push / deploy。禁 `--no-verify`。

---

## 施工步骤

- **S1** `vm_builtin_checkup.go` `builtinIsConnected`：改为从 `vm.ctx` 读取——如果 `vm.ctx` 有 `IsConnected()` 方法（或等价字段），返回该值；`vm.ctx == nil`（backtest 无 live context）时保留 `return BoolVal(true), nil`（backtest 默认 connected）。

- **S2** `vm_builtin_checkup.go` `builtinIsDemo`：改为从 `vm.ctx` 读取 `IsDemo()`；`vm.ctx == nil` 时保留 `return BoolVal(true), nil`（backtest 默认 demo）。

- **S3** `vm_builtin_checkup.go` `builtinIsTradeAllowed`：改为从 `vm.ctx` 读取 `IsTradeAllowed()`；`vm.ctx == nil` 时保留 `return BoolVal(true), nil`。investor 账户由 Batch 2 在 context 注入时设 `IsTradeAllowed=false`，本批只读不判断。

- **S4** `vm.go` 或 `vm_context.go`：确保 `vm.ctx` 暴露 `IsConnected()`/`IsDemo()`/`IsTradeAllowed()` 方法。如果 `BarContext`/`LiveStrategyContext` 没有 these methods，新增（从 proto 字段读取）。如果 `vm.ctx` 是接口，扩展接口。

- **S5** `vm_live_handlers.go`（如需）：确保 `vmHandleBar`/`vmHandleTick` 把 `lctx.IsConnected`/`IsDemo`/`IsTradeAllowed` 传入 VM context。如果 `UpdateLiveState` 已处理则跳过。

---

## 测试与对抗证明

- **T1** `TestBuiltinIsConnected_ReadsFromContext`：构造 VM + ctx.IsConnected=false → `builtinIsConnected` 返回 false。
- **T2** `TestBuiltinIsDemo_ReadsFromContext`：构造 VM + ctx.IsDemo=false → `builtinIsDemo` 返回 false。
- **T3** `TestBuiltinIsTradeAllowed_ReadsFromContext`：构造 VM + ctx.IsTradeAllowed=false → `builtinIsTradeAllowed` 返回 false。
- **T4** `TestBuiltinIsConnected_NilContextDefaultsTrue`：VM + ctx=nil → 返回 true（backtest 默认）。
- **T5** `TestVMLiveSession_IsConnectedEndToEnd`：端到端——mock context IsConnected=false → VM 执行 `IsConnected()` → g_isConnected=0。
- **T6** `TestVMLiveSession_IsTradeAllowedEndToEnd`：端到端——investor 账户 IsTradeAllowed=false → VM 执行 `IsTradeAllowed()` → g_isTradeAllowed=0。

### 对抗证明

- **P1**：revert `builtinIsConnected` 为 `return BoolVal(true)` → T1 RED（返回 true 而非 false）→ 恢复 → GREEN。
- **P2**：revert `builtinIsTradeAllowed` 为 `return BoolVal(true)` → T3 RED → 恢复 → GREEN。
- **P3**：revert end-to-end context 传递 → T5 RED（g_isConnected=1）→ 恢复 → GREEN。

---

## 红队自审

1. `vm.ctx == nil` 时返回 true 是否安全？（backtest 无 live context，默认 connected/demo/trade-allowed 是正确行为。）
2. `IsTradeAllowed` 在 VM 内是否被用于控制交易决策？引用策略代码示例。
3. context 传递路径是否完整（`buildLiveContext` → `vmHandleBar` → `UpdateLiveState` → `vm.ctx`）？
4. 其他 checkup builtins（`IsDllsAllowed` 等）为什么不改？（backtest context 下固定值正确，无 live 等价。）
5. 本批是否与 Batch 2 有耦合？（Batch 2 注入 context 字段，本批读取——顺序依赖已满足。）

---

## 验收门禁

```
gofmt -l <改动文件>
go build ./...
go vet ./tools/mql2go/...
go test ./tools/mql2go/... -count=1
go test -race ./tools/mql2go/... -count=1  # 连跑 3 次
go test ./internal/connect/strategy/... -count=1  # 确认无回归
go run ./tools/check-file-lines --strict
git diff --check
```

---

## 回填与收尾

registry 本条回填 + `handover-audit-plan.md` 追加一行。**状态填 `🟦open（施工完成，待独立复审）`。**

> **勿部署、勿 push、停手等 Devin CLI 复审。禁止 `--no-verify`。**
