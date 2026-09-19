# STATE — 当前状态 + 交接负载（T0）

> **轻量交接负载**。技术债务明细在 `docs/audits/tech-debt-registry.md`，本文件只放当前活跃条目指针。
> 收工必更新本文件（pre-commit 强制）。≤ 20KB。

## 交接负载

- **现状**: **VM-API-TRUTH-1 整债 ✅done 收官**（46 API 重分类，明细 registry）。VM-ARRAY-OOB-FAILCLOSED-1 ✅done（bdb3733f）。VM-ENUM-NUMBERING-1 ✅done（477e8273+bfb42ea3）。**VM-GLOBAL-ARRAY-DECL-1 ✅done**（da902af6）。**ORDERSEND-NILBROKER ✅done**（2739f100）/**TRADE-BUILTIN-ERR-SWALLOW ✅done**（6eae8160）/**VM-FUNC-FATAL-DELAY ✅done**（de6f672c+6ef18536）/**TEST-WAITSTATE-ACQUIRE-BCAST ✅done**（53e886e9）/**SNAPSHOT-SLICE-ALIAS-1 ✅done**（40148ede）/**PY-DECIMAL-CTOR-1 ✅done**（e928722c）。**PY-SCOPE-KNOWN-1 ✅done**（373ca8d6）。**TZ-PAIRED-CST-COLS-1 ✅done**（aed6ff70）。**LIVE-ACCOUNT-FIELDS-1 ✅done**（45767c9f）。**ACCOUNT-MARGIN-LEVEL-PCT-1 ✅done**（ac509e20，Devin CLI 验收 2026-09-19）。**DATA-TRUTH-3 ✅done**（de1d0975）。**MT5-ACCMETHOD-ADAPTER-1 ✅done**（4578cb4e，Devin CLI 验收 2026-09-19）。**MQL-LOOP-4 ✅done**（条目漂移翻正+弱 pin 补强）。**LLM-CONFIG-1 ✅done**（条目漂移翻正，43f1e20a 已修复）。**MQL-COMPILER-LOCAL-ARRAYS 设计实查完成**——派工单 `builder-handoff-mql-compiler-local-arrays.md`，另实证 `ArrayResize` 全局静默无效+局部初始化器静默误编译两活 bug。
- **方向校验**: ✅ 与 AGENTS.md §1 一致（策略市场平台）。
- **施工表**:

| 子任务 | 状态 | 锚点 |
|--------|------|------|
| POST-2 探针批（S1 VM 并发退化/S2 SSE 扇出成本/S3 paper 下单延迟/S4 容量基线文档） | ✅done | Devin CLI 验收 2026-09-19；2408d34c；设计表修正核实[SSE limiter 不覆盖 ConnectRPC binary=净无界]；G-POST2-1/2 登记 open；staging 残余 7 项落 docs/benchmarks/post2-capacity-baseline-2026-09.md |
| 2026-08-26/27 批次 + 2026-09-08 系列（D-006/D-007/D-REVERT×2/VM-CACHE-INTEGRITY-1/2/LIVE-ORDER-REENTRY-1/VM-TRADE-CONTEXT-1/2/VM-COMPILER-SEMANTICS-1/BT-FUNC-ENTRYPC-FWD/VM-TIMESERIES-SEMANTICS-1/VM-RUNTIME-FAILCLOSED-1/DATA-TRUTH-2b/VM-AUDIT-2026-08-27×3/VM round 4-5/P1 管线审计/P1 live bug 修复/FIX-2026-09-08-BYOK-MODEL-PICKER/TEMP-RETRY/CURL-IMPORT/AI-SETTINGS-BYOK/CHAT-CTX/ADVANCED-PARAMS/COMPILE-NOTIFY/WORKSPACE-IA×2/AI-SETTINGS-审计二） | ✅done | 已滚出 LOG.md 2026-09-16；详见 registry |
| QS 系列（QS-1.2a/1.3/1.4/1.6/1.7-INV/2.2/2.3/2.4/2.5/3-BASELINE） | ✅done | 已滚出 LOG.md 2026-09-16；10 子任务全 Devin CLI 验收 2026-09-16；详见 registry |
| RECONCILE-TZ-WINDOW-1 修复 | ✅done | Devin CLI 验收通过 2026-09-16；commit 1efbf678；`.UTC()` 一行+pin×2+sweep 报告；独立 mutation RED→GREEN；sweep 另立 2 债 |
| TZ-SWEEP-AFFECTED-1 修复 | ✅done | Devin CLI 验收通过 2026-09-16；commit 362d285e；analyticsSince()+worker 参数归一化；独立 mutation -8h 编码偏移复现 RED→GREEN |
| TZ-MIXED-ENCODING-1 根因修复 | ✅done | Devin CLI 验收通过 2026-09-16；commit da85f973；~50 站写入端 .UTC() 全枚举+读侧同步+migration 278 签名回填（10100 行 CST→UTC，幂等）；CST 配对列裁定不翻分立 TZ-PAIRED-CST-COLS-1；独立 mutation RED→GREEN |
| VM-RUNTIME-FAILCLOSED-2 静默算术/栈/槽位 fail-closed | ✅done | Devin CLI 验收通过 2026-09-16；commit 4fea9439；S1-S4 setStackError 全覆盖+OP_STORE_VAR 栈泄漏修复；7 行为测试+独立 mutation×4 重跑 RED→GREEN；机检独立复测全绿；分立债 VM-ARRAY-OOB-FAILCLOSED-1(P2)/VM-FUNC-FATAL-DELAY-1(P3) |
| VM-HONESTY-3-REVIEW 死分支解耦+R06 非致命对抗 | ✅done | Devin CLI 验收通过 2026-09-16；commit 5816d7e9；S1 死分支 iNonExistentIndicator+MA3/200bars 产 10 trades 证 IsReliable=false 仅来自 fatal loop（非 <10 trades 兜底）；S2 R06 warning blind spot 证 loop 不误伤+强断言 IsReliable=true（替换原弱容忍 false）；独立 mutation×2 RED→GREEN；机检独立复测全绿；零生产代码改动 |
| VM-COMPILER-SEMANTICS-3 switch default 顺序+break 栈清理 | ✅done | Devin CLI 验收通过 2026-09-16；commit c5d1a7e0；S1 保留 s.Cases 原始顺序（default 不抽出作 fallthrough target）+default 首位 skip JMP；S2 break JMPs patch 到 popPC（OP_POP 位置）消费 switch value；S3a 栈深度断言+S3b default 中间 fallthrough（1010）+S3c break 栈清理（15+stack=0）；独立 mutation×2 RED→GREEN；机检独立复测全绿 |
| VM-API-TRUTH-1 批次1/2a/2b-2e（46+ API 重分类+假分支修复+4 实接，整债收官） | ✅done | Devin CLI 验收 2026-09-16/17；commit e97a43b8/8f946579/1fb352f1/52add8ed/a306f54e/69d2330b；明细见 registry 行 126+LOG.md |
| VM-ARRAY-OOB-FAILCLOSED-1 数组 OOB+局部负编码 fail-closed | ✅done | Devin CLI 验收通过 2026-09-18；commit bdb3733f；明细见 registry 行 217 |
| VM-ENUM-NUMBERING-1 SymbolInfo*/MarketInfo 全枚举对齐+错标修复 | ✅done | Devin CLI 验收通过 2026-09-18；commit 477e8273+bfb42ea3；明细见 registry 行 219 |

- **阻塞/待决策**: D-COMMIT-SCOPE-001 部署闸仍有效。TRON-SECURITY-1 业主暂缓（不做）。
- **下一步**: **VM-LIVE-PARITY-1 实盘对账完成**（vm-live-parity-results-2026-09.md——Exness-Trial demo BTCUSDm 实测：P1 回显实锤双轴/P4 瘦数据实锤/拒绝 fail-closed 实证/SL 全链实证；新债 F1 回显/P2 F2 瘦数据/P4 F3 comment 编码登记 open）。G-POST2-1/2 ✅done（295d93f8——StreamLimitInterceptor user10/ip30+withSency 收口+metric+sseOwnerKey header 修复；T3 独立 mutation RED→GREEN）。 **POST-2 探针批 ✅done**（2408d34c 独立复审通过——三轴实测绿+设计表修正核实[SSE limiter 形态错位成立]+缺口 G-POST2-1[P2 流上限]/G-POST2-2[P3 metric]已登记 open）（S1 `BenchmarkVMExec_Concurrency` 4vCPU 饱和拐点 N≈4/吞吐 2.1×封顶；S2 SSE 扇出每流恒定 ~2 goroutine+45-50KB、N=500 p99 19.9ms；S3 paper 6.2µs/单、瓶颈在下游 DB；S4 `docs/benchmarks/post2-capacity-baseline-2026-09.md`。**设计表修正**：SSE 并非缺 limiter 而是 `isSSERequest` 不覆盖 ConnectRPC binary 主形态——净效果无界，缺口 G-POST2-1/G-POST2-2 待裁决登记）。LOWPRI-SWEEP-1 ✅done（51b01d87）。LOWPRI-SWEEP-2 ✅done（5c855630）。i18n/VM 债系已收官。（相位1 zh-tw bf35cc82+67b80ad8；相位2 ja+vi 3eb5c69a+2388e919——五 locale strict 全 0/0、pre-commit 全量 strict、守卫测试 9 绿）。剩余 🟦open 全为暂缓/低优：VM-LIVE-MTF-1（暂缓需求驱动）/TRON-SECURITY-1（业主暂缓）/FEAT-3（roadmap 待拍板）。
- **清扫上翻**: 无私有记忆需清扫。

## 活跃 registry 条目指针

> 完整明细见 `docs/audits/tech-debt-registry.md`。这里只列当前活跃（🟦open / ⚠️待独立复审）条目。

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
- **PY-DECIMAL-CTOR-1** ✅done — `Decimal(...)` 字面量折叠+非法编译拒绝（Devin CLI 验收通过 2026-09-18，commit e928722c+14591f22）
- **QS-1.6** ✅done — ConfirmByAuthoritativeRead 状态机迁移（Devin CLI 验收通过 2026-09-16）
- **QS-1.3** ✅done — Python 函数局部作用域（Devin CLI 验收通过 2026-09-16，commits 9940eda4+ee47292d）
- **PY-SCOPE-KNOWN-1** ✅done — Python 作用域已知限制文档化+3 pin+Agent 规则（Devin CLI 验收通过 2026-09-18，commit 373ca8d6，零行为变更）
- **QS-1.2a** ✅done — lastError 三 builtin（Devin CLI 验收通过 2026-09-16，commit 65e2cccf）
- **QS-1.7-INV** ✅done — ClientID 不可回显→QS-1.7 不立项，维持 fail-closed（Devin CLI 验收通过 2026-09-16）
- **QS-3-BASELINE** ✅done — 基线已落盘 `docs/audits/vm-perf-baseline-2026-09.md`；无项触及 >30% 阈值，生产 p99 待 metric 上线观测
- **RECONCILE-TZ-WINDOW-1** ✅done — `.UTC()` 修复+pin 测试，Devin CLI 验收通过 2026-09-16（1efbf678）
- **TZ-SWEEP-AFFECTED-1** ✅done — analyticsSince()+worker 归一化，Devin CLI 验收通过 2026-09-16（362d285e）
- **TZ-MIXED-ENCODING-1** ✅done — ~50 站 .UTC() 止血+migration 278 回填，Devin CLI 验收通过 2026-09-16（da85f973）；**部署注记**：migration 在 backend 启动时跑，生效后 trade_records 全列 UTC
- **TZ-PAIRED-CST-COLS-1** ✅done — CST 配对列规则文档化（Devin CLI 验收 2026-09-19，commit aed6ff70；constraints+pitfalls+catalog 4 注记+4 写入点注释，零行为变更机械实证）
- **VM-RUNTIME-FAILCLOSED-2** ✅done — 静默算术/栈/槽位 fail-closed（Devin CLI 验收通过 2026-09-16，commit 4fea9439，独立 mutation×4 RED→GREEN）
- **VM-ARRAY-OOB-FAILCLOSED-1** ✅done — 数组 OOB+局部负编码 fail-closed（Devin CLI 验收 2026-09-18，commit bdb3733f，独立 mutation×4）；明细 registry 行 217
- **VM-ENUM-NUMBERING-1** ✅done — SymbolInfo*/MarketInfo 全枚举对齐+静默错标修复+SymbolInfoString 重分类（Devin CLI 验收 2026-09-18，commit 477e8273+bfb42ea3 返修，独立 mutation×5）；明细 registry 行 219
- **VM-GLOBAL-ARRAY-DECL-1** ✅done — 全局数组端到端修复+4 错标点编译期显式拒（Devin CLI 验收 2026-09-18，commit da902af6，独立 mutation×4）；明细 registry 行 220
- **LIVE-ACCOUNT-FIELDS-1** ✅done — live runner Account() 四字段补全（Devin CLI 验收 2026-09-19，commit 45767c9f）：proto 31-33+accountIdentityLookup+live fail-closed+MT4=hedging/MT5=""+三站接线+T1-T5；独立 mutation M1/M3/M4 RED，M2 双层冗余不可观测核实准确；明细 registry 行 221
- **ACCOUNT-MARGIN-LEVEL-PCT-1** ✅done — ACCOUNT_MARGIN_LEVEL 比率→百分比 ×100（Devin CLI 验收 2026-09-19，commit ac509e20）：`Mul(decimalHundred)`+官方示例 pin 992.124+零边界保留；独立 mutation M1 RED（9.92124 复活）；明细 registry 行 224
- **DATA-TRUTH-3** ✅done — 死列 last_checked_at 删除+v2 凭据-only 视图标注（Devin CLI 验收 2026-09-19，commit de1d0975）：migration 279+sqlc 生成物等价手编（138/142 历史 migration 阻断独立复现）+引用清零+constraints 双源规则；明细 registry 行 84
- **MT5-ACCMETHOD-ADAPTER-1** ✅done — mtapi AccMethod→margin_mode 全栈通道（Devin CLI 验收 2026-09-19，commit 4578cb4e）：mt5 accMethodToString+mt4 恒 hedging+migration 280+NULLIF 写入+四列直读优先 MT4 兜底+T1-T4；独立 mutation M1/M2 精确 RED；明细 registry 行 223
- **ORDERSEND-NILBROKER-FAILCLOSED-1** ✅done — 12 交易写站点 signalMode 前移+nil-broker→fatal（Devin CLI 验收 2026-09-18，commit 2739f100，独立 mutation×3）；明细 registry 行 215
- **TRADE-BUILTIN-ERR-SWALLOW-1** ✅done — channel-split：err=infra→fatal/RetCode≠done→false+_LastError/""→fatal；13 站三态+SimBroker 搬迁+engine RetCode 日志（Devin CLI 验收 2026-09-18，commit 6eae8160，独立 mutation×4）；明细 registry 行 222
- **VM-FUNC-FATAL-DELAY-1** ✅done — executeCallUser 循环顶 fatalError 检查覆三泄漏路径（Devin CLI 验收 2026-09-18，commit de6f672c+6ef18536，独立 mutation×2）；明细 registry 行 218
- **VM-HONESTY-3-REVIEW** ✅done — 死分支解耦+R06 非致命对抗测试重构（Devin CLI 验收通过 2026-09-16，commit 5816d7e9，独立 mutation×2 RED→GREEN，零生产代码改动）
- **VM-COMPILER-SEMANTICS-3** ✅done — switch default 顺序+break 栈清理（Devin CLI 验收通过 2026-09-16，commit c5d1a7e0，独立 mutation×2 RED→GREEN）
- **VM-API-TRUTH-1** ✅done — 5 批全 Devin CLI 独立复审验收：46 API 重分类 StatusUnsupported（e97a43b8/8f946579/1fb352f1/a306f54e/69d2330b）+批次2c AccountInfo* 假分支 fail-closed+枚举对齐+批次2e 4 实接（52add8ed/69d2330b）；残余同族债 VM-ENUM-NUMBERING-1/LIVE-ACCOUNT-FIELDS-1 另立跟踪

## 最近变更日志

> 完整历史见 `docs/audits/handover-audit-plan.md` + `docs/handoff/LOG.md`。

- 2026-09-17~09-19 **VM 批七项 ✅done + registry 簿记修正 + MQL-LOOP-4/LLM-CONFIG-1 条目漂移翻正 + i18n 债系收官**——均已滚出至 `docs/handoff/LOG.md`（VM-API-TRUTH-1 收官/VM-ENUM-NUMBERING-1/VM-GLOBAL-ARRAY-DECL-1/ORDERSEND-NILBROKER/TRADE-BUILTIN-ERR-SWALLOW/VM-FUNC-FATAL-DELAY/TEST-WAITSTATE/SNAPSHOT-SLICE-ALIAS/簿记修正 20 处续行格/hook 续行校验修复/MQL-LOOP-4 弱 pin 补强/LLM-CONFIG-1 死字段 43f1e20a/I18N-MIXED-2 相位2 ja+vi 3eb5c69a+2388e919 五 locale strict 全 0/0；backend 批末回归 3663 绿/4 失=3 DB 环境缺+1 审计侧样品换新）。
- 2026-09-19 **LOWPRI-SWEEP-2 ✅done**（5c855630——CQ-11 死簇删净复核门零命中；CQ-12 242 行 lint 转绿自 0d52f0a6 起存量红消除） — CQ-11 internal/ai 传递性死簇删除（backend -289 行：两整文件+strategy_prompt.go 死成员）+ CQ-12 WorkspaceCenterColumn 拆修（313→242 行+4 新文件）；Devin CLI 独立复审全绿：复核门重跑零有效命中+build/vet/test 绿+eslint src 零输出+lint exit 0[存量红消]+tsc 0+vitest 217 绿+check-lines 0 errors+diff --check 净；明细 registry。

> 2026-09-08 及更早的变更日志（FIX-2026-09-08-TEMP-RETRY/FIX-2026-09-08-BYOK-MODEL-PICKER/VM-TRADE-CONTEXT-1/2 ✅done、LIVE-ORDER-REENTRY-1-R4-REVIEW ✅done、VM-CACHE-INTEGRITY-1/2 ✅done、DATA-TRUTH-2b ✅done、三个 spec 落档、D-REVERT-SCOPE-DRIFT-001、D-REVERT-CLEANUP-001、治理结构重构、D-006/D-007、VM-CACHE-INTEGRITY-1/2 commit、LIVE-ORDER-REENTRY-1 R4 commit、第三/四批施工提示词落档、VM-COMPILER-SEMANTICS-1 + BT-FUNC-ENTRYPC-FWD ✅done、第四批施工提示词落档）已滚出至 `docs/handoff/LOG.md` + `docs/audits/handover-audit-plan.md`。
