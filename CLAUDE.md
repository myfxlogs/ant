# Project "ant" — Mandatory Constraints

## Business Direction（经营方向 — 最高优先级）

- **产品定位**：策略市场，不是量化工具平台。对标 MQL5 Market，核心差异：代码不出平台、实盘战绩公开、AI 持续迭代策略。
- **服务群体**：MQL 策略开发者（供给侧）+ 零售交易者（需求侧）。不服务专业量化机构。
- **收入模型**：平台订阅（Free/Pro/Enterprise）+ 策略抽成（15-30%，Admin 后台可调）。不做自营策略、不做跟单、不拿牌照、不碰用户资金。
- **生态**：兼容所有 MT4/MT5 broker（用户自带）。Year 1 聚焦 MT 生态，IR 层保持语言中性，为未来接入加密市场保留技术前提。
- **终极壁垒**：不是技术，是市场流动性——策略最多、用户最多、战绩数据最全。
- 详见：`docs/roadmaps/business-direction.md`

## Collaboration Principle（协作原则 — 最高优先级）

- **讨论 → 决定 → 执行**，这个顺序不可跳过。
- 用户的指令不是最终决策。AI 的技术判断力优于用户时，**必须指出错误或提出更优解**，而不是直接执行。
- 必要时引入第三方视角（如让其他模型审设计文档）。
- 双方达成共识后，再动手。跳过讨论直接执行 = 失职。

## Root-Cause-First Rule（根因优先 — 最高优先级）

**当用户报告一个功能消失、行为改变、或出现退化时，禁止不查历史直接重写。**

1. **先查 git log**——用 `git log --all --oneline -- <path>` 找到相关文件的所有历史变更，定位功能最后一次正常工作的 commit。
2. **再用 git blame**——确认当前出问题的代码是谁引入的、在哪个 commit、为了什么目的。
3. **理解原始设计意图**——读原 commit message、关联的 ADR 或 spec 文档。当初为什么这样写？修了什么？改了哪个路径？
4. **判断是丢失还是故意移除**——如果是后续修改引入的 bug → 精准修复那个变更，保留原始设计。如果是有意移除 → 先讨论是否应该恢复。
5. **只在确认"从未实现过"后才新写**——如果 git 历史证明这个功能确实从来没有存在过，此时才能从零实现。

**禁止行为**：
- ❌ 看到功能消失，第一反应是"重写一个"
- ❌ 不读 git log 就开始写新代码
- ❌ 把以前的**最优解**替换成"我觉得更好的"新实现——尤其当原实现有明确的 ADR、spec、或复杂边界处理时。但如果原实现确实不是最优解（违反 Part 0.1），替换它不违反此规则——前提是已经通过 git log/blame 理解了原设计

**为什么**：重写取代查询，导致代码重复、设计退化、历史 bug 修复丢失、维护成本翻倍。

## Codebase Navigation（功能块导航）

用名字定位代码：说块名即可，AI 按表定位。调试时追管线（块名 → 目录 → 文件）。

| # | 块名 (EN) | 中文 | 目录 | 一句话 |
|---|---|---|---|---|
| 1 | `mt-gateway` | MT网关 | `backend/mt{4,5}/` `backend/internal/mdgateway/adapter/mt{4,5}/` `backend/internal/mthub/` | mtapi.io 连接 MT, 下单/查持仓/拉K线 |
| 2 | `strategy-runtime` | 策略运行时 | `backend/strategy/{sdk,runner,backtest,indicators}/` | Strategy接口, Bar重放, 回测指标, 技术指标库 |
| 3 | `mql-compiler` | MQL编译器 | `backend/tools/mql2go/` | MQL/Python → tree-sitter → IR → Bytecode → VM |
| 4 | `agent-engine` | Agent引擎 | `backend/internal/{agent,ai}/` | 策略生成, 盲区桥接, 画像, 解读, 记忆, 回溯 |
| 5 | `backtest-engine` | 回测引擎 | `backend/internal/backtest/` `backend/strategy/backtest/` | SimBroker, 撮合, 滑点, 手续费, 净值曲线 |
| 6 | `risk-gate` | 实盘风控 | `backend/internal/{risk,risksvc,paper,oms}/` | 6门管线, 仿真交易, 订单管理, 熔断 |
| 7 | `account-mgmt` | 账户管理 | `backend/internal/connect/{gateway,user}/` | MT账户CRUD, 经纪商搜索, 用户体系 |
| 8 | `market-data` | 市场数据 | `backend/internal/{mdgateway,source,symbol}/` | K线/Tick存储(PG), 实时报价流 |
| 9 | `frontend` | 前端界面 | `frontend/src/` | React, 策略工作区, 回测面板, Agent聊天 |
| 10 | `api-gateway` | API层 | `backend/internal/connect/*/` `proto/ant/v1/` | ConnectRPC handlers, SSE, proto定义 |
| 11 | `strategy-marketplace` | 策略市场 | `backend/internal/marketplace/` `backend/internal/connect/marketplace/` `frontend/src/pages/marketplace/` | 策略发布/发现/购买/冻结结算/AI迭代, 双边市场 |

**怎么用**：对话中提块名即可。"agent-engine 生成策略超时" → AI 定位 `backend/internal/agent/generator.go`。"mql-compiler IR 不兼容" → AI 定位 `backend/tools/mql2go/compile_py.go`。

**管线调试**（跨块追数据流，出问题时按线追）：

| 管线 | 路径 |
|------|------|
| 行情引入 | `mt-gateway(MT4/5) → market-data(去重/质量/归一化) → NATS + PG` |
| 策略执行 | `market-data(bar源) → strategy-runtime(runner) → risk-gate(信号管线) → oms(16状态机) → mt-gateway(下单)` |
| 订单对账 | `mt-gateway(订单事件) → mthub(幂等门/对账门) → oms(状态更新) → NATS(实时PnL)` |
| Agent循环 | `frontend(用户输入) → api-gateway(SSE) → agent-engine(generate/revise) → mql-compiler(compile) → backtest-engine(SimBroker) → agent-engine(分析/迭代)` |
| 回测 | `frontend(参数) → api-gateway → backtest-engine(VMRunner+SimBroker) → PG(结果) → SSE(通知)` |
| 实盘调度 | `frontend(计划) → api-gateway → strategy-runtime(LiveRunner) → market-data(实时bar) → mt-gateway(下单)` |
| 策略市场 | `frontend(市场页) → api-gateway → strategy-marketplace(发布/购买/冻结结算) → backtest-engine(回测验证) ‖ agent-engine(AI生成/迭代策略)` |

**常用调试入口**："净值曲线为空" → 追 策略执行 线，从 `frontend` filter 到 `PG` 逐层查。"Agent生成超时" → 追 Agent循环 线，看是 LLM 推理卡了还是回测卡了。

---

## Documentation Rules（文档约束 — 强制）

- 每个功能块有独立文档目录：`docs/blocks/<块名>/`。目录下包含 `README.md`（块入口）+ `plans/`（施工计划）。
- **新增/修改文档时，若内容只涉及一个功能块，必须写入对应块的目录。**
- 跨块文档放入 `docs/adr/`（决策）或 `docs/spec/`（规格），并在文档头标注涉及的功能块。
- 顶层 `docs/blocks/README.md` 是块索引。

**块文档入口**：说块名即可定位。`docs/blocks/<块名>/README.md` 包含代码位置、依赖、关键设计、施工计划。

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
- ❌ JSON 作为数据序列化/持久化/交换格式（包括 `json.load`/`json.dump`/`json.Marshal`/`json.Unmarshal`/`encoding/json`/`import json`）。所有跨进程数据交换用 proto，本地持久化用 PostgreSQL。豁免：自动生成产物（`gen/`、tree-sitter `grammar.json`/`node-types.json`）和 PG `JSONB` 列（由 DB 管理，不在应用层做 `json.Marshal`）。**追加豁免**：调用外部 LLM API（OpenAI/Anthropic 等）时，API 协议本身要求 JSON——此类外部边界调用不受此限
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

- Prices: `NUMERIC(20,8)` PG / `decimal.Decimal` Go
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

## Before Commit

```bash
go build ./...                                          # must pass
cd backend && go run ./tools/check-file-lines --strict   # file size check (🔴 blocks, 🟡🟢 pass)
bash scripts/gen_capability_map.sh                      # refresh docs/CAPABILITIES.md (reuse preflight)
```

Full constraint details: see `/root/.claude/projects/-opt-ant/memory/`
