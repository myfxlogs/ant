# Phase 5 冻结结算 · 数据流完整性审计

> 审计维度：每个 `tx_type` 查询在 settlement 机制下是否返回正确结果。

---

## 数据流变更

```
变更前:
  购买 → buyer(wallet) -amount       tx_type=purchase
       → provider(wallet) +net        tx_type=sale
       → system(wallet) +fee          tx_type=platform_fee

变更后:
  购买 → buyer(wallet) -amount       tx_type=purchase     ✅ 不变
       → marketplace_settlements     status=frozen        🆕 新表
  结算 → provider(wallet) +net       tx_type=settlement    🆕
       → system(wallet) +fee         tx_type=fee_settlement 🆕
```

---

## 🔴 P0 · 查询返回零（12 处）

### analytics.go

| 行 | 查询 | 返回 | 应改为 |
|----|------|------|--------|
| 195 | `tx_type = 'sale'` | 0 | `tx_type = 'settlement'` |
| 259 | `tx_type = 'sale'` | 0 | `tx_type = 'settlement'` |
| 66 | `tx_type = 'platform_fee'` | 0 | 加 `OR tx_type = 'fee_settlement'` |

### provider_earnings.go

| 行 | 函数 | 查询 | 返回 | 应改为 |
|----|------|------|------|--------|
| 46 | TotalEarnings | `tx_type = $2` / `TxTypeSale` | 0 | `tx_type = $2` / `TxTypeSettlement` |
| 88 | TotalSales | `tx_type = 'sale'` | 0 | `tx_type = 'settlement'` |
| 126 | ListTransactions | `IN ('sale','refund_reversal')` | 0 | `IN ('settlement','fee_settlement','refund_reversal')` |
| 142 | RecentTransactions | `IN ('sale','refund_reversal')` | 0 | `IN ('settlement','fee_settlement','refund_reversal')` |

### publisher_analytics.go

| 行 | 查询 | 返回 | 应改为 |
|----|------|------|--------|
| 16 | `tx_type = 'sale'` | 0 | `tx_type = 'settlement'` |
| 139 | `tx_type = 'sale'` | 0 | `tx_type = 'settlement'` |

### fee_tier.go

| 行 | 查询 | 返回 | 应改为 |
|----|------|------|--------|
| 46 | `tx_type = 'sale'` | 0 | `tx_type = 'settlement'` |
| 153 | `tx_type = 'sale'` | 0 | `tx_type = 'settlement'` |

---

## 🔴 P0 · 逻辑缺陷（3 处）

### 1. GetProviderEarnings 未触发惰性结算

`provider_earnings.go:34` — `GetProviderEarnings` 是提供者查看收入的核心入口，但未先调用 `SettleExpired`。对比 `GetPublisherStats`（`publish.go:108`）已正确触发。

提供者打开 Earnings 页面 → 看到收入为 0 → 误以为平台没给他钱。必须加上 `SettleExpired`。

### 2. 提现/购买时未触发惰性结算

`SettleExpired` 仅在 `GetPublisherStats` 中调用。以下场景未触发：
- 提供者申请提现 → 未结算的余额不可提
- 新购买时提供者已有过期结算 → 未触发结算

建议：`RequestWithdrawal` 和 `PurchaseStrategy` 入口也加 `SettleExpired`。

### 3. SettledCount 返回值不准确

`settlement.go:139` — `SettledCount: len(batch)`。循环中 `continue` 跳过的失败结算未扣除。应改用成功计数器。

---

## 🟡 P1 · 注释过时

`analytics.go:61` — 注释：`"reversals use TxTypeRefundReversal"` 已过时。当前退款走 settlement 机制（frozen→直接 refunded），settled 的退款才走 reversal。

---

## 🟢 P2 · 可优化

- `addDecimalStrings`：循环中反复 string→decimal→string。用 `decimal.Decimal` 累加器替代。
- `SettleExpiredAll`：串行调用，大量 provider 时慢。当前量小不需要优化。
- Refund window 不可配置：始终用 `DefaultRefundWindowDays=7`——按设计文档提供者可设 3/7/14/30/0。

---

## 审计结论

**12 个查询返回零 + 3 个逻辑缺陷 + 1 个 JOIN 遗漏 = 16 个问题，全在 Phase 5 settlement 引入的数据流变更中。**

`GetPublisherStats`（publish.go）已正确更新——证明 GLM 知道怎么修。其余文件遗漏了同样的变更。

**P0 全部修完后，P2 的 refund window 可配置化留到后续。**

---

## 修复记录（2026-07-22）

| # | 问题 | 文件 | 修复 |
|---|------|------|------|
| P0-1 | SettledCount 返回 len(batch) | settlement.go | 改用 settledCount 计数器，仅成功时递增 |
| P0-2 | TotalEarnings 查 tx_type=sale | provider_earnings.go:46 | → TxTypeSettlement |
| P0-3 | TotalSales 查 tx_type=sale | provider_earnings.go:88 | → TxTypeSettlement |
| P0-4 | ListTx 查 IN('sale','refund_reversal') | provider_earnings.go:126,142 | → IN('settlement','fee_settlement','refund_reversal') |
| P0-5 | Revenue trend 查 tx_type=sale | publisher_analytics.go:16 | → tx_type='settlement' + description filter 更新 |
| P0-6 | Strategy breakdown 查 tx_type=sale | publisher_analytics.go:139 | → tx_type='settlement' + description filter 更新 |
| P0-7 | Platform revenue 查 tx_type=platform_fee | analytics.go:66 | → tx_type='fee_settlement' |
| P0-8 | Top strategies 查 tx_type=sale | analytics.go:195 | → tx_type='settlement' |
| P0-9 | Top providers 查 tx_type=sale | analytics.go:259 | → tx_type='settlement' |
| P0-10 | Fee tier 查 TxTypeSale | fee_tier.go:46,153 | → TxTypeSettlement |
| P0-11 | GetProviderEarnings 未触发惰性结算 | provider_earnings.go:34 | 添加 SettleExpired 调用 |
| P0-12 | PurchaseStrategy 未触发惰性结算 | purchase.go:96 | 添加 SettleExpired（publisher）在 fee tier 计算前 |
| P0-13 | subJoinOnClause 不支持 settlement idem_key | types.go:141 | 添加 marketplace_settlements 子查询 JOIN 路径 |
| P1-1 | analytics.go 注释过时 | analytics.go:61 | 更新为 fee_settlement 说明 |
| P2-1 | addDecimalStrings 低效 | settlement.go | 改用 decimal.Decimal 累加器，删除 helper |

**验证**: `go build ./...` ✅ | `go test ./internal/marketplace/...` ✅ | `gofmt` ✅

---

## 二次复查（2026-07-22 02:00）

### 🔴 P0-14 — `publisher_analytics.go` description LIKE 永不匹配

`getRevenueTrend` 和 `getStrategyBreakdown` 使用 `description LIKE 'Settlement for purchase %'` 匹配策略标题，但结算描述格式是 `"Settlement for purchase {settlementUUID}"` — 不包含策略标题。查询永远返回 0。

**修复**: 改为通过 `marketplace_settlements.purchase_id → user_subscriptions` JOIN 获取策略/类型信息，不再依赖 description 文本匹配。

### 🟢 P2-2 — `refund.go` 对同一 settlement 行执行两次 SELECT

先 `SELECT status, id` 再 `SELECT provider_amount, platform_fee` — 同一行两次查询。

**修复**: 合并为单次 `SELECT status, id, provider_amount, platform_fee ... FOR UPDATE`。

### ✅ 已确认无问题

- **`SettleExpired` 在 `PurchaseStrategy` 事务内调用**: `SettleExpired` 开独立事务（`s.pg.Begin`），与 `PurchaseStrategy` 的 `tx` 分离。若 `PurchaseStrategy` 回滚，已结算的 publisher 余额不受影响 — 这是正确行为，因为结算与购买是独立操作。
- **`subJoinOnClause` REPLACE 安全性**: `mkt-settle-` 不包含 `mkt-sale-` 子串，`mkt-fee-settle-` 经 REPLACE 后不匹配任何 settlement UUID — 无误匹配。
- **连接池**: `SettleExpired` 在 `PurchaseStrategy` 内额外占 1 连接，pool=25，无耗尽风险。

**最终验证**: `go build ./...` ✅ | `go test ./internal/marketplace/... -count=1` ✅ | `gofmt` ✅

---

## 三次复查（2026-07-22 02:10）

### 🔴 P0-15 — `analytics.go` `providerRev` 计算错误

`providerRev = totalGMV - platformRev`。`totalGMV` 包含所有购买（含冻结未结算的），但 `platformRev` 只含已结算的 `fee_settlement`。两者时间窗口不对齐，`providerRev` 被高估。

**修复**: 直接从 `tx_type = 'settlement'` 查询 `providerRev`，不再从 GMV 减去平台收入。

### 🔴 P0-16 — `getStrategyBreakdown` JOIN 笛卡尔积导致评分膨胀

`LEFT JOIN wallet_transactions` + `LEFT JOIN marketplace_ratings` 产生 M×N 行。5 个评分 + 3 笔结算 → `COUNT(mr.rating)` 返回 15 而非 5。

**修复**: 收入用子查询（`SELECT SUM(...) FROM wallet_transactions JOIN marketplace_settlements JOIN user_subscriptions`），评分 JOIN 独立。

### 🔴 P0-17 — `ProviderEarningsResult` 缺少 `PendingSettlement` 字段

提供者查看收益时看不到待结算余额。

**修复**: 添加 `PendingSettlement` 字段 + 查询 `marketplace_settlements WHERE status='frozen'`，proto 新增 `pending_settlement = 7`，handler 传递，前端新增卡片展示。

### 🔴 P0-18 — 前端 tx type Tag 颜色仍匹配 `sale`

`ProviderEarningsPanel.tsx:54` — `v === 'sale'` 永不匹配新类型。

**修复**: 改为 `settlement` → green, `fee_settlement` → blue, `refund_reversal` → red。

### 修复文件

| 文件 | 修改 |
|------|------|
| `analytics.go` | `providerRev` 直接查 `settlement` tx_type |
| `publisher_analytics.go` | `getStrategyBreakdown` 收入改子查询，消除笛卡尔积 |
| `provider_earnings.go` | 添加 `PendingSettlement` 字段 + frozen 余额查询 |
| `marketplace_service.proto` | `ProviderEarnings` 新增 `pending_settlement = 7` |
| `marketplace_handler_provider_earnings.go` | 传递 `PendingSettlement` |
| `ProviderEarningsPanel.tsx` | 新增 Pending Settlement 卡片 + 修复 tx type 颜色 |

**最终验证**: `go build ./...` ✅ | `go test ./internal/marketplace/... -count=1` ✅ | `gofmt` ✅ | `buf generate` ✅

---

## 四次复查（2026-07-22 02:20）— 第一性原则 + 代码整洁

### 🔴 P0-19 — `AdjustBalanceTx` 幂等检查在余额更新之后 → 重试双扣

`AdjustBalanceTx` 先 `UPDATE balance` 再检查 `idem_key`。若 `ledgerChainInsert` 返回 `ErrIdempotentReplay`，余额已被修改但事务未回滚（`SettleExpired` 用 `continue` 跳过，事务仍提交）。下次重试时余额再次被修改 → **双扣/双贷**。

**根因修复** (`wallet_repo.go`): 将幂等检查移到 `UPDATE balance` 之前。若 `idem_key` 已存在，直接返回当前钱包状态，不修改余额。

**防御层** (`settlement.go`): `SettleExpired` 中对 `ErrIdempotentReplay` 做显式判断，视为成功（钱包已入账），不跳过结算。

### 🟡 P1-2 — frozen settlement INSERT SQL 重复 3 处

`purchase.go`、`bundle_purchase.go`、`service_subscription.go` 各有一份相同的 `INSERT INTO marketplace_settlements` SQL。

**修复**: 提取 `createFrozenSettlementTx` helper 到 `settlement.go`，3 处调用统一。

### 🟢 P2-3 — 遗留常量未标注 legacy

`TxTypeSale`、`TxTypePlatformFee` 及 `IdemKeySale`、`IdemKeyFee`、`IdemKeyRenewSale`、`IdemKeyRenewFee` 不再用于新代码，但 `subJoinOnClause` 仍需匹配历史数据。

**修复**: 分离 active/legacy 常量块，注释标注 legacy。

### 🟢 P2-4 — `refund.go` 注释过时

`RefundPurchase` 注释仍写 "debits the publisher and platform fee"，但实际是 settlement-based 退款。

**修复**: 更新注释。

### ✅ 已确认无问题（第一性原则验证）

| 原则 | 验证 |
|------|------|
| **幂等性** | `AdjustBalanceTx` 先查 `idem_key` 再改余额 → 重试安全；`SettleExpired` 对 replay 显式处理 |
| **原子性** | 所有写操作在单事务内；`SettleExpired` 开独立事务（与购买操作解耦） |
| **一致性** | `providerRev` 直接查 `settlement` tx_type，不从 GMV 推导；评分/收入 JOIN 独立无笛卡尔积 |
| **惰性结算** | 3 触发点（dashboard/earnings/purchase），无 timer/polling，符合 push-first |
| **数据精度** | 全程 `decimal.Decimal`，无 float64；`StringFixed(2)` 输出 |
| **代码复用** | `createFrozenSettlementTx` 消除 3 处 SQL 重复；`subJoinOnClause` 共享 JOIN 逻辑 |
| **向前兼容** | legacy 常量保留标注；`subJoinOnClause` 匹配 pre-5.4 历史 tx |

### 修复文件

| 文件 | 修改 |
|------|------|
| `wallet_repo.go` | 幂等检查移到余额更新前（根因修复） |
| `settlement.go` | `ErrIdempotentReplay` 处理 + `createFrozenSettlementTx` helper |
| `purchase.go` | 调用 helper 替代内联 SQL |
| `bundle_purchase.go` | 调用 helper 替代内联 SQL + 删除 unused import |
| `service_subscription.go` | 调用 helper 替代内联 SQL + 修复 UUID 类型 |
| `types.go` | 分离 active/legacy 常量 + 更新注释 |
| `refund.go` | 更新函数注释 |

**最终验证**: `go build ./...` ✅ | `go test ./internal/marketplace/... ./internal/repository/... -count=1` ✅ | `gofmt` ✅

---

## 五次复查（2026-07-22 02:47）— 边界情况 + 死代码

### 🔴 P0-20 — `bundle_purchase.go` settlement `purchase_id` 用了 bundle ID 而非 subscription ID

`createFrozenSettlementTx` 传入 `bid`（bundle ID）作为 `purchase_id`，但 `subJoinOnClause` 和 `refund.go` 都通过 `marketplace_settlements.purchase_id → user_subscriptions.id` 关联。bundle ID 不是 `user_subscriptions.id`，导致：
- bundle 购买的 settlement 交易无法 JOIN 到订阅 → analytics 收入归因失败
- bundle 购买的退款无法找到 settlement → 无法退款

**修复**: 将 subscription 创建移到 settlement 之前，使用第一个 `subID` 作为 `purchase_id`。

### 🟢 P2-5 — `SettleExpiredAll` 死代码

`SettleExpiredAll` 定义但从未被调用。惰性结算只使用 per-provider `SettleExpired`。

**修复**: 删除。

### 🟢 P2-6 — `SettleExpired` 注释提及 withdrawals 但实际未接入

注释写 "triggered by provider dashboard views, withdrawals, or new purchases"，但 withdrawals 因跨服务耦合被排除。

**修复**: 更新注释为 "earnings queries, or new purchases"。

### ⚠️ 已知限制（不在本次修复范围）

- **Pre-5.4 购买退款**: 退款逻辑通过 `marketplace_settlements` 查找 settlement，pre-5.4 购买无 settlement 行 → 退款不撤销 publisher 扣款。需要单独的迁移路径。
- **Bundle 收入归因**: bundle settlement 的 `purchase_id` 指向第一个 subscription，收入归因到第一个策略。按策略拆分需要 per-strategy settlement，属于更深层的架构变更。

### ✅ 已确认无问题

- **`fee_settlement` idem_key 与 `REPLACE` 无误匹配**: `mkt-fee-settle-` 不包含 `mkt-settle-` 子串，`tx_type = 'settlement'` 过滤双重保险
- **`SettleExpired` 部分失败重试**: provider credit 成功 + platform fee 失败 → settlement 保持 frozen，重试时 provider credit 返回 replay（不修改余额），platform fee 重试
- **`LastTransactionID = &uuid.Nil`**: `SettleExpired` 忽略 `AdjustBalanceTx` 返回值，无影响
- **连接池**: 并发购买每个占 2 连接（purchase tx + settle tx），pool=25，支持 12 并发

**最终验证**: `go build ./...` ✅ | `go test ./internal/marketplace/... ./internal/repository/... -count=1` ✅ | `gofmt` ✅

---

## 六次复查（2026-07-22 03:00）— 悬空引用

### 🔴 P0-21 — `bundle_purchase.go` `firstSubID` 在 INSERT 前赋值 → 悬空 purchase_id

`firstSubID = subID` 在 `INSERT INTO user_subscriptions` 之前执行。若 INSERT 失败或冲突，`firstSubID` 指向一个不存在的 subscription ID → settlement 的 `purchase_id` 成为悬空引用 → refund 无法找到 settlement，analytics JOIN 失败。

**修复**: 将 `firstSubID` 赋值移到 `tag.RowsAffected() > 0` 确认之后。

### ✅ 最终确认

| 检查项 | 状态 |
|--------|------|
| `PurchaseStrategy` 幂等检查 | ✅ 事务内 `lookupExistingPurchase` |
| `PurchaseBundle` 幂等检查 | ✅ 事务内查 `idem_key` |
| `AdjustBalanceTx` 幂等顺序 | ✅ 先查 idem_key 再改余额 |
| `SettleExpired` replay 处理 | ✅ `ErrIdempotentReplay` 视为成功 |
| `SettleExpired` 部分失败 | ✅ provider 成功 + fee 失败 → frozen 保持，重试安全 |
| `createFrozenSettlementTx` 复用 | ✅ 3 处调用统一 |
| Bundle settlement purchase_id | ✅ 使用实际存在的 subID |
| `subJoinOnClause` | ✅ 支持 legacy + settlement idem_key |
| Analytics tx_type | ✅ 全部使用 settlement/fee_settlement |
| Analytics JOIN | ✅ 无笛卡尔积，子查询隔离 |
| `providerRev` 计算 | ✅ 直接查询，不从 GMV 推导 |
| `PendingSettlement` | ✅ proto + handler + 前端完整链路 |
| 前端 tx type 颜色 | ✅ settlement/fee_settlement/refund_reversal |
| 死代码 | ✅ `SettleExpiredAll` 已删除 |
| 遗留常量 | ✅ 标注 legacy |
| 注释准确性 | ✅ 全部更新 |

**六轮审计总计**: 21 P0 + 2 P1 + 7 P2 = 30 个问题修复

**最终验证**: `go build ./...` ✅ | `go test ./internal/marketplace/... -count=1` ✅ | `gofmt` ✅

---

## 七次复查（2026-07-22 03:08）— 实现方法最优性验证

### 🔴 P0-22 — `AdjustBalanceTx` 并发竞争未回滚余额

两层 idempotency check 之间存在窗口：两个并发调用都通过第一层 check（无历史记录），都更新了余额，但只有一个 INSERT 成功。失败方返回 `ErrIdempotentReplay`，但余额已被修改 → **双贷**。

实践中调用方的并发控制（`FOR UPDATE SKIP LOCKED`、purchase idempotency check）使此场景不可能触发，但 `AdjustBalanceTx` 作为通用基础设施应自身保证正确性。

**修复**: `ledgerChainInsert` 返回 `ErrIdempotentReplay` 时，执行 `balance = balance - amount` 回滚余额变更，然后返回当前钱包状态。

### 🟢 P2-7 — `ListProviderTransactions` 包含无效 `fee_settlement` 过滤

`fee_settlement` 交易挂在 system wallet（`user_id = SystemUserID`），永远不会出现在 provider 的交易列表中。

**修复**: 从 `tx_type IN (...)` 中移除 `fee_settlement`。

### ✅ 实现方法最优性确认

| 方法 | 最优性 | 理由 |
|------|--------|------|
| 惰性结算触发 | ✅ | 3 触发点覆盖所有用户可见路径，无 timer/polling |
| 批量结算 | ✅ | 单事务 + `FOR UPDATE SKIP LOCKED`，典型负载 1-10 条 |
| 逐条入账 | ✅ | 每条 settlement 独立 idem_key，幂等 + hash chain |
| `createFrozenSettlementTx` 复用 | ✅ | 3 处调用统一，消除 SQL 重复 |
| `subJoinOnClause` | ✅ | REPLACE 方案无 false match，支持 legacy + settlement |
| Decimal 精度 | ✅ | 全程 `decimal.Decimal`，`StringFixed(2)` |
| 幂等性 | ✅ | 三层防护：pre-check + concurrent race undo + unique index |
| 退款状态机 | ✅ | frozen→refunded / settled→reversed+refunded / refunded→reject |
| 索引 | ✅ | partial index `WHERE status='frozen'` + provider_status 复合索引 |
| 并发安全 | ✅ | `FOR UPDATE SKIP LOCKED` + per-provider 独立事务 |

**七轮审计总计**: 22 P0 + 2 P1 + 8 P2 = 32 个问题修复

**最终验证**: `go build ./...` ✅ | `go test ./internal/marketplace/... ./internal/repository/... -count=1` ✅ | `gofmt` ✅
