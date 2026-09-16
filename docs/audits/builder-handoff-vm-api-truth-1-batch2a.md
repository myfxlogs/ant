# 施工派工单：VM-API-TRUTH-1（批次2a：platform checkup 12 API）

> **角色纪律**：你是施工方，不是验收方。完成后保持 `🟦open（施工完成，待独立复审）`，由 Devin CLI 独立复审后才可变更为 `✅done`。禁止扩 scope、提交、push、部署。commit 用 `ANT_ROLE=builder` 前缀。

## 立项背景

**触发**：`docs/audits/tech-debt-registry.md:126` VM-API-TRUTH-1（P1）。批次1 已 ✅done（commit e97a43b8，MQL5 order/deal/history 22 API）。本批 2a 继续 platform checkup 假实现重分类。

**本批范围**：platform checkup 12 个 API，全为固定值/空操作，无真实终端/平台数据源。`vm_builtin_checkup.go:9-10` 文件头自述"In backtest context, most of these return fixed values"。

**证据链**（HEAD `ba2b5a0e` 实拍，2026-09-16 核实）：

| API名 | 文件:行 | 实际返回 | 真实数据源 |
|---|---|---|---|
| `IsDllsAllowed` | `vm_builtin_checkup.go:34` | `true` | 无（回测无 DLL 权限概念） |
| `IsExpertEnabled` | `vm_builtin_checkup.go:38` | `true` | 无（回测无 Expert 启停概念） |
| `IsLibrariesAllowed` | `vm_builtin_checkup.go:42` | `true` | 无（回测无 library 权限概念） |
| `IsTradeContextBusy` | `vm_builtin_checkup.go:58` | `false` | 无（回测无 trade context 竞争） |
| `IsStopped` | `vm_builtin_checkup.go:62` | `false` | 无（回测无 stop 信号） |
| `UninitializeReason` | `vm_builtin_checkup.go:66` | `0` | 无（回测无 uninit 原因） |
| `MQLInfoInteger` | `vm_builtin_checkup.go:70` | `0` | 无（回测无 MQL 环境信息） |
| `MQLInfoString` | `vm_builtin_checkup.go:74` | `""` | 无（回测无 MQL 环境信息） |
| `TerminalInfoDouble` | `vm_builtin_checkup.go:78` | `0` | 无（回测无终端信息） |
| `TerminalInfoInteger` | `vm_builtin_checkup.go:82` | `0` | 无（回测无终端信息） |
| `TerminalInfoString` | `vm_builtin_checkup.go:86` | `""` | 无（回测无终端信息） |
| `SetReturnError` | `vm_builtin_checkup.go:121` | `NoneVal`（空操作） | 无（回测无终端 Return code） |

**依赖核查**：`grep -r` 全 backend 仅 mql2go 包内 4 文件引用（builtins.go/builtin_registry.go/vm_builtin_wiring.go/vm_builtin_checkup.go）+ analyze.go:269-273 `looksLikeMQLBuiltin` switch case（重分类后编译期拒绝，不进入运行时分析，安全）。无测试/策略/其他包依赖。

**不变量**：本批只改 platform checkup 12 API。`IsConnected`/`IsDemo`/`IsTradeAllowed`（vm_builtin_checkup.go:16-56）保留——VM-API-TRUTH-3 已修为读 `vm.ctx.Account()` 真实字段。`GetLastError`/`ResetLastError`/`SetUserError`/`CurTime`/`GetTickCount`/`GetTickCount64`/`GetMicrosecondCount` 保留（有真实实现）。`IsTesting`/`IsOptimization`/`IsVisualMode` 保留（回测语义合理——IsTesting=true/IsOptimization 视情况/IsVisualMode=false 是回测正确语义，非假数据）。

## 设计 SSOT 声明

- 设计文档：本派工单（唯一真相源）
- 相关契约：`AGENTS.md` §0 fail-closed 红线、§7.2 无死代码
- 裁定：`docs/audits/tech-debt-registry.md:126` 2026-09-16 Devin CLI 裁定（StatusUnsupported 重分类方向批准）

## 约束与目标

- **目标**：12 个 platform checkup API 从 implemented 重分类为 StatusUnsupported，编译期拒绝。删除假实现函数 + 注册 + 绑定（无死代码）。
- **范围**：仅以下 5 文件：
  - `backend/tools/mql2go/interp/api_registry.go`（unsupportedSymbols 添加 12 + 新 reason 常量）
  - `backend/tools/mql2go/interp/builtin_registry.go`（implementedPlatform 移除 12）
  - `backend/tools/mql2go/builtins.go`（删除 12 个 nil 注册）
  - `backend/tools/mql2go/vm_builtin_wiring.go`（删除 12 个 fn 绑定）
  - `backend/tools/mql2go/vm_builtin_checkup.go`（删除 12 个假实现函数）
- **测试**：`backend/tools/mql2go/vm_api_truth_test.go`（追加 S6d/e/f 测试到现有文件，不新建文件）
- **不碰**：IsConnected/IsDemo/IsTradeAllowed（VM-API-TRUTH-3 已修真实）、GetLastError/ResetLastError/SetUserError/CurTime/GetTickCount*（有真实实现）、IsTesting/IsOptimization/IsVisualMode（回测语义合理）、AccountInfo*（批次2b）、CopyBuffer/CopyRates（批次2c）、Symbol session/margin（批次2d）。

## 边界 / 不做

- 不改 `IsConnected`/`IsDemo`/`IsTradeAllowed`（vm_builtin_checkup.go:16-56，VM-API-TRUTH-3 已修读 `vm.ctx.Account()` 真实字段）。
- 不改 `GetLastError`/`ResetLastError`/`SetUserError`（vm_builtin_checkup.go:104-119，有真实 lastError 状态机）。
- 不改 `CurTime`/`GetTickCount`/`GetTickCount64`/`GetMicrosecondCount`（vm_builtin_checkup.go:90-99/125-127，有真实时间源）。
- 不改 `IsTesting`/`IsOptimization`/`IsVisualMode`（vm_builtin_impls.go:76-78，回测语义合理——IsTesting=true 表示在回测中是正确语义，非假数据）。
- 不改 `analyze.go:269-273` `looksLikeMQLBuiltin` switch case（重分类后编译期拒绝，这些名字不再进入运行时分析；保留 case 不影响——死分支但不删以避免 analyze.go 范围外改动）。
- 不改 `TestAllBuiltinsWired`/`TestNoDuplicateBuiltins`/`TestBuiltinCount`/`TestImplementedNamesHaveVMHandlers` 既有测试逻辑。

## 施工指令

### S1 — unsupportedSymbols 添加 12 API + 新 reason 常量

- **目标**：在 `api_registry.go` 的 `unsupportedSymbols` 列表添加 12 个 platform checkup API。
- **坐标**：`backend/tools/mql2go/interp/api_registry.go:47-63`（reason 常量块）+ `:170`（unsupportedSymbols 列表末尾，HistoryOrderGetString 行后）。
- **落点**：
  - `:63` reason 常量块末尾追加 `reasonPlatformCheckup = "platform/terminal checkup functions require a live terminal not available in the backtest VM"`。
  - `:170`（HistoryOrderGetString 行后）追加 12 行：
    ```go
    // VM-API-TRUTH-1 batch 2a: platform checkup stubs returning fixed values
    // (vm_builtin_checkup.go:9-10 file header). Reclassified StatusUnsupported.
    {Name: "IsDllsAllowed", Status: StatusUnsupported, Category: CatFunction, Reason: reasonPlatformCheckup},
    {Name: "IsExpertEnabled", Status: StatusUnsupported, Category: CatFunction, Reason: reasonPlatformCheckup},
    {Name: "IsLibrariesAllowed", Status: StatusUnsupported, Category: CatFunction, Reason: reasonPlatformCheckup},
    {Name: "IsTradeContextBusy", Status: StatusUnsupported, Category: CatFunction, Reason: reasonPlatformCheckup},
    {Name: "IsStopped", Status: StatusUnsupported, Category: CatFunction, Reason: reasonPlatformCheckup},
    {Name: "UninitializeReason", Status: StatusUnsupported, Category: CatFunction, Reason: reasonPlatformCheckup},
    {Name: "MQLInfoInteger", Status: StatusUnsupported, Category: CatFunction, Reason: reasonPlatformCheckup},
    {Name: "MQLInfoString", Status: StatusUnsupported, Category: CatFunction, Reason: reasonPlatformCheckup},
    {Name: "TerminalInfoDouble", Status: StatusUnsupported, Category: CatFunction, Reason: reasonPlatformCheckup},
    {Name: "TerminalInfoInteger", Status: StatusUnsupported, Category: CatFunction, Reason: reasonPlatformCheckup},
    {Name: "TerminalInfoString", Status: StatusUnsupported, Category: CatFunction, Reason: reasonPlatformCheckup},
    {Name: "SetReturnError", Status: StatusUnsupported, Category: CatFunction, Reason: reasonPlatformCheckup},
    ```
- **验证**：`go build ./tools/mql2go/` 通过；`LookupAPI("IsDllsAllowed")` 返回 `StatusUnsupported`。

### S2 — implementedPlatform 移除 12 API

- **目标**：从 `builtin_registry.go` 的 `implementedPlatform` 列表移除 12 个 API。
- **坐标**：`backend/tools/mql2go/interp/builtin_registry.go:109-115`（checkup 块）。
- **落点**：
  - `:109` 保留 `"IsConnected", "IsDemo",`（VM-API-TRUTH-3 已修真实）。
  - `:109` 删 `"IsDllsAllowed", "IsExpertEnabled",`。
  - `:110` 删 `"IsLibrariesAllowed",`，保留 `"IsTradeAllowed",`（VM-API-TRUTH-3 已修真实）。
  - `:110` 删 `"IsTradeContextBusy",`。
  - `:111` 删整行 `"IsStopped", "UninitializeReason",`。
  - `:112` 删整行 `"MQLInfoInteger", "MQLInfoString",`。
  - `:113` 删整行 `"TerminalInfoDouble", "TerminalInfoInteger", "TerminalInfoString",`。
  - `:115` 删 `"SetReturnError",`，保留 `"SetUserError",` 和 `"CurTime",`。
- **验证**：`TestImplementedNamesHaveVMHandlers` 不再校验这 12 个名字。

### S3 — builtins.go 删除 12 个 nil 注册

- **目标**：从 `builtins.go` 删除 12 个 API 的 nil 注册条目。
- **坐标**：`backend/tools/mql2go/builtins.go:362-378`（checkup 块）。
- **落点**：
  - `:362-364` 删 IsDllsAllowed/IsExpertEnabled/IsLibrariesAllowed。
  - `:365` 保留 `{"IsTradeAllowed", nil},`（VM-API-TRUTH-3 已修真实）。
  - `:366` 删 IsTradeContextBusy。
  - `:367` 删 IsStopped。
  - `:368` 删 UninitializeReason。
  - `:369-373` 删 MQLInfoInteger/MQLInfoString/TerminalInfoDouble/TerminalInfoInteger/TerminalInfoString。
  - `:378` 删 SetReturnError（保留 SetUserError :377）。
- **验证**：`TestNoDuplicateBuiltins` 通过；`TestBuiltinCount` total 仍 >=300。

### S4 — vm_builtin_wiring.go 删除 12 个 fn 绑定

- **目标**：从 `vm_builtin_wiring.go` 删除 12 个 API 的 fn 绑定。
- **坐标**：`backend/tools/mql2go/vm_builtin_wiring.go:125-141`（checkup 绑定块）。
- **落点**：
  - `:125-127` 删 IsDllsAllowed/IsExpertEnabled/IsLibrariesAllowed 的 fn 绑定。
  - `:128` 保留 `builtinRegistry[id("IsTradeAllowed")].fn = builtinIsTradeAllowed`（VM-API-TRUTH-3 已修真实）。
  - `:129-130` 删 IsTradeContextBusy/IsStopped。
  - `:131` 删 UninitializeReason。
  - `:132-136` 删 MQLInfoInteger/MQLInfoString/TerminalInfoDouble/TerminalInfoInteger/TerminalInfoString。
  - `:141` 删 SetReturnError（保留 SetUserError :140）。
- **验证**：`TestAllBuiltinsWired` 通过。

### S5 — vm_builtin_checkup.go 删除 12 个假实现函数

- **目标**：从 `vm_builtin_checkup.go` 删除 12 个假实现函数。
- **坐标**：`backend/tools/mql2go/vm_builtin_checkup.go:34-44,58-88,121-123`。
- **落点**：
  - `:9-10` 文件头注释更新为"Platform checkup — IsConnected/IsDemo/IsTradeAllowed/GetLastError/ResetLastError/SetUserError/CurTime/GetTickCount* only. VM-API-TRUTH-1 batch 2a: 12 fixed-value stubs removed (reclassified StatusUnsupported)"。
  - `:34-44` 删 builtinIsDllsAllowed/builtinIsExpertEnabled/builtinIsLibrariesAllowed。
  - `:58-88` 删 builtinIsTradeContextBusy/builtinIsStopped/builtinUninitializeReason/builtinMQLInfoInteger/builtinMQLInfoString/builtinTerminalInfoDouble/builtinTerminalInfoInteger/builtinTerminalInfoString。
  - `:121-123` 删 builtinSetReturnError。
  - 保留 `:16-32` builtinIsConnected/builtinIsDemo（VM-API-TRUTH-3 已修真实）。
  - 保留 `:46-56` builtinIsTradeAllowed（VM-API-TRUTH-3 已修真实）。
  - 保留 `:90-127` builtinGetTickCount/builtinGetTickCount64/builtinGetMicrosecondCount/builtinGetLastError/builtinResetLastError/builtinSetUserError/builtinCurTime。
  - 若删除后 import 未使用（`time` 仍被 GetTickCount 用，`interp` 仍被保留函数用），保留 import。
- **验证**：`go build ./tools/mql2go/` 通过（无未使用函数/未使用 import）。

### S6 — 追加对抗测试（`vm_api_truth_test.go` 现有文件）

- **目标**：追加批次2a 测试到现有 `vm_api_truth_test.go`，证明 12 API 编译期拒绝 + registry 一致性 + 真实实现未误伤。
- **坐标**：`backend/tools/mql2go/vm_api_truth_test.go`（现有文件，追加到末尾）。
- **落点**：
  - **S6d — 编译期拒绝测试**：`TestVM_API_TRUTH_1_PlatformCheckupRejected`：表驱动 `unsupportedPlatformCheckup = []string{12 个 API 名}`。每个 API 构造最小源码 `void OnTick() { API(); }`，断言 `CompileMQL` 返回 error 且 error 消息含 "unsupported" 或 API 名。复用 S6a 的断言模式。
  - **S6e — registry 一致性测试**：`TestVM_API_TRUTH_1_PlatformCheckupRegistryConsistency`：对每个 API 名，`LookupAPI` 返回 `StatusUnsupported` + `Reason` 非空 + `IsAPIImplemented=false` + `IsAPIUnsupported=true`。复用 S6b 的断言模式。
  - **S6f — 真实实现未误伤测试**：`TestVM_API_TRUTH_1_PlatformCheckupRealStillImplemented`：对 `IsConnected`/`IsDemo`/`IsTradeAllowed`/`GetLastError`/`ResetLastError`/`SetUserError`/`CurTime`/`GetTickCount`/`GetTickCount64`/`GetMicrosecondCount`/`IsTesting`/`IsOptimization`/`IsVisualMode`，`LookupAPI` 返回 `StatusImplemented` + `IsAPIImplemented=true`。证明重分类只影响 12 API，不误伤真实实现。
- **验证**：`go test ./tools/mql2go/ -run TestVM_API_TRUTH_1` 全绿（含批次1 S6a/b/c + 批次2a S6d/e/f）。

## 验收标准

- [ ] `go build ./...` 通过
- [ ] `go test ./tools/mql2go/` 全过（含现有测试 + 批次1 S6a/b/c + 批次2a S6d/e/f）
- [ ] `cd backend && go run ./tools/check-file-lines --strict` 零错误
- [ ] `gofmt` / `go vet` 零警告
- [ ] `go test -race -count=3 ./tools/mql2go/` 通过
- [ ] 对抗证明：注释 unsupportedSymbols 12 行 → S6d/S6e RED → 恢复 GREEN（附命令与输出）
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
6. **结束语**：`[施工完成:VM-API-TRUTH-1-批次2a] @<commit-hash>`（D-014，无此行 = 未交付，复审不启动）。

**停手等 Devin CLI 复审；不达标将由决策方开缺陷清单退回。勿部署，禁 `--no-verify`。**
