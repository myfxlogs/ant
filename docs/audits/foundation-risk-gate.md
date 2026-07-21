# 地基审计 · risk-gate

## 1. 核心设计替代方案

**当前**：6 门管线串行（Capability→HardLimit→PlatformLimits→PreCheck→Sizer→BlockAlloc）。11+ 风控规则可用户配置。16 状态 OMS。仿真交易。Guard（强安全网）+ Canary（比例放行新规则试验）。

**替代方案 A**：风控规则硬编码在每个策略里。

不选。集中风控 = 单点可审计、单点可关闭（kill switch）。分散风控 = 永远不知道哪个策略在失控。6 门管线串行是正确设计。

**替代方案 B**：第三方现成风控引擎。

不选。量化交易的风控不是通用逻辑——Kelly sizing、vol targeting、jurisdiction gate 都是领域特有的。自己写是对的。

**结论**：✅ 架构最优。管线模式正确，OMS 16 状态机覆盖订单全生命周期。Guard+Canary 的设计允许新规则灰度试验。

## 2. 上下游契约

**上游**：strategy-runtime 推送 Signal（`SignalAction`）→ risk-gate 管线。mt-gateway 推送订单事件（成交/拒绝/过期）→ OMS 状态更新。

**下游**：mt-gateway 执行已核准订单。NATS 推送实时 PnL。

**隐患**：Signal Pipeline 不检查保证金（RequiredMargin RPC 未封装）。这意味着"能不能买得起"的检查是在 MT 服务器端做的，不是在 risk-gate 做的。MT 拒绝 = 用户看到"订单被拒"，但不知道是因为保证金不够还是 broker 故障。

**另一个隐患**：OMS 16 状态机。`UNKNOWN` 状态有 30s 超时→fallback 到 `RECONCILING`。但如果对账也失败了，订单会永远卡在 RECONCILING 吗？**需要确认 OMS 有没有最终兜底状态。**

## 3. 已知架构债务

| 债务 | 严重度 | 方案 |
|------|--------|------|
| 不检查保证金（依赖 MT 拒绝） | 🔴 | 接入 RequiredMargin RPC（已写入 mt-gateway plans/rpc-expansion.md P0），在 PreCheck 阶段加保证金规则 |
| OMS 卡死兜底待验证 | 🟡 | 确认 RECONCILING 状态的超时和 fallback 逻辑 |
| 风控规则按 broker 差异化 | 🟡 | 当前规则全 broker 通用。不同 broker 的不同保证金/杠杆/交易时段未体现 |

## 4. 总评

架构最优。最紧迫的是保证金预检——当前依赖 MT 拒绝，用户体验和风控都不够好。P0 优先级正确。
