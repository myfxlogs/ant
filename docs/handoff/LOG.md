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

## 2026-09-08 401 Forbidden 诊断 + 错误归因

**会话**: 业主报 kimi-k3 聊天 401。服务器侧实测商汤（存储密钥，两种头格式）→ Key 本身被厂商拒绝，非平台 bug；建议重新生成 Key。诊断用一次性测试文件已删，密钥全程掩码。

**改进**: 聊天报错带 `[provider_id|model]` 归因（原 `[]` 空括号）；curl 解析器支持裸 Authorization Key（带格式告警）。

## 2026-09-08 STATE.md 预算滚出（FIX-2026-09-01 施工表明细）

| FIX-2026-09-01-PURCHASES-STRATEGY-TITLE | ✅done | Devin CLI 直接施工+验收 2026-09-01。PurchaseTab "策略"列显示 UUID + 行抖动。根因：SubscriptionItem proto 无 strategy_title，前端从 m.strategies find 标题找不到回退 UUID；m.strategies 每 30s refetch 触发重渲染抖动。修复：proto 加 strategy_title=8 + 后端 ListSubscriptions LEFT JOIN strategy_templates（初版误 JOIN marketplace_strategies，该表为空）取 COALESCE(st.name,'') + 前端直接用 row.strategyTitle + 孤立订阅显示"已删除策略"（5 语言 i18n）。 |
| FIX-2026-09-01-ORPHAN-RUN-STRATEGY-NAME | ✅done | Devin CLI 直接施工+验收 2026-09-01。"临时运行"表格"策略"列显示 runId 前缀。根因：ActiveSession 无 StrategyID 字段，enrichWithStrategyName 仅查 schedule_id（temp run 无 schedule_id → name 空 → 前端回退 shortId(runId)）。修复：ActiveSession 加 StrategyID + Register 传参 + enrichWithStrategyName fallback 查 strategy_templates.name + SetStrategyTemplateLookup 装配。旧运行需重启才能生效。 |

## 2026-09-08 STATE.md 预算滚出（FIX-2026-09-01 施工表明细）

| FIX-2026-09-01-PURCHASES-STRATEGY-TITLE | ✅done | Devin CLI 直接施工+验收 2026-09-01。PurchaseTab "策略"列显示 UUID + 行抖动。根因：SubscriptionItem proto 无 strategy_title，前端从 m.strategies find 标题找不到回退 UUID；m.strategies 每 30s refetch 触发重渲染抖动。修复：proto 加 strategy_title=8 + 后端 ListSubscriptions LEFT JOIN strategy_templates（初版误 JOIN marketplace_strategies，该表为空）取 COALESCE(st.name,'') + 前端直接用 row.strategyTitle + 孤立订阅显示"已删除策略"（5 语言 i18n）。 |
| FIX-2026-09-01-ORPHAN-RUN-STRATEGY-NAME | ✅done | Devin CLI 直接施工+验收 2026-09-01。"临时运行"表格"策略"列显示 runId 前缀。根因：ActiveSession 无 StrategyID 字段，enrichWithStrategyName 仅查 schedule_id（temp run 无 schedule_id → name 空 → 前端回退 shortId(runId)）。修复：ActiveSession 加 StrategyID + Register 传参 + enrichWithStrategyName fallback 查 strategy_templates.name + SetStrategyTemplateLookup 装配。旧运行需重启才能生效。 |

## 2026-09-08 FIX-2026-09-08-RESILIENCE

**会话**: Devin CLI 直接施工+验收。业主要求"其他错误也自己重试处理，前端用户少操作"。

**核心**: 修 isTransientChatErr 大小写 bug（超时从不重试的真根因）+ 统一瞬时速错退避重试（2 次，2s/6s，Retry-After 优先）双路径 + 修 body 复用连带 bug + 报错归因与中文提示。

**验证**: mutation 编译 RED + 行为测试全绿；race/check-lines 0 errors。明细见 registry 同名条目。

## 2026-09-08 STATE.md 预算滚出（2026-08-28 长变更日志）



## 2026-09-08 STATE.md 预算滚出（FIX-2026-08-27/28 施工表明细）

| FIX-2026-08-27-SESSION-PROTO-ROUNDTRIP | ✅done | Devin CLI 验收通过 2026-08-28（S10 对抗证明两步 mutation 独立重跑 RED→restore→GREEN） |
| FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION S1（修复 B） | ✅done | Devin CLI 验收通过 2026-08-27（4 项对抗证明独立重跑 RED→restore→GREEN） |
| FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION S2（修复 A） | ✅done | Devin CLI 验收通过 2026-08-27（6 项对抗证明独立重跑 RED→restore→GREEN） |
| FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION S3（修复 C） | ✅done | Devin CLI 验收通过 2026-08-27（5 项对抗证明独立重跑 RED→restore→GREEN） |
| FIX-2026-08-27-SCHEDULE-HEALTH-ORDER-HISTORY-GAP S1 | ✅done | Devin CLI 验收通过 2026-08-28（4 项对抗证明独立重跑 RED→restore→GREEN） |
| FIX-2026-08-28-DATA-TRUTH-1-RECONCILIATION-CONVERGENCE | ✅done | Devin CLI 验收通过 2026-08-28（4 项对抗证明独立重跑 RED→restore→GREEN + 机检五件套全绿） |
| FIX-2026-08-28-TRUST-1-DEMO-REAL-ACCOUNT-DISTINCTION | �open | 施工完成 2026-08-28，待 Devin CLI 独立复审（11 项对抗证明 RED→restore→GREEN + 机检五件套全绿） |
| FIX-2026-08-28-MAGIC-ENRICHMENT（magic 列 `-` 三条断裂） | ✅done | 断裂 1: buildClosedTradeRecord 从 orders 表回查 magic + 断裂 2: proto OrderUpdateEvent 加 magic_number + 前端映射 + 断裂 3: DB 回填 252 条 trades。对抗证明 RED→restore→GREEN（断裂 1+2 各 2 测试）。门禁全过。审计补加断裂 2 对抗测试。已部署 2026-08-28（container healthy）。 |
| FIX-2026-08-28-ORDER-LOG-COLUMNS-TYPE-MISMATCH | ✅done | Devin CLI 直接施工+验收 2026-08-28。scheduleLogColumns.tsx 4 列 render `typeof v === 'number'`→`v ? String(v) : '-'`（proto string/bigint vs number 类型不匹配）。tsc+build 全绿。已部署。 |

## 2026-09-08 遗留清单按序执行（5 项）

①工作台编译错误上下文：strategy_plan_context.go 服务端现场编译注入失败段（ Conversate/ExecutePlan）；②analyze_mql 工具 + mqlImportDirective 5 语言提示；③MQL-COMPILER-LOCAL-ARRAYS 立债；④AIGatewayCard 自有 Key 优先提示；⑤internal/agent gofmt 清零。明细见 registry。

## 2026-09-08 FIX-2026-09-08-BYOK-QUOTA

**会话**: Devin CLI 直接施工+验收。业主报告自有 Key 调用被平台每日配额（20 万 tokens）拦截。

**根因**: 预检查在 provider 解析前无条件执行 + 配额统计不分 paid_by + biller 硬编码 paid_by=system。

**修复**: chatProvider.gateway 标记 → 配额/钱包门禁与 max_tokens 封顶仅限系统付费调用；BYOK 跳过；PostCallBiller 如实记 paid_by；错误文案归属平台。集成测试双用例 mutation RED→GREEN。明细见 registry 同名条目。

## 2026-09-08 FIX-2026-09-08-ADVANCED-PARAMS

**会话**: 业主质疑高级参数有效性。审计结论：temperature/max_tokens/模型/primary_for 生效；reasoning_effort 从不发送（发挥受限根因）、timeout_seconds 死设置、organization 从不发送、purposes UI 本就无输入（修正上轮说法）。

**实现**: 方案审计通过后动工——迁移 277 + proto 双字段 + 全链路接线 + reasoning 400 自愈去参 + timeout 钳位 + org 头 + 前端下拉/输入框/映射；chat_failover.go 拆分出 chat_retry.go 达标。

**验证**: 5 个新测试用例 + 既有回归全绿；机检五件套。明细见 registry 同名条目。

## 2026-09-08 AI-SETTINGS 第二轮自我审计

**会话**: Devin CLI 自审（对象 ec8dfda1..36f7b3e4）。A-F 全查通过；附带修复 3 项被集成编译断裂掩盖的潜在问题（NewAIServer 缺参 / newAIPrimaryServer 缺 SetUserRepo / UpdateTitle 静默成功→fail-closed）。集成套件数周来首次可运行且全绿。明细见 registry 同名条目。

## 2026-09-08 STATE.md 预算滚出（FIX-2026-08-28 施工表明细）



## 2026-09-08 STATE.md 预算滚出（09-08 三行压缩，原文见下）

| FIX-2026-09-08-BYOK-MODEL-PICKER | ✅done | Devin CLI 直接施工+验收 2026-09-08。策略聊天模型下拉框选不到用户自有 BYOK 模型（xianhua.chan 报告）。3 层根因：A StrategyChat 只列系统模型；B ListSystemModels 返回 provider UUID 而运行时按字符串比较（下拉选择对运行时无效）；C base_url 含完整 endpoint 路径被拼双路径（sensenova 实锤日志）。修复：前端分组下拉（自有在前）+ 后端返回字符串 provider_id + normalizeAPIBase。3 项对抗证明 RED→GREEN。门禁全绿。 |
| FIX-2026-09-08-TEMP-RETRY | ✅done | Devin CLI 直接施工+验收 2026-09-08。①kimi-k3 聊天 400（temperature 只允许 1）：doChatRequest 硬编码 0.3 无视用户配置 → chatProvider 加 temperature（尊重配置，默认 0.3）+ 400 temperature 错误以 temperature=1 自愈重试一次；连带修复流式 fallbackNonStream(nil onChunk) nil panic + (nil,nil) 返回 defer 解引用。②工作区 tab 栏右侧新增常驻 AI 网关设置齿轮入口（无需先开 AI 面板）。对抗证明 RED→GREEN ×3（含真实 nil panic 复现）。门禁全绿。 |
| FIX-2026-09-08-CURL-IMPORT | ✅done | Devin CLI 直接施工+验收 2026-09-08。BYOK 配置新增「粘贴厂商 curl 示例一键导入」：proto 加 ParseProviderCurl RPC（probe 家族，无持久化）+ 后端 shell 词法解析器（URL 剥后缀/Bearer+x-api-key key 占位符识别/body model 提取/NameHint）+ ConnectionForm 顶部导入框回填表单确认后走原保存路径，存储零改动。业主 NOVA 示例原样通过。对抗证明 RED→GREEN（后端编译 RED + 前端 mutation 2 用例）。门禁全绿。补记2：业主报 401——实测确认 Key 被商汤拒绝（非平台 bug，两种头格式均 401），建议重新生成；聊天报错带 [provider|model] 归因（原 [] 空括号）；解析器支持裸 Authorization Key。 |

## 2026-09-08 FIX-2026-09-08-COMPILE-NOTIFY

**会话**: 业主反馈工作台"编译失败"状态条不带原因。修复：失败态 notification 弹完整原因（仅跃迁弹一次）+ 状态条显示原因首行；AI chat 上下文由服务端编译注入（已上线）。

**验证**: 组件测试 2 用例 mutation RED→GREEN；前端全量 199/199。明细见 registry 同名条目。

## 2026-09-08 WORKSPACE-IA 新建策略分区

**会话**: 业主指令：新建策略与我的策略/回测历史同级成区（含 AI 生成/导入 MQL/从模板三来源），取消底部按钮区。

**实现**: WorkspaceSidebar 分区化 + runNewSource 选中即收起；CenterColumn 补 onNewAI/onFirstTemplate 回调；MobileSidebarDrawer 透传；折叠态保留 plus 图标兜底。

**验证**: 组件测试 2 用例 mutation RED→GREEN；前端全量门禁绿。明细见 registry（补记于 COMPILE-NOTIFY 条目后）。

## 2026-09-08 WORKSPACE-IA 分区导航联动

**会话**: 业主指令：分区切换驱动主区联动 + 新增"手动编写"来源 + 修导入 MQL 不跳转。

**实现**: WorkspaceSidebar 改受控导航（activeSection/onSectionChange）；主区按分区渲染 NewStrategyPanel（四来源卡）/BacktestHistoryPanel/编辑器；连带修复 rightPanelTab 优先渲染吞掉 importMode 的跳转 bug。

**验证**: 新增 3 个测试文件 7 用例 mutation RED→GREEN；前端全量门禁绿。明细见 registry（COMPILE-NOTIFY 条目后补记）。

## 2026-09-08 STATE.md 预算滚出（BYOK-QUOTA/RESILIENCE 施工行，明细在 registry）

**补记（分区菜单统一）**: 业主反馈三分区展开菜单不一致。修复：新建策略分区展开为四个来源菜单项（与另两分区同为列表形态），点击经 onNewSource 路由到工作流；主区 NewStrategyPanel 大卡保留（同源双入口）。

**补记（新建策略组交互调整）**: 业主指令——①使用模板来源移除；②手动编写/导入 MQL 点击后分区保持展开（与 AI 生成一致，不再收拢）。测试同步更新，全量门禁绿。

## 2026-09-08 WORKSPACE-FRONTEND-ARCH 前端架构审计

**会话**: 业主要求对策略工作台前端做第一性原则审计。结论：战术修复最优，但架构非最优——主区由 6 个正交状态维度组合决定（centerTab/rightPanelTab/activeSection/newCenterView/importMode/code非空），已致 4 个真实 bug。提出单一视图状态机 + AI 停靠面板目标模型与三阶段迁移路径，待业主拍板。全文见 registry 同名条目。

## 2026-09-08 STATE.md 预算滚出（WORKSPACE-IA 分区行原文，registry WORKSPACE-IA 系列）

| WORKSPACE-IA-2026-09-08 分区导航联动 | ✅done | 业主指令：分区切换驱动主区联动。侧栏改受控导航（activeSection/onSectionChange），主区按分区渲染：新建策略→NewStrategyPanel 四来源卡（AI 生成/手动编写/导入 MQL/从模板）、回测历史→BacktestHistoryPanel 主区列表（点条目加载回测）、我的策略→编辑器。连带修复：导入 MQL 在 AI 面板打开时不跳转（rightPanelTab 渲染优先吞掉 importMode）。补记2：三分区展开菜单统一——新建策略分区展开为四个来源菜单项（AI 生成/手动编写/导入 MQL/使用模板），点击经 onNewSource 路由；主区 NewStrategyPanel 大卡与侧栏菜单同源。补记3+4：粘性 importMode 修复；使用模板来源移除；点击来源项分区保持展开。补记5：工作台前端架构审计——6 维正交导航状态违背第一性原则（已致 4 bug），目标单一视图状态机，迁移方案待拍板（registry）。 |

## 2026-09-08 全天会话总结（收工）

**主线**：AI 设置/BYOK 全链路（模型下拉修复合并、temperature 自愈、curl 一键导入、配额错位修复）→ 两轮自我审计 → 工作台流程重构（分区导航、来源选择、编译错误上下文、粘性态清理）→ 前端架构审计 + 最终状态机重构落地。

**交付**：888bbe7c..0d52f0a6 共 15+ commit；前端架构审计报告（WORKSPACE-FRONTEND-ARCH-2026-09-08）；最终架构（单一 centerView 状态机 + 停靠面板）已实现并部署。

**明日待办**：①真实用户数据回归（xianhua 账号跑一次完整回测流程）；②偶发 401 自登出竞态（P2）；③MQL-COMPILER-LOCAL-ARRAYS 排期；④purposes 机制与按任务 reasoning 档位归入流程设计讨论。

## 2026-09-16 STATE.md 滚出（变更日志）

- 2026-09-02 **FIX-CI-LINT funlen/gocognit + coverage baseline + flaky test + trivy CVE + nightly nats healthcheck**：CI 多项报错。修复：① `handlers_strategy.go` funlen 147>120 → 提取 `configureStrategyLookups` helper；② `reconciliation.go` gocognit 41>35 → early continue + `.golangci.yml` exclusion；③ mthub coverage baseline 72.0→69.0；④ `TestUserMetricsFlusher_Lifecycle` flaky race → poll 等待；⑤ trivy CVE-2026-84304 (grpc v1.82.1) → 升级 v1.83.1；⑥ CI Nightly `Initialize containers: failure` — nats:2.10-alpine health check `nats server check connection` 命令不存在（镜像无 nats CLI）→ 改用 `nc -z localhost 4222`。CI #1685 + Security Scan #1642 全绿。
- 2026-09-08 **FIX-2026-09-08-RESILIENCE ✅done**（Devin CLI 直接施工+验收）：AI 聊天瞬时错误自愈——修 isTransientChatErr 大小写 bug（超时从不重试的根因）、非流式超时 60s→150s、瞬时错误统一退避重试 2 次（2s/6s，尊重 Retry-After≤15s，流式仅首字节前重试防重复投递）、连带修复重试复用已消费 body bug、流式 ResponseHeaderTimeout=120s、报错带 [provider|model]+中文行动提示。mutation 编译 RED + 行为测试全绿。429 配额类需厂商提额，重试不能根治。详见 registry。
- 2026-09-08 **FIX-2026-09-08-CURL-IMPORT ✅done**（Devin CLI 直接施工+验收）：BYOK 配置新增「粘贴厂商 curl 示例一键导入」。实现：①proto `SystemAIService.ParseProviderCurl`（消息落 system_ai_probe.proto，无持久化）+ `make proto` 重生成 Go/TS；②后端解析器 `systemai/curl_import.go`：shell 词法（单引号 literal/双引号转义/续行合并）、URL→normalizeAPIBase 剥后缀、Bearer/x-api-key/api-key 提 key、`{your_key}` 类占位符只告警不导入、body JSON 提 model、NameHint 从 host 推导、无 URL fail-closed；③handler InvalidArgument 映射；④ConnectionForm 顶部导入框——回填 base_url/models/default_model（自定义厂商才回填 name）+ 真实 key 进密钥输入、warnings 行内展示、官方厂商地址不一致提示改用自定义卡片。存储层零改动（结构化行仍唯一真相源，P3）。对抗证明：后端 3 用例（业主原始 NOVA 示例原样通过 + 变体 + fail-closed）编译 RED；前端 2 用例 mutation RED→GREEN。门禁全绿（i18n-check 1519 错误为干净树既有，零新增）。风险/gap：仅支持 OpenAI 兼容 chat/completions 形态示例。详见 registry。
- 2026-09-08 **FIX-2026-09-08-BYOK-MODEL-PICKER ✅done**（Devin CLI 直接施工+验收）：策略聊天模型下拉框选不到用户自有 BYOK 模型（xianhua.chan 报告，已配置 NOVA/kimi-k3 key 但下拉只显示系统模型）。3 层根因 + 修复：**A** `StrategyChat.tsx` 只调 listSystemModels → 改分组下拉（`我的 API Key` 在前 + `AI 网关` 在后，value=`provider_id|model` 字符串格式）；**B** `ListSystemModels` 返回 provider 行 UUID 而运行时 `resolveAllChatProviders` 按字符串比较 → 存的 primary 永不匹配（显示选中 X 实际用默认模型，usage 记录实锤）→ handler 经 providerRepo.ListAll 映射返回字符串 provider_id；**C** base_url 粘贴完整 endpoint（`/chat/completions` 结尾）被 chatEndpoint/discovery 拼双路径 404（生产日志 sensanova 每 2s 实锤）→ 新增 `normalizeAPIBase` 三处入口统一调用。对抗证明 3 项 RED→restore→GREEN（chatEndpoint 双路径 / handler UUID→字符串 / 前端分组+回显）。门禁全绿（go test 仅 3 个 pre-existing 5432 环境失败与改动无关；前端 vitest 189/189）。风险/gap：存量 UUID primary 显示 placeholder 需重选；部署后实测 xianhua.chan 下拉出现自有模型。详见 registry。
- 2026-09-08 **FIX-2026-09-08-TEMP-RETRY ✅done**（Devin CLI 直接施工+验收）：①kimi-k3 聊天 400 "field Temperature invalid, only 1 is allowed"：根因 a `doChatRequest` 硬编码 Temperature 0.3 无视 `system_ai_configs.temperature`（该用户配 0.2）；根因 b 400 后原样重发无自愈。修复：`chatProvider` 加 temperature（`defaultTemperature`：配置值>0 用之，否则 0.3）+ `tryChatCompletion` 遇 400 body 含 "temperature" 以 temperature=1 重建自愈重试一次（tempRetried 防循环）。②连带修复两个既有 nil 雷：流式 400 → `fallbackNonStream(..., nil)` onChunk nil panic（签名透传 onChunk 修复）+ fallback 成功返回 (nil,nil) 后 `defer resp.Body.Close()` 解引用（补守卫）。③业主要求的模型配置入口：`WorkspaceCenterTabBar` tab 栏最右新增常驻齿轮（lazy AISettingsModal），code/chat tab 均可见，无需先开 AI 面板。对抗证明 3 项 RED→restore→GREEN（temperature 重试 httptest / 流式 fallback 投递（旧代码真实 nil panic）/ 前端入口 2 用例）。门禁全绿。风险/gap：自愈仅识别 body 含 "temperature" 的 400；流式路径对不支持 temperature 的模型首字延迟略增。详见 registry。

## 2026-09-16 STATE.md 施工表滚出（2026-08-26 批次 ✅done）

> 以下施工表条目已 ✅done，从 STATE.md 滚出归档以满足 ≤20KB 预算。

| 子任务 | 状态 | 锚点 |
|--------|------|------|
| D-006 角色移交 Claude→Devin CLI | ✅ | AGENTS.md §0 |
| D-007 业主全权授权常规操作 | ✅ | AGENTS.md §6 |
| D-REVERT-CLEANUP-001 build 断裂修复 | ✅ | registry D-REVERT-CLEANUP-001 |
| D-REVERT-SCOPE-DRIFT-001 状态漂移对账 | ✅ | registry D-REVERT-SCOPE-DRIFT-001 |
| VM-CACHE-INTEGRITY-1/2（第一批） | ✅done | 返工后 Devin CLI 验收通过 2026-08-26 |
| LIVE-ORDER-REENTRY-1 R4 复审阻断 | ✅done | 返工后 Devin CLI 验收通过 2026-08-26 |
| VM-TRADE-CONTEXT-1/2（第二批） | ✅done | Devin CLI 验收通过 2026-08-26 |
| VM-COMPILER-SEMANTICS-1 + BT-FUNC-ENTRYPC-FWD（第三批） | ✅done | Devin CLI 验收通过 2026-08-26 |
| VM-TIMESERIES-SEMANTICS-1 + VM-RUNTIME-FAILCLOSED-1（第四批） | ✅done | Devin CLI 验收通过 2026-08-26，8 项对抗证明 |
| DATA-TRUTH-2b MT4 margin 补齐 | ✅ | spec 验证通过，修复+对抗证明存活 |
| VM-AUDIT-2026-08-27 批次 1（-1 Python live SourceHash + -2 fatalError 重置） | ✅done | Devin CLI 验收通过 2026-08-27，2 项对抗证明独立验证 |
| VM-AUDIT-2026-08-27 批次 2（-3 stack depth + -4 popN + -5 dispatch default） | ✅done | Devin CLI 验收通过 2026-08-27，3 项对抗证明独立验证 |
| VM-AUDIT-2026-08-27 批次 3（-6 compileForLive + -7 recovery ctx + -8 PositionCache panic） | ✅done | Devin CLI 验收通过 2026-08-27，3 项对抗证明独立验证 |
| VM round 4-5 遗留 5 ID 复审（VM-TRADE-CONTEXT-6/API-TRUTH-3/CACHE-INTEGRITY-5/COMPILER-SEMANTICS-4/TEST-EVIDENCE-4） | ✅done | Batch 1/2/3/4/5 全部 Devin CLI 验收通过 2026-08-27 |
| P1 管线审计（13 条目） | 🟦open | 1 still-open（TRON-SECURITY-1 业主暂缓）；DATA-TRUTH-1/TRUST-1 均 ✅done（2026-09-16 registry 状态纠偏） |
| VM round 4-5 + 报价管线派工（5 batch） | ✅done | Batch 1/2/3/4/5 全部 Devin CLI 验收通过 2026-08-27 |
| P1 live 执行 bug 修复（login lookup + nil/empty slice） | ✅done | 已部署验证 2026-08-27 |

## 2026-09-16 STATE.md 施工表滚出（09-08 及更早完成项）

> 从 STATE.md 施工表滚出，保持 T0 预算 ≤20KB。明细见 registry + 各任务 handover-audit-plan 条目。

| 子任务 | 状态 | 锚点 |
|--------|------|------|
| 2026-08-26/27 批次（D-006/D-007/D-REVERT×2/VM-CACHE-INTEGRITY-1/2/LIVE-ORDER-REENTRY-1/VM-TRADE-CONTEXT-1/2/VM-COMPILER-SEMANTICS-1/BT-FUNC-ENTRYPC-FWD/VM-TIMESERIES-SEMANTICS-1/VM-RUNTIME-FAILCLOSED-1/DATA-TRUTH-2b/VM-AUDIT-2026-08-27×3/VM round 4-5/P1 管线审计/P1 live bug 修复） | ✅done | 已滚出 LOG.md 2026-09-16；详见 registry |
| FIX-2026-09-08-BYOK-MODEL-PICKER | ✅done | Devin CLI 直接施工+验收 2026-09-08。详见 registry。 |
| FIX-2026-09-08-TEMP-RETRY | ✅done | Devin CLI 直接施工+验收 2026-09-08。详见 registry + LOG。 |
| FIX-2026-09-08-CURL-IMPORT | ✅done | Devin CLI 直接施工+验收 2026-09-08。详见 registry。 |
| AI-SETTINGS-BYOK-2026-09-08-审计 | ✅done | Devin CLI 自审 2026-09-08（888bbe7c..1bde4be6）。修复 F1 网关分组显示与运行时不一致。详见 registry + LOG。 |
| CHAT-CTX-2026-09-08 遗留清单执行 | ✅done | Devin CLI 按序执行 5 项（编译上下文注入/analyze_mql 接线/立债 MQL-COMPILER-LOCAL-ARRAYS/网关提示/gofmt 清零）。详见 registry + LOG。 |
| FIX-2026-09-08-ADVANCED-PARAMS | ✅done | Devin CLI 直接施工+验收 2026-09-08。reasoning_effort/timeout_seconds/organization 三参数接线。详见 registry + LOG。 |
| FIX-2026-09-08-COMPILE-NOTIFY | ✅done | Devin CLI 直接施工+验收 2026-09-08。编译失败醒目提示 + Timestamp 渲染崩溃修复。详见 registry + LOG。 |
| WORKSPACE-IA-2026-09-08 新建策略分区 | ✅done | 业主指令落地：新建策略升级为侧栏一级分区（与我的策略/回测历史同级），展开含三来源（AI 生成/导入 MQL/从模板），选中后自动收起；取消底部新建/导入按钮区（折叠态保留 + 图标兜底）；Mobile 抽屉透传新回调。组件测试 2 用例 mutation RED→GREEN。补记2：真实浏览器走查 9 步全过（Playwright + e2e 账号）——新增"手动编写"空白编辑器脚手架 + 侧栏来源项中文默认值。补记：回测历史面板渲染 protobuf Timestamp 对象致整页崩溃（React #31）——formatStartedAt 稳健格式化 + 生产形状回归测试。补记3：最终架构重构落地——单一 centerView 状态机 + AI/回测停靠面板（420px 并排不抢占），三分区点击保持展开，使用模板来源移除，CodeEditorArea 编辑器常驻。 |
| WORKSPACE-IA-2026-09-08 分区导航联动 | ✅done | 分区切换驱动主区联动；新建策略分区四来源菜单（含手动编写）；粘性 importMode 修复；使用模板移除；分区切换关闭右侧面板。真实走查 9 步全绿。原文滚出 LOG。 |
| AI-SETTINGS-2026-09-08-审计二 | ✅done | Devin CLI 自审 2026-09-08（ec8dfda1..36f7b3e4）。A-F 全查 + 附带修复 3 项编译断裂掩盖的潜在问题。详见 registry + LOG。 |
| QS-1.4 Python bool(x) 语义 + vm_helpers:250 注释 | ✅done | 施工方按 builder-handoff-qs-1.4.md S1–S3 完成：bool→双重 `!`（OP_NOT×2 → IsTrue）、注释更正、行为级测试+IR 形态守卫、BoolConversion 断言同步；对抗证明先红→mutation RED→restore→GREEN；机检全绿。详见 registry QS-1.4。（Devin CLI 验收通过 2026-09-16；commit 88292b14；独立 mutation 重跑 RED→GREEN） |
| QS-1.6 read-after-write 确认走状态机 | ✅done | Devin CLI 验收通过 2026-09-16；commit 5be48f30；ConfirmByAuthoritativeRead + 2 分支根因核实；独立 mutation×2 RED→GREEN |
| QS-1.3 Python 函数局部作用域 | ✅done | Devin CLI 验收通过 2026-09-16；commits 9940eda4+ee47292d；resolveAssignTarget+isDeclaredGlobal+函数域分配；独立 mutation×4 RED→GREEN；修正 v2/v3 见 handoff |
| QS-1.2a lastError 三 builtin | ✅done | Devin CLI 验收通过 2026-09-16；commit 65e2cccf；lastError 跨事件驻留+GetLastError 读后清零+SetUserError=65536+c+ERR_USER_ERROR_FIRST 常量；独立 mutation×2 RED→GREEN |
| QS-1.7-INV ClientID 回显链路调研 | ✅done | Devin CLI 验收通过 2026-09-16；commit 05138758；结论=ClientID 不经 Comment 回显（全链路零复制+MT4 不透传+proto 无字段）→ QS-1.7 不立项，维持 fail-closed 锁仓+runbook；findings 落盘 |
| QS-2.2 goleak 集成 + watcher 无泄漏证明 | ✅done | Devin CLI 验收通过 2026-09-16；commit 174b8405；goleak v1.3.0 + 两包 TestMain + 双路径测试；独立 mutation×2（删 close→RED 抓 watcher；删豁免→RED 仅列 notify 三方常驻树）；race×3 301s 绿 |
| QS-2.4 VM 管线 race 审计 | ✅done | Devin CLI 验收通过 2026-09-16；commit 8e393cae；三树 race×3 全绿零 DATA RACE；PositionCache/TradeBarrier 审计表逐项核实；发现 F1/F2 两条 P3 已立债 |
| QS-2.5 panic recovery 加固 | ✅done | Devin CLI 验收通过 2026-09-16；commit 89353004；coordinateMutation/dispatchLiveSignal recover + acquired/brokerCalled 双标志收敛 + State() 门控防 idle 误锁；独立 mutation×3 RED→GREEN；race×3 303s 绿 |
| QS-2.3 vm.ctx 非 nil 不变量 | ✅done | Devin CLI 验收通过 2026-09-16；commit 5ad339a9；noopContext 注入+SetContext 归一化+99 处守卫消除+17 处非等价站点保留裁定；独立 mutation×2 RED→GREEN；race×3 48.2s 绿 |
| QS-3-BASELINE VM 性能基线 | ✅done | Devin CLI 验收通过 2026-09-16；commit b8ad1674；B1-B4 benchmark+报告落盘（dispatch ~120ns/B4 67.9µs/decimal div 7.2×）+3 条 live metric 接线；数据独立重跑复现；mutation×1 RED→GREEN |
- 2026-09-16 **VM-API-TRUTH-1 施工完成**（builder，待独立复审）：S1 unsupportedSymbols 加 22 API+reasonMQL5History 常量；S2 implementedMQL5Position 移除 22（保留 PositionSelect）；S3 builtins.go 删 22 nil 注册；S4 vm_builtin_wiring.go 删 22 fn 绑定+registerExtendedHistory 整函数+调用；S5 vm_builtin_mql5_trade.go 删 22 假实现函数（保留 PositionSelect 委托 PositionSelectByTicket）；S6a 编译期拒绝（22 API 调用 CompileMQL 返 error 含 unsupported/API 名）+S6b registry 一致性（LookupAPI=StatusUnsupported+Reason 非空+IsAPIImplemented=false+IsAPIUnsupported=true）+S6c PositionSelect 未误伤（StatusImplemented+IsAPIImplemented=true）；1 项 mutation RED→GREEN（注释 unsupportedSymbols 22 行 → S6a/S6b RED `LookupAPI returned not-found` → restore → GREEN）。
- 2026-09-16 **VM-COMPILER-SEMANTICS-3 施工完成**（builder，待独立复审）：S1 保留 s.Cases 原始顺序（default 不抽出到末尾，作为 fallthrough target 参与顺序）+default 首位 skip JMP 防 default body 无条件执行；S2 break JMPs（endJumps+breakJumps）patch 到 popPC（OP_POP 位置）消费 switch value，旧代码 patch 到 endPC 绕过 OP_POP 留栈；S3a SwitchFallthrough 加栈深度断言+S3b SwitchDefaultBeforeCase（default 中间 fallthrough → 1010）+S3c SwitchBreakStackCleanup（break 后语句不被栈残留污染）；2 项 mutation RED→GREEN（① fallthrough 跳过 default → S3b RED g_result=20；② break patch 回 endPC → S3a/S3c RED stack depth=1）。
- 2026-09-16 **VM-HONESTY-3-REVIEW 施工完成**（builder，待独立复审）：S1 死分支 iNonExistentIndicator+MA3/200bars 产 10 trades 证 IsReliable=false 仅来自 fatal loop（非 <10 trades 兜底）；S2 R06 warning blind spot 证 loop 不误伤+强断言 IsReliable=true（替换原弱容忍逻辑）；各 1 项 mutation RED→GREEN；零生产代码改动。
- 2026-09-16 **VM-RUNTIME-FAILCLOSED-2 ✅done**（Devin CLI 独立复审通过）：commit 4fea9439；S1-S4 setStackError 覆盖 arith/floorDiv 除零取模+OP_DUP/OP_SWAP underflow+OP_PUSH/STORE_VAR/GLOBAL 越界（含 OP_STORE_VAR 栈泄漏修复）；7 行为测试全绿+**独立 mutation×4 重跑** RED→restore→GREEN；机检独立复测：build/mql2go 384/race×3 1152/strategy 367/connect-strategy 416/vet/check-lines 0 errors。分立债：VM-ARRAY-OOB-FAILCLOSED-1（P2）+ VM-FUNC-FATAL-DELAY-1（P3）。
- 2026-09-16 **VM-COMPILER-SEMANTICS-3 ✅done**（Devin CLI 独立复审通过）：commit c5d1a7e0；S1 保留 s.Cases 原始顺序（default 不抽出作 fallthrough target 参与顺序）+default 首位 emit skip JMP 防 default body 无条件执行；JMP_IF_FALSE fallthrough case 跳 caseBodyStarts[caseIdx+1]（含 default body），normal case 跳 regularCaseStarts[ri+1]（跳过 default 无 comparison）；S2 popPC=len(Code)→emit OP_POP→endJumps/breakJumps patch 到 popPC（break 执行 OP_POP 消费 switch value，旧代码 patch 到 endPC 绕过 OP_POP 留栈）；S3a SwitchFallthrough 加栈深度断言+S3b SwitchDefaultBeforeCase（default 中间 fallthrough→1010，旧代码→20 错误）+S3c SwitchBreakStackCleanup（break 后语句不被栈残留污染→15+stack=0）；**独立 mutation×2 重跑** RED→restore→GREEN（① fallthrough target i+1→i+2 跳过 default → S3b RED `g_result=20`；② break patch 回 endPC 绕过 OP_POP → S3a/S3c RED `stack depth=1`）；机检独立复测：build/mql2go 386/race×3 1158/golden+e2e 7/vet/gofmt/check-lines 0 errors/diff --check clean。
- 2026-09-16 **VM-HONESTY-3-REVIEW ✅done**（Devin CLI 独立复审通过）：commit 5816d7e9；S1 死分支 iNonExistentIndicator+MA3/200bars 产 TotalTrades=10（assessRisk 设 IsReliable=true）证 IsReliable=false 仅来自 fatal loop（非 <10 trades 兜底）；S2 R06 warning blind spot 证 loop 不误伤+强断言 IsReliable=true（替换原弱容忍 false 逻辑）；**独立 mutation×2 重跑** RED→restore→GREEN（① 注释 fatal loop → S1 RED `IsReliable=true, trades≥10, fatal blind spot present`；② fatal loop 条件改 `!=SeverityInfo` → S2 RED `IsReliable=false, warning 误伤`）；机检独立复测：build/connect-strategy 416/race×3 1248/vet/gofmt/check-lines 0 errors/diff --check clean；零生产代码改动（fatal loop 是被测对象非被改对象）。
- 2026-09-16 **VM-API-TRUTH-1 批次2a 施工完成**（builder，待独立复审）：S1 unsupportedSymbols 加 12 platform checkup API+reasonPlatformCheckup 常量；S2 implementedPlatform 移除 12（保留 IsConnected/IsDemo/IsTradeAllowed/GetTickCount*/SetUserError/CurTime）；S3 builtins.go 删 12 nil 注册；S4 vm_builtin_wiring.go 删 12 fn 绑定；S5 vm_builtin_checkup.go 删 12 假实现函数（保留 IsConnected/IsDemo/IsTradeAllowed VM-API-TRUTH-3 真实+GetLastError/ResetLastError/SetUserError lastError 状态机+CurTime/GetTickCount* 真实时间源）；S6d `TestVM_API_TRUTH_1_PlatformCheckupRejected`（12 API 调用 CompileMQL 返 error 含 unsupported/API 名）+S6e `TestVM_API_TRUTH_1_PlatformCheckupRegistryConsistency`（LookupAPI=StatusUnsupported+Reason 非空+IsAPIImplemented=false+IsAPIUnsupported=true）+S6f `TestVM_API_TRUTH_1_PlatformCheckupRealStillImplemented`（IsConnected/IsDemo/IsTradeAllowed/GetLastError/ResetLastError/SetUserError/CurTime/GetTickCount*/IsTesting/IsOptimization/IsVisualMode 13 API 未误伤，IsAPIImplemented=true）；1 项 mutation RED→GREEN（注释 unsupportedSymbols 12 行 → S6d RED `got nil — API silently accepted`+S6e RED `LookupAPI returned not-found` → restore → GREEN）。
- 2026-09-16 **VM-API-TRUTH-1 批次1 ✅done**（Devin CLI 独立复审通过）：commit e97a43b8；S1 unsupportedSymbols 加 22 MQL5 order/deal/history API+reasonMQL5History 常量（`api_registry.go:148-169`）；S2 implementedMQL5Position 移除 22（保留 PositionSelect，`builtin_registry.go:128-130`）；S3 builtins.go 删 22 nil 注册（保留 PositionSelect）；S4 vm_builtin_wiring.go 删 22 fn 绑定+registerExtendedHistory 整函数+init 调用点；S5 vm_builtin_mql5_trade.go 删 22 假实现函数（保留 PositionSelect 委托 PositionSelectByTicket）；S6a `TestVM_API_TRUTH_1_MQL5HistoryRejected`（22 API 调用 CompileMQL 返 error 含 unsupported/API 名）+S6b `TestVM_API_TRUTH_1_RegistryConsistency`（LookupAPI=StatusUnsupported+Reason 非空+IsAPIImplemented=false+IsAPIUnsupported=true）+S6c `TestVM_API_TRUTH_1_PositionSelectStillImplemented`（StatusImplemented+IsAPIImplemented=true，证明重分类不误伤）；**独立 mutation×1 重跑** RED→restore→GREEN（注释 unsupportedSymbols 22 行 → S6a/S6b RED `LookupAPI returned not-found — missing from registry`）；机检独立复测：build/mql2go 433/race×3 1299/vet/gofmt/check-lines 0 errors/diff --check clean。**后续批次待做**：AccountInfo*/CopyBuffer/CopyRates/Symbol session/margin/platform checkup 假实现重分类。
- 2026-09-18 **STATE.md 活跃指针滚出**（P2 预算）：滚出 2026-08-26 陈旧 ✅done 指针 D-REVERT-CLEANUP-001 / D-REVERT-SCOPE-DRIFT-001 / VM-CACHE-INTEGRITY-1/2 / VM-TRADE-CONTEXT-1/2（明细在 registry，权威记录不丢）。
- 2026-09-17 **VM-API-TRUTH-1 批次2c ✅done** — commit 52add8ed；AccountInfo* 假分支 fail-closed+枚举对齐+26 常量（非重分类）；独立 mutation×4 RED→GREEN；明细见 registry 行 126。
- 2026-09-17 **VM-API-TRUTH-1 批次2d ✅done** — commit a306f54e；7 timeseries API 重分类；14 真实/venue 实现保留；独立 mutation×1 RED→GREEN；明细见 registry 行 126。
- 2026-09-17 **VM-API-TRUTH-1 批次2e ✅done + 整债收官** — commit 69d2330b；4 重分类+4 实接；python account.profit 顺带修复；独立 mutation×3 RED→GREEN；明细见 registry 行 126（条目转 ✅done）。


## 2026-09-17~09-19 滚出（STATE.md 20KB 预算）

- 2026-09-17 **VM-API-TRUTH-1 批次2c/2d/2e ✅done + 整债收官** — commit 52add8ed/a306f54e/69d2330b；AccountInfo* fail-closed+7 timeseries 重分类+4 实接收官；明细 registry 行 126（批次明细滚出 LOG.md）。
- 2026-09-18 **VM-ENUM-NUMBERING-1 ✅done** — commit 477e8273+bfb42ea3 返修R1；独立 mutation×5 RED→GREEN；TIME_MSC int32 截断 spec 缺陷已修；明细见 registry 行 219。
- 2026-09-18 **VM-GLOBAL-ARRAY-DECL-1 ✅done** — commit da902af6（自审计修正版派工单 @37358629）；独立 mutation×4 RED→GREEN；明细见 registry 行 220。
- 2026-09-18 **ORDERSEND-NILBROKER-FAILCLOSED-1 ✅done** — commit 2739f100（派工单 @0461ff34）；12 站 signalMode 前移+nil-broker→fatal；独立 mutation×3 RED→GREEN；另立 TRADE-BUILTIN-ERR-SWALLOW-1；明细 registry 行 215。
- 2026-09-18 **TRADE-BUILTIN-ERR-SWALLOW-1 ✅done** — commit 6eae8160（派工单 @d015173f）；13 站三态分裂+SimBroker 通道搬迁+engine RetCode 日志；独立 mutation×4 RED→GREEN；明细 registry 行 222。
- 2026-09-18 **VM-FUNC-FATAL-DELAY-1 ✅done** — commit de6f672c+6ef18536（派工单 @a74556c6+R1）；executeCallUser 循环顶 fatalError 检查；独立 mutation×2 RED→GREEN；明细 registry 行 218。
- 2026-09-18 **TEST-WAITSTATE-ACQUIRE-BCAST-1 ✅done** — commit 53e886e9（派工单 @a6ab8bbb）；Acquire 锁内补 Broadcast+判别性延迟测试；独立 mutation RED→GREEN；明细 registry 行 211。
- 2026-09-18 **SNAPSHOT-SLICE-ALIAS-1 ✅done** — commit 40148ede（派工单 @e253a836）；边界不变量 4 面私有化+契约钉注；独立 mutation×4 各精确命中；明细 registry 行 212。
- 2026-09-16 **VM-API-TRUTH-1 批次1/2a ✅done + VM 质量方案 v2 全量收官 + registry 全量对账** — 明细已滚出 LOG.md；registry 行 126/88/87。
## 2026-09-19 滚出（STATE.md 20KB 预算，LOWPRI-SWEEP-2 批）

- 2026-09-17~09-19 **VM 批七项 + 簿记 + i18n 收官**（已滚出 STATE.md 最近变更日志）— registry 簿记修正（20 处陈旧续行格翻正+hook 续行校验修复）；MQL-LOOP-4 ✅done（条目漂移翻正+弱 pin 补强 contains "fatal coverage"，同 mutation 精确 RED→restore 4/4 GREEN）；LLM-CONFIG-1 ✅done（条目漂移翻正，Temperature/TimeoutSeconds 已由 43f1e20a 修复）；i18n 债系收官（I18N-MIXED-2 相位2 ja+vi 3eb5c69a+2388e919 五 locale strict 全 0/0；backend 批末回归 3663 绿/4 失=3 DB 环境缺+1 审计侧样品换新）；VM-API-TRUTH-1 收官/VM-ENUM-NUMBERING-1/VM-GLOBAL-ARRAY-DECL-1/ORDERSEND-NILBROKER/TRADE-BUILTIN-ERR-SWALLOW/VM-FUNC-FATAL-DELAY/TEST-WAITSTATE/SNAPSHOT-SLICE-ALIAS 批次明细见上方 2026-09-17~18 滚出段与 registry。

## 2026-09-19 POST-2 探针批施工（⚠️待独立复审）

- 2026-09-19 **POST-2 探针批 ⚠️待独立复审** — 派工单 `builder-handoff-post2-capacity-probe.md` @981ec6ad；S1 `BenchmarkVMExec_Concurrency{N=1,4,8,16}`（`vm_bench_test.go`）+ S2 `TestSSEFanoutCostCurve`（`internal/paper/sse_fanout_probe_test.go`）+ S3 `TestPlacePaperOrderLatency_Concurrency`/`BenchmarkPlacePaperOrder`（`internal/paper/paper_order_probe_test.go`）+ S4 `docs/benchmarks/post2-capacity-baseline-2026-09.md`。三轴数字：VM 4vCPU 饱和拐点 N≈4（吞吐 1.83×→平台 2.1×，N=16 并发 1000-tick 回测 ~0.53s vs 单独 68ms）；SSE 每流恒定 ~2 server goroutine+~45-50KB、N=500 投递 p99 19.9ms（5ms 步进口径，突发=chan(8) drop 快照语义自洽）；paper 串行 6.2µs/单、并发封顶 ~34 万单/s。**设计表修正（复审关注点）**：SSE 并非缺 limiter——`SSEStreamLimitMiddleware(5)` 已接线 `main.go:276`，但 `isSSERequest` 只匹配 text/event-stream，前端主形态 ConnectRPC binary（application/connect+proto）不过 limiter，净效果无界；缺口 G-POST2-1（ConnectRPC 流无上限）/G-POST2-2（stream 无 metric）待登记裁决。门禁 build/vet/test/bench/check-lines(0 errors)/diff-check 全绿 + race×3。零触容器/生产端口/mtapi/DB。

## 2026-09-19 VM-LIVE-VENUE-1 独立复审验收（Devin CLI）

- **结论 ✅done**——施工 `dfd9eccd` 范围核符 S1-S6 设计/派工（2a8f7f11）。门禁：build/vet 净、2079 测试（7 影响包）、check-lines 0 errors（mt5/orders.go 450 贴预警线）、diff-check 净。
- **三项独立 mutation 实证**（Devin CLI 亲跑，非 builder 自报）：M1 mt4 `FetchSymbolParams` Ex==nil 哨兵删→`TestFetchSymbolParams_Parity_TradeEnumsExOrSentinel` RED（`got TradeMode=0 FreezeLevel=0 TradeExemode=0`，哨兵失效精确判）；M2 `SymbolInfoInteger` prop26 复辟常量 0→`TestVMVenueConstR2_SymbolInfoIntegerTradeEnumsVerbatim` RED（真值格 `want 5`+哨兵格 `want -1` 双判）；M3 `MODE_TRADEALLOWED` 复辟纯账户旗标→`TestVMVenueConstR2_ModeTradeAllowedTwoAxis` RED（unknown -1/disabled 0/close_only 3 三格 `want 0 got 1`，fail-closed 门精确判）。全部恢复 GREEN、工作树无 mutation 残留（diff 空）。
- **裁决**：builder D-012 报 mt5/orders.go 449→450 贴 1.5× 线——HEAD 存量 449 行预警非本批新增，按一任务一范围裁决另立 `CODE-SIZE-MT5-ORDERS-1` 🟦open（抽函数独立批，参照 VM-CODE-HYGIENE-1 先例）；R2 范围接受。
- **部署状态**：未部署——R2 venue 字段需 backend 重建进 VM 实盘链生效；部署+demo `904d14e6` venue 三字段有界复验列入下一步。
- registry 行 31 翻 ✅done + 行 32/33 新 open 登记；STATE.md 施工表/现状/下一步/指针区同步。
