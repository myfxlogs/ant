# strategy-marketplace — 策略市场

> 策略发布/发现/购买/冻结结算/AI 迭代。对标 MQL5 Market，但策略代码不出平台。

## 代码位置

```
backend/internal/marketplace/         ← 15 文件：publish/purchase/subscription/quality/refund/analytics
backend/internal/connect/marketplace/  ← 14 个 RPC handler + SSE 流
frontend/src/pages/marketplace/       ← MarketplacePage + 14 组件/hooks
```

## 产品边界

- ✅ 策略编写工具 + 策略市场 + 平台内策略执行
- ❌ 不代客交易、不管理资金、不做跟单、不拿牌照、不和白标 broker 绑定、不自营策略

## 核心设计

- **收入**：平台订阅（Free/Pro/Enterprise）+ 策略抽成（15-30%，Admin 后台可调）+ AI token
- **冻结结算**：settlements 表记录债务，冻结期内钱不进入提供者钱包。惰性结算，禁定时器
- **粘性**：策略代码不出平台 → 买一次跑多 MT 账户 → 离开=失去多账户能力
- **反盗版**：提供者策略不可下载，杜绝 MQL5 Market 的 .ex5 反编译问题

## 依赖

```
agent-engine(AI 生成策略) → strategy-marketplace
backtest-engine(质量门槛) → strategy-marketplace
mql-compiler(编译) → strategy-marketplace
account-mgmt(钱包) → strategy-marketplace
```

## 施工计划

- [plans/phase-1-trust-infrastructure.md](plans/phase-1-trust-infrastructure.md)
- [plans/phase-2-ai-strategy-supply.md](plans/phase-2-ai-strategy-supply.md)
- [plans/phase-3-growth-engine.md](plans/phase-3-growth-engine.md)
- [plans/phase-4-platform-ops.md](plans/phase-4-platform-ops.md)
- [plans/phase-5-moat.md](plans/phase-5-moat.md)
- [plans/seo-strategy.md](plans/seo-strategy.md)
- [plans/README.md](plans/README.md) — GLM 施工入口

## 关联文档

- `docs/roadmaps/strategy-marketplace.md` — 设计总纲（v4 定稿）
