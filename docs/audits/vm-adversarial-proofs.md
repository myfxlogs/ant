# VM Adversarial Proof Registry — Batch 1-3 (VM-TEST-EVIDENCE-4)

> **Status**: 🟦open（施工完成，待独立复审）— 2026-08-27 Batch 5 重写。
> **Supersedes**: round 5 proof registry (marked SUPERSEDED 2026-08-25 after D-REVERT-SCOPE-DRIFT-001).
>
> Each proof below has a specific mutation target (file:line), expected RED
> behavior, and restore instruction. An independent auditor can reproduce by
> applying the mutation, running the test, observing RED, then restoring and
> observing GREEN. All referenced test files and functions verified to exist.

## Verification commands

```bash
# Batch 1 (VM-COMPILER-SEMANTICS-4 + VM-CACHE-INTEGRITY-5)
go test ./tools/mql2go/ -run "<TestName>" -count=1 -v

# Batch 2 (VM-TRADE-CONTEXT-6)
go test ./internal/connect/strategy/ -run "<TestName>" -count=1 -v

# Batch 3 (VM-API-TRUTH-3)
go test ./tools/mql2go/ -run "<TestName>" -count=1 -v
```

---

## VM-COMPILER-SEMANTICS-4 proofs (Batch 1)

### Proof 1: comma_expression ExprSeq preserves side effects

- **Mutation target**: `tools/mql2go/compile_interp_expr.go:107` — revert the `case "comma_expression":` branch to only return the last child expression (discard earlier children's side effects).
- **Expected RED**: `TestCommaExpression_VMSideEffectsExecution` fails — `g_a=0, want 10` (first assignment not executed). Also `TestCommaExpression_VMFunctionCallSideEffects` fails — `g_counter=1, want 3`.
- **Restore**: re-add the `ExprSeq` generation that emits all children in sequence (`return &interp.Expr{Kind: interp.ExprSeq, Args: args}, nil` at line 143).
- **Test file**: `tools/mql2go/vm_round45_batch1_test.go:31` (`TestCommaExpression_VMSideEffectsExecution`) and `:383` (`TestCommaExpression_VMFunctionCallSideEffects`).

### Proof 2: hasMissingInitializer guard rejects invalid declarations

- **Mutation target**: `tools/mql2go/compile_interp.go:103` — remove the `if hasMissingInitializer(n) { return ... }` guard so declarations with missing initializers fall through.
- **Expected RED**: `TestCompileMQL_CompletelyInvalidSourceRejected` fails — `expected error for 'int x = ;' (missing initializer), got nil`. Also `TestCompileMQL_InvalidInputMissingInitializer` fails — `expected error for 'input int X = ;', got nil`.
- **Restore**: re-add the `hasMissingInitializer(n)` check at line 103.
- **Test file**: `tools/mql2go/vm_round45_batch1_test.go:114` (`TestCompileMQL_CompletelyInvalidSourceRejected`) and `:94` (`TestCompileMQL_InvalidInputMissingInitializer`).

### Proof 3: Structured input/extern detection + reserved keyword rejection

- **Mutation target**: `tools/mql2go/compile_interp.go:89` — remove the `checkReservedKeywordUsage(n, c)` call before the switch, AND revert `isInputDeclaration`/`isExternDeclaration` (lines 790/799) to `strings.Contains(sourceText, "input ")` / `strings.Contains(sourceText, "extern ")`.
- **Expected RED**: `TestCompileMML_ReservedKeywordAsIdentifierRejected` fails — `int x = input ;` accepted (reserved keyword used as identifier not caught). Also `TestCompileMQL_InvalidInputMissingInitializer` fails — `input int X = ;` accepted (strings.Contains matches "input " but doesn't validate structure).
- **Restore**: re-add `checkReservedKeywordUsage` before the switch (line 89) and restore the structured `isInputDeclaration`/`isExternDeclaration` checks (lines 790/799).
- **Test file**: `tools/mql2go/vm_round45_batch1_test.go:124` (`TestCompileMML_ReservedKeywordAsIdentifierRejected`) and `:94` (`TestCompileMQL_InvalidInputMissingInitializer`).

---

## VM-CACHE-INTEGRITY-5 proofs (Batch 1)

### Proof 4: CoverageResult restore on cache hit

- **Mutation target**: `tools/mql2go/interp_runner.go:112-127` — remove the coverage restore block in `CompilePythonCached` (the `var cov *CoverageResult` + `coverageRestoreHook` + `InjectCoverageResult(cov)` block).
- **Expected RED**: `TestCompilePythonCached_RestoresCoverageOnCacheHit` fails — `CoverageResult is nil on cache hit, expected restored`.
- **Restore**: re-add the coverage restore block that recompiles from source to recover `CoverageResult` and injects it via `r.InjectCoverageResult(cov)`.
- **Test file**: `tools/mql2go/vm_round45_batch1_test.go:160` (`TestCompilePythonCached_RestoresCoverageOnCacheHit`).

### Proof 5: Version=="python" check rejects MQL bytecode

- **Mutation target**: `tools/mql2go/interp_runner.go:111` — remove the `&& r.Bytecode().Version == "python"` condition so any cached bytecode with matching SourceHash is accepted regardless of language.
- **Expected RED**: `TestCompilePythonCached_RejectsMQLBytecodeForPythonSource` fails — `Version = "mql4", want "python"` (MQL bytecode accepted for Python source). Note: the test constructs a poisoned bytecode with matching SourceHash but Version="mql4" to ensure the Version check is independently exercised (not masked by SourceHash failure).
- **Restore**: re-add the `&& r.Bytecode().Version == "python"` condition at line 111.
- **Test file**: `tools/mql2go/vm_round45_batch1_test.go:259` (`TestCompilePythonCached_RejectsMQLBytecodeForPythonSource`).

### Proof 6: maxBytecodePayload guard rejects oversized payloads

- **Mutation target**: `tools/mql2go/bytecode_cache.go:168` — remove the `if len(data) > maxBytecodePayload { return nil, ... }` guard so oversized payloads fall through to the magic check.
- **Expected RED**: `TestUnmarshalBytecode_PayloadLimitExceeded` fails — `error should contain 'exceeds max', got: bytecode: invalid magic` (payload guard removed, magic check returns different error).
- **Restore**: re-add the `len(data) > maxBytecodePayload` check at line 168.
- **Test file**: `tools/mql2go/vm_round45_batch1_test.go:306` (`TestUnmarshalBytecode_PayloadLimitExceeded`).

---

## VM-TRADE-CONTEXT-6 proofs (Batch 2)

### Proof 7: OHLCV array length validation

- **Mutation target**: `internal/connect/strategy/vm_live_handlers.go:22` — remove the `validateOHLCVLengths(...)` call so mismatched OHLCV array lengths proceed to indexing (panic).
- **Expected RED**: `TestVMHandleBar_ArrayLengthMismatch` fails — panic: `runtime error: index out of range` (mismatched arrays indexed without validation).
- **Restore**: re-add the `validateOHLCVLengths` call at line 22.
- **Test file**: `internal/connect/strategy/vm_trade_context6_batch2_test.go:77` (`TestVMHandleBar_ArrayLengthMismatch`).

### Proof 8: Strict decimal parse rejects invalid values

- **Mutation target**: `internal/connect/strategy/vm_live_handlers.go:81` — revert `parseBarsStrict` to use `parseDecimal` (lenient, returns zero on error) instead of `parseDecimalStrict` (returns error).
- **Expected RED**: `TestVMHandleBar_InvalidDecimalRejected` fails — `vmHandleBar should fail on invalid decimal, got Success=true` (invalid "bad" silently converted to zero).
- **Restore**: re-add `parseDecimalStrict` in `parseBarsStrict` (line 85+).
- **Test file**: `internal/connect/strategy/vm_trade_context6_batch2_test.go:101` (`TestVMHandleBar_InvalidDecimalRejected`).

### Proof 9: Nil position rejection in live mode

- **Mutation target**: `internal/connect/strategy/vm_live_handlers.go:27` — remove the `rejectNilRepeatedInLive(...)` call so nil positions/pending_orders are accepted in live mode.
- **Expected RED**: `TestVMHandleBar_NilPositionRejected` fails — `vmHandleBar should fail on nil positions in live mode, got Success=true`.
- **Restore**: re-add the `rejectNilRepeatedInLive` call at line 27.
- **Test file**: `internal/connect/strategy/vm_trade_context6_batch2_test.go:125` (`TestVMHandleBar_NilPositionRejected`).

### Proof 10: Lookup fail-closed in live mode

- **Mutation target**: `internal/connect/strategy/live_context.go:255` — in `injectAccountTruth`, change the live-mode error returns to `return nil` (swallow DB lookup errors instead of failing closed).
- **Expected RED**: `TestBuildLiveContext_LiveModeLookupFailClosed` fails — `buildLiveContext should fail on lookup error in live mode, got nil` (DB error silently ignored).
- **Restore**: re-add the `return err` for lookup failures in live mode.
- **Test file**: `internal/connect/strategy/vm_trade_context6_batch2_test.go:203` (`TestBuildLiveContext_LiveModeLookupFailClosed`).

### Proof 11: validateFirstBarContext before Init

- **Mutation target**: `internal/connect/strategy/vm_live_session.go:109` — remove the `if err := validateFirstBarContext(bctx); err != nil { return ... }` call so `Start()` proceeds to `Init()` without validating the first bar context. Also `internal/connect/strategy/vm_live_dispatch.go:100` for the dispatch path.
- **Expected RED**: `TestVMLiveSession_StartRejectsInvalidFirstBarContext` fails — `Start should fail on invalid first bar context, got nil error` (invalid context reaches Init).
- **Restore**: re-add the `validateFirstBarContext(bctx)` call at line 109 (session) and line 100 (dispatch).
- **Test file**: `internal/connect/strategy/vm_trade_context6_batch2_test.go:269` (`TestVMLiveSession_StartRejectsInvalidFirstBarContext`).

### Proof 12: Login injection (SetLogin before Init)

- **Mutation target**: `internal/connect/strategy/vm_live_session.go:115` — remove the `s.runner.SetLogin(bctx.Login)` call so `AccountNumber()` returns 0 during OnInit. Also `internal/connect/strategy/vm_live_handlers.go:33` for the per-bar path.
- **Expected RED**: `TestVMLiveSession_EndToEndAccountNumberReadback` fails — `g_accountNumber = 0, want 12345` (Login not propagated to VM).
- **Restore**: re-add `s.runner.SetLogin(bctx.Login)` at line 115 and `r.SetLogin(lctx.Login)` at line 33.
- **Test file**: `internal/connect/strategy/vm_trade_context6_batch2_test.go:305` (`TestVMLiveSession_EndToEndAccountNumberReadback`).

---

## VM-API-TRUTH-3 proofs (Batch 3)

### Proof 13: builtinIsConnected reads from context (not hardcoded true)

- **Mutation target**: `tools/mql2go/vm_builtin_checkup.go:16` — revert `builtinIsConnected` to `return interp.BoolVal(true), nil` (hardcoded true, ignoring context).
- **Expected RED**: `TestBuiltinIsConnected_ReadsFromContext` fails — `IsConnected() = true, want false (from context IsConnected=false)`. Also `TestVMLive_IsConnectedEndToEnd` fails — `g_isConnected = 1, want 0`.
- **Restore**: re-add the `if vm.ctx == nil { return interp.BoolVal(true), nil }; return interp.BoolVal(vm.ctx.Account().IsConnected), nil` logic.
- **Test file**: `tools/mql2go/vm_api_truth3_batch3_test.go:25` (`TestBuiltinIsConnected_ReadsFromContext`) and `:132` (`TestVMLive_IsConnectedEndToEnd`).

### Proof 14: builtinIsTradeAllowed reads from context (not hardcoded true)

- **Mutation target**: `tools/mql2go/vm_builtin_checkup.go:51` — revert `builtinIsTradeAllowed` to `return interp.BoolVal(true), nil` (hardcoded true, ignoring context).
- **Expected RED**: `TestBuiltinIsTradeAllowed_ReadsFromContext` fails — `IsTradeAllowed() = true, want false (from context IsTradeAllowed=false)`. Also `TestVMLive_IsTradeAllowedEndToEnd` fails — `g_isTradeAllowed = 1, want 0` (investor account).
- **Restore**: re-add the `if vm.ctx == nil { return interp.BoolVal(true), nil }; return interp.BoolVal(vm.ctx.Account().IsTradeAllowed), nil` logic.
- **Test file**: `tools/mql2go/vm_api_truth3_batch3_test.go:73` (`TestBuiltinIsTradeAllowed_ReadsFromContext`) and `:149` (`TestVMLive_IsTradeAllowedEndToEnd`).

### Proof 15: End-to-end IsConnected context propagation via SetAccountStatus

- **Mutation target**: `internal/connect/strategy/vm_live_handlers.go:35` — remove the `r.SetAccountStatus(lctx.IsDemo, lctx.IsConnected, lctx.IsTradeAllowed)` call so the VM context never receives the authoritative IsConnected value. Also `internal/connect/strategy/vm_live_session.go:118` for the Start() path.
- **Expected RED**: `TestVMLive_IsConnectedEndToEnd` fails — `g_isConnected = 1, want 0` (IsConnected defaults to true via zero-value/nil-ctx fallback, never receives false from context).
- **Restore**: re-add `r.SetAccountStatus(...)` at line 35 (handlers) and line 118 (session).
- **Test file**: `tools/mql2go/vm_api_truth3_batch3_test.go:132` (`TestVMLive_IsConnectedEndToEnd`).

---

## Summary table

| Proof | Batch | ID | Mutation target | Test | Expected RED |
|-------|-------|----|-----------------|------|-------------|
| 1 | 1 | COMPILER-SEMANTICS-4 | `compile_interp_expr.go:107` comma→last only | `TestCommaExpression_VMSideEffectsExecution` | g_a=0, want 10 |
| 2 | 1 | COMPILER-SEMANTICS-4 | `compile_interp.go:103` remove hasMissingInitializer | `TestCompileMQL_CompletelyInvalidSourceRejected` | expected error, got nil |
| 3 | 1 | COMPILER-SEMANTICS-4 | `compile_interp.go:89` remove checkReservedKeywordUsage | `TestCompileMML_ReservedKeywordAsIdentifierRejected` | expected error, got nil |
| 4 | 1 | CACHE-INTEGRITY-5 | `interp_runner.go:112-127` remove coverage restore | `TestCompilePythonCached_RestoresCoverageOnCacheHit` | CoverageResult nil |
| 5 | 1 | CACHE-INTEGRITY-5 | `interp_runner.go:111` remove Version=="python" | `TestCompilePythonCached_RejectsMQLBytecodeForPythonSource` | Version="mql4" accepted |
| 6 | 1 | CACHE-INTEGRITY-5 | `bytecode_cache.go:168` remove payload guard | `TestUnmarshalBytecode_PayloadLimitExceeded` | "invalid magic" not "exceeds max" |
| 7 | 2 | TRADE-CONTEXT-6 | `vm_live_handlers.go:22` remove validateOHLCVLengths | `TestVMHandleBar_ArrayLengthMismatch` | panic: index out of range |
| 8 | 2 | TRADE-CONTEXT-6 | `vm_live_handlers.go:81` strict→lenient parse | `TestVMHandleBar_InvalidDecimalRejected` | Success=true (should fail) |
| 9 | 2 | TRADE-CONTEXT-6 | `vm_live_handlers.go:27` remove nil position check | `TestVMHandleBar_NilPositionRejected` | Success=true (should fail) |
| 10 | 2 | TRADE-CONTEXT-6 | `live_context.go:255` swallow lookup errors | `TestBuildLiveContext_LiveModeLookupFailClosed` | nil error (should fail) |
| 11 | 2 | TRADE-CONTEXT-6 | `vm_live_session.go:109` remove validateFirstBarContext | `TestVMLiveSession_StartRejectsInvalidFirstBarContext` | nil error (should fail) |
| 12 | 2 | TRADE-CONTEXT-6 | `vm_live_session.go:115` remove SetLogin | `TestVMLiveSession_EndToEndAccountNumberReadback` | g_accountNumber=0, want 12345 |
| 13 | 3 | API-TRUTH-3 | `vm_builtin_checkup.go:16` hardcode true | `TestBuiltinIsConnected_ReadsFromContext` | true, want false |
| 14 | 3 | API-TRUTH-3 | `vm_builtin_checkup.go:51` hardcode true | `TestBuiltinIsTradeAllowed_ReadsFromContext` | true, want false |
| 15 | 3 | API-TRUTH-3 | `vm_live_handlers.go:35` remove SetAccountStatus | `TestVMLive_IsConnectedEndToEnd` | g_isConnected=1, want 0 |
