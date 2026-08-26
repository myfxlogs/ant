# Builder Handoff — VM-CACHE-INTEGRITY-1/2 返工（第一批复审退回项）

> 日期：2026-08-26 ｜ 审计方：Devin CLI（项目第一负责人） ｜ 施工方：Devin IDE
> 设计 SSOT：`docs/spec/vm-revert-redo-spec.md`（不变）
> 原施工提示词：`docs/audits/builder-handoff-vm-revert-redo-2026-08-26.md`（第一批）
> 复审退回记录：本文件 §0 证据链

## 0. 立项背景

**触发**：VM-CACHE-INTEGRITY-1/2 第一批复审 conditional pass，2 项退回。

**证据链**：
- 复审 S5 对抗证明：突变 `return nil, nil, fmt.Errorf(...)` → `return r, nil, nil`（恢复吞 error 旧逻辑）→ `TestMarshalErrorNotSwallowed` **仍 GREEN** → 对抗证明不达标
- 根因：`TestMarshalErrorNotSwallowed` 只验证了 `MarshalBytecode(nil)` 返回 error（nil 检查路径）和成功路径返回非 nil bytecode，**没有构造 marshal 失败场景验证 `CompileMQLCached` 返回 error**
- 复审 C 洁净：`vm_cache_integrity_redo_test.go:317` 的 `var _ = binary.LittleEndian` 是 unnecessary（`binary` 包未在测试中实际使用）

**设计 SSOT 声明**：`docs/spec/vm-revert-redo-spec.md`（D1-D4 不变）

**约束与目标**：
- 只修复 2 项退回，不重做已通过的 S1-S4/S6-S10
- S5 对抗证明必须构造真实的 marshal 失败场景，验证 `CompileMQLCached` 返回 error
- 删除 unnecessary `binary` 引用

**边界/不做**：
- 不改已通过的 S1-S4/S6-S10 实现
- 不改 `bytecode.go` 的 gofmt 空格调整（已保留）
- 不部署（D-COMMIT-SCOPE-001 部署闸仍有效）
- 不 commit/push/deploy
- 禁 `--no-verify`

---

## S1 — 补充 S5 对抗证明：marshal 失败场景

**目标**：构造 `MarshalBytecode` 失败场景，验证 `CompileMQLCached` 和 `CompilePythonCached` 返回 error 而非 `return r, nil, nil`。

**问题分析**：
- `MarshalBytecode` 对 valid bytecode 永不失败（当前唯一失败路径是 nil bc）
- `CompileMQLCached` 在 `CompileMQL` 成功后调用 `MarshalBytecode(bc)`，此时 `bc` 是 valid 的 → marshal 不会失败
- 因此无法通过常规路径触发 marshal 失败 → 需要用 mock/注入方式

**精确坐标**：
- 文件：`backend/tools/mql2go/vm_cache_integrity_redo_test.go`
- 当前 `TestMarshalErrorNotSwallowed`（:68-93）：只测 nil bc + 成功路径非 nil bcData
- `CompileMQLCached` marshal 失败路径（`interp_runner.go:74`）：`return nil, nil, fmt.Errorf("marshal freshly compiled bytecode: %w", mErr)`
- `CompilePythonCached` marshal 失败路径（`interp_runner.go:101`）：`return nil, nil, fmt.Errorf("marshal freshly compiled Python bytecode: %w", mErr)`

**落点**：
方案——新增 `TestCompileMQLCached_MarshalFailureReturnsError`，通过 mutation 验证：

1. 新增测试函数，用 `t.Run` 子测试：
   - 子测试 1：验证 `CompileMQLCached` 成功时返回非 nil bytecode（已有逻辑）
   - 子测试 2（对抗）：临时 mutation `MarshalBytecode` 使其对 valid bc 也返回 error（如注入一个 `marshalHook` 变量），验证 `CompileMQLCached` 返回 error 且 runner==nil
2. 或者更简单的方案——直接测试 `MarshalBytecode` 的 nil 路径 + 验证 `CompileMQLCached` 的 error 传播：
   - 构造一个 `bc` 为 nil 的场景（如 `CompileMQL` 返回 nil runner 但 nil error——不可能，但可用 mock）
   - 或用 `MarshalBytecode(nil)` 验证 error 返回，然后验证 `CompileMQLCached` 的代码路径 `if mErr != nil { return nil, nil, fmt.Errorf(...) }` 会被执行

**推荐方案（最简且对抗有效）**：
在 `interp_runner.go` 新增一个 package-level `marshalHook` 变量（仅测试用，生产为 nil）：
```go
// marshalHook is used only by tests to inject marshal failures.
// Production code leaves this nil.
var marshalHook func(*Bytecode) ([]byte, error)

// 在 CompileMQLCached 和 CompilePythonCached 的 marshal 调用处：
marshal := MarshalBytecode
if marshalHook != nil {
    marshal = marshalHook
}
data, mErr := marshal(bc)
```

测试中设置 `marshalHook` 返回 error → 验证 `CompileMQLCached` 返回 error → RED（如恢复 `return r, nil, nil` 则返回 nil error → 测试断言失败）→ restore → GREEN。

**对抗证明**：
- 设置 `marshalHook` 返回 `errors.New("injected marshal failure")` → `CompileMQLCached` 返回 error 且 runner==nil → GREEN
- 突变：恢复 `return r, nil, nil`（吞 error）→ `CompileMQLCached` 返回 nil error + non-nil runner → 测试断言 `err != nil` 失败 → RED
- 恢复 → GREEN
- 同样验证 `CompilePythonCached`

**注意**：`marshalHook` 是测试注入点，生产代码中为 nil，不影响生产行为。注释明确标注"仅测试用"。

---

## S2 — 删除 unnecessary binary 引用

**目标**：删除 `vm_cache_integrity_redo_test.go` 中未实际使用的 `encoding/binary` 导入和 `var _ = binary.LittleEndian`。

**精确坐标**：
- 文件：`backend/tools/mql2go/vm_cache_integrity_redo_test.go`
- :4 — `"encoding/binary"` 导入
- :316-317 — `// Ensure binary package is used (for potential future extensions)` + `var _ = binary.LittleEndian`

**落点**：
- 删除 :4 的 `"encoding/binary"` 导入
- 删除 :316-317 的注释和 `var _ = binary.LittleEndian`
- 确认 `go build` 和 `go vet` 仍通过

---

## 验收标准

1. **S1 对抗证明**：marshal 失败注入 → `CompileMQLCached`/`CompilePythonCached` 返回 error → 突变恢复吞 error → RED → restore → GREEN
2. **S2 洁净**：`grep "binary" vm_cache_integrity_redo_test.go` 返回 0 行（除注释中"binary surgery"等无关词）
3. **门禁全绿**：
   - `go build ./...`
   - `go test ./tools/mql2go/... -count=1`
   - `go test -race ./tools/mql2go/... -count=1` ×3
   - `go vet ./tools/mql2go/...`
   - `go run ./tools/check-file-lines --strict`（0 errors）
   - `git diff --check`
4. **不破坏已通过的 S1-S4/S6-S10 测试**：原有 9 个测试仍全 GREEN

## 红队自审（施工方完工前必答）

1. `marshalHook` 在生产代码中是否为 nil？是否影响生产行为？
2. `marshalHook` 是否需要 reset（测试结束后恢复 nil）？用 `t.Cleanup` 确保。
3. `CompilePythonCached` 的 marshal 失败路径是否也测试了？
4. 删除 `binary` 导入后 `go build` 是否通过？

## 回填纪律

1. registry `VM-CACHE-INTEGRITY-1`（:110）和 `VM-CACHE-INTEGRITY-2`（:115）：更新对抗证明结果（S5 补充）
2. `handover-audit-plan.md` 变更日志加一行
3. **不自行宣告完成**——停手等 Devin CLI 复审

## 范围约束

One task = one scope：只动 `backend/tools/mql2go/interp_runner.go`（新增 `marshalHook`）+ `backend/tools/mql2go/vm_cache_integrity_redo_test.go`（S1 新测试 + S2 删除 binary）。不顺手重构、不改已通过的实现。

## 固定尾部

**勿部署，停手等 Devin CLI 复审。** 禁 `--no-verify`。禁 commit/push/deploy。只 add 本任务文件，禁 `git add -A`。
