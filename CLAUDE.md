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
- **AI 是第一责任人，自主推进**：用户非技术决策者。能从代码/约束/最佳实践推出的结论，**直接定方案并执行**，不把判断反推给用户（"你要我跑测试吗 / 选哪个方案"= 失职）。报告给"决策 + 证据 + 已执行"，非"选项 + 问你选"。只有真正需用户输入（业务偏好 / 外部凭证 / 不可逆外向操作）才问。技术方向仍先讨论达成共识，但执行细节自主定。

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

## AI 协作工作方法（审计方 ↔ 施工方 — 强制）

> 本文件是**项目宪法 + 单一完整源**，所有 AI 工具（Claude Code / Windsurf / Cursor / Codex）都以 CLAUDE.md 为准。`.windsurfrules`/`AGENTS.md` 是精简入口，不重复本文件，冲突以本文件为准。

本项目用「审计方 + 施工方」分工（详细 SOP + 范例见 `docs/audits/builder-sop.md`）：

- **审计方（Claude Code）**：只读、验证、记录、出 spec。代码级定位根因，把根因/位置/修复方向/验收标准写进 `docs/audits/tech-debt-registry.md`。**不直接改代码**（保持独立判断 + 省 token）。
- **施工方（Windsurf / 其他 agent）**：实现修复 + 回填进度。不重新审计、不扩大范围（one task = one scope）、不自由发挥。

**三层文档不丢失**（所有进度只进这三层，❌ 禁止新建并行进度文档）：

1. `docs/audits/tech-debt-registry.md` — 债务总账（每条 gap：根因/位置/状态/修复方向）。施工方工作台。
2. `docs/audits/handover-audit-plan.md` — 审计全局进度（管线状态表 + 变更日志）。
3. `memory/`（`open-items-registry.md` + `MEMORY.md`）— 高优摘要，Claude Code 跨会话自动注入。

**无损接手铁律（完工标 ✅ 不删）**：三层的目的 = 任何一方（审计方/施工方/后续 agent）休息，另一方读三层即可完整恢复"做了什么 / 为什么 / 验过没"。故完工项**标 ✅ 保留行，永不删除**——registry ✅ 行带根因/修复/对抗证明保留、memory 指针完工项标 ✅ + 指向 docs（不删行）、handover 变更日志 append-only。删一条完工记录 = 接手方少一块拼图 = 有损；**删了还以为没做，比没做更糟**。详见 `docs/audits/builder-sop.md` §2.6。**🆕 2026-08-11 修订（用户批准，省 token）**：✅done **明细行**允许归档 git——registry/handover 文件只留状态行 + 最近 changelog + "靠 git 追溯"指针；**open/返工中条目 + 根因 + 对抗测试记录 + changelog 追加**仍必留文件内。历史明细追溯：`git log --oneline -- <file>`。**自动执行（同日起，用户要求机器强制不靠提醒）**：git pre-commit 钩子 `scripts/hooks/pre-commit`（`core.hooksPath` 已注册；新 clone 需 `git config core.hooksPath scripts/hooks`）强制——变更日志条目 / ✅ 行 / 状态行禁删，唯一例外 = 文件仍保留"靠 git 追溯"指针的文档化裁剪；**任何 agent（含施工方）提交违规即被拦**，被拦 = 改好文档再提交，禁 `--no-verify` 绕过；施工方入口 `.windsurfrules`/`AGENTS.md` 同列该条。

**记忆分层**（多工具协作 — 强制）：

- **项目知识**（规则/状态/经验/决策/用户协作偏好）→ **只进项目文档**（`CLAUDE.md` / `docs/`）。所有工具的单一共享源，进 git。
- **私有 memory**（Claude Code `~/.claude`、Windsurf Cascade memory）→ **只放工具特定偏好 + 入口指针**，❌ 禁止存独立项目事实（否则跨工具信息孤岛 + 与项目文档漂移）。
- **原则：与其"两份同步"（必漂移），不如"一次写对地方"**。项目知识进项目文档，私有 memory 不重复它。
- 新会话状态感知：Claude Code 靠 `MEMORY.md` 指针；Windsurf/Cursor 靠 `.windsurfrules`/`AGENTS.md` 指引读 `docs/audits/handover-audit-plan.md`。入口不同，指向同一源。

**状态语义**：`❓待核`=记录过未对账当前代码 / `🟦open`=已核验仍存在 / `✅done`=已修且经审计方验收。

**完工回填纪律**（施工方，不做 = 任务判失败）：

1. `tech-debt-registry.md` 条目状态 `🟦open → ✅done`（标日期）+ 追加**真实根因/修复方式/对抗证明/测试结果**。若真实根因与审计方假设不同，**如实写明**（高价值纠偏）。只改状态列 + 追加，不删条目、不改审计方事实陈述。
2. 普遍 pitfall → 沉淀进本文件同类 Pitfalls 段（防再犯）。**沉淀时必须横扫 registry 所有同类前缀条目**（如修了 DATA-TRUTH-10，必须对账 DATA-TRUTH-1~9 的 pitfall 沉淀状态），不能只补最近一个——否则同类坑会随会话消失。2026-08-20 教训：第一轮只补了 DATA-TRUTH-10/LOG-UX-1，漏了 DATA-TRUTH-2~9，直到用户追问才发现。
3. `handover-audit-plan.md` 变更日志加一行。
4. **不自行宣告完成**——等审计方核对状态 + 实测。

**对抗证明**（任何修复必带）：删掉修复的关键一行，测试必红。删了还绿 = 测试无效 = 未完成。

**验收分离**：施工方不越权宣告完成；审计方核对状态 + 实测后，`✅done` 才权威。

**审计方验收 5 维**（区分平庸与优秀；"build/test 绿"只是底线，不是优秀）：
1. **意图理解**：解决 spec 背后真问题，还是字面 spec。
2. **可演进性**（工程最关键）：加同类功能改几处？绑死结构 vs 为变化留口子。
3. **测试质量**：验证"行为对"（含集成/边界/不变量语义），不是只跑路径。
4. **防御性**：主动想会出错处（空/nil/极值/负数/混合）。
5. **克制**：最简解，不炫技（过度设计）不偷懒（走捷径）。

## Codebase Navigation（功能块导航）

用名字定位代码：说块名即可，AI 按表定位。调试时追管线（块名 → 目录 → 文件）。

| # | 块名 (EN) | 中文 | 目录 | 一句话 |
| --- | --- | --- | --- | --- |
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
| ------ | ------ |
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

| Language   | 软性参考 | 函数参考 |
| ---------- | -------- | -------- |
| Go         | 300 行   | 50 行    |
| TypeScript | 250 行   | 50 行    |

- **拆分前先判断**：是否有明确的功能边界（CRUD/生命周期/实体类型）？有 → 拆。没有 → 保持内聚。
- **硬性红线**：Go >450 行、TS >375 行必须拆分（AI 明显退化）。
- 自动生成代码（`gen/`）、测试文件、i18n 文件豁免。
- 检查：`cd backend && go run ./tools/check-file-lines --strict`（🔴 阻断 CI，🟡🟢 通过）。
- 详细：见 `complexity-limits.md` 分级严重度系统。

## Command Output Discipline (Token Efficiency)

**优先级**: Claude Code 内置工具 > `rtk` 前缀 > 裸命令

| 操作 | ✅ 首选 | ⚠️ 次选 | ❌ 禁止 |
| ------ | -------- | -------- | -------- |
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
- ❌ 硬编码"本应来自外部权威源的可变数据"。当数据代表某**外部系统的当前状态**（broker symbol 清单、broker 参数、服务器地址、每账户/每经纪人不同的值），且**存在权威查询**（`FetchAllSymbols`、broker RPC 等）时，禁止写死静态列表——必然漂移 → 静默 bug。
  - **反例（LIVE-PRICE-4，2026-08-13 实盘无法开仓 P1）**：`defaultQuoteSymbols()` 硬编码 37 个 symbol，含 broker 上不存在的 `XAUJPYm`/`EURUSDm`；mtapi `SubscribeMany` 是**原子操作**，一个不存在 → 整批 37 个全被拒 → 连 100% 存在的 `XAUUSDm` 都订阅不上 → `OnQuote` 零交付 → 实盘策略收不到任何报价 → 无法开仓。修复 = 订阅前用 `FetchAllSymbols`（broker 真实 symbol 清单）过滤，只订存在的。
  - **正确做法**：查询权威源（建调度选 symbol 时早已这么做——实时拉 broker 列表；gateway 订阅必须用同一权威源，不得用硬编码 fallback）。
  - **豁免（合法硬编码）**：真正的**通用常量**——标准 timeframe 毫秒映射（`60_000`/`300_000`）、数学常量、固定枚举值。这些是普适固定值、非外部状态，不在此列。
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

## Frontend Zero-Trust (公开面刚性)

- **所有衍生统计必须后端算，前端只渲染**——前端可被篡改，公开面数字必须后端权威
- 公开面衍生统计必须后端算；回撤等指标须后端用 equity 算真值（peak-to-trough），勿用单笔最差冒充
- 前端允许：纯格式化（`.toFixed`/`toLocaleString`）、展示变换（用后端 winRate 拆饼图）

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

**部署验证 Pitfalls（QC-CACHE-LEAK/STALE-HTML-CACHE 2026-08-16 教训，两次翻车）**：

1. **施工方回填 ✅done ≠ 已部署**——Windsurf 曾把修复写对并回填 ✅done，但从未 build + docker cp，用户线上一直是旧包。验收必须实测容器内资产：`docker exec alphaforge-frontend ls -la /usr/share/nginx/html/assets/`（看时间戳是否晚于修复 commit）+ 对比 index.html hash。
2. **部署验证要到"入口响应头"层**——文件在容器里 ≠ 浏览器会拿新的。必查 `curl -sI http://localhost:8022/` 带 `Cache-Control: no-cache`。
3. **nginx `try_files /index.html =404` 是就地吐文件**，绕过 `= /index.html` location 的响应头——要应用该块的头必须让 try_files **内部重定向**（`try_files /不存在的守卫路径 /index.html`）。同理：location 内任一 `add_header` 会丢失**全部** server 级继承头（含 CSP/HSTS），需整组重声明。

## MQL2GO VM Pitfalls (必读)

> 回测不开单 / volume=0 / 指标全零但 MT4/MT5 客户端正常？先查 [`docs/runbook/mql2go-known-pitfalls.md`](docs/runbook/mql2go-known-pitfalls.md)

mql2go VM 的核心危险：**不报错、不崩溃、只产生错误行为**。四类已确认的静默失败：

| 类型 | 根因 | 症状 | 修复状态 |
| ------ | ------ | ------ | --------- |
| 未知常量 → 0 | `interp/constants.go` 缺常量 → 编译器 push 0 | 指标返回错误线（如 MODE_SIGNAL=0 → MACD==Signal → 永不开单） | ✅ 已补全 |
| map 迭代非确定 | `ir.Funcs` 是 map，遍历编译 → 前向引用落 "unknown function" → 返回值=0 | volume=0 flaky（同代码同命令时 PASS 时 FAIL） | ✅ 两遍编译 |
| OrderType 映射错误 | `builtinOrderType` 返回 SideBuy(1)/SideSell(-1) 而非 OP_BUY(0)/OP_SELL(1) | 持仓管理失效（平仓/止损条件永不触发） | ✅ 已修 |
| 固定长度滚动窗口 + append-only 指标缓存（LIVE-INDICATOR-1） | live seed 500 bars 后窗口恒长 500；`SeriesCache.EnsureUpdated()` 只比较 `Len()`，`n==c.n` 时跳过更新 | VM/bar/tick eval 持续增长但 EMA/MACD/RSI/ATR/ADX 永远停在启动首帧，策略静默 0 信号 | 🟦open（revisioned source 任意 revision 变化 reset+lazy rebuild；同 source+cache 500→500 + legacy start BAR→TICK 对抗） |

**编译确定性 — 强制**：

- ❌ 禁止裸遍历 Go map 处理有序依赖（编译器/链接器/任何有序 pipeline）
- ✅ 有前向引用 → 两遍编译：Pass 1 预注册所有 entry，Pass 2 编译体
- ✅ 无前向引用但需确定性 → 排序 key 后遍历
- Go map 迭代随机性是 **per-invocation**（非 per-process）

**测试数据确定性 — 强制**：

- ❌ 禁止 `time.Now()` 生成测试 bar timestamp（违反 spec 21 §10 Determinism Contract）
- ✅ 用固定 epoch：`time.Date(2024, 1, 1, 0, i, 0, 0, time.UTC)`

## Strategy Runner Rules (实盘执行 — 强制)

**Open bar 过滤（LIVE-1）**：

- ❌ 策略 runner 禁止处理未收盘 bar（`bar.Closed == false`）—— open bar 是行情快照，非策略事件
- ✅ 用 `shouldRunOnBar(bar, symbol, timeframe)` 纯函数过滤（`live_runner.go:214`）
- ✅ extra-symbol context window 也只用 finalized bar（`live_runner.go:231`）
- 后果：open bar 进 handleBar → 同一根 bar 重复执行 → 指标重复计数 → 实盘与回测发散

## Strategy Schedule Engine Pitfalls (SCHEDULE-HOTLOOP-1)

- **due timer occurrence 必须在所有 skip/deny/dispatch 分支前被持久化消费**：过期 `next_run_at` 若在 `isRunning`、`autoTrade=false`、entitlement/quota deny 等分支直接 `continue`，`GetEarliestNextRunAt` 会持续返回过去时间，timer delay=0 → CPU/DB/日志热循环。正确语义：timer schedule 每次 due 先推进 `next_run_at > now`，再决定是否 dispatch；autoTrade 关闭期间不补跑历史次数，恢复后从未来周期继续。
- **禁止在 live run 返回后才推进 next_run_at**：实盘 run 可以永久运行，`runOne` 完成路径不是 timer occurrence 的收敛点。event schedule 必须保持 `next_run_at=NULL`，timer repository 查询只选 interval/cron，startup 清理 event 脏 next 值。
- **持久化失败必须有界退避**：GetDue/ComputeNext/UpdateNext 失败时不 dispatch，ScheduleEngine 用 context-aware backoff timer 等待，`Notify` 可提前唤醒；invalid config 记录错误并 clear next 隔离。只降日志级别不能修复热循环。
- **autoTrade cache 必须由所有写入口主动失效**：`ToggleAutoTrade` 与 `UpdateGlobalSettings` 成功后都必须通过 callback 执行 `InvalidateAutoTradeCache(userID)+Notify()`，不能容忍关闭后 TTL 30s 内继续 dispatch。
- 对抗测试必须覆盖：autoTrade=false、already-running、eligible dispatch、GetDue/UpdateNext 失败退避+Notify、event 脏 next、两个 autoTrade 写入口、runOne 不二次更新。完整证据与方案见 registry `SCHEDULE-HOTLOOP-1`。

## Backtest Status Management (回测状态 — 强制)

**新增回测终态时的检查清单**（BT-5 教训：DEGRADED 漏了 4 处）：
新增或修改回测终态状态时，**必须同步更新以下所有位置**：

1. `status_constants.go` — 状态常量 + `isTerminalBacktestStatus()` 函数
2. `backtest_run_worker.go` — lease CASE 语句 + `pg_notify` 条件
3. `strategy_backtest_watch.go` — SSE watch 终态判断（用 `isTerminalBacktestStatus` helper）
4. `strategy_converters.go` — `backtestStatusToProto()` switch case + `IsTerminal` 字段
5. proto enum — `antv1.BacktestRunStatus` 枚举值

**缺少任一 = 状态推送断链**（DEGRADED 曾漏 4 处中 3 处 → 前端卡"运行中"30s + SSE 流不结束）

## Before Commit

```bash
go build ./...                                          # must pass
cd backend && go run ./tools/check-file-lines --strict   # file size check (🔴 blocks, 🟡🟢 pass)
bash scripts/gen_capability_map.sh                      # refresh docs/CAPABILITIES.md (reuse preflight)
```

Full constraint details: see `/root/.claude/projects/-opt-ant/memory/`

- docker compose 命令请使用 `rtk proxy docker compose ...` 避免原始输出

## RTK 兼容规范

- ⚠️ 避免使用 `cd && cmd` 或换行拼接的复合 Bash 块
- ✅ 将每条命令作为独立 Bash 工具调用发出（如 `git status`、`grep -rn ...`）
- ✅ RTK 会自动处理输出截断，无需手动 `| head/tail`
- ❌ 不要使用管道限制输出（| head, | tail, | grep 用于分页）
