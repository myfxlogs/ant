# 管线审计修复清单

> 审计日期: 2026-08-05 | 审计者: Claude Code | 状态: 待施工

---

## P0 — 阻塞项

### 1. 严重性管线断裂

**根因**: `backtest_worker_vm.go` 用 `CompileMQLCached` 而非 `CompileMQLWithCoverage`，丢失覆盖率报告；盲区 severity 写死 `"warning"`（英文），但规则引擎用 `interp.SeverityFatal`（中文"致命"）匹配。

**修复**:
- `backtest_worker_vm.go`: `executeVMBacktest` 改用 `CompileMQLCached` + 后续调用 `AnalyzeCoverage`；或改用 `CompileMQLWithCoverage`
- `backtest_worker_vm.go`: `buildBacktestResponse` 中盲区 severity 改用 `interp.SeverityFatal` / `interp.SeverityWarning` 常量，不要写死字符串
- `vm.go:151`: 运行时盲区 severity 应从注册表 `LookupAPI` 查询，而非硬编码 `"warning"`
- 字节码缓存命中时确保覆盖率不为 nil

**验证**: 
```bash
go test ./tools/mql2go/... ./internal/connect/strategy/...
```

### 2. 蜕变测试测错对象

**根因**: `d3_metamorphic_test.go` 的 7 个 MR 全测 `ref*` 参考实现，不测生产 SDK/VM。提案的五条蜕变关系未实现。

**修复**:
- 新建 `d3_metamorphic_production_test.go`（独立文件，不修改现有测试），实现 5 条蜕变关系：
  - **MR-P1**: `iMA(NULL,0,1,0,MODE_SMA,PRICE_CLOSE,0) ≡ Close[0]` — period=1 恒等
  - **MR-P2**: `平仓盈亏 ≡ (close−open) × lots × tickvalue × direction` — 盈亏恒等
  - **MR-P3**: `iHighest 返回的 bar.High ≥ 区间内所有 bar.High` — 边界约束
  - **MR-P4**: `EURUSD BUY 盈亏 = EURUSD SELL 盈亏取反`（同参数对称） — 买卖对称
  - **MR-P5**: `Lots×2 → 盈亏×2`（手数线性）
- 每条 MR 调用完整的 `CompileMQL → Engine.Run()` 管线，不调用 `ref*` 函数
- 参考: `tests/e2e/` 和 `e2e_real_grid_test.go` 的 TestRealGridEA_Backtest 模式

**验证**:
```bash
go test ./strategy/backtest/ -run "TestD3_MR-P" -v
```

---

## P1 — 重要项

### 3. ADX 算法 bug

**根因**: `strategy/indicators/core_oscillator.go` 中 ADX 计算 `dx := math.Abs(plusDI - minusDI)` 少除了 `/(plusDI + minusDI)`，未做 ×100 归一化。`d3_differential_test.go:282-302` 已发现但用公差 50.0 掩盖。

**修复**:
- `strategy/indicators/core_oscillator.go`: 修正 ADX 公式：`dx := math.Abs(plusDI - minusDI) / (plusDI + minusDI) * 100`
- `d3_differential_test.go:282-302`: 将 ADX 测试的公差从 50.0 改为 1e-6
- 确认 `vm_builtin_indicators.go` 的 `builtinIADX` 也使用修正后的值

**验证**:
```bash
go test ./strategy/indicators/ -run ADX -v
go test ./strategy/backtest/ -run TestD3_ADX -v
```

### 4. 注册表 StatusStubbed 死状态

**根因**: `api_registry.go` 的 `init()` 只添加 implemented/unsupported/constants，没有赋值 `StatusStubbed` 的路径。`IsAPIStubbed`、`compile_expr.go:384`、`analyze.go:96-100` 的 Stubbed 分支全是死代码。

**修复（选择一种）**:
- **方案 A**: 在 `buildInRegistryForStatus` 或 `init()` 中，对注册表中不在 `implemented*` 也不在 `unsupportedSymbols` 的函数批量标 `StatusStubbed`，让三态真正生效
- **方案 B**: 删除 `StatusStubbed` 和相关死代码，改为两态模型，更新文档

### 5. 注册表 _Point/_Symbol 虚假 implemented

**根因**: `builtin_registry.go` 的 `implementedMarketData` 包含 `_Point`/`_Symbol`/`ask`/`bid` 等 27 个名字，但 `builtinRegistry` 中无对应 handler。`_Symbol` 在 MQL5 EA 中触发编译错误。

**修复**:
- 从 `implementedMarketData` 移除这 27 个名字，或为它们添加 VM handler
- 补 CI 反向检查：`TestAPIRegistryImplementedConsistency` 应验证每个 implemented 名称都有 VM handler（检查 `builtinRegistry`）

**验证**:
```bash
go test ./tools/mql2go/interp/ -run TestAPIRegistry -v
```

---

## 修复后验证全部通过

```bash
cd backend
go build ./...
go test ./tools/mql2go/... -count=1
go test ./strategy/backtest/... -count=1
go test ./strategy/indicators/... -count=1
go run ./tools/check-file-lines --strict
```
