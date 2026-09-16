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
| FIX-2026-09-08-TEMP-RETRY | ✅done | Devin CLI 直接施工+验收 2026-09-08。详见 registry + LOG。 |
| FIX-2026-09-08-CURL-IMPORT | ✅done | Devin CLI 直接施工+验收 2026-09-08。详见 registry。 |
| AI-SETTINGS-BYOK-2026-09-08-审计 | ✅done | Devin CLI 自审 2026-09-08（888bbe7c..1bde4be6）。修复 F1 网关分组显示与运行时不一致。详见 registry + LOG。 |
| CHAT-CTX-2026-09-08 遗留清单执行 | ✅done | Devin CLI 按序执行 5 项（编译上下文注入/analyze_mql 接线/立债 MQL-COMPILER-LOCAL-ARRAYS/网关提示/gofmt 清零）。详见 registry + LOG。 |
| FIX-2026-09-08-ADVANCED-PARAMS | ✅done | Devin CLI 直接施工+验收 2026-09-08。reasoning_effort/timeout_seconds/organization 三参数接线。详见 registry + LOG。 |
| FIX-2026-09-08-COMPILE-NOTIFY | ✅done | Devin CLI 直接施工+验收 2026-09-08。工作台编译失败原因醒目提示：进入失败态右下角 notification 弹完整原因（仅跃迁时弹一次防打扰）+ 状态条显示原因首行 + Tooltip 全文；AI chat 上下文由服务端编译注入（遗留清单①已覆盖）。组件测试 2 用例 mutation RED→GREEN。补记2：真实浏览器走查 9 步全过（Playwright + e2e 账号）——新增"手动编写"空白编辑器脚手架 + 侧栏来源项中文默认值。补记：回测历史面板渲染 protobuf Timestamp 对象致整页崩溃（React #31）——formatStartedAt 稳健格式化 + 生产形状回归测试。 |
| WORKSPACE-IA-2026-09-08 新建策略分区 | ✅done | 业主指令落地：新建策略升级为侧栏一级分区（与我的策略/回测历史同级），展开含三来源（AI 生成/导入 MQL/从模板），选中后自动收起；取消底部新建/导入按钮区（折叠态保留 + 图标兜底）；Mobile 抽屉透传新回调。组件测试 2 用例 mutation RED→GREEN。补记2：真实浏览器走查 9 步全过（Playwright + e2e 账号）——新增"手动编写"空白编辑器脚手架 + 侧栏来源项中文默认值。补记：回测历史面板渲染 protobuf Timestamp 对象致整页崩溃（React #31）——formatStartedAt 稳健格式化 + 生产形状回归测试。补记3：最终架构重构落地——单一 centerView 状态机 + AI/回测停靠面板（420px 并排不抢占），三分区点击保持展开，使用模板来源移除，CodeEditorArea 编辑器常驻。 |
| WORKSPACE-IA-2026-09-08 分区导航联动 | ✅done | 分区切换驱动主区联动；新建策略分区四来源菜单（含手动编写）；粘性 importMode 修复；使用模板移除；分区切换关闭右侧面板。真实走查 9 步全绿。原文滚出 LOG。 |
| AI-SETTINGS-2026-09-08-审计二 | ✅done | Devin CLI 自审 2026-09-08（审计对象 ec8dfda1..36f7b3e4）。A-F 全查 + 机检独立重跑（含 integration tag 首次通过）。深查确认 analyze_mql ToolOutput 语义正确、systemPaidCall 不变量、拆分无符号丢失。附带修复 3 项被编译断裂掩盖的潜在问题：集成测试 NewAIServer 缺参（编译断裂修复，套件数周来首次可运行）、newAIPrimaryServer 缺 SetUserRepo、UpdateTitle 对不存在会话静默成功（补 fail-closed）。 |
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

- **阻塞/待决策**: D-COMMIT-SCOPE-001 部署闸仍有效。TRON-SECURITY-1 业主暂缓（不做）。
- **下一步**: VM 质量方案 v2 全部子任务 ✅done（QS-1.x/2.x/3-BASELINE）；剩新债处置排期（ORDERSEND-NILBROKER-FAILCLOSED-1 等 4 条 open）。S9 回填脚本仍待编写。
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
- **QS-1.4** ✅done — Python bool(x) 语义修复 + vm_helpers:250 注释（Devin CLI 验收通过 2026-09-16，独立 mutation 重跑 RED→GREEN）
- **PY-DECIMAL-CTOR-1** 🟦open — `Decimal("0")` 产 ValString 非 ValDecimal（QS-1.4 复审发现，P2）
- **QS-1.6** ✅done — ConfirmByAuthoritativeRead 状态机迁移（Devin CLI 验收通过 2026-09-16）
- **QS-1.3** ✅done — Python 函数局部作用域（Devin CLI 验收通过 2026-09-16，commits 9940eda4+ee47292d）
- **PY-SCOPE-KNOWN-1** 🟦open — Python 作用域已知限制 3 条（QS-1.3 遗留，文档化行为）
- **QS-1.2a** ✅done — lastError 三 builtin（Devin CLI 验收通过 2026-09-16，commit 65e2cccf）
- **QS-1.7-INV** ✅done — ClientID 不可回显→QS-1.7 不立项，维持 fail-closed（Devin CLI 验收通过 2026-09-16）
- **QS-3-BASELINE** ✅done — 基线已落盘 `docs/audits/vm-perf-baseline-2026-09.md`；无项触及 >30% 阈值，生产 p99 待 metric 上线观测
- **ORDERSEND-NILBROKER-FAILCLOSED-1** 🟦open — QS-2.3 连带记债：无 broker 静默 -1+nil error 非 fail-closed
- **TEST-WAITSTATE-ACQUIRE-BCAST-1 / SNAPSHOT-SLICE-ALIAS-1** 🟦open P3 — QS-2.4 审计发现（WaitState(submitting) 时序 footgun / retained 快照 slice 别名依赖 immutable 约定）

## 最近变更日志

> 完整历史见 `docs/audits/handover-audit-plan.md` + `docs/handoff/LOG.md`。

- 2026-09-16 **QS-1.7-INV ✅done**（Devin CLI 独立复审通过，commit 05138758）：ClientID 全链路五跳零复制进 Comment、mt4 adapter 不透传 Comment、proto 无 client_id 字段→QS-1.7 不立项，open outcomeUnknown 维持 fail-closed 锁仓+runbook；findings 落盘 docs/audits/qs-1.7-inv-findings.md。
- 2026-09-16 **QS-1.2a ✅done**（Devin CLI 独立复审通过，commit 65e2cccf）：`vm.lastError` 跨事件驻留 + GetLastError 读后清零 + SetUserError=65536+c + ERR_USER_ERROR_FIRST 常量 + 6 项行为测试；独立 mutation×2 RED→GREEN；OrderSend fatal 路径零触碰（FAILCLOSED-1 保持）。
- 2026-09-16 **QS-1.3 ✅done**（Devin CLI 独立复审通过，commits 9940eda4+ee47292d）：Python 函数内未声明赋值改落函数域局部槽（`resolveAssignTarget`+`isDeclaredGlobal` GlobalDecls 谓词+`compileDecl` localScopes[0]），`+=` 未声明名编译期 fail-closed；过程经修正 v2（施工方两处转交决策采信）+v3（复审退回 for 循环域消亡回退）；独立 mutation×4 RED→GREEN；3 条已知限制入 PY-SCOPE-KNOWN-1。
- 2026-09-16 **QS-1.6 ✅done**（Devin CLI 独立复审通过，commit 5be48f30）：waitForConfirmation 权威读确认改走 `TradeBarrier.ConfirmByAuthoritativeRead` 状态机迁移；根因=cancel 动作名不在自身 updateType 兼容集 + open ticket==0 早退，两分支测试覆盖；独立 mutation×2 RED→GREEN；范围干净零交接层改动。
- 2026-09-16 **QS-1.4 ✅done**（Devin CLI 独立复审通过，commit 88292b14）：bool(x)→双重 OP_NOT 复用 IsTrue；独立 mutation（恢复 !=）重跑语义级 RED→restore→GREEN。裁定：Decimal 构造器失真立债 PY-DECIMAL-CTOR-1（P2）；BoolConversion 旧断言同步批准。流程记录：施工方动交接层文件违 §4，内容核验准确保留。同日固化 D-012（施工方自审）/D-013（决策方出件自审）/D-014（自报末行带编号+hash）。
- 2026-09-16 **VM 管线质量方案 v1 评估→v2 定稿**（Devin CLI 决策 D-009）：源码逐条核验，否决 QS-1.1/1.5/2.1/阶段 3 池化、改修法 QS-1.3/1.4/1.6、拆 QS-1.2、QS-1.7 改调研；10 条 QS 入 registry。详见 handover-audit-plan 2026-09-16 条目。
- 2026-09-08 **FIX-2026-09-08-TEMP-RETRY ✅done**（滚出至 LOG.md）：kimi-k3 400 temperature 自愈重试 + 两个 nil 雷修复 + 模型配置常驻齿轮入口。
- 2026-09-08 **FIX-2026-09-08-BYOK-MODEL-PICKER ✅done**（滚出至 LOG.md）：BYOK 模型下拉选不到自有模型，3 层根因修复（分组下拉/UUID→字符串 provider_id/normalizeAPIBase）。

> 2026-08-26 及更早的变更日志（VM-TRADE-CONTEXT-1/2 ✅done、LIVE-ORDER-REENTRY-1-R4-REVIEW ✅done、VM-CACHE-INTEGRITY-1/2 ✅done、DATA-TRUTH-2b ✅done、三个 spec 落档、D-REVERT-SCOPE-DRIFT-001、D-REVERT-CLEANUP-001、治理结构重构、D-006/D-007、VM-CACHE-INTEGRITY-1/2 commit、LIVE-ORDER-REENTRY-1 R4 commit、第三/四批施工提示词落档、VM-COMPILER-SEMANTICS-1 + BT-FUNC-ENTRYPC-FWD ✅done、第四批施工提示词落档）已滚出至 `docs/handoff/LOG.md` + `docs/audits/handover-audit-plan.md`。
