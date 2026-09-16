# STATE — 当前状态 + 交接负载（T0）

> **轻量交接负载**。技术债务明细在 `docs/audits/tech-debt-registry.md`，本文件只放当前活跃条目指针。
> 收工必更新本文件（pre-commit 强制）。≤ 20KB。

## 交接负载

- **现状**: VM-AUDIT-2026-08-27 全 3 批 ✅done（-1~-8）+ round 4-5 全 5 batch ✅done。**P1 业务管线**：2 个 live 执行 bug 修复完成（login lookup 类型不匹配 + proto3 nil/empty slice 误拒）+ 1 个架构缺陷修复完成（FIX-2026-08-27-SESSION-PROTO-ROUNDTRIP ✅done，Devin CLI 验收通过 2026-08-28）。**FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION S1+S2+S3 ✅done**（Devin CLI 验收通过 2026-08-27）。**FIX-2026-08-27-SCHEDULE-HEALTH-ORDER-HISTORY-GAP ✅done**（Devin CLI 验收通过 2026-08-28）。**FIX-2026-08-28-DATA-TRUTH-1-RECONCILIATION-CONVERGENCE S1-S4 ✅done**（Devin CLI 验收通过 2026-08-28）。**FIX-2026-08-28-TRUST-1-DEMO-REAL-ACCOUNT-DISTINCTION S1-S7 ✅done**（Devin CLI 验收通过 2026-08-28：demo/real 区分——adapter 读 broker Type + mdtick 加字段 + service 写 account_type + CreateAccount 传值 + LinkLiveAccount real-only 校验 + marketplace 表+cache+leaderboard 过滤 + migration 276）。
- **方向校验**: ✅ 与 AGENTS.md §1 一致（策略市场平台）。
- **施工表**:

| 子任务 | 状态 | 锚点 |
|--------|------|------|
| 2026-08-26/27 批次 + 2026-09-08 系列（D-006/D-007/D-REVERT×2/VM-CACHE-INTEGRITY-1/2/LIVE-ORDER-REENTRY-1/VM-TRADE-CONTEXT-1/2/VM-COMPILER-SEMANTICS-1/BT-FUNC-ENTRYPC-FWD/VM-TIMESERIES-SEMANTICS-1/VM-RUNTIME-FAILCLOSED-1/DATA-TRUTH-2b/VM-AUDIT-2026-08-27×3/VM round 4-5/P1 管线审计/P1 live bug 修复/FIX-2026-09-08-BYOK-MODEL-PICKER/TEMP-RETRY/CURL-IMPORT/AI-SETTINGS-BYOK/CHAT-CTX/ADVANCED-PARAMS/COMPILE-NOTIFY/WORKSPACE-IA×2/AI-SETTINGS-审计二） | ✅done | 已滚出 LOG.md 2026-09-16；详见 registry |
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
| RECONCILE-TZ-WINDOW-1 修复 | ✅done | Devin CLI 验收通过 2026-09-16；commit 1efbf678；`.UTC()` 一行+pin×2+sweep 报告；独立 mutation RED→GREEN；sweep 另立 2 债 |
| TZ-SWEEP-AFFECTED-1 修复 | ✅done | Devin CLI 验收通过 2026-09-16；commit 362d285e；analyticsSince()+worker 参数归一化；独立 mutation -8h 编码偏移复现 RED→GREEN |
| TZ-MIXED-ENCODING-1 根因修复 | ✅done | Devin CLI 验收通过 2026-09-16；commit da85f973；~50 站写入端 .UTC() 全枚举+读侧同步+migration 278 签名回填（10100 行 CST→UTC，幂等）；CST 配对列裁定不翻分立 TZ-PAIRED-CST-COLS-1；独立 mutation RED→GREEN |
| VM-RUNTIME-FAILCLOSED-2 静默算术/栈/槽位 fail-closed | ✅done | Devin CLI 验收通过 2026-09-16；commit 4fea9439；S1-S4 setStackError 全覆盖+OP_STORE_VAR 栈泄漏修复；7 行为测试+独立 mutation×4 重跑 RED→GREEN；机检独立复测全绿；分立债 VM-ARRAY-OOB-FAILCLOSED-1(P2)/VM-FUNC-FATAL-DELAY-1(P3) |
| VM-HONESTY-3-REVIEW 死分支解耦+R06 非致命对抗 | ✅done | Devin CLI 验收通过 2026-09-16；commit 5816d7e9；S1 死分支 iNonExistentIndicator+MA3/200bars 产 10 trades 证 IsReliable=false 仅来自 fatal loop（非 <10 trades 兜底）；S2 R06 warning blind spot 证 loop 不误伤+强断言 IsReliable=true（替换原弱容忍 false）；独立 mutation×2 RED→GREEN；机检独立复测全绿；零生产代码改动 |

- **阻塞/待决策**: D-COMMIT-SCOPE-001 部署闸仍有效。TRON-SECURITY-1 业主暂缓（不做）。
- **下一步**: VM-HONESTY-3-REVIEW ✅done（Devin CLI 验收 2026-09-16）。后续队列：VM-COMPILER-SEMANTICS-3（switch default/break 栈）→ VM-API-TRUTH-1（StatusUnsupported 重分类，已裁定可派工）→ VM-ARRAY-OOB-FAILCLOSED-1（P2，数组越界静默）→ P3 批（ORDERSEND-NILBROKER/TEST-WAITSTATE/SNAPSHOT-SLICE/PY-SCOPE-KNOWN/PY-DECIMAL-CTOR/TZ-PAIRED-CST-COLS/VM-FUNC-FATAL-DELAY）。VM-LIVE-MTF-1 暂缓（需求驱动）；DATA-TRUTH-3 已裁定（v2 降级凭据-only + 删死列，P3）。
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
- **DATA-TRUTH-1** ✅done — orders 表 reconciliation 收敛（S1-S4 已验收，见 FIX-2026-08-28-DATA-TRUTH-1-RECONCILIATION-CONVERGENCE；2026-09-16 对账修正本指针漂移）
- **QUOTE-RECONNECT-LOOP** ✅done — 报价流自持重连循环修复（2026-08-27 Devin CLI 验收通过）
- **BROKER-SEARCH-1** ✅done — mtapi host 配置接线（2026-08-27 Devin CLI 验收通过）
- **TRUST-1** ✅done — Demo/真实账户战绩混展无标注（Devin CLI 验收通过 2026-08-28：adapter 读 broker Type + mdtick 加 AccountType 字段 + service 写 account_type + CreateAccount 传值 + LinkLiveAccount real-only 校验 + marketplace 表+cache+leaderboard 过滤 + migration 276，11 项对抗证明 + 4 项独立重跑 RED→restore→GREEN）
- **SCHEDULE-HOTLOOP-1** ✅done — 已部署验收（CPU 57%→6.49%，2026-09-16 对账修正本指针漂移）
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
- **RECONCILE-TZ-WINDOW-1** ✅done — `.UTC()` 修复+pin 测试，Devin CLI 验收通过 2026-09-16（1efbf678）
- **TZ-SWEEP-AFFECTED-1** ✅done — analyticsSince()+worker 归一化，Devin CLI 验收通过 2026-09-16（362d285e）
- **TZ-MIXED-ENCODING-1** ✅done — ~50 站 .UTC() 止血+migration 278 回填，Devin CLI 验收通过 2026-09-16（da85f973）；**部署注记**：migration 在 backend 启动时跑，生效后 trade_records 全列 UTC
- **TZ-PAIRED-CST-COLS-1** 🟦open P3 — next_run_at/trade_logs 等 CST 写读配对列禁单侧翻 UTC（registry 规则）
- **TEST-WAITSTATE-ACQUIRE-BCAST-1 / SNAPSHOT-SLICE-ALIAS-1** 🟦open P3 — QS-2.4 审计发现（WaitState(submitting) 时序 footgun / retained 快照 slice 别名依赖 immutable 约定）
- **VM-RUNTIME-FAILCLOSED-2** ✅done — 静默算术/栈/槽位 fail-closed（Devin CLI 验收通过 2026-09-16，commit 4fea9439，独立 mutation×4 RED→GREEN）
- **VM-ARRAY-OOB-FAILCLOSED-1** 🟦open P2 — OP_PUSH_ARRAY/OP_STORE_ARRAY 越界静默 NoneVal/丢弃（FAILCLOSED-2 复审分立）
- **VM-FUNC-FATAL-DELAY-1** 🟦open P3 — executeCallUser 内层循环无 fatalError 逐指令检查（FAILCLOSED-2 复审分立）
- **VM-HONESTY-3-REVIEW** ✅done — 死分支解耦+R06 非致命对抗测试重构（Devin CLI 验收通过 2026-09-16，commit 5816d7e9，独立 mutation×2 RED→GREEN，零生产代码改动）

## 最近变更日志

> 完整历史见 `docs/audits/handover-audit-plan.md` + `docs/handoff/LOG.md`。

- 2026-09-16 **VM-HONESTY-3-REVIEW ✅done**（Devin CLI 独立复审通过）：commit 5816d7e9；S1 死分支 iNonExistentIndicator+MA3/200bars 产 TotalTrades=10（assessRisk 设 IsReliable=true）证 IsReliable=false 仅来自 fatal loop（非 <10 trades 兜底）；S2 R06 warning blind spot 证 loop 不误伤+强断言 IsReliable=true（替换原弱容忍 false 逻辑）；**独立 mutation×2 重跑** RED→restore→GREEN（① 注释 fatal loop → S1 RED `IsReliable=true, trades≥10, fatal blind spot present`；② fatal loop 条件改 `!=SeverityInfo` → S2 RED `IsReliable=false, warning 误伤`）；机检独立复测：build/connect-strategy 416/race×3 1248/vet/gofmt/check-lines 0 errors/diff --check clean；零生产代码改动（fatal loop 是被测对象非被改对象）。
- 2026-09-16 **VM-HONESTY-3-REVIEW 施工完成**（builder，待独立复审）：S1 死分支 iNonExistentIndicator+MA3/200bars 产 10 trades 证 IsReliable=false 仅来自 fatal loop（非 <10 trades 兜底）；S2 R06 warning blind spot 证 loop 不误伤+强断言 IsReliable=true（替换原弱容忍逻辑）；各 1 项 mutation RED→GREEN；零生产代码改动。
- 2026-09-16 **VM-RUNTIME-FAILCLOSED-2 ✅done**（Devin CLI 独立复审通过）：commit 4fea9439；S1-S4 setStackError 覆盖 arith/floorDiv 除零取模+OP_DUP/OP_SWAP underflow+OP_PUSH/STORE_VAR/GLOBAL 越界（含 OP_STORE_VAR 栈泄漏修复）；7 行为测试全绿+**独立 mutation×4 重跑** RED→restore→GREEN；机检独立复测：build/mql2go 384/race×3 1152/strategy 367/connect-strategy 416/vet/check-lines 0 errors。分立债：VM-ARRAY-OOB-FAILCLOSED-1（P2）+ VM-FUNC-FATAL-DELAY-1（P3）。
- 2026-09-16 **VM 质量方案 v2 全量收官**（10 子任务全 Devin CLI 验收）：QS-1.4 bool 双否定（88292b14）/ QS-1.6 权威读状态机迁移（5be48f30）/ QS-1.3 函数域隔离 v3（9940eda4+ee47292d）/ QS-1.2a lastError 三 builtin（65e2cccf）/ QS-1.7-INV 不立项（05138758）/ QS-2.2 goleak（174b8405）/ QS-2.4 race 审计（8e393cae）/ QS-2.5 panic 加固（89353004）/ QS-2.3 noopContext（5ad339a9）/ QS-3-BASELINE 基线+metric（b8ad1674）。明细滚出至 handover-audit-plan.md。
- 2026-09-16 **registry 全量对账**（Devin CLI）：所有 ⚠️待独立复审项清零——翻正漂移 ✅done×9（LIVE-ORDER-REENTRY-1/LIVE-MQL-ORDER-CONTEXT-1/LIVE-REDESIGN-2TAB/LIVE-DIAG-TRUTH-1/VM-TEST-EVIDENCE-3/返工 Batch5 等），裁定决策项×5（DATA-TRUTH-3=v2 凭据-only 附属表/VM-API-TRUTH-1=批准 StatusUnsupported 派工/VM-LIVE-MTF-1=暂缓需求驱动/STREAM-FREEZE-1=代码验收+生产实测挂业主/LIVE-ORDER-REENTRY-1 三遗留裁定），标注待重施工×2（VM-RUNTIME-FAILCLOSED-2/VM-HONESTY-3-REVIEW 代码不在仓）。

> 2026-09-08 及更早的变更日志（FIX-2026-09-08-TEMP-RETRY/FIX-2026-09-08-BYOK-MODEL-PICKER/VM-TRADE-CONTEXT-1/2 ✅done、LIVE-ORDER-REENTRY-1-R4-REVIEW ✅done、VM-CACHE-INTEGRITY-1/2 ✅done、DATA-TRUTH-2b ✅done、三个 spec 落档、D-REVERT-SCOPE-DRIFT-001、D-REVERT-CLEANUP-001、治理结构重构、D-006/D-007、VM-CACHE-INTEGRITY-1/2 commit、LIVE-ORDER-REENTRY-1 R4 commit、第三/四批施工提示词落档、VM-COMPILER-SEMANTICS-1 + BT-FUNC-ENTRYPC-FWD ✅done、第四批施工提示词落档）已滚出至 `docs/handoff/LOG.md` + `docs/audits/handover-audit-plan.md`。
