# Project "ant" — Mandatory Constraints

## Codebase Navigation（功能块导航）

用名字定位代码：说块名即可，AI 按表定位。调试时追管线（块名 → 目录 → 文件）。

| # | 块名 (EN) | 中文 | 目录 | 一句话 |
|---|---|---|---|---|
| 1 | `mt-gateway` | MT网关 | `backend/adapter/mt{4,5}/` `backend/internal/mthub/` | mtapi.io 连接 MT, 下单/查持仓/拉K线 |
| 2 | `strategy-runtime` | 策略运行时 | `backend/strategy/{sdk,runner,backtest,indicators}/` | Strategy接口, Bar重放, 回测指标, 技术指标库 |
| 3 | `mql-compiler` | MQL编译器 | `backend/tools/mql2go/` | MQL/Python → tree-sitter → IR → Bytecode → VM |
| 4 | `agent-engine` | Agent引擎 | `backend/internal/{agent,ai}/` | 策略生成, 盲区桥接, 画像, 解读, 记忆, 回溯 |
| 5 | `backtest-engine` | 回测引擎 | `backend/internal/backtest/` `backend/strategy/backtest/` | SimBroker, 撮合, 滑点, 手续费, 净值曲线 |
| 6 | `risk-gate` | 实盘风控 | `backend/internal/{risk,risksvc,paper,oms}/` | 6门管线, 仿真交易, 订单管理, 熔断 |
| 7 | `account-mgmt` | 账户管理 | `backend/internal/connect/{gateway,marketplace,user}/` | MT账户CRUD, 经纪商搜索, 用户体系 |
| 8 | `market-data` | 市场数据 | `backend/internal/{mdgateway,source,symbol}/` | K线/Tick存储(CH+PG), 实时报价流 |
| 9 | `frontend` | 前端界面 | `frontend/src/` | React, 策略工作区, 回测面板, Agent聊天 |
| 10 | `api-gateway` | API层 | `backend/internal/connect/*/` `proto/ant/v1/` | ConnectRPC handlers, SSE, proto定义 |

**怎么用**：对话中提块名即可。"agent-engine 生成策略超时" → AI 定位 `backend/internal/agent/generator.go`。"mql-compiler IR 不兼容" → AI 定位 `backend/tools/mql2go/compile_py.go`。

**管线调试**（跨块追数据流，出问题时按线追）：

| 管线 | 路径 |
|------|------|
| 行情引入 | `mt-gateway(MT4/5) → market-data(去重/质量/归一化) → NATS + CH/PG` |
| 策略执行 | `market-data(bar源) → strategy-runtime(runner) → risk-gate(信号管线) → oms(16状态机) → mt-gateway(下单)` |
| 订单对账 | `mt-gateway(订单事件) → mthub(幂等门/对账门) → oms(状态更新) → NATS(实时PnL)` |
| Agent循环 | `frontend(用户输入) → api-gateway(SSE) → agent-engine(generate/revise) → mql-compiler(compile) → backtest-engine(SimBroker) → agent-engine(分析/迭代)` |
| 回测 | `frontend(参数) → api-gateway → backtest-engine(VMRunner+SimBroker) → PG(结果) → SSE(通知)` |
| 实盘调度 | `frontend(计划) → api-gateway → strategy-runtime(LiveRunner) → market-data(实时bar) → mt-gateway(下单)` |

**常用调试入口**："净值曲线为空" → 追 策略执行 线，从 `frontend` filter 到 `CH` 逐层查。"Agent生成超时" → 追 Agent循环 线，看是 LLM 推理卡了还是回测卡了。

---

## Mandatory Constraints

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
- ❌ JSON 作为数据序列化/持久化/交换格式（包括 `json.load`/`json.dump`/`json.Marshal`/`json.Unmarshal`/`encoding/json`/`import json`）。所有跨进程数据交换用 proto，本地持久化用 PostgreSQL。豁免：自动生成产物（`gen/`、tree-sitter `grammar.json`/`node-types.json`）和 PG `JSONB` 列（由 DB 管理，不在应用层做 `json.Marshal`）
- ❌ float64 in price calculations (use `decimal.Decimal` in Go)
- ❌ Cross-scope changes (one task = one scope)
- ❌ Hardcoded secrets / `.env` in repo
- ❌ `//nolint`, `# noqa`, `// @ts-ignore`
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

## Before Commit

```bash
go build ./...                                          # must pass
cd backend && go run ./tools/check-file-lines --strict   # file size check (🔴 blocks, 🟡🟢 pass)
bash scripts/gen_capability_map.sh                      # refresh docs/CAPABILITIES.md (reuse preflight)
```

Full constraint details: see `/root/.claude/projects/-opt-ant/memory/`
