# 施工提示词：QS-1.2a `lastError` 三 builtin 真实实现

> **[角色:施工]** — 你是本任务的施工方 agent（见 `.devin/rules/dual-terminal-roles.md`）。
> 严格按 S1–S4 施工，不做决策；超出提示词范围 = 违规，停下转 `[转交决策]`。
> 完成后先过"施工完成自审"（D-012）修至全绿再交回；自报末行带 `[施工完成:QS-1.2a] @<commit-hash>`（D-014）。
> 勿部署、勿 push、禁 `--no-verify`；只显式 add 本任务文件；commit 用 `ANT_ROLE=builder git commit` 前缀（D-015）；**不更新任何交接层文件**。
>
> **最终决策：Devin CLI（[角色:决策终] 激活）**

## 立项背景（触发 + 证据链）

- registry `QS-1.2a`；spec `docs/spec/vm-pipeline-quality-stability-improvement-plan.md` v2 §3.4（设计 SSOT）；决策 D-009（QS-1.2 拆分：本任务只做三 builtin + `lastError` 字段，**不动 OrderSend fatal 语义**——FAILCLOSED-1 不变量保留）。
- 实拍：`GetLastError` → `builtinNoopInt` 恒返 0（`vm_builtin_impls.go:73`）；`ResetLastError` → `builtinNoop`（`:74`）；`SetUserError` → `builtinSetUserError` 返回 NoneVal 什么都不写（`vm_builtin_checkup.go:102-104`，接线于 `vm_builtin_wiring.go:141`）。
- **决策方预核实的关键事实**（施工时复核，若不符 `[转交决策]`）：
  a. `ERR_USER_ERROR_FIRST` 常量在 `interp/constants.go` **不存在**——MQL4 值为 65536，需新增 `"ERR_USER_ERROR_FIRST": IntVal(65536)`（错误码区 :377-404 末尾追加）。
  b. VM 无 `lastError` 字段；VM 实例跨事件复用（`lastIndicators` 按事件清，`fatalError` 跨事件驻留，`vm.go:23-64`）——`lastError` 属后者，**不按事件清零**。
  c. 取 int 参数惯例 `args[0].ToInt()`（`interp/value.go:111`）。
  d. `runEvent`/`runOnBar` 等事件入口无 lastError 相关逻辑——字段生命周期 = VM 实例生命周期，无需挂钩。

## 设计 SSOT 声明

- 设计文档：spec v2 §3.4。契约：AGENTS.md §1 fail-closed；决策 D-009/D-012/D-014/D-015。

## 约束与目标（决策方已定的语义裁定）

- `vm.lastError int32`：**跨事件驻留**，仅被 `GetLastError`（读后清零）/`ResetLastError`（清零）/`SetUserError`（写入）触碰；**本轮唯一写入点是 SetUserError**。
- `GetLastError()` → 返回 `vm.lastError` 当前值**并清零**（MQL4 读后复位语义），返回值类型 `interp.IntVal`。
- `ResetLastError()` → `vm.lastError = 0`，返回 `interp.NoneVal()`。
- `SetUserError(c)` → `vm.lastError = 65536 + int32(c)`（`ERR_USER_ERROR_FIRST + c`），返回 `interp.NoneVal()`。
- **交易 fatal 路径不动**：OrderSend 参数非法/broker 错误仍是 fatalError 停 VM（FAILCLOSED-1），不改为返 -1+lastError——QS-1.2b 不立项。

## 边界 / 不做

- 不在任何 fatal/broker 错误路径写 `lastError`；不动 `SetReturnError`（`:106-108`，MQL5 语义另行评估）；不动 `ExpertRemove`/`IsTesting` 等其它 no-op；不改 api_registry 状态（三符号已在 builtins 表接线）。

## 施工指令

### S1 — 事实复核（只读）

- **目标**：逐项复核立项背景 a–d 四条事实。
- **验证**：自报逐条给出"属实/不符 + 代码行证据"；任何不符 → `[转交决策]`。

### S2 — VM 字段 + 常量

- **坐标**：`backend/tools/mql2go/vm.go:39`（`fatalError` 字段附近）；`interp/constants.go:404` 错误码区末尾。
- **落点**：VM struct 加 `lastError int32`（带注释：MQL4 `_LastError` 语义——跨事件驻留，GetLastError 读后清零，不按事件重置）；constants.go 加 `"ERR_USER_ERROR_FIRST": IntVal(65536)`。
- **验证**：S4 测试。

### S3 — 三 builtin 实现 + 换接线

- **坐标**：`vm_builtin_checkup.go`（`builtinSetUserError` :102 同文件就近实现）；`vm_builtin_impls.go:73-74`。
- **落点**：
  a. `vm_builtin_checkup.go` 新增 `builtinGetLastError`（读 `vm.lastError` → 清零 → 返回 `interp.IntVal(old)`）与 `builtinResetLastError`（清零 → `interp.NoneVal()`）；`builtinSetUserError` 改为 `vm.lastError = 65536 + args[0].ToInt()`（`len(args)>0` 守卫，无参视为 0 或按既有 arg 缺失惯例处理——自报注明）。
  b. `vm_builtin_impls.go:73-74` 换接线：`builtinGetLastError` / `builtinResetLastError`。
- **验证**：S4 测试。

### S4 — 测试（先红后绿 + mutation）

- **坐标**：新建 `backend/tools/mql2go/vm_lasterror_test.go`（复用 CompileMQL4/CompilePython + RunOnBar + GetGlobal 模式，MQL4 为主语义方）。
- **落点**（最小集）：
  a. `SetUserError(5)` → `GetLastError()` = 65541（ERR_USER_ERROR_FIRST+5）。
  b. `GetLastError()` 读后清零：连续两次调用，第二次 = 0。
  c. `SetUserError(5)` → `ResetLastError()` → `GetLastError()` = 0。
  d. 跨事件驻留：event#1 `SetUserError(7)`；event#2 `GetLastError()` = 65543（不按事件清零）。
  e. 默认值：`GetLastError()` 未设置时 = 0。
  f. `ERR_USER_ERROR_FIRST` 常量在 MQL 源码中可用（编译 `x = ERR_USER_ERROR_FIRST` → 65536）。
- **验证**：a/b/c 先红后绿（修复前 a 恒 0 / b 恒 0 / c 恒 0）。

## 对抗证明（缺一即未完成）

- mutation：`builtinSetUserError` 不写 `vm.lastError`（恢复 no-op）→ S4a/S4d RED；restore → GREEN。
- mutation：`builtinGetLastError` 不清零 → S4b RED（第二次返回旧值）；restore → GREEN。

## 验收标准

- [ ] `cd backend && go build ./...` 通过
- [ ] `cd backend && go test -count=1 ./tools/mql2go/...` 全过
- [ ] `go test -race -count=3 ./tools/mql2go/...` 通过
- [ ] `go run ./tools/check-file-lines --strict` 本任务文件零新增警告
- [ ] `gofmt -l` / `go vet ./tools/mql2go/...` 零输出
- [ ] S1 四条事实复核 + 对抗证明 RED→restore→GREEN（附命令与输出）
- [ ] diff 无交接层文件、无范围外改动（特别：`OrderSend`/fatal 路径零触碰）

## 施工完成自审（强制，D-012）

交付自报前必须完成并随报提交：
- [ ] 逐项重跑上方验收标准并贴真实输出
- [ ] 红队自审 diff 三问：更简等价方案 / 边界·nil·并发 / 逆向依赖或重复基础设施
- [ ] 自审发现的缺陷已修复至全绿（自报列出发现项+修复项）
- [ ] 无自审记录 = 复审直接退回

## 交付格式

自报：改动文件清单 + 每条验收项证据（命令+关键输出）+ 施工完成自审记录 + S1 复核结果 + 遗留疑问。
**自报最后一行固定为** `[施工完成:QS-1.2a] @<commit-hash>`（D-014：无此行 = 未交付，复审不启动）。
**停手等 Devin CLI 复审；不达标将由决策方开缺陷清单退回。**

[施工完成:QS-1.2a] @<本任务最终commit>
