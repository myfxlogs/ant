# Builder Handoff: TZ-PAIRED-CST-COLS-1 — CST 配对列规则文档化

> **角色**：施工方。**任务 ID**：`TZ-PAIRED-CST-COLS-1`（P3 文档债，TZ-MIXED-ENCODING-1 验收分立）。
> **性质**：**文档债**——不改任何写读行为，把"CST 配对列禁单侧翻 UTC"规则落到权威文档面 + 写入点行内注释。零行为变更。
> **设计实查**：Devin CLI 已完成，四列配对全部坐标核实 + DB `NOW()`=UTC 活库实测。

## 立项背景与实测结论

部署形态（实测）：backend `TZ=Asia/Shanghai`→`time.Now()`=CST 钟面；postgres 容器无 TZ→`NOW()`=UTC（活库 `SHOW timezone`=UTC 实测）。以下 `timestamp`（非 timestamptz）列当前 **CST 写入 + CST 参数读取**自洽配对：

| 列 | 写侧（CST） | 读侧（CST 参数） | 备注 |
|----|------------|-----------------|------|
| `strategy_schedules.next_run_at` | `model/strategy_schedule.go:221` `ComputeNextRunAtFromConfig` 包 `time.Now()`（cron/interval 语义本身是本地钟面） | `schedule_engine.go:187-191` `nowTime()=time.Now()` → `GetDueSchedules` 参数（`schedule_execute.go:108`） | 配对自洽 |
| `trade_logs.created_at` | `repository/auto_trading_settings.go:92` `log.CreatedAt = time.Now()` | `trade_log_repository.go:113-116` `ListByDateRange` start/end 参数（CST 惯例；仓外零调用，死 API 顺手记录） | 配对自洽 |
| `account_connection_logs.created_at` | `model/logs.go:50` `NewAccountConnectionLog` `time.Now()` | 查询侧 CST 参数惯例 | 配对自洽 |
| `system_operation_logs.created_at` | `model/logs.go:135` `NewSystemOperationLog` `time.Now()` | **混合读侧**：CST 参数 + `admin_repo_logs.go:120/:130` `NOW()-interval`——NOW()=UTC 对 CST 编码列 → 窗口 **+8h 虚高**（CST 值比 UTC instant 大 8h，`NOW()-1h` 实际捞回 ~9h 数据） | 混合读侧，现状如实记录 |

**规则**（registry 原文）：未来任何写入/读取这些列的新代码必须保持 **CST 参数一致**；或整列一次性迁移 UTC（写+读+存量回填**同批**）。**禁止单侧 `.UTC()`**——单侧翻转会制造同列混合编码（TZ-MIXED-ENCODING-1 正是此形态根因）。

**边界外（不修，如实记录）**：`admin_repo_logs.go:120/:130` NOW()-interval 查询的 +8h 虚高属存量混合读侧，改修 = 行为变更另债；`model/auto_trading.go:177/:195` `.UTC()` 写入属 RiskConfig/GlobalSettings 他表，非配对列。

## S1 — 规则落地 `docs/constraints.md`

`## Data Precision` 节（`Time: UTC` 行之后）追加一条 bullet：

```markdown
- **CST 配对 timestamp 列——禁止单侧 `.UTC()`（TZ-PAIRED-CST-COLS-1）**：以下列为 CST 写入+CST 参数读取自洽配对：`strategy_schedules.next_run_at`、`trade_logs.created_at`、`account_connection_logs.created_at`、`system_operation_logs.created_at`。新代码写/读这些列必须保持 CST 参数一致；或整列一次性迁移 UTC（写+读+存量回填同批）。单侧 `.UTC()` = 制造同列混合编码（TZ-MIXED-ENCODING-1 根因形态）。注意 `system_operation_logs` 读侧含 `NOW()-interval` 存量混合（DB NOW()=UTC，窗口 +8h 虚高，未修）。
```

## S2 — 坑库指针 `docs/pitfalls.md`

`### 已确认的静默失败模式` 段尾追加一条 bullet：

```markdown
- **CST 配对列单侧 `.UTC()` → 同列混合编码（TZ-PAIRED-CST-COLS-1，文档债）** — `next_run_at`/`trade_logs`/`account_connection_logs`/`system_operation_logs` 的 `created_at` 四列为 CST 写+CST 读配对；好心"修时区"单侧加 `.UTC()` 会制造半 CST 半 UTC 的混合列。另注意 `system_operation_logs` 的 `NOW()-interval` 查询因 DB NOW()=UTC 对 CST 列窗口 +8h 虚高。规则与列清单见 `docs/constraints.md` Data Precision 节。
```

## S3 — schema catalog 列注记 `docs/spec/09-postgres-schema-catalog.md`

四张表各加一行注记（位置：各表 `### <table>` 小节的 Key columns 行附近，保持文件既有注释风格）：

- `### trade_logs`（:180 附近）：`created_at` — CST 编码列（写读须同编码，见 `docs/constraints.md` CST 配对列规则）。
- `### strategy_schedules`（:293 附近）：`next_run_at` — 同上。
- `### system_operation_logs`（:778 附近）：`created_at` — 同上；另读侧 `NOW()-interval` 为 UTC 对 CST 列，窗口 +8h 虚高（存量未修）。
- `### account_connection_logs`（:787 附近）：`created_at` — 同上。

每处一行指针，不复制规则全文（P3 单一真相源：规则细节只在 constraints.md）。

## S4 — 写入点行内注释（4 处，各一行）

| 位置 | 追加注释（放该行尾或紧邻上行，保持 Go 注释风格） |
|------|------|
| `backend/internal/model/strategy_schedule.go:221`（`return ComputeNextRunAtFromConfigAt(..., time.Now())` 行紧邻上方既有注释区） | `// TZ: CST-paired column next_run_at — do not add .UTC() here (see docs/constraints.md CST pairing rule).` |
| `backend/internal/repository/auto_trading_settings.go:92`（`log.CreatedAt = time.Now()` 行尾） | `// CST-paired column trade_logs.created_at — keep CST encoding, no .UTC() (constraints.md)` |
| `backend/internal/model/logs.go:50`（`CreatedAt: time.Now(),` 行尾） | `// CST-paired column account_connection_logs.created_at — no .UTC() (constraints.md)` |
| `backend/internal/model/logs.go:135`（`CreatedAt: time.Now(),` 行尾） | `// CST-paired column system_operation_logs.created_at — no .UTC() (constraints.md)` |

英文短注释（代码注释惯例），指向 constraints.md，不复制规则。

## S5 — 收尾

- registry 行 61 追加施工完成记录（🟦open 施工完成待复审格式）。
- `docs/handoff/STATE.md` 施工表/下一步指针同步。

## 对抗证明

文档债无运行时变异可做——**如实声明 N/A**（无可 pin 的运行时行为；配对事实已由 Devin CLI 活库+代码双侧实证）。若施工方想出可判别化手段（如 grep 级 lint 防回归脚本）可提出但**不强制**——宁缺毋滥，不造形式主义断言。

## 验收门禁

`go build ./...`（注释改动编译）/ `go test -count=1 ./internal/model/ ./internal/repository/ ./internal/connect/strategy/`（受影响包回归）/ gofmt 本批文件 / `check-file-lines --strict` / `git diff --check`。文档改动为主，Go 仅 4 处注释。

## 边界（不做）

- **不改任何写读行为**：不动 `time.Now()`/`.UTC()`/SQL/NOW() 任何一处。
- 不修 `admin_repo_logs.go:120/:130` NOW()-interval（存量混合读侧，另债）。
- 不动 `model/auto_trading.go:177/:195`（他表）。
- 不迁移任何列、不写 migration、不动存量数据。
- 勿部署、勿 push、禁 `--no-verify`。完成报证据停手等 Devin CLI 复审。
