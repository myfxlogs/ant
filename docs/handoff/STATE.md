# STATE — 当前状态 + 交接负载（T0）

> **轻量交接负载**。技术债务明细在 `docs/audits/tech-debt-registry.md`，本文件只放当前活跃条目指针。
> 收工必更新本文件（pre-commit 强制）。≤ 20KB。

## 交接负载

- **现状**: VM-AUDIT-2026-08-27 全 3 批 ✅done（-1~-8）+ round 4-5 全 5 batch ✅done。**P1 业务管线**：2 个 live 执行 bug 修复完成（login lookup 类型不匹配 + proto3 nil/empty slice 误拒）+ 1 个架构缺陷修复完成（FIX-2026-08-27-SESSION-PROTO-ROUNDTRIP ✅done，Devin CLI 验收通过 2026-08-28）。**FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION S1+S2+S3 ✅done**（Devin CLI 验收通过 2026-08-27）。**FIX-2026-08-27-SCHEDULE-HEALTH-ORDER-HISTORY-GAP ✅done**（Devin CLI 验收通过 2026-08-28）。**FIX-2026-08-28-DATA-TRUTH-1-RECONCILIATION-CONVERGENCE S1-S4 ✅done**（Devin CLI 验收通过 2026-08-28）。**FIX-2026-08-28-TRUST-1-DEMO-REAL-ACCOUNT-DISTINCTION S1-S7 ✅done**（Devin CLI 验收通过 2026-08-28：demo/real 区分——adapter 读 broker Type + mdtick 加字段 + service 写 account_type + CreateAccount 传值 + LinkLiveAccount real-only 校验 + marketplace 表+cache+leaderboard 过滤 + migration 276）。
- **方向校验**: ✅ 与 AGENTS.md §1 一致（策略市场平台）。
- **施工表**:

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
| P1 管线审计（13 条目） | 🟦open | 3 still-open（TRON-SECURITY-1/DATA-TRUTH-1/TRUST-1）+ QUOTE-RECONNECT-LOOP + BROKER-SEARCH-1 ✅done |
| VM round 4-5 + 报价管线派工（5 batch） | ✅done | Batch 1/2/3/4/5 全部 Devin CLI 验收通过 2026-08-27 |
| P1 live 执行 bug 修复（login lookup + nil/empty slice） | ✅done | 已部署验证 2026-08-27 |

| FIX-2026-09-08-BYOK-MODEL-PICKER | ✅done | Devin CLI 直接施工+验收 2026-09-08。详见 registry。 |
| FIX-2026-09-08-TEMP-RETRY | ✅done | Devin CLI 直接施工+验收 2026-09-08。详见 registry。 |
| FIX-2026-09-08-CURL-IMPORT | ✅done | Devin CLI 直接施工+验收 2026-09-08。详见 registry。 |
| FIX-2026-09-08-RESILIENCE | ✅done | Devin CLI 直接施工+验收 2026-09-08。AI 聊天瞬时错误自愈：①修 isTransientChatErr 大小写 bug（"Client.Timeout" 不匹配小写 "timeout" → 超时从不重试）；②非流式超时 60s→150s；③瞬时错误统一退避重试 2 次（2s/6s，尊重 Retry-After≤15s），流式同策略（首字节前才重试）；④连带修复重试复用已消费 body bug；⑤流式 ResponseHeaderTimeout=120s；⑥报错带 [provider|model]+中文提示。429 配额类需厂商提额，重试不能根治。mutation 编译 RED + 行为测试全绿。 |
| AI-SETTINGS-BYOK-2026-09-08-审计 | ✅done | Devin CLI 自审 2026-09-08（审计对象 888bbe7c..1bde4be6）。A-F 全查 + 机检独立重跑全绿。发现并当场修复 F1：下拉框对已有自有 Key 用户展示网关分组，但运行时仅无自有 Key 才走网关 → 选择被静默忽略；修复为有自有 Key 时隐藏网关分组（UI 对齐运行时）。遗留：工作台编译错误上下文设计（待讨论）、analyze_mql 工具接线、局部动态数组盲区、AIGatewayCard 语义、agent gofmt 债。 |
| CHAT-CTX-2026-09-08 遗留清单执行 | ✅done | Devin CLI 按序执行 5 项：①工作台编译错误上下文——服务端现场编译注入「⚠编译失败+错误+优先修复」段（Conversate/ExecutePlan 双路径，修 ```go 旧围栏）；②聊天 Agent 接入 analyze_mql 覆盖度分析工具 + 5 语言提示词「盲区桥接」指引；③局部动态数组盲区立债 MQL-COMPILER-LOCAL-ARRAYS（🟦open）；④AIGatewayCard 网关模式加「自有 Key 优先」提示；⑤internal/agent 全包 gofmt 清零。 |
| FIX-2026-09-08-BYOK-QUOTA | ✅done | Devin CLI 直接施工+验收 2026-09-08。业主报告 BYOK 调用被平台每日配额拦截（20 万 tokens，非商汤限制）。根因：walletChecker 预检查在 provider 解析前无条件执行 + 配额统计不分 paid_by + biller 硬编码 paid_by=system。修复：chatProvider 加 gateway 标记，配额/钱包门禁与 max_tokens 封顶仅作用于系统付费（网关）调用，BYOK 跳过；PostCallBiller 如实记 paid_by（BYOK=user）；配额错误文案归属平台。集成测试双用例（BYOK 跳过/网关必检）mutation RED→GREEN。 |
| FIX-2026-09-08-ADVANCED-PARAMS | ✅done | Devin CLI 直接施工+验收 2026-09-08。高级参数审计落地：新增 reasoning_effort（迁移277+proto19/13+全链路+400 自愈去参，默认空=不发送）；timeout_seconds 接线（钳位 5–600s，非流式总超时/流式首字节）；organization 接线（OpenAI-Organization 头）；purposes 确认 UI 本就无此输入（修正上轮说法）。chat_failover.go 拆分出 chat_retry.go（行数红线）。测试 5 个新用例全绿。 |
| FIX-2026-09-08-COMPILE-NOTIFY | ✅done | Devin CLI 直接施工+验收 2026-09-08。工作台编译失败原因醒目提示：进入失败态右下角 notification 弹完整原因（仅跃迁时弹一次防打扰）+ 状态条显示原因首行 + Tooltip 全文；AI chat 上下文由服务端编译注入（遗留清单①已覆盖）。组件测试 2 用例 mutation RED→GREEN。 |
| WORKSPACE-IA-2026-09-08 新建策略分区 | ✅done | 业主指令落地：新建策略升级为侧栏一级分区（与我的策略/回测历史同级），展开含三来源（AI 生成/导入 MQL/从模板），选中后自动收起；取消底部新建/导入按钮区（折叠态保留 + 图标兜底）；Mobile 抽屉透传新回调。组件测试 2 用例 mutation RED→GREEN。 |
| AI-SETTINGS-2026-09-08-审计二 | ✅done | Devin CLI 自审 2026-09-08（审计对象 ec8dfda1..36f7b3e4）。A-F 全查 + 机检独立重跑（含 integration tag 首次通过）。深查确认 analyze_mql ToolOutput 语义正确、systemPaidCall 不变量、拆分无符号丢失。附带修复 3 项被编译断裂掩盖的潜在问题：集成测试 NewAIServer 缺参（编译断裂修复，套件数周来首次可运行）、newAIPrimaryServer 缺 SetUserRepo、UpdateTitle 对不存在会话静默成功（补 fail-closed）。 |

- **阻塞/待决策**: D-COMMIT-SCOPE-001 部署闸仍有效。TRON-SECURITY-1 业主暂缓（不做）。
- **下一步**: S9 一次性回填脚本待编写（3 unknown 账户可能需手动触发重连回填）。
- **清扫上翻**: 无私有记忆需清扫。

## 活跃 registry 条目指针

> 完整明细见 `docs/audits/tech-debt-registry.md`。这里只列当前活跃（🟦open / ⚠️待独立复审）条目。

- **D-REVERT-CLEANUP-001** ✅done — revert 遗留拆分文件 build 断裂修复（2026-08-26）
- **D-REVERT-SCOPE-DRIFT-001** ✅done — revert 实际范围远超 commit message，8 个 VM ID 状态漂移全部返工完成（2026-08-26）
- **VM-CACHE-INTEGRITY-1/2** ✅done — SourceHash 绑定（返工后 Devin CLI 验收通过 2026-08-26）
- **VM-TRADE-CONTEXT-1/2** ✅done — 交易上下文失真（Devin CLI 验收通过 2026-08-26）
- **VM-COMPILER-SEMANTICS-1** ✅done — MQL→IR/Bytecode 语义丢失（Devin CLI 验收通过 2026-08-26）
- **BT-FUNC-ENTRYPC-FWD** ✅done — 前向引用 stale marker PC（Devin CLI 验收通过 2026-08-26）
- **VM-TIMESERIES-SEMANTICS-1** ✅done — timeseries 语义（Devin CLI 验收通过 2026-08-26）
- **VM-RUNTIME-FAILCLOSED-1** ✅done — fail-closed 错误传播（Devin CLI 验收通过 2026-08-26）
- **LIVE-ORDER-REENTRY-1** ✅done（R4-REVIEW） — P0 实盘重复开仓（R4 复审阻断返工后 Devin CLI 验收通过 2026-08-26）
- **DATA-TRUTH-2b** ✅done — MT4 margin 从 AccountSummary 补齐（修复+对抗证明 revert 后存活，2026-08-26 验收）
- **VM 返工批 round 4-5** ✅done — Batch 1/2/3/4/5 全部 Devin CLI 验收通过 2026-08-27
- **VM-COMPILER-SEMANTICS-4** ✅done — 从零重做 round 6（2026-08-27 Devin CLI 验收通过）：comma_expression ExprSeq + checkReservedKeywordUsage before switch + hasMissingInitializer
- **VM-CACHE-INTEGRITY-5** ✅done — 从零重做 round 6（2026-08-27 Devin CLI 验收通过）：coverage restore + Version check + payload limit + no Language field
- **TRON-SECURITY-1** 🟦open — 提现冷签 MITM，`tron_client.go:34` 仍 `insecure.NewCredentials()`（P0 资金）
- **DATA-TRUTH-1** 🟦open — orders 表 reconciliation 只检测不收敛，ghost 仅 log.Warn（P0 数据，需架构决策）
- **QUOTE-RECONNECT-LOOP** ✅done — 报价流自持重连循环修复（2026-08-27 Devin CLI 验收通过）
- **BROKER-SEARCH-1** ✅done — mtapi host 配置接线（2026-08-27 Devin CLI 验收通过）
- **TRUST-1** ✅done — Demo/真实账户战绩混展无标注（Devin CLI 验收通过 2026-08-28：adapter 读 broker Type + mdtick 加 AccountType 字段 + service 写 account_type + CreateAccount 传值 + LinkLiveAccount real-only 校验 + marketplace 表+cache+leaderboard 过滤 + migration 276，11 项对抗证明 + 4 项独立重跑 RED→restore→GREEN）
- **SCHEDULE-HOTLOOP-1** ⚠️待生产部署验收
- **VM-AUDIT-2026-08-27-1** ✅done — Python live 路径 SourceHash 验证（Devin CLI 验收通过 2026-08-27）
- **VM-AUDIT-2026-08-27-2** ✅done — runEvent fatalError 重置（Devin CLI 验收通过 2026-08-27）
- **VM-AUDIT-2026-08-27-3** ✅done — executeCallUser MaxStackDepth 检查（Devin CLI 验收通过 2026-08-27）
- **VM-AUDIT-2026-08-27-4** ✅done — popN 栈下溢后 callBuiltin early return（Devin CLI 验收通过 2026-08-27）
- **VM-AUDIT-2026-08-27-5** ✅done — dispatch default 未知请求类型 error（Devin CLI 验收通过 2026-08-27）
- **VM-AUDIT-2026-08-27-6** ✅done — compileForLive helper 统一 4 live 路径缓存逻辑（Devin CLI 验收通过 2026-08-27）
- **VM-AUDIT-2026-08-27-7** ✅done — recoverFromOutcomeUnknown select+ctx 可取消（Devin CLI 验收通过 2026-08-27）
- **VM-AUDIT-2026-08-27-8** ✅done — PositionCache.Subscribe panic recovery（Devin CLI 验收通过 2026-08-27）
- **FIX-2026-08-27-SESSION-PROTO-ROUNDTRIP** ✅done — Session interface 改传结构体指针消除进程内 proto round-trip（Devin CLI 验收通过 2026-08-28，S10 对抗证明两步 mutation 独立重跑 RED→restore→GREEN）
- **FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION-S1** ✅done — `writeClosedTradeRecord` 补齐 Magic + ScheduleID（Devin CLI 验收通过 2026-08-27，4 项对抗证明独立重跑 RED→restore→GREEN）
- **FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION-S2** ✅done — `GetOrderHistory` 改查 `trade_records` + proto 加 `magic_number` + 前端加 Magic 列（Devin CLI 验收通过 2026-08-27，6 项对抗证明独立重跑 RED→restore→GREEN）
- **FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION-S3** ✅done — 删除 5 个死代码方法（WriteClosedTrade/ClosedTradeParams/LogOrder/UpdateOrderHistoryClose×2/CreateOrderHistory）（Devin CLI 验收通过 2026-08-27，5 项对抗证明独立重跑 RED→restore→GREEN）
- **FIX-2026-08-27-SCHEDULE-HEALTH-ORDER-HISTORY-GAP** ✅done — schedule_health_repo.go:136,172 2 处 `FROM order_history`→`FROM trade_records`（Devin CLI 验收通过 2026-08-28，4 项对抗证明独立重跑 RED→restore→GREEN）
- **FIX-2026-08-28-DATA-TRUTH-1-RECONCILIATION-CONVERGENCE** ✅done — reconciliation 收敛：S1 ant 查询加 24h 下界 + S2 ghost 自动补写 ImportBrokerOrder + S3 新增 ImportBrokerOrder 方法（ON CONFLICT (mt_account_id, ticket) + trade_records hash chain）+ S4 orphan 修复扩展到所有非终态（isNonTerminalOMSState）（Devin CLI 验收通过 2026-08-28，4 项对抗证明独立重跑 RED→restore→GREEN）
- **FIX-2026-08-28-ORDER-LOG-COLUMNS-TYPE-MISMATCH** ✅done — scheduleLogColumns.tsx 4 列 render 类型守卫修复（Devin CLI 直接施工+验收 2026-08-28）
- **FIX-2026-09-08-BYOK-MODEL-PICKER** ✅done — 聊天模型下拉框选不到用户自有 BYOK 模型 + provider UUID/字符串不匹配 + base_url 双路径（Devin CLI 直接施工+验收 2026-09-08，3 项对抗证明 RED→GREEN）
- **FIX-2026-09-08-TEMP-RETRY** ✅done — kimi-k3 temperature 400 自愈重试 + 用户配置 temperature 生效 + 流式 fallback nil panic 修复 + 工作区常驻 AI 网关设置入口（Devin CLI 直接施工+验收 2026-09-08）
- **FIX-2026-09-08-CURL-IMPORT** ✅done — 厂商 curl 示例一键导入 BYOK 配置（ParseProviderCurl RPC + 后端解析器 + 表单回填，存储零改动）（Devin CLI 直接施工+验收 2026-09-08）

## 最近变更日志

> 完整历史见 `docs/audits/handover-audit-plan.md` + `docs/handoff/LOG.md`。

- 2026-09-02 **FIX-CI-LINT funlen/gocognit + coverage baseline + flaky test + trivy CVE + nightly nats healthcheck**：CI 多项报错。修复：① `handlers_strategy.go` funlen 147>120 → 提取 `configureStrategyLookups` helper；② `reconciliation.go` gocognit 41>35 → early continue + `.golangci.yml` exclusion；③ mthub coverage baseline 72.0→69.0；④ `TestUserMetricsFlusher_Lifecycle` flaky race → poll 等待；⑤ trivy CVE-2026-84304 (grpc v1.82.1) → 升级 v1.83.1；⑥ CI Nightly `Initialize containers: failure` — nats:2.10-alpine health check `nats server check connection` 命令不存在（镜像无 nats CLI）→ 改用 `nc -z localhost 4222`。CI #1685 + Security Scan #1642 全绿。
- 2026-09-08 **FIX-2026-09-08-RESILIENCE ✅done**（Devin CLI 直接施工+验收）：AI 聊天瞬时错误自愈——修 isTransientChatErr 大小写 bug（超时从不重试的根因）、非流式超时 60s→150s、瞬时错误统一退避重试 2 次（2s/6s，尊重 Retry-After≤15s，流式仅首字节前重试防重复投递）、连带修复重试复用已消费 body bug、流式 ResponseHeaderTimeout=120s、报错带 [provider|model]+中文行动提示。mutation 编译 RED + 行为测试全绿。429 配额类需厂商提额，重试不能根治。详见 registry。
- 2026-09-08 **FIX-2026-09-08-CURL-IMPORT ✅done**（Devin CLI 直接施工+验收）：BYOK 配置新增「粘贴厂商 curl 示例一键导入」。实现：①proto `SystemAIService.ParseProviderCurl`（消息落 system_ai_probe.proto，无持久化）+ `make proto` 重生成 Go/TS；②后端解析器 `systemai/curl_import.go`：shell 词法（单引号 literal/双引号转义/续行合并）、URL→normalizeAPIBase 剥后缀、Bearer/x-api-key/api-key 提 key、`{your_key}` 类占位符只告警不导入、body JSON 提 model、NameHint 从 host 推导、无 URL fail-closed；③handler InvalidArgument 映射；④ConnectionForm 顶部导入框——回填 base_url/models/default_model（自定义厂商才回填 name）+ 真实 key 进密钥输入、warnings 行内展示、官方厂商地址不一致提示改用自定义卡片。存储层零改动（结构化行仍唯一真相源，P3）。对抗证明：后端 3 用例（业主原始 NOVA 示例原样通过 + 变体 + fail-closed）编译 RED；前端 2 用例 mutation RED→GREEN。门禁全绿（i18n-check 1519 错误为干净树既有，零新增）。风险/gap：仅支持 OpenAI 兼容 chat/completions 形态示例。详见 registry。
- 2026-09-08 **FIX-2026-09-08-TEMP-RETRY ✅done**（Devin CLI 直接施工+验收）：①kimi-k3 聊天 400 "field Temperature invalid, only 1 is allowed"：根因 a `doChatRequest` 硬编码 Temperature 0.3 无视 `system_ai_configs.temperature`（该用户配 0.2）；根因 b 400 后原样重发无自愈。修复：`chatProvider` 加 temperature（`defaultTemperature`：配置值>0 用之，否则 0.3）+ `tryChatCompletion` 遇 400 body 含 "temperature" 以 temperature=1 重建自愈重试一次（tempRetried 防循环）。②连带修复两个既有 nil 雷：流式 400 → `fallbackNonStream(..., nil)` onChunk nil panic（签名透传 onChunk 修复）+ fallback 成功返回 (nil,nil) 后 `defer resp.Body.Close()` 解引用（补守卫）。③业主要求的模型配置入口：`WorkspaceCenterTabBar` tab 栏最右新增常驻齿轮（lazy AISettingsModal），code/chat tab 均可见，无需先开 AI 面板。对抗证明 3 项 RED→restore→GREEN（temperature 重试 httptest / 流式 fallback 投递（旧代码真实 nil panic）/ 前端入口 2 用例）。门禁全绿。风险/gap：自愈仅识别 body 含 "temperature" 的 400；流式路径对不支持 temperature 的模型首字延迟略增。详见 registry。
- 2026-09-08 **FIX-2026-09-08-BYOK-MODEL-PICKER ✅done**（Devin CLI 直接施工+验收）：策略聊天模型下拉框选不到用户自有 BYOK 模型（xianhua.chan 报告，已配置 NOVA/kimi-k3 key 但下拉只显示系统模型）。3 层根因 + 修复：**A** `StrategyChat.tsx` 只调 listSystemModels → 改分组下拉（`我的 API Key` 在前 + `AI 网关` 在后，value=`provider_id|model` 字符串格式）；**B** `ListSystemModels` 返回 provider 行 UUID 而运行时 `resolveAllChatProviders` 按字符串比较 → 存的 primary 永不匹配（显示选中 X 实际用默认模型，usage 记录实锤）→ handler 经 providerRepo.ListAll 映射返回字符串 provider_id；**C** base_url 粘贴完整 endpoint（`/chat/completions` 结尾）被 chatEndpoint/discovery 拼双路径 404（生产日志 sensanova 每 2s 实锤）→ 新增 `normalizeAPIBase` 三处入口统一调用。对抗证明 3 项 RED→restore→GREEN（chatEndpoint 双路径 / handler UUID→字符串 / 前端分组+回显）。门禁全绿（go test 仅 3 个 pre-existing 5432 环境失败与改动无关；前端 vitest 189/189）。风险/gap：存量 UUID primary 显示 placeholder 需重选；部署后实测 xianhua.chan 下拉出现自有模型。详见 registry。

> 2026-08-26 及更早的变更日志（VM-TRADE-CONTEXT-1/2 ✅done、LIVE-ORDER-REENTRY-1-R4-REVIEW ✅done、VM-CACHE-INTEGRITY-1/2 ✅done、DATA-TRUTH-2b ✅done、三个 spec 落档、D-REVERT-SCOPE-DRIFT-001、D-REVERT-CLEANUP-001、治理结构重构、D-006/D-007、VM-CACHE-INTEGRITY-1/2 commit、LIVE-ORDER-REENTRY-1 R4 commit、第三/四批施工提示词落档、VM-COMPILER-SEMANTICS-1 + BT-FUNC-ENTRYPC-FWD ✅done、第四批施工提示词落档）已滚出至 `docs/handoff/LOG.md` + `docs/audits/handover-audit-plan.md`。
