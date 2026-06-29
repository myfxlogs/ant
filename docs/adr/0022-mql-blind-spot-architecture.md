# ADR-0022: MQL 盲区架构 — 静态分析 + 运行时追踪 + 致命阻断

## 状态

Accepted — 2026-06-29

## 背景

MQL4/MQL5 有数百个内置函数。我们的 Go 解释器（`tools/mql2go/interp/`）实现了其中约 120 个，覆盖交易、行情、指标、平台工具等核心功能。但 EA 可能调用我们尚未实现的函数。

之前的设计有两个问题：

1. **运行时盲区无追踪**：未实现函数被调用时，只设置一个 `errSet bool` 标记，没有任何代码读取它。盲区被静默忽略，EA 带着错误结果继续运行。
2. **致命盲区不阻断**：`OrderSend` 返回 `NoneVal`（= 0），EA 认为"下单失败"但继续执行；`iCustom` 返回 0，EA 用错误的指标值做决策。这些应该立即中止。

## 决策

### 三层盲区处理

```
编译时 (静态)          运行时 (动态)           上报
┌──────────┐         ┌──────────┐         ┌──────────┐
│ Analyze  │         │ execBlock│         │ Get      │
│  → IR    │         │  panic   │         │ Runtime  │
│  Report  │         │  recover │         │ BlindSpots│
└──────────┘         └──────────┘         └──────────┘
   预览展示            致命→中止             回测/实盘后
```

### 第 1 层：静态分析（编译时）

**文件**: `interp/analyze.go` — `Analyze(ir *IR) *IRReport`

在编译 IR 后、执行前，遍历所有调用点，与 `builtin_registry.go` 中的已实现列表交叉比对。

**分级**:
- `"致命"` — 交易/指标函数（`OrderXxx`, `PositionXxx`, `iXxx`），不实现会导致 EA 逻辑错误
- `"警告"` — 一般未实现函数
- `"永久盲区"` — `permanentBlindSpots` 列表中的函数（`ObjectCreate`, `FileOpen` 等 MT 客户端 UI/文件操作），设计上永不实现

**用途**: 预览页面展示覆盖率，用户在运行前就知道哪些功能缺失。

### 第 2 层：运行时追踪 + 致命阻断

**文件**: `interp/builtins.go` — `recordBlindSpot(name string)`

当 OnTick/OnBar 实际执行到未实现函数时：

1. **记录**: `it.runtimeBlindSpots[name]++` — 累计命中次数
2. **日志**: `ctx.Log("MQL interpreter: unimplemented function ...")`
3. **分级判定**: `classifyRuntimeSeverity(name)` 返回 `"致命"` / `"警告"` / `"永久盲区"`
4. **致命阻断**: 如果是 `"致命"` 级别，`panic(errFatalBlindSpot)`
5. **恢复**: `execBlock` 的 `defer recover()` 捕获 `errFatalBlindSpot`，转为 error 返回给 `OnTick`/`OnBar`

**非致命的 `"警告"` 和 `"永久盲区"`** 继续执行，返回 `NoneVal()`。这些函数（如 `ObjectCreate`、`Comment`）对 EA 交易逻辑无影响，静默跳过是安全的。

### 第 3 层：运行后上报

**文件**: `interp/exec.go` — `GetRuntimeBlindSpots() []RuntimeBlindSpot`

回测或实盘运行后，调用方可以获取运行时实际触发的盲区列表：

```go
type RuntimeBlindSpot struct {
    Builtin  string  // 函数名
    Count    int     // 命中次数
    Severity string  // "致命" | "警告" | "永久盲区"
}
```

结果按严重度排序（致命优先），同级别按命中次数降序。

**API**:
- `GetRuntimeBlindSpots()` — 获取累计的运行时盲区
- `ResetRuntimeBlindSpots()` — 清空记录（如多次回测之间重置）

## 严重度分类规则

```
classifyRuntimeSeverity(name)
├── permanentBlindSpots[name] → "永久盲区"
├── isTradeName(name) || isIndicatorName(name) → "致命"
├── name matches "i[A-Z]*" → "致命" (自定义指标)
├── name starts with "Order" | "Position" | "Account" → "致命"
└── otherwise → "警告"
```

## 设计原则

1. **静默跳过 ≠ 安全**: 交易/指标函数返回 `NoneVal(0)` 会让 EA 做出错误决策，必须阻断
2. **永久盲区是设计决策**: `ObjectCreate` 等客户端 UI 函数在服务端无意义，静默跳过
3. **静态 + 运行时双重保障**: 静态分析在运行前预警，运行时追踪捕获实际触发
4. **不自动生成 stub**: 盲区是"待实现"信号，需要人工评估后实现，不自动猜测行为

## 文件索引

| 文件 | 职责 |
|------|------|
| `interp/analyze.go` | 静态分析，生成 `IRReport` |
| `interp/builtin_registry.go` | 已实现函数名列表（单一真相源） |
| `interp/builtins.go` | `callBuiltin` 分发 + `recordBlindSpot` 追踪 |
| `interp/exec.go` | `execBlock` panic 恢复 + `GetRuntimeBlindSpots` API |

## 测试

- `TestBuiltin_UnimplementedFunction` — 非致命盲区记录
- `TestBuiltin_FatalBlindSpotAborts` — 致命盲区 panic
- `TestRuntimeBlindSpots_API` — GetRuntimeBlindSpots 排序和 Reset
