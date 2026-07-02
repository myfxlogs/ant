# ADR-0024 · Agent-Native 策略平台 — 双前端编译 + Python Agent 层

- **状态**：Accepted
- **日期**：2026-07-02
- **决策者**：人类负责人
- **关联 ADR**：ADR-0021（策略运行时迁移，部分保留）、ADR-0022（盲区架构，本 ADR 扩展）、ADR-0023（Bytecode VM，本 ADR 复用并扩展）

---

## 1. 背景

### 1.1 当前系统状态

ADR-0023 确立了 `MQL → tree-sitter CST → AST (interp.IR) → Bytecode → VM` 的执行管线。该管线是确定性的、快速的、可测试的，覆盖了 MQL4/MQL5 的核心交易 API（~120 个内置函数）和 30+ 技术指标。

**已验证的能力：**
- MQL4 `OrderSend` / `OrderClose` / `OrderModify` / `OrderSelect` 循环
- MQL5 `CTrade.Buy/Sell/PositionClose/PositionModify`
- 30+ 指标（iMA/iRSI/iATR/iBands/iMACD/iStochastic/iCCI/iADX 等）
- 覆盖率分析（`AnalyzeCoverage`）+ 盲区分级（致命/警告/永久盲区）
- SimBroker 回测 + LiveRunner 实盘

**已确认的局限：**
- ~40% 真实 EA 因盲区（iCustom / .mqh 库 / WebRequest / DLL / MQL5 原生 OrderSend）无法正确运行
- LLM 仅用于代码修复（`code_assist`），未发挥 Agent 级能力
- 无策略生成路径（非程序员用户无法使用平台）
- 无自我学习/进化能力

### 1.2 核心矛盾

```
用户期望：  上传任意 EA → 直接跑 / 用自然语言描述 → 自动生成策略
现实：     40% EA 遇到盲区 → 策略行为错误 / 无生成路径
```

### 1.3 范式判断

**LLM pass（单次变换）不足以解决上述矛盾。** 需要的是 Agent（自主循环：观察 → 思考 → 行动 → 学习 → 进化）。

Agent 的核心特征：
- 自主决定下一步做什么（不是管线硬编码的调用）
- 写代码 → 运行 → 看结果 → 修改 → 再运行（迭代循环）
- 积累经验，改进策略（跨策略学习）
- 分析回测结果，发现模式（pandas/numpy/optuna）

**Agent 的思考环境必须是 Python**——pandas/numpy/optuna/pgvector/LLM 框架生态全在 Python。Go 的 Agent 生态为零。

**Agent 的执行环境必须是 Go VM**——回测在 Agent 循环的内层，per-bar RPC 不可行（200x 性能差距）。Agent 每次迭代只有一次 RPC 往返（提交源码 + 取回结果），回测在 Go VM 内执行（100ms 级）。

---

## 2. 决策

### D1: 三层架构

```
┌─────────────────────────────────────────────────────────────┐
│  Python Agent 层 (新建)                                      │
│                                                              │
│  策略生成 Agent │ 盲区桥接 Agent │ 策略进化 Agent             │
│  知识库 (pgvector) │ pandas │ numpy │ optuna │ LLM           │
│                                                              │
│  职责: 生成策略 / 分析结果 / 优化参数 / 学习进化              │
│  不做: 执行策略 / 访问 MT / 撮合                              │
└──────────────────────────┬──────────────────────────────────┘
                           │ ConnectRPC + SSE (protobuf)
                           │ Agent 提交源码 → Go 编译执行 → 结果返回
┌──────────────────────────┴──────────────────────────────────┐
│  Go API 层 (现有，扩展)                                      │
│                                                              │
│  ConnectRPC handlers │ 策略管理 │ 账户管理 │ 回测调度         │
│  Agent Gateway (新建: 接收 Agent 源码提交，触发编译+回测)      │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────┴──────────────────────────────────┐
│  Go 编译+执行层 (现有，扩展)                                  │
│                                                              │
│  compile_mql.go (现有)    compile_py.go (新建)               │
│       │                        │                             │
│       │  ┌─────────────────────┘                             │
│       │  │                                                   │
│       │  │  覆盖率 100% ──────────┐                           │
│       │  │    (MQL 直通 VM)       │                           │
│       ▼  ▼                        ▼                           │
│          interp.IR (共用)         VM (直接执行)               │
│       │  │                        │                           │
│       │  └───────────────────────┘                           │
│                │                                              │
│          compile.go (现有, IR → Bytecode)                     │
│                │                                              │
│          Bytecode (共用)                                      │
│                │                                              │
│          VM (现有, 唯一执行引擎)                               │
│                │                                              │
│          SimBroker (回测) │ LiveRunner (实盘)                 │
│                │                                              │
│          mtapi.io Gateway (MT4/MT5 访问)                      │
└─────────────────────────────────────────────────────────────┘
```

### D2: 双编译前端，单一 VM

两条编译路径产出相同的 `interp.IR`，复用 `compile.go`（IR → Bytecode）和 `vm.go`（VM 执行）。

```
路径 1 (现有): MQL 源码 → PreprocessMQL → tree-sitter-mql → CST → interp.IR
路径 2 (新建): Python 子集 → tree-sitter-python → CST → interp.IR
                                                              │
                                              compile.go (IR → Bytecode)
                                                              │
                                              vm.go (Bytecode → 执行)
```

**IR 扩展：** `interp.IR.Version` 新增 `"python"` 值。`compile.go` 和 `vm.go` 对 Version 透明——它们操作的是 IR 结构和 Bytecode 指令，不关心上游语言。

**Bytecode 不变：** 现有 40 个 opcode（`OP_PUSH_CONST` 到 `OP_HALT`）覆盖 Python 子集的全部语义需求：

| Python 语法 | Bytecode 指令 |
|---|---|
| `if/elif/else` | `OP_JMP_IF_FALSE` + `OP_JMP` |
| `for i in range(n)` | `OP_JMP_IF_FALSE` + counter + `OP_JMP` back |
| `while cond` | `OP_JMP_IF_FALSE` + `OP_JMP` back |
| `def func()` | `OP_CALL_USER` + `OP_ENTER_FUNC` + `OP_LEAVE_FUNC` |
| `return val` | `OP_RETURN` |
| `a + b` | `OP_ADD` |
| `a == b` | `OP_EQ` |
| `arr[i]` | `OP_PUSH_ARRAY` |
| `obj.field` | `OP_GET_FIELD` |
| `ctx.indicators.ima(...)` | `OP_CALL_BUILTIN` |

### D3: Python 子集规范

不是完整 Python，是约束子集——与 Bytecode 指令集几乎一一对应。

**允许：**
- `if/elif/else`、`for i in range(n)`、`while cond`
- `def func(args) -> type:`、`return val`
- 类型标注强制：`int`、`float`、`bool`、`str`、`Decimal`、`list[T]`（T 为标量）
- `ctx.*` SDK 调用（indicators、broker、bars、account）
- `Decimal` 算术（价格/仓位/盈亏）
- `class StrategyBase` 子类（单继承，仅 `__init__` + 事件方法）
- 事件方法：`on_init`、`on_bar`、`on_tick`、`on_timer`、`on_trade`、`on_deinit`
- `None` / `True` / `False` 字面量
- 算术/比较/逻辑运算符
- `break` / `continue`
- `# 注释`

**禁止：**
- list comprehension、generator、lambda、decorator
- `try/except`、`with`、`async/await`
- `import` / `from ... import` — **唯一例外：`from decimal import Decimal`**（编译器硬编码识别此行，其余一律拒绝）
- f-string、unpacking、slicing
- 多重继承、metaclass、`__getattr__`/`__setattr__` override
- `*args`、`**kwargs`
- 全局可变状态（除 `self.*` 实例属性）

**类型标注强制：** 所有变量声明、函数参数、返回值必须有类型标注。`compile_py.go` 在编译期检查类型一致性，不做类型推断。Python 的动态类型问题通过子集约束 + 标注检查消除。

**示例：**

```python
from decimal import Decimal

class MyStrategy(StrategyBase):
    def __init__(self, period: int = 14, lot: Decimal = Decimal("0.1")):
        self.period: int = period
        self.lot: Decimal = lot
        self.last_signal: int = 0

    def on_bar(self, ctx: BarContext) -> None:
        ema_fast: float = ctx.indicators.ima(ctx.symbol(), 10, 0)
        ema_slow: float = ctx.indicators.ima(ctx.symbol(), 30, 0)
        rsi: float = ctx.indicators.irsi(ctx.symbol(), self.period, 0)

        if ema_fast > ema_slow and rsi < 70:
            if ctx.positions.count() == 0:
                ctx.broker.buy(lot=self.lot, sl=ctx.ask() - Decimal("0.0050"))
                self.last_signal = 1

        for pos in ctx.positions:
            if pos.profit < -Decimal("50"):
                ctx.broker.close(pos.ticket)
```

### D4: Python SDK 接口（IR 层面）

Python 子集中的 `ctx.*` 调用映射到 VM 内置函数，与 MQL 路径共享同一套 `OP_CALL_BUILTIN` 实现。

| Python SDK | MQL 等价 | VM Builtin |
|---|---|---|
| `ctx.bars().close(1)` | `Close[1]` | `OP_PUSH_SERIES` |
| `ctx.bars().open(1)` | `Open[1]` | `OP_PUSH_SERIES` |
| `ctx.ask()` | `Ask` | `OP_CALL_BUILTIN("Ask")` |
| `ctx.bid()` | `Bid` | `OP_CALL_BUILTIN("Bid")` |
| `ctx.symbol()` | `Symbol()` | `OP_CALL_BUILTIN("Symbol")` |
| `ctx.point()` | `Point` | `OP_CALL_BUILTIN("Point")` |
| `ctx.indicators.ima(sym, period, shift)` | `iMA(sym, tf, period, 0, MODE_EMA, PRICE_CLOSE, shift)` | `OP_CALL_BUILTIN("iMA")` |
| `ctx.indicators.irsi(sym, period, shift)` | `iRSI(sym, tf, period, PRICE_CLOSE, shift)` | `OP_CALL_BUILTIN("iRSI")` |
| `ctx.broker.buy(lot, sl, tp)` | `OrderSend(sym, OP_BUY, lot, Ask, 3, sl, tp, ...)` | `OP_CALL_BUILTIN("OrderSend")` |
| `ctx.broker.close(ticket)` | `OrderClose(ticket, lots, Bid, 3)` | `OP_CALL_BUILTIN("OrderClose")` |
| `ctx.broker.modify(ticket, sl, tp)` | `OrderModify(ticket, price, sl, tp, 0, clrNone)` | `OP_CALL_BUILTIN("OrderModify")` |
| `ctx.positions.count()` | `OrdersTotal()` | `OP_CALL_BUILTIN("OrdersTotal")` |
| `ctx.positions[magic]` | `OrderSelect(i, SELECT_BY_POS, MODE_TRADES)` 循环 | `OP_CALL_BUILTIN` 组合 |

**关键设计：** Python SDK 的语义和 MQL 内置函数完全等价，底层调用同一个 VM builtin 实现。不存在"Python 版 iMA"和"MQL 版 iMA"——只有一个 `builtinIMA` 函数。

### D5: Agent 层架构

#### 5.1 Agent 进程模型

```
┌─ Python Agent 服务 (K8s Pod / Docker container) ──────────┐
│                                                            │
│  ConnectRPC Client ──── Go API 层                          │
│  (protobuf, 无 JSON)                                       │
│                                                            │
│  ┌─ 策略生成 Agent ──────────────────────────────────┐     │
│  │ 输入: 自然语言描述 + 策略画像请求                  │     │
│  │ 循环: LLM 生成 Python → 提交回测 → 分析 → 改进     │     │
│  │ 输出: 最终 Python 策略 + 回测报告 + 策略画像       │     │
│  └───────────────────────────────────────────────────┘     │
│                                                            │
│  ┌─ 盲区桥接 Agent ─────────────────────────────────┐     │
│  │ 输入: MQL 源码 + 覆盖率报告 + 盲区列表            │     │
│  │ 循环: LLM 翻译为 Python → 提交回测 → 分析 → 改进   │     │
│  │ 输出: Python 策略 + 变更说明 + 覆盖率对比          │     │
│  └───────────────────────────────────────────────────┘     │
│                                                            │
│  ┌─ 策略进化 Agent ─────────────────────────────────┐     │
│  │ 输入: 实盘绩效数据 + 策略源码 + 市场状态           │     │
│  │ 循环: 检测退化 → LLM 推理改进 → 回测验证 → 建议    │     │
│  │ 输出: 改进建议 + 改进后策略 + 对比报告             │     │
│  └───────────────────────────────────────────────────┘     │
│                                                            │
│  ┌─ 知识库检索 (Go 侧) ─────────────────────────────┐     │
│  │ Go API 层负责 pgvector 检索 (tenant_id 隔离)       │     │
│  │ Agent 请求时附带相似经验 → 注入 LLM prompt          │     │
│  │ Agent 不直接访问 PG / pgvector                      │     │
│  └───────────────────────────────────────────────────┘     │
│                                                            │
│  工具: pandas (数据分析) │ numpy (数值计算)               │
│        optuna (参数搜索) │ LLM (推理/生成)                 │
└────────────────────────────────────────────────────────────┘
```

#### 5.2 Agent 循环

```
Agent 收到任务
  │
  ▼
LLM 推理 → 生成 Python 策略源码
  │
  ▼
ConnectRPC: SubmitStrategy(source_code, params, backtest_config)
  │                    │
  │                    ▼
  │           Go API 层接收
  │                    │
  │                    ▼
  │           compile_py.go → interp.IR → compile.go → Bytecode
  │                    │
  │                    ▼
  │           VM + SimBroker 回测 (100ms 级)
  │                    │
  │                    ▼
  │           回测结果 → protobuf
  │                    │
  ▼◄───────────────────┘
  │  (一次 RPC 往返)
  │
  ▼
pandas 分析回测结果 (DataFrame)
  │
  ▼
LLM 推理: "回撤集中在伦敦盘开盘，建议加时间过滤"
  │
  ▼
修改 Python 策略源码
  │
  ▼
重新提交 → 编译 → 回测 → 结果返回
  │
  ▼
对比: 回撤 23% → 8% ✓
  │
  ▼
LLM 推理: "还可以优化 RSI 阈值"
  │
  ▼
... 自主循环 (max_iterations=50, cost_ceiling=$0.50) ...
  │
  ▼
Agent 满意 → 存入知识库 → 返回最终策略 + 分析报告
```

**性能：** VM 回测不是 Agent 循环的瓶颈——LLM 推理才是。

```
单次迭代耗时分解:
  LLM 推理:     1-5 秒    ← 主导因素 (占总时间 80-95%)
  compile_py:   300-500ms
  VM 回测:      100-200ms (H1, 1 年数据)
  RPC 往返:     ~5ms (同机)

50 次迭代实际总时间: 50-250 秒 (LLM 主导)
```

compile_py.go 的价值不是让 Agent 循环变快（LLM 是瓶颈），而是**确保回测不成为额外瓶颈**。per-bar RPC 方案下回测需要 16 分钟/次，50 次迭代 = 13 小时——这才是不可接受的。VM 内执行让回测从 16 分钟降到 200ms，使其相对 LLM 推理时间可忽略。

#### 5.3 六个 LLM 注入点

| 注入点 | 位置 | 输入 | 输出格式 | 输出语言 |
|---|---|---|---|---|
| [1] 策略画像 | 编译后 | AST + 覆盖率报告 | protobuf text format → `StrategyProfile` proto | 非代码 |
| [2] 盲区桥接 | 覆盖率 < 100% | MQL 源码 + 盲区列表 + 策略画像 | Python 策略源码 | Python 子集 |
| [3] 回测执行 | 回测时 | Bytecode + SimBroker | 回测结果 (protobuf) | 非代码 (确定性) |
| [4] 回测解读 | 回测后 | 回测结果 + 策略画像 | protobuf text format → `BacktestAnalysis` proto | 非代码 |
| [5] 参数优化 | 回测后 | 参数空间 + 回测结果 | protobuf text format → `OptimizedParams` proto | 非代码 |
| [6] 策略改进 | 回测后 | 源码 + 回测结果 + 市场状态 | Python 策略源码 (Agent 路径) 或 MQL 源码 (用户路径) | Python/MQL |

**[3] 不是 LLM 注入点**——回测执行是确定性的 VM + SimBroker，无 LLM 参与。

**所有结构化输出用 protobuf text format**，Go 端用 `prototext.Unmarshal` 解析。禁止 JSON（项目约束）。

#### 5.4 LLM 输出验证门

每个 LLM 注入点都有确定性验证：

| 注入点 | 验证方式 | 失败处理 |
|---|---|---|
| [1] 策略画像 | proto schema 验证 + 字段完整性检查 | 重试 (max 2) → 降级为空画像 |
| [2] 盲区桥接 | compile_py.go 编译成功 + 覆盖率提升 | 重试 (max 3) → 降级（见下） |
| [4] 回测解读 | proto schema 验证 | 重试 (max 2) → 降级为模板文案 |
| [5] 参数优化 | 参数值在合法范围内 | 重试 (max 2) → 跳过优化 |
| [6] 策略改进 | compile_py.go 编译成功 + 回测不劣化 | 重试 (max 3) → 丢弃改进 |

**LLM 不是编译器 pass**（不满足确定性、可重现、可测试）。正确定位是**带验证门的变换层**：LLM 变换 → 确定性验证 → 通过则继续 / 不通过则回退或重试。

**盲区桥接失败降级路径：**

当盲区桥接 Agent 在 3 次重试后仍无法将 MQL 翻译为可编译的 Python 子集（或覆盖率未提升），系统执行以下降级：

```
盲区桥接失败
  │
  ▼
标记策略为 "需人工适配" (strategy_status = 'bridge_failed')
  │
  ▼
生成盲区报告 (哪些函数无法桥接 + 建议替代方案)
  │
  ▼
前端展示:
  ├─ "此 EA 包含平台不支持的函数 (iCustom / WebRequest / DLL)
  │    Agent 已尝试自动桥接但未成功。"
  ├─ 列出无法桥接的函数 + 说明
  ├─ 建议 1: "在 MetaTrader 客户端中直接运行此 EA (MT 托管模式)"
  │         → 策略标记为 'mt_hosted'，平台仅做信号跟踪，不做回测
  └─ 建议 2: "手动修改 EA 移除不支持函数后重新上传"
```

**MT 托管模式**：对于无法桥接的 EA，平台不执行策略代码，而是通过 mtapi.io 监控 MT 客户端的交易信号，仅做信号跟踪和绩效展示。用户保留 MT 客户端完整功能，平台提供监控和分析。

#### 5.5 知识库设计

```sql
-- pgvector 扩展，复用现有 PostgreSQL
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE agent_experience (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    category    VARCHAR(32) NOT NULL,   -- strategy_pattern / market_regime / optimization_result
    content     TEXT NOT NULL,           -- 自然语言描述
    embedding   VECTOR(1536) NOT NULL,   -- LLM embedding
    metadata    JSONB NOT NULL DEFAULT '{}',  -- 结构化补充 (PG JSONB, 不在应用层 json.Marshal)
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_agent_experience_embedding
    ON agent_experience USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);

CREATE INDEX idx_agent_experience_tenant_category
    ON agent_experience(tenant_id, category);
```

Agent 遇到新策略时：
1. Agent 通过 ConnectRPC 调用 Go Gateway 的 `SearchExperience` RPC
2. Go API 层生成 embedding → pgvector 检索相似经验（cosine similarity > 0.85，`tenant_id` 过滤）
3. Go API 层返回相似经验列表（protobuf）
4. Agent 注入 LLM prompt："历史上类似策略的经验：..."
5. 生成初始策略时主动加入已知优化

**设计决策：知识库检索放在 Go API 层而非 Python Agent 层。**

理由：Phase 1 Agent 是无状态子进程（启动→完成→销毁），子进程内做 pgvector 检索意味着每次调用都要重新建立 PG 连接 + 检索。将检索放在 Go API 层：(1) 复用现有 PG 连接池；(2) `tenant_id` 隔离在 Go 层统一处理；(3) Agent 子进程保持无状态，不需要 PG 依赖；(4) Phase 2 长驻 Agent 仍可通过 Go Gateway 检索，不需要改为直连 PG。

### D6: ConnectRPC Agent Gateway

Go API 层新增 Agent Gateway service，接收 Agent 的策略提交和回测请求。

```protobuf
// proto/ant/v1/agent_gateway.proto

service AgentGatewayService {
  // Agent 提交策略源码，触发编译 + 回测
  // Phase 1: 同步模式 — 阻塞等待回测完成（回测 <30s），直接返回结果
  // Phase 2: 异步模式 — mode=ASYNC 时立即返回 strategy_id，Agent 通过
  //          GetBacktestResult 轮询或 SubscribeBacktestCompletion SSE 等待
  rpc SubmitStrategy(SubmitStrategyRequest) returns (SubmitStrategyResponse);

  // Agent 获取回测结果（异步模式或历史结果查询）
  rpc GetBacktestResult(GetBacktestResultRequest) returns (BacktestResult);

  // Agent 获取策略覆盖率分析
  rpc AnalyzeCoverage(AnalyzeCoverageRequest) returns (CoverageReport);

  // Agent 获取实盘策略绩效
  rpc GetLivePerformance(GetLivePerformanceRequest) returns (LivePerformance);

  // Agent 检索知识库相似经验（Go 层负责 pgvector 检索 + tenant_id 隔离）
  rpc SearchExperience(SearchExperienceRequest) returns (SearchExperienceResponse);

  // Agent 存储经验到知识库
  rpc StoreExperience(StoreExperienceRequest) returns (StoreExperienceResponse);

  // SSE: Agent 订阅策略退化告警
  rpc SubscribeDegradationAlert(SubscribeDegradationRequest) returns (stream DegradationAlert);

  // SSE: 异步回测完成通知 (Phase 2)
  rpc SubscribeBacktestCompletion(SubscribeBacktestCompletionRequest) returns (stream BacktestCompletion);
}

enum SubmitMode {
  SUBMIT_SYNC = 0;   // 同步: 阻塞等待回测完成 (Phase 1, 回测 <30s)
  SUBMIT_ASYNC = 1;  // 异步: 立即返回 strategy_id, 通过 SSE/轮询获取结果 (Phase 2)
}

message SubmitStrategyRequest {
  string source_code = 1;           // 策略源码
  string language = 2;              // "mql4" | "mql5" | "python"
  repeated ParamValue params = 3;   // 参数覆盖
  BacktestConfig backtest_config = 4; // 回测配置
  SubmitMode mode = 5;              // 同步/异步模式 (默认 SYNC)
}

message SubmitStrategyResponse {
  string strategy_id = 1;
  BacktestResult result = 2;        // 同步模式: 回测结果; 异步模式: 空
  CoverageReport coverage = 3;      // 覆盖率报告
  repeated BlindSpot blind_spots = 4; // 盲区列表
  SubmitMode mode = 5;              // 实际使用的模式
}
```

**同步/异步语义：**
- Phase 1（同步）：`SubmitStrategy` 阻塞直到回测完成（<30s），直接在 response 中返回 `BacktestResult`。适用于 Agent 循环内的单次回测。
- Phase 2（异步）：`mode=SUBMIT_ASYNC` 时立即返回 `strategy_id`，Agent 通过 `SubscribeBacktestCompletion` SSE 流等待完成通知，或通过 `GetBacktestResult` 主动查询。适用于长时间回测（多品种/多年）或批量回测场景。

**约束：** 所有数据交换用 protobuf，零 JSON。Agent 通过 ConnectRPC client 调用，SSE 接收推送。

### D7: 多租户与 SaaS

| 层 | 隔离方式 |
|---|---|
| Go 执行层 | VM 进程内隔离 + 指令计数器 + 超时 + MaxSourceSize |
| Go API 层 | `tenant_id` 上下文传播 + PG RLS |
| Python Agent 层 | 每 tenant 独立 Agent 进程（K8s Pod）或进程内 tenant 隔离 |
| 知识库 | `tenant_id` 列 + RLS |

**Agent 进程模型演进：**
- Phase 1: Go API 收到 Agent 请求 → 启动 Python 子进程 → 完成后销毁（无状态，简单）
- Phase 2: 长驻 Python Agent 服务 → K8s 部署，按 tenant 分 Pod（有状态，知识库缓存）

**计费：** Agent 每次调用有 LLM 成本，可按 LLM token 用量 + VM 执行时间计费。

**LLM 成本估算（基于 DeepSeek-V3 定价 $0.27/M input, $1.10/M output）：**

```
注入点              tokens_in   tokens_out  次数  小计
─────────────────────────────────────────────────────────
[1] 策略画像         ~500        ~200        1     $0.0004
[2] 盲区桥接         ~1500       ~1000       3     $0.004
[4] 回测解读         ~800        ~300        1     $0.0006
[5] 参数优化         ~500        ~200        1     $0.0004
[6] Agent 循环
    策略生成         ~2000       ~1500       20    $0.004
    修复循环         ~1000       ~800        10    $0.002
─────────────────────────────────────────────────────────
总计（典型策略）                                   ~$0.012
最坏情况（50 次迭代 + 多次修复）                     ~$0.05
配额上限                                            $0.50
```

注：实际成本取决于 LLM 模型选择和迭代次数。$0.50 配额上限覆盖最坏情况并留有余量。

**配额：**
- Agent 迭代次数上限：50 次/策略
- LLM 成本上限：$0.50/策略（典型 ~$0.012，最坏 ~$0.05）
- VM 执行超时：30 秒/回测
- 源码大小上限：500KB（复用 `MaxSourceSize`）

### D8: 安全模型

```
第一层: Python 子集验证 (compile_py.go 拒绝禁止语法)
  → 恶意代码无法进入 IR

第二层: Bytecode VM (只能调用 SDK 定义的函数)
  → 没有文件、网络、系统调用的入口

第三层: SDK 函数 (所有交易操作都经过风控)
  → PlaceOrder → Gate.Evaluate() → 风险评估通过才提交

第四层: Agent 进程隔离 (K8s Pod / 子进程)
  → Agent 进程崩溃不影响 Go 层
```

Python 子集禁止 `import`、`try/except`、`with`、`async/await`——Agent 生成的代码无法访问文件系统、网络、或执行任意 Python 代码。所有外部交互通过 `ctx.*` SDK 接口，受 VM 和风控 gate 约束。

---

## 3. 备选方案

### 方案 A: LLM 输出 MQL（DeepSeek 原始方案）

| 优点 | 缺点 | 否决理由 |
|---|---|---|
| 保留 MQL 生态，用户可审查 | LLM 生成 MQL 质量不稳定（语料少） | LLM 对 MQL 的生成正确率 ~70%，对 Python ~95% |
| 走现有 tree-sitter 管线 | tree-sitter 验证弱（只检查语法） | 无法捕获类型错误、未定义变量 |
| 不需要新建编译器前端 | MQL 表达能力有限（无 pandas/numpy） | Agent 的指标计算能力受限 |

### 方案 B: LLM 输出 Go SDK 代码

| 优点 | 缺点 | 否决理由 |
|---|---|---|
| go build 是最强验证门 | Go 的 Agent 生态为零 | Agent 需要 pandas/numpy/optuna，Go 里做这些要重新造轮子 |
| 复用现有 SimBroker | LLM 生成 Go 的迭代循环不自然 | Agent 的"写代码→运行→分析→修改"循环在 Python 里最自然 |
| 不需要新建编译器前端 | Go 语法比 Python 复杂（指针、接口、goroutine） | LLM 生成 Go 的复杂度高于 Python |

### 方案 C: per-bar RPC（Python 策略通过 ConnectRPC 逐 bar 调用 Go SimBroker）

| 优点 | 缺点 | 否决理由 |
|---|---|---|
| Python 策略可直接执行 | 200x 性能差距（16 分钟 vs 20 秒/50 次迭代） | Agent 循环内层不可接受 |
| 不需要 compile_py.go | Python 进程状态管理复杂 | 进程崩溃 → 状态丢失 → 故障模型不统一 |
| SimBroker 只需暴露 RPC | 两条 API 路径维护成本高 | 同一 SimBroker 被两种方式调用，行为必须一致 |

### 方案 D: 完全在 Python 中实现回测引擎

| 优点 | 缺点 | 否决理由 |
|---|---|---|
| Agent 和回测在同一进程 | 违反"回测即实盘"原则（ADR-0012） | 回测和实盘代码路径分叉 |
| 最快迭代速度 | Python 回测引擎与 Go SimBroker 行为不一致 | 回测结果不可信 |
| 不需要 Go ↔ Python 通信 | 重复实现撮合/风控/指标 | 维护成本极高 |

### 选定方案: 双编译前端 + Python Agent 层（§2 D1-D8）

| 维度 | 决策 |
|------|------|
| 策略语言 | MQL（现有管线） + Python 子集（新增，Agent 生成目标） |
| 编译 | compile_mql.go + compile_py.go → 共用 interp.IR → Bytecode → VM |
| 执行 | 单一 Bytecode VM（回测 SimBroker / 实盘 LiveRunner） |
| Agent 层 | Python（LangChain/CrewAI + pandas/optuna/pgvector），通过 ConnectRPC 调用 Go Gateway |
| Agent ↔ Go 通信 | ConnectRPC + SSE（protobuf），每迭代一次 RPC 往返 |
| 安全 | 四层：Python 子集验证 → Bytecode VM 沙箱 → SDK 风控 Gate → Agent 进程隔离 |

选取理由：方案 A（MQL）LLM 生成质量不足，方案 B（Go SDK）Agent 生态为零，方案 C（per-bar RPC）性能不可接受，方案 D（Python 回测）破坏回测即实盘原则。双编译前端 + 单 VM 是唯一同时满足 LLM 生成质量、Agent 生态需求和回测性能约束的架构。

---

## 4. 后果

### 正面

- **MQL 覆盖率从 60% 提升到 90%+**：盲区 EA 通过 Agent 翻译为 Python 子集 → VM 执行，绕过 MQL 语法限制
- **策略生成路径**：非程序员用户通过自然语言描述 → Agent 生成 Python 策略 → 回测验证
- **自我学习进化**：知识库 + pgvector 积累经验，Agent 跨策略学习
- **LLM 发挥核心价值**：从"代码修复工具"升级为"编译管线有机组成部分"
- **200x Agent 循环加速**：compile_py.go 让回测在 VM 内执行（100ms 级），而非 per-bar RPC
- **单一 VM**：两条编译前端，一个执行引擎，零行为不一致风险
- **SaaS 就绪**：多租户隔离 + 计费 + 配额

### 负面

- **compile_py.go 开发成本**：3-4 周（tree-sitter Python grammar Go binding + CST → IR + 类型检查 + 子集验证器 + 测试）
- **Python Agent 服务运维成本**：额外语言栈、K8s Pod 管理、LLM API 成本
- **LLM 非确定性风险**：同一输入可能产生不同输出，需要缓存（source hash → 结果）和重试机制
- **protobuf text format LLM 可靠性**：LLM 对 protobuf text format 的训练语料少于 JSON，需要 prompt 模板引导和 fallback 解析
- **Python 子集不是标准 Python**：LLM 需要被约束在子集内，可能生成被拒绝的语法（list comprehension 等），需要 prompt 工程和重试

### 中性

- Go 层和 Python 层通过 ConnectRPC 解耦，各自独立演进
- ADR-0023 的 Bytecode VM 不变，本 ADR 只新增编译前端和 Agent 层
- ADR-0022 的盲区分级原则不变，本 ADR 用 Agent 桥接替代"人工评估后实现"

---

## 5. 实施约束

### 5.1 文件结构

```
backend/tools/mql2go/
  compile_mql.go        (现有，MQL → IR)
  compile_py.go         (新建，Python 子集 → IR)
  compile.go            (现有，IR → Bytecode，共用)
  vm.go                 (现有，VM 执行，共用)
  compile_py_test.go    (新建，Python 子集编译测试)
  python_subset_test.go (新建，子集验证器测试)

backend/internal/agent/  (新建)
  gateway.go            (Agent Gateway ConnectRPC handler)
  profiler.go           (策略画像: 编排 LLM 调用 + proto 解析)
  bridge.go             (盲区桥接: 编排 Agent 循环)
  interpreter.go        (回测解读: 编排 LLM 调用 + proto 解析)
  optimizer.go          (参数优化: 编排 LLM 调用 + proto 解析)
  evolution.go          (策略进化: 编排 Agent 循环)
  cache.go              (LLM 输出缓存: source hash → 结果)

proto/ant/v1/
  agent_gateway.proto   (新建，Agent Gateway RPC 定义)
  agent_profile.proto   (新建，策略画像 proto message)
  agent_analysis.proto  (新建，回测解读 proto message)
  agent_optimization.proto (新建，参数优化 proto message)

python/agent/           (新建)
  client.py             (ConnectRPC client)
  base.py               (StrategyBase 基类)
  generators/           (策略生成/桥接/进化 Agent)
  knowledge/            (知识库 + pgvector 检索)
  analysis/             (pandas/numpy/optuna 工具)
```

### 5.2 编译管线约束

- `compile_py.go` 产出的 `interp.IR` 必须与 `compile_mql.go` 产出的 IR 结构完全兼容
- `compile.go`（IR → Bytecode）对 `IR.Version` 透明，不写 `if version == "python"` 分支
- `vm.go` 对上游语言透明
- Python 子集的类型标注在编译期检查，不做运行时类型推断
- 编译错误信息必须对 Agent 友好（行号 + 类型 + 期望/实际），支持 Agent 自主修复

### 5.3 Agent 层约束

- Agent 不直接访问 MT，所有 MT 操作通过 Go Gateway
- Agent 不直接执行策略，所有策略执行通过 Go VM
- Agent 不使用 JSON，所有数据交换用 protobuf
- Agent 不使用 REST/WebSocket，所有通信用 ConnectRPC + SSE
- Agent 每次迭代只有一次 RPC 往返（提交源码 + 取回结果）
- Agent 循环有上限：max 50 次迭代，$0.50 成本上限，30 秒 VM 超时
- LLM 输出缓存：按 source hash + prompt hash 缓存，避免重复调用

### 5.4 数据精度约束

- 价格/仓位/盈亏：`Decimal`（Python 子集）↔ `decimal.Decimal`（Go VM）↔ `NUMERIC(20,8)`（PG）
- 时间：UTC，毫秒精度（`int64 ts_unix_ms`）
- Symbol：raw broker symbol = canonical（不做后缀剥离）
- 禁止 float64 用于价格计算（复用现有约束）

### 5.5 项目约束遵守

| 约束 | 遵守方式 |
|---|---|
| ❌ JSON | 全用 protobuf（ConnectRPC + prototext） |
| ❌ REST | ConnectRPC + SSE |
| ❌ WebSocket | SSE 推送 |
| ❌ float64 in price | Decimal in Python 子集 + decimal.Decimal in Go |
| ✅ MT access via mtapi.io | Python 不直接访问 MT，通过 Go Gateway |
| ✅ Push-first | SSE 推送退化告警，Agent 不轮询 |
| ✅ 回测即实盘 | Python 策略走同一 VM + SimBroker，回测/实盘代码路径统一 |

---

## 6. 验证方式

### 6.1 compile_py.go 验证

```go
// Python 子集 → IR → Bytecode → VM 执行
func TestCompilePy_BasicStrategy(t *testing.T) {
    source := `
class MyStrategy(StrategyBase):
    def on_bar(self, ctx: BarContext) -> None:
        ema: float = ctx.indicators.ima(ctx.symbol(), 14, 0)
        if ema > ctx.bars().close(0) and ctx.positions.count() == 0:
            ctx.broker.buy(lot=Decimal("0.1"))
    `
    ir, err := CompilePythonToIR(source)
    assertNoError(t, err)
    bc := CompileAST(ir)
    assertCoverage(t, bc, 1.0)  // 100% coverage
}
```

### 6.2 Agent 循环验证

```go
// Agent 提交 Python 策略 → Go 编译+回测 → 结果返回
func TestAgentGateway_SubmitStrategy(t *testing.T) {
    resp, err := agentClient.SubmitStrategy(ctx, &pb.SubmitStrategyRequest{
        SourceCode: pythonStrategy,
        Language:   "python",
        BacktestConfig: testConfig,
    })
    assertNoError(t, err)
    assertCoverage(t, resp.Coverage, 1.0)
    assertNotEmpty(t, resp.Result.Trades)
}
```

### 6.3 端到端验证

```
1. 用户上传 MQL EA (覆盖率 70%)
   → Go 编译 + 覆盖率分析
   → 触发盲区桥接 Agent
   → Agent 翻译为 Python 子集
   → Go 编译 Python → VM 回测
   → 结果返回 Agent → 分析 → 迭代改进
   → 最终策略 + 变更说明返回用户

2. 用户描述策略 ("EURUSD H1，EMA10 上穿 EMA30 买入")
   → 策略生成 Agent
   → LLM 生成 Python 子集
   → Go 编译 → VM 回测
   → Agent 分析 → 改进 → 迭代
   → 最终策略 + 回测报告

3. 实盘策略运行 30 天
   → 策略进化 Agent 检测绩效退化
   → LLM 推理改进方案
   → 生成改进后 Python 策略
   → Go 编译 → VM 回测验证
   → 对比报告 → 建议用户确认
```

### 6.4 性能验证

```
Agent 50 次迭代循环:
  - 每次 LLM 推理: 1-5 秒 (主导因素, 占总时间 80-95%)
  - 每次 VM 回测: < 200ms (H1, 1 年数据)
  - 每次编译: < 500ms
  - 每次 RPC 往返: < 10ms (同机)
  - 总循环时间: 50-250 秒 (LLM 主导)
  - vs per-bar RPC: 回测从 16 分钟/次 → 200ms/次 (回测不再是瓶颈)
```

### 6.5 安全验证

```
Python 子集注入测试:
  - "import os; os.system('rm -rf /')" → compile_py.go 拒绝 (import 禁止)
  - "open('/etc/passwd').read()" → compile_py.go 拒绝 (无 open 函数)
  - "[x*2 for x in range(10)]" → compile_py.go 拒绝 (list comprehension 禁止)
  - "exec('malicious code')" → compile_py.go 拒绝 (exec 禁止)
```

---

## 7. 实施路线

### Phase 0: compile_py.go (3-4 周)

**目标：** Python 子集 → IR → Bytecode → VM 执行

| 工作项 | 周期 |
|---|---|
| tree-sitter Python grammar Go binding 验证 | 2 天 |
| Python CST → interp.IR 映射 | 5 天 |
| 类型标注检查（非推断） | 3 天 |
| 子集验证器（拒绝禁止语法） | 2 天 |
| Agent 友好的编译错误信息 | 2 天 |
| Python SDK 接口映射（ctx.* → VM builtins） | 3 天 |
| 测试（子集覆盖 + 边界 + 安全） | 4 天 |

**产出：** `compile_py.go` + `compile_py_test.go` + `python_subset_test.go`

**验证标准：**
- Python 子集策略能编译为 Bytecode 并在 VM 中执行
- 覆盖率 100%（Python 子集的所有语法结构都有对应的 Bytecode 指令）
- 禁止语法被正确拒绝并给出 Agent 可理解的错误信息
- 与 MQL 路径产出的 IR 结构兼容（`compile.go` 无需修改）

### Phase 1: Agent Gateway + 策略画像 + 回测解读 (2 周)

**目标：** Agent ↔ Go 通信 + 两个 LLM 分析注入点

| 工作项 | 周期 |
|---|---|
| `agent_gateway.proto` + ConnectRPC handler | 3 天 |
| `agent_profile.proto` + `agent_analysis.proto` | 1 天 |
| 策略画像 prompt + prototext 解析 + 缓存 | 3 天 |
| 回测解读 prompt + prototext 解析 + 缓存 | 3 天 |
| 前端策略卡片 + 回测解读 UI | 4 天 |

**产出：** 用户上传 EA 后立即看到策略画像，回测后看到 LLM 解读

### Phase 2: 盲区桥接 Agent (2 周)

**目标：** MQL 盲区 EA → Agent 翻译为 Python → VM 回测

| 工作项 | 周期 |
|---|---|
| Python Agent SDK (ConnectRPC client + 策略提交) | 3 天 |
| 盲区桥接 prompt + Agent 循环编排 | 4 天 |
| 变更审计存储 + UI (diff 前后代码) | 3 天 |
| 端到端测试 (MQL EA → Agent → Python → VM → 结果) | 4 天 |

**产出：** 覆盖率 60% → 90%+

### Phase 3: 策略生成 Agent (2 周)

**目标：** 自然语言描述 → Agent 生成 Python 策略 → 回测验证

| 工作项 | 周期 |
|---|---|
| 策略生成 prompt + Agent 循环编排 | 4 天 |
| 自然语言 → 策略画像 → Python 策略 | 3 天 |
| 前端策略生成 UI (对话式) | 3 天 |
| 端到端测试 | 4 天 |

**产出：** 非程序员用户获客路径

### Phase 4: 知识库 + 策略进化 (4 周)

**目标：** 经验积累 + 实盘监控 + 自动优化

| 工作项 | 周期 |
|---|---|
| pgvector 知识库 schema + 语义检索 | 3 天 |
| 经验存储 + 检索 + prompt 注入 | 3 天 |
| 策略进化 Agent (退化检测 + 改进循环) | 5 天 |
| SSE 退化告警 + 前端通知 | 3 天 |
| 跨策略学习 + 主动建议 | 5 天 |
| 端到端测试 | 4 天 |

**产出：** 自我学习进化能力

### 总周期：13-14 周

---

## 8. 与现有 ADR 的关系

| ADR | 关系 |
|---|---|
| ADR-0012 (回测即实盘) | **保留**。Python 策略走同一 VM + SimBroker，回测/实盘代码路径统一 |
| ADR-0020 (EA 替代 SDK) | **保留**。Go SDK 接口不变，Python 子集通过 VM 内置函数间接使用 |
| ADR-0021 (Python→Go 迁移) | **部分保留**。Go 仍是执行层，Python 作为 Agent 层回归（不同角色） |
| ADR-0022 (盲区架构) | **扩展**。三层盲区处理原则不变，Agent 桥接替代"人工评估后实现" |
| ADR-0023 (Bytecode VM) | **复用并扩展**。VM 不变，新增 compile_py.go 编译前端 |

---

## 9. 关键设计决策记录

### Q: 为什么 LLM 输出 Python 而不是 MQL？

A: 三个理由：(1) LLM 对 Python 的训练语料远多于 MQL，生成正确率 ~95% vs ~70%；(2) Python 子集与 Bytecode 指令集几乎一一对应，编译映射简单；(3) Agent 的思考环境（pandas/numpy/optuna）在 Python 生态中。

### Q: 为什么 LLM 输出 Python 而不是 Go？

A: Go 的 Agent 生态为零。Agent 需要 pandas 分析回测结果、optuna 搜索参数空间、pgvector 语义检索经验——这些工具全在 Python。Go 适合做确定性基础设施，Python 适合做智能 Agent。

### Q: 为什么需要 compile_py.go 而不是 per-bar RPC？

A: Agent 循环内层是回测，per-bar RPC 导致 200x 性能差距（16 分钟 vs 20 秒/50 次迭代）。compile_py.go 让 Python 策略编译为 Bytecode 在 VM 内执行，Agent 每次迭代只有一次 RPC 往返。

### Q: 为什么 protobuf text format 而不是 JSON？

A: 项目约束禁止 JSON（`encoding/json`、`json.Marshal`/`json.Unmarshal`）。protobuf text format 是合规的结构化输出格式，Go 端用 `prototext.Unmarshal` 解析。

### Q: 为什么 Agent 不直接访问 MT？

A: 项目约束要求 MT 访问只能通过 mtapi.io gRPC（Go Gateway）。Python Agent 通过 ConnectRPC 调用 Go Gateway，Go Gateway 调用 mtapi.io。隔离 MT 访问权限，防止 Agent 直接操作交易账户。

### Q: 为什么 Python 子集禁止 import？

A: 安全约束。Agent 生成的代码不能访问文件系统、网络、或执行任意 Python 代码。所有外部交互通过 `ctx.*` SDK 接口，受 VM 和风控 gate 约束。禁止 import 确保策略代码只能使用 SDK 提供的功能。

### Q: 为什么不是"LLM 是编译器的一个 pass"？

A: 编译器 pass 的定义：确定性、可重现、可测试、相同输入永远产生相同输出。LLM 不满足任何一条。正确定位是"带验证门的变换层"：LLM 变换 → 确定性验证门 → 通过则继续 / 不通过则回退或重试。

---

## 审核

| 角色 | 审核人 | 日期 | 结论 |
|------|--------|------|------|
| 作者 | GLM | 2026-07-02 | — |
| 审核人 | Claude | 2026-07-02 | Approved |

**审核意见：** 文档完整、逻辑自洽、约束合规。3 项必须修正（导入规则、同步/异步语义、知识库与子进程冲突）和 5 项建议补充（LLM 耗时标注、MQL 直通路径、proto language 字段、LLM 成本计算、桥接失败降级）均已落实。ADR-0024 正式接受为项目架构决策。
