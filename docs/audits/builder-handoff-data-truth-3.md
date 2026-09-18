# Builder Handoff — DATA-TRUTH-3

> 双账户源并存命名陷阱 + `mt_accounts.last_checked_at` 死列删除。
> 设计 SSOT：Devin CLI。施工只执行、不决策。完成报证据等独立复审。

## 0. 立项证据链（设计实查已核实）

**裁定回顾**（2026-09-16 Devin CLI）：`mt_accounts`（33 列）= 运行时唯一真源；`mt_accounts_v2` 降级为凭据-only 附属——不合并。

**实查修正——v2 是 VIEW 不是表**：
- `migrations/112_brokers.up.sql:44-59` `DROP VIEW IF EXISTS mt_accounts_v2; CREATE VIEW mt_accounts_v2 AS SELECT ... FROM mt_accounts a LEFT JOIN brokers b ... WHERE NOT a.is_disabled`——15 列全为凭据/连接字段（id/user_id/platform/broker/mtapi_host/mtapi_port/login/password/mt_token/broker_host/server/is_active/canonical_subscribed_symbols/created_at/updated_at），**零写入方**（视图不可写）。
- 唯一消费方：`mdgateway/wiring.go:40`（`FROM mt_accounts_v2 WHERE is_active=true` 全量加载）、`:204`（按 id 加载）、`:248`（join 查询）——gateway runner 账号配置读取专用。
- `mdtick.go:258-270` `AccountConfig` 注释已声明来源——命名 "v2" 误导新代码误以为迁移目标（真源是 v1）。

**死列实证**：`mt_accounts.last_checked_at`——`migrations/001_init.up.sql:51` 建列后**全仓零 UPDATE/INSERT/触发器**（Go+SQL 全扫确认）；**活库实测 16 行全 NULL**。读侧仅 SELECT/scan 四处：`accounts.sql.go:15/140`（sqlc 生成，`SELECT *` 展开）、`admin_repo_accounts.go:28` SELECT+`:52/:108` scan、`model/mt_account.go:35` 字段、`admin_account_handler.go:70-71` proto 填充（恒 unset）。

**工具链**：sqlc **未安装**——生成文件头标 `sqlc v1.31.1`，须 `go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1`（精确版本，输出确定性）后 `make sqlc` 重生成。migration 机制：backend 容器启动自动执行（Dockerfile:48 COPY + constraints.md:113）；下一编号 **279**。

**REUSE/NEW**：`cap.sh` 已核——REUSE：既有 migration 编号序列+sqlc 生成链+constraints.md Data Precision 节（紧邻 TZ-PAIRED-CST-COLS-1 规则落点，同族"禁止单侧"规则并排）。NEW：无新函数/新文件（仅新 migration 文件对）。

## S1 — migration 279（新建文件对）

`backend/migrations/279_data_truth_3_cleanup.up.sql`：

```sql
-- 279_data_truth_3_cleanup.up.sql
-- DATA-TRUTH-3: drop dead column + annotate credentials-only compat view.
--
-- last_checked_at: created in 001, zero writers codebase-wide, 16/16 rows
-- NULL on live DB (2026-09-19 measured). Read plumbing removed in the same
-- commit (sqlc regen + hand-edited scan sites).
ALTER TABLE mt_accounts DROP COLUMN IF EXISTS last_checked_at;

-- mt_accounts_v2 is a credentials-only compat view for the mdgateway runner
-- config path (wiring.go) — NOT a migration target. mt_accounts (v1) is the
-- runtime source of truth. Do not extend this view or add new consumers.
COMMENT ON VIEW mt_accounts_v2 IS 'credentials-only compat view for mdgateway runner config (wiring.go); mt_accounts is the runtime truth — do not extend, do not add consumers';
```

`backend/migrations/279_data_truth_3_cleanup.down.sql`：

```sql
-- 279_data_truth_3_cleanup.down.sql
ALTER TABLE mt_accounts ADD COLUMN IF NOT EXISTS last_checked_at TIMESTAMP;
COMMENT ON VIEW mt_accounts_v2 IS NULL;
```

## S2 — sqlc 重生成（删列波及面）

1. `cd backend && go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1`（精确匹配生成头版本）
2. `make sqlc`（或 `cd backend && sqlc generate`）——`accounts.sql` 的 `SELECT *` 展开自动丢列
3. **验收点**：生成 diff 只允许移除 `last_checked_at` 相关行（`accounts.sql.go` 两 SELECT 列表+两 scan、`sqlc_models.go:33` `LastCheckedAt` 字段）；**出现任何其他 drift 必须停下报告**（版本/格式漂移不入本债）

## S3 — 手写代码删读侧（4 处）

- `internal/model/mt_account.go:35`：删 `LastCheckedAt *time.Time` 字段行
- `internal/repository/admin_repo_accounts.go:28`：SELECT 列表删 `ma.last_checked_at,`；`:52` 与 `:108` 各删 `&a.LastCheckedAt,` scan 目标
- `internal/connect/admin/admin_account_handler.go:70-71`：删 `if a.LastCheckedAt != nil { p.LastCheckedAt = timestamppb.New(...) }` 块
- **proto `admin_account.proto:59` `last_checked_at = 25` 字段保留不动**（非破坏：服务端不再填充恒 unset，前端读取语义不变；字段删除是 API break 另案）

## S4 — v2 代码注释标注

- `internal/mdgateway/wiring.go:40` 查询上行内注释：`// mt_accounts_v2: credentials-only compat view — mt_accounts is the runtime truth; do not extend (DATA-TRUTH-3)`（`:204/:248` 若同文件三处可加统一注释或仅首处——按可读性，建议三处短注释同文或首处详注）
- `internal/mdgateway/adapter/mdtick/mdtick.go:258` `AccountConfig` doc 注释扩写：`comes from PG mt_accounts_v2 view (credentials-only compat view — do not extend; mt_accounts is the runtime truth)`

## S5 — constraints.md 规则落点

`docs/constraints.md` Data Precision 节（TZ-PAIRED-CST-COLS-1 规则 bullet 之后）追加：

```md
- **账户表双源陷阱（DATA-TRUTH-3）**：`mt_accounts`（v1, 33 列）是运行时唯一真源；`mt_accounts_v2` 是 mdgateway runner 配置读取专用的**凭据-only 兼容视图**（credentials+canonical_subscribed_symbols，不可写、勿扩展、勿新增消费方）。新代码一律读写 `mt_accounts`；视图列需要新字段时扩展 `mt_accounts` 并评估视图是否真需要透出（默认不透出）。
```

## S6 — registry/STATE 同步

registry DATA-TRUTH-3 行追加施工完成记录（🟦open 施工完成待复审）；STATE.md 现状/指针同步。

## 验收门

1. `cd backend && go build ./...`（LastCheckedAt 引用全删后编译是"删干净"的编译期证明——任何残留引用即 RED）
2. `go test ./internal/repository/ ./internal/connect/admin/ ./internal/mdgateway/ ./internal/model/`（波及面包回归；若这些包测试需 DB 而无 DB 环境，如实报告跳过原因）
3. `go vet ./...` / `gofmt -l` 改动文件净 / `git diff --check`
4. `go run ./tools/check-file-lines --strict`
5. **对抗证明如实 N/A**（schema+文档债无可 pin 运行时行为）——但有两条机械验证必须做：
   - `go build` 编译绿 = 死列引用清零证明
   - `grep -rn 'last_checked_at\|LastCheckedAt' backend/internal/ backend/cmd/` 零命中（除 proto gen 产物 `last_checked_at = 25` 生成代码保留）
6. **sqlc 版本纪律**：生成头 `sqlc v1.31.1` 必须一致；生成 diff 超死列范围=阻断上报。
7. **禁止**：动 `mt_accounts_v2` 视图定义本身（列清单不动）、动 `admin_account.proto` 字段、动 wiring.go 查询语义、跑任何 DB 写操作、部署。

## 范围边界

只改：`migrations/279_*` 新文件对、`internal/repository/`（sqlc 生成物+admin_repo_accounts.go）、`internal/model/mt_account.go`、`internal/connect/admin/admin_account_handler.go`、`internal/mdgateway/wiring.go`、`internal/mdgateway/adapter/mdtick/mdtick.go`、`docs/constraints.md`、registry/STATE。勿部署勿 push。
