# Ant 主线化与 AlfQ 内核迁移计划

**Status**: Draft v1
**Date**: 2026-05-23
**Author**: 架构决策讨论沉淀
**关联**: `/opt/alfq/docs/design/multi-broker-symbol.md`

---

## 0. 决策背景（一页纸读懂为什么）

### 0.1 三项目盘点（实测）

| 项目 | 后端 LOC | 前端页面 | 状态 | 定位 |
|---|---|---|---|---|
| **ant** | 146k Go | 50 | **未投产**（不在 git，无运行容器） | "面向外汇量化的 SaaS，AI 自然语言生成 Python 策略，散户" |
| anttrader | 147k Go | 69 | 生产运行中 (healthy 37h+) | ant 的前身 |
| AlfQ | 25k Go | 25 | 内部测试 | B2B 多租户量化引擎（重构中） |

### 0.2 产品愿景

> **让普通散户用自然语言把交易想法变成自动执行的策略，并把好策略变现。**

### 0.3 主线决定

**ant 是主线项目。** 理由：

1. ant 文档明文写着"AI 驱动策略生成 / 降低量化交易门槛 / 自然语言生成 Python 策略 / 沙箱执行"，**字字命中需求**
2. ant 已经实现了 `debate_v2_*`（多模型 AI 辩论）+ `strategy-service`（Python 沙箱）+ `ai_wizard`（自然语言向导）+ 完整 auth/Register
3. anttrader 是前身，ant 已经覆盖其核心功能；AlfQ 走的是 B2B 多租户路线，与散户产品定位错配
4. ant 文档体系最完整（86 篇 docs），工程治理基础好

### 0.4 三项目处置

| 项目 | 处置 |
|---|---|
| **ant** | 主线，所有未来开发进 ant |
| anttrader | 保持运行至 ant 上线，迁移用户/数据后退役 |
| AlfQ | 降级为"内核仓库"。把专业模块移植到 ant，移植完成归档 |

### 0.5 待用户确认的前置事项

- [ ] **ant 必须立即纳入 git**：`cd /opt/ant && git init && 推送到远端`
- [ ] 确认 `deploy-quant-engine` `deploy-assistant-svc` 容器是否服务于 ant
- [ ] 用户/数据从 anttrader 迁到 ant 的时间窗

---

## 1. 迁移总览：哪些金子要从 AlfQ 搬过来

### 1.1 金子清单（按价值密度排序）

| # | AlfQ 模块 | 行数 | 价值 | 解决 ant 的什么问题 |
|---|---|---|---|---|
| 1 | `internal/oms/` OMS 状态机 + 持久化 | ~2k | ★★★★★ | ant 当前 strategy_schedule_runner 直接发单，缺状态机；订单状态丢失/重复下单/风控旁路风险 |
| 2 | `internal/risksvc/` 风控规则引擎 | ~1.5k | ★★★★★ | ant 风控写在 `RiskControl` JSONB 字段+硬编码逻辑；不可插拔、不可审计 |
| 3 | `canonical_symbols` + `broker_symbols` 设计 | 设计文档 + ~1k 实现 | ★★★★★ | ant `strategies.symbol` 是裸字符串，跨 broker 名字不同会下单失败 |
| 4 | `internal/symbolsync/` 归一化 | ~1k | ★★★★ | ant 没有 symbol 元数据自动同步 |
| 5 | `internal/mthub/` MT 会话池 + `mtapi` 抽象 | ~3k | ★★★★ | ant 直连 broker，重连/会话复用待加固 |
| 6 | `OrderExecutor → BrokerAdapter` 接口抽象 | ~0.5k | ★★★★★ | ant 当前耦合 MT4/MT5 直接调用；未来加币圈/股票 broker 时痛苦 |
| 7 | `internal/factorsvc/` 因子计算 + window buffer | ~2k | ★★★ | ant 已有指标，但若 AI 生成策略需要更多因子可扩展 |
| 8 | NOTIFY 热加载模式（strategy_revisions） | ~0.5k | ★★★ | ant 策略变更后能否实时生效需确认 |
| **小计** | — | **~11k 行** | — | — |

### 1.2 不要搬的部分

| AlfQ 模块 | 不搬理由 |
|---|---|
| 多租户 schema（`tenant_id` + RLS tenant policy） | ant 是 user-centric，对的方向 |
| `cmd/quant-engine/` 服务壳子 | ant 已有 `strategy_schedule_runner` 等价物 |
| `configs/specs/*.yaml` 文件式 spec | ant 用 DB 存策略，对的方向 |
| AlfQ 的前端 25 页 | ant 的 50 页更完整 |
| ConnectRPC 协议层 | ant 已用 ConnectRPC，无需替换 |
| `Tenants/Users/Audit/ServiceManagement` 后台 | 散户产品不需要 |

### 1.3 ant 需要新建的部分（三项目都没有）

| 模块 | 优先级 |
|---|---|
| 策略市场（listing + subscription + 跟单引擎 + 分润结算） | P1（M5 阶段） |
| 自然语言 → 策略意图结构化解析层（在 debate_v2 之上） | P1（增强已有） |
| 平台冷锁回测/实盘指标（防作者刷数据） | P1（市场上线前提） |

---

## 2. ant 现状契入分析

### 2.1 ant 当前 Strategy 数据模型

```go
// @/opt/ant/backend/internal/model/strategy.go:13-27
type Strategy struct {
    ID          uuid.UUID
    UserID      uuid.UUID    // ✅ user-centric (对的)
    AccountID   uuid.UUID    // 单账户绑定
    Name        string
    Symbol      string       // ⚠️ 裸字符串，没 canonical 概念
    Conditions  JSONB        // 策略条件
    Actions     JSONB
    RiskControl JSONB        // ⚠️ 风控参数塞 JSON，无插件式规则引擎
    Status      string
    AutoExecute bool
}
```

### 2.2 关键缺口（迁移要解决的）

| 缺口 | 后果 | 解法（来自 AlfQ） |
|---|---|---|
| `Symbol` 是裸字符串 | 跨 broker 下单失败、不能批量 | canonical_symbols + broker_symbols 体系 |
| `RiskControl` JSONB | 不可插件、不可审计 | risksvc.Engine + Rule 接口 |
| 无 OMS 状态机 | 订单丢状态、重复发单 | oms.OrderExecutor + state machine |
| Broker 直接调用 | 不可换 broker | OrderExecutor → BrokerAdapter 接口 |
| 单账户绑定 | 不能多账号跑同策略 | strategy_accounts N:N 关联 |

---

## 3. 分阶段迁移路线（M0 → M6）

### M0｜准备（1 周）

**目标**: 让 ant 具备承接迁移的工程基线。

```bash
# 任务清单
[ ] cd /opt/ant && git init && 推送远端，建立分支策略
[ ] docker compose 跑通 ant 完整本地栈，截屏存档（基线）
[ ] 写自检脚本 scripts/self_check.sh：build + test + 起容器 + smoke
[ ] 在 docs/进度/ 新建本计划的进度看板 alfq-port-progress.md
```

**验收**: ant 能 commit/push、自检脚本绿色、本地栈跑通。

---

### M1｜canonical symbol 基础设施（2 周）

**目标**: 解决"不同 broker symbol 不同"的根本问题，是后续所有移植的依赖。

#### M1.1 数据模型

新增 migrations（编号接续 ant 现有）：

```sql
-- 087_canonical_symbols.up.sql
CREATE TABLE canonical_symbols (
    canonical    text PRIMARY KEY,
    asset_class  text NOT NULL,
    base_ccy     text NOT NULL,
    quote_ccy    text NOT NULL,
    description  text NOT NULL,
    enabled      boolean NOT NULL DEFAULT true,
    created_at   timestamptz NOT NULL DEFAULT now()
);

-- 088_broker_symbols.up.sql  (从 AlfQ 复刻)
CREATE TABLE broker_symbols (
    broker_id        uuid NOT NULL,
    symbol_raw       text NOT NULL,
    canonical        text REFERENCES canonical_symbols(canonical),
    digits           smallint, point numeric, tick_size numeric,
    contract_size    numeric, min_lot numeric, max_lot numeric, lot_step numeric,
    swap_long        numeric, swap_short numeric,
    trade_mode       smallint,        -- 0=disabled 1=long_only 2=short_only 3=full
    description      text,
    sessions_quote   jsonb, sessions_trade jsonb,
    raw_payload      jsonb,
    partial          boolean DEFAULT false,
    updated_at       timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (broker_id, symbol_raw)
);
CREATE INDEX idx_broker_symbols_canonical ON broker_symbols(canonical);

-- 089_strategy_symbols.up.sql  (用户白名单，user-scoped)
CREATE TABLE strategy_symbols (
    strategy_id  uuid NOT NULL REFERENCES strategies(id) ON DELETE CASCADE,
    canonical    text NOT NULL REFERENCES canonical_symbols(canonical),
    enabled      boolean NOT NULL DEFAULT true,
    created_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (strategy_id, canonical)
);

-- 注：ant 是 user-centric，不要加 tenant 维度
-- RLS: 通过 user_id (Strategy.user_id 关联) 间接隔离
```

#### M1.2 移植 symbolsync 模块

```
源: /opt/alfq/backend/go/internal/symbolsync/
目标: /opt/ant/backend/internal/symbolsync/

文件清单:
- service.go        (主入口)
- repo.go           (UPSERT broker_symbols)
- types.go          (BrokerSymbol struct)
- mt4_fetch.go      (MT4 拉取实现)
- mt5_fetch.go      (MT5 拉取实现)
- canonical_mapper.go  (新建：归一化规则 BTCUSDm→BTCUSD)
```

**适配点**:
- AlfQ 的 `pgxpool.Pool` → ant 用什么 PG 抽象需要确认（看 `/opt/ant/backend/internal/repository/`）
- 移除 AlfQ 的 RLS `app.tenant_id` 调用，改成无租户上下文

#### M1.3 SymbolResolver

```
源: /opt/alfq/backend/go/internal/adminapi/symbol_resolver.go
目标: /opt/ant/backend/internal/symbol/resolver.go

接口 (保持不变):
ResolveCanonical(accountID, canonical) → (symbolRaw, tradeMode, ok)
ListSupportedCanonicals(accountID) → [canonical]
```

#### M1.4 Strategy 迁移

```sql
-- 090_strategies_canonical_migration.up.sql
-- 1. 加 canonical_symbols 表 join 字段（保留 strategies.symbol 兼容）
-- 2. 写迁移脚本：根据现有 strategies.symbol + accounts.broker_id
--    反查 broker_symbols → 填 strategy_symbols
-- 3. 后续策略写入只用 canonical
```

**工程任务**:
```
[ ] 实现 87/88/89/90 migration
[ ] 移植 symbolsync 包并适配 ant 的 PG 层
[ ] seed canonical_symbols 字典 ~50 个主流商品（BTCUSD/EURUSD/XAUUSD/...）
[ ] symbolsync 接到 ant 现有 broker 连接成功事件
[ ] 实现 SymbolResolver
[ ] 数据迁移脚本 + 反查回填
[ ] 单测：归一化规则覆盖（BTCUSDm/BTCUSD.x/BTCUSDpro/BTCUSD#）
[ ] 集成测：跑通"绑账户→symbolsync→canonical 可解析→trade_mode 正确"
```

**验收**: 任意 broker 账户绑定后，能 `SELECT * FROM broker_symbols WHERE canonical='BTCUSD'` 查到 ≥1 行；`SymbolResolver.ResolveCanonical(accountID, 'BTCUSD')` 返回正确 symbol_raw。

---

### M2｜OMS 状态机移植（2 周）

**目标**: 替换 ant 现有的"策略直接发单"路径，所有订单走 OMS 状态机。

#### M2.1 移植 oms 包

```
源: /opt/alfq/backend/go/internal/oms/
目标: /opt/ant/backend/internal/oms/

文件清单:
- executor.go           (OrderExecutor 主入口)
- order_state.go        (状态机 Transition)
- broker_adapter.go     (BrokerAdapter 接口)
- repo/orders.go        (PG 持久化)
- repo/risk_events.go   (审计事件)
- transition_test.go    (状态机测试)
```

#### M2.2 适配 ant 的 broker 层

```go
// 新建 ant 自己的 broker adapters，实现 oms.BrokerAdapter 接口:
/opt/ant/backend/internal/broker/mt4_adapter.go
/opt/ant/backend/internal/broker/mt5_adapter.go

interface BrokerAdapter {
    Submit(ctx, req *pb.OrderRequest) (*BrokerResp, error)
    Cancel(ctx, ticket) error
    Modify(ctx, ticket, price, stopPrice float64) error
    Query(ctx, ticket) (*pb.Order, error)
}
```

适配方法：把 ant 现有 `internal/service/` 下直接调 mt4/mt5 的代码包装成 BrokerAdapter 实现。

#### M2.3 接入策略调度器

```go
// /opt/ant/backend/internal/service/strategy_schedule_runner.go
// 替换：直接发单 → 走 OMS executor.Submit()
```

#### M2.4 orders 表协调

ant 已有 `orders` 表（migration 001）。需要：
- 加字段 `state int` (NEW/VALIDATED/RISK_APPROVED/SUBMITTED/REJECTED/FAILED)
- 加 `broker_symbol_raw text`（canonical 解析后的实际 symbol）
- 历史订单兼容：旧数据 state 默认设为 SUBMITTED

**工程任务**:
```
[ ] 移植 oms 包，删除 RLS 相关代码
[ ] 新建 broker adapter 包装层
[ ] migration 091: orders 表加 state/broker_symbol_raw 字段
[ ] 修改 strategy_schedule_runner 走 OMS
[ ] 单测：状态机转移合法性 + Insert 持久化
[ ] 集成测：跑通"信号→OMS→broker→ticket 入库 state=SUBMITTED"
[ ] 回归测：ant 现有的下单路径全部替换后 e2e 跑通
```

**验收**: 所有订单 100% 经过 OMS；`SELECT state, count(*) FROM orders GROUP BY state` 显示状态分布合理；故意触发风控拒单时 state=REJECTED + risk_events 有审计行。

---

### M3｜风控引擎移植（1.5 周）

**目标**: 替换 `RiskControl JSONB` 硬编码逻辑为可插件化规则引擎。

#### M3.1 移植 risksvc 包

```
源: /opt/alfq/backend/go/internal/risksvc/
目标: /opt/ant/backend/internal/risksvc/

文件清单:
- engine.go             (Engine + Rule 接口)
- max_position.go
- daily_loss.go
- drawdown.go
- session.go            (交易时段)
- margin.go             (保证金)
- canonical_auth.go     (新建，Gate 1+2)
```

#### M3.2 三层授权链落地

```
Gate-1: strategy_symbols 是否包含 canonical?
Gate-2: 用户级（user_canonical_whitelist）是否包含？  
        ※ 散户产品可省 tenant 层；user 级即可
Gate-3: broker 是否支持 + trade_mode > 0?
```

#### M3.3 ant 老 RiskControl 字段迁移

```go
// 写迁移脚本：strategy.RiskControl JSONB → 用户级风控偏好表
type UserRiskProfile struct {
    UserID         uuid.UUID
    MaxLossPerDay  float64
    MaxDrawdown    float64
    MaxPositions   int
    SymbolWhitelist []string  // canonical[]
}
```

**工程任务**:
```
[ ] 移植 risksvc 8 条规则
[ ] 新增 user_risk_profiles 表 + migration
[ ] 数据迁移 Strategy.RiskControl → UserRiskProfile
[ ] OMS executor 接入 risk engine
[ ] 单测：每条规则的拒单/通过分支
[ ] 集成测：超日亏 → 拒单 + risk_events 写入
[ ] 前端：风控偏好编辑页（基于 ant 现有 settings 模板）
```

**验收**: `grep -r "RiskControl" backend/` 旧硬编码逻辑路径下零；老风控字段进库的策略迁移后行为一致。

---

### M4｜AI 自然语言层 → canonical 对接（1 周）

**目标**: ant 的 debate_v2 / ai_wizard 生成的策略代码必须用 canonical，不能写 broker-specific 名。

#### M4.1 改造 prompts

```
源: /opt/ant/backend/internal/service/debate_v2_prompts.go
改造: prompt 模板里加约束 "symbol 必须是 canonical_symbols 字典中的值"
       附上当前用户 strategy_symbols 白名单作为可选范围
```

#### M4.2 AI 输出校验

```go
// 新建: internal/service/ai_strategy_validator.go
// AI 生成的 Python 代码或 JSON spec 提取出的 symbol 必须命中 canonical_symbols
// 不命中则反馈给 LLM 重生成（最多 3 次）
```

#### M4.3 沙箱执行器适配

```
/opt/ant/strategy-service/app/engine/runner.py
如果策略输出信号的 symbol 不在用户白名单 → 信号被丢弃 + 日志告警
```

**工程任务**:
```
[ ] 改 prompts 加 canonical 约束
[ ] AI 输出校验器
[ ] Python 沙箱信号过滤逻辑
[ ] 端到端：用户说"BTC 跌 3% 后买入" → AI 生成 → 校验 → 实盘走 OMS
```

**验收**: 自然语言生成的 10 条样例策略，100% 输出 canonical 而非 broker-specific 名。

---

### M5｜策略市场（4-6 周，独立大模块）

**目标**: 用户能把好策略上架租/卖，其他用户订阅跟单。

#### M5.1 数据模型

```sql
-- 092_strategy_marketplace.up.sql
CREATE TABLE strategy_listings (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id          uuid NOT NULL REFERENCES strategies(id),
    author_user_id       uuid NOT NULL REFERENCES users(id),
    listing_type         text NOT NULL,  -- 'rent_monthly'|'sell_once'|'copy_trade'
    price_cents          int NOT NULL,
    profit_share_pct     numeric(5,2),   -- 跟单分润 (0-100)
    status               text NOT NULL,  -- 'draft'|'active'|'delisted'
    verified_period      tstzrange,      -- 平台验证时段
    verified_metrics     jsonb NOT NULL, -- 平台冷锁: 年化/最大回撤/胜率/sharpe/样本数
    description          text,
    created_at           timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE strategy_subscriptions (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    buyer_user_id   uuid NOT NULL REFERENCES users(id),
    listing_id      uuid NOT NULL REFERENCES strategy_listings(id),
    start_at        timestamptz NOT NULL DEFAULT now(),
    expires_at      timestamptz,
    status          text NOT NULL,  -- 'active'|'expired'|'paused'|'refunded'
    revenue_share_pct numeric(5,2),
    paid_amount_cents int NOT NULL DEFAULT 0
);

CREATE TABLE strategy_performance_snapshots (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id uuid NOT NULL REFERENCES strategies(id),
    period_start timestamptz NOT NULL,
    period_end   timestamptz NOT NULL,
    annual_return numeric, max_drawdown numeric, win_rate numeric,
    sharpe       numeric, sample_count int,
    is_live      boolean,        -- 实盘 vs 回测
    is_verified  boolean,        -- 平台是否验证
    locked_hash  text,           -- 防篡改
    created_at   timestamptz NOT NULL DEFAULT now()
);
```

#### M5.2 关键设计点（避坑）

| 风险 | 规避 |
|---|---|
| 作者刷假 PnL | 只承认平台跑过且签名锁死的回测/实盘指标 |
| 跟单分润合规 | 首期只做"月租 / 一次性买断"，不做利润分成 |
| 策略产权 | 卖出仅授权运行权，不交付 spec 源码（spec 由平台代管） |
| 表现劣化 | 自动 delist：连续 X 天回撤超阈值 → 状态置 delisted + 通知订阅者 |
| 跟单延迟 | 跟单引擎从作者 strategy 信号 fan-out 到订阅者 OMS（异步队列） |

#### M5.3 跟单引擎

```
作者策略产生信号 → 写入 strategy_signals
↓
copytrade_dispatcher (新增 worker):
   按 listing.status='active' 找出所有订阅者
   按订阅者风控偏好 + 仓位大小放缩
   向每个订阅者的 OMS 提交订单
```

**工程任务**:
```
[ ] migration 092 + seed
[ ] 上架/下架/订阅 RPC
[ ] 平台 backtest 验证 worker (产出 verified_metrics + locked_hash)
[ ] 自动 delist worker
[ ] 跟单 dispatcher
[ ] 前端：策略市场列表页 / 策略详情 / 订阅按钮 / 我订阅的
[ ] 前端：作者侧 上架表单 / 收入面板
[ ] 计费：月租首期接 Stripe 测试模式（或国内通道）
[ ] 单测：分润计算、自动下架边界
[ ] 集成测：跑通"上架→他人订阅→跟单成功→订阅过期"
```

**验收**: 三个测试用户：A 作者、B/C 订阅者；A 策略发信号，B/C 各自账户被跟单；订阅期满后 B 不再被跟单。

---

### M6｜anttrader 退役 & 数据迁移（2 周）

**目标**: 现有 anttrader 用户平滑切到 ant。

```
[ ] 写 anttrader → ant 数据迁移脚本（users / mt_accounts / strategies / orders 历史）
[ ] ant 加"从 anttrader 导入"一键功能
[ ] 灰度切量：邀请用户 / 全量切换 / anttrader 只读模式 / 关停
[ ] 文档：用户切换通告
```

**验收**: 100% anttrader 活跃用户成功登入 ant，账户数据完整。

---

## 4. 整体时间线

| 阶段 | 周数 | 累计 | 关键产出 |
|---|---|---|---|
| M0 准备 | 1 | 1 | git 化、自检脚本 |
| M1 canonical | 2 | 3 | symbol 体系打通 |
| M2 OMS | 2 | 5 | 状态机持久化 |
| M3 风控 | 1.5 | 6.5 | 规则引擎 |
| M4 AI 对接 | 1 | 7.5 | NL → canonical 全链路 |
| M5 策略市场 | 5 | 12.5 | 上架/订阅/跟单 |
| M6 退役 | 2 | 14.5 | anttrader 切换 |

总计 **约 14-15 周**（3.5 个月）。期间 P0 部分（M1-M4）约 **7.5 周**即可让 ant 具备"专业量化内核 + AI 自然语言"的核心能力。

---

## 5. 全局工程纪律（AI 执行约束）

> 以下条款适用于本计划下所有 AI 执行任务。违反任一条 = 任务失败。

1. **不得跳步**：M1 完成验收前不开 M2；每阶段必须有可运行 demo
2. **不得手工 mock 通过测试**：所有验证必须跑真实代码路径
3. **每次提交前必跑**：`go build ./... && go test ./... && bash scripts/self_check.sh`
4. **每次完成阶段必须**：① 贴真实日志证据 ② 更新 `docs/进度/alfq-port-progress.md` ③ 推送 git
5. **不得静默吞错**：所有 error 必须明确处理或上抛；不得 `_ = err`
6. **删除旧路径要彻底**：M2 完成后 `grep "directly call mt4/5 from runner"` 应为零
7. **数据迁移必须可回滚**：每个 migration 配 `.down.sql` + 验证脚本
8. **风险面前停下来问人**：设计不确定的优先 ask，不要自由发挥

---

## 6. 文件级移植映射表（M1-M3 用）

| AlfQ 路径 | ant 目标路径 | 备注 |
|---|---|---|
| `internal/symbolsync/service.go` | `internal/symbolsync/service.go` | 删 tenant 上下文 |
| `internal/symbolsync/repo.go` | `internal/symbolsync/repo.go` | 适配 ant PG 抽象 |
| `internal/symbolsync/mt5_fetch.go` | `internal/symbolsync/mt5_fetch.go` | 直接复制 |
| `internal/adminapi/symbol_resolver.go` | `internal/symbol/resolver.go` | 改包名 |
| `internal/oms/executor.go` | `internal/oms/executor.go` | 删 RLS 调用 |
| `internal/oms/order_state.go` | `internal/oms/order_state.go` | 直接复制 |
| `internal/oms/repo/orders.go` | `internal/oms/repo/orders.go` | 适配 ant 既有 orders 表 |
| `internal/risksvc/engine.go` | `internal/risksvc/engine.go` | 直接复制 |
| `internal/risksvc/*.go` (8 rules) | `internal/risksvc/*.go` | 直接复制 |
| `docs/design/multi-broker-symbol.md` | `docs/专项设计/multi-broker-symbol.md` | 移到 ant 设计文档区 |

---

## 7. 风险登记

| 风险 | 影响 | 缓解 |
|---|---|---|
| ant 不在 git → 移植过程任何 bug 不可回滚 | 致命 | M0 第一件事 git init |
| ant 现有代码风格 vs AlfQ 风格差异 | 中 | 移植时按 ant 风格重写关键 import 和 logger |
| AlfQ 风控规则依赖 AccountState 实时数据，ant 是否提供？ | 中 | M3 前先盘点 ant 的 account state 来源 |
| AI prompt 改动可能导致老用户已生成策略行为变化 | 中 | 老策略冻结模板版本，不强制升级 |
| 策略市场合规（中国境内卖策略涉证券业务？） | 高 | 上架前法务咨询；首期"租"模式更安全 |
| anttrader 用户数据格式不兼容 | 中 | M6 单独立项；提前 1 月做兼容性 dry-run |

---

## 8. 立即下一步

```
[ ] 用户确认本计划方向
[ ] 决定 M0 的 git 远端仓库（GitHub org / 私有 git）
[ ] 评估能投入的人力（决定 M0-M6 是否并行）
[ ] 起 M0 的 issue / 进度看板
```

---

## 9. 附录：与已有 ant 文档的关系

| 已有 ant 文档 | 与本计划关系 |
|---|---|
| `docs/项目技术全景分析报告.md` | 本计划的"现状基线"参考 |
| `docs/接口与数据流架构约定.md` | M2 OMS 接入需遵守 |
| `docs/技术债清单与治理状态.md` | 把"风控 JSONB 不可插件"加进去 |
| `docs/专项设计/` | M1 canonical / M5 marketplace 落到此目录 |
| `docs/进度/模块进度登记簿.yaml` | 新增 alfq-port 条目 |
| `docs/任务交接清单模板.md` | M0-M6 每阶段产出对齐此模板 |
