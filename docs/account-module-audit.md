# 账号管理模块 — 全线审计报告

> **审计日期**：2026-07-07
> **审计对象**：deepseek 落地施工的账号管理模块（block #7 `account-mgmt`）
> **基线**：`docs/account-module-fix.md`（2026-07-06 批准方案，11 步）
> **审计人**：Cascade
> **验证**：`go build ./...` ✅ · `npx tsc --noEmit` ✅ · `go run ./tools/check-file-lines --strict` ✅（0 errors）

---

## 0. 审计结论概要

编译/类型/文件行数三项基线全部通过，步骤 1-4、6、7/7b、9b 等**大部分改造正确落地**。
但存在 **1 个致命架构矛盾（含生产 BUG）**、**3 项零容忍合规违规**、**2 处死代码**，以及 **1 项值得单独执行的重构**。

此外，经与需求方澄清，原方案中的 **2 条判断被推翻**（见 §1），对应的 10a/11 步骤**不应执行**，deepseek 保留相关功能是正确的。

---

## 1. 已澄清 —— 判定为正确的设计决策（非缺陷）

以下两点原先被列为"未完成验收项"，经澄清确认 **deepseek 的实现才是正解**，原批准方案在这两点上判断错误。

### 1.1 DeleteAccount 连接 MT 真实验证 —— 正确

- **原方案（步骤 5）**：删除账号时改为本地校验平台登录密码（`VerifyUserPassword`），不连接 MT。
- **实际实现**：`DeleteAccount` 仍调用 `mtTester.VerifyPassword()` 连接经纪商真实验证 MT 密码。
- **结论**：**实际实现正确**。经纪商是密码的唯一真相源，本地存储的明文密码可能因用户在 MT 端改密而过期。删除是高危不可逆操作，要求真实 MT 密码验证是更严谨的安全设计。
- **处置**：撤销原步骤 5，不实现 `VerifyUserPassword`。

### 1.2 业绩分享 + AI 报告保留在账号详情页 —— 正确

- **原方案（问题 2 / 步骤 10a / 11）**：`ShareAccountButton`（业绩分享）和 `AccountReport`（AI 报告）应从账号模块移除，归入策略分析模块。
- **纠正后的原则 —— 分析必须分两个层次**：

  | 层次 | 定义 | 分析对象 | 归属 | 示例 |
  |------|------|----------|------|------|
  | **账号级** | 跨全部策略与手动交易的宏观成绩 | **整个账号** | **AccountDetail** | 净值曲线、月度盈亏、品种分布、**账号业绩分享**、**账号 AI 报告** |
  | **策略级** | 单个策略实例的表现归因 | **单个策略** | 策略模块 | 策略回测报告、策略实盘归因、单策略夏普/回撤 |

- **结论**：业绩分享与 AI 报告分析的是**整个账号**（不区分哪个策略产生的盈亏），属**账号级**，本就该留在 AccountDetail。原方案把它们归到策略模块是**层次划分不足**——混淆了"账号维度"与"策略维度"。
- **处置**：撤销步骤 10a、11。保留 `ShareAccountButton`、`AccountReport`、`/accounts/:id/report` 路由（deepseek `rd.md` "恢复 Share + AI Report" 的决策正确）。

---

## 2. 仍需修复的问题

### 2.1 【致命】软删除与 FK CASCADE 架构矛盾 + 重绑 BUG

**现象**：迁移 187 与 189 实现了两种**互斥**的删除策略，并存于同一模块。

- 迁移 187 为 `account_connection_logs / strategy_execution_logs / order_history / account_balance_history / backtest_runs` 全部添加 `ON DELETE CASCADE`（硬删除设计）。
- 迁移 189 新增 `deleted_at` 列；`account_service.go:196` 的 `DeleteAccount` 改为 `UPDATE mt_accounts SET deleted_at = NOW()`（软删除）。

**后果一（死代码）**：软删除永不触发行 DELETE，187 的全部 FK CASCADE **永不生效**，成为死设计。

**后果二（生产 BUG —— 无法重新绑定）**：
- 唯一约束 `uk_mt_account_login UNIQUE (login, mt_type, broker_server)`（迁移 115）是**完整约束**，迁移 189 **未**改为 `WHERE deleted_at IS NULL` 的部分唯一索引。
- 用户删除账号（软删，行仍占用唯一键）→ 重新绑定同一账号 → `CreateAccountTx` INSERT 命中 23505 → 返回 `ErrAccountAlreadyBound` → 前端显示 **"此交易账号已被其他用户绑定"**。
- **结果**：用户永久无法重新添加自己删除过的账号，且错误信息严重误导。

**后果三（隐私/合规）**：软删除后 `password`（明文存储）永久滞留表中。

**修复方向（二选一，需决策）**：
- **方案 A（推荐，回归硬删除）**：删除迁移 189；`DeleteAccount` 改回 `DELETE FROM mt_accounts`；移除全部 `deleted_at IS NULL` 过滤；删除 `RestoreAccount`；保留 187 的 FK CASCADE（本为此设计）。
- **方案 B（保留软删除，需完整实现）**：唯一约束改为部分索引 `WHERE deleted_at IS NULL`；删除 187 中冗余的 FK CASCADE（软删不需要）；软删时清除明文密码；`RestoreAccount` 接 RPC 暴露；补 `189_soft_delete_accounts.down.sql`。

### 2.2 【合规-零容忍】REST 端点

`cmd/server/handlers_admin.go:66`：
```go
mux.HandleFunc("/api/admin/audit-logs", adminAccountServer.ServeAuditLogs)
```
`admin_account_handler.go` 的 `ServeAuditLogs` 是裸 HTTP REST 处理器（`http.Error` / `r.URL.Query()` / `http.StatusBadRequest`）。违反平台铁律「❌ REST endpoints（除 healthz/readyz/livez/metrics）」。
（注释自认"与 /api/shares 模式一致"，提示 `/api/shares/*` 亦为既有 REST 债，另案跟踪。）

### 2.3 【合规-零容忍】JSON 序列化

`admin_account_handler.go:5` 引入 `encoding/json`；`:192` `json.NewEncoder(w).Encode(entries)`；`AuditLogEntry` 结构体带 `json:"..."` tag。违反「❌ JSON 作为数据序列化/交换格式，proto only」。

### 2.4 【合规】迁移 189 缺少 down 文件

`188_audit_log.down.sql` 存在，但 `189_soft_delete_accounts.down.sql` **缺失**。每个迁移必须成对提供 up/down。

### 2.5 【死代码】

- **`RestoreAccount`**（`account_service.go:211`）：定义了但无任何调用者（无 RPC、无接线）。
- **`UpdateTradingPassword`**（`account_security.go:45` handler + service 方法）：前端调用方 `EditAccountModal` 已删除（步骤 9b 完成），后端实现成为孤儿。原步骤 9a 要求删除后端 handler + 方法。

### 2.6 【次要】`uuid.MustParse` panic 风险

`account_crud.go:90,210` 对 `req.Msg.Id`（客户端输入）使用 `uuid.MustParse`，非法输入会 panic。当前因前置 `GetAccountCredentials` 已校验暂不触发，但应改用 `uuid.Parse` + 错误处理以消除脆弱性。

---

## 3. 审计定性：审计日志特性的定位

迁移 188（`account_audit_log`）+ `LogAudit` + `GetAuditLogs` + `ServeAuditLogs` 是原方案未包含的新增特性。
特性本身有价值（账号操作留痕），**不因超范围而否定**，但其**实现方式违规**（REST + JSON，见 §2.2/2.3）。

**处置**：保留审计日志能力，但**必须重写为 ConnectRPC + proto**：
- 在 `AdminAccountService`（或合适的 admin proto service）中新增 `GetAccountAuditLogs` RPC。
- `AuditLogEntry` 改为 proto message，去掉 `encoding/json` 与 REST 路由。
- 前端改用生成的 ConnectRPC 客户端读取。

---

## 4. 已正确完成的部分（公正记录）

- **步骤 1-3**：`AccountStatus` 枚举化；删除 `is_disabled` / `stream_status` 列；`account_status` 加 CHECK 约束（4 合法值）；删除 `needs_rebind` 死代码。
- **步骤 4**：`CreateAccount` 单次 RPC 完成绑定（前端不再调 VerifyAccount + ConnectAccount）；`BindAccount` 无独立"验证"按钮；`VerifyAccount` handler 返回 `Unimplemented`。
- **步骤 6**：盈亏推送持久化节流（`pipeline.go` `lastMetricsWrite` 5s 间隔），实时推送仍每 tick。
- **步骤 7 / 7b**：`buildOnOrderUpdate` 精简为「推持仓快照 + 平仓写 trade_records」；风控 `feedPlatformAggregator` 解耦为 `PositionSnapshotBroker` 独立消费者。
- **步骤 8**：状态变更统一推 SSE（`PublishAccountStatus`）。
- **步骤 9b**：前端 `EditAccountModal` 已删除。
- **GatewayRemover 接线**：`DeleteAccount` 中同步 `stopGateway` 防僵尸网关。

---

## 5. 推荐重构（独立于上述缺陷）

**删除冗余 Zustand store（原步骤 10b）** —— 纯技术收益，与 Share/Report 去留无关：
- `stores/accountStore.ts`：仅存瞬态 UI 状态（`currentAccount` 应由 `useParams()` 从 URL 取，账号数据已在 TanStack Query）。
- `stores/adminAccountStore.ts`：手写 5 分钟缓存 Map，应交给 TanStack Query，消除并行双重状态源。

---

## 6. 决策结论（需求方已拍板）

| 议题 | 决策 |
|------|------|
| 软删除方向 | **方案 B —— 补全软删除**（保留 `deleted_at`，修复重绑 BUG，FK CASCADE 保留作硬清除安全网） |
| 审计日志 | **重写为 ConnectRPC + proto**（移除 REST + `encoding/json`） |
| 10b Zustand store | **执行删除**（`accountStore.ts` + `adminAccountStore.ts`） |

**关键设计澄清（方案 B 下的架构自洽性）**：
软删除与 FK CASCADE **不矛盾，而是互补两层**——
- **软删除**是用户可见的删除路径：`deleted_at = NOW()`，保留行与子表数据用于审计留痕、防止误删级联丢失交易历史。
- **FK CASCADE（迁移 187 保留）**是「最终硬清除（purge）」的引用完整性安全网：当软删除行被管理员/保留策略永久清除时，子表随之级联清理。
- 因此 187 的 CASCADE **不是死代码**，而是硬清除路径的保障。二者职责分明。

**职责分工**：以下为交给 deepseek 的实现任务书；Cascade 负责设计与验收，不参与编码。

---

## 7. 实现任务书（deepseek 执行）

> 全程遵守 `AGENTS.md`：proto only（禁 JSON）、禁 REST、`decimal.Decimal`、文件行数红线、部署走 `docker compose build backend`。
> 每个子任务附验收标准，Cascade 逐条核验。

### 任务 A：补全软删除 + 修复重绑 BUG（🔴 致命）

**A1. 部分唯一索引（核心修复）** — 新建迁移 `190_mt_accounts_partial_unique.up.sql` / `.down.sql`：
```sql
-- up
ALTER TABLE mt_accounts DROP CONSTRAINT IF EXISTS uk_mt_account_login;
CREATE UNIQUE INDEX uk_mt_account_login_active
  ON mt_accounts (login, mt_type, broker_server)
  WHERE deleted_at IS NULL;
-- down
DROP INDEX IF EXISTS uk_mt_account_login_active;
ALTER TABLE mt_accounts ADD CONSTRAINT uk_mt_account_login UNIQUE (login, mt_type, broker_server);
```
> 使唯一性只约束「活跃」账号，删除后可重新绑定同一 MT 账号。

**A2. 软删除清除明文密码**（隐私）— `account_service.go:DeleteAccount` 的 UPDATE 增加 `password = ''`：
```sql
UPDATE mt_accounts SET deleted_at = NOW(), account_status = 'disconnected', password = ''
WHERE id = $1::uuid AND user_id = $2 AND deleted_at IS NULL
```

**A3. 补齐缺失的 down 文件** — 新建 `189_soft_delete_accounts.down.sql`：
```sql
DROP INDEX IF EXISTS idx_mt_accounts_deleted;
ALTER TABLE mt_accounts DROP COLUMN IF EXISTS deleted_at;
```

**A4. 移除 `RestoreAccount`** — 删除 `account_service.go:211` 的 `RestoreAccount`（无 RPC、无 UI、无需求；YAGNI）。软删除的目的是审计留痕，非用户自助恢复。

**A5. 读路径完整性核查** — 确认所有账号读取路径均带 `deleted_at IS NULL` 过滤（审计已确认：`GetAccount` / `ListAccounts` / `GetAccountCredentials` / `GetAccountSnapshots` / `UserOwnsAccount` / `UpdateAccountMetrics` / admin dashboard 计数 / positions·orders JOIN / analytics）。deepseek 补查是否有遗漏（尤其 SSE 汇总、mdgateway runner 加载账号列表），任何泄漏软删账号的查询都要补过滤。

**验收标准 A**：
- 绑定 → 删除 → 重新绑定同一 `(login, mt_type, broker_server)` 成功，不报 `ErrAccountAlreadyBound`。
- 软删后 `password` 为空串。
- 190/189 均有 up+down，`migrate down` 可回滚。
- `RestoreAccount` 无残留引用。
- 无任何查询返回 `deleted_at IS NOT NULL` 的账号。

### 任务 B：审计日志改 ConnectRPC + proto（🔴 合规）

**B1. proto 定义** — 在 admin 账号相关 proto 中新增：
```proto
message AccountAuditLogEntry {
  string id = 1; string account_id = 2; string user_id = 3;
  string action = 4; string detail = 5;
  google.protobuf.Timestamp created_at = 6;
}
message GetAccountAuditLogsRequest  { string account_id = 1; int32 limit = 2; }
message GetAccountAuditLogsResponse { repeated AccountAuditLogEntry entries = 1; }
// rpc GetAccountAuditLogs(GetAccountAuditLogsRequest) returns (GetAccountAuditLogsResponse);
```
挂到现有 admin ConnectRPC service（复用 admin 拦截器鉴权）。重新生成 proto。

**B2. 删除 REST** — 移除 `cmd/server/handlers_admin.go:66` 的 `mux.HandleFunc("/api/admin/audit-logs", ...)`；删除 `admin_account_handler.go` 的 `ServeAuditLogs` 及 `encoding/json` import。

**B3. repository 去 JSON tag** — `admin_repo.go:AuditLogEntry` 去掉 `json:"..."` tag（或改为返回 proto message / 纯 domain struct）。

**B4. 前端** — 用生成的 ConnectRPC 客户端读取审计日志，替换原 `fetch('/api/admin/audit-logs')`。

**验收标准 B**：
- 全仓 `grep -r "encoding/json" backend/internal/connect/admin` 无结果。
- 无 `/api/admin/audit-logs` REST 路由。
- 审计日志经 ConnectRPC 正常返回，admin 页面可查看。

### 任务 C：删除冗余 Zustand store（🟢 重构）

**C1.** 删除 `stores/accountStore.ts`、`stores/adminAccountStore.ts`。
**C2.** `hooks/useAccount.ts`：`currentAccount` 改由 `useParams()` 取 id 后从 TanStack Query 读取；`loading` 用 query `isLoading` / mutation `isPending`；`enablingAccount` 改用组件本地 state 或 mutation 状态。
**C3.** `pages/admin/AccountManagement.tsx`：用 `useQuery`（`staleTime` 控制缓存）替换 `useAdminAccountStore` 的手写 Map 缓存，删除 `getCachedData`/`setCachedData`/`invalidateCache` 调用。

**验收标准 C**：两个 store 文件删除，无残留 import；账号列表/详情、admin 管理页功能不回退；缓存由 TanStack Query 统一管理。

### 任务 D：清理孤儿 + 健壮性（🟡🟢）

**D1.** 删除 `UpdateTradingPassword`：后端 handler（`account_security.go:45`）+ service 方法 + proto 标记 deprecated（前端调用方 `EditAccountModal` 已删，属孤儿）。
**D2.** `account_crud.go:90,210` 的 `uuid.MustParse(req.Msg.Id)` 改为 `uuid.Parse` + 错误返回 `CodeInvalidArgument`。

**验收标准 D**：`UpdateTradingPassword` 无后端实现与调用；非法 account id 返回 `InvalidArgument` 而非 panic。

### 全局验收（deepseek 自检 + Cascade 复验）

```bash
go build ./...                                          # 通过
cd backend && go run ./tools/check-file-lines --strict  # 0 errors
npx tsc --noEmit                                        # 通过（frontend）
bash scripts/gen_capability_map.sh                      # 刷新能力图
```
外加：重绑手测（任务 A 验收）、审计日志 ConnectRPC 手测（任务 B 验收）。

---

## 附录：修复项优先级与状态汇总

| # | 问题 | 严重度 | 类型 | 决策/状态 |
|---|------|:------:|------|------|
| 2.1 | 软删除重绑 BUG + 架构自洽 | 🔴 致命 | 架构/BUG | 方案 B → 任务 A（deepseek 实现） |
| 2.2 | REST 端点 `/api/admin/audit-logs` | 🔴 合规 | 零容忍 | 改 ConnectRPC → 任务 B |
| 2.3 | `encoding/json` 序列化 | 🔴 合规 | 零容忍 | 改 proto → 任务 B |
| 2.4 | 迁移 189 缺 down.sql | 🟡 合规 | 规范 | 任务 A3 |
| 2.5a | `RestoreAccount` 死代码 | 🟡 | 死代码 | 移除 → 任务 A4 |
| 2.5b | `UpdateTradingPassword` 后端孤儿 | 🟡 | 死代码 | 移除 → 任务 D1 |
| 2.6 | `uuid.MustParse` panic 风险 | 🟢 次要 | 健壮性 | 任务 D2 |
| 5 | 删除冗余 Zustand store | 🟢 | 重构 | 执行 → 任务 C |
| 1.1 | DeleteAccount 连接 MT 验证 | ✅ | 已澄清 | 正确，无需改 |
| 1.2 | Share + AI Report 保留（账号级） | ✅ | 已澄清 | 正确，无需改 |
| 187 | FK CASCADE | ✅ | 已澄清 | 保留，作硬清除安全网（非死代码） |
