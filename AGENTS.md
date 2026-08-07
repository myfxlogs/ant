# Project "ant" — Mandatory Constraints

> 🔴 **ACTIVE TASK**: 市场数据架构简化 — 详见 [`docs/adr/0012-remove-tick-persistence.md`](docs/adr/0012-remove-tick-persistence.md)

## 协作模式：审计方 ↔ 施工方（动工必读）

> **🔴 动工前必读 [CLAUDE.md](CLAUDE.md)（项目宪法，单一完整源）。本文件是精简入口，完整规则 + 工作方法以 CLAUDE.md 为准；详细施工 SOP 见 [docs/audits/builder-sop.md](docs/audits/builder-sop.md)。**

本项目用「审计方(Claude Code) + 施工方(Windsurf / 其他 agent)」分工：

- **审计方**：代码级定位根因，把根因/位置/修复方向/验收标准写进 `docs/audits/tech-debt-registry.md` 条目。
- **施工方（你）**：实现修复 + **回填进度记录**。你不重新审计、不扩大范围（one task = one scope）、不自由发挥。

**新会话第一步**：读 `docs/audits/handover-audit-plan.md` 知当前任务/全局进度 + `tech-debt-registry.md` 知 open gap（对齐状态，无 memory 自动注入则必须主动读）。

**动工前**：① 读 `docs/audits/tech-debt-registry.md` 对应条目（根因/位置/验收标准已写好）；② `git log --all --oneline -- <path>` + `git blame` 理解原设计意图，先判断"bug→精准修"还是"有意移除→先讨论"，**禁止不读历史就重写**；③ `bash scripts/cap.sh <动词/符号>` 查是否已有现成能力（Reuse Preflight）。

**完工后必须回填，否则该任务判失败**：
1. `docs/audits/tech-debt-registry.md`：条目状态 `🟦open → ✅done`（标日期），末尾追加 **真实根因 + 修复方式（改了哪些文件/函数）+ 对抗证明（删关键一行则测试必红）+ 测试结果（如"50 次连跑 0 失败"）**。若真实根因与审计方假设不同，**如实写明**（如 BT-6 审计方假设 time.Now，真根因是 map 迭代）。只改状态列 + 追加，不删条目、不改审计方事实陈述（保留决策轨迹）。
2. **普遍 pitfall → 沉淀进本文件同类段**（如「MQL2GO VM Pitfalls」），写成永久约束防再犯。
3. 发现新 gap → 在 registry **新增条目**（`🟦open` + 根因/位置/修复方向），不要塞进现有条目。
4. **禁止新建并行进度文档**（另起 progress.md 之类 = 碎片化事实源，违规）；**禁止自行宣告"完成"**——完工汇报后等审计方核对状态 + 实测，确认才 `✅done`。

状态语义：`❓待核` = 记录过未对账当前代码 / `🟦open` = 已核验仍存在 / `✅done` = 已修且经审计方验收。详细 SOP 与红线见 [`docs/audits/builder-sop.md`](docs/audits/builder-sop.md)。

These constraints are enforced at implementation time. Violation = fix before commit.

## File & Function Size

**原则**: 按语义域（功能边界）拆分优先，行数作为软性参考。拆分的目的是帮助 AI 阅读代码——如果文件逻辑内聚，适度超标优于碎片化。

| Language | 软性参考 | 函数参考 |
|----------|---------|---------|
| Go       | 300 行  | 50 行   |
| TypeScript | 250 行 | 50 行   |

- **拆分前先判断**：是否有明确的功能边界（CRUD/生命周期/实体类型）？有 → 拆。没有 → 保持内聚。
- **硬性红线**：Go >450 行、TS >375 行必须拆分（AI 明显退化）。
- 自动生成代码（`gen/`）、测试文件、i18n 文件豁免。
- 检查：`cd backend && go run ./tools/check-file-lines --strict`（🔴 阻断 CI，🟡🟢 通过）。
- 详细：见 `complexity-limits.md` 分级严重度系统。

## Command Output Discipline (Token Efficiency)

**优先级**: Claude Code 内置工具 > `rtk` 前缀 > 裸命令

| 操作 | ✅ 首选 | ⚠️ 次选 | ❌ 禁止 |
|------|--------|--------|--------|
| 读文件 | Read 工具 | `rtk read` | `cat` / `head` / `tail` |
| 搜索文本 | Grep 工具 | `rtk grep` | `grep -rn` |
| 查找文件 | Glob 工具 | `rtk find` | `find` |
| 统计行数 | — | `rtk wc` | `wc -l` |
| 列目录 | — | `rtk ls` | `ls -la` |

- **内置工具（Read/Grep/Glob）零 token 开销**，且结果格式化，始终优先使用。
- 内置工具无法满足时（如需要复杂管道、非文件操作），使用 `rtk` 前缀命令，利用 RTK 过滤器压缩输出。
- **裸 `grep -rn` / `find` / `cat` / `head` / `tail` 禁止在 Bash 中直接使用。**
- 验证：`rtk discover` 定期检查遗漏，目标裸命令占比 <5%。

## Prohibited (Zero Tolerance)

- ❌ REST endpoints (except healthz/readyz/livez/metrics)
- ❌ WebSocket
- ❌ JSON 作为数据序列化/持久化/交换格式（包括 `json.load`/`json.dump`/`json.Marshal`/`json.Unmarshal`/`encoding/json`/`import json`）。所有跨进程数据交换用 proto，本地持久化用 PostgreSQL。豁免：自动生成产物（`gen/`、tree-sitter `grammar.json`/`node-types.json`）、PG `JSONB` 列（由 DB 管理，不在应用层做 `json.Marshal`）、以及外部 HTTP API 响应解析（如 TronGrid、OpenAI、ZhipuAI 等第三方 API 返回 JSON，必须用 `encoding/json` 解析——此为外部协议约束，非本项目选择）
- ❌ float64 in price calculations (use `decimal.Decimal` in Go)
- ❌ Cross-scope changes (one task = one scope)
- ❌ Hardcoded secrets / `.env` in repo
- ❌ `//nolint`, `# noqa`, `// @ts-ignore`, `// #nosec`
- ❌ 因困难而妥协最优解。遇到阻碍时禁止退而求其次——必须回到根因，找到正确的修复方式，哪怕需要推翻旧架构、完全重构。快捷方式（回退代替重新生成、标记 legacy 代替移除、沉默代替修复）视为违规。

## Reuse Preflight (避免重复造轮子 — 强制)

动工任何**新 file/function** 前必须执行复用核对：

1. 用 `bash scripts/cap.sh <动词/别名/符号>` 查能力（自动刷新 + 只返回命中行 + 代码层兜底）。**禁止整篇 Read `docs/CAPABILITIES.md`**（浪费 token）。
2. 按**动词 + 别名**（见该文件「动词/别名索引」）多换几个词查，确认是否已有现成能力。
3. 在 PR 描述里逐条给结论，二选一：`REUSE: <symbol> @ <file:line>`（复用现成）或 `NEW: 无现成能力（已搜：<关键词>）`（确认真空白）。
4. 发现能力状态/注释过时（注释说"不存在"但其实已实现）→ 同步修正注释与 `docs/CAPABILITIES.md`。

**缺少 `REUSE:`/`NEW:` 引用 = 该任务判失败。** 既防重复造轮子，也防误判 shelf-ware（分层状态：gateway-rpc / executor / mthub-method / connect-rpc / wired-live）。

## Platform Protocol

- External API: **ConnectRPC + SSE ONLY**
- Internal: in-process function calls OR NATS JetStream
- MT access: mtapi gRPC ONLY (via `adapter/mt4/` and `adapter/mt5/`)
- MT4 and MT5 adapters MUST NOT share code (except `adapter/mdtick/` shared DTO)

## Push-First Architecture

- **gRPC streaming + SSE is the default.** Prefer server-push over client-pull in every scenario.
- ❌ Polling / cron / `setInterval` / `time.Ticker` — ONLY when the data source has no push capability AND the data is not latency-sensitive
- ❌ Never poll when a streaming equivalent exists (e.g. MT5 `OnQuote` stream over polling `GetQuote`, SSE `bar_update` over polling `PriceHistory`)
- ✅ If adding a new data feed, ask first: "Can this be a stream?" If yes, make it a stream

## Data Precision

- Prices: `NUMERIC(20,8)` PG / `Decimal(18,6)` CH / `decimal.Decimal` Go
- Time: UTC, millisecond precision (`int64 ts_unix_ms`)
- Symbol: raw broker symbol = canonical (no suffix stripping)

## Deployment (强制 — 禁止手动)

- **后端部署唯一方式**: `docker compose build backend && docker compose up -d backend`
- **前端部署唯一方式**: `docker cp frontend/dist/. alphaforge-frontend:/usr/share/nginx/html/ && docker exec alphaforge-frontend nginx -s reload`
- **❌ 禁止宿主机 `go build` → `docker cp` 到容器**（宿主 glibc，容器 Alpine musl，二进制不兼容）
- **❌ 禁止在运行中容器里 `go build` 或 `apk add build-base`**（污染运行时环境，容器重建即丢失）
- **迁移文件**: `git status backend/migrations/` — 未提交的 `.up.sql` 会随 Docker build 自动执行；WIP 文件先移走再 build
- 项目使用 multi-stage Docker build（`backend/Dockerfile`）：builder stage 在 `golang:alpine` 里编译 CGO 代码，runtime stage 只拷贝二进制 + `mql.so`
- 运行中二进制名是 `/app/alphaforge`（不是 `alphaforge-backend` / `server`）

## MQL2GO VM Pitfalls (必读)

> 回测不开单但 MT4 客户端正常？先查 [`docs/runbook/mql2go-known-pitfalls.md`](docs/runbook/mql2go-known-pitfalls.md)
>
> - **未知常量静默替换为 0** — `interp/constants.go` 缺少常量定义时编译器不报错，直接 push 0，导致指标返回错误线
> - **`builtinOrderType` 返回值映射错误** — 返回 PositionSide(1/-1) 而非 OP_BUY/OP_SELL(0/1)，持仓管理失效
> - **Go map 迭代序非确定 → 用户函数前向引用静默返回 0** — `ir.Funcs` 是 map，遍历编译用户函数时 caller 可能先于 callee 编译 → callee 落入 "unknown function" 盲区 → 返回值静默替换为 0 → volume=0 / 指标值=0。修复：两遍编译（Pass 1 预注册 entry PC，Pass 2 编译体）。**通用规则：编译器/链接器中禁止裸遍历 map 处理有序依赖**

## Strategy Runner Rules (实盘执行 — 强制)

**Open bar 过滤（LIVE-1）**：
- ❌ 策略 runner 禁止处理未收盘 bar（`bar.Closed == false`）—— open bar 是行情快照，非策略事件
- ✅ 用 `shouldRunOnBar(bar, symbol, timeframe)` 纯函数过滤（`live_runner.go:214`）
- ✅ extra-symbol context window 也只用 finalized bar（`live_runner.go:231`）
- 后果：open bar 进 handleBar → 同一根 bar 重复执行 → 指标重复计数 → 实盘与回测发散

## Backtest Status Management (回测状态 — 强制)

**新增回测终态时的检查清单**（BT-5 教训：DEGRADED 曾漏 4 处）：
新增或修改回测终态状态时，**必须同步更新以下所有位置**，缺任一 = 状态推送断链：

1. `status_constants.go` — 状态常量 + `isTerminalBacktestStatus()` 函数
2. `backtest_run_worker.go` — lease CASE 语句 + `pg_notify` 条件
3. `strategy_backtest_watch.go` — SSE watch 终态判断（用 `isTerminalBacktestStatus` helper）
4. `strategy_converters.go` — `backtestStatusToProto()` switch case + `IsTerminal` 字段
5. proto enum — `antv1.BacktestRunStatus` 枚举值

## Before Commit

```bash
go build ./...                                          # must pass
cd backend && go run ./tools/check-file-lines --strict   # file size check (🔴 blocks, 🟡🟢 pass)
bash scripts/gen_capability_map.sh                      # refresh docs/CAPABILITIES.md (reuse preflight)
```
