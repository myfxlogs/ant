# 施工派工单：VM-API-TRUTH-1（批次2e：account noop 整 API——4 重分类 + 4 实接）

> **角色纪律**：你是施工方，不是验收方。完成后保持 `🟦open（施工完成，待独立复审）`，由 Devin CLI 独立复审后才可变更为 `✅done`。禁止扩 scope、提交、push、部署。commit 用 `ANT_ROLE=builder` 前缀。

## 立项背景

**触发**：`docs/audits/tech-debt-registry.md:126` VM-API-TRUTH-1（P1）。批次1 ✅done、批次2a ✅done、批次2b/2c/2d 派工单已出。本批 2e 处理 `vm_builtin_impls.go:143-151` 的 account noop 整 API——一部分重分类 `StatusUnsupported`（无权威源），一部分实接权威源（"可由 VM 已有权威输入完整实现的 API 另行补齐"裁定）。

**本批范围**：8 个 API——`vm_builtin_impls.go` 当前全接 `builtinNoopDecimal`/`builtinNoopString`/`builtinNoopInt` 固定假值。

**证据链**（HEAD 实拍，2026-09-17 Devin CLI 设计复审）：

| API名 | 文件:行 | 现接线/返回 | 真实数据源 | 裁定 |
|---|---|---|---|---|
| `AccountProfit` | `vm_builtin_impls.go:143` | `builtinNoopDecimal`→0 | `Account().Equity.Sub(Balance)`（同 AccountInfoDouble prop=2 真实分支） | **实接**（含 python `account.profit`→`AccountProfit` 静默 0 修复，`compile_py_mapping.go:62`） |
| `AccountCurrency` | `:144` | `builtinNoopString`→"" | `Account().Currency`（空→error，同批次2c 裁定） | **实接** |
| `AccountCompany` | `:145` | `builtinNoopString`→"" | `Account().Company`（空→error） | **实接** |
| `AccountFreeMarginCheck` | `:150` | `builtinNoopDecimal`→0 | 可推导：`FreeMargin − volume·contractSize·price/leverage`（全部输入权威：Ask/Bid/ContractSize/Leverage/FreeMargin） | **实接** |
| `AccountName` | `:148` | `builtinNoopString`→"" | 无（`sdk.AccountInfo` 无 Name 字段） | 重分类 |
| `AccountServer` | `:149` | `builtinNoopString`→"" | 无（无 Server 字段） | 重分类 |
| `AccountStopoutLevel` | `:147` | `builtinNoopInt`→0 | 无（回测无 stopout 档位概念，同批次2b AccountStopoutMode 裁定） | 重分类 |
| `AccountFreeMarginMode` | `:151` | `builtinNoopInt`→0 | 无（MQL4 margin mode 枚举无源） | 重分类 |

**保留（不误伤）**：`AccountBalance`/`AccountEquity`/`AccountFreeMargin`/`AccountMargin`/`AccountLeverage`/`AccountNumber`（真实实现，`vm_builtin_account.go` + `vm_builtin_string.go:183`）。

**不变量**：本批只改 8 API。`builtinNoopDecimal`/`builtinNoopString`/`builtinNoopInt`/`builtinNoop` shared helper 保留（其他 API 仍用）。`implementedAccount` 保留实接 4 + 既有真实 6。

## 设计 SSOT 声明

- 设计文档：本派工单（唯一真相源）
- 相关契约：`AGENTS.md` §0 fail-closed 红线、§7.2 无死代码
- 裁定：`docs/audits/tech-debt-registry.md:126` 2026-09-16 Devin CLI 裁定 + 本单分支处置（Devin CLI 设计复审 2026-09-17）

## 约束与目标

- **目标**：4 API 重分类 StatusUnsupported 编译期拒绝；4 API 实接权威源；删除假接线无死代码。
- **范围**：仅以下 5 文件：
  - `backend/tools/mql2go/interp/api_registry.go`（unsupportedSymbols 添加 4，reason 复用 `reasonAccountSymbolStub`）
  - `backend/tools/mql2go/interp/builtin_registry.go`（implementedAccount 移除 4）
  - `backend/tools/mql2go/builtins.go`（删除 4 个 nil 注册）
  - `backend/tools/mql2go/vm_builtin_impls.go`（4 个 noop 绑定删除 + 4 个改接真实函数）
  - `backend/tools/mql2go/vm_builtin_account.go`（新增 4 个真实实现函数）
- **测试**：`backend/tools/mql2go/vm_api_truth_test.go`（追加 S6t/u/v 测试到现有文件）

## 边界 / 不做

- **不改** `builtinNoop*` shared helper（`vm_builtin_util.go`，其他 API 仍依赖）。
- **不改** `AccountBalance`/`AccountEquity`/`AccountFreeMargin`/`AccountMargin`/`AccountLeverage`/`AccountNumber`（真实）。
- **不改** `AccountInfoDouble/Integer/String`（批次2c）、`SymbolInfo*`/`MarketInfo`（VM-ENUM-NUMBERING-1 另立）、`analyze.go`/`compile_py_mapping.go`。
- **AccountFreeMarginCheck 推导边界**：仅 cmd=0(OP_BUY)/1(OP_SELL)，其他 cmd→error；`Broker().SymbolInfo` err→error；`ctx.Broker()==nil`→error（非静默 0）。公式：`FreeMargin − volume·ContractSize·price/Leverage`（price=BUY→Ask/SELL→Bid）；**不含潜在亏损项**（开仓价处亏损=0，文档化收窄）。
- **坐标漂移**：行号以符号锚点为准。

## 施工指令

### S1 — unsupportedSymbols 添加 4 API

- **坐标**：`backend/tools/mql2go/interp/api_registry.go`（unsupportedSymbols 批次2b/2d 块后）。
- **落点**：追加 4 行（注释块标明批次2e，reason 复用 `reasonAccountSymbolStub`——若批次2b 落名不同则复用其常量）：
  ```go
  // VM-API-TRUTH-1 batch 2e: account stubs returning fixed values
  // without authoritative data. Reclassified StatusUnsupported.
  {Name: "AccountName", Status: StatusUnsupported, Category: CatFunction, Reason: reasonAccountSymbolStub},
  {Name: "AccountServer", Status: StatusUnsupported, Category: CatFunction, Reason: reasonAccountSymbolStub},
  {Name: "AccountStopoutLevel", Status: StatusUnsupported, Category: CatFunction, Reason: reasonAccountSymbolStub},
  {Name: "AccountFreeMarginMode", Status: StatusUnsupported, Category: CatFunction, Reason: reasonAccountSymbolStub},
  ```
- **验证**：`LookupAPI("AccountName")` → `StatusUnsupported`。

### S2 — implementedAccount 移除 4 API

- **坐标**：`backend/tools/mql2go/interp/builtin_registry.go`（implementedAccount `:56-67`）。
- **落点**：删 `"AccountName"`, `"AccountServer"`, `"AccountStopoutLevel"`, `"AccountFreeMarginMode"`。**保留** `"AccountFreeMarginCheck"`（实接）+ `AccountInfoDouble/Integer/String` + 真实 6。
- **验证**：`TestImplementedNamesHaveVMHandlers` 通过。

### S3 — builtins.go 删除 4 个 nil 注册

- **坐标**：`backend/tools/mql2go/builtins.go`（AccountName:135 / AccountCompany:132 / AccountServer / AccountStopoutLevel / AccountFreeMarginMode 附近，行号以符号为准）。
- **落点**：删 `{"AccountName", nil}`、`{"AccountServer", nil}`、`{"AccountStopoutLevel", nil}`、`{"AccountFreeMarginMode", nil}` 4 行。**保留** `AccountCurrency`/`AccountCompany`/`AccountFreeMarginCheck`/`AccountProfit`（实接）+ 真实 6。
- **验证**：`TestNoDuplicateBuiltins` 通过。

### S4 — vm_builtin_impls.go 换接 8 个绑定

- **坐标**：`backend/tools/mql2go/vm_builtin_impls.go:143-151`（registerAccountBuiltins）。
- **落点**：
  - 删 4 行：`id("AccountName")`/`id("AccountServer")`/`id("AccountStopoutLevel")`/`id("AccountFreeMarginMode")` 的 noop 绑定。
  - 改接 4 行：`id("AccountProfit").fn = builtinAccountProfit`、`id("AccountCurrency").fn = builtinAccountCurrency`、`id("AccountCompany").fn = builtinAccountCompany`、`id("AccountFreeMarginCheck").fn = builtinAccountFreeMarginCheck`。
- **验证**：`TestAllBuiltinsWired` 通过。

### S5 — vm_builtin_account.go 新增 4 个真实实现函数

- **坐标**：`backend/tools/mql2go/vm_builtin_account.go`（追加到 account builtins 块，`builtinAccountLeverage` 后、symbol info 块前）。
- **落点**：
  ```go
  func builtinAccountProfit(vm *VM, args []interp.Value) (interp.Value, error) {
      return interp.DecimalVal(vm.ctx.Account().Equity.Sub(vm.ctx.Account().Balance)), nil
  }
  func builtinAccountCurrency(vm *VM, args []interp.Value) (interp.Value, error) {
      c := vm.ctx.Account().Currency
      if c == "" {
          return interp.StringVal(""), fmt.Errorf("AccountCurrency: no authoritative currency in the VM")
      }
      return interp.StringVal(c), nil
  }
  func builtinAccountCompany(vm *VM, args []interp.Value) (interp.Value, error) {
      c := vm.ctx.Account().Company
      if c == "" {
          return interp.StringVal(""), fmt.Errorf("AccountCompany: no authoritative company in the VM")
      }
      return interp.StringVal(c), nil
  }
  func builtinAccountFreeMarginCheck(vm *VM, args []interp.Value) (interp.Value, error) {
      if vm.ctx.Broker() == nil {
          return interp.DecimalVal(decimal.Zero), fmt.Errorf("AccountFreeMarginCheck: no broker in the VM")
      }
      sym := argS(args, 0); if sym == "" { sym = vm.ctx.Symbol() }
      cmd := argI(args, 1)
      volume := argD(args, 2) // argD: out-of-range → decimal.Zero（volume=0 → required=0 → FreeMargin）
      info, err := vm.ctx.Broker().SymbolInfo(sym)
      if err != nil {
          return interp.DecimalVal(decimal.Zero), fmt.Errorf("AccountFreeMarginCheck: %w", err)
      }
      var price decimal.Decimal
      switch cmd {
      case 0: price = vm.ctx.Ask()  // OP_BUY
      case 1: price = vm.ctx.Bid()  // OP_SELL
      default:
          return interp.DecimalVal(decimal.Zero), fmt.Errorf("AccountFreeMarginCheck: invalid cmd %d (must be 0/1)", cmd)
      }
      lev := decimal.NewFromInt(int64(vm.ctx.Account().Leverage))
      if lev.IsZero() {
          return interp.DecimalVal(decimal.Zero), fmt.Errorf("AccountFreeMarginCheck: leverage is zero")
      }
      required := volume.Mul(info.ContractSize).Mul(price).Div(lev)
      return interp.DecimalVal(vm.ctx.Account().FreeMargin.Sub(required)), nil
  }
  ```
  `import "fmt"` 按需（文件当前 imports 核对）。
- **验证**：`builtinAccountProfit` 返回 Equity−Balance；`builtinAccountFreeMarginCheck` cmd=2 → error。

### S6 — 追加对抗测试（`vm_api_truth_test.go` 现有文件末尾）

- **S6t — 重分类编译期拒绝**：`TestVM_API_TRUTH_1_AccountNoopRejected`：`unsupportedAccountNoop = []string{"AccountName","AccountServer","AccountStopoutLevel","AccountFreeMarginMode"}`，最小源码断言 `CompileMQL` error 含 "unsupported" 或 API 名。
- **S6u — 实接行为**：`TestVM_API_TRUTH_1_AccountNoopRealImpl`：`AccountProfit()` → Equity−Balance；`AccountCurrency()` ctx currency="EUR"→"EUR"、""→error；`AccountCompany()` 同；`AccountFreeMarginCheck(sym,0,vol)` → FreeMargin−required（断言推导公式数值）、cmd=9→error、broker nil→error。
- **S6v — registry 一致性+未误伤**：`TestVM_API_TRUTH_1_AccountNoopConsistency`：4 重分类 `StatusUnsupported`/`IsAPIImplemented=false`/`IsAPIUnsupported=true`；`AccountProfit`/`AccountCurrency`/`AccountCompany`/`AccountFreeMarginCheck`/`AccountBalance`/`AccountLeverage`/`AccountNumber` `IsAPIImplemented=true`。
- **验证**：`go test ./tools/mql2go/ -run TestVM_API_TRUTH_1` 全绿。

## 验收标准

- [ ] `go build ./...` 通过
- [ ] `go test ./tools/mql2go/` 全过（含 S6t/u/v）
- [ ] `cd backend && go run ./tools/check-file-lines --strict` 零错误
- [ ] `gofmt` / `go vet` 零警告
- [ ] `go test -race -count=3 ./tools/mql2go/` 通过
- [ ] 对抗证明：① `builtinAccountProfit` 改回 noopDecimal 接线 → S6u RED；② 注释 unsupportedSymbols 4 行 → S6t/S6v RED；③ `builtinAccountFreeMarginCheck` 删 cmd default 分支 → S6u cmd=9 变静默 → RED
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
6. **结束语**：`[施工完成:VM-API-TRUTH-1-批次2e] @<commit-hash>`（D-014，无此行 = 未交付，复审不启动）。

**停手等 Devin CLI 复审；不达标将由决策方开缺陷清单退回。勿部署，禁 `--no-verify`。**
