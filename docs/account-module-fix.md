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

## 问题 5：管线设计矛盾

对账号模块全部 6 条数据管线进行第一性审计，每条管线均存在设计矛盾。

### 管线 1：账号绑定（BindAccount）

**当前路径**（4 个 RPC，2 次 MT 连接，3 次重复 DB UPDATE，2 次重复 NATS 发布）：

```
前端 Step 1: SearchBroker RPC
前端 Step 3: 用户点"验证" → VerifyAccount RPC → mtTester.Test()（不保存结果）
前端 Step 3: 用户点"绑定" → CreateAccount RPC
  → INSERT (status='connecting')           // 第 1 次 status='connecting'
  → mtTester.Test()（再次连接 MT）           // 与 VerifyAccount 重复
  → UpdateAccountInfoTx                    // 第 2 次 status='connecting'（重复）
  → PublishConnect NATS                    // 第 1 次 NATS
前端: accountApi.connect(id) → ConnectAccount RPC
  → ConnectAccount                         // 第 3 次 status='connecting'（重复）
  → PublishConnect NATS                    // 第 2 次 NATS（重复）
```

**矛盾**：
- `VerifyAccount` 和 `CreateAccount` 各自连接 MT 一次——`mtTester.Test()` 调用两次
- `account_status='connecting'` 被写入 3 次（INSERT + CreateAccount 末尾 + ConnectAccount）
- NATS `account.connect.<id>` 事件被发布 2 次，mdgateway subscriber 收到两次，尝试启动网关两次
- 4 个 RPC 中，`VerifyAccount` 和 `ConnectAccount` 是冗余的——CreateAccount 已经做完了验证和连接

**第一性原则**：绑定账号 = 输入凭证 + 保存 + 启动连接。应该是一次 RPC + 一次 MT 验证 + 一次 NATS 发布。

**修复方案**：

1. **后端合并**：`CreateAccount` RPC 内部做完所有事——验证 MT 凭证、INSERT、发布 NATS 事件。不再需要前端额外调用 `VerifyAccount` 和 `ConnectAccount`

2. **前端简化**：BindAccount wizard 去掉"验证"按钮。Step 3 只有一个"绑定"按钮，调 `CreateAccount` RPC。MT 验证结果由 CreateAccount 返回（成功→跳转详情页，失败→显示错误）

3. **消除重复 NATS**：`CreateAccount` handler 末尾去掉 `s.svc.ConnectAccount()` + `s.publisher.PublishConnect()` 调用。改为在 `verifyAndUpdateAccount` 成功后，**仅发一次** NATS 事件。前端不再调用 `ConnectAccount` RPC

```
修复后:
  用户填凭证 → 点"绑定" → CreateAccount RPC
    → INSERT (status='connecting')
    → mtTester.Test()（一次 MT 连接）
    → UpdateAccountInfoTx
    → PublishConnect NATS（一次）
    → 返回 account + 验证结果
    → 前端跳转 /accounts/:id
```

**RPC 调用**：4 → **1**
**MT 连接**：2 → **1**
**DB 写入**：3 → **2**（INSERT + UpdateAccountInfoTx）
**NATS 发布**：2 → **1**

---

### 管线 2：账号删除（DeleteAccount）

**当前路径**：

```
前端: 用户输入 MT 密码 → DeleteAccount RPC
  → GetAccountCredentials（查本地 DB 取密码）
  → mtTester.VerifyPassword()（连接 MT 服务器验证密码）
    → 如果 MT 服务器不可达 → 删除失败，账号永久不可删
  → DeleteAccount()（5 条手工 DELETE）
```

**矛盾**：删除是纯本地数据操作，但被外部 MT 连接的可用性阻止。用户丢了 MT 密码或 MT 服务器维护时，账号变成平台上的僵尸——不可用、不可删。

**第一性原则**：删除平台账号不应依赖外部系统。用户在平台侧确认操作意图即可。

**修复方案**：

1. **验证方式改为本地**：删除时不需要连接 MT。改为要求用户输入平台登录密码（或用二次确认对话框 "输入 DELETE 确认删除"），验证的是用户对平台账号的所有权，而非 MT 凭证

2. **删除前断开网关**：先发 NATS `account.disconnect.<id>` 事件让 gateway 断开 MT 连接，再删 DB 记录

```go
// 修复后的 DeleteAccount handler
func (s *AccountServer) DeleteAccount(ctx, req) {
    // 1. 验证平台用户身份（不连 MT）
    s.svc.VerifyPlatformPassword(ctx, userID, req.Password)
    
    // 2. 断开 MT 连接
    s.publisher.PublishDisconnect(ctx, accountID)
    
    // 3. 删除（FK CASCADE 自动清理关联表）
    s.svc.DeleteAccount(ctx, accountID)
}
```

---

### 管线 3：实时盈亏推送（Profit/Equity Updates）

**当前路径**（5 跳，每 tick DB 写入）：

```
MT5 → gRPC stream → MT5 gateway adapter → profitRecvLoop
  → OnAccountProfit 回调
    ├── UpdateAccountMetrics() → DB WRITE（每 tick）
    ├── RecordBalanceSnapshot() → DB WRITE（每小时，合理）
    ├── PublishAccountProfit() → AccountProfitBroker（内存 channel）
    ├── UpdateSummaryCache() → 内存缓存
    └── 保证金检测 → DB read

  → AccountProfitBroker → StreamServer.subscribeEvents
    → SSE server-stream → 前端 TanStack Query

同时：useAccountFinancials 有 staleTime=60000
  → SSE 静默 60 秒后 → RPC accountApi.get() 拉取 DB 数据
```

**矛盾**：
1. **每 tick 写 DB**：实时展示路径（gateway → 内存 broker → SSE → 前端）不需要 DB。DB 写入只服务于持久化和首次加载。但当前每 tick 都在写
2. **重叠写入**：`OnAccountProfit` 和 `OnOrderUpdate` 回调各自调用 `UpdateAccountMetrics()`。当一次交易成交时，两个回调先后触发，DB 写入两次相同指标
3. **双路冗余**：SSE 推送 + RPC 轮询（staleTime 60s）提供同一条数据。RPC 作为 fallback 合理，但两条路径没有明确的主备关系

**第一性原则**：实时数据走内存管道，持久化走节流写入。两条路径各司其职，不互相竞争。

**修复方案**：

1. **节流 DB 写入**：`UpdateAccountMetrics()` 改为**最多每 5 秒写一次**（用内存时间戳去重）。去掉实时展示路径中的同步 DB 写入

```go
// pipeline.go OnAccountProfit 回调
func onAccountProfit(profit *mdtick.ProfitUpdate) {
    // 实时推送：内存路径，无 DB
    mthubSvc.PublishAccountProfit(profit)
    
    // 持久化：节流写入（5 秒间隔）
    if time.Since(lastDBWrite) > 5*time.Second {
        accountSvc.UpdateAccountMetrics(ctx, profit)
        accountSvc.RecordBalanceSnapshot(ctx, profit) // 保持每小时
        lastDBWrite = time.Now()
    }
}
```

2. **去重 OnAccountProfit / OnOrderUpdate 的写入**：两个回调共享同一个节流时间戳，避免同一秒内重复写入

3. **明确定义双路关系**：SSE 为主路径（实时），RPC 为 fallback（首次加载 + SSE 断连后恢复）。改 `staleTime: 60000` 为 `staleTime: Infinity`（首次加载后完全依赖 SSE），在 SSE 断开事件中触发 `invalidateQueries` 切回 RPC

---

### 管线 4：订单事件流（Order Events）

**当前路径**：

```
MT5 → orderUpdateRecvLoop → buildOnOrderUpdate 回调
  ├── UpdateAccountMetrics() → DB WRITE
  ├── publishProfitEvent() → AccountProfitBroker
  ├── UpdateSummaryCache()
  ├── publishPositionSnapshot() → PositionSnapshotBroker → SSE → 前端
  ├── feedPlatformAggregator() → 风控计算
  └── writeClosedTradeRecord() → DB WRITE（仅平仓时）
```

**矛盾**：
1. **`UpdateAccountMetrics()` 冗余**：订单更新回调写入盈亏指标，但盈亏数据也通过 `OnAccountProfit` 推过来了。同一笔成交触发两条写入
2. 绕过 OMS 对只读展示是正确的（OMS 是下单路径）
3. 风控计算（`feedPlatformAggregator`）嵌入展示路径中——展示和风控不应耦合

**第一性原则**：订单展示管道只做一件事——把 MT5 的持仓数据推到前端。不应附带 DB 写入和风控计算。

**修复方案**：

1. **去掉 `UpdateAccountMetrics()` 调用**：`buildOnOrderUpdate` 不再写 `mt_accounts` 的余额/净值字段。这些数据由管线 3（盈亏推送）独立负责

2. **风控计算移到独立路径**：`feedPlatformAggregator()` 不嵌入 `buildOnOrderUpdate`，改为监听 `PositionSnapshotBroker` 的独立消费者

```
修复后:
MT5 → orderUpdateRecvLoop → buildOnOrderUpdate
  ├── publishPositionSnapshot() → PositionSnapshotBroker
  │     ├── SSE → 前端持仓展示
  │     └── feedPlatformAggregator() → 风控计算（独立消费者）
  └── writeClosedTradeRecord() → DB WRITE（仅平仓时）
```

---

### 管线 5：状态通知（Account Status）

**当前路径**：

```
Gateway 层:
  MT4/MT5 各流建立 → reportStatus("connected")
  流断开重连中   → reportStatus("reconnecting")

OnAccountStatus 回调 (pipeline.go):
  DB UPDATE account_status = <reported_status>
  SSE → 前端

独立设置（不经过 SSE）:
  CreateAccountTx → 'connecting'
  ConnectAccount → 'connecting'
  DisconnectAccount → 'disconnected'
  FreezeAccount → 'frozen'
  UnfreezeAccount → 'active'
```

**矛盾**：
1. **三方不一致**：网关能产生的状态（`connected`/`reconnecting`）≠ DB 定义的状态（6 个）≠ 前端认识的状态。前端 `AccountDetail` 的 status display 缺少 `reconnecting` 分支 → 显示 "unknown"
2. **状态变化不通知**：`connecting`/`disconnected`/`frozen` 由 RPC handler 直接写 DB，不经过 SSE 推送。前端只能通过 TanStack Query 的 RPC fallback 感知，有延迟
3. **`reconnecting` 语义歧义**：三股流（报价/盈亏/订单）各自独立汇报。可能出现报价流在 `connected` 但订单流在 `reconnecting` 的情况——但 `account_status` 是单值字段，无法表达部分连接

**第一性原则**：状态变化必须推送到前端。状态语义必须精确——单值字段不能表达多流的部分连接。

**修复方案**：

1. **统一状态推送**：所有状态变化（无论来自 RPC handler 还是 gateway 回调）都发布到同一个 SSE channel。Handler 写完 DB 后调用 `mthubSvc.PublishAccountStatus()`

2. **前端补全**：`AccountDetail` 的状态显示增加 `reconnecting` 分支（黄色标签 + "重连中"）。或在问题 1 的状态收敛后，此分支不再需要

3. **简化网关汇报**：网关不再汇报 `reconnecting` 状态——它是瞬态且部分性的。网关只在稳定连接时汇报 `connected`，断开时通过 `OnAccountDisconnect` 回调处理（已经是 `disconnected`）

4. **状态枚举统一**：与问题 1 的 `AccountStatus` 枚举保持一致，4 个值：`connecting`/`connected`/`disconnected`/`frozen`

```
修复后:
  所有状态变更 → UPDATE mt_accounts SET account_status = ?
    → mthubSvc.PublishAccountStatus() → SSE → 前端

  网关层职责：
    流全部建立 → 不单独汇报（startGatewayForAccount 直接写 'connected'）
    流断开     → OnAccountDisconnect → 'disconnected'
    （删除 reportStatus 的 'reconnecting' 汇报）
```

---

### 管线 6：密码修改（UpdateTradingPassword）

**当前路径**：

```
前端 EditAccountModal:
  用户输入旧密码 + 新密码 → UpdateTradingPassword RPC

后端 account_security.go:
  UpdateTradingPassword(ctx, id, oldPassword, newPassword)
    → SQL: UPDATE mt_accounts SET password = $4
            WHERE id = $1 AND password = $3
    → 只写本地 DB，不连 MT broker
```

**矛盾**：
1. **UI 误导**：按钮标签为 "Update Trading Password"，用户以为改了 MT 经纪商密码，实际只改了平台本地存储的副本
2. **旧密码验证用本地 DB 值**：如果用户在 MT 端改过密码，本地 DB 存储的旧密码已过时。用户输入正确的 MT 旧密码会被 `WHERE password = $3` 拒绝
3. **外部变更不可检测**：用户在经纪商网站改密码后，平台不知道。网关最终会认证失败，但表面为 `reconnecting`，无"密码错误"提示

**第一性原则**：如果平台不负责修改 MT 密码，就不要提供这个功能。只做 "存储密码供网关连接使用"，不应包装为 "修改密码"。

**修复方案**：

**删除此功能**。平台不应充当 MT 密码管理器。用户改密码应该去经纪商网站。平台只负责在绑定（CreateAccount）时存储密码。

- 删除 `UpdateTradingPassword` RPC
- 删除 `EditAccountModal.tsx`
- `BindAccount` wizard 增加说明："密码仅用于平台连接 MT，不会同步修改。如需改密码，请在经纪商平台操作后重新绑定本平台账号"

如果 mtapi 后续提供密码修改接口，可以重新实现完整功能。

---

### 管线问题汇总

| 管线 | RPC 调用 | DB 写入 | MT 连接 | 核心矛盾 |
|------|:---:|:---:|:---:|------|
| 1. 绑定 | 4→**1** | 3→**2** | 2→**1** | 重复的验证 + 重复的 NATS |
| 2. 删除 | 1→**1** | 5→**0**（FK自动） | 1→**0** | 本地操作依赖外部系统 |
| 3. 盈亏推送 | 0 | 每 tick→**节流5s** | 0 | 实时路径不需要每 tick 写 DB |
| 4. 订单事件 | 0 | 每事件→**仅平仓** | 0 | 展示路径附带 DB 写入和风控 |
| 5. 状态通知 | 0 | 1→**1** | 0 | 三方状态不一致 |
| 6. 密码修改 | 1→**0** | 1→**0** | 0→**0** | 功能本不应存在 |

---

## 问题 6：灰色地带裁决

修复后仍有两处设计决策需要定案。

### 6a. 持仓/订单展示

**问题**：AccountDetail 展示的持仓/挂单/历史订单，属于账号模块还是交易模块？

**裁决**：**留在账号模块。** 不跨账号聚合。理由：

- 数据源是 MT5 网关的单账号流（`OrderUpdateRecvLoop`），天然按 accountId 隔离
- 用户场景是"看我的这个 MT 账号上有什么持仓"，这是账号视图，不是交易面板
- 不需要跨账号聚合，也就不需要交易模块的统一持仓层

### 6b. 保证金追缴检测

**问题**：`account_sync.go` 的 `CheckMarginCall` 属于账号模块还是风控模块？

**裁决**：**检测留在账号模块，响应交给风控模块。** 理由：

- 检测所需数据全部在账号模块内部（实时余额来自管线 3、持仓浮亏来自管线 4、经纪商阈值在 `mt_accounts` 表、连接状态是自己的字段）
- 如果拆到风控模块，风控需要订阅 AccountProfitBroker + PositionSnapshotBroker + 查 DB 阈值 + 查连接状态——凭空增加 4 个跨模块依赖，只为实现已存在的功能
- 正确模式是**检测 + 发布事件，响应 + 订阅事件**：

```
账号模块（检测）:
  CheckMarginCall(balance, margin, positions, thresholds)
    → 触发 → margin_warning / margin_call 事件

风控模块（响应）:
  订阅 margin_call 事件
    → 关仓 / 限单 / 通知用户
```

**不写代码，不引入新的事件系统**。本次修复只做职责划分，代码位置保持不变。如果未来风控模块需要消费保证金事件，再加事件发布。

---

## 实现步骤（GLM 照此执行，不自行设计）

每步完成后运行 `go build ./...` 或 `npx tsc --noEmit`。

---

### 步骤 1：数据库 migration

**文件**：`backend/migrations/` 新建 `XXX_cleanup_account_schema.up.sql`

```sql
-- 1. 删除冗余列
ALTER TABLE mt_accounts DROP COLUMN IF EXISTS is_disabled;
ALTER TABLE mt_accounts DROP COLUMN IF EXISTS stream_status;

-- 2. 添加 CHECK 约束
ALTER TABLE mt_accounts ADD CONSTRAINT chk_account_status 
  CHECK (account_status IN ('connecting', 'connected', 'disconnected', 'frozen'));

-- 3. 补 FK 约束（缺了 ON DELETE CASCADE 的表）
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

-- 4. 补完全缺失的 FK
ALTER TABLE backtest_runs 
  ADD CONSTRAINT fk_backtest_runs_account 
  FOREIGN KEY (account_id) REFERENCES mt_accounts(id) ON DELETE CASCADE;
```

同时创建 `XXX_cleanup_account_schema.down.sql` 做反向操作。

---

### 步骤 2：后端状态枚举化 + 死代码删除

**2a. 新建文件 `backend/internal/service/account_status.go`**：

```go
package service

type AccountStatus string

const (
    StatusConnecting   AccountStatus = "connecting"
    StatusConnected    AccountStatus = "connected"
    StatusDisconnected AccountStatus = "disconnected"
    StatusFrozen       AccountStatus = "frozen"
)

func (s AccountStatus) String() string { return string(s) }
```

**2b. 修改 `backend/internal/service/account_lifecycle.go`**：
- `ConnectAccount` / `DisconnectAccount` / `ReconnectAccount` 的参数和 SQL 中用 `AccountStatus` 替换字符串 `"connecting"` / `"disconnected"`
- **删除** `MarkAccountNeedsRebind` 方法（整段删除）
- **删除** `DisconnectAccountByID` 中对 `needs_rebind` 的任何引用

**2c. 修改 `backend/internal/service/account_service.go`**：
- `CreateAccountTx` 的 SQL 中 `'connecting'` 替换为 `string(StatusConnecting)`
- `UpdateAccount` 方法中，任何设置 `is_disabled = true` 的地方改为 `account_status = string(StatusDisconnected)`。**删除** SQL 中的 `is_disabled = COALESCE($6, is_disabled)` 片段
- `DeleteAccount` 方法中，**删除**下面 4 行手工 DELETE（FK CASCADE 在步骤 1 的 migration 中已加）：
  ```go
  DELETE FROM account_balance_history WHERE account_id = $1
  DELETE FROM account_connection_logs WHERE account_id = $1
  DELETE FROM strategy_execution_logs WHERE account_id = $1
  DELETE FROM order_history WHERE account_id = $1
  ```
  删除 `related` 数组和相关循环，只保留 `DELETE FROM mt_accounts WHERE id = $1`

**2d. 修改 `backend/internal/connect/user/account_security.go`**：
- `VerifyTradePermission` 方法中，删除对 `needs_rebind` 的检查。只保留对 `frozen` 的检查

**2e. 修改 `backend/internal/connect/user/account_crud.go`**：
- `UpdateAccount` handler：去掉 `is_disabled` 字段映射。如果请求里传了 `is_disabled=true`，改为 `svc.DisconnectAccount`

**2f. 修改 `backend/internal/connect/user/account_handler.go`**：
- `AccountDTO` 和 proto `Account` 的转换中，去掉 `is_disabled` 字段

**2g. 修改 `backend/internal/repository/queries/accounts.sql`**：
- `UpdateAccountMetrics` 查询中删除 `is_disabled` 和 `stream_status` 引用
- `ListAccounts` / `GetAccount` 查询中删除 `is_disabled` 和 `stream_status` 列

**2h. 重新生成 sqlc**：
```bash
cd backend && sqlc generate
```

---

### 步骤 3：RPC 合并 + Service 文件合并

**3a. 合并 Connect/Disconnect/Reconnect RPC**：

在 `backend/internal/connect/user/account_connection.go` 中，保留 3 个 handler 函数签名不变（proto 不变），但内部实现统一委托：

```go
func (s *AccountServer) ConnectAccount(...) {
    return s.updateAccountStatus(ctx, req.Msg.Id, service.StatusConnecting)
}
func (s *AccountServer) DisconnectAccount(...) {
    return s.updateAccountStatus(ctx, req.Msg.Id, service.StatusDisconnected)
}
func (s *AccountServer) ReconnectAccount(...) {
    return s.updateAccountStatus(ctx, req.Msg.Id, service.StatusConnecting)
}

func (s *AccountServer) updateAccountStatus(ctx, id, status) {
    s.svc.SetStatus(ctx, id, status)
    s.publisher.Publish(ctx, id, status)
}
```

`AccountService` 中新增 `SetStatus(ctx, id, status AccountStatus) error` 方法（一个 SQL UPDATE）。

**3b. Service 文件合并**：

将以下文件的内容合并到 `account_service.go`：
- `account_lifecycle.go`（4 个方法：Connect/Disconnect/Reconnect/SetStatus）
- `account_snapshot.go`（快照/缓存/汇总）

合并后 `account_service.go` 约 300 行。

删除文件：
- `account_lifecycle.go`
- `account_snapshot.go`

保留文件：
- `account_service.go` — CRUD + 连接管理 + 状态流转 + 快照/缓存
- `account_sync.go` — 历史同步 + 保证金检测（独立职责，保留）

**3c. account_number.go 搬迁**：

- 将 `backend/internal/service/account_number.go` 移动到 `backend/internal/service/user/account_number.go`
- 更新所有 import 路径
- 这个文件的唯一调用方在 `RegistrationService` 中——确认路径正确

---

### 步骤 4：管线修复 — 绑定路径

**4a. 后端合并 CreateAccount**：

修改 `backend/internal/connect/user/account_crud.go` 的 `CreateAccount` handler：

```go
func (s *AccountServer) CreateAccount(ctx, req) {
    // 1. INSERT (status='connecting')
    tx, account, _ := s.svc.CreateAccountTx(ctx, ...)
    
    // 2. 验证 MT 凭证（一次 MT 连接）
    info, err := s.mtTester.Test(ctx, platform, brokerHost, login, password)
    if err != nil {
        tx.Rollback()
        return error("MT 验证失败: " + err)
    }
    
    // 3. 更新账号信息
    s.svc.UpdateAccountInfoTx(ctx, tx, account.ID, info)
    tx.Commit()
    
    // 4. 发布 NATS 事件（仅一次）
    s.publisher.PublishConnect(ctx, account.ID)
    
    // 5. 返回
    return s.svc.GetAccount(ctx, account.ID)
}
```

删除 handler 末尾的 `s.svc.ConnectAccount()` 调用和重复的 NATS 发布。前端不再调 `ConnectAccount` RPC 时，这已经是唯一的 NATS 发布点。

**4b. 前端简化 BindAccount**：

修改 `frontend/src/pages/accounts/BindAccount.tsx`：
- Step 3 中**删除** "验证" 按钮及其 `handleVerify` 逻辑
- Step 3 只保留 "绑定" 按钮，调用 `accountApi.create(data)`
- 调用链中**删除** `accountApi.connect(account.id)` ——CreateAccount 后端已处理连接
- 如果 CreateAccount 返回错误（MT 验证失败），在 UI 显示错误信息

**4c. 删除 VerifyAccount RPC**：

确认 `accountApi.verifyAccount()` 只在 `BindAccount.tsx` 中使用（步骤 4b 已删除调用方）。删除：
- `backend/internal/connect/user/account_security.go` 中的 `VerifyAccount` handler
- 前端 `frontend/src/client/` 中的 `verifyAccount` 方法
- Proto 中标记 `VerifyAccount` RPC 为 deprecated（不删除，保留 proto 兼容性）

---

### 步骤 5：管线修复 — 删除路径

修改 `backend/internal/connect/user/account_crud.go` 的 `DeleteAccount` handler：

```go
func (s *AccountServer) DeleteAccount(ctx, req) {
    // 1. 验证平台用户密码（不连接 MT，本地验证）
    if err := s.svc.VerifyUserPassword(ctx, userID, req.Password); err != nil {
        return error("密码错误")
    }
    
    // 2. 先断开 MT 连接
    s.publisher.PublishDisconnect(ctx, req.Msg.Id)
    
    // 3. 删除（FK CASCADE 自动清理关联表）
    s.svc.DeleteAccount(ctx, req.Msg.Id)
}
```

**删除** `mtTester.VerifyPassword()` 调用。`req.Password` 改为平台登录密码，不是 MT 密码。

如果 `AccountService` 没有 `VerifyUserPassword` 方法，在 `account_service.go` 中添加：
```go
func (s *AccountService) VerifyUserPassword(ctx, userID uuid.UUID, password string) error {
    // SELECT password_hash FROM users WHERE id = $1
    // 比较 hash
}
```

---

### 步骤 6：管线修复 — 盈亏推送节流

修改 `backend/cmd/server/pipeline.go` 的 `OnAccountProfit` 回调：

```go
// 在 pipeline 结构体中添加字段
type accountPipeline struct {
    lastMetricsWrite time.Time  // 新增
    // ... 已有字段
}

func (p *accountPipeline) OnAccountProfit(profit *mdtick.ProfitUpdate) {
    // 实时推送：始终走内存管道
    p.mthubSvc.PublishAccountProfit(profit)
    p.accountSvc.UpdateSummaryCache(profit)
    
    // 持久化：节流写入（5 秒间隔）
    if time.Since(p.lastMetricsWrite) > 5*time.Second {
        p.accountSvc.UpdateAccountMetrics(ctx, profit)
        p.accountSvc.RecordBalanceSnapshot(ctx, profit)
        p.lastMetricsWrite = time.Now()
    }
}
```

---

### 步骤 7：管线修复 — 订单事件去重

修改 `backend/cmd/server/pipeline_callbacks.go` 的 `buildOnOrderUpdate` 回调：

- **删除** `accountSvc.UpdateAccountMetrics()` 调用（盈亏推送管线 6 独立负责）
- **删除** `publishProfitEvent()` 调用（同上）
- **删除** `accountSvc.UpdateSummaryCache()` 调用（同上）
- **删除** `feedPlatformAggregator()` 调用（风控不应嵌入展示路径）
修改后 `buildOnOrderUpdate` 只做两件事：
```go
func buildOnOrderUpdate(update) {
    // 1. 推送持仓快照到前端（只读展示）
    publishPositionSnapshot(update)
    // 2. 平仓时写 trade_records
    if update.IsClose {
        writeClosedTradeRecord(update)
    }
}
```

---

### 步骤 7b：解耦风控计算

`feedPlatformAggregator` 从订单更新回调中移出，改为 `PositionSnapshotBroker` 的独立消费者：

修改 `backend/cmd/server/pipeline.go`，在初始化 `PositionSnapshotBroker` 的地方，新增一个独立 goroutine 订阅 broker 并调用 `feedPlatformAggregator`：

```go
go func() {
    for snap := range positionSnapshotBroker.Subscribe() {
        feedPlatformAggregator(snap)
    }
}()
```

这样风控计算和前端展示完全解耦——展示路径不管风控，风控不阻塞展示。

---

### 步骤 7c：前端 SSE/RPC 双路关系明确化

修改 `frontend/src/queries/useAccountFinancials.ts`：

```typescript
// 改前
staleTime: 60000  // 60 秒后 RPC 拉取

// 改后
staleTime: Infinity  // 首次加载后完全依赖 SSE

// SSE 断开时切回 RPC
// 在 SSE 断连回调中：
queryClient.invalidateQueries({ queryKey: queryKeys.accounts.financials })
```

同时修改 `frontend/src/hooks/useWatchBacktestRun.ts` 或 SSE 连接管理代码，在 SSE `onDisconnect` 事件中调用 `invalidateQueries`。

---

### 步骤 8：管线修复 — 状态通知统一

**8a. 所有状态变更推 SSE**：

找到所有直接写 `account_status` 的代码点，确保写完后调用 `mthubSvc.PublishAccountStatus()`：

| 代码位置 | 状态值 | 当前是否推送 SSE |
|---------|--------|:---:|
| `CreateAccountTx` | `connecting` | ❌ → **加** |
| `ConnectAccount` | `connecting` | ❌ → **加** |
| `DisconnectAccount` | `disconnected` | ❌ → **加** |
| `startGatewayForAccount` | `connected` | ✅ 已有 |
| `OnAccountStatus` 回调 | `reconnecting` | ✅ 已有（但删掉 `reconnecting` 汇报后不再触发） |
| admin `FreezeAccount` | `frozen` | ❌ → **加** |
| admin `UnfreezeAccount` | `active` | ❌ → **加**（改为 `connected` 如果当时连着） |

**8b. 删除网关的 `reconnecting` 汇报**：
- 在 `mt4/connection_account.go` 和 `mt5/connection_account.go` 的 `reportStatus` 方法中，去掉 `"reconnecting"` 分支
- 流断开直接走 `OnAccountDisconnect` 回调（→ `disconnected`）
- 流恢复不单独汇报（`startGatewayForAccount` 已经处理）

---

### 步骤 9：删除密码修改功能

**9a. 后端**：
- 删除 `backend/internal/connect/user/account_security.go` 中的 `UpdateTradingPassword` handler
- 删除 `backend/internal/service/account_lifecycle.go` 中的 `UpdateTradingPassword` 方法
- 从 proto `AccountService` 中移除 `UpdateTradingPassword` RPC（或标记 `deprecated`）

**9b. 前端**：
- 删除 `frontend/src/pages/accounts/components/EditAccountModal.tsx`
- AccountDetail 页面中删除 "修改密码" 按钮
- BindAccount wizard 增加提示文字（i18n key `accounts.bind.passwordNote`）："密码仅用于平台连接 MT，不会同步修改。如需修改 MT 密码，请在经纪商平台操作后重新绑定"

---

### 步骤 10：前端 Detail 页面精简

**10a. 分析组件移除**：

从 `AccountDetail.tsx` 中删除以下导入和渲染：
- `AccountAnalyticsSection`（权益曲线/月度分析/品种分布/PnL 图表）
- `ShareAccountButton`（性能分享）
- `AccountReport` 引用

保留：
- `AccountMetricsCards`（余额/净值/保证金）
- `AccountTradeTabs`（持仓/挂单/历史）
- `AccountCard`（基础信息）
- `AccountDeleteModal`（删除确认，改用平台密码验证）

**10b. Zustand 删除**：
- 删除 `frontend/src/stores/accountStore.ts`
- 删除 `frontend/src/stores/adminAccountStore.ts`
- `useAccount` hook 中，`currentAccount` 改为从 URL 参数 `useParams()` 获取

**10c. 组件合并**：
- `AccountMetricsCards.tsx` + `AccountCard.tsx` + `AccountDetail.shared.tsx` → 合并为 `AccountInfo.tsx`（一个文件，约 120 行）
- `AccountTradeTabs.tsx` + 内嵌子组件 → 保持或合并为 `AccountTrades.tsx`

---

### 步骤 11：前端路由清理

**11a. 删除路由**：
- 删除 `/accounts/:id/report` 路由和 `AccountReport.tsx`
- 旧 URL redirect：`/accounts/:id/report` → `/accounts/:id`

**11b. 更新导航**：
- `MainLayout.tsx` 侧边栏无需改动（账号本身不在侧边栏）
- Dashboard 的账号卡片链接保持指向 `/accounts/:id`

---

## 验收标准

- `go build ./...` 和 `npx tsc --noEmit` 通过
- `cd backend && go run ./tools/check-file-lines --strict` 通过
- `account_status` 有 CHECK 约束，合法值 4 个
- `is_disabled` 和 `stream_status` 列已删除
- 所有关联表有 FK ON DELETE CASCADE（包括 `backtest_runs`）
- `account_service.go` 中无手工 DELETE 关联表的 SQL
- `needs_rebind` 的 setter / checker 已删除
- `UpdateTradingPassword` RPC + `EditAccountModal` 已删除
- CreateAccount 一次 RPC 完成绑定（前端不调 VerifyAccount + ConnectAccount）
- DeleteAccount 不连接 MT
- 盈亏推送每 tick 不写 DB（改为节流 5 秒）
- 订单更新回调不调用 `UpdateAccountMetrics`
- BindAccount 无单独"验证"按钮
- AccountDetail 无分析图表、无分享按钮
- accountStore.ts / adminAccountStore.ts 已删除
- account_number.go 已移至 user 模块
