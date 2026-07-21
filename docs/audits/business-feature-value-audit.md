# 业务功能减法审计

> 审的不是代码——是"零售交易者会不会用"。
> 目标用户：不懂编程、想买策略赚钱的散户。

---

## 逐功能评估

### Market Regime Detection（市场环境检测）

**做什么**：AI 分析当前市场处于什么"状态"（趋势/震荡/高波动/低波动）。

**零售用户会用吗**：❌ 不会。散户不知道"regime"是什么意思，也不在乎——他们只看策略赚不赚钱。这是一个机构量化概念，放在零售产品里是认知噪音。

**建议**：从面向用户的界面移除。后端保留（agent-engine 内部用 regime 信息辅助策略生成可以，但不要作为独立产品功能展示给用户）。

---

### Indicator Catalog（指标目录）

**做什么**：列出所有可用技术指标和参数。

**零售用户会用吗**：❌ 不会。他们不写策略——指标目录是给开发者看的。MQL5 文档里已经有完整的指标参考。

**建议**：移除。提供者写策略时用 MQL 文档，不需要我们的指标目录。

---

### Execution Algo（TWAP/VWAP 算法执行）

**做什么**：大单拆分执行（时间加权/成交量加权平均价）。

**零售用户会用吗**：❌ 不会。零售交易者下的是 0.01-1 手的小单，不需要算法执行。这是机构功能。

**建议**：移除。8 个 Go 文件 + proto + 前端 AlgoDashboard——全是给机构用的。

---

### Strategy Experiment（策略实验/A/B 测试）

**做什么**：并排对比策略变体的回测效果。

**零售用户会用吗**：❌ 不会。这是量化研究员的工作流。散户一次只买一个策略。

**建议**：移除。

---

### Strategy Asset（多资产策略分析）

**做什么**：分析策略在不同品种上的表现。

**零售用户会用吗**：❌ 不太可能。散户选好一个策略、一个品种就买了。

**建议**：从面向用户移除。后端保留（agent-engine 可能用来自动推荐品种）。

---

### Economic Data（经济数据日历）

**做什么**：展示经济指标发布时间和预测值（非农、CPI、利率决议等）。

**零售用户会用吗**：🟡 可能会看——但 TradingView/ForexFactory 已经做了，我们不可能比他们更好。

**建议**：移除。不是核心差异化的功能。

---

### Schedule Health（策略调度健康）

**做什么**：监控策略定时执行的状态。

**零售用户会用吗**：❌ 不会。这是运维功能——Admin 才需要看。

**建议**：从用户面移除，保留 Admin 面。

---

### Factor DSL（因子领域语言）

**做什么**：17 个文件。让用户用 DSL 定义技术因子。

**零售用户会用吗**：❌ 不会。这是给量化开发者用的，散户不可能写 DSL。

**建议**：评估是否被 agent-engine 内部使用。如果只暴露给用户——移除。

---

### Batch Tuning / Smart Tuning（批量/Smart 参数优化）

**做什么**：自动搜索策略最优参数组合。

**零售用户会用吗**：❌ 不会。散户买策略，不调策略参数。

**建议**：从用户面移除。后端保留（agent-engine 内部优化时用）。

---

### Log Management（日志管理）

**做什么**：Admin 查看系统日志。

**零售用户会用吗**：❌ 不会。

**建议**：保留——Admin 功能，不出现在用户面。

---

## 核实后的重新评估

> 实际核查：这些功能**全在线上运行**。`handlers_sre.go` 注册了 7 个 service，前端有对应页面。不是废弃实验——是上线了的完整功能。

| 功能 | 线上? | 有前端? | 谁用? | 决定 |
|------|-------|---------|-------|------|
| Market Regime | ✅ handlers_sre.go | ✅ MarketToolsPage(regime tab) | ⚠️ 散户看不懂"regime"概念，但可能好奇看 | 暂时保留，观察用量 |
| Execution Algo | ✅ connect/algo/ | ✅ AlgoDashboard(/trading/algos) | ❌ 机构功能。TWAP/VWAP 散户不会用 | 建议移除或隐藏 |
| Strategy Experiment | ✅ handlers_sre.go | ⚠️ BatchTuningPanel 部分引用 | ⚠️ 进阶用户可能调参数 | 暂时保留，观察用量 |
| Strategy Asset | ✅ handlers_sre.go | ❌ 无独立页面，API only | ⚠️ 后端可能被 agent 使用 | 确认 agent 依赖后决定 |
| Indicator Catalog | ✅ handlers_sre.go | ❌ 无前端页面 | ❌ 只有 API，没人用 | 建议移除 |
| Schedule Health | ✅ handlers_sre.go | ❌ 无前端页面 | ✅ Admin 监控用 | 保留 |
| Economic Data | ❌ 未注册 | ❌ 无前端 | ❌ proto 存在但未接入 | 移除 proto |
| Batch/Smart Tuning | — | ✅ 策略工作区面板 | ⚠️ 进阶用户功能 | 暂时保留，属于策略工作区 |
| Log Management | ⚠️ 仅 Admin | ✅ Admin 页 | ✅ Admin | 保留 |

**修正结论**：

- **明确该移除的**：Execution Algo（机构功能）、Indicator Catalog（零前端用途）、Economic Data（未接入）
- **该观察用量的**：Market Regime、Strategy Experiment、Strategy Asset——它们已经上线了，在用户面前。不知道有没有人用，先不动，看数据。
- **该保留的**：Schedule Health、Log Management、Batch/Smart Tuning——Admin 或进阶用户功能

**减法幅度从"砍 9 个"修正为"砍 3 个 + 观察 3 个"。** 之前的判断太激进——上线了的功能不能凭推测定生死。
