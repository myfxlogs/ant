# 施工派工单：VM-ARRAY-OOB-FAILCLOSED-1（用户数组越界静默 → fail-closed）

> **角色纪律**：你是施工方，不是验收方。完成后保持 `🟦open（施工完成，待独立复审）`，由 Devin CLI 独立复审后才可变更为 `✅done`。禁止扩 scope、提交、push、部署。commit 用 `ANT_ROLE=builder` 前缀。

## 立项背景

**触发**：`docs/audits/tech-debt-registry.md:217` VM-ARRAY-OOB-FAILCLOSED-1（P2，VM-RUNTIME-FAILCLOSED-2 独立复审分立）。真实 MQL `array out of range` 是致命运行时错误，VM 却静默返回假值/丢写。

**本批范围**：2 个缺陷面——A（registry 原文：全局数组静默 OOB）+ B（设计实查新发现：局部数组下标错读/写全局槽）。

**证据链**（HEAD 实拍，2026-09-17 Devin CLI 设计复审）：

### A. 全局数组静默 OOB（registry 原文）

| 站点 | 文件:行 | 现状 | 裁定 |
|---|---|---|---|
| `executePushArray` | `vm_execute.go:298-309` | `ins.A >= len(globals)`（槽越界）/ `Kind != ValArray`（非数组）/ `i<0 \|\| i>=len`（索引越界）→ 静默 `NoneVal` 假值入栈 | 三分支均 `setStackError` |
| `executeStoreArray` | `vm_execute.go:311-323` | 同三类条件 → 静默丢写 | 三分支均 `setStackError` |

真实 MQL5 动态下标运行时可达：`compile_expr.go:144,156` 对 `arr[i]` 直接 emit `OP_*_ARRAY`，索引是运行时值——与 FAILCLOSED-2 修复的 slot 越界同类。

### B. 局部数组下标错读/写全局槽（新发现，比 registry 记的更深）

`compile_expr.go:140-156` `compileSubscript`：非全局变量记录 blind spot（"local array read/write"）后**仍 emit `OP_*_ARRAY`，slot 是局部索引**；`resolveVar`（`compile.go:257-267`）对局部变量返回 `(localSlotID, false)`。运行时 `executePushArray`/`executeStoreArray` 拿 `ins.A` 直接索引 `vm.globals[ins.A]`——**局部索引被当全局索引用**：

- `globals[localIdx]` 非数组 → 读侧 NoneVal/写侧丢写（当前静默）
- `globals[localIdx]` 恰为数组 → **静默读写一个无关全局数组**——真实正确性破坏，不是 blind spot 语义内的"降级"

**裁定**：局部数组访问编译期契约不变（blind spot 仍记录），运行时必须 fail-closed 而非误操作错误槽。机制：非全局时 `emit` 负编码 `-(slot)-1` → 运行时 `ins.A < 0` → `setStackError("local array access: <op> not supported")`。无新 opcode；全局路径 `ins.A >= 0` 行为不变。

## 设计 SSOT 声明

- 设计文档：本派工单（唯一真相源）
- 相关契约：`AGENTS.md` §0 fail-closed 红线、§7.2 无死代码
- 裁定：`docs/audits/tech-debt-registry.md:217` + 本单 B 面裁定（Devin CLI 设计复审 2026-09-17）

## 约束与目标

- **目标**：OP_PUSH_ARRAY/OP_STORE_ARRAY 全部静默分支 → `setStackError`（经 `runLoop` 顶检查 → `OnBar`/`Engine.Run` fail-closed）；局部数组下标 → 诚实 error 不再误访问错误全局槽。
- **范围**：仅以下 3 文件：
  - `backend/tools/mql2go/vm_execute.go`（executePushArray/executeStoreArray + dispatch）
  - `backend/tools/mql2go/compile_expr.go`（compileSubscript 非全局负编码）
  - 测试文件（新增 `vm_array_oob_test.go` 或追加既有文件——文件预算内自选）
- **error 消息格式**（统一）：
  - `OP_PUSH_ARRAY slot %d out of range (globals=%d)`
  - `OP_PUSH_ARRAY slot %d is not an array`
  - `OP_PUSH_ARRAY index %d out of range (len=%d)`
  - `OP_STORE_ARRAY ...`（同三式换 OP_STORE_ARRAY）
  - `local array %s: %s not supported`（局部，name 不可用时用 slot 号——`compileSubscript` 侧把 `e.Name` 编入负编码不可行时，消息含 slot 编号即可；blind spot 已记名字）

## 边界 / 不做

- **不改** OP_PUSH_SERIES（series 下标，`getSeries` 路径）——独立语义不在本批。
- **不改** OP_GET_FIELD/OP_SET_FIELD。
- **不改** `resolveVar`/`localScopes` 结构、python 数组路径（python 语义另行审计）。
- **不改** compile 契约：局部数组仍编译通过 + blind spot 仍记录（覆盖率的已知限制声明不变，只是运行时行为从"误访问"变"诚实 error"）。
- **新 opcode 不引入**：负编码复用 `ins.A`。
- **坐标漂移**：行号以符号锚点为准。

## 施工指令

### S1 — compileSubscript 非全局负编码

- **坐标**：`backend/tools/mql2go/compile_expr.go:140-156`（write 分支 `:140-146` + read 分支 `:152-157`）。
- **落点**：`!isGlobal` 时 emit 的 slot 参数从 `int32(slot)` 改为 `-int32(slot)-1`（blind spot 记录行不动）。两处对称修改。
- **验证**：局部数组源码 `void OnTick(){ int a[3]; a[0]=1; }` 编译仍通过 + `Coverage` 含 local array blind spot。

### S2 — executePushArray 三分支 setStackError + 局部负编码分支

- **坐标**：`backend/tools/mql2go/vm_execute.go:298-309`。
- **落点**：
  ```go
  func (vm *VM) executePushArray(ins Instruction, idx interp.Value) interp.Value {
      if ins.A < 0 {
          vm.setStackError(fmt.Sprintf("OP_PUSH_ARRAY local array slot %d not supported", -ins.A-1))
          return interp.NoneVal()
      }
      if int(ins.A) >= len(vm.globals) {
          vm.setStackError(fmt.Sprintf("OP_PUSH_ARRAY slot %d out of range (globals=%d)", ins.A, len(vm.globals)))
          return interp.NoneVal()
      }
      arrVal := vm.globals[ins.A]
      if arrVal.Kind != interp.ValArray {
          vm.setStackError(fmt.Sprintf("OP_PUSH_ARRAY slot %d is not an array", ins.A))
          return interp.NoneVal()
      }
      i := int(idx.ToInt())
      if i < 0 || i >= len(arrVal.Array) {
          vm.setStackError(fmt.Sprintf("OP_PUSH_ARRAY index %d out of range (len=%d)", i, len(arrVal.Array)))
          return interp.NoneVal()
      }
      return arrVal.Array[i]
  }
  ```
  dispatch 处（`:160-162`）push 加 fatalError 守卫（对称 OP_CALL_BUILTIN `:128-131` 防御形态）：
  ```go
  case OP_PUSH_ARRAY:
      idx := vm.pop()
      v := vm.executePushArray(ins, idx)
      if vm.fatalError == "" {
          vm.push(v)
      }
  ```
- **验证**：全局 `int g[2]; g[5]` → OnBar error 含 "index 5 out of range"；`g[-1]` 同。

### S3 — executeStoreArray 三分支 setStackError + 局部负编码分支

- **坐标**：`backend/tools/mql2go/vm_execute.go:311-323`。
- **落点**：同 S2 形态——`ins.A < 0` → setStackError；槽越界/非数组/索引越界 → 各 setStackError（消息换 OP_STORE_ARRAY）。值已弹出无需回栈。
- **验证**：`g[9]=1` → OnBar error 含 "index 9 out of range"；`int scalar; scalar[0]=1` → error "is not an array"。

### S4 — 行为测试

- **坐标**：新建 `backend/tools/mql2go/vm_array_oob_test.go`（或追加既有 vm_*_test.go，文件预算内）。
- **落点**（每例真实 MQL→CompileMQL→VMRunner→ctx→OnBar/OnInit 断言）：
  1. `TestGlobalArrayReadOOB`：`int g[2]; int OnInit(){ g[5]=1; return 0; }` 写 OOB → error；`x=g[9]` 读 OOB → error 含 "index 9 out of range (len=2)"。
  2. `TestGlobalArrayNegativeIndex`：`g[-1]` → error。
  3. `TestNonArraySubscript`：`int s; s[0]` → error "is not an array"。
  4. `TestArraySlotBeyondGlobals`：槽越界（构造 ins.A >= globals 的边界——可用局部变量耗尽 globals 数后越界下标不可达，改用手工 bytecode 或直接调 executePushArray 单测槽越界分支）。
  5. `TestLocalArrayAccessFails`：`void f(){ int a[3]; a[0]=1; }` → OnBar error 含 "local array" + **不误写 globals**（断言同索引全局数组值未变）。
  6. `TestInBoundsStillWorks`：`g[0]/g[1]` 正常读写不回归（防误伤合法下标）。
- **验证**：`go test ./tools/mql2go/ -run 'TestGlobalArray|TestNonArray|TestLocalArray|TestInBounds'` 全绿。

## 验收标准

- [ ] `go build ./...` 通过
- [ ] `go test ./tools/mql2go/` 全过（含 S4 新增）
- [ ] `cd backend && go run ./tools/check-file-lines --strict` 零错误
- [ ] `gofmt` / `go vet` 零警告
- [ ] `go test -race -count=3 ./tools/mql2go/` 通过
- [ ] **对抗证明**（逐项 mutation RED→restore→GREEN）：
  1. `executePushArray` 索引越界分支恢复静默 `return interp.NoneVal()`（删 setStackError）→ S4-1 RED
  2. `executeStoreArray` 非数组分支恢复静默 `return` → S4-3 RED
  3. `compile_expr.go` 负编码恢复 `int32(slot)` → S4-5 RED（重新误写全局）——**须断言同索引全局数组未被写**，证明是"误写修复"而非"error 文案"
  4. dispatch fatalError 守卫删除（恢复无条件 push）→ 设计自选断言验证（如入栈假值可被后续指令读）
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
6. **结束语**：`[施工完成:VM-ARRAY-OOB-FAILCLOSED-1] @<commit-hash>`（D-014，无此行 = 未交付，复审不启动）。

**停手等 Devin CLI 复审；不达标将由决策方开缺陷清单退回。勿部署，禁 `--no-verify`。**

---

## 修订记录（2026-09-17 Devin CLI 决策终裁定：降范围施工）

**起因**：施工方 S1-S3 完成后 S4 阻断上报——测试 1/2/6 前提为假。Devin CLI 独立实证（探针编译 `double g_arr[3]` → `ir.Globals` 无该条、字节码 `STORE_GLOBAL` 替代 `OP_STORE_ARRAY`）确认两个**前端层新缺陷**：

- **发现 1**：`collectGlobalVar`（`compile_interp.go:174-223`）无 `array_declarator` 分支 → 全局数组声明整体丢弃（无 IsArray 槽，`vm.go:243` 的 ValArray 初始化永不触发）。
- **发现 2**：`compileAssignment`（`compile_interp_expr.go:286`）先查 `findIdent(lhs)` 后查 `subscript_expression` → `arr[i]=v` 恒退化为整槽标量写，line 304 分支不可达。

两发现另立新债 **VM-GLOBAL-ARRAY-DECL-1**（前端层，本债范围外）；option 3 的波及面审计并入新债设计阶段。

**S4 修订（行为断言不变，可达路径修正）**：

- **主路径 = 手工构造**（确定性强、覆盖全分支）：直接构造 VM + `vm.globals[i]=ValArray` + 手工 `Instruction{Op:OP_PUSH_ARRAY/OP_STORE_ARRAY}`，断言槽越界/非数组/索引越界/负槽/界内成功/dispatch 守卫全分支。
- **源码可达路径**（仅以下形态）：
  a. builtin 产数组入全局（`g = StringSplit("a,b",",")` → `g[9]` 读 → "index 9 out of range" error；`g[-1]` → error）——覆盖原 S4-1/2 的 OOB 语义。
  b. `int s; s[0]` → "is not an array" error（原 S4-3 可达）。
  c. 局部标量下标（`void f(){int a;a[0];}` 形态——局部变量下标引用）→ "local array" error + **断言同索引 globals 未被误写**（原 S4-5 语义保留；注意局部数组声明本身是编译错，用局部**标量**下标触发负编码路径）。
- **删除/迁移**：原 S4-1/2/6 的 `int g[2]` 全局声明形态 → 移入 VM-GLOBAL-ARRAY-DECL-1 验收（该债修复后全局数组才可达）。
- mutation 项 1/2/3/4 不变（均可由手工构造路径驱动 RED）。
