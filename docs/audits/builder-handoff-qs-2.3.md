# builder-handoff-qs-2.3 — `vm.ctx` 非 nil 不变量（noopContext 注入，P2 洁净）

> v1 @2026-09-16（随 QS-2.5 验收后发单，VM 质量方案 v2 阶段 2 收尾单）

## 立项背景

spec §4.4，registry QS-2.3。`vm.ctx == nil` 散布检查实拍 **101 处** + 反向 `vm.ctx != nil` **15 处**（合计 116 处，15+ 个 builtin 文件）。VM 构造只经 `NewVM`（`vm.go:68`，全仓唯一 `&VM{}` 字面量）——把 noop 注入 ctor 即可建立"ctx 永非 nil"不变量，116 处守卫全部可机械消除。

## 设计 SSOT

- spec §4.4 + 本文件坐标（D-013 已核实）。
- **不变量**：`NewVM` 返回的 VM `ctx` 永非 nil；`SetContext(nil)` 归一化为 noop（防御性，不许外部把 nil 打回来）。
- **语义等价第一**：每处守卫删除必须产出与原 nil 分支**完全相同**的可观测行为；任何 noop 路径结果 ≠ 原 nil 分支返回值的站点 → `[转交决策]`，禁止静默改语义。

## 约束与目标

1. 只改 `tools/mql2go/`（含新增 `vm_context.go` 类文件）+ 测试；不动 `strategy/sdk` 接口定义、不动 runner/backtest 两个真实 Context 实现。
2. **保留** `vm.ctx.Broker() == nil` 检查（spec 原文）——noop.Broker() 返回 nil，这些检查照常生效。
3. 串行，勿部署、勿 push、禁 `--no-verify`；commit 用 `ANT_ROLE=builder git commit` 前缀。
4. 完工按 D-012 自审 + D-014 末行 `[施工完成:QS-2.3] @<commit-hash>` 自报，停手等复审。

## S1 — 站点清单（先产清单再动手）

```bash
cd backend
grep -rn "vm.ctx == nil" tools/mql2go/ --include="*.go" | grep -v "_test.go"   # 预期 101
grep -rn "vm.ctx != nil" tools/mql2go/ --include="*.go" | grep -v "_test.go"   # 预期 15
```

清单落进 commit message 或自报附件（每文件计数）。若与 101/15 不符 → `[转交决策]`。

## S2 — `noopContext` 实现（新文件 `tools/mql2go/vm_context.go`）

非导出 `noopContext` 实现 `sdk.Context`（`strategy/sdk/context.go:12-105`，~30 方法），返回值逐一对齐"原 nil 分支零值"：

| 方法族 | noop 返回 | 等价依据 |
|---|---|---|
| `Bid/Ask/Point/Pip/Spread` | `decimal.Zero` | nil 分支返回 `DecimalVal(decimal.Zero)`（impls.go:356-373 实拍） |
| `Digits` | `0` | `IntVal(0)`（:377-380） |
| `Symbol/Timeframe` | `""` | `StringVal("")`（:384-387） |
| `Account()` | `AccountInfo{}` 零结构体 | 值类型，`Account().Balance`→0 与 nil 分支零值同（account.go:17+） |
| `Mode()` | `""`（AccountMode 零值） | 待逐站核对（见 S4 等价表） |
| `Broker()` | `nil` | 保留 `Broker()==nil` 检查路径（spec 原样） |
| `Bars()/BarsTF()/BarsForSymbol()` | `sdk.BarsToSlice(nil)`（**非 nil**） | 既有零 bar 实现（series.go:58-90，`at()` 越界返回 `Bar{}`，Len()=0）——返回 nil interface 会让 `.Close()` 调用 panic |
| `Indicators()` | 新 `noopIndicatorSet`（非 nil，全方法 `decimal.Zero`） | `strategy/sdk/indicators.go` 全部 ~34 签名逐一实现（含 Bollinger/Stochastic 等多返回值） |
| `Param*(name, default)` | `default` 原样返回 | nil 分支亦返回默认值语义（逐站核对） |
| `SetTimer/KillTimer/Log` | no-op | `builtinPrint` 的 `vm.ctx != nil` 守护删除后 Log 空转（impls.go:317） |
| `ServerTime()` | `0` | 零值 |
| `GoContext()` | `context.Background()` | 接口文档原语义（context.go:101-104） |

## S3 — 不变量接线

- `NewVM`（`vm.go:68-79`）：`ctx: noopContext{}` 入字面量。
- `SetContext`（`vm.go:83-85`）：`if ctx == nil { ctx = noopContext{} }` 归一化。
- 调用方 `interp_runner.go:287-388` 八处 `SetContext(ctx)` 不改——传入非 nil 照常工作，传 nil 被归一化。

## S4 — 116 处守卫消除（逐文件，等价表随 commit）

消除规则：
- `if vm.ctx == nil { return zero }` → 删守卫直走主路径（noop 零值与原返回值一致）。
- `vm.ctx == nil || vm.ctx.Broker() == nil` → 简化为 `vm.ctx.Broker() == nil`。
- `vm.ctx != nil { X } else { zero }` → 去 else 直走 X（noop 下 X 的结果须 = 原 else 值）。
- `vm.ctx != nil && vm.ctx.Broker() != nil` → `vm.ctx.Broker() != nil`（trade.go:167/191/194 等缓存填充守卫照旧有效）。

**等价表（必交）**：对每处站点给一行"文件:行 → 原 nil/else 分支值 → noop 路径值 → 相同?"。发现不等价的站点**不许改**，列入 `[转交决策]` 清单连证据上报（重点怀疑对象：nil 分支返回 -1/errcode 而 noop 路径返回 0 的站点、依赖 `Len()==0` 短路语义的序列 builtin、`GoContext` 的取消语义差异）。

## S5 — 测试（`vm_context_test.go` 新文件或就近）

- **S5a 不变量**：`NewVM(bc)` 后 `vm.ctx != nil`；`SetContext(nil)` 后仍 `vm.ctx != nil`。
- **S5b 等价锚点**（无 SetContext 的 VM 直接调 builtin，与"改前 nil-ctx"期望值对比——先红后绿不便时以行为断言替代并注明）：`builtinBid/Ask/Point`→0、`builtinDigits`→0、`builtinSymbol`→""、`builtinAccountBalance`→0、`builtinPrint` 不 panic、`builtinOrderSend` 无 broker→`-1`（既有行为保留，见边界）、指标 builtin（如 iMA/iRSI 经 `Indicators()`）→0 不 panic、`Close(0)` 经 `Bars()`→0 不 panic。
- **S5c Param 语义**：`ParamInt("x", 42)`→42（default 透传）。
- **S5d `vm.ctx == nil` 计数归零**：`grep -c "vm.ctx == nil" tools/mql2go/`（非测试文件）= 0 进自报证据；`!= nil` 残留只允许在"仍有意义的复合条件"中且须逐一说明（目标同样归零或仅剩合理项）。

## 验收标准

- [ ] `go build ./...`、`go test -count=1 ./tools/mql2go/...` 全绿
- [ ] `go test -race -count=3 ./tools/mql2go/...` 通过（goleak 门禁仍绿）
- [ ] `check-file-lines --strict` 零新增错误（vm_context.go 新方法多属 getter，行数须自控）
- [ ] gofmt/vet 零新增；`vm.ctx == nil` 非测试文件计数 = 0
- [ ] S4 等价表完整（每站点一行）；不等价站点清单（若有）逐条 [转交决策] 记录
- [ ] mutation×2：①`NewVM` 不注入 noop（ctx 留 nil）→ S5a RED；②`noopContext.Bars()` 改返 nil interface → 序列类 builtin panic → 对应测试 RED；restore→GREEN

## 边界/不做

- 不修 `builtinOrderSend` 无 broker 静默 `-1`+nil error 的 fail-closed 缺陷本身——spec §4.4 指定"另立"，本单只保留其行为不变（验收时决策方记债 `ORDERSEND-NILBROKER-FAILCLOSED-1`）。
- 不改 `strategy/runner`、`strategy/backtest` 的 Context 实现；不改 sdk 接口。
- 测试文件里的 mock ctx（各 `*_test.go` 的 SetContext 调用）不动，除非与不变量冲突。

## D-013 出件自审记录（决策方）

- 坐标回验：101/15 计数实测（非测试文件）；`NewVM` 唯一 ctor（全仓 `&VM{` 唯一命中）；`sdk.Context` 接口 105 行全文已读；`barSlice` 零值安全实现已读（series.go:77-83 `at()` 越界返 `Bar{}`）。
- 路径可达性：`SetContext` 仅 `interp_runner.go` 八处调用（无测试外路径传 nil 的实证，但归一化防御成本一行，保留）。
- 生命周期：`vm.cachedHistory/cachedPositions/cachedOrders` 的 `!=nil && Broker()!=nil` 复合守卫是每事件缓存填充条件——noop 下语义保持（Broker nil → 不填充 → 走既有空缓存分支），已列入 S4 规则。
- 等价性盲区预判：指标 builtin 的 nil 分支返回值 vs `noopIndicatorSet` 零值路径——S4 等价表强制逐站核对，不允许批量 sed 式替换后跑测试了事。
- 冲突检查：与 QS-2.5 panic 收敛无交互（noop 不 panic）；与 FAILCLOSED-1 无冲突（无 broker 路径行为原样）。
- **署名**：最终决策：Devin CLI（[角色:决策终] 激活）
