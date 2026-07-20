# Strategy Marketplace — 设计文档

> **定位**：📐 架构总纲 — 本文档是策略市场的权威设计来源。市场定位、数据模型、API 面、Phase 规划、依赖关系、指标目标均在此定义。
> **施工文档**：`docs/plan/marketplace/` — GLM 按 Phase 逐模块执行，每个任务有精确文件路径+代码+验收标准。
>
> **最后更新**：2026-07-20 (v3 定稿)
> **状态**：Phase 0 完成；v3 明确产品边界、商业模式与获客策略
>
> **v3 定稿要点**（覆盖 v2）：
> 1. **产品边界**：策略市场，对标 MQL5 Market，不做跟单平台、不代客交易、不碰资金。
> 2. **商业模式**：纯第三方市场（路线 A），平台不自营策略、不参与客户竞争。收入 = 平台订阅 + 策略抽成（非交易佣金，不和 broker 绑定）。
> 3. **核心粘性**：策略代码不离开平台 → 用户买一次可跑多 MT 账户 → 天然防流失（离开平台就失去多账户能力）。
> 4. **合规策略**：不持牌、不管理资金。交易对手=用户自己的 broker。有问题辖区直接封 IP。
> 5. **获客策略**：初期免费，先做供给和买方规模。
> 6. **技术护城河**：编译器是供给侧工具；市场真壁垒 = 流动性 + 代码不出平台的反盗版/多账户锁。
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

**AlphaForge 的独特位置**：唯一一家做 **MQL → tree-sitter → IR → Bytecode → Go VM 编译器** 的平台。竞品全部选择”生成 MQL 源码 → 在 MetaEditor 编译”的轻路径。

> **护城河定义（v3）**：编译器的真价值不在于它是技术壁垒——买方（散户）对编译器零感知，他们只看收益。编译器的价值在于**让策略代码不出平台成为可能**。这带来两个商业结果：
> 1. **防盗版**：提供者的策略不会被下载、反编译、传播——这是 MQL5 Market 做不到的。
> 2. **多账户锁**：用户买一个策略可以在平台内跑多个 MT 账户。一旦离开平台，就失去这个能力——这是天然的用户粘性，不是靠营销，是靠架构。
>
> **真护城河 = 流动性（买卖双方规模）+ 代码不出平台的锁定效应。** 编译器是供给侧获客工具，不是锁需求的壁垒。

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
| 散户交易者（买策略） | 千万级 | 中（不会编程但想用策略赚钱） | 中（一次性购买+多账户跑=持续留存） | ✅ **核心买方** |
| 散户（多账户使用者） | 百万级 | — | 中（买一次→多账户=粘性来源） | ✅ **留存引擎** |
| MQL 脚本开发者 | 十万级 | 中 | 低 ($30-500/EA) | ⚠️ 过度设计 |
| **专业 MQL 量化开发者** | 万级 | **强** | 中高 | ✅ **核心策略提供者**（供给侧）|
| 跨市场量化团队 | 千级 | 强 | 高 | ✅ **最佳供给侧匹配** |
| broker（兼容对象） | 不限 | — | — | 兼容所有 MT broker，不做绑定合作 |
| 机构量化 | 极小 | 弱（不用 MQL） | 高但不匹配 | ❌ 不会离开 Python 生态 |

> **用户粘性模型（v3）**：核心留存机制不是”跟单”，而是**买一次策略→在平台内跑多个 MT 账户**。MQL5 Market 卖的是 .ex5 文件，下载后用户离开平台也能用——没有留存。我们策略代码不出平台，用户想同时跑 5 个账户就必须反复登录我们的平台。这是利用人性弱点（买一份跑多份=占便宜心理）的架构级粘性，不需要跟单功能。

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

### 产品边界（v3 明确）

**我们做什么**：策略编写工具 + 策略市场 + 平台内策略执行。用户写策略/买策略 → 在平台内跑 → 通过用户自己的 MT broker 账户交易。

**我们不做**：
- ❌ 不代客交易、不管理资金、不做跟单服务
- ❌ 不拿金融牌照、不做投顾
- ❌ 不和 broker 绑定合作
- ❌ 平台不自营策略、不参与客户竞争

**对标**：MQL5 Market，但策略不离开平台。MQL5 Market 卖 .ex5 文件让用户下载→我们卖的是平台内策略使用权。

### 平台锁定的商业逻辑（v3 新增）

```
MQL5 Market 模式：
  用户买 EA → 下载 .ex5 → 本地 MT 跑 → 可复制/传播/盗版 → 用户离开平台

AlphaForge 模式：
  用户买策略 → 代码不出平台 → 云端 VM 执行 → 绑定多 MT 账户
                                            ↓
  买一个策略 → 平台内加账户槽（计费）→ 离开平台 = 失去低成本多账户编排
                                            ↓
  性价比 + 订阅制 = 粘性来源（平台内比线下 DIY 更省钱省心）
```

这意味着我们的留存不是靠”更好的 UI”或”品牌忠诚”，是靠**架构级的切换成本**——用户在我们的 VM 里跑策略，账户绑定/风控配置/实盘历史都在我们这里。离开的成本不是”再找一个平台”，而是”重建整套账户编排 + 历史归零”。

#### 多账户 = 计费轴，不是免费漏洞（v4 修正）

> 早期把“买一个跑多个账户”当免费卖点，会**直接漏掉提供者收入**——一个资金管理者买一份策略挂 50 个账户，提供者只收 1 份的钱。MQL5 按账户号锁激活正是提供者要求的。**双边市场不能靠抢卖方来讨好买方。**

**修正**：多账户是**计费维度**，不是白送——
- 买一份策略 = 默认 1 个执行账户槽；每多绑一个账户 = 平台订阅升级（Pro 解锁 N 槽 / Enterprise 更多）或提供者“多账户附加费”。
- 代码在我们 VM、账户绑定在我们手里 → **我们天然能精确计量“这份策略在跑几个账户”**（MQL5 做不到、我们能做到，别浪费在白送上）。
- 效果一举三得：买方仍有多账户便利（粘性还在）+ 每多一个账户平台/提供者都赚钱（供给侧不流血）+ 贪便宜变成收入而非漏损。

#### 线下跟单：技术上拦不住，靠性价比 + 订阅制化解（v4 诚实声明）

> **残酷事实**：策略把真实订单打到用户自己的账户 A，用户开多个 MT 客户端 + 任意 copier 把 A 的成交镜像到 B/C/D，**全程在平台外，我们代码不在 B/C/D 上，无从设卡**。这也是 MQL5 靠“EA 内校验账户号”锁激活的原因——它代码跑在每个账户上，我们跑在自有 VM 上，管不到线下账户。**任何依赖“阻止线下镜像”成立的方案都会失败；要设计成“不需要阻止它”。**

三层化解：
- **让平台内多账户比线下 DIY 更划算**：线下镜像要自建 VPS 常年跑多终端 + 买 copier + 承受镜像延迟滑点（B/C/D 成交更差）。把“平台内加账户槽”的价格定在**低于线下总成本 + 麻烦**，贪便宜的人自然选平台。**用性价比赢，不用技术锁赢。**
- **策略定价优先订阅制（关键）**：线下镜像者**必须让账户 A 常年在平台上跑**才能持续产信号——A 一停，B/C/D 全停。所以他必须持续付账户 A 的策略订阅 + 平台订阅。**一次性买断是重灾区（付一次镜像一辈子）；改订阅后，漏损从“N× 变 1× 永久”缩成“N× 变 1× 且每月续费”，还能把重度用户升到 Enterprise。**
- **对残余漏损保护提供者**：抓不到线下账户，但能保证“跑得越多、账户 A 这条线交的钱越多”；高档订阅让提供者拿更高分成，安抚被薅感。

### 竞争壁垒总结

| 壁垒 | 类型 | 可持续性 |
|------|------|---------|
| MQL→Go 编译器 | 技术壁垒（供给侧获客） | ⚠️ 可被复制，且依赖 MetaQuotes |
| 代码不出平台（防盗版） | 架构壁垒（供给侧留存） | ✅ 竞品需重做执行引擎 |
| 多账户执行（用户粘性） | 架构壁垒（需求侧留存） | ⚠️ **依赖订阅制 + 供给持续更新**；单策略锁会随 alpha 衰减失效，且线下镜像技术上拦不住 |
| 账户编排 + 实盘历史 + 风控配置 | 迁移成本壁垒（需求侧留存） | ✅ 离开=整套重建 + 历史归零，与单策略是否赚钱无关，才是持久锁 |
| 策略市场流动性 | 网络效应壁垒 | ✅ 买卖双边规模越大越难迁移 |
| 实盘战绩数据 | 数据壁垒 | ✅ 历史数据积累不可复制 |

> **锁定的真相（v4）**：别指望“锁住一个策略”——策略会 alpha 衰减（R-A），一旦不赚钱，被锁的东西就变废纸，用户照样走。**真护城河 = ①这里持续产出还能赚钱的新策略（供给侧更新速度，AI 辅助加速）+ ②用户自己攒下的账户编排/历史/配置（离开要整套重建）。** 前者随衰减失效，后者才持久。

### 收入模型（v3）

**两条收入线：平台订阅 + 策略抽成。**（注：不是“交易佣金”——我们不和 broker 绑定，拿不到成交返佣；抽成指策略销售时对提供者的分成。）

```
平台订阅（平台使用权 + 账户槽）:
  ├─ 初期: 免费获客
  ├─ 成熟期: Free / Pro / Enterprise 订阅制
  ├─ Pro+ 解锁高级功能（AI 生成额度、高级回测等）
  └─ ★ 账户槽 = 计费轴: 每档解锁不同数量的执行账户绑定数（多账户不白送）

策略抽成（策略销售收入分成）:
  ├─ 策略定价权: 提供者自行设定（free / once / subscription）
  ├─ 平台默认引导 subscription，但提供者自由选择，不做强制
  ├─ ★ AI 策略迭代更新: 提供者可启用 AI 自动监控策略实盘表现 →
  │     alpha 衰减时自动生成优化版本 → 回测验证 → 推送给订阅者升级。
  │     这是对抗 R-A 的核心手段，也是区别于 MQL5 Market 的供给侧卖点。
  ├─ 平台抽成比例: 15-30%，在 Admin 后台可随时调整
  │    - 支持全市场统一下调（如营销活动期间降为 5%）
  │    - 支持按提供者等级差异化（Bronze 25% → Gold 12%）；重度多账户用户落高档订阅，provider 拿更高分成
  │    - 调整即时生效，无需重新部署
  └─ 提供者到手 = 策略售价 × (1 - 当前抽成比例)

AI token: 已完成。平台提供免费额度 + 超额付费 + 用户自带 API key。
```

**设计原则**：
- 平台订阅控制"能不能用 + 能绑几个账户"，策略抽成分享"策略价值"。
- **策略定价优先订阅制**：既对齐 alpha 衰减，又把“买一次镜像一辈子”的漏损转成持续收入。
- 抽成比例可动态调整——这是运营工具，不是写死的常量。
- 提供者知道当前比例是多少（前端 Author Dashboard 实时显示），调整历史有记录。

### 退款与冻结结算

**背景**：买方需要试用期验证策略，提供者需要防止恶意退款。

**退款套利防护（v4）**：策略在冻结期内**真实盘运行**，买方可能跑满 7 天薅完信号/收益再退款、provider 颗粒无收。规则：**一旦该购买产生实盘成交，则不可自助全额退款**——只能走人工争议审核（判定策略明显失效/欺诈才退）。未产生成交的购买才走自助全额退。

**核心设计**：冻结期钱不动。`marketplace_settlements` 表记录债务关系，结算或退款时直接走终态转账。不走托管账户——引入中间态增加流水复杂度且污染 SystemUserID 钱包。

```
购买时:
  buyer 钱包: AdjustBalanceTx(-amount)    ← 终态，不可撤销
  marketplace_settlements: INSERT         ← 债务记录（frozen）
  SystemUserID: 不受影响

7 天后结算（惰性触发）:
  provider 钱包: AdjustBalanceTx(+provider_amount)
  SystemUserID: AdjustBalanceTx(+platform_fee)
  settlement: status='settled'

7 天内退款:
  buyer 钱包: AdjustBalanceTx(+amount)    ← 全额退回
  settlement: status='refunded'
  （provider 和平台无任何流水——退款对他们不可见）
```

**冻结期配置**：

| 项 | 值 |
|----|-----|
| 默认冻结期 | 7 天 |
| 提供者可调范围 | 3 / 7 / 14 / 30 / 0（0 = 不可退，买即到账） |
| 退款次数限制 | 同一策略同一用户只能退一次 |

**惰性结算（禁用定时器）**：提供者查看 Dashboard、申请提现时，`SettleExpired` 扫描该提供者所有 `frozen` 且 `settles_at <= now()` 的记录，批量执行转账。不需要定时器。

**钱包流水精简**：每笔交易最终只有 2-3 条 `wallet_transactions`（buyer 出 + provider 入 + 平台入），冻结态不产生钱包流水。settlements 表提供完整审计轨迹——谁在什么时候买了什么、冻结多久、最终结算还是退款。

**数据模型**：

```sql
CREATE TABLE marketplace_settlements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    purchase_id UUID NOT NULL REFERENCES user_subscriptions(id),
    buyer_id UUID NOT NULL,
    provider_id UUID NOT NULL,
    amount NUMERIC(20,8) NOT NULL,           -- 总付款
    platform_fee NUMERIC(20,8) NOT NULL,      -- 平台抽成
    provider_amount NUMERIC(20,8) NOT NULL,   -- 提供者到手
    status VARCHAR(20) NOT NULL DEFAULT 'frozen',  -- frozen / settled / refunded
    refund_window_days INT NOT NULL DEFAULT 7,
    freezes_at TIMESTAMPTZ NOT NULL,
    settles_at TIMESTAMPTZ NOT NULL,          -- freezes_at + refund_window_days
    settled_at TIMESTAMPTZ,
    refunded_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

**提供者 Dashboard 余额展示**：
- "可用余额" = 已结算（`status='settled'` 的 `provider_amount` 总和 + 过往 wallet 余额）
- "待结算" = 冻结中（`status='frozen'` 的 `provider_amount` 总和 + 预计解冻日期）
- 提现只能提取"可用余额"

### 战略取舍（v3 已决）

**选定路线 A：纯第三方策略市场。** 平台不自营策略，不参与客户竞争。

理由：
1. 平台自营策略 = 和提供者抢客户 = 供给端信任崩塌。没有提供者会在一个和他们竞争的市场上卖策略。
2. 不自营 → 平台收入来自平台订阅 + 策略抽成（非自营策略收益）→ 平台和提供者利益完全对齐。
3. AI 的作用是**帮提供者更快生成策略**（供给侧工具），不是替代提供者（供给侧竞争）。

| 平台做 | 平台不做 |
|--------|---------|
| 提供策略编写工具（AI 辅助生成） | 平台自有策略销售 |
| 运营策略市场（上架/搜索/交易/结算） | 代客交易/跟单服务 |
| 平台内策略执行（VM + broker 对接） | 资产管理/投顾 |
| 平台订阅 + 策略销售抽成 | 自营策略收益 / 交易成交返佣 |
| 兼容所有 MT broker | 和特定 broker 绑定合作 |

### 关键风险与应对（v3）

| ID | 风险 | 严重度 | 应对 |
|----|------|--------|------|
| R-P | **平台依赖 MetaQuotes**：MQL 语言/生态/授权归其单方控制 | 🔴 | IR 层可接入非 MQL 前端、监测 MT 政策 |
| R-A | **Alpha 衰减（模式命门）**：策略卖爆即失效，侵蚀续费率/口碑；且直接决定“多账户锁”能撑多久 | 🔴🔴 | **主推订阅制**（衰减即退订、三方对齐）+ 策略容量上限 + 失效自动退役/降权 + 供给侧持续上新维持“永远有能赚钱的策略” + 滚动展示实盘 |
| R-1 | **回测美化**：实盘远差于回测 | 🔴 | 实盘跟踪（1.1）+ walk-forward/purged CV（1.2） |
| R-J | **辖区限制**：某些辖区可能禁止策略销售 | 🟡 | IP 封禁 + 用户注册时辖区声明 |
| R-C | **冷启动**：无买方↔无卖方 | 🟡 | 初期全部免费 + AI 辅助提供者快速上架 |
| R-T | **信任鸿沟**：用户不信任陌生策略 | 🟡 | 实盘业绩公开 + 提供者验证 + 回测质量标准 |

**关于合规**：暂不考虑。目前阶段体量太小，不值得为合规投入资源。有问题辖区直接封 IP。

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
| ~~跟单引擎~~ （已建，但**不纳入产品**） | `copytrade.go` | 代码已存在，但按 v3/v4 产品边界**不做跟单**（见「产品边界」）。不接 UI、不对外暴露；保留为内部史料，待确认无依赖后可下线 |
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

## Phase 0.5：辖区风险评估 🟡（轻量前置）

> **定位**：确认哪些辖区可以运营、哪些需要封 IP。我们不碰资金、不做投顾，策略在用户自己的 broker 账户执行。

**Tasks**:
- [ ] 列出主要金融监管辖区（US/UK/EU/JP/AU/SG/HK 等）对”策略销售平台”的监管态度
- [ ] 确定 IP 封禁机制：`ip_blocks` 表 + middleware
- [ ] 免责声明文案（5 语言）——平台是工具提供者，不是投资顾问
- [ ] 用户注册时声明辖区（self-declaration），配合 IP 校验

**Gate**：IP 封禁上线后即可。不阻塞任何 Phase。

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

### 1.4 风险声明与合规

**Why**: 避免法律风险，建立专业形象。平台定位是工具+市场，不涉及资金管理和投顾。

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
- [ ] 试用过期使用惰性求值（`CanAccessCode` 调用时自动清理过期试用）——禁用定时任务
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

**Why**: 后端退款逻辑已完成（`refund.go`），缺少用户入口。需配合冻结结算机制。

**Tasks**:
- [ ] 前端已购策略列表增加"申请退款"按钮（条件：购买后 < 冻结期天数 + 未退过）
- [ ] 退款申请 → 买方钱包直接加回 → settlement 标记 refunded（不经过托管账户，提供者和平台无感知）
- [ ] Admin 退款审核面板（自动化退款不需要人工审核；争议退款才走人工）

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

### 4.6 策略提现

**Why**: 提供者赚到的钱需要能提出来。

**Tasks**:
- [ ] 复用 HD 钱包系统（ADR-0026）的冷签名提现流程
- [ ] 提供者在 Author Dashboard 发起提现（输入金额 + 目标地址）
- [ ] 平台审核 → 冷签名 → 广播交易
- [ ] 提现记录 + 状态追踪

---

## Phase 5：增值 + AI 供给侧持续 🟢 P3

> **目标**：核心策略市场稳固后，用 AI 保持策略供给侧新鲜度（对抗 R-A Alpha 衰减），同时提升客单价与提供者激励。

### 5.1 AI 策略迭代更新

**Why**: Alpha 衰减（R-A）是策略市场的结构性风险——策略赚钱→买的人多→失效。AI 迭代更新是对抗衰减的系统性手段，也是区别于 MQL5 Market 的供给侧卖点（MQL5 的 EA 是死的，我们的策略可以自我进化）。

**Tasks**:
- [ ] AI 自动监控已上架策略的实盘表现（收益率趋势、夏普下滑、回撤突破）
- [ ] 检测到衰减信号 → AI 自动生成优化版本（参数调优 or 逻辑修正）
- [ ] 优化版本自动回测 + walk-forward 验证 → 达标则生成新版本
- [ ] 提供者 Dashboard 收到”策略 v1.2.0 优化建议”，一键审核发布
- [ ] 订阅者收到”你订阅的策略有新版本”通知，可选择升级

**依赖**: Phase 4.4（策略版本管理）

### 5.2 策略捆绑包

**Why**: 提升客单价，促进交叉销售。

**Tasks**: 同上。

### 5.3 阶梯费率

**Why**: 激励高质量提供者，提升平台收入。

**Tasks**: 同上。

---

## 依赖关系

```
Phase 0.5 (辖区风险评估) ← 轻量前置，不阻塞任何 Phase
        │
Phase 1 (信任基础设施)
  ├── 1.5 价格模型修复 [BUG-1] ← 立即修
  ├── 1.2 回测门槛（walk-forward/purged CV）
  ├── 1.1 实盘跟踪 ← 需日积月累，不能压缩
  ├── 1.3 提供者验证
  └── 1.4 风险声明
        │
Phase 2 (AI 策略供给) ← 依赖 Phase 1.2（质量门槛）
        │
Phase 3 (增长引擎) ← 依赖 Phase 1+2
        │
Phase 4 (平台运营) ← 可与 Phase 3 并行
        │
Phase 5 (增值 + AI 供给侧持续) ← 用户量达临界质量后启动
```

**排序摘要**：
- Phase 1 的核心是**信任**：实盘跟踪 + 回测质量 + 提供者验证。
- AI 的作用是**供给侧工具**：Phase 2 帮提供者生成策略，Phase 5 帮策略自我迭代对抗 alpha 衰减。
- 跟单/白标/broker 集成已移除，不在任何 Phase。
- 平台抽成比例在 Admin 后台可随时调整，不写死在代码里。
- 合规暂不考虑，有问题辖区直接封 IP。

---

## 关键指标

| 指标 | 现状 | Phase 1 目标 | Phase 2 目标 | Phase 3 目标 |
|------|------|-------------|-------------|-------------|
| 上架策略数 | ~10 | 30+ | 80+ (AI 辅助) | 200+ |
| 活跃提供者数 | <10 | 20 | 50 | 100+ |
| 月活跃买方 | <50 | 200 | 1000 | 5000+ |
| 买方留存率（30 日） | — | 建立度量 | ≥40% | ≥50% |
| 月 GMV | — | $2K | $20K | $100K+ |
| 平台月收入（抽成） | — | $500 | $5K | $25K+ |
| 策略平均评分 | — | ≥4.0 | ≥4.2 | ≥4.3 |
| 退款率 | — | <15% | <12% | <10% |

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
| 风控管线 | `risk-gate` 6 门管线 + OMS | 平台内策略执行（多账户实盘下单风控） |
| i18n 框架 | 5 语言已部署 | 所有前端新页面 |
| 平台订阅 | Free/Pro/Enterprise tiers | GAP-3 联动修复 |
