# 账号管理模块 — 问题分析与修复方案

## 概述

账号管理模块（block #7 `account-mgmt`）存在五类问题：**状态机残缺**、**模块边界模糊**、**设计过度复杂**、**数据越界**、**管线设计矛盾**。

---

## 问题 1：账号状态机残缺

### 现状

文档声称的状态转换中，2 个未实现、1 个实现方式错误：

| 转换 | 状态 | 实际行为 |
|------|:---:|------|
| CreateAccount → `connecting` | ✅ | `account_service.go` INSERT 时写入 |
| `connecting` → `connected` | ✅ | `runner_gateway.go` 网关连上后写入 |
| `connected` → 禁用 → `disconnected` | ❌ | 设置 `is_disabled=true`，但 `account_status` 未改变。`is_disabled` 是独立布尔字段，与状态机无关联 |
| `disconnected` → 重连 → `connected` | ✅ | `account_lifecycle.go` + NATS 触发 |
| MT 断连 → `error` | ❌ | **不存在**。`pipeline.go` 调用 `DisconnectAccountByID` 写入 `disconnected`。网关只汇报 `connected`/`reconnecting`，从未发送 `error` |
| 密码失效 → `needs_rebind` | ❌ | `MarkAccountNeedsRebind()` 方法存在但**无任何代码调用**。`VerifyTradePermission` 会检查此状态，但永不会被自动设置——死代码 |
| 冻结 → `frozen` | ✅ | `admin_account_handler.go` |
| 解冻 → `active` | ✅ | `admin_account_handler.go` |
| DeleteAccount | ✅ | 硬删除 |

### 额外问题

- `account_status` 列是 `VARCHAR(20)`，**无 CHECK 约束，无枚举类型**，任意字符串可写入
- 三个字段表达同一概念：`account_status`（6 个值，2 个死）+ `is_disabled`（布尔）+ `stream_status`（网关内部状态泄漏到业务层）

### 修复方案

**数据库层**：

1. 删除 `is_disabled` 列（其功能由 `account_status = 'disconnected'` 替代）
2. 删除 `stream_status` 列（网关内部状态不应暴露在业务表中）
3. 为 `account_status` 添加 CHECK 约束，限定合法值

```sql
-- 新 migration
ALTER TABLE mt_accounts DROP COLUMN IF EXISTS is_disabled;
ALTER TABLE mt_accounts DROP COLUMN IF EXISTS stream_status;
ALTER TABLE mt_accounts ADD CONSTRAINT chk_account_status 
  CHECK (account_status IN ('connecting', 'connected', 'disconnected', 'frozen'));
```

**代码层**：

4. 删除 `needs_rebind` 状态的设置方法和检查逻辑（死代码，无调用者，无触发路径）
5. `UpdateAccount` 中设置 `is_disabled=true` 改为 `UPDATE account_status = 'disconnected'`
6. 枚举化 `account_status`：定义 Go 常量类型 `type AccountStatus string`，替换所有字符串硬编码

```go
// backend/internal/service/account_status.go (新文件)
type AccountStatus string

const (
    StatusConnecting   AccountStatus = "connecting"
    StatusConnected    AccountStatus = "connected"
    StatusDisconnected AccountStatus = "disconnected"
    StatusFrozen       AccountStatus = "frozen"
)
```

7. `pipeline.go` 的 `OnAccountDisconnect` 回调保持写入 `disconnected`（不引入 `error` 状态——无法从网关获取，无实际业务价值）

**修改文件**：
- `backend/migrations/` — 新 migration
- `backend/internal/service/account_service.go` — UpdateAccount
- `backend/internal/service/account_lifecycle.go` — 删除 MarkAccountNeedsRebind
- `backend/internal/connect/user/account_security.go` — 删除 needs_rebind 检查
- `backend/cmd/server/pipeline.go` — OnAccountDisconnect
- `backend/internal/repository/queries/accounts.sql` — 删除 is_disabled, stream_status 引用
- `backend/internal/connect/user/account_handler.go` — DTO 去掉 is_disabled 字段

---

## 问题 2：模块边界模糊

### 边界外功能清单

账号模块当前承载了 15 项功能，其中仅 6 项属于本职：

| 功能 | 当前位置 | 应归属模块 |
|------|---------|-----------|
| MT 账号 CRUD | account_service.go | ✅ 账号模块 |
| 连接/断连/重连 | account_lifecycle.go | ✅ 账号模块 |
| 经纪商搜索 | account_connection.go | ✅ 账号模块 |
| 密码验证/修改 | account_security.go | ✅ 账号模块 |
| 实时余额/净值推送 | SSE 管道 | ✅ 账号模块 |
| 提供凭证给其他模块 | AccountService | ✅ 账号模块 |
| | | |
| AI 交易报告 | AccountReport 页面 | ❌ 策略分析模块 |
| 权益曲线图表 | EquityChart.tsx | ❌ 策略分析模块 |
| 月度盈亏分析 | MonthlyAnalysis* | ❌ 策略分析模块 |
| 品种分布分析 | AccountAnalyticsSection | ❌ 策略分析模块 |
| 每小时/每日 PnL | HourlyDailyChart | ❌ 策略分析模块 |
| 交易统计 | AccountAnalyticsSection | ❌ 策略分析模块 |
| 风险指标 | AccountAnalyticsSection | ❌ 风控模块 |
| 分享性能链接 | ShareAccountButton | ❌ 策略分析模块 |
| | | |
| 手工删 strategy_execution_logs | account_service.go | ❌ 策略模块（改用 FK CASCADE） |
| 手工删 order_history | account_service.go | ⚠️ 交易模块（改用 FK CASCADE） |
| 平台登录号生成 | account_number.go | ❌ 用户模块 |

### 修复方案

**前端：AccountDetail 精简为纯账号信息页**

AccountDetail 页面只保留：
- 账号基础信息（login, broker, server, platform, currency, leverage）
- 连接状态 + 连接/断连操作
- 实时余额/净值/保证金指标
- 删除/修改密码操作

移除到分析模块：
- 权益曲线、月度分析、品种分布、PnL 图表 → 移至策略模块的 Analytics 页面（通过 `accountId` 参数查询）
- AI 交易报告 → 删除独立路由 `/accounts/:id/report`，报告生成移至策略分析模块

**前端：Dashboard 保持**

Dashboard 的账号列表卡片 + 聚合统计是合理的入口功能，保留。

**后端：删除账号清理改为 FK CASCADE**

```sql
-- 新 migration：为缺少 CASCADE 的表补 FK
ALTER TABLE strategy_execution_logs 
  ADD CONSTRAINT fk_exec_logs_account 
  FOREIGN KEY (account_id) REFERENCES mt_accounts(id) ON DELETE CASCADE;

ALTER TABLE order_history 
  ADD CONSTRAINT fk_order_history_account 
  FOREIGN KEY (account_id) REFERENCES mt_accounts(id) ON DELETE CASCADE;

ALTER TABLE account_balance_history 
  ADD CONSTRAINT fk_balance_history_account 
  FOREIGN KEY (account_id) REFERENCES mt_accounts(id) ON DELETE CASCADE;

ALTER TABLE account_connection_logs 
  ADD CONSTRAINT fk_conn_logs_account 
  FOREIGN KEY (account_id) REFERENCES mt_accounts(id) ON DELETE CASCADE;

-- 补上完全缺失的 FK
ALTER TABLE backtest_runs 
  ADD CONSTRAINT fk_backtest_runs_account 
  FOREIGN KEY (account_id) REFERENCES mt_accounts(id) ON DELETE CASCADE;
```

然后删除 `account_service.go` 中的手工 DELETE 循环（4 条 SQL，约 12 行代码）。

**后端：account_number.go 移到 user 模块**

`account_number.go` 生成的 5-6 位号码是平台登录号，不是 MT 账号。移到 `backend/internal/service/user/` 或 `backend/internal/connect/user/`。

---

## 问题 3：设计过度复杂

### 3a. RPC 过度拆分

`ConnectAccount`、`DisconnectAccount`、`ReconnectAccount` 三个端点的实现高度重复：

```
ConnectAccount:    UPDATE status='connecting' + NATS publish
DisconnectAccount: UPDATE status='disconnected' + NATS publish
ReconnectAccount:  UPDATE status='connecting' + NATS publish
```

**修复**：合并为一个 `UpdateAccountStatus(accountId, newStatus)` RPC。

或保留现有 RPC 签名不变，内部统一委托给一个公共方法：

```go
func (s *AccountServer) updateStatus(ctx, accountID, newStatus) error {
    s.svc.SetStatus(ctx, accountID, newStatus)
    s.publisher.Publish(ctx, accountID, newStatus)
}
```

### 3b. 后端 Service 文件碎片化

```
account_service.go     — CRUD
account_lifecycle.go   — 4 个 UPDATE（连接/断连/重连/重绑）
account_snapshot.go    — 快照/缓存/汇总
account_sync.go        — 历史同步/保证金检测
account_number.go      — 生成随机号（应移到 user 模块）
```

**修复**：合并为 2 个文件：

- `account_service.go` — CRUD + 连接管理 + 状态流转（当前 service + lifecycle + snapshot 合并，约 300 行）
- `account_sync.go` — 历史同步 + 保证金检测（保留，职责独立）

`account_number.go` 移至 user 模块。

### 3c. Detail 页面组件碎片化

15+ 个组件文件，每个 50-150 行，多数只被一个父组件使用。

**修复**：按功能域合并：

```
AccountMetricsCards.tsx + AccountCard.tsx + AccountDetail.shared.tsx
  → AccountInfo.tsx

EquityChart.tsx + HourlyDailyChart.tsx + MonthlyAnalysis*.tsx
  → 移至分析模块

AccountTradeTabs.tsx + 内嵌的 Positions/Orders/History 子组件
  → AccountTrades.tsx

ShareAccountButton.tsx + AccountDeleteModal.tsx + EditAccountModal.tsx
  → AccountActions.tsx
```

综合分析组件移到新模块后，AccountDetail 自身减少到 ~5 个内部组件。

### 3d. 前端状态三层管理的简化

现状：TanStack Query + Zustand store + SSE 桥接 = 三层。

**修复**：
- 删除 `accountStore.ts`（Zustand）—— `currentAccount` 改为 URL 参数 `/accounts/:id`，`loading` 用 TanStack Query 的 `isLoading`
- 删除 `adminAccountStore.ts`（Zustand）—— 5 分钟缓存 TTL 是 TanStack Query 的 `staleTime` 配置项，不需要自己实现
- SSE 桥接保留（实时推送是 Push-First 架构要求）

### 3e. 删除 AccountReport 页面

`/accounts/:id/report` 是 AI 生成的交易报告。它不属于账号管理——它在"分析交易结果"，是策略/分析模块的职责。

**修复**：删除路由和页面组件，报告生成功能后续由分析模块提供（或不在本次修复范围内）。

---

## 问题 4：数据越界

### 4a. account_service.go 操作策略表

```go
// account_service.go 的 DeleteAccount 方法
DELETE FROM strategy_execution_logs WHERE account_id = $1
```

账号服务直接操作策略模块的表。如果策略模块改名、加分表、改分区策略，账号模块没有理由知道。

**修复**：用 FK ON DELETE CASCADE 替代（见问题 2 的 migration）。

### 4b. backtest_runs 无 FK，删账号留下孤儿行

`backtest_runs` 表有 `account_id UUID NOT NULL` 但无 FK 约束。删除账号后，`backtest_runs` 保留孤立行。

**修复**：加 FK ON DELETE CASCADE（见问题 2 的 migration）。

### 4c. AccountDTO 承载了分析指标

proto `Account` 消息 28 个字段，包含 `profit`、`profit_percent` 等计算值——这些不是 MT 连接的数据，是平台分析层的派生值。

**修复**：`Account` proto 只保留连接相关字段。分析指标通过独立的 Analytics RPC 按需查询。

---

## 执行顺序

按依赖关系排列：

```
任务 1：数据库 migration（清理列 + 加 FK 约束 + CHECK 约束）
  → 风险最低，无代码依赖

任务 2：后端状态机修正（枚举化 + 删除死代码 + 合并 UpdateAccount 逻辑）
  → 依赖任务 1 的 schema

任务 3：后端删除清理改为 FK CASCADE（删除手工 DELETE 循环）
  → 依赖任务 1 的 FK 约束

任务 4：后端 RPC 合并 + Service 文件合并 + account_number 搬迁
  → 独立任务

任务 5：前端 Detail 页面精简（分析组件移除 + Zustand 删除）
  → 独立任务

任务 6：前端路由清理（删除 AccountReport + 旧路由 redirect）
  → 依赖任务 5
```

每完成一个任务运行：
- `cd backend && go build ./...`
- `cd frontend && npx tsc --noEmit`
- `cd backend && go run ./tools/check-file-lines --strict`

---

## 验收标准

- `account_status` 有 CHECK 约束，合法值 4 个：connecting/connected/disconnected/frozen
- `is_disabled` 和 `stream_status` 列已删除
- `needs_rebind` 的 setter 和 checker 已删除
- 所有关联表有 FK ON DELETE CASCADE
- `account_service.go` 中无手工 DELETE 策略/交易表的 SQL
- `account_number.go` 已移至 user 模块
- AccountDetail 页面只展示账号信息 + 连接状态，分析图表已移除
- AccountReport 路由已删除
- accountStore.ts 和 adminAccountStore.ts 已删除
- TanStack Query 为 Account 数据的唯一状态管理
- `go build ./...` 和 `npx tsc --noEmit` 通过
