# ADR 0006 · 平台共享资源 vs 用户私有资源（C2C 架构订正）

- **日期**：2026-05-23
- **状态**：Accepted
- **影响范围**：全部业务表、service 层、marketplace、admin 鉴权
- **取代**：02-overview.md §7 表格中"多租户 user_id（单租户）"的旧表述

## 1. 背景

ant 当前数据模型是 **user-as-tenant**：每个 user 是独立的小宇宙，自有 `mt_accounts` / `strategy_templates` / `factor_definitions` / `ai_agent_definitions`，平台精选数据通过 `seed_default_templates.go` per-user 复制一份。

这是 **B2B SaaS 模型**，但 ant 产品定位是 **C2C 散户平台**（"让普通人能用上专业级量化"）。两者对数据所有权的假设完全相反：

| 维度 | B2B SaaS（user-as-tenant，当前）| C2C 散户平台（应有目标）|
|---|---|---|
| 数据归属 | 每 user 独立，互不可见 | 平台共享 + 用户私有两段 |
| 官方策略 | 给每个 user 复制一份 | 平台单实例，所有 user 读 |
| 跨用户能力 | 不需要 | 跟单/订阅/排行/marketplace 是核心 |
| 管理员模型 | tenant admin（user.role=admin）| 平台运营者独立鉴权 |

**症状**（直接证据）：

| 资源 | 现状 | 问题 |
|---|---|---|
| `strategy_templates` | per-user 复制（`seed_default_templates.go`）| 改一处官方策略需遍历所有 user |
| `factor_definitions` | per-user 复制 | 同上；MA/RSI 这些通用因子重复万次 |
| `ai_agent_definitions` | `seedDefaults(userID)` per-user | 同上 |
| `marketplace/` | 仅 `repo.go` 空壳 | **做不出来**——user-as-tenant 无跨 user 共享通路 |
| `admins` | 不存在；走 `users.role='admin'` | 业务用户与平台运营混为一谈 |
| `follows` / `subscriptions` | 不存在 | 跟单/社交是 C2C 的核心，但模型不支持 |
| `broker_symbols` | 平台共享（无 user_id）✓ | 已对，验证了"平台层"是必要的 |

## 2. 决策

采用 **B 模型：单平台 + 平台共享层 + 用户消费层**，并为将来 **C 模型（多 tenant 白标输出）** 预留命名空间。

### 2.1 数据归属二分法

**平台共享**（无 `user_id` 外键，所有用户可读）：
- `platform_strategies`（含官方精选 + 用户上架的跟单策略）
- `platform_factors`（内置 MA/RSI/MACD/Bollinger 等通用因子）
- `platform_ai_agents`（默认 AI agent 模板，替代 `defaultAgentTemplates.ts` 硬编码 + per-user `seedDefaults`）
- `broker_symbols` ✓（已经是）
- `admins`（平台运营者，独立鉴权）

**用户私有**（必须 `user_id` 外键 + RLS）：
- `mt_accounts`（用户绑定的 broker 账户）
- `user_strategies`（fork 自 platform 或自建）
- `user_factor_overrides`（仅个性化参数，不是新 factor 实例）
- `user_ai_agents`（fork 自 platform_ai_agents 后的个性化）
- `orders`、`positions`、`trades`
- `user_subscriptions`、`copy_trade_links`（跨用户关系，user_id 是订阅者）

### 2.2 跨用户层（marketplace 真正落地的前提）

```
platform_strategies
    ▲
    │ publish (user 自创策略上架)
    │
user_strategy_publishes (user_id → platform_strategy_id, royalty_pct)
    │
    │ subscribe
    ▼
user_subscriptions (subscriber_user_id, target_user_id|target_strategy_id, kind)
    │
    │ kind=copy_trade
    ▼
copy_trade_links (subscription_id, from_account_id, to_account_id, ratio)
```

**没有平台共享层 → 没有 marketplace**。这是 ant 产品形态成立的前提。

### 2.3 admin 鉴权独立

废弃 `users.role` 字段中的 `admin` 取值；新建 `admins` 表 + 独立 JWT scope `platform:admin`。

理由：业务用户和平台运营是两类主体，权限模型完全不同：
- 业务用户：管理自己的账户/策略/订单
- 平台运营：审核策略上架、运营 marketplace、看全局指标、处置违规账户

复用 `users.role` 会导致：①平台运营必须先注册成消费者；②权限边界模糊，admin user 同时能下单容易出事故。

### 2.4 为 tenants 预留（M10+ 不动现表结构）

所有 `platform_*` 表当前**不加** `tenant_id` 列。但代码层引入 `PlatformScope` 接口：

```go
// backend/internal/platform/scope.go
type PlatformScope interface {
    // 当前实现：始终返回 "ant"（隐式单平台）
    // M10+ 实现：从 JWT/header 解析 tenant_id（broker 白标）
    Current(ctx context.Context) string
}
```

所有读 platform_* 表的 query **必须** `WHERE platform_id = scope.Current(ctx)`（当前都返回 'ant'，是 no-op）。M10+ 切换 tenants 表时只需：① 表加 `tenant_id` 列默认 'ant'；② 改 `PlatformScope` 实现；③ 业务代码零改动。

## 3. 备选方案

| 方案 | 否决 |
|---|---|
| 维持 user-as-tenant + 在每个 user 下复制官方数据（现状）| marketplace/跟单做不出；改一处官方策略需 N 倍写入 |
| 直接上 multi-tenant（C 模型）| 当前用户基数小，引入 tenant 维度过早；学习/迁移成本高 |
| 用 `user_id = NULL` 标记平台数据 | NULL 语义脆弱；JOIN 复杂；与 RLS 冲突 |
| 用预留 user `id='00000000-...-platform'` | 误用风险（被当作普通账户操作）|

## 4. 后果

### 正面
- marketplace、跟单、社交功能成立
- 官方策略改一处全用户生效
- admin 权限边界清晰
- 为白标输出预留路径

### 负面
- 需要数据迁移（per-user 复制的官方数据 → 平台单实例 + 删 per-user 副本）
- 现有 `seed_default_templates.go` 逻辑变更（seed 到 platform 而非 per-user）
- service 层 List 接口变为 `platform ∪ user_*` 联合查询

### 中性
- platform_* 表读多写少，cache-friendly
- RLS 仅对 user_* 表启用（平台表全用户可读）

## 5. 实施路径（M8 子里程碑）

> 详见 `docs/plan/BACKLOG.md` M8.0-M8.3。本 ADR 仅做架构决策，不指定实施细节。

| 子里程碑 | 内容 |
|---|---|
| **M8.0** | migrations：新建 `platform_strategies` `platform_factors` `platform_ai_agents` `admins` + ETL（per-user 副本去重 + 数据迁出）|
| **M8.1** | service 重构：`StrategyTemplateService` `FactorService` `AIAgentService` 改为 `platform ∪ user_*` 联合 |
| **M8.2** | marketplace 真做：`user_strategy_publishes` + `user_subscriptions` + `copy_trade_links` + 前端 marketplace 页 |
| **M8.3** | admin 鉴权迁移：`users.role='admin'` → `admins` 表；JWT scope `platform:admin`；前端 admin 页 401 路径独立 |

## 6. 验证方式

### M8 完成时

```bash
# (1) per-user 官方副本清零
docker exec ant-postgres psql -U ant -d ant -tAc \
  "SELECT count(*) FROM strategy_templates WHERE is_official=true GROUP BY user_id" \
  | awk '$1>0{exit 1}'

# (2) platform_* 表存在且非空
for t in platform_strategies platform_factors platform_ai_agents admins; do
  docker exec ant-postgres psql -U ant -d ant -tAc \
    "SELECT count(*) FROM $t" | awk '$1<1{exit 1}'
done

# (3) admin 鉴权独立
! docker exec ant-postgres psql -U ant -d ant -tAc \
    "SELECT count(*) FROM users WHERE role='admin'" \
  | awk '$1>0{exit 1}'

# (4) PlatformScope 接口被调用
grep -rE 'scope\.Current\(ctx\)' backend/internal/repository/queries/ | wc -l \
  | awk '$1<5{exit 1}'

# (5) marketplace 跨用户订阅闭环
# 用户 A 发布策略 → 用户 B 订阅 → B 的 mt 账户出现 copy 单
# 详见 tests/e2e/copy_trade_test.go (M8.2)
```

## 7. 关联

- **取代**：`docs/architecture/02-overview.md` §7 行"多租户 user_id（单租户）"
- **新增**：`02-overview.md` §8 不变量 11/12/13（见下文）
- **影响**：`AGENT.md §0.3` 阅读清单加入本 ADR；M8 子里程碑替换 BACKLOG 现有 "M8-X1 拆 service"
- **不影响**：M7 mdgateway/mthub/factorsvc/quantengine 重做（这些是 L2-L5，不涉及业务数据归属）
