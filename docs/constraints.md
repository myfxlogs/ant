# 技术约束（T1 知识层）

> 项目技术约束 SSOT。agent 开工前按需读取相关章节。≤ 450 行。

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

**优先级**: Claude 内置工具 > `rtk` 前缀 > 裸命令

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
- ❌ 硬编码"本应来自外部权威源的可变数据"（broker symbol 清单、broker 参数、服务器地址等外部系统当前状态）。存在权威查询（`FetchAllSymbols`、broker RPC）时禁止写死静态列表——必然漂移→静默 bug。**反例 LIVE-PRICE-4**：`defaultQuoteSymbols()` 硬编码 37 个 symbol，含 broker 上不存在的 `XAUJPYm`/`EURUSDm`；mtapi `SubscribeMany` 是**原子操作**，一个不存在 → 整批 37 个全被拒 → 连 100% 存在的 `XAUUSDm` 都订阅不上 → `OnQuote` 零交付 → 实盘策略收不到任何报价 → 无法开仓。修复 = 订阅前用 `FetchAllSymbols`（broker 真实 symbol 清单）过滤，只订存在的。**豁免**：通用常量（标准 timeframe 毫秒 `60_000`、数学常量、固定枚举）。
- ❌ **用本地计算/推导替代服务器权威数据（用户 2026-08-19 确立："服务器有的数据，一律以服务器为准，这是唯一真相"）**。broker/外部服务器返回的字段（balance/equity/margin/free_margin/margin_level、持仓、订单、symbol 参数…）是唯一真相，本地一律**只做透传与持久化，不做重算**。推论：① 存在权威 RPC 时禁止自算（如禁止按 contractSize 反推 margin，必须取 `AccountSummary`）；② **禁止用"不含该字段的次级数据源"覆盖权威值**——反例 DATA-TRUTH-2：MT4 `OnOrderProfit` 帧不填 margin/margin_level，却每 5s 把 `mt_accounts.margin` 覆盖成 0，导致 `MarginLevel > 0` 门槛永不成立、MT4 爆仓预警完全不触发（同一字段 MT5 帧填得完整，`AccountSummary` 也填得完整）；③ 一个事实只允许一个写入方 + 一张真相表——反例 DATA-TRUTH-4：两个同名 `RecordBalanceSnapshot` 写两张不同表，生产那份写进**不存在**的表且错误被 `log.Debug` 吞，净值曲线静默断供 28 天。豁免：纯展示派生值（百分比、涨跌幅）可本地算，但不得回写覆盖权威字段。
- ❌ `//nolint`, `# noqa`, `// @ts-ignore`, `// #nosec`
- ❌ **缺失/陈旧 broker 快照静默转零或继续执行**：权威金融快照必须带来源与采集/接收时间；缺失、非法、过期时策略 VM 与 Risk Gate 必须 fail-closed。订单流只可更新持仓事实，不得覆盖 AccountSummary 的金融字段；无 session 查询不得返回空订单成功。
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
- **mtapi gateway host configuration** (BROKER-SEARCH-1): the mtapi gRPC gateway
  addresses used by broker search and per-account gateways are configurable via
  environment variables `MTAPI_MT4_HOST` and `MTAPI_MT5_HOST` (e.g.
  `MTAPI_MT4_HOST=mt4grpc3.mtapi.io:443`). Empty/unset values fall back to the
  hardcoded defaults (`mt4grpc3.mtapi.io:443` / `mt5grpc3.mtapi.io:443`) so
  existing deployments are not broken. Set these in the container environment to
  route mtapi traffic through a regional proxy or failover endpoint.

## Push-First Architecture

- **gRPC streaming + SSE is the default.** Prefer server-push over client-pull in every scenario.
- ❌ Polling / cron / `setInterval` / `time.Ticker` — ONLY when the data source has no push capability AND the data is not latency-sensitive
- ❌ Never poll when a streaming equivalent exists (e.g. MT5 `OnQuote` stream over polling `GetQuote`, SSE `bar_update` over polling `PriceHistory`)
- ✅ If adding a new data feed, ask first: "Can this be a stream?" If yes, make it a stream

## Frontend Zero-Trust (公开面刚性)

- **所有衍生统计必须后端算，前端只渲染**——前端可被篡改，公开面数字必须后端权威
- 公开面衍生统计必须后端算；回撤等指标须后端用 equity 算真值（peak-to-trough），勿用单笔最差冒充
- 前端允许：纯格式化（`.toFixed`/`toLocaleString`）、展示变换（用后端 winRate 拆饼图）

## Data Precision

- Prices: `NUMERIC(20,8)` PG / `Decimal(18,6)` CH / `decimal.Decimal` Go
- Time: UTC, millisecond precision (`int64 ts_unix_ms`)
- Symbol: raw broker symbol = canonical (no suffix stripping)
- **md_bars 查询的 `DISTINCT ON` / `ORDER BY` 不能把 `broker` 排在时间列前面**（反例 BT-MULTIBROKER-ORDER，2026-08-24）：`GetKlines(broker="")` 曾把 `broker` 放进 distinct key + `ORDER BY broker, ...open_ts` → 多 broker 写同一 canonical 时按 broker 名排序而非按时间排序 → 回测崩 `bars are not chronologically ordered`。正确做法：distinct key 恒为 `(canonical, period, open_ts_unix_ms)`，`ORDER BY ...open_ts_unix_ms, tick_count DESC`，`broker` 仅作可选 WHERE 过滤——跨 broker 去重（最高 tick_count 胜出）+ 全局时序。所有 backtest/market-data 调用方都传 `broker=""` 且都需要单一时序。

## Documentation Rules（文档约束 — 强制）

- 每个功能块有独立文档目录：`docs/blocks/<块名>/`。目录下包含 `README.md`（块入口）+ `plans/`（施工计划）。
- **新增/修改文档时，若内容只涉及一个功能块，必须写入对应块的目录。**
- 跨块文档放入 `docs/adr/`（决策）或 `docs/spec/`（规格），并在文档头标注涉及的功能块。
- 顶层 `docs/blocks/README.md` 是块索引。

**块文档入口**：说块名即可定位。`docs/blocks/<块名>/README.md` 包含代码位置、依赖、关键设计、施工计划。

## Deployment (强制 — 禁止手动)

- **后端部署唯一方式**: `docker compose build backend && docker compose up -d backend`
- **每次 build 前先清理 Docker build cache**: `docker builder prune -f`（每次构建约产生 2-3GB cache，58G 磁盘不清理会快速吃满）
- **前端部署唯一方式**: `docker cp frontend/dist/. alphaforge-frontend:/usr/share/nginx/html/ && docker exec alphaforge-frontend nginx -s reload`
- **❌ 禁止宿主机 `go build` → `docker cp` 到容器**（宿主 glibc，容器 Alpine musl，二进制不兼容）
- **❌ 禁止在运行中容器里 `go build` 或 `apk add build-base`**（污染运行时环境，容器重建即丢失）
- **迁移文件**: `git status backend/migrations/` — 未提交的 `.up.sql` 会随 Docker build 自动执行；WIP 文件先移走再 build
- 项目使用 multi-stage Docker build（`backend/Dockerfile`）：builder stage 在 `golang:alpine` 里编译 CGO 代码，runtime stage 只拷贝二进制 + `mql.so`
- 运行中二进制名是 `/app/alphaforge`（不是 `alphaforge-backend` / `server`）

### 部署验证 Pitfalls（QC-CACHE-LEAK/STALE-HTML-CACHE 2026-08-16 教训，两次翻车）

1. **施工方回填 ✅done ≠ 已部署**——Windsurf 曾把修复写对并回填 ✅done（QC-CACHE-LEAK，2026-08-16），但从未 build + docker cp，用户线上一直是旧包。验收必须实测容器内资产：`docker exec alphaforge-frontend ls -la /usr/share/nginx/html/assets/`（看时间戳是否晚于修复 commit）+ 对比 index.html hash。
2. **部署验证要到"入口响应头"层**——文件在容器里 ≠ 浏览器会拿新的。必查 `curl -sI http://localhost:8022/` 带 `Cache-Control: no-cache`。
3. **nginx `try_files /index.html =404` 是就地吐文件**，绕过 `= /index.html` location 的响应头——要应用该块的头必须让 try_files **内部重定向**（`try_files /不存在的守卫路径 /index.html`）。同理：location 内任一 `add_header` 会丢失**全部** server 级继承头（含 CSP/HSTS），需整组重声明。

## RTK 兼容规范

- ⚠️ 避免使用 `cd && cmd` 或换行拼接的复合 Bash 块
- ✅ 将每条命令作为独立 Bash 工具调用发出（如 `git status`、`grep -rn ...`）
- ✅ RTK 会自动处理输出截断，无需手动 `| head/tail`
- ❌ 不要使用管道限制输出（| head, | tail, | grep 用于分页）

## Before Commit

```bash
go build ./...                                          # must pass
cd backend && go run ./tools/check-file-lines --strict   # file size check (🔴 blocks, 🟡🟢 pass)
bash scripts/gen_capability_map.sh                      # refresh docs/CAPABILITIES.md (reuse preflight)
```
