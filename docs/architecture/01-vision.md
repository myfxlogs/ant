# 01 · 设计哲学：MT 是地基

## 1. 第一性原理

ant 是量化交易平台。量化交易的全部价值链：

```
行情数据 → 因子 → 信号 → 回测验证 → 仿真交易 → 风控 → 下单 → 持仓 → 复盘
                                                ↑
                                         AI 策略生成（自然语言 → 策略代码）
```

**链条最左端的"行情数据"决定其他一切的上限**。如果 tick 丢失/失序/被脏数据污染，下游所有计算都是垃圾。所以：

> **MetaTrader 接入层的可靠性 = ant 整个项目的天花板。**

地基不能凑合。这是 v2 重写的根本动机。

### 1.1 AI 策略生成使命

ant 的设计初衷：**"让普通人都能用上量化交易系统"**。

非专业散户通过自然语言描述交易想法（"BTC 突破 7 日均线就买入"），AI 自动生成专业量化策略代码，并在回测/仿真/实盘环境中运行。策略的代码生成、回测验证、风险检查都由系统自动完成，用户只需要用中文描述意图。

AI 策略生成不是附加功能——它是 ant 区别于传统量化平台的**核心差异化能力**，必须作为正式架构层（L6 `ai/`）纳入系统设计。

## 2. 六条设计准则

### 准则 1 · 单一数据路径（One Source of Truth）

所有业务读 K 线/tick 必须走 ClickHouse `md_bars/md_ticks`。**禁止任何业务代码直接调用 mt4client/mt5client/kline_service**。

→ 反模式（v1 现状）：`connect/market_service.go` 直接 import `service/kline_service` 直接 import `internal/mt4client` → 三层耦合，canonical 不一致。

### 准则 2 · 可观测性是一等公民

每一层都必须暴露：
- **Prometheus 指标**：tick rate、drop count by reason、ch write latency、circuit state、account state、signal-to-execution latency
- **健康端点**：`/healthz`（进程活）、`/readyz`（关键依赖就绪）；单账户状态走 ConnectRPC `MtHubService.GetAccountStatus` + Prom Gauge `mt_account_connected`（决策 RV-C4）
- **结构化日志**：每条 ERROR/WARN 必带 `account_id`/`broker`/`symbol`/`trace_id` 四字段

→ "不可观测的地基 = 摸黑救火"。

### 准则 3 · 故障恢复内建

行情链路必须默认能扛住：
- 单个 broker 不可达（其他账户照常）→ **CircuitBreaker**
- ClickHouse 短暂宕机（30s-5min）→ **SpillWriter + Replay**
- 进程意外重启 → 启动时 replay 未提交 spill + **订单状态对账恢复**
- broker 重连后重发历史 tick → **Tick dedup window**

每一种故障必须有：测试用例 + Prometheus 计数 + runbook 条目。

### 准则 4 · canonical 在最早的位置

`(broker_X, "BTCUSDm")` 与 `(broker_Y, "BTCUSD.pro")` 在**进入 ant 系统的第一行代码**就规范化为 `BTCUSD`。下游全部用 canonical。

→ Normalizer 在 adapter 出口、mdgateway 入口运行，写入 CH 时同时保留 `symbol_raw` 与 `canonical` 两列做审计。

### 准则 5 · 代码量是质量指标

**MT 接入总 LOC ≤ 1000 行**（不含测试、不含生成代码）。超过即代码异味。

→ 核心是抽象层次正确：mtapi proto → adapter (~80 行/平台) → Gateway 接口 → mdgateway 编排（~200 行）→ mthub 会话（~150 行）。多余的 pool/connection_methods/search_methods 都是错误的抽象层次。

### 准则 6 · 回测即实盘（Same Code Path）

回测和实盘必须走**完全相同的** factor → signal → execution 代码路径。唯一差异是数据源（`LiveSource` vs `ReplaySource`）和执行终端（`LiveExecutor` vs `PaperExecutor`）。

factorsvc 和 quantengine 通过 `Source` interface 接收数据，下游业务逻辑零差异。禁止为回测建立独立的因子计算或信号推理代码。

→ 回测赚钱、实盘亏钱的根本原因就是代码路径分叉。消除分叉 = 消除回测不可信。

## 3. 与 alfq 的关系

ant v2 **不是 alfq 的克隆**。设计采纳 alfq 的层次划分（这是经过验证的好架构），但在以下维度做了**有意识的增强**：

| 维度 | alfq | ant v2 | 增强理由 |
|---|---|---|---|
| 熔断 | 无 | CircuitBreaker（滑动窗口） | broker 故障常见，单 broker 不该带垮全局 |
| Spill 旋转 | 单文件 append | 按大小/时间旋转 | 长时间 CH 故障下避免单文件超过几 GB |
| Tick dedup | 无 | 100 条窗口 hash | broker 重连重发 tick 在生产中验证存在 |
| Quality dropped reason | 单一计数 | 三类 label（bid_gt_ask/outlier/gap） | 排障时区分 broker 端 vs 网络端问题 |
| 健康检查粒度 | 服务级 | 服务级 + 账户级 | 单账户掉线对 SRE 的可见度 |
| 回测/实盘统一 | 分离路径 | Source 接口抽象（Live/Replay） | 回测结果可信度 |
| AI 策略生成 | 无 | 自然语言→策略代码（L6 ai/） | 散户零编程门槛 |
| 订单状态机 | 无 | 10-state 镜像 + 幂等 + 对账恢复 | 崩溃后自动恢复 |
| 仿真交易 | 无 | PaperExecutor + 滑点/延迟模型 | 策略上线前安全验证 |
| 持仓风控 | 无 | 4 项同步阻断 + 异步 margin call | 零售用户安全网 |

而**坚决不偏离** alfq 的：
- 7 层架构（adapter → mdgateway → mthub → factorsvc → quantengine → oms → ai）
- ClickHouse 4 表 schema
- 因子 DSL Go/Py 双引擎
- DSL+ONNX 替代 Python 沙箱（生产路径）

## 4. 不做什么（划清边界）

| 不做 | 原因 |
|---|---|
| 不做自研 mtapi 替代 | mtapi 协议复杂；让 mtapi.io 维护 broker 兼容性 |
| 不做行情聚合服务（bar 重组） | broker 提供的 1m/5m bar 不可信但可作 hint；以本地 tick→bar 为准 |
| 不做"撮合模拟"在 mdgateway | 撮合属于 paper/ 仿真交易模块职责 |
| 不做 PG 时序存储 | PG 不是时序库，硬上撑不过 1000 acct × 1m sample |
| 不在生产路径跑 Python | 见 ADR（沙箱降级） |
| 不做回测专用代码路径 | 回测与实盘必须走同一 Factor/Quant/OMS 管线 |
| 不做策略市场变现（M11 前） | 核心量化系统上线稳定后再启动 |
| 不做"因子计算双写"（回测版+实盘版） | 单一 factorsvc 接受 Source interface |

## 5. 验收哲学

**"代码完成"不等于"地基达标"。**

地基达标的唯一标准是：

```
连续 7 天，10 个测试账户跨 3 个 broker：
  - tick → CH 写入零丢失
  - 模拟 broker 网络中断 → 自动恢复，spill 全部 replay
  - 订单状态机崩溃恢复：kill -9 后重启，30s 内订单对账一致
  - 回测路径执行结果与同期实盘信号偏差 < 0.1%
  - AI 生成策略在回测中表现与人工策略同分布
  - Prometheus 指标完整，无未告警的异常
  - 全部业务读路径走 CH，grep 验证 0 处直调 mt4client/mt5client
```

达不到，就不算 v2 落地。
