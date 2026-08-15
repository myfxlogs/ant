# Spec — 策略运行诊断（Strategy Runtime Diagnostics, SRD）

> 涉及功能块：`strategy-runtime`（runner/VM/session registry）+ `mql-compiler`（VM builtins）+ `api-gateway`（proto/SSE）+ `frontend`
> 日期：2026-08-15 ｜ 作者：审计方（Claude Code）｜ 状态：**设计方案 v1.1（自审修订后，待用户批准出施工 handoff）**
> 缘起：2026-08-15 全天 MACD 监控实战的产物化——"策略 3 天没开单是不是坏了"的困惑只能靠人肉诊断（本文作者当天做了：评估计数、指标距离计算、交叉历史审计、条件拦截归因），系统应内建此能力。
> **v1.1 自审修订（6 项，见 §10）**：R1 传输层设计错误（心跳是空事件不带数据）；R2 指标键字符串热路径分配；R3 orders_total 捕获点错位（VM 内部值非事件循环可观测）；R4 对抗对账需确定性窗口；R5/R6 文件红线与重启重置语义。

---

## 0. 问题与原则

**问题**：买家/用户面对"运行中但零信号"的策略无法区分四种状态：①健康·条件未满足 ②数据饥饿（上游断流）③暖机中（窗口不足）④已僵死（goroutine 活着但不评估）。今天只能靠人肉（刷屏日志、独立算指标、扫交叉历史）。

**第一性原则**：
1. **VM 执行的一切皆已计算，只是被丢弃**——评估在发生、指标值在产出、条件在判断；在丢弃点捕获，而非新建计算。
2. **MT4 生态结构性做不到**（EA 对终端是黑盒）——我们拥有 VM，这是"代码不出平台"战略在可观测性上的直接红利，也是策略市场的信任护城河放大器（透明度可售卖）。
3. **Push-First**：诊断数据随既有事件流/SSE 通道走，零轮询、零新基础设施。
4. **通用性优先**：不解码任意策略的语义（"条件2被EMA拦截"是 Phase 3 AI 的按需工作），系统层只提供语义无关的事实（计数/指标值/窗口），让 90% 的困惑在 UI 层自答。

## 1. 分层架构

```
VM 事件执行（每 tick/bar）
  ├─ L1 评估事实：eval_count / last_eval_at / window_bars / orders_total_seen
  ├─ L2 指标值：builtin 调用值环形缓存（iMACD/iMA/iRSI/... 最近 N 值）
  └─（丢弃点捕获 → ActiveSession 内存态）
       ↓ 随既有 SSE（WatchActiveStrategies，节流聚合）
前端 行展开「诊断」tab：状态徽章 + 计数器 + 指标 sparkline
       ↓ （按需）
L3 AI 诊断：agent-engine 读策略代码 + L1/L2 快照 → 自然语言归因（credit 计费）
L4 夜间回放审计：回测引擎重放当日 bar → "全条件满足但未触发"检测（离线）
```

## 2. Layer 1 — 评估事实（计数器）

**捕获点**：`connect/strategy/vm_live_handlers.go` 的 vmHandleBar/vmHandleTick/vmHandleTrade——**VM 真实执行处**（而非事件到达处：session 尚未建立的 bar 到达不是评估；session 建立后每事件都是评估，含暖机期——暖机期策略在跑门槛检查同样算评估）。

**ActiveSession（session_registry）新增字段**（纯内存）：
```go
diag struct {
    mu            sync.Mutex
    evalCount     int64     // 总评估次数（tick+bar+trade）
    tickCount     int64     // 分类型计数
    barCount      int64
    lastEvalAt    int64     // unix ms —— 数据饥饿判定的核心
    windowBars    int       // 服务端窗口长度（handleBar 处 len(*bars)，暖机可见性）
    lastSignalAt  int64     // 已有，纳入统一快照
}
```

⚠️ **ordersTotal 不在 L1**（v1.1 R3 修正）：它是 VM 内部值（`vm.cachedPositions/cachedOrders` 长度），事件循环观测不到——归入 L2 的 VM 捕获面（`VMRunner.LastOrdersTotal()` 暴露，vmHandleX 尾部读取），随 L2 路径写入 diag。

**状态机（UI 徽章，由前端从字段推导，后端不算状态）**：
- `评估中`（绿）：lastEvalAt < 2×预期周期（1m 策略 = 2 分钟内有事件）
- `数据饥饿`（橙）：lastEvalAt 超期——**今天"刷屏=0"歧义的直接解药**
- `暖机`（蓝）：windowBars < 100（通用阈值；策略各异无法精确，展示原始值即可）
- `已停止/错误`（红）：既有 status/last_error

**Push 通道（v1.1 R1 修正——原方案有传输层设计错误）**：既有 20s 心跳是**空事件**（`Heartbeat:true` + 空 strategies——LIVE-SSE-HEARTBEAT 刻意如此，前端跳过空事件防表闪空），**不携带任何数据**；而变更通知只在 session 注册/注销/状态变化时触发——计数器更新没有现成的周期载体。修正设计：**心跳节拍复用、内容分频**——每 3 个心跳（60s）发一次携带完整 strategies+diagnostics 的数据事件，其余心跳保持空事件。前端既有"跳过空事件"逻辑天然兼容（数据事件才更新表，无闪空）；60s 粒度足够人读计数器。

```proto
message StrategyDiagnostics {
  int64 eval_count = 1;
  int64 tick_count = 2;
  int64 bar_count = 3;
  int64 last_eval_at_unix_ms = 4;
  int32 window_bars = 5;
  int32 orders_total_seen = 6;   // 经 L2/VM 路径填充（见上）
}
message ActiveStrategy { ... StrategyDiagnostics diagnostics = <next>; }
```
proto 字段号一次加齐；TS 侧同步 buf generate（工具链已具备，PARITY 批用过）。

## 3. Layer 2 — 指标值捕获（builtin 环形缓存）

**捕获点**：`tools/mql2go/vm_builtin_indicators.go` 等 builtin 实现层——**在返回值处记录**，键 = builtin 名 + 参数签名：

```go
// VM 新增（runEvent 结束时由 vmHandleX 读取回收，不跨事件累积）
lastIndicators map[string]decimal.Decimal   // key 如 "iMACD[12,26,9].main"
lastOrdersTotal int                          // v1.1 R3：与指标同路径捕获
```

- 键规范：`名字[参数逗号分隔][.子线]`，同一事件内多次调用（不同 shift）只记 shift=0 的最新值——诊断要的是"当前水平"，不是全序列。
- ⚠️ **热路径纪律（v1.1 R2）**：键是 `fmt.Sprintf` 产物——若每 tick 现拼字符串 = 分配/GC 洪峰，成本表作废。**必须 memoize**：VM 持 `diagKeyCache map[uint64]string`（hash(参数序列)→键），参数组合对给定策略恒定，命中后零分配； miss 时构建一次。
- **环形缓存放 ActiveSession**：`map[string][]decimal` 每键 cap 64（≈1 小时 @1 事件/分钟；tick 策略在写入侧节流**每 5s 最多记一次**——节流在 server 侧 vmHandleX 尾部做，VM 内每事件照记最新值零负担）。
- 覆盖 builtin 清单（首批）：iMA / iMACD(main+signal) / iRSI / iATR / iBands(上中下) / iStochastic / iADX——覆盖 MQL4 EA 的 95%。
- **proto**（嵌进 StrategyDiagnostics，节流后推送）：
```proto
message IndicatorSnapshot { string key = 1; string value = 2; int64 at_unix_ms = 3; }
repeated IndicatorSnapshot indicators = 7;
```
- **隐私/持久化**：仅内存 + SSE 瞬时；不落库（诊断非审计；审计是 L4 的事）。

## 4. 前端 — 行展开「诊断」tab

- 行展开 tabs 加第五个「诊断」（持仓/信号/日志/配置/**诊断**）——网格对齐规则覆盖前 4 个 tab，第 5 个自然顺排（COL_PCT 不动，`live-expanded-align` CSS 通用）。
- 内容三块：
  1. **状态徽章**（状态机四态）+ 一句话自答："健康运行中，近 30 分钟评估 1,847 次，窗口 500 根，未满足入场条件"
  2. **计数器行**：评估总数 / tick:bar 比 / 最后评估 / OrdersTotal
  3. **指标 sparkline**（每键一条，64 点，复用现有 chart 组件或轻量 SVG）
- i18n：新 key ×5 locale。

## 5. Layer 3/4 — 方向性设计（后批，不在 SRD-1 范围）

- **L3 AI 诊断**：新 RPC `DiagnoseStrategy(schedule_id)`（SSE 流式）→ agent-engine 组装 prompt = 策略源码 + L1/L2 快照 + 近期信号 + 行情概要 → 归因报告（今天人肉诊断的自动化）。**credit 计费**（挂现有 credit_accounts），每用户限频。REUSE agent-engine 既有流式管道。
- **L4 夜间审计**：回测引擎以当日真实 bar 重放（编译缓存命中），VM 加"分支结果标记"探针（顶层 if 求值记录——探索性，先 PoC），产出"全条件满足但未触发"清单 → 通知。成本 ≈ 每策略每晚 <1s CPU。

## 6. 成本（实测基线：当前后端 74MB / 1.5% CPU @1 策略）

| 项 | 1000 并发策略 | 基础设施 |
|---|---|---|
| L1 计数器 | ~1% 单核 / 0.1MB | 无 |
| L2 环形缓存 | <1% 单核 / ~8MB | 无 |
| SSE 增量 | 每心跳 +~1KB | 无（既有通道） |
| L3（按需） | 0 服务器侧 | LLM API 按次（credit 回收） |
| L4（夜间） | ~17 CPU 分钟/晚 | 无 |

结论：**SRD-1 边际成本 <2% CPU / <20MB，当前 $20-40/月 机器零扩容承载**。

## 7. 验收与对抗（SRD-1 施工时必带）

1. L1：发 N 个 tick 事件 → eval_count==N（删计数行 → RED）；断流 2× 周期 → 前端徽章转"数据饥饿"（删判定字段 → RED）。
2. L2：执行含 `iMACD` 的策略一个事件 → diagnostics.indicators 含 `iMACD[...].main` 且值 == 期望值。**对账必须确定性（v1.1 R4）**：期望值在测试内用**同一组固定 500 根 bar**独立计算 EMA——禁止与"PG 实时数据独立计算"对账（实时窗口长度/暖机与 VM 的 500 根不一致，浮点尾差必闪断）。当天人肉对账用的是符号/方向级一致，测试要收紧到同窗口确定性等值（epsilon 1e-9）。删 builtin 记录行 → RED。
3. SSE：每 3 个心跳出现一次携带 diagnostics 的数据事件（断言字段非零）；空心跳仍为空（前端防闪空回归）。
4. 红队：tick 洪峰下 L2 写入频率被 5s 节流压平（断言环形写次数 ≈ 时长/5s，不随 tick 线性）；map 无界防护（键数 cap 32）；diag.mu 细粒度勿复用 session 全局锁；**热路径零分配断言**（R2：开 -benchmem 跑 100 万次 builtin 记录，allocs/op == 0 after memoize）。
5. 门禁：既有全套 + check-file-lines（**R5：`session_registry.go` 现 403 行 🟡，diag 结构+方法进新文件 `session_diag.go`；`vm_builtin_indicators.go` 增行后超 300 软限同理拆 `vm_builtin_diag.go`**）。

## 8. 分批与范围

- **SRD-1（本批）**：L1 + L2 + proto + 诊断 tab + 对抗。改动面：`session_registry.go`、`live_runner_events.go`、`vm_builtin_indicators.go`、`vm.go`（runEvent 尾部回收）、proto ActiveStrategy、前端行展开。
- **SRD-2**：L3 AI 诊断 RPC（依赖 agent-engine 排期）。
- **SRD-3**：L4 夜间审计（依赖 L2.5 分支探针 PoC）。
- **明确不做**：通用语义条件解码（L3 的活）、环形缓存持久化、跨策略横向分析（那是 marketplace analytics 的既有范畴）。

## 9. 语义与边界补记（v1.1 R6 + 隐私界定）

- **重启重置语义**：diag 挂在 run 级（ActiveSession），E2 参数自动重启 → 计数器归零重新累计——与 run 生命周期一致，属正确语义而非缺陷；前端诊断区显示"本次运行"字样明确归零原因。
- **市场租赁策略的隐私界定**：指标实时值与持仓同属**运行透明**层级（买家本就看得到持仓/PnL），不泄露策略代码与逻辑结构——"代码不出平台"边界不受影响；作者侧若未来要"隐藏诊断"，加 per-strategy 开关即可（记录在案，不做进 SRD-1）。

## 10. 修订记录

| # | 级别 | 问题（v1.0） | 修正 |
|---|---|---|---|
| R1 | 🔴 传输层设计错误 | "随既有 20s 心跳携带"——但心跳是**刻意设计的空事件**（防表闪空），变更通知又只在 session 变化时触发，诊断更新没有周期载体 | 心跳节拍复用+内容分频：每 3 心跳（60s）带完整数据事件，其余保持空 |
| R2 | 🟡 热路径隐患 | 指标键每 tick `fmt.Sprintf` → 分配/GC 洪峰，成本表作废 | `diagKeyCache` memoize（参数组合恒定→命中后零分配）+ 对抗加 allocs/op==0 断言 |
| R3 | 🟡 捕获点错位 | `orders_total` 放在 L1（事件循环可观测）——实际是 VM 内部值，事件循环拿不到 | 移入 L2/VM 捕获面（`VMRunner.LastOrdersTotal()`） |
| R4 | 🟡 对抗会闪断 | L2 对账写"与 PG bar 独立计算对账"——实时窗口长度/暖机与 VM 500 根不一致，浮点尾差必 flaky | 改为同窗口确定性对账：固定 500 根 + 测试内独立 EMA + epsilon 1e-9 |
| R5 | 🟢 红线预防 | session_registry.go 403 行（🟡）、vm_builtin_indicators.go 增行风险 | diag 结构进新文件 `session_diag.go` / `vm_builtin_diag.go`，check-file-lines 门禁 |
| R6 | 🟢 语义缺口 | 未定义重启后计数器行为 + 租赁策略指标值隐私边界 | 归零=run 语义（前端标注"本次运行"）；指标值=运行透明层级，与持仓同级 |

## 11. 关联

- 实战依据：registry「实盘无法开仓调查」+「LIVE 语义一致性审计」+ 2026-08-15 监控日志（MACD 全天诊断）
- 与 DEPLOY-PARAMS 批零文件冲突可并行；与 EDIT-PARAMS 批的 EditParamsModal 相邻但不同文件
- 信任护城河定位：`docs/roadmaps/market-strategy-review.md`（trust = 基本盘）的可观测性延伸
