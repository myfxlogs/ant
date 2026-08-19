---
description: Understand and modify the mql2go transpiler/runtime architecture (tree-sitter → IR → Bytecode VM)
---

# mql2go Architecture

## Runtime pipeline (ADR-0023)

```text
MQL source → PreprocessMQL → tree-sitter parse → CompileToIR (AST) → CompileAST (Bytecode) → VMRunner (sdk.Strategy)
```

- In-process execution: `for { switch op }` + explicit data stack + instruction counter.
- `MaxSourceSize = 500_000` bytes.
- All compile paths use `defer recover()` for tree-sitter cgo panics.
- VM has instruction counter + context timeout + `safeRun()` panic recovery.

## Key files

- `tools/mql2go/preprocess.go` — MQL preprocessing (`#include` / `#define` / `#ifdef` / `#property`)
- `tools/mql2go/analyze.go` — tree-sitter parse + `detectMQLVersion`
- `tools/mql2go/compile_interp.go` — CST → AST (`interp.IR`)
- `tools/mql2go/compile.go` — AST → Bytecode compiler
- `tools/mql2go/vm.go` — Bytecode VM
- `tools/mql2go/interp_runner.go` — `CompileMQL` / `CompileMQLWithCoverage`
- `tools/mql2go/ast_coverage.go` — static + compilation coverage merge
- `tools/mql2go/ast_params.go` — `ExtractParamInfos` from Bytecode

## `interp/` subpackage

- `ir.go` — IR type definitions
- `exec.go` — VM execution engine
- `analyze.go` — static analysis (coverage + blind spots)
- `builtins.go` — MQL function → SDK method mapping
- `builtin_registry.go` — implemented function name list
- `builtin_trade.go` / `builtin_indicators.go` / `builtin_math.go` / `builtin_tools.go`

## MQL4 vs MQL5 version awareness

- MQL4: `OrderSend`, `OrderClose`, `OrderModify`, `OrderSelect` + `OrdersTotal` → `OrderLoopRule`.
- MQL5: `CTrade` class, `PositionGetTicket`, `PositionsTotal` → `PositionLoopRule`.
- Version detection uses `detectMQLVersion` signals: `class`, `CTrade`, `#include <Trade`, `MqlTradeRequest`, `PositionGetDouble/Integer/String/Ticket`, etc.
- `extern` (MQL4) parses as `primitive_type`; `input` (MQL5) parses as `type_identifier` with type in `ERROR` node.
- `_Point` / `_Symbol` are MQL5 equivalents of `Point` / `Symbol()`.

## Deprecated (do not revive)

- Python `ast_transpiler.py` / `ast_bridge.py` / `quality_gate.py`
- `go/ast` Go source code generation for runtime
- `ir_serialize.go` and `wasm_harness.go` (deleted)
- WASM execution and IR serialization

## Important notes

- `gen.go` / `gen_ir.go` are retained only for CLI dev debugging, **not** runtime.
- MQL source is the single source of truth (`imported_strategies.source_code`).
- MT4 and MT5 adapters must not share code except `adapter/mdtick/` shared DTO.
