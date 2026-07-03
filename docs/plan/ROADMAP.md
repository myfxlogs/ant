# ROADMAP — M11: MQL→VM 单一执行管线

> **Status: COMPLETE** — All cards verified via `go test ./tools/mql2go/...` (33 tests pass). Handover logs removed during repo cleanup (2026-07-03).

## M11.1 — 清除旧解释器执行路径

| ID | Status | Description | Tests |
|----|--------|-------------|-------|
| M11.1-1 | ☑ | 删除 interp/ 旧执行文件 (exec.go, eval.go, builtins.go, builtin_*.go, orderpool.go, pospool.go, series.go, userfunc.go) | TestAllBuiltinsWired |
| M11.1-2 | ☑ | 删除 interp/ 旧测试 (completeness, integration, ontrade, ordercloseby, pool, onarray, symbolinfo, venus_e2e, builtin_tools, blindspot, interp_test) | TestAnalyze_MQL4_FullSupport |
| M11.1-3 | ☑ | 移动 RuntimeBlindSpot 类型到 value.go | TestVMRunner |

## M11.2 — 清除 Go 代码生成路径

| ID | Status | Description | Tests |
|----|--------|-------------|-------|
| M11.2-1 | ☑ | 删除 gen_ir*.go (5 文件) — ADR-0023 已退役 | TestCompileToIR_BasicMQL4 |
| M11.2-2 | ☑ | 删除 cmd/mql2go CLI (仅调用 GenerateFromIR) | TestCompileToIR_MQL5Detection |
| M11.2-3 | ☑ | 删除 behavioral_test.go + 清理 mql2go_test.go 中 GenIR 测试 | TestAnalyze_MQL4FullCoverage |

## M11.3 — VM 内置函数完善

| ID | Status | Description | Tests |
|----|--------|-------------|-------|
| M11.3-1 | ☑ | 实现 328 个 VM 内置函数 (Math/String/Time/Array/Convert/Checkup/Trade/Indicators/Market/Account/Globals) | TestBuiltinCount |
| M11.3-2 | ☑ | 实现 24 个 MQL5 指标内置函数 (iAlligator/iIchimoku/iEnvelopes 等) | TestE2E_VMRunner_iMA |
| M11.3-3 | ☑ | 接线完整性测试 (20 个永久 blind spot: Object/Chart/File) | TestNoDuplicateBuiltins |

## M11.4 — 代码清理

| ID | Status | Description | Tests |
|----|--------|-------------|-------|
| M11.4-1 | ☑ | 清理 constants.go 注释 (移除 eval.go/gen_ir_expr.go 引用) | TestCompileToIR_ThreadSafety |
| M11.4-2 | ☑ | 清理 analyze.go + builtin_registry.go 注释 (移除 interpreter 引用) | TestAnalyze_MQL5CTradeCoverage |
| M11.4-3 | ☑ | 删除未使用的 MQLConstantInt() 函数 | TestAnalyze_StubIndicatorIsWarning |

## M11.5 — 前端修复

| ID | Status | Description | Tests |
|----|--------|-------------|-------|
| M11.5-1 | ☑ | 修复 CodeEditorPanel.tsx JSX 语法错误 (缺少 }) | npm run build |
