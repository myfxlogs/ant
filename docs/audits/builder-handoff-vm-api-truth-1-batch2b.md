# 施工派工单：VM-API-TRUTH-1（批次2b：Account 全假 + Symbol by-reference 未填充 5 API）

> **角色纪律**：你是施工方，不是验收方。完成后保持 `🟦open（施工完成，待独立复审）`，由 Devin CLI 独立复审后才可变更为 `✅done`。禁止扩 scope、提交、push、部署。commit 用 `ANT_ROLE=builder` 前缀。

## 立项背景

**触发**：`docs/audits/tech-debt-registry.md:126` VM-API-TRUTH-1（P1）。批次1 ✅done（commit e97a43b8，MQL5 order/deal/history 22 API）。批次2a ✅done（commit 8f946579，platform checkup 12 API）。本批 2b 继续 Account 全假 + Symbol by-reference 未填充 API 重分类。

**本批范围**：5 个 API，全为固定值/by-reference 未填充，无真实数据源。

**证据链**（HEAD `562940ac` 实拍，2026-09-16 核实）：

| API名 | 文件:行 | 实际返回 | 真实数据源 | 裁定 |
|---|---|---|---|---|
| `AccountStopoutMode` | `vm_builtin_mql5_info.go:112` | `0`（固定） | 无（回测无 stopout 模式概念） | 重分类（全假） |
| `AccountCredit` | `vm_builtin_mql5_info.go:116` | `0`（固定） | 无（回测无信用额度概念） | 重分类（全假） |
| `SymbolInfoMarginRate` | `vm_builtin_mql5_info.go:37` | `true`（不填充 by-reference 输出） | 无（回测无 margin rate 数据源） | 重分类（by-reference 未填充） |
| `SymbolInfoSessionQuote` | `vm_builtin_mql5_info.go:41` | `true`（不填充 by-reference 输出） | 无（回测无 quote session 数据源） | 重分类（by-reference 未填充） |
| `SymbolInfoSessionTrade` | `vm_builtin_mql5_info.go:45` | `true`（不填充 by-reference 输出） | 无（回测无 trade session 数据源） | 重分类（by-reference 未填充） |

**依赖核查**：`grep -r` 全 backend 仅 mql2go 包内 4 文件引用（builtins.go/builtin_registry.go/vm_builtin_wiring.go/vm_builtin_mql5_info.go）。无测试/策略/其他包依赖。

**不变量**：本批只改 5 API。`AccountInfoDouble`/`AccountInfoInteger`/`AccountInfoString`（vm_builtin_mql5_info.go:54-110）保留——混合实现（部分真实+部分假），属于批次2c（修复假分支，非重分类）。`SymbolSelect`/`SymbolsTotal`/`SymbolIsSynchronized`（vm_builtin_mql5_info.go:29-35,49-51）保留——回测单 symbol 场景下固定值是合理语义（SymbolSelect=true 表示选中 symbol 存在，SymbolsTotal=1 表示单 symbol，SymbolIsSynchronized=true 表示数据已同步），类似 IsTesting=true。`SymbolInfoTick`/`SymbolName`（vm_builtin_mql5_info.go:9-27）保留——有真实实现（读 vm.ctx.Bid()/Ask()/Symbol()）。

## 设计 SSOT 声明

- 设计文档：本派工单（唯一真相源）
- 相关契约：`AGENTS.md` §0 fail-closed 红线、§7.2 无死代码
- 裁定：`docs/audits/tech-debt-registry.md:126` 2026-09-16 Devin CLI 裁定（StatusUnsupported 重分类方向批准）

## 约束与目标

- **目标**：5 个 API 从 implemented 重分类为 StatusUnsupported，编译期拒绝。删除假实现函数 + 注册 + 绑定（无死代码）。
- **范围**：仅以下 5 文件：
  - `backend/tools/mql2go/interp/api_registry.go`（unsupportedSymbols 添加 5 + 新 reason 常量）
  - `backend/tools/mql2go/interp/builtin_registry.go`（implementedAccount/implementedMarketData 移除 5）
  - `backend/tools/mql2go/builtins.go`（删除 5 个 nil 注册）
  - `backend/tools/mql2go/vm_builtin_wiring.go`（删除 5 个 fn 绑定）
  - `backend/tools/mql2go/vm_builtin_mql5_info.go`（删除 5 个假实现函数）
- **测试**：`backend/tools/mql2go/vm_api_truth_test.go`（追加 S6g/h/i 测试到现有文件，不新建文件）
- **不碰**：AccountInfoDouble/Integer/String（批次2c）、SymbolSelect/SymbolsTotal/SymbolIsSynchronized（回测语义合理）、SymbolInfoTick/SymbolName（真实实现）、CopyBuffer/CopyRates（批次2d）。

## 边界 / 不做

- 不改 `AccountInfoDouble`/`AccountInfoInteger`/`AccountInfoString`（vm_builtin_mql5_info.go:54-110，混合实现——BALANCE/EQUITY/MARGIN/LEVERAGE 真实，CREDIT/TRADE_ALLOWED/TRADE_EXPERT/全 String 假，属批次2c 修复假分支）。
- 不改 `SymbolSelect`/`SymbolsTotal`/`SymbolIsSynchronized`（vm_builtin_mql5_info.go:29-35,49-51，回测单 symbol 场景固定值是合理语义）。
- 不改 `SymbolInfoTick`/`SymbolName`（vm_builtin_mql5_info.go:9-27，有真实实现读 vm.ctx）。
- 不改 `AccountBalance`/`AccountEquity`/`AccountMargin`/`AccountLeverage`（vm_builtin_account.go，有真实实现）。
- 不改 `analyze.go` `looksLikeMQLBuiltin` switch case（重分类后编译期拒绝，不进入运行时分析）。
- 不改既有测试逻辑。

## 施工指令

### S1 — unsupportedSymbols 添加 5 API + 新 reason 常量

- **目标**：在 `api_registry.go` 的 `unsupportedSymbols` 列表添加 5 个 API。
- **坐标**：`backend/tools/mql2go/interp/api_registry.go:47-64`（reason 常量块）+ `:185`（unsupportedSymbols 列表末尾，SetReturnError 行后）。
- **落点**：
  - `:64` reason 常量块末尾追加 `reasonAccountSymbolStub = "account/symbol stub functions return fixed values without authoritative data in the backtest VM"`。
  - `:185`（SetReturnError 行后）追加 5 行：
    ```go
    // VM-API-TRUTH-1 batch 2b: account/symbol stubs returning fixed values
    // or by-reference unfilled outputs. Reclassified StatusUnsupported.
    {Name: "AccountStopoutMode", Status: StatusUnsupported, Category: CatFunction, Reason: reasonAccountSymbolStub},
    {Name: "AccountCredit", Status: StatusUnsupported, Category: CatFunction, Reason: reasonAccountSymbolStub},
    {Name: "SymbolInfoMarginRate", Status: StatusUnsupported, Category: CatFunction, Reason: reasonAccountSymbolStub},
    {Name: "SymbolInfoSessionQuote", Status: StatusUnsupported, Category: CatFunction, Reason: reasonAccountSymbolStub},
    {Name: "SymbolInfoSessionTrade", Status: StatusUnsupported, Category: CatFunction, Reason: reasonAccountSymbolStub},
    ```
- **验证**：`go build ./tools/mql2go/` 通过；`LookupAPI("AccountStopoutMode")` 返回 `StatusUnsupported`。

### S2 — implemented* 列表移除 5 API

- **目标**：从 `builtin_registry.go` 的 `implementedAccount` 和 `implementedMarketData` 列表移除 5 个 API。
- **坐标**：`backend/tools/mql2go/interp/builtin_registry.go`（需先 grep 确认 AccountStopoutMode/AccountCredit 在 implementedAccount，SymbolInfoMarginRate/SymbolInfoSessionQuote/SymbolInfoSessionTrade 在 implementedMarketData）。
- **落点**：
  - `implementedAccount` 删 `"AccountStopoutMode", "AccountCredit",`（保留 AccountInfoDouble/Integer/String 属批次2c，保留 AccountBalance/Equity/Margin/Leverage 真实实现）。
  - `implementedMarketData` 删 `"SymbolInfoMarginRate", "SymbolInfoSessionQuote", "SymbolInfoSessionTrade",`（保留 SymbolInfoTick/SymbolName/SymbolSelect/SymbolsTotal/SymbolIsSynchronized）。
- **验证**：`TestImplementedNamesHaveVMHandlers` 不再校验这 5 个名字。

### S3 — builtins.go 删除 5 个 nil 注册

- **目标**：从 `builtins.go` 删除 5 个 API 的 nil 注册条目。
- **坐标**：`backend/tools/mql2go/builtins.go`（需先 grep 确认 AccountStopoutMode/AccountCredit 在 account 块，SymbolInfoMarginRate/SymbolInfoSessionQuote/SymbolInfoSessionTrade 在 market info 块）。
- **落点**：删除 5 行 nil 注册，保留 AccountInfoDouble/Integer/String + SymbolInfoTick/SymbolName/SymbolSelect/SymbolsTotal/SymbolIsSynchronized。
- **验证**：`TestNoDuplicateBuiltins` 通过；`TestBuiltinCount` total 仍 >=300。

### S4 — vm_builtin_wiring.go 删除 5 个 fn 绑定

- **目标**：从 `vm_builtin_wiring.go` 删除 5 个 API 的 fn 绑定。
- **坐标**：`backend/tools/mql2go/vm_builtin_wiring.go:161-166,179-180`（Symbol session/margin 块 + Account 块）。
- **落点**：
  - `:161-162` 保留 SymbolSelect/SymbolsTotal（回测语义合理）。
  - `:163-165` 删 SymbolInfoMarginRate/SymbolInfoSessionQuote/SymbolInfoSessionTrade。
  - `:166` 保留 SymbolIsSynchronized（回测语义合理）。
  - `:179-180` 删 AccountStopoutMode/AccountCredit。
- **验证**：`TestAllBuiltinsWired` 通过。

### S5 — vm_builtin_mql5_info.go 删除 5 个假实现函数

- **目标**：从 `vm_builtin_mql5_info.go` 删除 5 个假实现函数。
- **坐标**：`backend/tools/mql2go/vm_builtin_mql5_info.go:37-51,112-118`。
- **落点**：
  - `:37-51` 删 builtinSymbolInfoMarginRate/builtinSymbolInfoSessionQuote/builtinSymbolInfoSessionTrade/builtinSymbolIsSynchronized（保留 SymbolIsSynchronized :49-51——等等，SymbolIsSynchronized 保留，只删 MarginRate/SessionQuote/SessionTrade）。
  - **修正**：`:37-47` 删 builtinSymbolInfoMarginRate/builtinSymbolInfoSessionQuote/builtinSymbolInfoSessionTrade。保留 `:49-51` builtinSymbolIsSynchronized（回测语义合理）。
  - `:112-118` 删 builtinAccountStopoutMode/builtinAccountCredit。
  - 保留 `:9-35` builtinSymbolInfoTick/builtinSymbolName/builtinSymbolSelect/builtinSymbolsTotal（真实实现+回测语义合理）。
  - 保留 `:54-110` builtinAccountInfoDouble/Integer/String（批次2c）。
  - 若删除后 import 未使用（`interp` 仍被保留函数用），保留 import。
- **验证**：`go build ./tools/mql2go/` 通过（无未使用函数/未使用 import）。

### S6 — 追加对抗测试（`vm_api_truth_test.go` 现有文件）

- **目标**：追加批次2b 测试到现有 `vm_api_truth_test.go`，证明 5 API 编译期拒绝 + registry 一致性 + 真实实现未误伤。
- **坐标**：`backend/tools/mql2go/vm_api_truth_test.go`（现有文件，追加到末尾）。
- **落点**：
  - **S6g — 编译期拒绝测试**：`TestVM_API_TRUTH_1_AccountSymbolStubRejected`：表驱动 `unsupportedAccountSymbolStub = []string{5 个 API 名}`。每个 API 构造最小源码 `void OnTick() { API(); }`，断言 `CompileMQL` 返回 error 且 error 消息含 "unsupported" 或 API 名。复用 S6a/S6d 的断言模式。
  - **S6h — registry 一致性测试**：`TestVM_API_TRUTH_1_AccountSymbolStubRegistryConsistency`：对每个 API 名，`LookupAPI` 返回 `StatusUnsupported` + `Reason` 非空 + `IsAPIImplemented=false` + `IsAPIUnsupported=true`。复用 S6b/S6e 的断言模式。
  - **S6i — 真实实现未误伤测试**：`TestVM_API_TRUTH_1_AccountSymbolStubRealStillImplemented`：对 `AccountInfoDouble`/`AccountInfoInteger`/`AccountInfoString`（批次2c 保留混合实现）+ `SymbolSelect`/`SymbolsTotal`/`SymbolIsSynchronized`（回测语义合理）+ `SymbolInfoTick`/`SymbolName`（真实实现）+ `AccountBalance`/`AccountEquity`/`AccountMargin`/`AccountLeverage`（真实实现），`LookupAPI` 返回 `StatusImplemented` + `IsAPIImplemented=true`。证明重分类只影响 5 API，不误伤真实/合理实现。
- **验证**：`go test ./tools/mql2go/ -run TestVM_API_TRUTH_1` 全绿（含批次1 S6a/b/c + 批次2a S6d/e/f + 批次2b S6g/h/i）。

## 验收标准

- [ ] `go build ./...` 通过
- [ ] `go test ./tools/mql2go/` 全过（含现有测试 + 批次1 S6a/b/c + 批次2a S6d/e/f + 批次2b S6g/h/i）
- [ ] `cd backend && go run ./tools/check-file-lines --strict` 零错误
- [ ] `gofmt` / `go vet` 零警告
- [ ] `go test -race -count=3 ./tools/mql2go/` 通过
- [ ] 对抗证明：注释 unsupportedSymbols 5 行 → S6g/S6h RED → 恢复 GREEN（附命令与输出）
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
6. **结束语**：`[施工完成:VM-API-TRUTH-1-批次2b] @<commit-hash>`（D-014，无此行 = 未交付，复审不启动）。

**停手等 Devin CLI 复审；不达标将由决策方开缺陷清单退回。勿部署，禁 `--no-verify`。**
