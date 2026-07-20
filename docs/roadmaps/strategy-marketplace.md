# Strategy Marketplace — 设计文档

> **定位**：📐 架构总纲 — 本文档是策略市场的权威设计来源。市场定位、数据模型、API 面、Phase 规划、依赖关系、指标目标均在此定义。
> **施工文档**：`docs/plan/marketplace/` — GLM 按 Phase 逐模块执行，每个任务有精确文件路径+代码+验收标准。
>
> **最后更新**：2026-07-20 (v2 修订)
> **状态**：Phase 0 完成；v2 重排了战略取舍、风险优先级与散户价值闭环顺序
>
> **v2 修订要点**（详见「战略取舍」与「执行顺序（修订）」）：
> 1. 明确商业叉路：“第三方市场” vs “AI 自营策略产品”，不再叠加态。
> 2. 护城河重新定义：真壁垒是**信任 + 流动性**（Phase 1），编译器是供给侧获客工具。
> 3. 新增 **Phase 0.5 合规/牌照**为生死级前置门（早于任何 AI 供给/跟单）。
> 4. 散户价值闭环（实盘战绩 → 跟单 UI → 提现）上移至 Phase 1-2；AI 批量灌库降级。
> 5. KPI 北极星改为**买方结果（盈利用户数/留存）**，剔除 “AI 策略占比” 作为目标。
> **关联**：[[project-roadmap]] · [[ADR-0024]]
> **实施文档**：
> - Phase 1: [信任基础设施](plan/marketplace/phase-1-trust-infrastructure.md)
> - Phase 2: [AI 策略供给](plan/marketplace/phase-2-ai-strategy-supply.md)
> - Phase 3: [增长引擎](plan/marketplace/phase-3-growth-engine.md)
> - Phase 4: [平台运营](plan/marketplace/phase-4-platform-ops.md)
> - Phase 5: [护城河](plan/marketplace/phase-5-moat.md)

---

## 市场分析

### 市场空间

| 维度 | 数据 |
|------|------|
| MT 生态规模 | MT4 **1500 万**活跃用户，MT5 占零售 CFD 交易量 **62%**，**85%** 经纪商至少部署一个 MT 平台 |
| 算法交易软件市场 | 2025 年 **$27.4 亿**，9.3% CAGR → 2029 年 $38.5 亿 |
| 量化投研平台 | 2024 年 **$19.6 亿**，9.9% CAGR → 2031 年 $37.8 亿 |

### 竞品格局

| 竞品 | 模式 | 与 AlphaForge 的差异 |
|------|------|---------------------|
| Pineify 4.0 | AI 生成 MQL 源码 → 手动 MetaEditor 编译 | 无编译器、无原生执行引擎 |
| MetaTrader Cloud IDE | 云端 IDE + AI 辅助写 MQL | 仍需 MT 终端运行 |
| VisualMQL | 可视化节点 + AI → MQL5 | 无编译到 Go、无独立执行 |
| eToro Tori | AI Agent on eToro 封闭生态 | C 端散户、不做开发者工具 |
| CoinQuant | No-code 加密策略 | 加密专属、无 MT 集成 |
| AgentQuant (OSS) | ReAct Agent + 回测循环 | 研究用途、无生产执行 |

**AlphaForge 的独特位置**：唯一一家做 **MQL → tree-sitter → IR → Bytecode → Go VM 编译器** 的平台。竞品全部选择"生成 MQL 源码 → 在 MetaEditor 编译"的轻路径。我们的编译器是确定性管道，不可绕过。

> **护城河定位纠正（v2）**：编译器是真技术壁垒，但它服务的是**供给侧（开发者）**——买方（散户）只看收益曲线，对编译器零感知。因此它是**获取供给的工具**，不是锁住需求的壁垒。且它建在 MetaQuotes 单方控制的私有语言上（平台依赖风险，见风险表 R-P）。**市场的真护城河是流动性（买卖双边）+ 可验证的真实战绩（Phase 1）**；编译器是特性，信任才是壁垒。“护城河”实际在 Phase 1，而非 Phase 5。

### 用户痛点（MQL5 社区验证）

| 痛点 | 严重度 | AlphaForge 解决方案 |
|------|--------|-------------------|
| **MQL ↔ Python 集成地狱** — IPC (ZeroMQ/DLL/REST) 比策略开发本身更耗时 | 🔴 致命 | 编译器消除 IPC：MQL → Go VM 原生执行，零跨进程通信 |
| **MT 策略测试器太慢/功能受限** — 无法 CI/CD、无法复杂统计 | 🔴 致命 | Go 原生回测引擎（SimBroker），服务端运行，可 CI/CD |
| **MQL 语言弱** — 无包管理、无测试框架、统计库有严重 BUG | 🟡 高频 | 编译到 Go 生态（测试/包管理/并发/性能） |
| **Windows 绑定** — 无法在 Linux 服务器运行 | 🟡 高频 | Go 跨平台，Docker 部署 |
| **策略盗版** — .ex5 可被反编译 | 🟡 中频 | 策略代码不离开平台，云端执行 |

### 用户分层

| 用户层 | 规模 | 痛点强度 | 付费意愿 | 对 AlphaForge 的匹配度 |
|--------|------|----------|----------|------------------------|
| **跟单/信号跟随者** | 千万级 | **强（想赚钱但不会做）** | **中高（持续分成/订阅）** | ✅ **散户侧真正核心买方**（现被低估）|
| 散户交易者（买策略）| 千万级 | 弱（不写代码） | 低→中，**但复购差（alpha 衰减）** | ⚠️ 一次性买方，留存弱 |
| MQL 脚本开发者 | 十万级 | 中 | 低 ($30-500/EA) | ⚠️ 过度设计 |
| **专业 MQL 量化开发者** | 万级 | **强** | 中高 | ✅ **核心策略提供者**（供给侧）|
| 跨市场量化团队 | 千级 | 强 | 高 | ✅ **最佳供给侧匹配** |
| **broker（分销渠道）** | 百家级 | —（拥有用户）| **高（B2B2C）** | ✅ **最强分销杠杆**（现被排到 5.4）|
| 机构量化 | 极小 | 弱（不用 MQL） | 高但不匹配 | ❌ 不会离开 Python 生态 |

**漏掉的两个关键角色（v2 补）**：
- **跟单跟随者**：散户真正肯持续付费的不是“买一份代码”，而是“跟着有真实战绩的人自动交易”（eToro/ZuluTrade/Myfxbook 验证的最大零售资金池）。因此跟单是核心产品，不是 Phase 5 点缀。
- **broker 渠道**：broker 手里才有现成百万用户，B2B2C 可能应是主 GTM。

> **双边网络效应的前提（v2 纠正）**：双边效应要求供、需**各自独立增长**。若平台自营（层级 4）成为主供给，则实质上**不存在真双边网络效应**——它是一个产品/类基金，而非市场。这个矛盾必须在「战略取舍」里先做决策。

---

## 市场定位

策略市场是 AlphaForge 触达**散户交易者**的核心渠道——从卖工具转变为卖结果。双边市场模型：

```
策略提供者（开发者 / AI 生成）──→ 策略市场 ──→ 散户交易者
          ↑                          ↓
      获得收益分成              购买 / 订阅策略
          ↑                          ↓
   AlphaForge 收取平台抽成（按策略定价百分比）
```

**核心价值**：
- 对散户：无需编程，浏览器即可发现、购买、运行量化策略
- 对提供者：策略代码不离开平台，杜绝盗版，收入可预测
- 对平台：AI 可直接创造可售卖资产（策略），编译器从产品变成基础设施

### 差异化内核：白盒风险透明的跟单（v2 新增）

> 质疑：“若跟单是核心，市面上 eToro/ZuluTrade/Myfxbook/MQL5 Signals 已经一大堆，我们的亮点在哪？”

**一句话**：市面上的跟单全是**黑盒·镜像人的交易**；我们能做的是**白盒·跑可验证的代码**。差异不在“跟单”这个动作，而在于**我们是唯一端到端拥有策略编译代码的平台**。

**买方真正的 JTBD**：不是“跟单”，是**“赚钱 + 不要爆仓”**。现有平台只解决了前半句（接入一个交易员），却系统性在后半句翻车——你跟的是黑盒，看不到逻辑，无法判断对方是有 edge 还是在偷跑马丁/网格/无止损，直到一波行情一次性归零。**“跟单爆仓”是这个品类第一大痛点，而它源于黑盒。**

**四个结构性不可复制的差异点**（因为策略是编译成代码、跑在我们自有 VM 上）：

- **白盒风险透明（最强差异点）**：对编译后的 IR/字节码做**静态分析**，在用户“跟”之前就自动识别马丁/网格/无止损/超杠杆/加仓失控并打标或拦截。eToro 永远做不到，因为它手里没代码，只有镜像出来的成交。我们卖的是“**跟不会让你一夜爆仓的策略**”。
- **回测 = 实盘同一份代码**：跟单平台的“业绩”与任何回测脱节、无法证伪；我们被回测的那份编译产物就是实盘运行的那份，从机制上堵死回测美化（R-1）。
- **确定性服务端执行**：策略在我们引擎上跑一次、确定性生成信号，而非从某个人 MT 终端镜像到千个账户。延迟更低、无逐账户滑点分化、可复现可审计。
- **MQL 存量生态桥接（供给侧护城河）**：全球最大零售算法代码库是 MQL EA；编译器能把海量现成 EA 吃进来 → 编译 → 回测 → 风险体检 → 变成风险透明的可跟策略。eToro/ZuluTrade 吃不进 EA；MQL5 能跑 EA 但不做静态风险体检式的透明跟单。且代码不落地、不可反编译，对开发者形成拉力。

**定位收敛为一句**：不是“又一个社交跟单平台”，而是——

> **“风险透明的白盒量化跟单”** —— 跟单市场唯一能告诉你“这个策略会怎么亏、最坏亏多少、逻辑是不是赌博”的平台。

**诚实的短板**：（a）没有百万用户的流动性/品牌——靠“少量自有策略 + skin-in-the-game”做种子换早期信任；（b）代客自动交易需牌照（Phase 0.5 R-L），既是门槛也是护城河；（c）竞品可抄部分风险指标展示，但“从编译代码静态分析 + 回测实盘同源”需拥有代码与执行引擎，是架构级改造。

### 收入模型

```
层级 1: 免费策略提供者
  ├─ 可以上架策略
  ├─ 收益 100% 归自己
  └─ 目的：吸引供给，丰富市场

层级 2: 标准策略提供者
  ├─ 平台抽成 15-30%
  └─ 获得：更多曝光、数据分析、优先展示

层级 3: 专业策略提供者
  ├─ 月费/年费订阅
  └─ 获得：AI 策略生成工具、高级回测、优先支持

层级 4: AlphaForge 自有策略
  ├─ AI 自动生成 + 回测验证后的策略
  ├─ 平台直接上架，100% 归平台
  └─ 预期最大收入来源
```

### 战略取舍（v2 — 动工前必须先决策）

文档原本处于叠加态：既讲“第三方策略市场（双边网络效应）”，又把“AI 自营策略（100% 归平台）”定为最大收入来源。两条路对**信任、网络效应、合规、度量**的要求相反，不能同时最优。必须择一为主线：

| 维度 | 路线 A：第三方市场 | 路线 B：AI 自营策略产品/类基金 |
|------|-------------------|-------------------------------|
| 核心资产 | 第三方 provider 的战绩与流动性 | 平台自有 AI 策略的真实收益 |
| 网络效应 | 真双边（需供需独立增长）| 无——是产品，不是市场 |
| 信任来源 | provider 实盘战绩 + 身份验证 | 平台自有资金背书（skin-in-the-game）|
| 合规重心 | 撮合/信息平台（较轻，但跟单可能触发牌照）| 资产管理/投顾（重，几乎必然需牌照）|
| 供给侧关键 | **提现必须极早**（否则无人供给）| 提现次要（平台自持）|
| 北极星 | 活跃 provider 数 + GMV | 盈利用户数 + 自有策略净收益 |

**建议**：先做**路线 A 为主 + 少量平台自有策略做冷启动种子**，但**明确放弃“AI 占比≥70%”这类自营指标**，否则会把平台推成一个“无战绩的 EA 农场”。此决策直接决定下面 Phase 排序。

### 关键风险与应对（v2 重排 + 补齐）

> 原风险表**排序错、且不完整**。已按“能否一枪毙命”重排，并补 3 个被漏掉的风险。

| ID | 风险 | 严重度 | 应对 | 兜底是否充分 |
|----|------|--------|------|-------------|
| R-L | **合规/牌照**：销售/分成他人代交易策略、尤其跟单，多辖区构成受监管的投顾/资产管理，**免责声明治不了无牌经营** | 🔴🔴 致命 | **Phase 0.5 前置法律评估**：目标辖区牌照映射、是否限定“信息展示不代客操作”、跟单是否单独持牌 | ❌ 原 1.4 仅“底部风险提示组件”，严重不成比例 |
| R-P | **平台依赖 MetaQuotes**：MQL 语言/生态/授权归其单方控制，可改语言、改授权、或自强化 MT5 云测试直接绕过我方 | 🔴 | 降低语言耦合（IR 层可接入非 MQL 前端）、监测 MT 政策、储备迁移路径 | ⚠️ 原文档未列 |
| R-A | **Alpha 衰减**：策略卖爆即失效，是产品固有属性，直接侵蚀续费率/退款率/口碑 | 🔴 | 策略容量上限、失效自动退役/降权、按实盘滚动业绩动态展示、诚实披露衰减 | ⚠️ 原文档未列 |
| R-1 | 回测美化（实盘远差于回测） | 🔴 | 实盘跟踪（1.1）+ **样本外/walk-forward/purged CV**（升级 1.2，而非阈值门） | ⚠️ 原 1.2 阈值门=为过拟合做选择，见 1.2 修订 |
| R-C | 冷启动（无买方↔无卖方） | 🟡 | **少量平台自有策略做种子**（非批量灌库）+ 先补供给侧提现闭环吸引真 provider | ⚠️ 原“AI 批量灌 200+”与信任命题冲突 |
| R-T | 信任鸿沟 | 🟡 | 实盘业绩公开 + 提供者验证 + **skin-in-the-game（自有资金在跑）** | ⚠️ 原仅展示型特性，缺利益对齐 |

---

## Phase 0：已完成（后端核心 + 基础前端）

### 后端 — `backend/internal/marketplace/`（9 文件）

| 功能 | 文件 | 关键实现 |
|------|------|----------|
| 策略上架/下架 | `publish.go` | 原子双表写入（`user_strategy_publishes` + `marketplace_strategies`），所有权校验 |
| 策略列表+搜索 | `publish.go` | pg_trgm 模糊搜索，支持 newest/popular/performance 排序，60s 缓存 |
| 三种定价模型 | `types.go` | `free` / `once` / `subscription`，平台费率从 `system_config` 读取 |
| 付费购买 | `purchase.go` | 原子事务（FOR UPDATE 锁钱包 → 扣款 → 分账 → 创建订阅），幂等键防重 |
| 免费订阅 | `service_subscription.go` | `Subscribe()` 含价格门禁，付费策略必须走 `PurchaseStrategy` |
| 订阅续费 | `service_subscription.go` | 每日午夜批量续费，余额不足自动停用，发布者+平台分账 |
| 退款 | `refund.go` | 全额退款（买家退款 + 发布者扣回 + 平台费处理），发布者余额不足时优雅降级 |
| 评分/评论 | `interactions.go` | UPSERT 评分（1-5），分页评论+用户名关联 |
| 跟单引擎 | `copytrade.go` | ProRata 按权益分仓，信号去重（`copytrade_signals`），并发控制（semaphore 8），风控管线集成 |
| 市场回测 | `backtest.go` | 代码服务端保护（`CanAccessCode`），回测结果流式推送 |
| 发布者统计 | `publish.go` | `GetPublisherStats`：上架数/订阅数/总收入/月收入/平均评分/TOP 策略 |

### 后端 — `backend/internal/connect/marketplace/`（8 文件）

| 功能 | 文件 | 关键实现 |
|------|------|----------|
| 14 个 RPC 端点 | `marketplace_handler.go` + `_subs.go` + `_social.go` | PublishStrategy / Subscribe / Unsubscribe / PurchaseStrategy / ListPublished / ListSubscriptions / RateStrategy / ListRatings / CommentOnStrategy / ListComments / SetStrategyPricing / UnpublishStrategy / GetPublisherStats / RunMarketBacktest |
| 回测 SSE 流 | `marketplace_stream.go` | PG NOTIFY on `backtest_status_change`，30s 轮询兜底，终端状态检测 |
| 认证+鉴权 | 所有 handler | 全部需要认证；SetStrategyPricing/UnpublishStrategy 有 admin guard |

### 数据库迁移（20 个 migration）

核心表：`marketplace_strategies`、`user_strategy_publishes`、`user_subscriptions`、`marketplace_ratings`、`marketplace_comments`、`copytrade_signals`、`subscription_plans`、`user_platform_subscriptions`

### 前端 — `frontend/src/pages/marketplace/`

| 组件 | 功能 |
|------|------|
| `MarketplacePage` | 3 Tab 布局：Market / Purchases / Author |
| `MarketTab` | 搜索+筛选+排序+分页卡片网格 |
| `StrategyMarketCard` | 策略卡片（名称/价格/评分/KPI/标签/购买状态） |
| `StrategyDetailModal` | 策略详情（指标+评分+评论+操作按钮） |
| `PaymentModal` | 支付确认（钱包余额检查+不足引导充值） |
| `ProtectedBacktestPanel` | 回测表单+服务端流式结果+净值曲线图 |
| `PurchaseTab` | 已购策略列表 |
| `AuthorTab` | 发布者统计卡片+已发布策略表 |
| `PublishToMarketModal` | 上架表单（策略库入口） |
| i18n | 5 语言（en/zh-cn/zh-tw/vi/ja） |

### 已知缺陷

| # | 问题 | 影响 |
|---|------|------|
| BUG-1 | 前端 `PublishToMarketModal` 价格模型选项为 `free/once/monthly`，后端只认 `free/once/subscription`，`monthly` 会导致付费购买逻辑失效 | 🔴 阻塞付费订阅 |
| BUG-2 | `copy_trade_links` 表（migration 110）未被任何代码引用，CopyTradeEngine 直接查 `user_subscriptions` | 🟡 冗余 schema |
| GAP-1 | 退款流程无前端 UI | 🟡 用户体验缺失 |
| GAP-2 | Admin 策略管理无前端 UI（SetStrategyPricing/Unpublish 只有 RPC） | 🟡 运营能力缺失 |
| GAP-3 | 平台订阅（Free/Pro/Enterprise）与策略市场无实际联动，所有 tier 标记 `marketplace: true` 但无差异化控制 | 🟡 商业化不完整 |

---

## Phase 0.5：合规/牌照前置门 🔴🔴 P-1（生死级，早于一切）

> **目标**：在灌任何 AI 策略、开任何跟单、收任何分成之前，先确认我们**在法律上能不能这么做**。这不是前端任务，是存亡线（风险 R-L）。

**Tasks**:
- [ ] 目标辖区（先定 1-2 个主战场）牌照映射：销售策略 / 分成 / 跟单代客操作 分别落在哪类监管（信息服务 vs 投顾 vs 资产管理）
- [ ] 产品边界法律定性：坚守“信息展示 + 用户自主执行”还是要做“代客自动交易（跟单）”——后者大概率需单独牌照
- [ ] 免责/风险提示文案由法务确认（不是抄 MQL5/TradingView 了事）
- [ ] KYC/AML 与提现合规评估（涉及资金流转，联动 ADR-0026 HD 钱包）
- [ ] 输出《合规红线清单》：哪些功能在拿牌前**禁止上线**（很可能包括跟单代客）

**Gate**：合规红线清单未产出前，Phase 2（AI 供给规模化）与跟单 UI **不得启动**。

---

## Phase 1：信任基础设施 + 散户价值闭环 🔴 P0

> **目标**：让散户看到策略后愿意掏钱。核心解决"这策略真的赚钱吗？"

### 1.1 实盘业绩跟踪

**Why**: 回测可以美化，实盘不能造假。这是信任的基石。

**Tasks**:
- [ ] 新建 `marketplace_live_performance` 表（strategy_id, date, daily_pnl, daily_return, equity, drawdown, total_trades, winning_trades, created_at）
- [ ] 提供者绑定实盘 MT 账户后，平台自动采集日度交易数据写入 performance 表
- [ ] 新增 RPC `GetLivePerformance(strategy_id) → {daily_performance[], summary_stats}`
- [ ] 策略详情页前端展示实盘净值曲线 + 月度收益热力图
- [ ] 区分"回测业绩"和"实盘业绩"两个 Tab

**Files**:
- `backend/migrations/xxx_marketplace_live_performance.up.sql` (new)
- `backend/internal/repository/marketplace_performance_repo.go` (new)
- `backend/internal/connect/marketplace/marketplace_handler_performance.go` (new)
- `proto/ant/v1/marketplace_service.proto` (extend)
- `frontend/src/pages/marketplace/components/LivePerformancePanel.tsx` (new)

### 1.2 回测质量门槛（v2 升级：抗过拟合）

**Why**: 防止低质量策略泛滥，损害市场信誉。

> ⚠️ **不要用纯阈值门**：单靠 Sharpe/回撤/胜率阈值筛选，恰恰是过拟合策略最会刷的指标——阈值门=**为幸存者偏差/过拟合做选择**，反而放大 R-1。必须叠加样本外验证。

**Tasks**:
- [ ] 在 `Publish` 方法中新增 `validateBacktestQuality` 步骤
- [ ] **样本外/walk-forward 验证**：持仓期切分 IS/OOS，OOS 表现显著劣化则拒绝；多段 rolling window
- [ ] **Purged/embargoed CV**（防未来函数/泄露）；记录 IS vs OOS 指标差作为过拟合评分
- [ ] 门槛规则可配置（`system_config` 表）：`min_sharpe_ratio`、`max_drawdown_pct`、`min_total_trades`、`min_win_rate`、**`max_is_oos_degradation`**
- [ ] 不达标返回明确错误码 + 中文提示（哪些指标不达标及阈值）
- [ ] Admin 可针对特定提供者/策略豁免门槛（`marketplace_quality_waivers` 表）

**Files**:
- `backend/internal/marketplace/publish.go` (extend)
- `backend/migrations/xxx_marketplace_quality_waivers.up.sql` (new)

### 1.3 提供者身份验证

**Why**: 让用户知道策略背后是人还是 AI，是否有真实身份验证。

**Tasks**:
- [ ] `users` 表新增 `verified_provider` boolean，默认 false
- [ ] 提供者提交验证材料（Admin 审核流程）
- [ ] `PublishedStrategy` proto 新增 `provider_verified` 和 `provider_type`（human/ai/hybrid）字段
- [ ] 前端策略卡片 + 详情页显示提供者认证徽章 + 类型标签

### 1.4 风险声明与合规（仅 UI 层；结构性合规已上移至 Phase 0.5）

**Why**: 避免法律风险，建立专业形象。

> **v2 注**：本节仅处理前端展示层。**「能不能卖策略/做跟单」这个生死问题在 Phase 0.5（R-L）已前置**；免责声明不能代替牌照。

**Tasks**:
- [ ] 策略详情页底部强制风险提示组件（不可折叠）
- [ ] 首次购买弹窗二次确认：显示风险声明 + 勾选"我已知晓风险"
- [ ] 策略页底部显示策略免责：过往业绩不代表未来表现
- [ ] 参考 MQL5 Market / TradingView 合规文案

### 1.5 价格模型修复 [BUG-1]

**Why**: 当前前端 `monthly` 与后端 `subscription` 不匹配，付费订阅功能实际不可用。

**Tasks**:
- [ ] 前端 `PublishToMarketModal` 将 `monthly` 改为 `subscription`
- [ ] 后端 `PurchaseStrategy` 增加输入校验，拒绝不识别的 price_model
- [ ] 回归测试覆盖 free/once/subscription 三种发布+购买流程

---

## Phase 2：AI 策略供给 🔴 P0

> **目标**：用 AI 批量生产高质量策略，解决冷启动。串联现有 agent-engine → mql-compiler → backtest-engine 管线。

### 2.1 AI 一键生成 + 自动上架

**Why**: 这是平台最大的差异化能力。用户可以零代码获得可售策略。

**Tasks**:
- [ ] 新建 `AutoGenerateStrategy` RPC：接收自然语言需求描述 → 调用 agent-engine 生成策略 → mql-compiler 编译 → backtest-engine 回测 → 达标自动 `Publish`
- [ ] 全流程 SSE 推送进度（生成中 → 编译中 → 回测中 → 评估中 → 已上架/不达标）
- [ ] 前端新建 `AutoGeneratePanel` 组件：需求输入 + 进度条 + 结果预览 + 一键发布
- [ ] 用户可在发布前编辑策略标题、描述、定价
- [ ] 失败时返回具体原因（编译失败/回测不达标/超时）

**Files**:
- `backend/internal/connect/marketplace/marketplace_handler_autogen.go` (new)
- `proto/ant/v1/marketplace_service.proto` (extend)
- `frontend/src/pages/marketplace/components/AutoGeneratePanel.tsx` (new)

### 2.2 批量策略生成队列

**Why**: 平台侧主动扩充策略库，覆盖主流品种×周期组合。

**Tasks**:
- [ ] 后台异步任务：扫描品种（EURUSD, BTCUSD, XAUUSD...）× 周期（M15, H1, H4, D1）× 策略类型（趋势/均值回归/突破）
- [ ] 每个组合生成 3-5 个策略变体，写入 `auto_generated_strategies` 表（待审核状态）
- [ ] Admin 审核面板：预览回测结果 → 批量批准/拒绝 → 批准后自动 Publish
- [ ] 每日限额控制，避免 API 费用失控

**Files**:
- `backend/internal/marketplace/batch_generator.go` (new)
- `backend/migrations/xxx_auto_generated_strategies.up.sql` (new)

### 2.3 策略参数模板

**Why**: 提供"填空式"策略创建，降低策略构思门槛。

**Tasks**:
- [ ] 预置策略模板（trend_following / mean_reversion / breakout / arbitrage），每个模板定义可配置参数
- [ ] `ListStrategyTemplates` RPC：返回可用模板列表 + 参数说明
- [ ] 前端模板选择器：卡片式模板浏览 → 选品种+周期 → AI 填充参数 → 生成回测 → 预览 → 上架

### 2.4 提供者工具面板增强

**Why**: 当前 `AuthorTab` 只有基础统计，需要完整的提供者工作站。

**Tasks**:
- [ ] 收益趋势图（日/周/月维度，销售收入 + 订阅收入分开展示）
- [ ] 订阅者分析（总数、新增、流失、续费率）
- [ ] 单策略分析（每个策略的订阅数趋势、评分分布、收入贡献）
- [ ] 提现入口（跳转钱包提现页）

---

## Phase 3：增长引擎 🟡 P1

> **目标**：构建双边网络效应 — 买方越多 → 提供者越多 → 策略越多 → 买方更多。

### 3.1 策略排行榜

**Why**: 给用户一个"从哪里开始"的入口，给提供者一个竞争激励。

**Tasks**:
- [ ] 新增 RPC `ListLeaderboard(type, period, limit)`：type ∈ {return, popular, new, copytrade}，period ∈ {week, month, quarter, all}
- [ ] 收益榜按实盘业绩排序（无实盘的策略不参与）
- [ ] 人气榜按订阅数排序
- [ ] 新锐榜按最近 30 天上架的业绩排序
- [ ] 前端独立 Leaderboard 页面，Tab 切换榜单类型
- [ ] 策略卡片在榜单中显示排名徽章

### 3.2 免费试用

**Why**: 降低购买决策门槛，付费转化率的直接杠杆。

**Tasks**:
- [ ] 新建 `marketplace_trials` 表（id, user_id, strategy_id, started_at, expires_at, status）
- [ ] `StartTrial` RPC：7 天免费试用，一个用户同一策略只能试用一次
- [ ] 后台 `CheckTrialExpiry` 定时任务：到期自动取消试用，恢复代码访问限制
- [ ] `CanAccessCode` 扩展：试用期内可访问代码
- [ ] 前端策略详情页显示"免费试用"按钮（未试用过 + 策略支持试用）
- [ ] 试用到期前 24h 邮件/站内通知

### 3.3 策略对比工具

**Why**: 帮助用户在多个策略之间做出理性选择。

**Tasks**:
- [ ] `CompareStrategies(strategy_ids[])` RPC：批量返回标准化对比数据
- [ ] 前端对比组件：并排表格（回测指标、实盘业绩、费率、风险指标），高亮最优值
- [ ] 添加到对比的快捷操作（策略卡片上的复选框）

### 3.4 通知系统

**Why**: 激活沉默用户，提升留存。

**Tasks**:
- [ ] 新建 `marketplace_notifications` 表（id, user_id, type, title, body, strategy_id, is_read, created_at）
- [ ] SSE 推送通知到前端导航栏 Bell 图标
- [ ] 触发场景：新策略上架（关注品种）、价格变动、订阅即将到期、策略业绩大幅异动、收到新评分/评论
- [ ] 通知偏好设置（用户可选择接收哪些类型的通知）

### 3.5 社交分享

**Why**: 免费获客渠道，利用用户社交网络传播。

**Tasks**:
- [ ] 策略详情页生成 SEO 友好的 OpenGraph 标签（标题、描述、回测摘要缩略图）
- [ ] 分享按钮（复制链接 / Twitter / Telegram / 微信）
- [ ] 分享落地页（非登录用户可浏览策略基本信息 + 注册 CTA）

---

## Phase 4：平台运营 🟢 P2

> **目标**：平台方具备完整的运营能力，商业化可管理。

### 4.1 Admin 策略管理面板

**Why**: 运营人员需要一个统一的后台来管理市场内容。

**Tasks**:
- [ ] 策略审核列表（待审核/已发布/已隐藏/违规下架），支持批量操作
- [ ] 策略详情查看（完整元数据 + 回测结果 + 销售数据）
- [ ] 推荐/置顶策略（`marketplace_strategies` 加 `is_featured`、`featured_until` 字段）
- [ ] Admin 修改定价 + 平台费率（已有 RPC，补前端 UI）
- [ ] 违规策略下架 + 下架原因记录 + 通知提供者

### 4.2 退款 UI [GAP-1]

**Why**: 后端退款逻辑已完成，缺少用户入口。

**Tasks**:
- [ ] 前端已购策略列表增加"申请退款"按钮（条件：购买后 < 7 天）
- [ ] 退款申请表单（原因选择 + 补充说明）
- [ ] Admin 退款审核面板（待处理/已批准/已拒绝）
- [ ] 退款处理结果通知用户

### 4.3 收入仪表盘（Admin）

**Why**: 平台方需要实时了解市场运行状态。

**Tasks**:
- [ ] `GetMarketplaceAnalytics` RPC：GMV、平台收入、付费用户数、ARPU、退款率、续费率
- [ ] 按时间维度（日/周/月/自定义）聚合
- [ ] TOP 策略（按收入）、TOP 提供者（按收入）、策略数量趋势
- [ ] 前端 Admin Dashboard 新增"策略市场"模块

### 4.4 策略版本管理

**Why**: 策略会迭代优化，已购用户需要知道版本变化。

**Tasks**:
- [ ] `marketplace_strategies` 新增 `version` 字段（SemVer，默认 1.0.0）
- [ ] 提供者更新策略时创建新版本记录（`marketplace_strategy_versions` 表）
- [ ] 已购用户可在策略详情页查看版本历史 + changelog
- [ ] 已购用户可选择升级到新版本（免费升级 or 补差价，视定价策略而定）
- [ ] 前端展示"v1.2.0 · 3 天前更新"

### 4.5 折扣/优惠券

**Why**: 促销活动是拉动 GMV 的标准手段。

**Tasks**:
- [ ] 新建 `marketplace_coupons` 表（code, discount_type[percentage/fixed], discount_value, min_purchase, max_uses, used_count, expires_at, applicable_strategies[]）
- [ ] `ValidateCoupon` RPC：校验优惠券有效性 + 计算折后价格
- [ ] `PurchaseStrategy` 扩展：接受 coupon_code 参数，应用折扣后扣款
- [ ] Admin 优惠券管理面板（创建/启用/停用/查看使用统计）
- [ ] 前端支付弹窗增加优惠券输入框

### 4.6 策略提现 ⬆️（v2 已上移至 Phase 1）

**Why**: 提供者赚到的钱需要能提出来。

> **v2 重排**：若走路线 A（第三方市场），**提现是供给侧命脉，必须在 Phase 1就绪**——买方能进钱而卖方不能提钱是招商死结。任务依旧，优先级提到 P0。

**Tasks**:
- [ ] 复用 HD 钱包系统（ADR-0026）的冷签名提现流程
- [ ] 提供者在 Author Dashboard 发起提现（输入金额 + 目标地址）
- [ ] 平台审核 → 冷签名 → 广播交易
- [ ] 提现记录 + 状态追踪

---

## Phase 5：护城河 🟢 P3

> **目标**：建立长期竞争壁垒，拓展收入来源。

### 5.1 策略捆绑包

**Why**: 提升客单价，促进交叉销售。

**Tasks**:
- [ ] 新建 `marketplace_bundles` 表（id, title, description, strategy_ids[], bundle_price, original_total, discount_pct）
- [ ] `ListBundles` / `PurchaseBundle` RPC
- [ ] 前端捆绑包卡片（显示包含策略数 + 原价 vs 捆绑价 + 节省金额）

### 5.2 跟单 UI [引擎已有] ⬆️（v2 已上移至 Phase 1）

**Why**: CopyTradeEngine 已完成，缺少让用户使用的 UI。

> **v2 重排**：跟单是**散户侧真正核心产品**（跟有战绩的人自动交易），不是护城河点缀，已上移至 Phase 1。⚠️ **受 Phase 0.5 制约**：跟单=代客自动交易，很可能需单独牌照，拿牌前不得上线（R-L）。
>
> **差异化内核（见「市场定位 · 白盒风险透明的跟单」）**：不是再做一个 eToro，而是**风险透明的白盒跟单**。本阶段 UI 除了基础跟单配置，必须展示从编译代码静态分析得出的**风险画像**（是否含马丁/网格/无止损、最大理论亏损、杠杆上限），让用户在“跟”之前就看得到风险。

**Tasks**:
- [ ] 策略详情页新增"跟单"按钮
- [ ] 跟单配置弹窗：选择跟单账户 → 设置跟单比例（10%-100%） → 最大仓位限制 → 止损设置
- [ ] `StartCopyTrade` / `StopCopyTrade` RPC
- [ ] 已购策略列表显示跟单状态（跟单中/已暂停）+ 跟单收益

### 5.3 阶梯费率

**Why**: 激励高质量提供者，提升平台收入。

**Tasks**:
- [ ] 提供者等级制度（Bronze/Silver/Gold/Platinum），按月收入/评分/订阅数自动升级
- [ ] 不同等级对应不同 `platform_fee_rate`（如 Bronze 25% → Gold 10%）
- [ ] 新提供者默认 Bronze，每月 1 号重新计算等级
- [ ] 前端 Author Dashboard 显示当前等级 + 升级条件进度条

### 5.4 白标 / API

**Why**: B2B 收入渠道，让 broker 集成策略市场。

**Tasks**:
- [ ] 策略市场公开 API（RESTful 风格，虽然项目禁止 REST 但这是外部集成场景，需评估）
- [ ] Broker 可通过 API 获取策略列表 + 嵌入到自己平台的 iframe
- [ ] 自定义品牌（Logo、主题色）
- [ ] 收入分账（Broker 引入的用户购买策略，Broker 获得分成）

---

## 依赖关系

```
Phase 0.5 (合规/牌照) ← 生死级前置，产出《合规红线清单》后才解锁 Phase 2/跟单
        │
Phase 1 (信任 + 散户价值闭环)
  ├── 1.5 价格模型修复 [BUG-1] ← 立即修，阻塞付费闭环（其实是 bug，非 phase）
  ├── 1.1 实盘跟踪 ← 散户价值闭环第 1 环；注意：真实战绩需日历时间，无法用开发压缩
  ├── 1.2 回测门槛（升级）← 用样本外/walk-forward/purged CV，而非阈值门（否则加剧 R-1）
  ├── ★ 跟单 UI（原 5.2 上移）← 引擎已完成，是散户核心产品；受 Phase 0.5 合规门约束
  └── ★ 提供者提现（原 4.6 上移）← 供给侧命脉，拿不到钱就没人供给
        │
Phase 2 (供给：种子优先，非批量灌库) ← 依赖 Phase 0.5 + 1.2
  ├── 2.1 AI 生成 ← 串联 agent-engine + mql-compiler + backtest-engine
  ├── ▽ 2.2 批量生成队列（降级）← 仅做“少量平台自有种子”，放弃 AI 占比 KPI
  └── 2.3 参数模板 ← 依赖 2.1 的管线
        │
Phase 3 (增长) ← 依赖 Phase 1+2 有足够可信策略和用户
  ├── 3.2 试用 ← 依赖 Phase 1.1 (实盘跟踪作为试用期对比)
  └── 3.1 排行榜 ← 依赖 Phase 1.1 (实盘数据排序，只收实盘策略)
        │
Phase 4 (运营) ← 可与 Phase 3 并行（4.6 提现已上移到 Phase 1）
  └── 4.1 Admin ← 可用时即做，不阻塞其他 Phase
        │
Phase 5 (护城河：broker 渠道/白标) ← 若走 B2B2C，5.4 应评估上移为主 GTM
```

**v2 再排序摘要**：
- **上移到 Phase 1**：跟单 UI（原 5.2）、提供者提现（原 4.6）——散户价值闭环 + 供给侧命脉。
- **新增前置**：Phase 0.5 合规门（早于 Phase 2/跟单）。
- **降级**：2.2 批量灌库 → 仅平台自有种子；剔除 AI 占比目标。
- **待评估上移**：5.4 broker 白标/API，若确认 B2B2C 是主渠道。

---

## 关键指标

> **v2 北极星纠正**：原表把 `AI 策略占比≥70%` 当目标——那是在优化**供给成分**而非**买方结果**，且把 Phase1/Phase2 的矛盾固化进度量。北极星应是**盈利用户数/留存**。`AI 占比` 降为观测量，不设目标。

| 指标 | 现状 | Phase 1 目标 | Phase 2 目标 | Phase 3 目标 |
|------|------|-------------|-------------|-------------|
| **★ 盈利买方占比（实盘正收益）** | — | 建立度量 | ≥30% | ≥40% |
| **★ 90 天买方留存** | — | 建立度量 | ≥40% | ≥50% |
| **★ 有实盘战绩的策略数** | ~0 | 10+（真实盘）| 30+ | 100+ |
| 上架策略数 | ~10 | 30+ (含回测验证) | 80+（重质不重量）| 200+ |
| 月活跃买方 | <50 | 200 | 1000 | 5000+ |
| 月 GMV | — | $2K | $20K | $100K+ |
| 平台月收入 | — | $500 | $5K | $25K+ |
| 策略平均评分 | — | ≥4.0 | ≥4.2 | ≥4.3 |
| 付费转化率 | — | 2%（保守，alpha 衰减）| 3% | 5%+ |
| 订阅续费率 | — | — | ≥50%（交易策略难做高）| ≥60% |
| 退款率（新增监控）| — | <15% | <12% | <10% |
| AI 策略占比（观测量，非目标）| 0% | — | — | — |

---

## 复用清单（Reuse Preflight）

实现每个 Phase 前必须验证以下现有能力可复用，避免重复造轮子：

| 能力 | 位置 | 被 Phase 使用 |
|------|------|-------------|
| 策略编译管线 | `mql-compiler` → IR → Bytecode VM | Phase 2 (AI 生成) |
| AI 策略生成 | `agent-engine` | Phase 2 (AI 生成) |
| 回测引擎 | `backtest-engine` / SimBroker | Phase 1.2 (门槛), Phase 2 (验证) |
| 钱包交易 | `walletRepo.AdjustBalanceTx` (hash chain) | Phase 4.5 (优惠券), Phase 4.6 (提现) |
| HD 钱包提现 | ADR-0026 冷签名流程 | Phase 4.6 (提现) |
| SSE 推送管道 | ConnectRPC server-stream + PG NOTIFY | Phase 1.1 (实盘推送), Phase 3.4 (通知) |
| pg_trgm 搜索 | migration 172 (GIN index) | Phase 1 (搜索增强) |
| 风控管线 | `risk-gate` 6 门管线 + OMS | Phase 5.2 (跟单 UI) |
| i18n 框架 | 5 语言已部署 | 所有前端新页面 |
| 平台订阅 | Free/Pro/Enterprise tiers | GAP-3 联动修复 |
