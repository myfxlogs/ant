# 施工派工单：VM-FUNC-FATAL-DELAY-1

> **状态**：设计实查完成（Devin CLI，2026-09-18），待施工 → 独立复审
> **registry**：`docs/audits/tech-debt-registry.md` 行 218
> **前置**：VM-RUNTIME-FAILCLOSED-2 ✅done（本债即其独立复审分立项）；TRADE-BUILTIN-ERR-SWALLOW-1 ✅done（`6eae8160`）

---

## 1. 缺陷实证（Devin CLI 独立核实，坐标已复核未漂移）

`vm_execute.go` **仅两处** fetch-execute 循环：`runLoop:38`（:14 有 `fatalError` 顶部检查）与 `executeCallUser:385`（**缺**）。

`executeCallUser` 内层循环 `:375-411` 依序查：ctx 取消（:376）、OP_RETURN/OP_HALT break（:387）、MaxTicks（:391）、MaxStackDepth（:401）、`execute(ins2)` 返回 err（:406）——**独缺 `vm.fatalError`**。

后果链（实测语义）：
- 函数内 `x=10/0` → `arith` 经 `setStackError` 写 `fatalError` + 推 0 + 返回 nil → **后续指令照执行**：`OP_STORE_GLOBAL` 全局写泄漏、`OP_CALL_BUILTIN` 虽被 execute 内 :123-131 检查门控（防 broker 副作用）但纯栈/槽位指令全部放行 → 直到 OP_RETURN break → 回 runLoop 顶部才发现。
- **入口态更糟**（registry 未载）：`executeCallUser:356` `args := vm.popN(nArgs)`——popN 下溢也经 `setStackError`（`vm_helpers.go:56`）→ **带 fatal 入口函数体照执行**（嵌套调用时可达）。
- **调用方续跑泄漏**：`inner()` fault → inner OP_RETURN 净返回 nil → 调用方 `inner(); g=9` 的 `g=9` 照执行（fatalError 要回 runLoop 才拦）。

缓解面（registry 原载，仍成立）：事件级 fail-closed 不破——最终错误经 runLoop 传 Engine.Run；broker 副作用被 execute 内 OP_CALL_BUILTIN 检查门控。缺陷是**延迟窗口内的写入泄漏**。

## 2. 设计裁定

**D1 修复**：`executeCallUser` 内层循环**顶部**加 fatalError 检查——镜像 runLoop:14 序位（fatal→ctx→ticks→depth→fetch→execute），错误出口须恢复 `locals`/`callDepth`（同本函数其他出口 :379/:392/:403/:407 惯例）：

```go
for vm.pc < int32(len(vm.bc.Code)) {
    // Mirror runLoop's top-of-loop fatal check (ADR §5.4): a stack/slot/
    // arith fault inside a user function must stop the function now, not
    // leak writes until OP_RETURN.
    if vm.fatalError != "" {
        vm.locals = oldLocals
        vm.callDepth--
        return fmt.Errorf("VM fatal: %s", vm.fatalError)
    }
    if vm.ticks%10000 == 0 && vm.runCtx != nil {
    ...
```

此一处改动同时覆盖三条泄漏路径：函数内 fault 后续指令、嵌套调用入口 fatal（popN :356）、调用方 post-call 续跑（inner 错误经 execute() err 传播使外层不续跑）。

**D2 语义边界**（如实记录，勿动）：
- `OP_RETURN` 遇上 pending fatal → **error 优先于净返回**（fault 不得被 return 吞没）——顶部检查天然先于此。
- `OP_HALT` 在函数内=return（既有语义）不动。
- `vm.pc` 错误出口不恢复（与其他出口一致——error 对整个事件终结）。
- runLoop 侧行为不变（事件终结路径相同，只是拦截提前）。

**D3 依赖面**：零存量断言依赖"函数内 fault 后续执行"（FAILCLOSED-2 测试全是顶层代码；`vm_audit_2026_08_27_batch2_test.go` 查的是 MaxStackDepth/popN-builtin 不同检查）。断言通道 `getGlobalInt(t, vmRunner, "g_after")` 先例在 `vm_timeseries_failclosed_redo_test.go:438`。

## 3. 施工步骤

### S1：vm_execute.go executeCallUser 加检查

`:375` `for` 循环体首行插入 D1 代码块（fatalError 检查+locals/callDepth 恢复+`fmt.Errorf("VM fatal: %s", ...)`）。注释引用本债 ID。

### S2：新测试 `vm_func_fatal_delay_test.go`

编译/运行通道镜像 `vm_audit_test.go:26-56`：`CompileMQL(src)` → `vm.SetContext(&tsTestContext{bars: sdk.BarsToSlice(makeFailClosedBars(3))})` → `RunOnInit` → `RunOnBar` → `getGlobalInt(t, vmRunner, "...")`（同包 helper 直接复用，勿重写）。`10/0` 常量不折叠、运行时触发已实证。

```go
// (a) 函数内 fault 全局写不泄漏：
//   src := "int g_after=0; void f(){ int x=10/0; g_after=42; }\n" +
//          "int OnInit(){ return 0; } void OnBar(){ f(); }"
//   → RunOnBar err 非 nil（含 "division by zero"/"VM fatal"）+ g_after==0
//   修复前：g_after==42 泄漏。
// (b) 调用方续跑不泄漏：
//   "int g2=0; void inner(){ int x=1/0; } void outer(){ inner(); g2=9; }\n" +
//   "int OnInit(){ return 0; } void OnBar(){ outer(); }"
//   → err 非 nil + g2==0（修复前 inner 净返回 nil，g2=9 执行）。
// (c) 入口 fatal 边例（手工字节码，镜像 vm_audit_2026_08_27_batch2_test.go:58 风格）：
//   OP_CALL_USER nArgs=2 栈深 0 → popN :356 setStackError → 函数体首指令不执行
//   → err 含 "VM fatal"。
```

### S3：mutation 证据

1. 删/短路新检查 → (a)+(b) 用例 RED（g_after=42/g2=9 泄漏复活）
2. 检查改为 `break`（fault 被当净返回吞没）→ (a) `err==nil` 断言 RED + (c) RED
3. 检查移出循环放函数入口（只查一次）→ (a) 用例 RED（函数内 fault 仍漏）

## 4. 验收门

```bash
cd backend && go build ./...
go test ./tools/mql2go/
go test -race -count=3 ./tools/mql2go/
go vet ./tools/mql2go/
gofmt -l tools/mql2go/
go run ./tools/check-file-lines --strict
git diff --check
```

## 5. 文件/规模预估

| 文件 | 改动 |
|---|---|
| `tools/mql2go/vm_execute.go` | +7 行（循环顶检查） |
| `tools/mql2go/vm_func_fatal_delay_test.go` | 新 ~120 行 |

`vm_execute.go` 417→~424 行，远低于 check-lines 阈值。

## 6. 回报格式

`[施工完成:VM-FUNC-FATAL-DELAY-1] @<commit-hash>` + 机检五件套 + mutation 证据。勿部署、勿 push、禁 `--no-verify`。

---

## 修订记录（2026-09-18 Devin CLI 复审 R1）

**施工方 spec 缺陷报告核实**：mutation#2（break 形态）经独立实证**确无可观测 RED**——break 后各帧 loop-top 同型检查级联净返回至 runLoop，err 文本/g_after/g2 断言全部等价（仅上报层不同）。施工方如实上报而非伪造 RED，正确处置。S3#2 从验收项剔除。

**复审新发现（需返工）**：测试 (c) `TestEntryFatalStopsFunctionBody` 只断言 err 含 "VM fatal"——M1（摘除检查）下仍 PASS：popN fatal 后 body `OP_PUSH_CONST` 照执行（栈推 7）→ OP_RETURN 净返回 → runLoop 顶检查才拦，错误文本相同。**未观测"函数体不执行"**。

**返工项**：(c) 补判别断言 `len(vm.stack)==0`（修复后检查先于取指，栈空；变异后 body 推 7→栈=1）。一行改动使 (c) 独立判别入口 fatal 场景。
