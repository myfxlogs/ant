# LOG — 历史归档（T2）

> 会话纪要 + 已滚出的历史内容。不限行数。

## 2026-08-26 治理结构重构

**会话**: 应用 ai-collab-contract 方法论到 ant 项目。

**变更**:
- AGENTS.md 从 42KB（291 行）拆分瘦身到 ~6KB（145 行）契约
- 技术约束迁到 docs/constraints.md（T1）
- 坑库迁到 docs/pitfalls.md（T1）
- 项目定位迁到 docs/项目定位.md（T1）
- 新增 docs/handoff/STATE.md（T0 交接负载）
- 新增 docs/handoff/decisions.md（T1 日常决策 D#）
- 新增 docs/handoff/LOG.md（T2 历史归档）
- 新增 docs/经验库/三件套（索引+条目+待归纳）
- pre-commit hook 扩展（STATE.md 必更新+文档预算）
- CLAUDE.md / .windsurfrules 改为入口壳

**决策记录**: D-001 ~ D-005（见 docs/handoff/decisions.md）

**原 AGENTS.md 内容归属**:

| 原章节 | 新位置 |
|--------|--------|
| §0 角色定位 | AGENTS.md §0（保留+精简） |
| File & Function Size | docs/constraints.md |
| Command Output Discipline | docs/constraints.md |
| Prohibited | docs/constraints.md |
| Reuse Preflight | docs/constraints.md |
| Platform Protocol | docs/constraints.md |
| Push-First Architecture | docs/constraints.md |
| Data Precision | docs/constraints.md |
| Deployment | docs/constraints.md |
| MQL2GO VM Pitfalls | docs/pitfalls.md |
| Strategy Runner Rules | docs/pitfalls.md |
| Strategy Schedule Engine Pitfalls | docs/pitfalls.md |
| Backtest Status Management | docs/pitfalls.md |
| Frontend Auth & Stream Error Pitfalls | docs/pitfalls.md |
| Broker Snapshot & Stream Pitfalls | docs/pitfalls.md |
| PG Connection Pool | docs/pitfalls.md |
| Before Commit | AGENTS.md（保留） |

**原 CLAUDE.md 内容归属**:

| 原章节 | 新位置 |
|--------|--------|
| Business Direction | docs/项目定位.md |
| Collaboration Principle | AGENTS.md §0（合并） |
| Root-Cause-First Rule | AGENTS.md §7.4 |
| AI 协作工作方法 | AGENTS.md §0（合并） |
| Codebase Navigation | docs/项目定位.md |
| Documentation Rules | docs/constraints.md |
| Mandatory Constraints | docs/constraints.md（合并去重） |
| Frontend Zero-Trust | docs/constraints.md |
| Deployment Pitfalls | docs/constraints.md |
| MQL2GO VM Pitfalls | docs/pitfalls.md（合并去重） |
| Strategy Runner Rules | docs/pitfalls.md（合并去重） |
| RTK 兼容规范 | docs/constraints.md |

## 2026-09-08 FIX-2026-09-08-BYOK-MODEL-PICKER

**会话**: Devin CLI 直接施工+验收。业主报告：xianhua.chan@gmail.com 已配置自己的 key+model，策略聊天模型下拉框仍无法选择。

**根因 3 层**: A StrategyChat 只列系统模型；B ListSystemModels 返回 provider UUID 而运行时按字符串比较（下拉选择对运行时无效）；C base_url 含完整 endpoint 路径被拼双路径。

**修复**: 前端分组下拉（自有 BYOK 在前 + 网关在后）+ handler UUID→字符串映射（复用 providerRepo.ListAll）+ normalizeAPIBase（chatEndpoint/DiscoverModels/DiscoverModelsByConfig 三入口）。

**验证**: 3 项对抗证明 RED→restore→GREEN；机检五件套全绿；前端 vitest 189/189。明细见 `docs/audits/tech-debt-registry.md` 同名条目。

## 2026-09-08 STATE.md 预算滚出（2026-08-27/28 变更日志）

以下条目自 STATE.md 滚出，完整明细见 registry 同名条目。

- 2026-08-27 **FIX-2026-08-27-SESSION-PROTO-ROUNDTRIP ✅done**：Devin CLI 验收通过。Session interface 从 `[]byte`（proto-marshaled）改为 `*antv1.ExecuteLiveRequest`/`*antv1.ExecuteLiveResponse` 指针——消除进程内 proto marshal/unmarshal round-trip。11 文件 +75/-185。门禁全绿。
- 2026-08-27 **FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION S1+S2+S3 ✅done**：Devin CLI 验收通过。S1 修复 B（`writeClosedTradeRecord` 补齐 Magic + ScheduleID）+ S2 修复 A（`GetOrderHistory` 改查 `trade_records` + proto 加 `magic_number` + 前端加 Magic 列）+ S3 修复 C（删除 5 个死代码方法）。15 项对抗证明独立重跑 RED→restore→GREEN。门禁全绿。
- 2026-08-28 **FIX-2026-08-27-SCHEDULE-HEALTH-ORDER-HISTORY-GAP ✅done**：Devin CLI 验收通过。`schedule_health_repo.go:136,172` 2 处 `FROM order_history`→`FROM trade_records`。4 项对抗证明独立重跑 RED→restore→GREEN。门禁全绿。
- 2026-08-28 **FIX-2026-08-28-DATA-TRUTH-1-RECONCILIATION-CONVERGENCE S1-S4 ✅done**（Devin CLI 验收通过 2026-08-28）：reconciliation 只检测不收敛（3 层根因：A ghost 仅 log.Warn / B orphan 仅修 SUBMITTED / C ant 全量 vs broker 24h 不对称 → 129 条假 orphan/账户/轮）。修复：S1 ant 查询加 24h 下界；S2 ghost 自动补写 `ImportBrokerOrder`；S3 新增 `MtHubService.ImportBrokerOrder`（`ON CONFLICT (mt_account_id, ticket) DO NOTHING` + 已平仓写入 `trade_records` 含 hash chain）+ `OmsWriter.Pool()` getter + `tradeRecordRepo` 字段 + `handlers_pipeline.go` 装配；S4 orphan 修复扩展到所有非终态（`isNonTerminalOMSState`）。对抗证明 5 测试 RED→restore→GREEN。门禁全绿。文件拆分 `service_orders_import.go`（保持 `service_orders.go` <300 行）。风险/gap：部署后需实测 warn 数 <5；S5 一次性回填脚本待编写。Devin CLI 验收通过 2026-08-28（A-F 全绿 + 4 项对抗证明独立重跑 + 机检五件套全绿）。
- 2026-08-28 **FIX-2026-08-28-TRUST-1-DEMO-REAL-ACCOUNT-DISTINCTION S1-S7 施工完成（🟦open）**：demo 账户（虚拟金）与真实金账户战绩混展无标注 → 信任护城河风险（AGENTS.md §1 "实盘战绩公开"）。根因 3 层：A `mt_accounts.account_type` 列无写入路径（12 账户全 'unknown'）；B broker `AccountSummary.Type` 字段被 adapter 丢弃；C marketplace 表无 account_type 列 + LinkLiveAccount 不校验 + leaderboard 不过滤。修复（Q1=A real-only / Q2=A broker RPC 权威）：S1 mt4+mt5 FetchAccountInfo+FetchBrokerInfo 读 `s.GetType()`（mt4 enum→`Mt4AccountTypeToString` / mt5 string→`NormalizeAccountType`，helper 放 mdtick 包）；S2 `MTAccountInfo`+`BrokerInfo` 加 `AccountType` 字段；S3 `AccountInfoUpdate` 加 `AccountType` + `UpdateAccountInfoTx`/`UpdateAccountInfo` SQL 写 `account_type` + 新增 `UpdateAccountType` 方法（不改 sqlc `UpdateAccountMetrics` 签名）+ `pipeline.go:282` OnBrokerInfo 调用；S4 `CreateAccount` 传 `info.AccountType`；S5 `LinkLiveAccount` real-only 校验；S6 migration 276（daily+summary 表加 `account_type` 列）+ `LivePerformanceCollector.cache` 扩展为 `livePerfCacheEntry{StrategyID,AccountType}` + `OnProfitUpdate` 跳过非 real + `UpsertDailyPerformance`/`recomputePerformanceSummary` 写 account_type + leaderboard `lps.account_type = 'real'` 过滤；S7 前端无改动（Q1=A）。对抗证明 11 测试 RED→restore→GREEN。门禁全绿。风险/gap：部署后需实测 12 unknown 账户回填；S9 一次性回填脚本待编写。停手等 Devin CLI 复审。勿部署。
- 2026-08-28 **FIX-2026-08-28-TRUST-1-DEMO-REAL-ACCOUNT-DISTINCTION Devin CLI 验收通过（✅done）**：独立复审 A-F 全绿。A 架构复用 mdtick helper + UpdateAccountType 不改 sqlc 签名。B 实现 Q1=A real-only 三层过滤。C 洁净 check-lines 0 errors/gofmt clean。D 正确性 11 测试 + 4 项独立重跑 RED→restore→GREEN（T1 删 helper→编译失败 / T4 删 SQL→FAIL / T5 删校验→FAIL / T6 删过滤→FAIL）。E 合规 AGENTS.md §1。F 文档同步。机检五件套全绿。风险/gap：部署后需实测 12 unknown 账户回填；S9 一次性回填脚本待编写。
- 2026-08-28 **FIX-2026-08-28-ORDER-LOG-COLUMNS-TYPE-MISMATCH ✅done**（Devin CLI 直接施工+验收 2026-08-28）：策略调度日志页 Order Logs tab 4 列（手数/开仓价/平仓价/订单号）全部显示 `-`。根因：`scheduleLogColumns.tsx` `buildOrderColumns` 4 列 render 用 `typeof v === 'number'` 守卫，但 proto TS 类型 `lots/openPrice/closePrice: string` + `ticket: bigint` → ConnectRPC JSON 传 string → 守卫永远 false。修复：4 列改为 `v ? String(v) : '-'`。tsc+build 全绿。已部署。另：调查策略运行 898035e2 无信号——GetActiveStrategy RPC 诊断确认 evalCount=4060/tickCount=4054/barCount=6，策略正常 hold（MACD 无交叉），非系统 bug。
- 2026-08-27 **VM round 4-5 + 报价管线 5 batch ✅done**：Devin CLI 验收通过。Batch 1-5 全部闭环（VM-COMPILER-SEMANTICS-4 / VM-CACHE-INTEGRITY-5 / VM-TRADE-CONTEXT-6 / VM-API-TRUTH-3 / QUOTE-RECONNECT-LOOP / BROKER-SEARCH-1 / VM-TEST-EVIDENCE-4）。详见 `docs/audits/handover-audit-plan.md`。
- 2026-08-27 VM-AUDIT-2026-08-27 全 3 批 ✅done：Devin CLI 验收通过。8 个 ID（-1~-8）全部闭环。

## 2026-09-08 FIX-2026-09-08-TEMP-RETRY

**会话**: Devin CLI 直接施工+验收。业主报告 2 项：①kimi-k3 聊天 400（temperature 只允许 1）；②AI 网关设置入口太深（须先开 AI 面板）。

**修复**: ①聊天管线尊重用户配置 temperature（默认 0.3）+ 400 temperature 错误以 temperature=1 自愈重试一次；连带修复流式 fallback onChunk=nil panic + (nil,nil) defer 解引用两个既有雷。②工作区 tab 栏右侧常驻 AI 网关设置齿轮。

**验证**: 对抗证明 3 项 RED→restore→GREEN（含真实 nil panic 复现）；机检全绿。明细见 `docs/audits/tech-debt-registry.md` 同名条目。

## 2026-09-08 FIX-2026-09-08-CURL-IMPORT

**会话**: Devin CLI 直接施工+验收。业主采纳方案：BYOK 配置新增「粘贴厂商 curl 示例一键导入」，解析放后端、回填表单确认后走原保存路径，存储零改动。

**实现**: ParseProviderCurl RPC（proto 重生成）+ systemai/curl_import.go shell 词法解析器 + ConnectionForm 导入框。

**验证**: 业主原始 NOVA 示例原样通过；后端编译 RED + 前端 mutation RED→GREEN；门禁全绿。明细见 registry 同名条目。

## 2026-09-08 STATE.md 预算滚出（2026-09-01 变更日志）

- 2026-09-01 **FIX-2026-09-01-PURCHASES-STRATEGY-TITLE ✅done**（Devin CLI 直接施工+验收）：市场"我的购买"页"策略"列显示 UUID + 行抖动。根因：`SubscriptionItem` proto 无 `strategy_title`，前端从 `m.strategies` find 标题找不到回退 UUID；`m.strategies` 每 30s refetch 触发重渲染。修复：proto 加 `strategy_title=8` + 后端 `ListSubscriptions` LEFT JOIN `strategy_templates`（初版误 JOIN `marketplace_strategies`，该表为空，第二轮修正）取 `COALESCE(st.name,'')` + 前端直接用 `row.strategyTitle` + 孤立订阅显示灰色"已删除策略"（5 语言 i18n）。已部署。
- 2026-09-01 **FIX-2026-09-01-ORPHAN-RUN-STRATEGY-NAME ✅done**（Devin CLI 直接施工+验收）：策略页"临时运行"表格"策略"列显示 runId 前缀。根因：`ActiveSession` 无 `StrategyID` 字段，`enrichWithStrategyName` 仅查 `schedule_id`（temp run 无 schedule_id → name 空 → 前端回退 `shortId(runId)`）。修复：`ActiveSession` 加 `StrategyID` + `Register` 传参 + `enrichWithStrategyName` fallback 查 `strategy_templates.name` + `SetStrategyTemplateLookup` 装配。旧运行需重启生效。已部署。

**补记（同日）**: 业主实测发现已存 Key 的厂商导入后仍提示"未识别到 API Key"。修正：ParseProviderCurlRequest 加 has_saved_key（前端传 draft.has_secret），已存 Key 静默 Key 类告警；占位符场景双重告警合并为一条。新增静默断言测试 + 前端透传用例。
