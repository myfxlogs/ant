# 施工派工单：VM-API-TRUTH-1（批次2c：AccountInfo* 混合实现假分支修复 + 枚举编号对齐）

> **角色纪律**：你是施工方，不是验收方。完成后保持 `🟦open（施工完成，待独立复审）`，由 Devin CLI 独立复审后才可变更为 `✅done`。禁止扩 scope、提交、push、部署。commit 用 `ANT_ROLE=builder` 前缀。

## 立项背景

**触发**：`docs/audits/tech-debt-registry.md:126` VM-API-TRUTH-1（P1）。批次1 ✅done（commit e97a43b8，MQL5 order/deal/history 22 API）。批次2a ✅done（commit 8f946579，platform checkup 12 API）。批次2b ✅done（commit 1fb352f1，Account 全假 + Symbol by-reference 5 API）。本批 2c 修复 `AccountInfoDouble`/`AccountInfoInteger`/`AccountInfoString` 的混合实现——**非重分类**，函数保持 implemented，假分支改为 fail-closed。

**本批范围**：3 个函数的运行时分支 + `constants.go` 枚举常量。机制：`fmt.Errorf` 返回 → `callBuiltin` 置 `vm.fatalError` + `recordBlindSpot`（`vm_helpers.go:241-242` 自动双记）→ `runLoop` 顶检查 → `OnInit`/`OnTick`/`Engine.Run` fail-closed。不用 recordBlindSpot-only（静默遥测让策略继续跑在假数据上，违反 fail-closed 红线）。

**证据链**（HEAD 实拍，2026-09-17 Devin CLI 设计复审）：

### A. 假分支（无权威源却返回固定值）

| 函数:分支 | 文件:行 | 现返回 | 裁定 |
|---|---|---|---|
| `AccountInfoDouble` prop=1 (ACCOUNT_CREDIT) | `vm_builtin_mql5_info.go:47-48` | `decimalZero` | error（同批次2b `AccountCredit` 裁定：回测无 credit 概念） |
| `AccountInfoDouble` default | `:62-63` | `decimalZero` 静默 | error |
| `AccountInfoInteger` prop=35/36 (TRADE_ALLOWED/TRADE_EXPERT) | `:75-78` | `BoolVal(true)` 固定 | 接 `Account().IsTradeAllowed`（见分支处置表） |
| `AccountInfoInteger` default | `:79-80` | `IntVal(0)` 静默 | error |
| `AccountInfoString` prop=0..3 + default | `:84-98` | `"USD"`/`"Backtest"`/`"SimBroker"`/`""` 固定 | 见分支处置表 |

### B. prop 编号偏离真 MQL5 枚举（设计实查新发现）

| 枚举 | 真 MQL5 值 | 代码现值 | 后果 |
|---|---|---|---|
| ENUM_ACCOUNT_INFO_INTEGER | LOGIN=0, TRADE_MODE=1, **LEVERAGE=2**, LIMIT_ORDERS=3, MARGIN_SO_MODE=4, **TRADE_ALLOWED=5**, **TRADE_EXPERT=6**, MARGIN_MODE=7, CURRENCY_DIGITS=8, FIFO_CLOSE=9, HEDGE_ALLOWED=10 | 代码 `case 32/35/36` | `AccountInfoInteger(2)`（真值 LEVERAGE）落 default→0；32 是越界值却返回 Leverage |
| ENUM_ACCOUNT_INFO_STRING | **NAME=0, SERVER=1, CURRENCY=2**, COMPANY=3 | 代码 case 0→"USD"(自称CURRENCY), 1→NAME, 2→SERVER | prop 全错位：NAME(0) 得 "USD"、CURRENCY(2) 得 "SimBroker" |
| constants.go double 常量 | MARGIN_SO_CALL=7, SO_SO=8, INITIAL=9, MAINTENANCE=10 | `IntVal(7)`→INITIAL、`IntVal(8)`→MAINTENANCE、`IntVal(9)`→SO_CALL、`IntVal(10)`→SO_SO | 4 个常量值全部错（SO_* 与 INITIAL/MAINTENANCE 互换） |
| 命名常量 | ACCOUNT_LEVERAGE/TRADE_ALLOWED/LOGIN/NAME/CURRENCY 等 | **未定义** | 真 MQL5 源码 `AccountInfoInteger(ACCOUNT_LEVERAGE)` 编译失败（unknown identifier） |

### C. 权威源实拍（`sdk.AccountInfo` `strategy/sdk/broker.go:46-64` + `ctx.Mode()` `strategy/sdk/context.go:77`）

| 字段/通道 | SimBroker 回测 `strategy/backtest/broker.go:627` | live runner `strategy/runner/broker.go:203` | 判定 |
|---|---|---|---|
| Balance/Equity/Margin/FreeMargin/Leverage | ✅ | ✅（Leverage 未填充→0） | 真实通道 |
| IsDemo/IsConnected/IsTradeAllowed | ✅ | ✅ | 真实通道 |
| Login | ❌（→0） | ✅ | 真实通道 |
| Currency | ✅"USD" | ❌（→""） | 回测真实；live 空=缺失事实 |
| Company/Mode/MarginLevel | ❌ 无人填充 | ❌ | 无数据源 |

**不变量**：本批只改 3 函数 + constants.go + 测试文件。`AccountInfoDouble/Integer/String` 保留在 `implementedAccount`（函数仍 implemented）。`AccountBalance`/`AccountEquity`/`AccountFreeMargin`/`AccountMargin`/`AccountLeverage`/`AccountNumber`（真实实现）不动。QS-2.3 批准的 `AccountInfoInteger` LEVERAGE 分支 `vm.ctx != nil → IntVal(100)` 守卫（17 站保留裁定，nil→100 vs noop→0 非等价）**必须保留**，不得"顺手清理"。

## 设计 SSOT 声明

- 设计文档：本派工单（唯一真相源）
- 相关契约：`AGENTS.md` §0 fail-closed 红线、§7.2 无死代码
- 裁定：`docs/audits/tech-debt-registry.md:126` 2026-09-16 Devin CLI 裁定（StatusUnsupported 方向）+ 本单分支处置表（Devin CLI 设计复审 2026-09-17）

## 分支处置表（裁定 SSOT，逐格执行）

### `builtinAccountInfoDouble`（真 ENUM_ACCOUNT_INFO_DOUBLE 编号）

| prop | 分支 | 处置 |
|---|---|---|
| 0 ACCOUNT_BALANCE | `Account().Balance` | 保留（真实） |
| 1 ACCOUNT_CREDIT | — | **error** |
| 2 ACCOUNT_PROFIT | `Account().Equity.Sub(Balance)` | 保留（真实） |
| 3 ACCOUNT_EQUITY | `Account().Equity` | 保留（真实） |
| 4 ACCOUNT_MARGIN | `Account().Margin` | 保留（真实） |
| 5 ACCOUNT_MARGIN_FREE | `Account().FreeMargin` | 保留（真实） |
| 6 ACCOUNT_MARGIN_LEVEL | `Equity.Div(Margin)`（保留 Margin.IsZero 守卫） | 保留（真实） |
| 7-13 SO_CALL/SO_SO/INITIAL/MAINTENANCE/ASSETS/LIABILITIES/COMMISSION_BLOCKED | — | **error**（无源；回测无 stopout 档位，与批次2b AccountStopoutMode 裁定一致） |
| default（含 prop<0/prop>13） | — | **error** |

### `builtinAccountInfoInteger`（真 ENUM_ACCOUNT_INFO_INTEGER 编号）

| prop | 分支 | 处置 |
|---|---|---|
| 0 ACCOUNT_LOGIN | `IntVal(Account().Login)` | **新接**（真实通道；SimBroker 未填充→0=未上报，诚实字段值） |
| 1 ACCOUNT_TRADE_MODE | `IsDemo → IntVal(0)`(DEMO) / `else → IntVal(2)`(REAL) | **新接**（文档化语义收窄：contest 并入 demo——IsDemo 字段定义已含 contest，见 broker.go:61 注释） |
| 2 ACCOUNT_LEVERAGE | `if vm.ctx != nil → IntVal(Account().Leverage)` / `else → IntVal(100)` | 修正 case 32→2；**nil 守卫原样保留**（QS-2.3 批准） |
| 5 ACCOUNT_TRADE_ALLOWED | `IsTradeAllowed → IntVal(0/1)` | 接源（修 35→5；返回值改 IntVal——函数签名 long，BoolVal 是潜在类型失真） |
| 6 ACCOUNT_TRADE_EXPERT | `IsTradeAllowed → IntVal(0/1)` | 接源（修 36→6；**文档化环境语义**：本 VM 唯一交易行为体即 EA，"EA 许可"≡"账户许可"；mtapi 单标志无法区分 manual-only 账户，如实声明） |
| 7 ACCOUNT_MARGIN_MODE | `ctx.Mode()`：`ModeNetting → IntVal(0)`(RETAIL_NETTING) / `ModeHedging → IntVal(2)`(RETAIL_HEDGING) / 其他(含"")→error | **新接**（真实通道；Mode 未填充=缺失事实→error） |
| 10 ACCOUNT_HEDGE_ALLOWED | `ctx.Mode()==ModeHedging → IntVal(1)` / `ModeNetting → IntVal(0)` / 其他→error | **新接** |
| 3 LIMIT_ORDERS / 4 MARGIN_SO_MODE / 8 CURRENCY_DIGITS / 9 FIFO_CLOSE | — | **error**（无源） |
| default | — | **error** |

### `builtinAccountInfoString`（真 ENUM_ACCOUNT_INFO_STRING 编号）

| prop | 分支 | 处置 |
|---|---|---|
| 0 ACCOUNT_NAME | — | **error**（无字段） |
| 1 ACCOUNT_SERVER | — | **error**（无字段） |
| 2 ACCOUNT_CURRENCY | `Account().Currency`；**`=="" → error`** | 接源（修正 case 2；空=缺失事实 fail-closed） |
| 3 ACCOUNT_COMPANY | `Account().Company`；**`=="" → error`** | 接源（当前无生产者→恒 error，等价 fail-closed 直到 live 管线填充） |
| default | — | **error** |

### error 消息格式

统一 `fmt.Errorf("AccountInfoDouble: prop %d (%s) has no authoritative source in the VM", prop, propName)`；未知 prop 用 `fmt.Errorf("AccountInfoDouble: unknown prop %d", prop)`（Integer/String 同理换函数名）。需要 `import "fmt"`（该文件当前只 import interp）。

## 约束与目标

- **目标**：3 函数假分支全部 fail-closed 或接权威源；prop 编号对齐真 MQL5；命名常量补齐使真 MQL5 源码可编译。
- **范围**：仅以下 4 文件：
  - `backend/tools/mql2go/interp/constants.go`（常量修正+补齐）
  - `backend/tools/mql2go/vm_builtin_mql5_info.go`（3 函数重写）
  - `backend/tools/mql2go/vm_api_truth_test.go`（追加 S6j–S6o 测试）
  - `backend/tools/mql2go/vm_api_truth3_batch3_test.go`（仅扩展 `accountStatusTestContext` 字段，加性不改签名）
- **测试**：复用同包 `compileAndInit`/`accountStatusTestContext`（`vm_api_truth3_batch3_test.go` 已定义，直接引用）。

## 边界 / 不做

- **不重分类** `AccountInfoDouble/Integer/String`——保留 `implementedAccount`（`builtin_registry.go:66`）与 `LookupAPI=StatusImplemented`。
- **不改** noop 整 API：`AccountProfit`/`AccountCurrency`/`AccountCompany`/`AccountName`/`AccountServer`/`AccountStopoutLevel`/`AccountFreeMarginCheck`/`AccountFreeMarginMode`（`vm_builtin_impls.go:143-151`，属后续重分类批次）。
- **不改** `SymbolInfoDouble/Integer/String`/`MarketInfo`/`SymbolInfoTick`/`SymbolName`/`SymbolSelect`/`SymbolsTotal`/`SymbolIsSynchronized`。
- **不改** live 管线（live Leverage/Currency/Mode 未填充是管线缺口，本批如实暴露为 error/0，另立债 `LIVE-ACCOUNT-FIELDS-1`）。
- **不改** `analyze.go looksLikeMQLBuiltin`、python 映射 `compile_py_mapping.go`。
- **不加** nil ctx 守卫到新分支（生产不变量 `NewVM` 注入 noopContext；nil 只在直接调 builtin 的测试里，现有 ctx 读取函数同此约定）。LEVERAGE 既有守卫除外（保留）。

## 施工指令

### S1 — constants.go：修正 4 个错值 + 补齐命名常量

- **坐标**：`backend/tools/mql2go/interp/constants.go:455-468`（account info double 常量块，批次2b 后实测）。
- **落点**：
  - 修正：`"ACCOUNT_MARGIN_SO_CALL": IntVal(7)`、`"ACCOUNT_MARGIN_SO_SO": IntVal(8)`、`"ACCOUNT_MARGIN_INITIAL": IntVal(9)`、`"ACCOUNT_MARGIN_MAINTENANCE": IntVal(10)`（现为 9/10/7/8 错位）。
  - 追加 double 余量：`"ACCOUNT_ASSETS": IntVal(11)`、`"ACCOUNT_LIABILITIES": IntVal(12)`、`"ACCOUNT_COMMISSION_BLOCKED": IntVal(13)`。
  - 追加 int props 块（真 ENUM_ACCOUNT_INFO_INTEGER）：
    ```go
    // ── Account info integer properties (MQL5 ENUM_ACCOUNT_INFO_INTEGER) ──
    "ACCOUNT_LOGIN":           IntVal(0),
    "ACCOUNT_TRADE_MODE":      IntVal(1),
    "ACCOUNT_LEVERAGE":        IntVal(2),
    "ACCOUNT_LIMIT_ORDERS":    IntVal(3),
    "ACCOUNT_MARGIN_SO_MODE":  IntVal(4),
    "ACCOUNT_TRADE_ALLOWED":   IntVal(5),
    "ACCOUNT_TRADE_EXPERT":    IntVal(6),
    "ACCOUNT_MARGIN_MODE":     IntVal(7),
    "ACCOUNT_CURRENCY_DIGITS": IntVal(8),
    "ACCOUNT_FIFO_CLOSE":      IntVal(9),
    "ACCOUNT_HEDGE_ALLOWED":   IntVal(10),
    ```
  - 追加 string props 块（真 ENUM_ACCOUNT_INFO_STRING）：
    ```go
    // ── Account info string properties (MQL5 ENUM_ACCOUNT_INFO_STRING) ──
    "ACCOUNT_NAME":     IntVal(0),
    "ACCOUNT_SERVER":   IntVal(1),
    "ACCOUNT_CURRENCY": IntVal(2),
    "ACCOUNT_COMPANY":  IntVal(3),
    ```
  - 追加枚举值块：
    ```go
    // ── Account trade modes (ENUM_ACCOUNT_TRADE_MODE) ──
    "ACCOUNT_TRADE_MODE_DEMO":    IntVal(0),
    "ACCOUNT_TRADE_MODE_CONTEST": IntVal(1),
    "ACCOUNT_TRADE_MODE_REAL":    IntVal(2),
    // ── Account margin modes (ENUM_ACCOUNT_MARGIN_MODE) ──
    "ACCOUNT_MARGIN_MODE_RETAIL_NETTING": IntVal(0),
    "ACCOUNT_MARGIN_MODE_EXCHANGE":       IntVal(1),
    "ACCOUNT_MARGIN_MODE_RETAIL_HEDGING": IntVal(2),
    ```
- **验证**：`LookupMQLConstant("ACCOUNT_LEVERAGE")` → `IntVal(2)`；`LookupMQLConstant("ACCOUNT_MARGIN_SO_CALL")` → `IntVal(7)`。

### S2 — `builtinAccountInfoDouble` 重写

- **坐标**：`backend/tools/mql2go/vm_builtin_mql5_info.go:42-65`（批次2b 后实测）。
- **落点**：按分支处置表重写 switch——保留 0/2/3/4/5/6 真实分支；case 1 + case 7-13（有名 prop 可合并 `case 1, 7, 8, 9, 10, 11, 12, 13:`）+ default 全部返回 error。`import "fmt"` 加入文件头。
- **验证**：`AccountInfoDouble(0)` 返回 Balance；`AccountInfoDouble(1)`/`AccountInfoDouble(99)` 返回 error。

### S3 — `builtinAccountInfoInteger` 重写

- **坐标**：`backend/tools/mql2go/vm_builtin_mql5_info.go:67-82`（批次2b 后实测）。
- **落点**：按分支处置表重写 switch——
  ```go
  case 0:  return interp.IntVal(int32(vm.ctx.Account().Login)), nil
  case 1:  if vm.ctx.Account().IsDemo { return interp.IntVal(0), nil }; return interp.IntVal(2), nil
  case 2:  if vm.ctx != nil { return interp.IntVal(vm.ctx.Account().Leverage), nil }; return interp.IntVal(100), nil
  case 5, 6: if vm.ctx.Account().IsTradeAllowed { return interp.IntVal(1), nil }; return interp.IntVal(0), nil
  case 7:  switch vm.ctx.Mode() { case sdk.ModeNetting: return interp.IntVal(0), nil; case sdk.ModeHedging: return interp.IntVal(2), nil; default: → error }
  case 10: switch vm.ctx.Mode() { case sdk.ModeHedging: return interp.IntVal(1), nil; case sdk.ModeNetting: return interp.IntVal(0), nil; default: → error }
  default: → error
  ```
  注意 `sdk` import（`alphaforge/strategy/sdk`）。
- **验证**：`AccountInfoInteger(2)` 返回 Leverage；`AccountInfoInteger(32)` 返回 error（旧编号回归）。

### S4 — `builtinAccountInfoString` 重写

- **坐标**：`backend/tools/mql2go/vm_builtin_mql5_info.go:84-98`（批次2b 后实测）。
- **落点**：按分支处置表重写 switch——
  ```go
  case 2: if c := vm.ctx.Account().Currency; c != "" { return interp.StringVal(c), nil } → error
  case 3: if c := vm.ctx.Account().Company; c != "" { return interp.StringVal(c), nil } → error
  case 0, 1, default: → error
  ```
- **验证**：`AccountInfoString(2)` 返回 Currency；`AccountInfoString(0)` 返回 error。

### S5 — `accountStatusTestContext` 扩展（仅加性）

- **坐标**：`backend/tools/mql2go/vm_api_truth3_batch3_test.go:185-196`。
- **落点**：结构体追加字段 `leverage int32`、`login int64`、`currency string`、`company string`、`mode sdk.AccountMode`；`Account()` 返回体追加对应字段赋值。`Mode()` 方法改返回 `c.mode`（当前固定 `sdk.ModeHedging`——改为字段后既有 truth3 测试默认值变 ""，若造成断言漂移则保留 `Mode()` 默认 `sdk.ModeHedging` 仅当 `c.mode==""`，即 `if c.mode != "" { return c.mode }; return sdk.ModeHedging`）。
- **验证**：`go test ./tools/mql2go/ -run 'TestBuiltinIsTradeAllowed|TestVMLive_IsTradeAllowed'` 不回归。

### S6 — 追加对抗测试（`vm_api_truth_test.go` 现有文件末尾）

- **S6j — 命名常量+真分支端到端**：`TestVM_API_TRUTH_1_AccountInfoLeverageReal`——`long g=0; int OnInit(){ g=AccountInfoInteger(ACCOUNT_LEVERAGE); return 0; }`，ctx leverage=25 → `compileAndInit` + `GetGlobal("g")` 断言 25。证明：命名常量可编译 + 真枚举值 2 + 真实分支。
- **S6k — TRADE_ALLOWED 双向**：`TestVM_API_TRUTH_1_AccountInfoTradeAllowedReadsSource`——`AccountInfoInteger(ACCOUNT_TRADE_ALLOWED)`，ctx IsTradeAllowed=false→g=0 / true→g=1。
- **S6l — 新接真实分支**：`TestVM_API_TRUTH_1_AccountInfoNewRealBranches`——表驱动：(ACCOUNT_LOGIN→login=77)、(ACCOUNT_TRADE_MODE→IsDemo=true→0 / false→2)、(ACCOUNT_MARGIN_MODE→mode=hedging→2 / netting→0)、(ACCOUNT_HEDGE_ALLOWED→hedging→1 / netting→0)、(ACCOUNT_TRADE_EXPERT→IsTradeAllowed→0/1)。
- **S6m — 假分支 fail-closed**：`TestVM_API_TRUTH_1_AccountInfoFakePropsError`——表驱动 `AccountInfoDouble(1)/AccountInfoDouble(7)/AccountInfoDouble(99)`、`AccountInfoInteger(3)/AccountInfoInteger(4)/AccountInfoInteger(8)/AccountInfoInteger(9)/AccountInfoInteger(99)`、`AccountInfoString(0)/AccountInfoString(1)/AccountInfoString(99)`，各构造 `int OnInit(){ AccountInfoX(P); return 0; }` 直接调 `runner.OnInit(ctx)`（非 compileAndInit），断言 `err != nil`。
- **S6n — 枚举编号回归**：`TestVM_API_TRUTH_1_AccountInfoEnumNumbering`——`AccountInfoInteger(32)`（旧假编号）→ OnInit error；`AccountInfoString(ACCOUNT_CURRENCY)` ctx currency="EUR"→g="EUR"；ctx currency=""→OnInit error。
- **S6o — 常量值钉住**：`TestVM_API_TRUTH_1_AccountInfoConstantValues`——`LookupMQLConstant` 断言：`ACCOUNT_LEVERAGE=2`、`ACCOUNT_TRADE_ALLOWED=5`、`ACCOUNT_TRADE_EXPERT=6`、`ACCOUNT_LOGIN=0`、`ACCOUNT_MARGIN_SO_CALL=7`、`ACCOUNT_MARGIN_SO_SO=8`、`ACCOUNT_MARGIN_INITIAL=9`、`ACCOUNT_MARGIN_MAINTENANCE=10`、`ACCOUNT_NAME=0`、`ACCOUNT_SERVER=1`、`ACCOUNT_CURRENCY=2`、`ACCOUNT_COMPANY=3`。
- **S6p — 函数未误重分类**：`TestVM_API_TRUTH_1_AccountInfoStillImplemented`——`IsAPIImplemented("AccountInfoDouble"/"AccountInfoInteger"/"AccountInfoString")` 全 true（误删 implementedAccount → RED）。
- **验证**：`go test ./tools/mql2go/ -run TestVM_API_TRUTH_1` 全绿（批次1/2a/2b + 2c 测试共存）。

## 验收标准

- [ ] `go build ./...` 通过
- [ ] `go test ./tools/mql2go/` 全过（含 S6j–S6p 新增）
- [ ] `cd backend && go run ./tools/check-file-lines --strict` 零错误
- [ ] `gofmt` / `go vet` 零警告
- [ ] `go test -race -count=3 ./tools/mql2go/` 通过
- [ ] **对抗证明**（逐项 mutation RED→restore→GREEN，附命令与输出）：
  1. `builtinAccountInfoDouble` case 1 恢复 `return interp.DecimalVal(decimalZero), nil` → S6m RED
  2. `builtinAccountInfoInteger` case 2 改回 `case 32:` → S6j RED（prop 2 落 default→error）
  3. `constants.go` 删 `"ACCOUNT_LEVERAGE"` 行 → S6j 编译 RED（unknown identifier）
  4. `builtinAccountInfoString` case 2 恢复固定 `"USD"` → S6n RED
- [ ] diff 通读无死代码 / TODO / 调试残留 / 范围外改动

## 施工完成自审（强制，D-012）

交付自报前必须完成并随报提交：
- [ ] 逐项重跑上方验收标准并贴真实输出
- [ ] 红队自审 diff 三问：更简等价方案 / 边界·nil·并发 / 逆向依赖或重复基础设施
- [ ] 自审发现的缺陷已修复至全绿（自报列出发现项+修复项）
- [ ] 无自审记录 = 复审直接退回

## 交付格式

自报必须按以下六段顺序（D-016，缺一 = 复审直接退回）：
1. **变更文件清单**：列出本任务改动的所有文件。
2. **S1-Sn 实现摘要**：每步落点对码（改了什么、在哪、是否符合派工单坐标）。
3. **对抗证明**：mutation RED → restore → GREEN 的命令与关键输出。
4. **机检门禁**：build / test / race×3 / vet / gofmt / check-lines / diff --check 逐项真实输出。
5. **范围确认**：仅改派工单列出的文件，无范围外改动。
6. **结束语**：`[施工完成:VM-API-TRUTH-1-批次2c] @<commit-hash>`（D-014，无此行 = 未交付，复审不启动）。

**停手等 Devin CLI 复审；不达标将由决策方开缺陷清单退回。勿部署，禁 `--no-verify`。**

## 复审注记（Devin CLI 设计复审记录，非施工内容）

- **LIVE-ACCOUNT-FIELDS-1 候选债**：live runner `Account()` 未填充 Leverage/Currency/Company/Mode（harness `live*` 字段无对应通道），本批如实暴露为 0/error；如需 live 完整 AccountInfo 语义，须从 mt_accounts/AccountSummary 经 live context 补管线，另立债。
- **同族剩余假 API 队列**：`AccountProfit`(noopDecimal→可用 Equity-Balance 实接)/`AccountCurrency`/`AccountCompany`/`AccountName`/`AccountServer`/`AccountStopoutLevel`/`AccountFreeMarginCheck`/`AccountFreeMarginMode` 8 个 noop 整 API（`vm_builtin_impls.go:143-151`），其中 `AccountCurrency/AccountCompany` 有字段可实接、其余重分类；含 python `account.profit`→noopDecimal 静默 0 的暴露面。归后续批次（2e/2f）。
