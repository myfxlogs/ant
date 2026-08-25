# VM-TRADE-CONTEXT-6/7, VM-API-TRUTH-3, VM-CACHE-INTEGRITY-5, VM-COMPILER-SEMANTICS-4
# STATUS: SUPERSEDED (2026-08-25) — historical round 5 proof record only.
# The next proof registry must be regenerated from D-VM-LIVE-001 and current code.
# Adversarial Proof Registry — independently verifiable mutation targets.
#
# VM-TEST-EVIDENCE-4 (返工第五阶段): each proof below has a specific mutation
# target (file:line), expected RED behavior, and restore instruction. An
# independent auditor can reproduce by applying the mutation, running the
# test, observing RED, then restoring and observing GREEN.
#
# 返工第五阶段说明: 旧版 Proof 9d 仍假绿 (covErr 删除后被 cov==nil 分支掩盖),
# Proof 11 的 strings.Contains 放行非法 input/extern, buildTradeContext 未知 enum
# 归一成 buy/fill, ExecuteLive 信任客户端身份, live 空财务可执行, IsTradeAllowed
# 用 connected proxy, accountIsInvestorLookup 可选。本次返工:
#   - Proof 9d → 注入 non-nil coverage + error (删除 covErr 后 InjectCoverage 成功 → RED)
#   - Proof 11 → 结构化 input/extern 检测 (isInputDeclaration/isExternDeclaration)
#   - 新增 Proof 2f/2g/2h (live 空财务/buildTradeContext enum/ExecuteLive 身份)
#   - 新增 Proof 6i (accountIsInvestorLookup 必选)
#   - 新增 Proof 6j (IsTradeAllowed 非 connected proxy)
#   - 新增 Proof 6e/6f/6g/6h/9c/9d/9e/11b (lookup error/investor/trailing/
#     coverage restore failure/invalid declaration)

# ── VM-TRADE-CONTEXT-6 ────────────────────────────────────────────────

# Proof 1: Array length validation in vmHandleBar
# Mutation: remove the OHLCV array length check in vm_live_handlers.go
# Test: TestVMHandleBar_ArrayLengthMismatch
# Expected RED: test fails (validation removed, execution continues with bad data)
# Restore: re-add the length check block
# File: internal/connect/strategy/vm_live_handlers.go
# Test file: internal/connect/strategy/vm_trade_context6_test.go:TestVMHandleBar_ArrayLengthMismatch

# Proof 2: Login injection in buildLiveContext
# Mutation: remove `lctx.Login = s.accountLoginLookup(ctx, cfg.AccountID)` in live_context.go
# Test: TestBuildLiveContext_InjectsLoginAndCompany
# Expected RED: Login=0, want 123456
# Restore: re-add the Login injection line
# File: internal/connect/strategy/live_context.go
# Test file: internal/connect/strategy/vm_trade_context6_test.go:TestBuildLiveContext_InjectsLoginAndCompany

# Proof 2b: validateFirstBarContext wired into Start() + dispatchVMLive pre-Init
# Mutation: remove `validateFirstBarContext(bctx)` call in vm_live_session.go Start()
#   OR remove `validateLiveContext` call before r.Init() in vm_live_dispatch.go
# Test: TestValidateFirstBarContext_InvalidDecimalRejected (Start-level)
#   AND TestDispatchVMLive_RejectsInvalidBeforeInit (dispatch-level, committed)
# Expected RED: Start() with invalid decimal ("bad_decimal") does NOT fail
#   AND dispatchVMLive executes OnInit before rejecting invalid context
# Restore: re-add the validateFirstBarContext call AND validateLiveContext call
# File: internal/connect/strategy/vm_live_session.go, vm_live_dispatch.go
# Test file: internal/connect/strategy/vm_trade_context6_test.go
#   AND vm_trade_context6_round4_test.go:TestDispatchVMLive_RejectsInvalidBeforeInit
# 返工第四阶段说明: 旧 Proof 2b 指向 temporary test, 仓库内只有 helper tests。
#   本次改用已提交的 TestDispatchVMLive_RejectsInvalidBeforeInit, 验证
#   dispatchVMLive 在 r.Init() 前调用 validateLiveContext, invalid context
#   不执行 OnInit (g_init=0, not 1)。

# Proof 2c: Strict decimal parsing in vmHandleBar
# Mutation: replace parseDecimalStrict with parseDecimal (lenient) in vmHandleBar
# Test: TestVMHandleBar_InvalidDecimalRejected
# Expected RED: invalid decimal "bad" accepted instead of rejected
# Restore: re-add parseDecimalStrict
# File: internal/connect/strategy/vm_live_handlers.go
# Test file: internal/connect/strategy/vm_trade_context6_test.go:TestVMHandleBar_InvalidDecimalRejected

# Proof 2d: Nil position rejection in vmHandleBar
# Mutation: remove nil check on positions in vmHandleBar
# Test: TestVMHandleBar_NilPositionRejected
# Expected RED: nil position array accepted instead of rejected
# Restore: re-add the nil check
# File: internal/connect/strategy/vm_live_handlers.go
# Test file: internal/connect/strategy/vm_trade_context6_test.go:TestVMHandleBar_NilPositionRejected

# Proof 2e: Live-mode lookup fail-closed
# Mutation: remove fail-closed error returns for Login/Company in buildLiveContext
# Test: TestBuildLiveContext_LiveModeLookupFailClosed
# Expected RED: buildLiveContext returns nil error instead of failing closed
# Restore: re-add the fail-closed error returns
# File: internal/connect/strategy/live_context.go
# Test file: internal/connect/strategy/vm_trade_context6_test.go:TestBuildLiveContext_LiveModeLookupFailClosed

# ── VM-TRADE-CONTEXT-7 ────────────────────────────────────────────────

# Proof 3: MT4 Deviation→Slippage mapping
# Mutation: remove `Slippage: req.Deviation` in mt4/orders.go
# Test: TestPlaceOrder_PassesDeviationAsSlippage
# Expected RED: OrderSend.Slippage = 0, want 20
# Restore: re-add `Slippage: req.Deviation`
# File: internal/mdgateway/adapter/mt4/orders.go
# Test file: internal/mdgateway/adapter/mt4/mt4_test.go:TestPlaceOrder_PassesDeviationAsSlippage

# Proof 4: MT5 Deviation→Slippage mapping
# Mutation: remove `Slippage: pUint64(uint64(req.Deviation))` in mt5/orders.go
# Test: TestPlaceOrder_PassesDeviationAsSlippage
# Expected RED: OrderSend.Slippage = nil (not set)
# Restore: re-add the Slippage line
# File: internal/mdgateway/adapter/mt5/orders.go
# Test file: internal/mdgateway/adapter/mt5/mt5_test.go:TestPlaceOrder_PassesDeviationAsSlippage

# ── VM-API-TRUTH-3 (返工) ─────────────────────────────────────────────

# Proof 5: IsDemo reads from context (not hardcoded true) — VM builtin level
# Mutation: revert builtinIsDemo to `return interp.BoolVal(true), nil`
# Test: TestVMLiveSession_IsDemoEndToEnd
# Expected RED: IsDemo() = 1 (true), want 0 (false, real account)
#   (test reads back VM global g_isDemo after OnInit execution)
# Restore: re-add the context-based return
# File: tools/mql2go/vm_builtin_checkup.go:builtinIsDemo
# Test file: internal/connect/strategy/vm_api_truth3_test.go:TestVMLiveSession_IsDemoEndToEnd
# 返工说明: 旧 Proof 6 用 TestBuildLiveContext_InjectsIsDemo — mutation 后
#   IsDemo 默认 false (零值) 与 lookup 返回 false 相同 → 假绿。改用端到端
#   VMLiveSession.Start → OnInit → IsDemo() → 读回 VM global, mutation 后
#   builtin 返回 true → g_isDemo=1 → RED.

# Proof 5b: IsTradeAllowed from lookup (not hardcoded true) — live mode
# Mutation: hardcode `lctx.IsTradeAllowed = true` in buildLiveContext live branch
# Test: TestBuildLiveContext_LiveModeIsTradeAllowedFromLookup
# Expected RED: IsTradeAllowed=true, want false (lookup returns false)
# Restore: re-add `lctx.IsTradeAllowed = s.accountTradeAllowedLookup(ctx, cfg.AccountID)`
# File: internal/connect/strategy/live_context.go
# Test file: internal/connect/strategy/vm_api_truth3_test.go:TestBuildLiveContext_LiveModeIsTradeAllowedFromLookup

# Proof 5c: IsConnected from lookup (not hardcoded true) — live mode
# Mutation: hardcode `lctx.IsConnected = true` in buildLiveContext live branch
# Test: TestBuildLiveContext_LiveModeIsConnectedFromLookup
# Expected RED: IsConnected=true, want false (lookup returns false)
# Restore: re-add `lctx.IsConnected = s.accountConnectedLookup(ctx, cfg.AccountID)`
# File: internal/connect/strategy/live_context.go
# Test file: internal/connect/strategy/vm_api_truth3_test.go:TestBuildLiveContext_LiveModeIsConnectedFromLookup

# Proof 5d: IsTradeAllowed false propagates to VM
# Mutation: revert builtinIsTradeAllowed to `return interp.BoolVal(true), nil`
# Test: TestVMLiveSession_IsTradeAllowedFalseEndToEnd
# Expected RED: IsTradeAllowed() = 1 (true), want 0 (false)
#   (test reads back VM global g_isTradeAllowed after OnInit)
# Restore: re-add the context-based return
# File: tools/mql2go/vm_builtin_checkup.go:builtinIsTradeAllowed
# Test file: internal/connect/strategy/vm_api_truth3_test.go:TestVMLiveSession_IsTradeAllowedFalseEndToEnd

# ── VM-CACHE-INTEGRITY-5 (返工) ───────────────────────────────────────

# Proof 7: CoverageResult restore on cache hit
# Mutation: remove the CoverageResult restore block in CompilePythonCached
# Test: TestCompilePythonCached_RestoresCoverageOnCacheHit
# Expected RED: cache hit should restore CoverageResult, got nil
# Restore: re-add the coverage restore block
# File: tools/mql2go/interp_runner.go:CompilePythonCached
# Test file: tools/mql2go/vm_cache_integrity5_test.go:TestCompilePythonCached_RestoresCoverageOnCacheHit

# Proof 8: Language (Version) validation in CompilePythonCached
# Mutation: remove `&& r.Bytecode().Version == "python"` check
# Test: TestCompilePythonCached_RejectsMQLBytecode
# Expected RED: Version="mql4", want 'python' (MQL bytecode accepted for Python source)
# Restore: re-add the Version == "python" check
# File: tools/mql2go/interp_runner.go:CompilePythonCached
# Test file: tools/mql2go/vm_cache_integrity5_test.go:TestCompilePythonCached_RejectsMQLBytecode

# Proof 9 (返工): Total payload size limit — specific error message assertion
# Mutation: remove the `len(data) > maxBytecodePayload` check in UnmarshalBytecode
# Test: TestUnmarshalBytecode_PayloadLimitExceedsMax
# Expected RED: error does NOT contain "exceeds max" (falls through to magic
#   check which returns "invalid magic" — different error message)
# Restore: re-add the payload size check
# File: tools/mql2go/bytecode_cache.go:UnmarshalBytecode
# Test file: tools/mql2go/vm_cache_integrity5_test.go:TestUnmarshalBytecode_PayloadLimitExceedsMax
# 返工说明: 旧 Proof 9 用 TestUnmarshalBytecode_PayloadLimit 只检查 err != nil —
#   mutation 后 magic check 仍返回 error → 假绿。改用断言 "exceeds max" +
#   "payload size" 特定 error message, mutation 后 error 变为 "invalid magic"
#   → 不包含 "exceeds max" → RED.

# Proof 9b: Bytecode.Language dead field removed — reflection check
# Mutation: re-add `Language string` field to Bytecode struct
# Test: TestBytecode_NoLanguageField
# Expected RED: reflect.TypeOf(Bytecode{}).FieldByName("Language") returns true
#   (field exists) → t.Fatal triggers
# Restore: remove the Language field from Bytecode struct
# File: tools/mql2go/bytecode.go
# Test file: tools/mql2go/vm_cache_integrity5_test.go:TestBytecode_NoLanguageField
# 返工第四阶段说明: 旧 Proof 9b 只检查 bc.Version != "python" — 重新加入
#   Language 字段后测试仍 GREEN (没有检查字段不存在)。改用 reflect.TypeOf
#   检查 FieldByName("Language") 返回 false, mutation 后字段存在 → RED.

# ── VM-COMPILER-SEMANTICS-4 (返工) ────────────────────────────────────

# Proof 10: Comma expression side effects preserved — VM execution
# Mutation: revert comma_expression to only return last child (discard side effects)
# Test: TestCommaExpression_VMSideEffectsExecution
# Expected RED: g_a=0, g_b=0 (first two assignments not executed), want 10, 20
#   Also: TestCommaExpression_VMFunctionCallSideEffects → g_counter=1, want 3
# Restore: re-add the ExprSeq generation
# File: tools/mql2go/compile_interp_expr.go:comma_expression case
# Test file: tools/mql2go/vm_compiler_semantics4_test.go:TestCommaExpression_VMSideEffectsExecution
# 返工说明: 旧 Proof 10 只检查 IR 有 ExprSeq — mutation 后 IR 无 ExprSeq →
#   RED 但只验证 IR 结构, 不验证 VM 执行副作用。新增 VM 执行测试读回 globals
#   g_a/g_b/g_c 和 g_counter, mutation 后只有最后一个赋值执行 → RED.

# Proof 11 (返工第五阶段): Structured input/extern exception in HasError guard
# Mutation: revert isInputDeclaration/isExternDeclaration to strings.Contains
# Test: TestCompileMQL_InvalidInputMissingInitializer
#   AND TestCompileMQL_InvalidExternMissingInitializer
#   AND TestCompileMQL_InvalidInputAsValue
# Expected RED: "input int X = ;" accepted (strings.Contains matches "input ")
#   "extern int X = ;" accepted (strings.Contains matches "extern ")
#   "int x = input ;" accepted (strings.Contains matches "input ")
# Restore: re-add the structured isInputDeclaration/isExternDeclaration checks
# File: tools/mql2go/compile_interp.go:CompileToIR
# Test file: tools/mql2go/vm_compiler_semantics4_round4_test.go
# 返工第五阶段说明: 旧 Proof 11 用 strings.Contains 放行所有含 "input " 或
#   "extern " 的 source, 包括 "int x = input ;" 等非法用法。本次改用结构化
#   检测: isInputDeclaration 检查第一个 named child 是 type_identifier "input",
#   isExternDeclaration 检查第一个 named child 是 storage_class_specifier "extern".
#   isValidInputDeclaration 检查 init_declarator 最后一个 named child 非空
#   (区分 "input int X = 5;" 和 "input int X = ;"). checkReservedKeywordUsage
#   拒绝 "input"/"extern" 作为 identifier (catches "int x = input ;").
#
# Proof 11b: HasError guard allows input/extern (no false positive)
# Mutation: remove the `input `/`extern ` exception in the HasError check
# Test: TestCompileMQL_ValidInputDeclarationAccepted
# Expected RED: valid MQL5 with "input int X = 5;" rejected (false positive)
# Restore: re-add the input/extern exception
# File: tools/mql2go/compile_interp.go:CompileToIR
# Test file: tools/mql2go/vm_compiler_semantics4_round4_test.go:TestCompileMQL_ValidInputDeclarationAccepted
#
# Proof 11c: Error recovery does not silently skip invalid declarations
# Mutation: remove the HasError check (same as Proof 11)
# Test: TestCompileMQL_ErrorRecoveryValidAfterInvalid
# Expected RED: source with invalid declaration followed by valid OnInit
#   is accepted (error recovery silently skips the invalid declaration)
# Restore: re-add the HasError check
# File: tools/mql2go/compile_interp.go:CompileToIR
# Test file: tools/mql2go/vm_compiler_semantics4_round4_test.go:TestCompileMQL_ErrorRecoveryValidAfterInvalid

# ── VM-API-TRUTH-3 (返工第四阶段) ────────────────────────────────────

# Proof 6e: Lookup query error blocks execution (fail-closed)
# Mutation: change `return false, queryErr` to `return false, nil` in any lookup
# Test: TestBuildLiveContext_LookupQueryErrorBlocksExecution
# Expected RED: buildLiveContext succeeds (DB error silently ignored, IsDemo=false)
# Restore: re-add the error return
# File: internal/connect/strategy/live_context.go:buildLiveContext
# Test file: internal/connect/strategy/vm_api_truth3_round4_test.go:TestBuildLiveContext_LookupQueryErrorBlocksExecution

# Proof 6f: Investor account gets IsTradeAllowed=false even when connected
# Mutation: remove the `if isInvestor { tradeAllowed = false }` block
# Test: TestBuildLiveContext_InvestorConnectedIsTradeAllowedFalse
# Expected RED: IsTradeAllowed=true (investor account can trade, wrong)
# Restore: re-add the investor gating block
# File: internal/connect/strategy/live_context.go:buildLiveContext
# Test file: internal/connect/strategy/vm_api_truth3_round4_test.go:TestBuildLiveContext_InvestorConnectedIsTradeAllowedFalse

# Proof 6g: Investor lookup query error blocks execution
# Mutation: change `return false, queryErr` to `return false, nil` in is_investor lookup
# Test: TestBuildLiveContext_InvestorLookupQueryErrorBlocksExecution
# Expected RED: buildLiveContext succeeds (investor lookup error ignored)
# Restore: re-add the error return
# File: internal/connect/strategy/live_context.go:buildLiveContext
# Test file: internal/connect/strategy/vm_api_truth3_round4_test.go:TestBuildLiveContext_InvestorLookupQueryErrorBlocksExecution

# Proof 6h: Real false not confused with query error
# Mutation: change `return false, nil` to `return false, errors.New("fake")` in connected lookup
# Test: TestBuildLiveContext_RealFalseNotConfusedWithError
# Expected RED: buildLiveContext fails (real false treated as query error)
# Restore: re-add the nil error return
# File: internal/connect/strategy/live_context.go:buildLiveContext
# Test file: internal/connect/strategy/vm_api_truth3_round4_test.go:TestBuildLiveContext_RealFalseNotConfusedWithError

# ── VM-CACHE-INTEGRITY-5 (返工第四阶段) ──────────────────────────────

# Proof 9c: Trailing garbage rejected — specific error assertion
# Mutation: remove the `if r.pos != len(data)` check in UnmarshalBytecode
# Test: TestUnmarshalBytecode_TrailingGarbage
# Expected RED: trailing garbage accepted (no trailing data check)
# Restore: re-add the trailing data check
# File: tools/mql2go/bytecode_cache.go:UnmarshalBytecode
# Test file: tools/mql2go/vm_cache_integrity5_test.go:TestUnmarshalBytecode_TrailingGarbage
# 返工第四阶段说明: 旧 Proof 9c 用 t.Log and pass — trailing garbage 可能被
#   接受。本次改用断言 err != nil AND err contains "trailing", mutation 后
#   trailing data check 删除 → err=nil → RED.

# Proof 9d: Coverage restore failure returns error (injectable, hits covErr branch)
# Mutation: remove the `if covErr != nil` check in CompilePythonCached
# Test: TestCompilePythonCached_CoverageRestoreFailureReturnsError
# Expected RED: cache hit succeeds (covErr deleted → cov != nil → InjectCoverage
#   succeeds → returns nil error → test expects error → RED)
# Restore: re-add the covErr check
# File: tools/mql2go/interp_runner.go:CompilePythonCached
# Test file: tools/mql2go/vm_cache_integrity5_test.go:TestCompilePythonCached_CoverageRestoreFailureReturnsError
# 返工第五阶段说明: 旧 Proof 9d 注入 nil coverage + error, 删除 covErr 后
#   cov==nil 分支也返回 error → 假绿。本次注入 non-nil runner + non-nil coverage
#   + error (sentinel COVERAGE_RESTORE_FAIL_5F3A), 删除 covErr 后 cov != nil →
#   跳过 cov==nil 检查 → InjectCoverage 成功 → 返回 nil error → test expects
#   error → RED. 断言 error 包含 sentinel 证明来自 covErr 分支.

# Proof 9e: Coverage restore nil coverage returns error
# Mutation: remove the `if cov == nil` check in CompilePythonCached
# Test: TestCompilePythonCached_CoverageRestoreNilCoverageReturnsError
# Expected RED: cache hit succeeds with nil coverage (silent degradation)
# Restore: re-add the cov == nil check
# File: tools/mql2go/interp_runner.go:CompilePythonCached
# Test file: tools/mql2go/vm_cache_integrity5_test.go:TestCompilePythonCached_CoverageRestoreNilCoverageReturnsError

# Proof 9f: Cache hit vs cold compile coverage identity comparison
# Mutation: change `r.InjectCoverageResult(cov)` to inject a different coverage
# Test: TestCompilePythonCached_CacheHitVsColdCompileCoverageEqual
# Expected RED: BlindSpot Builtin/Severity or DefenseAViolation Rule mismatch
# Restore: re-add the correct coverage injection
# File: tools/mql2go/interp_runner.go:CompilePythonCached
# Test file: tools/mql2go/vm_cache_integrity5_test.go:TestCompilePythonCached_CacheHitVsColdCompileCoverageEqual
# 返工第四阶段说明: 旧 Proof 9f 只比较 count, 不比较 identity。本次改用比较
#   BlindSpot.Builtin/Severity 和 DefenseAViolation.Rule, mutation 后 identity
#   不匹配 → RED.

# ── VM-TRADE-CONTEXT-6 (返工第五阶段) ────────────────────────────────

# Proof 2f: Live mode rejects empty financial fields
# Mutation: change validateFinancialFieldsForMode to always use validateFinancialFields
#   (not validateLiveFinancialFields for live mode)
# Test: TestVMHandleBar_LiveModeEmptyFinancialRejected
# Expected RED: live mode with empty Balance/Equity/Margin/FreeMargin accepted
#   (should be rejected — authoritative broker data missing)
# Restore: re-add the mode == "live" branch using validateLiveFinancialFields
# File: internal/connect/strategy/vm_live_helpers.go:validateFinancialFieldsForMode
# Test file: internal/connect/strategy/vm_trade_context6_round5_test.go

# Proof 2g: buildTradeContext rejects unknown broker side/event type
# Mutation: revert brokerSideFromString/brokerTradeEventTypeString to default
#   sideBuy/"fill" for unknown values
# Test: TestBuildTradeContext_UnknownSideRejected
#   AND TestBuildTradeContext_UnknownEventTypeRejected
# Expected RED: unknown side/event type accepted (silently normalized to buy/fill)
# Restore: re-add the fail-closed error returns
# File: internal/connect/strategy/live_context.go:buildTradeContext
# Test file: internal/connect/strategy/vm_trade_context6_round5_test.go

# Proof 2h: ExecuteLive rejects client-submitted identity in live mode without account_id
# Mutation: remove the `if bctx.Mode == "live" && req.GetAccountId() == ""` check
#   in dispatchVMLive
# Test: TestDispatchVMLive_LiveModeRejectsClientIdentityWithoutAccountID
# Expected RED: live mode with client-submitted Login/Company/status accepted
#   (should be rejected — no server-side account truth)
# Restore: re-add the account_id required check
# File: internal/connect/strategy/vm_live_dispatch.go:dispatchVMLive
# Test file: internal/connect/strategy/vm_trade_context6_round5_test.go

# ── VM-API-TRUTH-3 (返工第五阶段) ────────────────────────────────────

# Proof 6i: accountIsInvestorLookup required in live mode
# Mutation: change `if s.accountIsInvestorLookup == nil` to `if false`
# Test: TestBuildLiveContext_MissingInvestorLookupRejected
# Expected RED: live mode without accountIsInvestorLookup succeeds
#   (investor safety gate bypassed)
# Restore: re-add the nil check
# File: internal/connect/strategy/live_context.go:buildLiveContext
# Test file: internal/connect/strategy/vm_api_truth3_round5_test.go

# Proof 6j: IsTradeAllowed not derived from connected status
# Mutation: change `return status == "trade_allowed"` to `return status == "connected"`
#   in handlers_strategy.go accountTradeAllowedLookup
# Test: TestAccountTradeAllowedLookup_NotConnectedProxy
# Expected RED: connected account gets IsTradeAllowed=true (connected proxy)
# Restore: re-add the `status == "trade_allowed"` check
# File: backend/cmd/server/handlers_strategy.go:accountTradeAllowedLookup
# Test file: internal/connect/strategy/vm_api_truth3_round5_test.go
# Note: This is a SQL wiring test, not a buildLiveContext callback test.
#   The mutation target is the production SQL query, not a test callback.
