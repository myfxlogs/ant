# 施工派工单：VM-API-TRUTH-1（批次1：MQL5 order/deal/history）

> **角色纪律**：你是施工方，不是验收方。完成后保持 `🟦open（施工完成，待独立复审）`，由 Devin CLI 独立复审后才可变更为 `✅done`。禁止扩 scope、提交、push、部署。commit 用 `ANT_ROLE=builder` 前缀。

## 立项背景

**触发**：`docs/audits/tech-debt-registry.md:126` VM-API-TRUTH-1（P1，2026-08-24 全面 VM 审计）。372 个 builtin 中 352 个非 nil，但 MQL5 order/deal/history 等 handler 是固定值/空操作/close proxy；coverage 会把它们计为 implemented，策略因此可能基于假数据继续运行。**2026-09-16 Devin CLI 裁定**：批准 StatusUnsupported 重分类方向——无法忠实实现的 API 编译期拒绝优于运行时假数据（fail-closed 原则一致）；MQL5 handle/history/session/margin 能力边界按"显式限制"路线，不做 handle 子系统扩展（YAGNI，需求出现时另立）。

**本批范围**：MQL5 order/deal/history 22 个 API。`vm_builtin_mql5_trade.go:8-9` 文件头自述"stubs returning safe defaults — they require MQL5 broker integration that is not available in the backtest VM"。全 22 API 返回固定值（0/""/true/false），无真实 pending order/deal history 数据源。

**证据链**（HEAD `03b0dfe8` 实拍，2026-09-16 核实）：

| API名 | 文件:行 | 实际返回 |
|---|---|---|
| `OrderCalcMargin` | `vm_builtin_mql5_trade.go:11` | `true` |
| `OrderCalcProfit` | `vm_builtin_mql5_trade.go:15` | `true` |
| `OrderCheck` | `vm_builtin_mql5_trade.go:19` | `true` |
| `OrderGetTicket` | `vm_builtin_mql5_trade.go:29` | `0` |
| `OrderGetDouble` | `vm_builtin_mql5_trade.go:33` | `0` |
| `OrderGetInteger` | `vm_builtin_mql5_trade.go:37` | `0` |
| `OrderGetString` | `vm_builtin_mql5_trade.go:41` | `""` |
| `OrdersTotalMQL5` | `vm_builtin_mql5_trade.go:45` | `0` |
| `HistorySelect` | `vm_builtin_mql5_trade.go:50` | `true` |
| `HistorySelectByPosition` | `vm_builtin_mql5_trade.go:54` | `true` |
| `HistoryDealsTotal` | `vm_builtin_mql5_trade.go:58` | `0` |
| `HistoryDealSelect` | `vm_builtin_mql5_trade.go:62` | `false` |
| `HistoryDealGetTicket` | `vm_builtin_mql5_trade.go:66` | `0` |
| `HistoryDealGetDouble` | `vm_builtin_mql5_trade.go:70` | `0` |
| `HistoryDealGetInteger` | `vm_builtin_mql5_trade.go:74` | `0` |
| `HistoryDealGetString` | `vm_builtin_mql5_trade.go:78` | `""` |
| `HistoryOrdersTotal` | `vm_builtin_mql5_trade.go:82` | `0` |
| `HistoryOrderSelect` | `vm_builtin_mql5_trade.go:86` | `false` |
| `HistoryOrderGetTicket` | `vm_builtin_mql5_trade.go:90` | `0` |
| `HistoryOrderGetDouble` | `vm_builtin_mql5_trade.go:94` | `0` |
| `HistoryOrderGetInteger` | `vm_builtin_mql5_trade.go:98` | `0` |
| `HistoryOrderGetString` | `vm_builtin_mql5_trade.go:102` | `""` |

**依赖核查**：`grep -r` 全 backend 仅 mql2go 包内 4 文件引用（builtins.go/builtin_registry.go/vm_builtin_wiring.go/vm_builtin_mql5_trade.go），无测试/策略/其他包依赖——重分类安全。

**不变量**：本批只改 MQL5 order/deal/history 22 API。`PositionSelect`（:23-26）保留（委托 PositionSelectByTicket，有真实实现）。MQL4 `OrderSelect`/`OrderSend`/`OrderClose`/`OrdersTotal`/`OrderTicket` 等不在本批（MQL4 trade 有真实 broker 实现，后续批次另行审计）。

## 设计 SSOT 声明

- 设计文档：本派工单（唯一真相源）
- 相关契约：`AGENTS.md` §0 fail-closed 红线、§7.2 无死代码
- 裁定：`docs/audits/tech-debt-registry.md:126` 2026-09-16 Devin CLI 裁定（StatusUnsupported 重分类方向批准）

## 约束与目标

- **目标**：22 个 MQL5 order/deal/history API 从 implemented 重分类为 StatusUnsupported，编译期拒绝（不再返回假数据）。删除假实现函数 + 注册 + 绑定（无死代码）。
- **范围**：仅以下 5 文件：
  - `backend/tools/mql2go/interp/api_registry.go`（unsupportedSymbols 添加 22 + 新 reason 常量）
  - `backend/tools/mql2go/interp/builtin_registry.go`（implementedMQL5Position 移除 22）
  - `backend/tools/mql2go/builtins.go`（删除 22 个 nil 注册，保留 PositionSelect）
  - `backend/tools/mql2go/vm_builtin_wiring.go`（删除 22 个 fn 绑定）
  - `backend/tools/mql2go/vm_builtin_mql5_trade.go`（删除 22 个假实现函数，保留 PositionSelect）
- **测试**：`backend/tools/mql2go/vm_api_truth_test.go`（新文件，编译期拒绝 + registry 一致性 + mutation）
- **不碰**：MQL4 trade（OrderSelect/OrderSend/OrderClose/OrdersTotal/OrderTicket 等，有真实 broker 实现）、PositionSelect/PositionSelectByTicket、AccountInfo*（后续批次）、CopyBuffer/CopyRates（后续批次）、Symbol session/margin（后续批次）、platform checkup（后续批次）。

## 边界 / 不做

- 不改 MQL4 `OrderSelect`/`OrderSend`/`OrderClose`/`OrderModify`/`OrderDelete`/`OrdersTotal`/`OrderTicket`/`OrderProfit`/`OrderLots` 等（vm_builtin_trade.go，有真实 broker 实现）。
- 不改 `PositionSelect`（vm_builtin_mql5_trade.go:23-26，委托 PositionSelectByTicket，有真实实现）。
- 不改 AccountInfo*（vm_builtin_mql5_info.go，后续批次）。
- 不改 CopyBuffer/CopyRates/CopyTime（vm_builtin_mql5_ts.go，后续批次）。
- 不改 Symbol session/margin（vm_builtin_mql5_info.go，后续批次）。
- 不改 platform checkup（vm_builtin_checkup.go/vm_builtin_impls.go，后续批次）。
- 不引入 MQL5 handle/history 子系统（YAGNI，需求出现时另立）。
- 不改 `TestAllBuiltinsWired`/`TestNoDuplicateBuiltins`/`TestBuiltinCount`/`TestImplementedNamesHaveVMHandlers` 既有测试逻辑（这些测试会自动适应：从 implemented* 移除后不再校验，从 builtins.go 删除后不再计入 total）。
- `TestBuiltinCount` 阈值 `total < 300` 会失败（删 22 后 total 从 ~372 降到 ~350，仍 >300）——若失败需调整阈值，但预期不会失败。

## 施工指令

### S1 — unsupportedSymbols 添加 22 API + 新 reason 常量

- **目标**：在 `api_registry.go` 的 `unsupportedSymbols` 列表添加 22 个 MQL5 order/deal/history API，编译期拒绝。
- **坐标**：`backend/tools/mql2go/interp/api_registry.go:47-62`（reason 常量块）+ `:66-144`（unsupportedSymbols 列表，在 ResourceFree 后追加）。
- **落点**：
  - `:62` reason 常量块末尾追加 `reasonMQL5History = "MQL5 order/deal/history requires broker integration not available in the backtest VM"`。
  - `:144`（ResourceFree 行后）追加 22 行：
    ```go
    {Name: "OrderCalcMargin", Status: StatusUnsupported, Category: CatFunction, Reason: reasonMQL5History},
    {Name: "OrderCalcProfit", Status: StatusUnsupported, Category: CatFunction, Reason: reasonMQL5History},
    {Name: "OrderCheck", Status: StatusUnsupported, Category: CatFunction, Reason: reasonMQL5History},
    {Name: "OrderGetTicket", Status: StatusUnsupported, Category: CatFunction, Reason: reasonMQL5History},
    {Name: "OrderGetDouble", Status: StatusUnsupported, Category: CatFunction, Reason: reasonMQL5History},
    {Name: "OrderGetInteger", Status: StatusUnsupported, Category: CatFunction, Reason: reasonMQL5History},
    {Name: "OrderGetString", Status: StatusUnsupported, Category: CatFunction, Reason: reasonMQL5History},
    {Name: "OrdersTotalMQL5", Status: StatusUnsupported, Category: CatFunction, Reason: reasonMQL5History},
    {Name: "HistorySelect", Status: StatusUnsupported, Category: CatFunction, Reason: reasonMQL5History},
    {Name: "HistorySelectByPosition", Status: StatusUnsupported, Category: CatFunction, Reason: reasonMQL5History},
    {Name: "HistoryDealsTotal", Status: StatusUnsupported, Category: CatFunction, Reason: reasonMQL5History},
    {Name: "HistoryDealSelect", Status: StatusUnsupported, Category: CatFunction, Reason: reasonMQL5History},
    {Name: "HistoryDealGetTicket", Status: StatusUnsupported, Category: CatFunction, Reason: reasonMQL5History},
    {Name: "HistoryDealGetDouble", Status: StatusUnsupported, Category: CatFunction, Reason: reasonMQL5History},
    {Name: "HistoryDealGetInteger", Status: StatusUnsupported, Category: CatFunction, Reason: reasonMQL5History},
    {Name: "HistoryDealGetString", Status: StatusUnsupported, Category: CatFunction, Reason: reasonMQL5History},
    {Name: "HistoryOrdersTotal", Status: StatusUnsupported, Category: CatFunction, Reason: reasonMQL5History},
    {Name: "HistoryOrderSelect", Status: StatusUnsupported, Category: CatFunction, Reason: reasonMQL5History},
    {Name: "HistoryOrderGetTicket", Status: StatusUnsupported, Category: CatFunction, Reason: reasonMQL5History},
    {Name: "HistoryOrderGetDouble", Status: StatusUnsupported, Category: CatFunction, Reason: reasonMQL5History},
    {Name: "HistoryOrderGetInteger", Status: StatusUnsupported, Category: CatFunction, Reason: reasonMQL5History},
    {Name: "HistoryOrderGetString", Status: StatusUnsupported, Category: CatFunction, Reason: reasonMQL5History},
    ```
- **验证**：`go build ./tools/mql2go/` 通过；`LookupAPI("OrderCalcMargin")` 返回 `StatusUnsupported`。

### S2 — implementedMQL5Position 移除 22 API

- **目标**：从 `builtin_registry.go` 的 `implementedMQL5Position` 列表移除 22 个 API，使 `api_registry.go:init()` 不再把它们标为 StatusImplemented（unsupportedSymbols 先加，implemented 覆盖——移除 implemented 后 unsupported 生效）。
- **坐标**：`backend/tools/mql2go/interp/builtin_registry.go:128-138`（MQL5 trade helpers + order history + order functions 块）。
- **落点**：
  - `:128-129` 注释 + `OrderCalcMargin/OrderCalcProfit/OrderCheck` 删除（保留 `:130` PositionSelect）。
  - `:131-136` MQL5 order history 块整块删除（HistorySelect 到 HistoryOrderGetString）。
  - `:137-138` MQL5 order functions 块整块删除（OrderGetTicket 到 OrderGetString）。
  - 保留 `:130` `"PositionSelect",`。
- **验证**：`TestImplementedNamesHaveVMHandlers` 不再校验这 22 个名字（从列表移除后 check 循环不遍历它们）。

### S3 — builtins.go 删除 22 个 nil 注册

- **目标**：从 `builtins.go` 删除 22 个 API 的 nil 注册条目（保留 PositionSelect）。
- **坐标**：`backend/tools/mql2go/builtins.go:413-440`（MQL5 trade helpers + order functions + deal/history 块）。
- **落点**：
  - `:413` 注释 `// ── MQL5 trade helpers ───` 保留。
  - `:414-416` OrderCalcMargin/OrderCalcProfit/OrderCheck 删除。
  - `:417` `{"PositionSelect", nil},` 保留。
  - `:419` 注释 `// ── MQL5 order functions (pending orders) ───` 删除（整块删除）。
  - `:420-424` OrderGetTicket/OrderGetDouble/OrderGetInteger/OrderGetString/OrdersTotalMQL5 删除。
  - `:426` 注释 `// ── MQL5 deal/history functions ───` 删除（整块删除）。
  - `:427-440` HistorySelect 到 HistoryOrderGetString 删除（22 行）。
- **验证**：`TestNoDuplicateBuiltins` 通过（无重复）；`TestBuiltinCount` total 仍 >=300（~350）。

### S4 — vm_builtin_wiring.go 删除 22 个 fn 绑定

- **目标**：从 `vm_builtin_wiring.go` 删除 22 个 API 的 fn 绑定（保留 PositionSelect）。
- **坐标**：`backend/tools/mql2go/vm_builtin_wiring.go:180-207`（registerExtendedTrade + registerExtendedHistory）。
- **落点**：
  - `:181-183` OrderCalcMargin/OrderCalcProfit/OrderCheck 的 fn 绑定删除。
  - `:184` `builtinRegistry[id("PositionSelect")].fn = builtinPositionSelect` 保留。
  - `:185-189` OrderGetTicket/OrderGetDouble/OrderGetInteger/OrderGetString/OrdersTotalMQL5 的 fn 绑定删除。
  - `:192-207` `registerExtendedHistory` 整函数删除（全 14 行是 22 API 中的 14 个 history API 绑定）。
  - 若 `registerExtendedHistory` 删除后无调用方，检查 `init()` 或其他调用点，删除调用。
- **验证**：`TestAllBuiltinsWired` 通过（这 22 个 API 不在 builtinRegistry 里，不会被校验）。

### S5 — vm_builtin_mql5_trade.go 删除 22 个假实现函数

- **目标**：从 `vm_builtin_mql5_trade.go` 删除 22 个假实现函数（保留 PositionSelect）。
- **坐标**：`backend/tools/mql2go/vm_builtin_mql5_trade.go:11-21,29-104`（全 22 个函数）。
- **落点**：
  - `:7-9` 文件头注释更新为"PositionSelect only — MQL5 order/deal/history stubs removed (VM-API-TRUTH-1, now StatusUnsupported)"。
  - `:11-21` OrderCalcMargin/OrderCalcProfit/OrderCheck 删除。
  - `:23-26` builtinPositionSelect 保留。
  - `:28-104` 全部删除（OrderGetTicket 到 HistoryOrderGetString，含注释 :28/:49）。
  - 若删除后 import 未使用（`interp` 仍被 PositionSelect 用），保留 import。
- **验证**：`go build ./tools/mql2go/` 通过（无未使用函数/未使用 import）。

### S6 — 新增对抗测试（`vm_api_truth_test.go` 新文件）

- **目标**：证明 22 API 编译期拒绝 + registry 一致性 + mutation RED→GREEN。
- **坐标**：`backend/tools/mql2go/vm_api_truth_test.go`（新文件）。
- **落点**：
  - **S6a — 编译期拒绝测试**：`TestVM_API_TRUTH_1_MQL5HistoryRejected`：对每个 API 名，`CompileMQL` 含该 API 调用的源码应返回 error（或 compile 失败）。用表驱动：`unsupportedMQL5History = []string{22 个 API 名}`。每个 API 构造最小源码如 `void OnTick() { OrderCalcMargin(...); }`（参数数量按 API 签名，可用 0 参数或最小参数——若编译器对参数数量宽松则用空参，否则按 MQL5 签名）。断言 `CompileMQL` 返回 error 且 error 消息含 "unsupported" 或 API 名。
  - **S6b — registry 一致性测试**：`TestVM_API_TRUTH_1_RegistryConsistency`：对每个 API 名，`interp.LookupAPI(name)` 返回 `Status == StatusUnsupported` 且 `Reason == reasonMQL5History`（或非空 reason）。同时 `interp.IsAPIImplemented(name)` 返回 false，`interp.IsAPIUnsupported(name)` 返回 true。
  - **S6c — PositionSelect 未误伤测试**：`TestVM_API_TRUTH_1_PositionSelectStillImplemented`：`interp.LookupAPI("PositionSelect")` 返回 `Status == StatusImplemented`，`interp.IsAPIImplemented("PositionSelect")` 返回 true。证明重分类只影响 22 API，不误伤 PositionSelect。
  - **S6d — mutation 对抗**：`TestVM_API_TRUTH_1_MutationEvidence`：构造一个假"恢复 implemented"的 mutation 场景——注释 `api_registry.go` 的 22 个 unsupportedSymbols 条目（或从 unsupportedSymbols 移除），期望 S6a/S6b RED（API 不再被拒绝，registry 不再是 StatusUnsupported）。**简化**：mutation 证据在施工自审时手动做（注释 unsupportedSymbols 的 22 行 → S6a/S6b RED → 恢复 GREEN），不必写入测试文件（避免测试文件依赖被注释的生产代码）。测试文件只含 S6a/S6b/S6c。
- **验证**：`go test ./tools/mql2go/ -run TestVM_API_TRUTH_1` 全绿。

## 验收标准

- [ ] `go build ./...` 通过
- [ ] `go test ./tools/mql2go/` 全过（含现有测试 + 新增 S6a/b/c）
- [ ] `cd backend && go run ./tools/check-file-lines --strict` 零错误
- [ ] `gofmt` / `go vet` 零警告
- [ ] `go test -race -count=3 ./tools/mql2go/` 通过
- [ ] 对抗证明：注释 unsupportedSymbols 22 行 → S6a/S6b RED → 恢复 GREEN（附命令与输出）
- [ ] diff 通读无死代码 / TODO / 调试残留 / 范围外改动

## 施工完成自审（强制，D-012）

交付自报前必须完成并随报提交：
- [ ] 逐项重跑上方验收标准并贴真实输出
- [ ] 红队自审 diff 三问：更简等价方案 / 边界·nil·并发 / 逆向依赖或重复基础设施
- [ ] 自审发现的缺陷已修复至全绿（自报列出发现项+修复项）
- [ ] 无自审记录 = 复审直接退回

## 交付格式

自报：改动文件清单 + 每条验收项证据（命令+关键输出）+ 施工完成自审记录 + 遗留疑问。
**自报最后一行固定为** `[施工完成:VM-API-TRUTH-1] @<commit-hash>`（D-014：无此行 = 未交付，复审不启动）。
**停手等 Devin CLI 复审；不达标将由决策方开缺陷清单退回。勿部署，禁 `--no-verify`。**
