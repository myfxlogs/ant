# EA 完全替代 · 实施说明书（交付 DeepSeek 执行）

> **日期**：2026-06-23
> **性质**：施工说明书 / 实现依据。本文件**不重复论证架构**，只把已确定的方案翻译成「可直接编码的契约 + 任务清单 + 验收标准」。
> **配套阅读（必读，按顺序）**：
> 1. `docs/audit/2026-06-22-策略EA执行系统-现状地图.md`（现状基线：什么能用、什么是空壳）
> 2. `docs/plan/2026-06-22-EA完全替代-完整解决方案.md`（北极星架构与理由）
> 3. 本文件（怎么落地）
> **执行者**：DeepSeek（写代码）。**审阅者**：人类 + Cascade。

---

## 0. 给执行者（DeepSeek）的工作守则

阅读并严格遵守以下规则，违反任何一条都视为该任务失败：

- **必须遵守 `CLAUDE.md`**：文件/函数行数上限、push-first 架构、金额用 Decimal/定点而非 float、提交前检查。任何新文件超限要拆分。
- **契约先行**：第 1 章「契约」是冻结接口。**禁止在未实现接口前先写实现**；也禁止擅自修改已冻结的接口签名——如需改动，先在 PR 描述里提出并标注 `CONTRACT-CHANGE`，等人类确认。
- **一次只做一个 Task**：本文件第 3 章把工作拆成带编号的 Task（T0.x ~ T4.x）。每个 Task 有「输入 / 产出文件 / 接口 / 验收标准」。**逐个完成，每个 Task 配测试，跑绿再下一个。**
- **不删测试、不弱化测试**：已有测试不许为了通过而改松。新功能先写测试（或与实现同 PR）。
- **复用优先**：第 4 章列了「必须复用」的现有资产（回测引擎、MT 网关 RPC、调度器）。不要重写已成熟的部分。
- **不引入空壳**：不要再造"建好但没人调用"的模块（参考现状里 factor DSL / ONNX 的教训）。每个新模块必须有真实调用方或测试驱动。
- **不确定就停下提问**：遇到方案未覆盖的语义分叉（尤其 MT 交易语义边角），**不要猜**，在 PR/注释里用 `NEEDS-DECISION:` 标注并停在该点。

---

## 1. 已定决策（Decision Record，作为实现依据）

| # | 决策 | 取舍理由（详见方案文档） |
|---|---|---|
| D1 | **目标语言 = Python Strategy SDK**（非自研 VM） | 最佳翻译目标：产物可读可改、LLM 友好；性能/安全短板用 D4/D5 补齐。自研 MQL→IR VM 作为后手，本期不做。 |
| D2 | **建模对象 = 「券商 + 事件模型」，不是「语言」** | EA 只依赖 `OnInit/OnTick/OnTimer/OnTrade/OnDeinit` + 交易服务器语义。忠实复刻这两样，翻译即退化为机械映射。 |
| D3 | **回测/实盘共用同一份策略代码**，差异只在注入的 Broker 实现 | 真正落地 ADR-0012「回测=实盘同路径」，消除语义漂移，这是"实盘可信"的前提。 |
| D4 | **引擎编排用 Go，策略运行在 Python worker** | 沿用现状分工（Go 调度/网关，Python 执行）。 |
| D5 | **安全边界下移到 OS/VM 级**（seccomp+断网+只读FS+非root+cgroup，强档 gVisor/microVM） | RestrictedPython 降级为 lint，不再充当安全边界。爆炸半径锁死后反而可放开完整 Python。 |
| D6 | **所有下单意图（sim 和 live）必经 Go 风控门** | 金钱安全与策略对错解耦；策略疯了坏单也到不了市场。 |
| D7 | **退役 signal-dict 实盘路径** | 用 Broker 模型替换 `dispatchLiveSignal` 的裸买卖；factor DSL/ONNX 不用于 EA。 |

> 这些决策已确认。DeepSeek 在实现中**以此为准**，不需重新论证；若实现中发现某决策技术上不可行，用 `NEEDS-DECISION:` 上报。

---

## 2. 总体落地顺序（先契约，后实现，分期降风险）

```
Phase 0  契约冻结 + 表达力验证（手写样本 EA）         ← 不写引擎，先证明 SDK 够用
Phase 1  SimBroker + SDK 运行时（回测保真度基线）     ← 复用现有回测引擎
Phase 2  翻译层（确定性 transpiler + LLM 补全 + 校验）
Phase 3  LiveBroker + Go 风控门 + OS 沙箱（实盘闭环）
Phase 4  上线（kill-switch / 灰度 / 漂移看板）
```

每个 Phase 内的 Task 见第 3 章。**不允许跨 Phase 并行实现**（契约除外），因为后一阶段依赖前一阶段冻结的产物。

---

## 3. 任务清单（Task Breakdown，逐个交付）

> 格式约定：每个 Task 给出 **目标 / 产出文件 / 接口或数据契约 / 验收标准（DoD）**。
> 路径为建议位置，DeepSeek 可按仓库现有结构微调，但需在 PR 说明实际落点。

### Phase 0 — 契约冻结

#### T0.1 冻结 Strategy SDK 规格（文档 + 类型桩）
- **目标**：把"策略能调用的一切"写成精确规格，作为翻译层与运行时的共同真相源。
- **产出**：
  - `docs/spec/30-strategy-sdk.md`（SDK 权威规格）
  - `strategy-service/app/sdk/__init__.py` + 类型桩（`context.py` / `broker.py` / `symbol.py` / `account.py` / `series.py` / `indicators.py` 的接口签名，先桩后实现）
- **契约（必须包含，禁止缩水）**：
  - **生命周期**：`on_init()`、`on_tick()`、`on_bar(timeframe)`、`on_timer()`（配 `set_timer(seconds)`）、`on_trade()`、`on_deinit(reason)`。策略以**类**承载（`class Strategy(StrategyBase)`），便于翻译承接 EA 的全局状态。
  - **Series（MQL 逆序索引）**：`close[0]` = 当前 bar，`close[1]` = 上一根。提供 `open/high/low/close/volume/time`，多时间框可取。
  - **Symbol 元数据**：`digits, point, tick_value, tick_size, contract_size, volume_min, volume_max, volume_step, stops_level, freeze_level, swap_long, swap_short, margin_rate`。
  - **Account**：`balance, equity, margin, free_margin, margin_level, leverage, currency`。
  - **Indicators**：`ma, ema, rsi, bands, macd, atr, stoch, cci, ...` + `i_custom(name, params)` 等价物（自定义指标挂载点）。
  - **Trade API（全部经 `self.broker`）**：`order_send(request) -> OrderResult`、`position_modify(ticket, sl, tp)`、`position_close(ticket, volume=None)`、`order_delete(ticket)`、全挂单类型（buy/sell limit/stop/stop_limit）。请求体支持 `magic, comment, deviation, type_filling(FOK/IOC/Return)`。
  - **持仓查询**：`positions(symbol=None, magic=None) -> list[Position]`、`orders()`（挂单）。
  - **账户模式**：`netting` / `hedging`；执行模式 `market/instant/request/exchange`。
- **DoD**：规格文档评审通过；类型桩可被 `mypy`/import 通过；**无任何实现逻辑**（全 `raise NotImplementedError` 或 `...`）。

#### T0.2 冻结 Broker 接口
- **目标**：定义 SimBroker / LiveBroker 都要实现的统一接口。
- **产出**：`strategy-service/app/sdk/broker.py`（`Broker` 抽象基类）。
- **契约**：
  - 方法：`order_send / position_modify / position_close / order_delete / positions / orders / account / symbol_info / server_time`。
  - 所有方法**同步返回结构化结果**（`OrderResult{retcode, ticket, price, volume, comment}`），retcode 对齐 MT 返回码语义（成交/部分成交/requote/拒绝/保证金不足…）。
  - **金额/价格字段一律用 Decimal 或定点整数，禁止 float**（遵守 CLAUDE.md）。
- **DoD**：接口桩通过类型检查；SimBroker / LiveBroker 两个空实现类能继承它而不报缺方法。

#### T0.3 冻结风控门协议（Go 侧）
- **目标**：定义"下单意图 → 裁决"的协议，sim/live 共用。
- **产出**：`proto/ant/v1/risk_gate.proto`（新增）+ `docs/spec/31-risk-gate.md`。
- **契约**：
  - `OrderIntent{ user_id, account_id, symbol, side, volume, type, price, sl, tp, magic, source(sim/live) }`
  - `RiskDecision{ allow(bool), reason, adjusted_volume(optional), rule_hit }`
  - 规则项：最大手数、最大持仓数、最大敞口、日亏/回撤熔断、品种白名单、杠杆上限、下单频率、重复单防护、保证金预检、全局 kill-switch、每用户 autotrade 开关。
- **DoD**：proto 编译通过；规格文档列出每条规则的判定输入与默认阈值。

#### T0.4 手写样本 EA 验证 SDK 表达力（关键防呆）
- **目标**：**在写引擎前**证明 SDK 能表达真实 EA 形态。
- **产出**：`strategy-service/tests/sdk_samples/`，按 SDK 手写 5 个代表性策略：单进单出、网格、马丁、多单对冲、带自定义指标。
- **DoD**：5 个样本能 import、类型检查通过、逻辑可读；评审确认 SDK **无表达力缺口**。若发现缺口 → 回到 T0.1 修规格（这正是 Phase 0 的目的）。

---

### Phase 1 — SimBroker + SDK 运行时

#### T1.1 实现 SDK 运行时容器
- **目标**：把策略类按生命周期驱动起来，注入 context/broker/series/indicators。
- **产出**：`strategy-service/app/sdk/runtime.py`（驱动 `on_init→on_tick/on_bar/on_timer→on_deinit`）。
- **复用**：现有 `engine/context.py`、`engine/indicators.py`、`engine/live_sandbox.py` 的 worker 模式。
- **DoD**：能加载样本 EA 并按事件回调；单测覆盖生命周期顺序与异常隔离（策略抛错不崩 runtime）。

#### T1.2 实现 SimBroker（在现有回测引擎之上扩展）
- **目标**：让回测以 Broker 语义运行，而非 signal-dict。
- **产出**：`strategy-service/app/engine/sim_broker.py`。
- **复用（强制，不要重写）**：`runner.py` 的 `fill / cost / margin / portfolio / market / metrics` 模块。
- **补齐**：ticket 管理、部分平仓、`position_modify`、**对冲 + 净持两种持仓模式**、magic number、多持仓迭代、挂单全类型。
- **DoD**：T0.4 的 5 个样本 EA 全部能在 SimBroker 下跑完一段历史；网格/马丁的跨 tick 状态正确保留；对冲与净持各有单测。

#### T1.3 回测保真度基线
- **目标**：建立"翻译对不对"的度量基准。
- **产出**：`strategy-service/tests/fidelity/`：对至少 1 个有 MT Strategy Tester 报告的 EA，跑 SimBroker 并与 MT 报告做行为 diff（开平仓时点/方向/手数/盈亏）。
- **DoD**：diff 报告生成；偏差项被逐条解释（成本模型/tick 合成假设等），无法解释的偏差用 `NEEDS-DECISION:` 上报。

---

### Phase 2 — 翻译层

#### T2.1 确定性 Transpiler（MQL → SDK 骨架）
- **目标**：用真实 MQL4/5 文法做 AST，机械映射 ~80% 构造，零幻觉、可审计。
- **产出**：`tools/mql-transpiler/`（独立工具，语言可选 Go/Python，需说明）。
- **映射表（最少覆盖）**：`OnTick→on_tick`、`OnInit→on_init`、`OrderSend→broker.order_send`、`OrderClose→broker.position_close`、`OrderModify→position_modify`、`iMA→indicators.ma`、`iRSI→indicators.rsi`、`OrderSelect循环→positions()迭代`、`ArrayInitialize`、`extern/input→@param`。
- **DoD**：对 T0.4 样本对应的 MQL 源码，transpiler 产出可编译 SDK 骨架；无法机械翻译处**显式标记 `// TRANSPILER-GAP:`**，不静默丢弃。

#### T2.2 LLM 补全（只补 transpiler 标记的缺口）
- **目标**：用 LLM 仅填 `TRANSPILER-GAP`，面向**精确 SDK 规格 + few-shot**。
- **产出**：改造现有 `backend/internal/connect/ai/code_assist_handler.go` 的 `TransformCode`：
  - 提示词**严格对齐 T0.1 的 SDK 规格**（彻底废弃现状里那套不存在的 class API，见现状地图第五节）。
  - 修复长度限制冲突（现状：翻译允许 64KB 但 `sandbox_scan.py:24` 拒绝 >10000 字符）——统一上限。
- **DoD**：翻译产物能通过 SDK 运行时加载（不再 100% 校验失败）；few-shot 与规格同源。

#### T2.3 行为校验 Harness（闭合"翻译对不对"）
- **目标**：翻译产物在 SimBroker 跑，与 MT 金标准做行为 diff。
- **产出**：`tools/translation-verify/` + 金标准语料库 `strategy-service/tests/golden/`。
- **DoD**：给定 EA + MT 参考运行，自动产出行为 diff 与覆盖度报告（哪些 MQL 特性未支持：DLL/WebRequest/GUI 等）。

---

### Phase 3 — 实盘闭环（LiveBroker + 风控门 + 沙箱）

> **注意**：Phase 3 同时也修复现状地图第三节列出的实盘三大硬伤。

#### T3.1 LiveBroker（代理到 MT 网关）
- **目标**：用与 SimBroker **完全相同的接口**，把下单代理到 MT 网关现有 RPC。
- **产出**：`strategy-service/app/engine/live_broker.py` + Go 侧对接。
- **复用**：MT 网关 RPC（mt4 43 / mt5 57：`OrderSend/PositionClose/PositionModify/...`）。
- **修复硬伤①（持仓回传）**：`backend/internal/connect/strategy/live_runner.go:129-163` 的 `buildLiveContext` 必须回填 `position/positions/equity/balance/margin`（现状全空）。
- **修复硬伤②（动作集）**：替换 `signalToSide`（`live_runner.go:251-260`）的裸 buy/sell，改走 Broker 接口，支持 `close/close_all/挂单`（现状静默丢弃）。
- **DoD**：paper↔live↔backtest **三方一致性测试**：同策略同输入，三条路径下单意图一致。

#### T3.2 Go 风控门（金钱安全边界）
- **目标**：实现 T0.3 协议，所有下单意图（sim+live）经**同一份 Go gate** 评估。
- **产出**：`backend/internal/risk/gate.go`（按 CLAUDE.md 行数限制拆分）。
- **【决策 D6-A：gate 单点权威，实盘在线过门、回测离线重放过门】**（2026-06-23 拍板，见 ADR-0020）：
  - **金钱安全只允许一份实现**——gate 逻辑只在 Go，**禁止在 Python 镜像/重写 gate 规则**（重写=两套实现漂移，正是本项目要消灭的反模式）。
  - **实盘（在线，权威）**：gate 用**构造期强制注入**进 `LiveBroker` 派发路径，编译期保证不可绕过。每笔意图 `order_send` → `Gate.Evaluate()` → 通过才到 `mthub.PlaceOrder`。
  - **回测（离线，批量）**：SimBroker 照常全速撮合（**不逐 tick RPC**），把意图流喂给**同一个 Go gate** 跑离线重放，产出"第 N 笔会被规则 X 拦"的预检报告。回测目的不是金钱安全（不碰真钱），而是提前暴露策略会被风控拦。
- **⚠️ 现状必须修复（已核实）**：
  - `internal/risk/gate.go` 当前是 **shelf-ware**——全仓库只有 `gate_test.go` 引用它，**无任何生产路径接线**（与 factor DSL 同病）。本 Task 必须把它真正接进 `live_runner.go` 派发路径。
  - 旧的 `internal/risksvc` 是另一套已接线的风控；**不要混用**，明确新 EA gate 与它的边界。
- **DoD**：逐条规则有单测；**`live_runner.go` 派发路径有 `risk.Gate.Evaluate()` 调用且不可绕过（集成测试佐证）**；回测离线重放预检报告产出；超限单被拦截并留痕；全局 kill-switch 可一键停所有实盘；与 schedule 级 `autoTradeEnabled` 协同（后者保留为粗粒度开关）。

#### T3.2b 实盘账户状态真回传（fail-safe equity）
- **目标**：消除实盘风控对假数据的依赖。
- **⚠️ 现状必须修复（已核实）**：`live_runner.go:175-183` 的 `backfillLiveState` 硬编码 `Equity=10000.0 / Balance=10000.0`。依赖 equity 的规则（回撤/日亏/可用保证金）当前在对**假数据**做判断。
- **产出**：接 `AccountStatus` 订阅，把真实 `equity/balance/margin/positions` 回填进 `LiveStrategyContext` 与 gate 的 `AccountState`。
- **fail-safe 约束**：在真实状态接通前，**禁止启用依赖 equity 的规则**——宁可拒单（fail-closed），不可用假 equity 放行。
- **DoD**：实盘 gate 评估读取的是真实账户状态；接通前 equity 相关规则默认拒单或停用，有测试佐证。

#### T3.3 OS 级沙箱
- **目标**：把执行 worker 从语言级隔离下移到 OS/VM 级（D5）。
- **产出**：worker 启动改造 + 部署配置（seccomp-bpf、net namespace 断网、只读 rootfs、非 root、cgroup 限 CPU/内存/PIDs；强档 gVisor/Firecracker）。
- **变更**：RestrictedPython 降级为 lint（不再当安全边界）；合并 `sandbox.py` 与 `sandbox_scan.py` 重复的静态扫描规则。
- **DoD**：逃逸测试（联网/写盘/越权 syscall）全部被内核/VM 拦截；预热 worker 池延迟达标；放开 numpy/pandas 后仍隔离。

---

### Phase 4 — 上线

#### T4.1 可观测性与漂移看板
- **产出**：每 tick 决策 + 下单意图 + 风控裁决 + broker 回执全留痕；回测/实盘一致性仪表盘；翻译覆盖度与行为 diff 报表。
- **DoD**：能按单次策略运行回放 trace；漂移可视化。

#### T4.2 灰度上线
- **产出**：canary 账户 + 小手数灰度；kill-switch 演练。
- **DoD**：灰度流程文档化；一键回滚验证通过。

---

## 4. 复用 / 新建 / 退役清单（防止重复造轮子与造空壳）

| 类别 | 项 |
|---|---|
| **必须复用** | 回测引擎核心（`runner.py` 的 fill/cost/margin/portfolio/market/metrics）、MT 网关 RPC、`schedule_engine.go` 调度、`live_sandbox.py` worker 池模式、前端策略页 |
| **新建** | Strategy SDK、Broker 抽象、SimBroker/LiveBroker、MQL transpiler、Go 风控门、OS 沙箱、行为校验 harness、自定义指标库 |
| **退役/降级** | signal-dict 实盘路径（→ Broker 模型）、RestrictedPython 安全边界（→ lint）、factor DSL（保留给"AI 生成简单策略"档，**不用于 EA**）、`is_production_mode`/production-block 标志（移除或重定义）、`sandbox.py` 误引的 ADR-0016 注释（修正） |

---

## 5. 验收与测试总则

- 每个 Task 的 DoD 必须有**自动化测试**佐证，测试与实现同 PR。
- **关键不变量（贯穿所有 Phase）**：
  1. **同码可信**：同一策略代码在 SimBroker 与 LiveBroker 下，给定相同输入，产生**语义一致**的下单意图（D3）。
  2. **风控不可绕过**：不存在任何绕过 Go 风控门直达 broker 的路径（D6）。
  3. **金额精度**：全链路无 float 承载金额/价格（CLAUDE.md）。
  4. **沙箱不可逃逸**：执行 worker 无法联网/写宿主盘/越权 syscall（D5）。
- **回归基线**：Phase 1 建立的保真度基线（T1.3）在后续每个 Phase 重跑，偏差不得扩大。

---

## 6. 风险与「停下提问」清单（遇到即 `NEEDS-DECISION:`）

| 触发点 | 为什么要停 |
|---|---|
| MT 交易语义边角：requote / 滑点 / swap / 周末跳空 / 部分成交 / stops·freeze level 的精确规则 | 必须 Sim 与 Live 两侧**同规则**建模，定义错会导致实盘漂移 |
| 对冲 vs 净持的差异处理 | MT4 hedging / MT5 两者皆有，SimBroker 必须都支持 |
| 真实 tick 数据缺失时的合成假设 | 影响回测保真度，假设必须显式记录 |
| `iCustom`/自定义指标无法翻译 | 需决定：翻译指标本身 vs 桩化 vs 拒绝 |
| DLL / WebRequest / 文件 IO / GUI | 明确不支持并桩化，覆盖度报告告知用户 |

---

## 7. 一句话交底

**按"契约先行 → Phase 0~4 逐 Task 交付"推进；核心是把 MT 的「事件模型 + 券商语义」忠实复刻成统一 Strategy SDK + 双实现 Broker（回测/实盘同码），下单必经 Go 风控门，执行跑在 OS 级沙箱里；翻译走「确定性 transpiler + LLM 补缺口 + 行为校验」三件套。遇到 MT 语义边角不要猜，停下提问。**
