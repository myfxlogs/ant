# MQL Interpreter in Go — 完整方案

> **供 GLM 5.2 参考后做最终架构判断。**
>
> 本方案不假设"解释器更优"，仅以同等严谨度给出完整设计方案，以便与纯翻译器路线做定量比较。方案基于现有代码基础设施（tree-sitter parser、SDK、WASM 运行时），估算了具体的工作量、性能影响和风险。

## 1. 架构概览

```
MQL 源码 (.mq4 / .mq5)
    │
    ▼  tree-sitter parse (复用 mql_lang.go)
CST (具体语法树)
    │
    ▼  MQLRuntime.Compile(cst) — 一次性预处理
┌──────────────────────────────────────────────┐
│  Executable                                   │
│    onInit:   []Statement   (变量初始化+EventSetTimer)  │
│    onBar:    []Statement   (指标+风控+出场+入场+循环)   │
│    onTick:   []Statement   (同上, tick 级)            │
│    onTimer:  []Statement   (定时器回调)               │
│    onDeinit: []Statement   (清理)                     │
│    globals:  map[string]Value  (全局变量/static 变量)   │
│    ordPool:  []OrderRecord     (MQL4 虚拟 order pool) │
│    posPool:  []PositionRecord  (MQL5 虚拟 position pool) │
└──────────────────────────────────────────────┘
    │
    ▼  WASM 沙箱内执行 (复用 live_runner + harness)
sdk.Strategy 接口
    │
    ▼  现有 pipeline (LiveStrategyRunner → Gate → OMS)
```

**关键设计决策**:
- **不生成 Go 代码**。MQL AST 直接求值，无 `go build` 链路。
- **复用现有基础设施**: tree-sitter 解析器、SDK 全部接口、WASM 运行时、风控 Gate、OMS。
- **实现 `sdk.Strategy` 接口**，与翻译器生成的策略完全同构——对执行层透明。
- **全链路使用 `decimal.Decimal`**（符合项目硬性规则）。MT Tester 使用 double 产生的精度差异在 parity 容差范围内（±2 pips），翻译器已验证可通过对齐校验。

## 2. 核心组件设计

### 2.1 可执行 IR（Compile 产物）

```
MQL 语句 → CST 节点 → Compile (一次性预遍历) → 纯 Go Expr/Statement 树
```

**关键设计：Compile 阶段将 CST 全部提取为纯 Go 数据结构。** 执行时不依赖 tree-sitter 运行时（无 cgo，无 `*sitter.Node`）。

```go
// Expr 是纯 Go 表达式树 — 不持有任何 tree-sitter 引用
type Expr struct {
    Kind     ExprKind
    Val      Value           // 字面量 (LiteralExpr)
    Name     string          // 变量名 / 常量名 (VarExpr / ConstExpr)
    Op       string          // 运算符 (BinaryExpr / UnaryExpr)
    Args     []Expr          // 子表达式 (BinaryExpr / CallExpr / ArrayExpr)
    Index    Expr            // 数组索引 (SubscriptExpr)
    Cond     *Expr           // 三元条件 (TernaryExpr)
    ThenExpr *Expr           // 三元 then 分支
    ElseExpr *Expr           // 三元 else 分支
}

type ExprKind uint8
const (
    ExprLiteral    ExprKind = iota // 字面量: Val
    ExprVar                        // 变量引用: Name
    ExprConst                      // 预定义常量: Name (OP_BUY, PRICE_CLOSE, ...)
    ExprBinary                     // 二元运算: Op + Args[0], Args[1]
    ExprUnary                      // 一元运算: Op + Args[0]
    ExprCall                       // 函数调用: Name + Args
    ExprSubscript                  // 数组/序列索引: Name[Index] → Close[1], High[shift]
    ExprField                      // 对象字段/方法: Args[0].Name.Args[1:]
    ExprTernary                    // 三元: Cond ? ThenExpr : ElseExpr
    ExprUpdate                     // i++, i--: Name + Op
    ExprAssignment                 // a = b: Name + Args[0]
)

// Statement 是纯 Go 语句树 — 不持有任何 tree-sitter 引用
type Statement struct {
    Kind        StatementKind
    Expr        *Expr          // ExprStmt / ReturnStmt / SwitchStmt (switch 表达式)
    Cond        *Expr          // IfStmt / WhileStmt 条件
    Init        *Statement     // ForStmt init
    Update      *Statement     // ForStmt update
    Body        []Statement    // 子语句块 (SwitchStmt: case 列表)
    ElseBody    []Statement    // IfStmt else 分支
    Cases       []SwitchCase   // SwitchStmt case 列表 (含 default)
}

type SwitchCase struct {
    Expr  *Expr           // case 值 (nil = default)
    Body  []Statement     // case 体
}

type StatementKind uint8
const (
    StmtExpr     StatementKind = iota // 表达式语句
    StmtIf                            // if/else
    StmtFor                           // for(init; cond; update)
    StmtWhile                         // while(cond)
    StmtReturn                        // return expr
    StmtBlock                         // { ... }
    StmtSwitch                        // switch/case/default
)
```

**内存占用**: 典型 EA 的 OnBar 有 ~50 个 statement，每个 ~120 字节 → ~6KB。表达式树 ~80 个节点，每个 ~80 字节 → ~6KB。总计 ~12KB。远小于 WASM 模块大小 (~1-5MB)。

**关键优势**: 
- 纯 Go 数据结构 → 可编译进 WASM（无 cgo，无 tree-sitter 依赖）
- 保持 WASM 沙箱隔离（不从 WASM 降级到 host 进程）
- 执行时无指针追踪，更好的缓存局部性

### 2.2 执行引擎（每 bar 执行）

```go
type Interpreter struct {
    stmts    []Statement          // 纯 Go IR，无 tree-sitter 依赖
    ctx      sdk.Context
    globals  map[string]Value
    locals   map[string]Value
    series   SeriesAccessor       // Close[i], High[i], Open[i], Low[i], Volume[i], Time[i]
    orderPool *MQL4OrderPool      // MQL4: OrdersTotal/OrderSelect 语义
    posPool   *MQL5PositionPool   // MQL5: PositionsTotal/PositionSelect 语义
    classes   map[string]*ClassInstance // MQL5 类实例 (CTrade, ...)
    lastErr   int
    lastErrSet bool
}

func (it *Interpreter) OnBar(ctx sdk.Context, timeframe string) (*sdk.Signal, error) {
    it.ctx = ctx
    it.locals = make(map[string]Value) // 每 bar 重置局部变量
    it.orderPool.Reset()               // 从 ctx.Broker().Positions() 重建
    it.posPool.Reset()
    
    for i := range it.stmts {
        sig, err := it.execStmt(&it.stmts[i])
        if err != nil { return nil, err }
        if sig != nil { return sig, nil }
    }
    return nil, nil
}

func (it *Interpreter) execStmt(stmt *Statement) (*sdk.Signal, error) {
    switch stmt.Kind {
    case StmtExpr:
        it.evalExpr(stmt.Expr)
    case StmtIf:
        if it.evalExpr(stmt.Cond).IsTrue() {
            return it.execBlock(stmt.Body)
        } else if len(stmt.ElseBody) > 0 {
            return it.execBlock(stmt.ElseBody)
        }
    case StmtFor:
        if stmt.Init != nil { it.execStmt(stmt.Init) }
        for {
            if stmt.Cond != nil && !it.evalExpr(stmt.Cond).IsTrue() { break }
            sig, err := it.execBlock(stmt.Body)
            if err != nil { return nil, err }
            if sig != nil { return sig, nil }
            if stmt.Update != nil { it.execStmt(stmt.Update) }
        }
    case StmtWhile:
        for it.evalExpr(stmt.Cond).IsTrue() {
            sig, err := it.execBlock(stmt.Body)
            if err != nil { return nil, err }
            if sig != nil { return sig, nil }
        }
    case StmtReturn:
        if stmt.Expr != nil {
            return it.evalReturnSignal(stmt.Expr)
        }
        return nil, nil
    case StmtBlock:
        return it.execBlock(stmt.Body)
    case StmtSwitch:
        switchVal := it.evalExpr(stmt.Expr)
        for _, c := range stmt.Cases {
            if c.Expr == nil { // default
                return it.execBlock(c.Body)
            }
            caseVal := it.evalExpr(c.Expr)
            if switchVal.Kind == ValDecimal && caseVal.Kind == ValDecimal {
                if switchVal.Decimal.Equal(caseVal.Decimal) { return it.execBlock(c.Body) }
            } else if switchVal.Int == caseVal.Int {
                return it.execBlock(c.Body)
            }
        }
    }
    return nil, nil
}
```

**与翻译器的关键区别**: `ForStmt` 直接执行，无论循环体包含什么——不需要"识别循环模式"。任何 `for(init; cond; update) { ... }` 都能正确执行。

### 2.3 表达式求值器

纯 Go Expr 树求值——无 tree-sitter 依赖，可编译进 WASM：

```go
func (it *Interpreter) evalExpr(e *Expr) Value {
    switch e.Kind {
    case ExprLiteral:
        return e.Val
    case ExprVar:
        return it.getVar(e.Name)
    case ExprConst:
        return it.lookupConstant(e.Name)   // OP_BUY=0, PRICE_CLOSE=0, ...
    case ExprBinary:
        left := it.evalExpr(&e.Args[0])
        right := it.evalExpr(&e.Args[1])
        return it.applyOp(left, e.Op, right)
    case ExprUnary:
        val := it.evalExpr(&e.Args[0])
        return it.applyUnary(val, e.Op)    // -x, !x
    case ExprCall:
        return it.callBuiltin(e.Name, e.Args)
    case ExprSubscript:
        return it.evalSeriesAccess(e.Name, e.Index) // Close[1], High[shift]
    case ExprField:
        obj := it.evalExpr(&e.Args[0])
        return it.evalField(obj, e.Name, e.Args[1:]) // CTrade.Buy(...)
    case ExprTernary:
        if it.evalExpr(e.Cond).IsTrue() {
            return it.evalExpr(e.ThenExpr)
        }
        return it.evalExpr(e.ElseExpr)
    case ExprAssignment:
        val := it.evalExpr(&e.Args[0])
        it.setVar(e.Name, val)
        return val
    case ExprUpdate:
        val := it.getVar(e.Name)
        if e.Op == "++" { val = val.Add(decimal.NewFromInt(1)) } 
        else { val = val.Sub(decimal.NewFromInt(1)) }
        it.setVar(e.Name, val)
        return val
    }
    return Value{Kind: ValNone}
}
```

**不需要识别的模式**: `Close[1] > Close[2]` — 翻译器需要专用的识别器。解释器不需要：`ExprSubscript("Close", 1)` → `it.series.Close(1)` → `decimal.Decimal`，`ExprBinary(">")` → `.GreaterThan()`。任何表达式组合都能正确求值。

**不需要识别的模式**:
- `Close[1] > Close[2]` → 翻译器需要识别这个模式生成 `ctx.Bars().Close(1).GreaterThan(ctx.Bars().Close(2))`。解释器直接求值 `Close[1]` = `it.series.Close(1)`, `Close[2]` = `it.series.Close(2)`, `>` = `left.Double.GreaterThan(right.Double)`。
- `OrderSelect(i, SELECT_BY_POS) && OrderMagicNumber() == magic` → 翻译器需要识别这个模式来生成 `ctx.Broker().Positions()` 循环。解释器直接执行 `OrderSelect` = `it.orderPool.Select(i)`, `OrderMagicNumber` = `it.orderPool.MagicNumber()`。

### 2.4 类型系统

```go
type Value struct {
    Kind     ValueKind
    Int      int32
    Decimal  decimal.Decimal // 所有数值统一用 decimal.Decimal
    Str      string
    Bool     bool
    Array    []Value
    Datetime int64           // unix timestamp (不是价格，可以用 int64)
    Class    *ClassInstance  // MQL5 类/结构体实例 (ValClass)
}

type ValueKind uint8
const (
    ValNone     ValueKind = iota
    ValInt
    ValDecimal               // 替代 ValDouble — 符合 CLAUDE.md 禁 float64 规则
    ValBool
    ValString
    ValDatetime
    ValColor
    ValArray
    ValClass               // MQL5 类/结构体实例 (CTrade, MqlTradeRequest, 用户自定义 struct)
)
```

**全链路 `decimal.Decimal`**。MQL 的 `double` 虽然是 IEEE 754 64-bit，但项目硬性规则禁止 `float64` 用于价格计算。解释器内部统一使用 `decimal.Decimal`，与 SDK 保持一致。

**隐式类型转换**（遵循 MQL 语义）:
- `int + decimal` → `decimal`（`decimal.NewFromInt(intVal).Add(decimalVal)`）
- `decimal` 用于 `if` 条件 → `!decimalVal.IsZero()` → `true`
- `int` 用于 `if` 条件 → `int != 0` → `true`

**与 MT Tester 的精度差异**:
- MT Tester 使用 `double`（float64），解释器使用 `decimal.Decimal`
- 差异在对齐校验容差范围内（`DefaultParityConfig`: 价格 ±2 pips, 手数 ±0.01 lot）
- **翻译器已证明 `decimal.Decimal` 路径可以通过 parity test** — 解释器同样可以

**与 SDK 边界的转换**:
- `ctx.Bars().Close(0)` 返回 `decimal.Decimal` → 直接使用，零转换
- Signal 字段: `decimal.Decimal` → `.String()` 在 proto 边界序列化
- 指标计算: SDK `IndicatorSet` 方法全部返回 `decimal.Decimal`

### 2.5 内置函数表

全部需要实现 ~80 个函数，分四层：

**Layer 1 — 交易核心 (22 个，MQL4+MQL5)**:
```
OrderSend OrderClose OrderModify OrderDelete OrderSelect OrdersTotal
OrderTicket OrderLots OrderType OrderMagicNumber OrderOpenPrice 
OrderStopLoss OrderTakeProfit OrderProfit OrderCommission OrderSwap
OrderComment OrderSymbol OrderCloseTime OrderClosePrice

PositionSelect PositionsTotal PositionGetTicket PositionGetDouble 
PositionGetInteger PositionGetString
```

**Layer 2 — 市场数据+指标 (28 个)**:
```
iMA iRSI iMACD iATR iBollinger iStochastic iCCI iADX iMFI iOBV iSAR
iStdDev iMomentum iWPR iEnvelopes iAlligator iIchimoku iDeMarker
iOsMA iRVI iForce iFractals iGator iAC iAD iAO iBearsPower iBullsPower
Ask Bid Point Digits Spread Symbol Period
MarketInfo SymbolInfoDouble SymbolInfoInteger
AccountBalance AccountEquity AccountFreeMargin AccountMargin
```

**Layer 3 — 工具函数 (30 个)**:
```
MathAbs MathMax MathMin MathRound MathFloor MathCeil MathSqrt MathPow MathLog MathExp MathSin MathCos MathTan MathArcsin MathArccos MathArctan MathRand MathSrand
StringConcatenate StringFind StringLen StringSubstr StringReplace StringToUpper StringToLower
ArrayResize ArraySize ArrayCopy ArrayFill ArraySort ArrayMaximum ArrayMinimum ArrayRange
```

**Layer 4 — 平台函数 (10 个)**:
```
TimeCurrent TimeLocal TimeDayOfWeek TimeHour TimeMinute TimeSeconds
Print Alert Comment GetLastError SetLastError
EventSetTimer EventKillTimer
IsTradeAllowed IsExpertEnabled
```

**调度表**:
```go
var builtins = map[string]func(*Interpreter, []Value) (Value, error){
    // Layer 1 — 直接映射到 SDK
    "OrderSend":     ord.OrderSend,
    "OrderClose":    ord.OrderClose,
    "OrderSelect":   ord.OrderSelect,
    "OrdersTotal":   ord.OrdersTotal,
    
    // Layer 2 — 直接映射到 SDK.Indicators()
    "iRSI":          ind.RSI,
    "iMA":           ind.MA,
    "Ask":           mkt.Ask,
    "Bid":           mkt.Bid,
    
    // Layer 3 — decimal.Decimal 实现 (不使用 math.Abs/math.Max — 它们返回 float64)
    "MathAbs":       func(it *Interpreter, args []Value) (Value, error) { return Value{Kind: ValDecimal, Decimal: args[0].Decimal.Abs()}, nil },
    "MathMax":       func(it *Interpreter, args []Value) (Value, error) {
        if args[0].Decimal.GreaterThan(args[1].Decimal) { return args[0], nil }
        return args[1], nil
    },
    
    // Layer 4 — 混合
    "TimeCurrent":   time.Now,
    "Print":         func(it *Interpreter, args []Value) (Value, error) {
        it.ctx.Log(fmt.Sprint(args...))
        return Value{}, nil
    },
}
```

**函数增量实现**: 对具体 EA 启动时扫描使用的内置函数，按需验证是否已实现。未实现的函数 → 返回明确错误（不是静默跳过），在实盘运行前阻断。

### 2.6 MQL4 Order Pool（虚拟订单池）

MQL4 的 `OrdersTotal()` + `OrderSelect(i, SELECT_BY_POS, MODE_TRADES)` 模型需要一个虚拟 order pool：

```go
type MQL4OrderPool struct {
    orders  []OrderRecord
    current *OrderRecord  // 当前选中的订单 (OrderSelect 后的状态)
    total   int           // OrdersTotal 返回值
}

type OrderRecord struct {
    Ticket       int64
    Symbol       string
    Type         int32     // OP_BUY=0, OP_SELL=1, OP_BUYLIMIT=2, ...
    Lots         decimal.Decimal
    OpenPrice    decimal.Decimal
    StopLoss     decimal.Decimal
    TakeProfit   decimal.Decimal
    ClosePrice   decimal.Decimal
    OpenTime     int64
    CloseTime    int64
    Profit       decimal.Decimal
    Commission   decimal.Decimal
    Swap         decimal.Decimal
    MagicNumber  int32
    Comment      string
}

func (p *MQL4OrderPool) Reset(ctx sdk.Context) {
    // 从 SDK 重建订单池
    posList := ctx.Broker().Positions(0)     // 全部持仓
    ordList := ctx.Broker().Orders(0)        // 全部挂单
    
    p.orders = make([]OrderRecord, 0, len(posList)+len(ordList))
    for _, pos := range posList {
        p.orders = append(p.orders, posToRecord(pos))
    }
    for _, ord := range ordList {
        p.orders = append(p.orders, pendingToRecord(ord))
    }
    p.total = len(p.orders)
    p.current = nil
}

func (p *MQL4OrderPool) Select(index int, mode int) bool {
    if index < 0 || index >= p.total {
        return false
    }
    p.current = &p.orders[index]
    return true
}

func (p *MQL4OrderPool) OrderTicket() int64 {
    if p.current == nil { return 0 }
    return p.current.Ticket
}
// ... 其他 Order*() 函数类似
```

**行为等价于 MQL4**:
- `OrderSelect(i, SELECT_BY_POS)` → 选中第 i 个
- `OrderSelect(ticket, SELECT_BY_TICKET)` → 遍历查找 ticket
- `OrdersTotal()` → 返回 pool 大小
- `OrderMagicNumber()` → 返回当前选中订单的 magic

MQL5 的 Position pool 类似实现。

### 2.7 时间序列访问器

MQL 的 `Close[1]` / `High[i]` 是逆序索引。BT 已有 `sdk.BarSeries` 提供完全相同的语义：

```go
type SeriesAccessor struct {
    bars sdk.BarSeries
}

func (s *SeriesAccessor) Close(shift int) Value {
    if shift < 0 || shift >= s.bars.Len() {
        return Value{Kind: ValNone}
    }
    d := s.bars.Close(shift)
    return Value{Kind: ValDecimal, Decimal: d}
}

func (s *SeriesAccessor) Open(shift int) Value { ... }
func (s *SeriesAccessor) High(shift int) Value { ... }
func (s *SeriesAccessor) Low(shift int) Value  { ... }
func (s *SeriesAccessor) Volume(shift int) Value { ... }
func (s *SeriesAccessor) Time(shift int) Value { ... }
```

### 2.8 MQL5 类/结构体 + 预处理器

**MQL5 类系统**。MQL5 EA 大量使用 `CTrade` 类：

```cpp
CTrade trade;
trade.SetExpertMagicNumber(12345);
trade.Buy(lotSize, _Symbol, 0, sl, tp, "comment");
trade.PositionClose(ticket, 20);
```

翻译器识别 `CTrade.Buy/Sell/PositionClose` 作为命名方法调用。解释器需要在 Compile 阶段解析类实例和方法分发：

```go
type ClassInstance struct {
    Name   string              // "CTrade"
    Fields map[string]Value    // magic, slippage, etc.
}

func (it *Interpreter) evalField(obj Value, method string, args []Expr) Value {
    switch obj.Kind {
    case ValClass:
        switch obj.ClassName {
        case "CTrade":
            switch method {
            case "Buy":
                return it.builtinCTradeBuy(args)
            case "Sell":
                return it.builtinCTradeSell(args)
            case "PositionClose":
                return it.builtinCTradePositionClose(args)
            case "PositionModify":
                return it.builtinCTradePositionModify(args)
            case "SetExpertMagicNumber":
                obj.Fields["magic"] = it.evalExpr(&args[0])
                return Value{}
            // ... BuyLimit, SellLimit, BuyStop, SellStop, etc.
            }
        }
    }
    return Value{Kind: ValNone}
}
```

**MQL5 结构体**。`MqlTradeRequest` 和 `MqlTradeResult`：

```cpp
MqlTradeRequest req = {};
req.action = TRADE_ACTION_DEAL;
req.symbol = _Symbol;
req.volume = 0.1;
req.type = ORDER_TYPE_BUY;
req.price = SymbolInfoDouble(_Symbol, SYMBOL_ASK);
OrderSend(req, result);
```

Compile 阶段识别 `MqlTradeRequest` 类型声明并创建字段映射。执行时：字段赋值 → 构建 `MqlTradeRequest` → `OrderSend(req)` → 映射到 `ctx.Broker().OrderSend(sdkReq)`。

**用户自定义 struct**。MQL5 允许用户定义 struct：
```cpp
struct MyConfig {
    int magic;
    double lotSize;
};
MyConfig cfg;
cfg.magic = 12345;
cfg.lotSize = 0.1;
```

Compile 阶段解析 struct 声明，创建字段映射表。`ClassInstance` 通用化处理所有 struct 类型：
```go
type ClassInstance struct {
    Name   string              // "CTrade" / "MqlTradeRequest" / "MyConfig"
    Fields map[string]Value    // 所有字段统一用 Value 存储
}
```

字段读写通过 `evalField` 统一分发：
```go
func (it *Interpreter) evalField(obj Value, field string, args []Expr) Value {
    if obj.Kind != ValClass || obj.Class == nil {
        return Value{Kind: ValNone}
    }
    // 写入: cfg.magic = 12345
    if len(args) == 1 && /* is assignment context */ {
        obj.Class.Fields[field] = it.evalExpr(&args[0])
        return obj.Class.Fields[field]
    }
    // 读取: cfg.magic
    if val, ok := obj.Class.Fields[field]; ok {
        return val
    }
    // 内置类方法分发 (CTrade.Buy, CTrade.Sell, ...)
    return it.dispatchClassMethod(obj.Class, field, args)
}
```

用户自定义 struct 的字段操作（赋值/读取）不需要 hardcoded — 通过通用 `Fields map[string]Value` 处理。只有内置类（CTrade）的方法调用需要 hardcoded 分发。

**预处理器**。MQL 的 `#include` / `#define` / `#property`:

```cpp
#include <Trade/Trade.mqh>   // 引入 CTrade 类定义
#define MAGIC 12345
#property strict
```

**处理策略**: 复用 tree-sitter MQL grammar 的能力。tree-sitter 在 parse 之前运行预处理器（通过外部命令或内联逻辑），展开 `#include` 文件和 `#define` 宏。这发生在 Compile 阶段（host 进程内），不进入 WASM。

```go
func preprocess(source string, includeDirs []string) (string, error) {
    // 1. 展开 #include — 递归读取文件并内联
    // 2. 展开 #define — 文本替换
    // 3. 移除 #property — 不进入执行逻辑
    return expanded, nil
}
```

预处理器与翻译器共享——两者都需要处理 `#include` / `#define`。不需要实现两次。

## 3. 与现有管线的集成

### 3.1 Harness 改造

解释器策略实现 `sdk.Strategy` 接口，在 WASM harness 中运行：

```go
// 新 harness 模板: generateInterpreterHarness (backtest_harness.go)
func generateInterpreterHarness(strategyCode string) string {
    return fmt.Sprintf(`package main
    import (
        "anttrader/strategy/runner"
        "anttrader/strategy/sdk"
        "anttrader/tools/mql2go/interp"  // 新包
    )
    func main() {
        runtime := interp.NewRuntime()
        if err := runtime.Compile(%q); err != nil {
            os.Exit(1)
        }
        strategy := runtime.Strategy()
        r := runner.New(cfg)
        r.SetStrategy(strategy)
        // ... 同现有 harness
    }`, strategyCode)
}
```

**但更简洁的方案**: 解释器不在 harness 中运行——它在 host 进程内运行：

```go
// LiveStrategyRunner 直接支持两种策略类型
func (s *StrategyExecutionServer) RunLiveStrategy(ctx, cfg) {
    var strategy sdk.Strategy
    
    if cfg.InterpreterMode {
        runtime := interp.NewRuntime()
        runtime.Compile(cfg.Code)
        strategy = runtime.Strategy()
    } else {
        // WASM session (现有路径)
        session = NewLiveSession(wasm, cfg.Code, log)
    }
    
    // strategy 统一执行
    for bar := range barCh {
        sig, _ := strategy.OnBar(ctx, timeframe)
        dispatch(sig)
    }
}
```

**但这种方案失去了 WASM 沙箱隔离**。解释器运行在 host 进程内，策略 bug 可能导致 panic。

**选定方案 B（WASM 沙箱）**。纯 Go IR 已消除 tree-sitter 运行时依赖，方案 B 的唯一技术障碍已解决。解释器编译为 WASM 在 wazero sandbox 内运行，与翻译器路径保持同构的沙箱隔离。

~~方案 A（host 进程内运行）~~ 已排除 — 失去沙箱隔离是架构降级，`recover()` 不能覆盖所有风险（goroutine panic、map 并发读写、nil pointer 在特定路径下仍危险）。

### 3.2 翻译器/解释器路径选择

```
RunLiveStrategy(cfg, code)
    │
    ▼
  尝试 mql2go.Analyze(code)
    │
    ├─ BlindSpots[] 全部 severe="信息" → 翻译器路径 (WASM)
    │
    └─ BlindSpots[] 包含 "警告" 或 "致命" → 解释器路径
            │
            ▼
          编译: interp.Compile(code) → Executable
          执行: strategy.OnBar() → AST 求值 → Signal
```

翻译器覆盖率通过扩展 recognizer 持续提升，解释器覆盖率随着翻译器改进而缩小使用范围。

### 3.3 对齐校验双重闭环

```
MQL EA
    ├──→ MT Strategy Tester (原生执行)
    │        │
    │        ▼ HTML 报告 → ParseMTReport
    │
    ├──→ 翻译器 → Go 回测 → SimBroker → Trades[]
    │        │
    │        ▼
    │     CompareParity ←───── 闭环 1: 翻译器对齐
    │
    └──→ 解释器 → Go 回测 → SimBroker → Trades[]
             │
             ▼
          CompareParity ←───── 闭环 2: 解释器对齐
                 │
                 ▼
          闭环 1 vs 闭环 2 比较
          → 如果解释器对齐但翻译器不对齐 → 翻译器 bug
          → 如果翻译器对齐但解释器不对齐 → 解释器 bug
          → 如果两者都不对齐 → 基准问题（数据/SimBroker 差异）
```

这提供了**独立验证**: 两个不同实现产生相同结果 = 高置信度。

## 4. 工作量估算

### 4.1 代码行数

| 组件 | 文件 | 估算 LOC | 复杂度 |
|------|------|---------|--------|
| **Value 类型 + 转换** | `interp/value.go` | 300 | 低 |
| **Expr/Statement IR 类型** | `interp/ir.go` | 200 | 低 |
| **Compile (CST→纯 Go IR)** | `interp/compile.go` | 600 | 高 |
| **表达式求值器** | `interp/eval.go` | 350 | 中 |
| **语句执行器** | `interp/exec.go` | 300 | 中 |
| **内置函数表 (Layer 1)** | `interp/builtin_trade.go` | 400 | 中 |
| **内置函数表 (Layer 2)** | `interp/builtin_market.go` | 500 | 低 |
| **内置函数表 (Layer 3+4)** | `interp/builtin_util.go` | 400 | 低 |
| **MQL5 类/结构体支持** | `interp/class_mql5.go` | 500 | 高 |
| **预处理器** | `interp/preprocess.go` | 150 | 中 |
| **MQL4 Order Pool** | `interp/orderpool_mql4.go` | 350 | 中 |
| **MQL5 Position Pool** | `interp/pospool_mql5.go` | 300 | 中 |
| **sdk.Strategy 适配器** | `interp/strategy.go` | 150 | 低 |
| **时间序列访问器** | `interp/series.go` | 100 | 低 |
| **Harness 模板** | `backtest_harness.go` | 100 | 低 |
| **测试 (10+ EA)** | `interp/*_test.go` | 800 | 中 |
| **总计** | | **5500** | |

### 4.2 分阶段计划

| 阶段 | 内容 | 工时 | 产出 |
|------|------|------|------|
| **Phase 1**: 核心 IR + 引擎 | Expr/Statement IR 类型, Compile(CST→纯Go), 表达式求值器, 语句执行器 | 3-4 天 | 能执行简单 MQL 表达式和 if/for 控制流 |
| **Phase 2**: 内置函数 Layer 1+2 | 交易函数, 市场数据, 指标 | 2 天 | 能执行完整 MQL 交易逻辑 |
| **Phase 3**: MQL5 类 + 预处理器 | CTrade/MqlTradeRequest 类, 预处理器 | 2 天 | MQL5 EA 完整支持 |
| **Phase 4**: Order/Position Pool | MQL4 OrderPool, MQL5 PositionPool | 1 天 | OrdersTotal/OrderSelect 模式兼容 |
| **Phase 5**: 集成 + WASM 验证 | sdk.Strategy 适配, harness, 纯 Go IR 在 WASM 内运行验证 | 1 天 | 端到端可跑, 沙箱隔离确认 |
| **Phase 6**: 对齐校验闭环 2 | ParityTest(解释器 vs MT Tester) | 2 天 | 验证正确性 + 交叉验证 |
| **Phase 7**: 工具函数 + 容错 | 内置函数 Layer 3+4, 未实现函数报错, 测试 | 1-2 天 | 生产就绪 |
| **总计** | | **12-14 天** | |

## 5. 性能分析

### 5.1 每 bar 延迟分解

| 操作 | 估算时间 | 说明 |
|------|---------|------|
| AST 节点检索 (50 个 statement × ~5 个子节点) | ~5µs | 纯内存访问 |
| 表达式求值 (20 个 expression × ~5 个节点) | ~10µs | switch 分发 + 递归求值 |
| 内置函数调用 (5 个调用/bar) | ~5µs | SDK 方法直接调用 |
| Order pool 重建 (`Reset()`) | ~10µs | 遍历 `ctx.Broker().Positions()` |
| **总计** | **~30µs/bar** | |

对比:
- **翻译器 (WASM 原生)**: ~1-5µs/bar
- **解释器 (AST 遍历)**: ~30µs/bar
- **MT 原生解释器 (C++)**: ~10-20µs/bar (估算, 闭源)

**差距**: 30µs vs 1µs。对于 M1 bar (60秒间隔) → 无关紧要。对于 backtest 扫描 (10⁴ bars) → ~300ms vs ~10ms → 显著但可接受。

### 5.2 最坏情况分析

极端复杂 EA: 200 个 statement, 100 个 expression, 20 个内置函数调用 → ~200µs/bar。仍然远小于 1ms 的 bar 通道延迟。

**性能瓶颈不在解释器，在 MT 网关的网络 I/O (~1-10ms) 和 OMS 持久化 (~5-50ms)。**

## 6. 风险评估

| 风险 | 概率 | 影响 | 缓解 |
|------|------|------|------|
| **MQL 语义与 MT 不一致** | 中 | 高: 解释器产生错误信号 | 对齐校验闭环 2: 解释器 vs MT Tester 交易序列比对 |
| **解释器 panic 导致进程崩溃** | 低 | 中: 影响其他策略 | `recover()` 在 OnBar 入口, 单个策略异常不影响整体 |
| **内置函数覆盖不全** | 中 | 中: 部分 EA 无法运行 | 启动时扫描: 未实现的函数 → 明确拒绝（不是静默跳过） |
| **解释器和翻译器双重维护成本** | 高 | 中: SDK 变更需同步 | SDK 接口稳定 (非频繁变更), 解释器直接调用 SDK (无间接层) |
| **解释器路径成为技术债** | 中 | 中: 长期被翻译器路径取代 | 解释器用途明确: 翻译器盲点的兜底。如果翻译器覆盖率到 99%+, 可废弃解释器 |
| **解释器性能不达标 (backtest)** | 低 | 低: backtest 可 fallback 翻译器 | 解释器优先用于实盘, backtest 优先使用翻译器 (同 EA 双路径) |

## 7. 与纯翻译器路线的定量比较

| 维度 | 纯翻译器 | 翻译器 + 解释器 (修正后方案) |
|------|----------|---------------------------|
| **覆盖率上限** | 取决于 recognizer 覆盖（盲点收敛性待验证） | 100%（解释器兜底 — 所有 CST 节点类型都可求值） |
| **新增 EA 模式成本** | 新 recognizer（每个模式 ~50-200 LOC） | 已覆盖的 CST 节点类型：零成本。新节点类型：需扩展 Expr/Statement（罕见，~50 LOC） |
| **盲点处理** | 启动时拒绝（blind spot gate） | 解释器 fallback（无拒绝） |
| **开发成本 (一次性)** | 持续维护 recognizer | +5500 LOC（约 12-14 天，含 MQL5 类/预处理器/纯 Go IR/数组函数/switch/用户自定义 struct） |
| **维护成本 (持续)** | SDK 变更：生成器同步 (~100 LOC/次) | SDK 变更：builtin 函数映射同步 (~100 LOC/次，等价） |
| **性能 (每条 bar)** | 原生 WASM ~1-5µs | 解释器 WASM ~30µs（delta: ~25µs，bar 间隔无关） |
| **对齐校验** | 1 个闭环（翻译器 vs MT） | 2 个闭环（翻译器 vs MT + 解释器 vs MT → 交叉验证） |
| **语义调试** | 读生成的 Go 代码 | 运行时日志 + Expr/Statement dump |
| **沙箱隔离** | WASM（全隔离） | WASM（全隔离 — 纯 Go IR 可编译进 wasm） |
| **`.ex4`/`.ex5` 支持** | ❌ 无源码 | ❌ 同样需要源码 |

## 8. 结论

在已确定的 **100% EA 覆盖率** 目标下，解释器不是可选组件——它是必需的兜底层。

**翻译器负责性能，解释器负责覆盖率。** 翻译器能识别的模式走原生 Go 路径。识别不了的走解释器路径——无需 blind spot gate 拒绝运行。

**修正后的五个关键设计保证了解释器的可行性和项目合规性：**
1. **全链路 `decimal.Decimal`** — 符合 AGENTS.md 强制规则，对齐校验容差已验证
2. **纯 Go IR (Expr/Statement)** — 无 tree-sitter 运行时依赖，可编译进 WASM，保持沙箱隔离
3. **MQL5 类/结构体 + 预处理器** — 覆盖 CTrade 方法调用、MqlTradeRequest 模式、用户自定义 struct、#include/#define
4. **switch/case/default 语句** — `SwitchCase` 结构体 + `StmtSwitch` 执行分支
5. **数组操作函数** — ArrayResize/Size/Copy/Fill/Sort/Maximum/Minimum/Range

**开发成本**：5500 LOC，约 12-14 天。翻译器扩展（补齐已知缺口）+ 解释器实现可并行推进。

**最终架构**：
```
MQL 源码
    │
    ├──→ mql2go 翻译器 (盲点 ≤ "信息")
    │        └──→ Go 源码 → WASM 原生执行 (~1µs/bar)
    │
    └──→ MQL 解释器 (盲点 ≥ "警告")
             └──→ 纯 Go IR → WASM 解释执行 (~30µs/bar)
                      │
              统一实现 sdk.Strategy
                      │
              现有 pipeline (Gate → OMS)
```
