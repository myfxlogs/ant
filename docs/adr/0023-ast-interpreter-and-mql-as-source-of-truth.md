# ADR-0023 · AST 分析 + Bytecode VM 执行 + MQL 源码为唯一真实来源

- **状态**：Accepted
- **日期**：2026-06-30
- **决策者**：人类负责人
- **关联 ADR**：ADR-0021（本 ADR 覆盖 G1 代码生成方式决策、§3.1 代码生成方式）、ADR-0022（盲区架构，本 ADR 保留其三层处理原则但实现方式变更）

## 1. 背景

### 1.1 当前架构的问题

ADR-0021 确立了 `tree-sitter CST → StrategyIntent IR → Go 代码生成 → WASM 解释器` 的执行链路。经过实际落地和深度审计，暴露出以下问题：

#### 问题 A：IR 是有损提取，静默丢弃逻辑

当前 recognizer 从 tree-sitter CST 中**只提取它认识的模式**，不认识的 MQL 代码被静默丢弃。tree-sitter 已经解析出了完整的语法树（CST），但 recognizer 只取了一部分。信息在 CST 里都有，是我们扔掉了。

这比"不支持某函数"更危险——用户以为 EA 完整转译了，实际部分逻辑被丢弃了，回测结果看似正常但基于不完整的逻辑。

#### 问题 B：Go 预览代码与实际执行脱节

系统同时维护两条路径：
- **展示路径**：`GenerateFromIR` 生成 Go 预览代码，存入 `strategy_templates.code`，用于参数提取（正则匹配 `ctx.Param()`）和前端展示
- **执行路径**：`CompileToIR` → `SerializeIR` → WASM interpreter，使用 MQL 源码

用户看到的是 Go 代码，实际执行的是 IR 解释器。两者共享 `CompileToIR` 入口但之后完全分叉。这导致：
- 模板存储 Go 代码但执行需要 MQL 源码，需要 `findImportedByGoCode()` 启发式匹配
- 参数提取依赖正则匹配 Go 代码而非从 IR 结构化提取
- 用户无法从 Go 预览代码判断 EA 是否被正确转译

#### 问题 C：WASM 沙箱是过度防护

WASM 沙箱的实际作用是在隔离环境中运行我们自己的 IR 解释器。被解释的不是任意用户代码，是我们的 IR 数据结构。WASM 带来的代价（预编译 17MB×2 .wasm 文件、IR 序列化/反序列化开销、host function 桥接、Docker build 复杂度、调试困难）远超其安全收益。

tree-sitter 解析 MQL 源码（cgo）在 WASM 沙箱外执行，因此 WASM 并不保护解析阶段。IR 解释器是纯 Go，内存安全，无需沙箱隔离。

### 1.2 安全模型分析

去掉 WASM 后，安全由三层 SDK 防护保障：

```
第一层: MQL → AST 提取 (tree-sitter 只解析已知语法，不认识的标记为 ERROR)
  → 恶意代码无法进入 AST

第二层: Bytecode VM (只能调用 SDK 定义的函数)
  → 没有文件、网络、系统调用的入口

第三层: SDK 函数 (所有交易操作都经过风控)
  → PlaceOrder → Gate.Evaluate() → 风险评估通过才提交
  → CloseOrder → 幂等性检查
  → 速率限制 → 防轰炸
```

用户写的 MQL 代码经过这三层后，能做的只有：读取 K 线数据、计算指标、发出交易信号。信号经过 SDK → 风控 gate → mthub → mtapi.io，恶意策略最多能"发很多信号"，但风控 gate 会拦住。

Bytecode VM + SDK 已经构成了一个比 WASM 更精确的沙箱。WASM 是通用沙箱（什么都防），我们的三层是专用沙箱（只允许策略该做的事）。

### 1.3 方案对比

| 维度 | IR（当前） | AST 树遍历 | AST+Bytecode（本决策） |
|------|----------|-----------|---------------------|
| 覆盖度 | 只能跑 recognizer 认识的模式 | tree-sitter 能解析的都能跑 | tree-sitter 能解析的都能跑 |
| 未知代码 | 静默丢弃 | 报错，用户可见 | 报错，用户可见 |
| 表达式 | 有限支持 | 任意嵌套表达式 | 任意嵌套表达式 |
| 复杂控制流 | 简化为模式 | 递归 visitor，break/continue 别扭 | 跳转指令，自然 |
| 用户自定义函数 | 需要特殊处理 Funcs 字段 | AST 中自然存在 | AST 中自然存在 |
| 参数提取 | 正则匹配 Go 代码 | 从 AST 直接提取 | 从 AST 直接提取 |
| 覆盖度报告 | 需额外分析（analyze.go） | 遍历 AST 自动生成 | 遍历 AST 自动生成 |
| 重复执行性能 | IR 解释器较快 | 每次递归整棵树 | **编译一次，线性扫描** |
| 安全隔离 | WASM 沙箱（过度防护） | recover()，栈溢出风险 | **指令计数器 + 显式数据栈，无栈溢出** |
| 实现复杂度 | recognizer + IR + gen.go + interp | AST visitor + interp | AST + 编译器 + VM（模块分化，各司其职） |

## 2. 决策

### D1: MQL 源码是唯一真实来源（Single Source of Truth）

`imported_strategies.source_code` 存储的 MQL 源码是策略的唯一真实来源。所有派生产物（参数、覆盖度报告、执行）都从 MQL 源码实时生成。

- **模板存储**：`strategy_templates.code` 存储 MQL 源码（不再是 Go 预览代码）
- **模板关联**：`strategy_templates` 新增 `strategy_id` 列，FK 指向 `imported_strategies.id`
- **Go 预览代码不再存储**：`GenerateFromIR` 不再用于运行时，仅保留为开发调试工具

### D2: AST 分析 + Bytecode VM 执行，替代 IR 提取 + WASM 沙箱

**原则**: 分析用 AST，执行用 Bytecode。不是二选一，而是各取所长：

```
                 AST                        Bytecode
适合做的事       ──────────────              ──────────────
                 • 遍历查找、结构化提取       • 线性执行、跳转控制流
                 • 模式匹配、报告生成         • 栈式计算、指令计数
不适合做的事     • 递归执行（性能差）         • 树遍历（结构信息已丢失）
                 • 控制流（break/continue 别扭）
                 • 栈溢出风险
```

执行链路：

```
MQL源码
  → tree-sitter 解析 → CST → AST
  ├─ 覆盖度报告 (遍历 AST，标记未实现的函数)
  ├─ 参数提取 (遍历 AST，找 extern/input 声明)
  └─ AST → 编译 → Bytecode (一次性，~300ms)
       └─ VM 执行 (每根 K 线线性扫描)
            ├─ 回测: VM + SimBroker (同进程)
            └─ 实盘: VM + mtapi.io gRPC (同进程)

隔离: 指令计数器 + context.WithTimeout + recover()
安全: SDK 三层防护 (不变)
```

**为什么 Bytecode 优于 AST 树遍历用于执行**：

1. **重复执行效率**——回测中 `OnBar` 执行几千到几万次。AST 每次递归整棵树，Bytecode 编译一次后续线性扫描。
2. **控制流**——`break`/`continue` 在 AST 递归 visitor 中需要特殊 error 类型或 context flag 传递。Bytecode 里就是一个跳转指令。
3. **安全隔离**——AST 递归有两个真实风险：深度嵌套表达式导致栈溢出（Go goroutine 栈有上限），死循环检测需要在每个 visitor 节点检查 context 容易遗漏。Bytecode VM 是扁平的 `for { switch op }`，指令计数器天然嵌入循环，栈是显式数据栈不是调用栈，不存在栈溢出。
4. **模块分化**——AST 专注分析（覆盖度、参数、盲区），Bytecode 专注执行。编译器（CST→Bytecode）和 AST 转换器工作量相当，VM 和 AST visitor 工作量相当。

### D3: 去掉 WASM 沙箱

WASM 沙箱、预编译 .wasm 文件、IR 序列化/反序列化全部移除。解释器在 Go 进程内直接执行，SDK 函数直接调用（同进程，无跨边界开销）。

### D4: 去掉 Go 代码生成（运行时）

`gen.go` / `gen_ir.go` 不再用于运行时。Go 代码生成仅保留为 `mql2go` CLI 的开发调试功能，不参与回测或实盘执行。

### D5: Agent 架构（未来方向）

LLM Agent 工作在 MQL 层面，具备 tool calling 能力：
- `read_strategy` — 获取 MQL + 覆盖度 + 参数 + 回测历史
- `modify_strategy` — 修改 MQL → 转译 → 验证 → 覆盖度 diff
- `run_backtest` — 执行回测 → 返回 metrics + trades + equity
- `analyze_result` — 自然语言分析回测结果
- `compare_runs` — 对比两次回测，输出差异
- `suggest_optimize` — 基于覆盖度 + 回测历史，主动建议优化

用户通过自然语言与 Agent 交互，Agent 修改 MQL 源码（而非 Go 代码），系统自动转译执行。

### D6: 盲区处理原则不变

ADR-0022 的三层盲区处理原则保留，但实现方式变更：
- **静态分析**：遍历 AST 而非 IR，标记未实现的函数调用
- **运行时追踪**：Bytecode VM 执行到未实现函数时，按严重度分级处理（致命→中止，警告→记录，永久盲区→跳过）
- **运行后上报**：API 返回运行时实际触发的盲区列表

关键改进：AST 方案下，tree-sitter 能解析的代码都在 AST 中，都会被解释器访问。tree-sitter 不能解析的会产生 ERROR 节点，我们检测到就报错。**不存在"能解析但静默丢弃"的灰色地带。**

## 3. 备选方案

| 方案 | 优点 | 缺点 | 否决理由 |
|------|------|------|----------|
| 继续用 IR + WASM | 已有实现 | 有损提取、Go预览脱节、WASM过度防护、静默丢弃 | 三个根本问题不可修复 |
| IR + 去掉 WASM | 减少复杂度 | IR 有损提取问题仍在 | 治标不治本 |
| AST + WASM | 完整覆盖 + 沙箱隔离 | WASM 开销仍在，AST 无需沙箱 | WASM 对 AST 解释器（纯Go）无额外安全价值 |
| AST 树遍历 + 进程内执行 | 完整覆盖、无静默丢失 | 每次递归整棵树、栈溢出风险、break/continue 别扭 | 执行效率和安全隔离不如 Bytecode |
| **AST 分析 + Bytecode VM（本决策）** | 完整覆盖、编译一次线性扫描、指令计数器隔离、显式数据栈 | 需写编译器+VM两个模块 | 采纳 |

## 4. 后果

### 4.1 正面

- **消除静默丢失**：tree-sitter 能解析的代码一定被执行，不能解析的报错给用户
- **消除 Go 预览与执行的脱节**：用户看到 MQL 源码，系统从 MQL 执行，无中间产物
- **消除 WASM 复杂度**：无预编译、无序列化、无 host function 桥接、Docker build 简化
- **参数提取更可靠**：从 AST 结构化提取，不再正则匹配 Go 代码
- **覆盖度报告免费**：遍历 AST 时自动生成
- **用户自定义函数天然支持**：AST 中自然存在，无需特殊处理
- **任意嵌套表达式**：AST 中自然存在，编译为字节码后由 VM 栈式求值，无需手动拆解
- **性能更好**：编译一次（~300ms），后续每次 `OnBar` 线性扫描。无序列化/反序列化、无 WASM 边界开销、无递归调用栈
- **安全隔离更强**：指令计数器 + 显式数据栈，无栈溢出风险。死循环检测天然嵌入 VM 主循环，不存在遗漏检查点的问题
- **控制流自然**：`break`/`continue`/`return` 编译为跳转指令，无需在 visitor 中传递特殊 error 或 context flag
- **Agent 友好**：MQL 是 Agent 的工作语言，LLM 懂 MQL4/MQL5

### 4.2 负面

- **需重写编译器+VM**：从 IR 解释器改为 AST→Bytecode 编译器 + VM（工作量约 2000 行，比纯 AST 方案多一个编译模块）
- **IR 相关代码退役**：recognizer.go、ir.go、gen.go、gen_ir.go、IR 序列化等不再用于运行时
- **WASM 相关代码退役**：wasm_executor.go、wasm_executor_interp.go、cmd/compile-wasm 等
- **多一个模块**：相比于纯 AST 方案多了编译步骤（但首次执行只需一次，模块分化带来更好的可测试性）

### 4.3 中性

- `mql2go` CLI 保留 Go 代码生成功能作为开发调试工具
- tree-sitter 仍然是 MQL 解析器（不变）
- SDK 接口不变（`Strategy`/`Context`/`Broker`/`IndicatorSet`/`BarSeries`）
- 风控 gate 不变（ADR-0020 D6）
- 盲区三层处理原则不变（ADR-0022），实现方式变更

## 5. 实施约束

### 5.1 数据模型变更

```sql
-- strategy_templates 新增 strategy_id 列
ALTER TABLE strategy_templates ADD COLUMN strategy_id UUID REFERENCES imported_strategies(id);
```

### 5.2 退役清单

| 模块 | 路径 | 处置 |
|------|------|------|
| IR 类型定义 | `tools/mql2go/ir.go` | 运行时不再使用，保留供 CLI 参考 |
| CST 模式匹配 | `tools/mql2go/recognizer.go` | 运行时不再使用 |
| Go 代码生成 | `tools/mql2go/gen.go`, `gen_ir.go` | 运行时不再使用，CLI 保留 |
| IR 序列化 | `tools/mql2go/interp/ir_serialize.go` | 退役 |
| WASM 执行器 | `internal/connect/strategy/wasm_executor.go` | 退役 |
| WASM interp 执行器 | `internal/connect/strategy/wasm_executor_interp.go` | 退役 |
| WASM 预编译 | `cmd/compile-wasm/` | 退役 |
| Go 代码执行器 | `internal/connect/strategy/go_runner.go` | 退役。原生 Go 策略不再支持——所有策略统一走 MQL→AST→Bytecode 路径。如未来需要支持 Go 原生策略，可独立规划，不影响本架构 |
| 启发式匹配 | `strategy_backtest_crud.go` 中的 `findImportedByGoCode` | 删除 |
| 转译检测 | `strategy_execution_handler.go` 中的 `isTranspiledMQL` | 删除 |
| Dockerfile WASM 编译步骤 | `backend/Dockerfile` line 33-36 | 删除 |

### 5.3 新增清单

| 模块 | 路径 | 职责 |
|------|------|------|
| CST → AST 转换 | `tools/mql2go/ast.go` | tree-sitter CST → 语义 AST |
| AST → Bytecode 编译器 | `tools/mql2go/compile.go` | AST 遍历 → 线性字节码（一次性，~300ms） |
| Bytecode VM | `tools/mql2go/vm.go` | `for { switch op }` + 显式数据栈 + 指令计数器 |
| SDK 函数绑定 | `tools/mql2go/builtins.go` | MQL 函数名 → SDK 方法映射（VM 通过 Go 函数直接调用） |
| AST 参数提取 | `tools/mql2go/ast_params.go` | 从 AST 提取 extern/input 声明 |
| AST 覆盖度分析 | `tools/mql2go/ast_coverage.go` | 遍历 AST 生成覆盖度报告 |
| 进程内执行入口 | `internal/connect/strategy/interp_runner.go` | 替代 WASM 执行入口，编排编译+VM |

### 5.4 安全约束

**MQL 预处理**（tree-sitter 解析前）：

| 指令 | 处理方式 |
|------|---------|
| `#include "file.mqh"` | 提取文件名，提示用户提供被引用文件内容，拼接后重新解析。最大递归深度 10 层，超过则报错 |
| `#define MACRO value` | 简单文本替换。**不支持函数式宏**（如 `#define MAX(a,b) ((a)>(b)?(a):(b))`）——这是永久限制。替代方案：Agent 在预处理阶段识别并展开函数式宏，或提示用户改写为普通函数 |
| `#ifdef/#ifndef` | 简单预处理器模拟条件判断 |
| `#property` | 忽略（不影响运行逻辑） |

**Bytecode 指令集设计**（stack-based）：

```
数据栈:   decimal.Decimal / int64 / string / bool 值
控制流:    JMP, JMP_IF_FALSE, JMP_IF_TRUE
函数调用:  CALL_BUILTIN (SDK 函数), CALL_USER (用户自定义函数)
函数边界:  ENTER_FUNC n (分配 n 个局部变量槽位), LEAVE_FUNC (释放局部变量，返回调用者)
事件入口:  ENTER_ONINIT, ENTER_ONBAR, ENTER_ONTICK, ENTER_ONTRADE
返回:      RETURN
算术:      ADD, SUB, MUL, DIV, MOD, NEG
比较:      EQ, NE, LT, LE, GT, GE
逻辑:      AND, OR, NOT
栈操作:    PUSH_CONST, PUSH_VAR, POP, DUP, SWAP
赋值:      STORE_VAR
```

**函数调用约定**：

```
CALL_USER addr    ; 调用 addr 处的用户函数。参数从栈顶弹出（逆序），返回值压入栈顶。
ENTER_FUNC n      ; 函数序言：分配 n 个局部变量槽位
LEAVE_FUNC        ; 函数尾声：释放局部变量，返回调用者
```

- 调用者按正序 PUSH 参数，CALL_USER 弹出 N 个参数到局部变量槽
- 被调用者执行完毕后，返回值 PUSH 到栈顶，LEAVE_FUNC 跳回调用者
- 全局变量通过 STORE_VAR/PUSH_VAR 的全局段寻址，局部变量通过帧偏移寻址

选择 stack-based 而非 register-based：实现更简单，指令更短，I/O 密集型场景下性能差异可忽略。

**Bytecode 缓存策略**：

Bytecode 缓存在内存中（首次执行时编译，同进程内复用）。跨重启持久化缓存可选（serialize bytecode to `[]byte` 存入 `imported_strategies` 或独立缓存表），后续迭代按需实现。同一 MQL 源码 + 参数集 → 同一 bytecode，可用源码 hash 做 cache key。

**VM 错误恢复粒度**（ADR-0022 三层原则落地到 VM 指令）：

| 级别 | 触发条件 | VM 行为 |
|------|---------|---------|
| 致命 (Fatal) | 交易/指标函数未实现（如 `OrderSend`、`iMA` 缺少实现） | VM 立即停止，返回错误 |
| 警告 (Warning) | 一般未实现函数（如 `TimeCurrent`） | 记录盲区到 `runtimeBlindSpots`，继续执行，返回 `NoneVal` |
| 跳过 (Skip) | 永久盲区（`ObjectCreate`、`FileOpen` 等） | 静默跳过，返回 `NoneVal`，无记录 |

**Result 类型**：

```go
// Result 对应回测或实盘的执行结果。
// 回测：包含 metrics、trades、equity curve。
// 实盘：包含信号列表，由 OMS 处理。
type Result struct {
    Signals     []*sdk.Signal      // 策略发出的交易信号
    Metrics     *BacktestMetrics   // 回测指标（仅回测）
    Trades      []*BacktestTrade   // 交易记录（仅回测）
    EquityCurve []EquityPoint      // 资金曲线（仅回测）
    BlindSpots  []RuntimeBlindSpot // 运行时触发的盲区
}
```

```go
func runStrategySafe(ctx context.Context, mqlSource string, ...) (result *Result, err error) {
    // 1. 输入限制
    if len(mqlSource) > 500_000 {
        return nil, errors.New("source too large")
    }
    // 2. 解析 → AST → 编译
    ast, err := mql2go.ParseToAST(mqlSource)
    if err != nil { return nil, err }
    bytecode, err := mql2go.Compile(ast)
    if err != nil { return nil, err }

    // 3. 超时（防死循环）+ panic 恢复（防 VM bug）
    ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
    defer cancel()
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("vm panicked: %v", r)
        }
    }()

    // 4. VM 执行
    vm := mql2go.NewVM(bytecode, sdkContext)
    vm.SetMaxInstructions(10_000_000) // ~10M 指令，正常 EA 不会超过
    return vm.Run(ctx)
}
```

VM 的 `Run` 主循环天然嵌入超时检查和指令限制：

```go
func (vm *VM) Run(ctx context.Context) (*Result, error) {
    for vm.pc < len(vm.code) {
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        default:
        }
        vm.step()
        vm.ticks++
        if vm.ticks > vm.maxInstructions {
            return nil, errors.New("strategy exceeded instruction limit")
        }
    }
    return vm.result, nil
}
```

与 AST 递归方案的关键区别：
- **指令计数器**嵌入循环体，无遗漏检查点的可能
- **显式数据栈**而非 Go 调用栈，深度由 VM 控制，无栈溢出风险
- **扁平的 `for { switch op }`**，无需在每个 visitor 节点手动检查 context

### 5.5 可靠性修复（同步实施）

| 问题 | 修复 |
|------|------|
| 模板无 FK | `strategy_templates.strategy_id` → `imported_strategies.id` |
| 回测无超时 | `context.WithTimeout`，默认 5 分钟 |
| 实盘 nil 指针 | executor 为 nil 时报错，不 panic |
| EquityCurve 无时间戳 | proto 加 `repeated int64 equity_times_ms`，VM 输出时填充 |
| Risk 未填充 | VM 计算最大回撤、风险因子，填充 `resp.Risk` |

## 6. 实施顺序

```
Phase 1 — 架构正确性 (P0)
  1. DB migration: strategy_templates 加 strategy_id 列
  2. 模板 save 时写入 strategy_id，code 字段改为存 MQL 源码
  3. 现有模板数据迁移：通过 strategy_id 关联 imported_strategies，回填 MQL 源码
  4. CST → AST 转换器
  5. AST → Bytecode 编译器
  6. Bytecode VM + SDK 函数绑定
  7. AST 参数提取 + 覆盖度报告
  8. 进程内执行入口（替代 WASM，编排 解析→编译→VM运行）
  9. 删除 findImportedByGoCode + isTranspiledMQL
  10. 前端编辑器显示 MQL，去掉 Go 预览

Phase 2 — 可靠性 (P1)
  11. 指令计数器上限 + context 超时（VM 内建，无需额外实现）
  12. 实盘 nil 指针修复
  13. EquityCurve 时间戳
  14. Risk/ExecutionAssumptions 填充
  15. 退役 WASM 相关代码 + Dockerfile 步骤

Phase 3 — Agent (P2)
  16. Agent tool 接口定义
  17. LLM 集成 (tool calling)
  18. 前端 Agent 聊天面板
  19. 覆盖度报告 UI
```

## 7. 验证方式

- **AST 完整性**：tree-sitter 能解析的 MQL 代码，AST 中每个节点都被编译器覆盖，转换为对应的字节码指令
- **无静默丢弃**：不认识的函数调用在编译阶段报错而非跳过（ERROR 节点检测 + 未实现函数报错）
- **参数提取**：从 AST 提取的参数与 MQL 源码中的 extern/input 声明一致
- **覆盖度报告**：报告中的盲区与实际运行时触发的盲区一致
- **回测一致性**：同一 MQL EA 在 Bytecode VM 和 MT Strategy Tester 的信号序列对齐
- **实盘一致性**：Bytecode VM 执行策略信号与回测结果一致（同码不变量）
- **风控不可绕过**：VM 的所有交易调用必经 `Gate.Evaluate`
- **安全隔离**：VM 无法访问文件/网络/系统调用（只能调 SDK 函数）
- **超时有效**：指令计数器 + context 超时双重保护，死循环策略被中止

## 8. 对 ADR-0021 的影响

本 ADR 覆盖 ADR-0021 的以下部分：
- **G1**（目标语言 = Go Strategy SDK）：改为 MQL 源码直接解释执行，不再转译为 Go 代码
- **§3.1**（代码生成方式决策）：Go 代码生成不再用于运行时，仅保留 CLI 开发调试
- **§6**（Go 侧待补齐）：iCustom 待规划项不变，其余已完成项的完成状态不变

保留不变的部分：
- **G2**（Python 服务退役）
- **G3**（Python 工具退役）
- **G5**（风控门不变）
- **G6**（Decimal 一致性）
- **G7**（行为对齐 harness 用 Go SimBroker 重写）
