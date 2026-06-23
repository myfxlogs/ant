# ADR-0020 · EA 完全替代：统一 Strategy SDK + 双实现 Broker

- **状态**：Proposed
- **日期**：2026-06-23
- **决策者**：架构组 + 人类负责人
- **关联 spec**：`docs/spec/30-strategy-sdk.md`（T0.1 已冻结）、`docs/spec/31-risk-gate.md`（T0.3 已冻结）、`docs/spec/26-ai-strategy-generation.md`（需修订漂移）
- **关联 ADR**：ADR-0012（统一回测/实盘路径，本 ADR 是其在 EA 场景的具体落地）、ADR-0014（持仓级风控）、ADR-0015（仿真交易）
- **关联文档**：`docs/audit/2026-06-22-策略EA执行系统-现状地图.md`、`docs/plan/2026-06-22-EA完全替代-完整解决方案.md`、`docs/plan/2026-06-23-EA完全替代-实施说明书(DeepSeek).md`

## 1. 背景

目标是**完全替代 MetaTrader 跑 EA**：任意 MT4/MT5 EA 经翻译后，在回测与实盘中行为与 MT 一致，安全、可多租户运行。

现状勘察（见现状地图）暴露三个根因级问题：

1. **翻译目标与运行时错配**：`TransformCode` 提示词让 LLM 生成一套运行时根本不存在的 class API（`self.on_bar/self.buy/bar.close`），而真实运行时只认 `def run(context)` + signal dict → 翻译产物近乎 100% 校验失败。
2. **实盘是薄原型**：`buildLiveContext` 不回传持仓/权益；`signalToSide` 只映射裸 buy/sell，`close`/挂单被静默丢弃；无逐单风控门。回测支持的丰富语义，实盘只落地了裸市价买卖。
3. **安全是"虚假安全感"**：真正隔离只有多进程；RestrictedPython 可绕过且对 C 扩展无效；无 seccomp/断网/只读 FS/非 root/cgroup。

更深层的判断：**EA 真正依赖的只有「事件模型」（OnInit/OnTick/OnTimer/OnTrade/OnDeinit）和「交易服务器语义」（订单/持仓/成交/保证金/成交规则/netting·hedging…）。** 把这两样建模忠实，翻译就退化为机械映射。当前方案的错误在于试图把 EA 塞进"每根 bar 返回一个 signal 字典"的薄模型。

此外存在文档漂移需一并修正：
- `strategy-service/app/engine/sandbox.py` 注释误引 ADR-0016（实为 bar-revision-cascade，与沙箱无关）。
- `docs/spec/26-ai-strategy-generation.md` 宣称的 "生产=DSL+ONNX、Python 仅研究" 与代码现实（Python 即实盘运行时）相反；factor DSL 与 ONNX 实为货架闲置。

## 2. 决策

建立**统一 Strategy SDK + 双实现 Broker**架构，回测与实盘共用同一份策略代码，差异只在注入的 Broker 实现。固化以下 7 条决策（D1–D7）作为后续实现依据：

| # | 决策 |
|---|---|
| D1 | **目标语言 = Python Strategy SDK**（产物可读可改、LLM 友好）；自研 MQL→IR VM 列为后手，本期不做。 |
| D2 | **建模对象 = 「券商 + 事件模型」**，而非"语言"。SDK 忠实镜像 MQL 生命周期与交易服务器语义。 |
| D3 | **回测/实盘共用同一份策略代码**，差异仅在注入的 Broker 实现（SimBroker / LiveBroker）。 |
| D4 | **引擎编排用 Go，策略运行在 Python worker**（沿用现状分工）。 |
| D5 | **安全边界下移到 OS/VM 级**（seccomp + 断网 + 只读 FS + 非 root + cgroup；强档 gVisor/microVM）；RestrictedPython 降级为 lint。 |
| D6 | **所有下单意图（sim 和 live）必经 Go 风控门**；金钱安全与策略对错解耦。 |
| D7 | **退役 signal-dict 实盘路径**；factor DSL/ONNX 不用于 EA。 |

## 3. 备选方案

| 方案 | 优点 | 缺点 | 否决理由 |
|------|------|------|----------|
| 继续修补 signal-dict 模型 | 改动小 | 永远无法忠实表达 EA 的持仓/挂单/事件语义 | 治标不治本，实盘不可信 |
| 确定性 MQL→IR + 自研 VM（Go/Rust） | 构造即安全、原生速度、无 RCE | IR 不可读、LLM 难翻译、工程量巨大 | 牺牲可读性与可改性；列为后手 |
| 沿用 RestrictedPython 作安全边界 | 无需运维改造 | 可绕过、对 C 扩展无效 | 不满足多租户安全 |
| **统一 SDK + 双 Broker + OS 沙箱（本决策）** | 翻译友好、回测=实盘同码、安全成熟解 | tick 回测性能需补偿 | 采纳；性能用编译版 tick-core/向量化补齐 |

## 4. 后果

- **正面**：翻译目标与运行时对齐（翻译可通过校验）；回测结论可预测实盘（同码）；金钱安全收敛到一道 Go 风控门；安全爆炸半径被内核/VM 锁死，反而可放开完整 Python（numpy/pandas）。
- **负面**：需新建 Strategy SDK、Broker 抽象、SimBroker/LiveBroker、MQL transpiler、Go 风控门、OS 沙箱、行为校验 harness——工程量大；tick 事件回测性能需专门补偿。
- **中性**：现有回测引擎（fill/cost/margin/portfolio）被复用并包进 SimBroker；signal-dict 路径退役；factor DSL 保留给"AI 生成简单策略"档，不用于 EA。

## 5. 实施约束

1. **契约先行**：先冻结 SDK 规格（spec/30）、Broker 接口、风控门协议（spec/31），再写实现。接口桩落点：`strategy-service/app/sdk/`。
2. **SDK 必含**：生命周期 `on_init/on_tick/on_bar/on_timer/on_trade/on_deinit`；Series 采用 MQL 逆序索引（`close[0]`=当前）；Symbol 元数据、Account、Indicators（含 `i_custom`）、Trade API（经 `self.broker`，全挂单类型 + magic/comment/deviation/type_filling）；netting/hedging。
3. **金额/价格全链路用 Decimal 或定点整数，禁止 float**（遵守 CLAUDE.md / 现状漂移修正）。
4. **Broker 接口**：`order_send/position_modify/position_close/order_delete/positions/orders/account/symbol_info/server_time`；retcode 对齐 MT 返回码语义。
5. **风控门**：proto `OrderIntent`/`RiskDecision`（`proto/ant/v1/risk_gate.proto`）；下单意图经同一份 Go gate 评估，不可绕过。
   - **决策 D6-A（2026-06-23）：gate 单点权威，实盘在线过门、回测离线重放过门。** 金钱安全只允许一份实现（Go），**禁止在 Python 镜像/重写 gate 规则**。实盘：gate 构造期强制注入 `LiveBroker` 派发路径，编译期保证不可绕过；每笔 `order_send → Gate.Evaluate() → mthub.PlaceOrder`。回测：SimBroker 全速撮合，意图流喂给同一 Go gate 离线批量重放，产出风险预检报告（不逐 tick RPC）。
   - **已核实待修**：`internal/risk/gate.go` 当前为 shelf-ware（仅 `gate_test.go` 引用，零生产接线），须接进 `live_runner.go`；`backfillLiveState`（live_runner.go:175-183）硬编码 `equity/balance=10000.0`，实盘前必须接 `AccountStatus` 真回传，接通前 equity 相关规则 fail-closed。与旧 `internal/risksvc` 明确边界，不混用。
6. **复用，不重写**：回测引擎核心、MT 网关 RPC、`schedule_engine.go`、`live_sandbox.py` worker 模式。
7. **文档漂移修正**：删除/修正 `sandbox.py` 对 ADR-0016 的误引；修订 spec/26 关于 DSL/ONNX 生产路径的错误叙述。
8. **不造空壳**：每个新模块必须有真实调用方或测试驱动（吸取 factor DSL/ONNX 教训）。

详细任务拆解见 `docs/plan/2026-06-23-EA完全替代-实施说明书(DeepSeek).md`（T0.1 ~ T4.2）。

## 6. 验证方式

- **表达力（Phase 0）**：5 个代表性 EA（单进单出/网格/马丁/多单对冲/带自定义指标）能用 SDK 手写表达，无缺口。
- **同码可信（核心不变量）**：同一策略代码在 SimBroker 与 LiveBroker 下，给定相同输入，下单意图语义一致。
- **保真度**：至少 1 个有 MT Strategy Tester 报告的 EA，SimBroker 行为 diff（开平仓时点/方向/手数/盈亏）偏差逐条可解释。
- **风控不可绕过**：不存在绕过 Go 风控门直达 broker 的路径；逐条规则有单测；kill-switch 可一键停所有实盘。
- **沙箱不可逃逸**：执行 worker 联网/写宿主盘/越权 syscall 全部被内核/VM 拦截。
