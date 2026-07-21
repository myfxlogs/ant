# 地基审计 · 功能第一性原则

> 每块四问：最小集？超出？缺失？边界？

---

## 1. mt-gateway — 连接 MT 交易服务器

**最小功能集**：连接 MT4/MT5 服务器，执行下单/平仓/改单，查询持仓/订单历史，接收实时报价/订单事件。没有它，平台无法和任何 broker 通信。

**超出最小集？** 有。mthub 里的幂等门和对账门是风险控制逻辑——它们应该属于 risk-gate。mthub 的职责应该是**转发**订单到 MT 并**返回**结果，不是**判断**这个订单是否有效。

当前：`PlaceOrder` → mthub 做幂等校验 → OMS 状态机 → MT 下单。幂等校验放在 mthub 意味着所有路径（包括内部测试、手动调 API）都经过了它——这是防御性设计，但模糊了 mt-gateway 和 risk-gate 的边界。

**缺失？** 没有。核心交易能力完整。熔断器、告警、降级是质量属性，不是功能缺失。

**边界**：`OrderExecutor` 接口是清晰的分界线。但 mthub 持有 account-mgmt 的凭据解密逻辑和 risk-gate 的幂等逻辑——越界了。mthub 不应该知道"账户属于谁"和"订单有没有重复"。

**结论**：功能集正确，但幂等门和对账门的归属不对。它们应该在 risk-gate 的 PreCheck 阶段，而不是在 mthub 内部。

---

## 2. market-data — 行情数据管线

**最小功能集**：接收 MT 报价流 → 去重 → 质量检测 → 归一化 → 存储（PG + ClickHouse） → 推送（NATS）。没有它，回测没历史数据，实盘没实时 bar。

**超出最小集？** 在线指标计算（SMA/EMA/Bollinger/RSI/MACD 流式）。这属于 strategy-runtime 的指标库——market-data 的职责是**存储和分发原始数据**，不应该计算衍生指标。当前把流式指标放在 market-data 可能是因为"在数据源头算更高效"——但这是性能优化，不是正确的功能归属。

**缺失？** 回填系统的健康监控——backfiller 跑失败了没人知道。

**边界**：market-data → strategy-runtime 的边界是 `BarSource` 接口。market-data 提供原始 bar，strategy-runtime 自己算指标。这个边界是干净的——但在线指标越过了它。

**结论**：在线指标应该移到 strategy-runtime/indicators/。market-data 只做采集+存储+分发。回填健康监控缺失。

---

## 3. mql-compiler — 策略编译管线

**最小功能集**：MQL/Python 源码 → tree-sitter 解析 → IR → 字节码 → VM 执行。没有它，所有策略都无法运行。

**超出最小集？** 盲区追踪（`ast_coverage.go`）。这不是编译器该做的事——编译器只管"能不能编译"，不应该管"编译出来的策略功能完不完整"。盲区分析属于 agent-engine 的职责。当前放在编译器里是因为方便——在编译阶段顺手做了覆盖率统计。但正确的边界是：编译器输出 IR + 编译错误列表，agent-engine 拿这两个输出去判断有没有盲区。

**缺失？** IR 层的独立性验证。如果 MetaQuotes 改变 MQL 语法，IR 层是否需要跟着改？如果 IR 是 MQL 的超集，改 MQL 不需要改 IR。如果 IR 是 MQL 的子集，新特性需要扩展 IR。**需要确认 IR 的语义定位。**

**边界**：mql-compiler → strategy-runtime 的边界是 Bytecode。mql-compiler 输出字节码，strategy-runtime 加载执行。这个边界是干净的——没有任何泄漏。

**结论**：盲区追踪应该移到 agent-engine。IR 语义定位待确认。

---

## 4. strategy-runtime — 策略执行引擎

**最小功能集**：加载编译后的字节码 → 按 bar/tick 触发策略逻辑 → 产生 Signal → 通过 Broker 接口执行交易。Strategy 接口是回测和实盘的统一入口。

**超出最小集？** 50+ 技术指标库。这些指标是策略的**输入**，不是策略执行引擎的职责。它们应该是一个独立的共享库（类似 `math` 或 `ta` 包），被策略引用，不属于 strategy-runtime 本身。但当前目录结构把指标放在 `strategy/indicators/` 下——这是合理的：指标跟着策略接口走，不独立打包。不算超出，只是命名上容易让人误以为指标是 runtime 的一部分。

**缺失？** 没有。Strategy 接口 + LiveRunner + Engine 构成了完整的执行层。

**边界**：strategy-runtime 通过 Signal 接口输出到 risk-gate。Signal 包含 `SignalAction` + `Volume` + `SL/TP`——这是策略和执行之间的最小契约。干净。

**结论**：✅ 功能集正确。指标库的位置合理（随策略接口一起发布）。没有超出，没有缺失。

---

## 5. backtest-engine — 回测验证

**最小功能集**：消费历史 K 线，通过 SimBroker 模拟真实 broker 行为，执行策略，输出净值曲线 + 交易统计。没有它，策略质量无法验证。

**超出最小集？** `mt_report.go` — MT 原生报告格式。这不是回测引擎的职责——报告格式属于前端展示层或策略市场（质量门槛附件）。回测引擎应该输出标准化的 `BacktestMetrics` proto，前端/市场自己去格式化。

**缺失？** 多品种回测的品种间相关性模拟。当前多品种回测是独立运行的——每个品种各自的 bar 序列，各自触发策略。真实场景中 EURUSD 和 GBPUSD 的 bar 是同步到达的，策略需要同时看到两个品种的当前 bar。**当前实现不支持真正的同步多品种回测。**

**边界**：backtest-engine 消费 market-data 提供的历史 K 线，消费 mql-compiler 提供的字节码。输出 `BacktestMetrics` 给 agent-engine 和 strategy-marketplace。边界干净。

**结论**：`mt_report.go` 应该移出。真正的多品种同步回测是功能缺失。

---

## 6. risk-gate — 风控 + OMS

**最小功能集**：每笔 Signal 经过 6 门管线 → 核准 → OMS 状态机跟踪 → 下发 mt-gateway 执行。没有它，策略可以直接向 broker 下单，无安全网。

**超出最小集？** 仿真交易（paper trading）。仿真交易的功能和真实风控 99% 一样——只是下单走 SimBroker 而不是 mt-gateway。它应该是一个**配置切换**：Signal → risk-gate → 选择 broker（真实 or 仿真）→ 执行。当前 paper_trading 作为独立模块存在是合理的——它需要自己的 SimBroker 和账户管理。不算超出。

**缺失？** 保证金预检（RequiredMargin RPC 未接入）——已在层1红标。broker-specific 风控规则——不同 broker 的不同保证金/杠杆/交易时段未体现。

**边界**：风险控制逻辑应该在 risk-gate，但 mt-gateway/mthub 里的幂等门也属于这里。这是边界模糊——不是功能缺失。

**结论**：保证金预检缺失。mthub 中的幂等逻辑应该移入 risk-gate 的 PreCheck 阶段。其余正确。

---

## 7. api-gateway — API 层

**最小功能集**：定义 proto → 生成代码 → ConnectRPC handler → SSE 流。所有后端模块通过这层暴露接口。没有它，前端和策略引擎无法通信。

**超出最小集？** 335 个 RPC。不是每一个都是"必须的"。52 个 service 中有些是试验性的（MarketRegime、IndicatorCatalog、ExecutionAlgo）——它们上线了但可能没人用。**RPC 数量本身不是问题，问题是没有淘汰机制。**

**缺失？** API 版本管理。proto 服务改了字段，旧版前端会不会崩？当前依赖"前后端一起部署"来保证一致——但这要求所有客户端同时升级。如果将来有外部 broker 通过 API 接入，版本不一致会导致静默的数据错误。

**边界**：api-gateway 是薄层——它不包含业务逻辑，只管序列化/反序列化/路由/认证。这个边界是正确的。

**结论**：RPC 膨胀需要周期性清理。API 版本管理缺失——当前规模不需要，但将来外部接入时需要。

---

## 8. account-mgmt — 账户管理

**最小功能集**：用户注册/登录/认证，MT 凭据管理（CRUD + 验证），钱包，WebAuthn，HD 冷签名提现。没有它，多用户隔离不存在。

**超出最小集？** AI token 额度管理（`quota_checker.go`）。这应该属于 agent-engine 或独立计费模块。AI token 的配额消耗和使用统计放在 account-mgmt 是因为"和用户绑定"——但正确的归属是计费系统。

**缺失？** 用户数据可移植性（GDPR 导出）。用户注销时的数据清理完整性。

**边界**：account-mgmt 被 13 个模块依赖。这个依赖广度意味着 account-mgmt 的任何修改都会波及整个系统——这是正确的：认证和账户是基础设施。但 AI token 额度管理不应该在这里。

**结论**：AI token 额度管理应该移到计费模块。用户数据可移植性缺失。

---

## 汇总

| 块 | 最小集正确？ | 超出？ | 缺失？ | 边界？ |
|----|------------|--------|--------|--------|
| mt-gateway | ✅ | 幂等门属于 risk-gate | — | 越界 |
| market-data | ✅ | 在线指标属于 strategy-runtime | 回填监控 | 越界 |
| mql-compiler | ✅ | 盲区追踪属于 agent-engine | IR 独立性待确认 | ✅ |
| strategy-runtime | ✅ | — | — | ✅ |
| backtest-engine | ✅ | mt_report 属于前端 | 多品种同步回测 | ✅ |
| risk-gate | ✅ | — | 保证金预检 + broker 规则 | 越界（接收 mthub 逻辑） |
| api-gateway | ✅ | RPC 膨胀 | 版本管理 | ✅ |
| account-mgmt | ✅ | AI token 额度属于计费 | GDPR 导出 | 越界 |

**共同模式**：功能集都是正确的——没有块做了不该做的事，也没有块漏了地基该有的能力。问题集中在**边界**——5 个块有功能归属错误。不是架构缺陷，是增量开发中自然形成的职责漂移。
