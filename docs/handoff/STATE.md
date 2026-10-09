# STATE — 当前状态 + 交接负载（T0）

> **轻量交接负载**。技术债务明细在 `docs/audits/tech-debt-registry.md`，本文件只放当前活跃条目指针。
> 收工必更新本文件（pre-commit 强制）。≤ 20KB。

## 交接负载

- **现状**: **VM-API-TRUTH-1 整债 ✅done 收官**（46 API 重分类，明细 registry）。VM-ARRAY-OOB-FAILCLOSED-1 ✅done（bdb3733f）。VM-ENUM-NUMBERING-1 ✅done（477e8273+bfb42ea3）。**VM-GLOBAL-ARRAY-DECL-1 ✅done**（da902af6）。**ORDERSEND-NILBROKER ✅done**（2739f100）/**TRADE-BUILTIN-ERR-SWALLOW ✅done**（6eae8160）/**VM-FUNC-FATAL-DELAY ✅done**（de6f672c+6ef18536）/**TEST-WAITSTATE-ACQUIRE-BCAST ✅done**（53e886e9）/**SNAPSHOT-SLICE-ALIAS-1 ✅done**（40148ede）/**PY-DECIMAL-CTOR-1 ✅done**（e928722c）。**PY-SCOPE-KNOWN-1 ✅done**（373ca8d6）。**TZ-PAIRED-CST-COLS-1 ✅done**（aed6ff70）。**LIVE-ACCOUNT-FIELDS-1 ✅done**（45767c9f）。**ACCOUNT-MARGIN-LEVEL-PCT-1 ✅done**（ac509e20，Devin CLI 验收 2026-09-19）。**DATA-TRUTH-3 ✅done**（de1d0975）。**MT5-ACCMETHOD-ADAPTER-1 ✅done**（4578cb4e，Devin CLI 验收 2026-09-19）。**MQL-LOOP-4 ✅done**（条目漂移翻正+弱 pin 补强）。**LLM-CONFIG-1 ✅done**（条目漂移翻正，43f1e20a 已修复）。**MQL-COMPILER-LOCAL-ARRAYS ✅done**（5c947ce3，Devin CLI 验收 2026-09-19——局部数组端到端+ArrayResize 槽回写+initializer 拒收，mutation×4 实证）。 **VM-LIVE-PARITY-F1/F2/F3 ✅done**——d39afc62+审计侧修补，M1/M2/M3 mutation 实证。**VM-LIVE-VENUE-1 ✅done**（dfd9eccd，Devin CLI 复审验收——M1/M2/M3 独立 mutation 全 RED→恢复 GREEN；mt5/orders.go 450 贴线裁决存量债另立 CODE-SIZE-MT5-ORDERS-1 open）。**backend 已部署 healthy**——Dockerfile stale `COPY configs`+entrypoint `-config` 残留修复（8fdc5ff5 删 configs 后遗留）；go-build cache 18G 清盘；migration 278 部署实锤两处盲区修复（hash 触发器 session_replication_role 旁路+879 重复行 NOT EXISTS 守卫→UPDATE 10623，留残 TRADE-RECORDS-DUP-1）。**ENV-TO-PG-1 ✅done**（d09f6795，Devin CLI 验收 2026-10-09——业务配置唯一真相=PG；未部署，migration 282 随下批镜像走）。
- **方向校验**: ✅ 与 AGENTS.md §1 一致（策略市场平台）。
- **施工表**:

| 子任务 | 状态 | 锚点 |
|--------|------|------|
| （2026-09-19~09-20 六项 ✅done 已滚出 LOG.md：VERIFY-CHAIN-SEMANTIC-1/TRADE-RECORDS-DUP-1/VM-LIVE-PARITY-F1F3/VM-LIVE-VENUE-1/KB-SEED-DRIFT-1/VM-LIVE-SYNC-DISPATCH-1——全已部署，明细 registry 行 28-36） | — | LOG.md 2026-09-19~20 段 |
| ENV-TO-PG-1 env 收口 PG——业务配置唯一真相=PG（C 档 21 键→system_config / D 档 10 键→platform_secrets 密文轨 / A·B 档留 env；seed-once+DB-wins overlay+check-env-reads 门禁入 pre-commit） | ✅done | Devin CLI 独立复审验收 2026-10-09，施工 d09f6795；ADR-0031+派工单 docs/plan/2026-10-env-to-pg-consolidation.md；独立重跑：build/vet/gofmt/check-lines 0 ERROR/race 53 测/integration 5/5（arb-postgres-test ant_boot_it 独立库）；独立 mutation×4 RED→GREEN（seed 覆写/overlay 摘除/解密吞没/门禁探针）；残余观察 3 项留档 registry；部署验证随 CI-RESTORE-1 批镜像 |
|| SG-PROD-CLEANUP-1 sg 生产机开发残留清理（sg=ant+arb 双项目生产机，业主裁决 arb 整机不碰） | ✅done | Devin CLI 直接执行+终态验收 2026-10-09（业主改令不派 zcode）；磁盘 40G→34G（69%→58%，释放 ~6G）；12 容器零中断全 healthy；脏目录 quarantine 归档非删除（/root/cleanup-quarantine 245M）；S9 盘点项待业主裁决 ~1.5-2.3G（claude 残留 1.1G/go-bin 346M/docker tagged-unused 742M） |
| CI-RESTORE-1 CI 转绿——govulncheck/trivy 漏洞清零 + CQ-12 拆分遗留 tsc 修复 | ✅done | Zcode 施工 2026-10-09（业主令"github 一直在报错"）：CI 自 09-08 红一月。ada7e10e=grpc v1.83.2（CVE-2026-84445）+x/net v0.60.0+otel v1.45.0（GO-2026-6505）+Go 1.26.6→1.27.2 全链（go.mod/ci/nightly/security-scan/Dockerfile），govulncheck 12 affected 归零+trivy HIGH 清零；frontend tsc 20 错=LOWPRI-SWEEP-2 CQ-12 抽出组件手写 Props 假形状（unknown[]/string 字段/错路径/重复 import），改引 BacktestSummary/BtSummary·RecentSummary/QuickTradePosition·RecentTrade·AccountMeta 真类型+tsc(app/base) 0 错。**机器分工定**：本机=开发机、sg=生产部署机（constraints.md 部署节+ssh alias sg） |

- **阻塞/待决策**: TRON-SECURITY-1 业主暂缓（不做）。R4（CloseBy dispatch）为缺功能待需求驱动。副本侧观察：trade_records 无 user_id FK（150 意图未落实仅 NOT NULL——另债观察）。
- **下一步**: ①SG-PROD-CLEANUP-1 ✅done——S9 盘点明细在 LOG.md 待业主裁决（agent 残留/go-bin/docker tagged-unused ~1.5-2.3G）②ENV-TO-PG-1+CI-RESTORE-1 部署至 sg 进行中（2026-10-09 业主放行）：build→up→backend healthy+migration 282 boot seed 运行时证据；生产脏改动已保护性 stash（sg:aff4aebf）。剩余外部触发：R4·VM-LIVE-MTF-1（需求）、FEAT-3（产品决策）、TRON 系（业主）。
- **清扫上翻**: 2026-10-09 ENV-TO-PG-1 验收收工——条目转 ✅done（独立复审证据在 registry 行），待部署验证项移交 sg 下批镜像。

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
- **VM-STATIC-LOCAL-1** ✅done（Devin CLI 复审 2026-09-21，0af39991）— 函数内 static 脱糖 mangled-global+init-guard（懒初始化）；独立 mutation M1/M3 RED→GREEN；**已部署+实盘复验**（BTCUSDm 探针 counter 1→90 跨 tick 持久、acc 累积、初始化器单次）；明细 registry 行 47
- **VM-BLOCK-SCOPE-1** ✅done（Devin CLI 复审 2026-09-21，施工 2f188877）— IR 层 pushScope 包裹 5 插入点；B1–B12+错误形状断言（拒收全纯读位）；独立 mutation M1/M2/M3 亲手复红→恢复零偏差；mql2go 811 绿、race×3、CompatScan 20 CLEAN；**已部署+实盘复验**（拒收探针 `unknown variable: n` 编译拒、接受探针 a=1/b=2/cnt=3 全对）；明细 registry 行 48
- **LIVE-POS-SNAPSHOT-LAG-1 + LIVE-HISTORY-POOL-1** ✅done — Runner confirmed-mutation 保留窗（120s）：已确认开/平票号不被滞后快照回退/复活（OrdersTotal 闪烁→实盘超开仓根因，XAUUSD 实证 36 仓）；HistoryOrders 经 session 暂存注入 Start() 后接 mtHub.OrderHistory，MODE_HISTORY 实盘真值（BTCUSDm hist=1051 / XAUUSD hist=131、OrdersTotal=92 全量枚举实证）；mutation×2 RED→GREEN；明细 registry 行 45-46
- **VM-LIVE-PARITY-F1/F2/F3** ✅done — 实盘对账修复（Devin CLI 验收 2026-09-19，d39afc62+审计侧修补，M1/M2/M3 mutation 实证）；明细 registry 行 28-30
- **VM-LIVE-VENUE-1** ✅done — venue 常量真值化 -1=unknown 哨兵全链+TRADEALLOWED 双轴（Devin CLI 验收 2026-09-19，dfd9eccd，M1/M2/M3 独立 mutation；**已部署**+demo RPC 边界复验 tradeMode=2 真值）；明细 registry 行 31
- **CODE-SIZE-MT5-ORDERS-1** ✅done — mt5/orders.go 450→348 拆分（Devin CLI 直接施工+验收 2026-09-19：符号元数据三函数 verbatim move→symbol_params.go，diff 逐字节一致=零行为变更）；明细 registry 行 32
- **TRADE-RECORDS-DUP-1** ✅done — 去重+写入止血+VerifyChain 豁免（Devin CLI 验收 2026-09-19，7fce4558，产数据克隆实证+mutation×3）；**已部署**（91ba089d——migration 281 产库应用+第三出血口 ImportBrokerOrder epoch 语义修复+残余 17 行归档清除，产库 epoch=0/log=3,920）；明细 registry 行 33
- **VERIFY-CHAIN-SEMANTIC-1** ✅done+已部署 c74d8d3 — 复审含根因 C 审计侧修补；产克隆 16 测试全绿+mutation×4 实证；明细 registry 行 34
- **KB-SEED-DRIFT-1** ✅done — R2 实盘探针抓出 KB 种子漂移（43 陈旧枚举压过内建表）；reconcile 化修复+mutation×2；明细 registry 行 35
- **VM-LIVE-SYNC-DISPATCH-1（R1）** ✅done — signal-mode 假票号/丢单/cancel_all 空转根治；VM 内同步派发+confirmed 事实注入+防双发；mutation×4 实证；明细 registry 行 36
- **实盘探针四缺陷批** ✅done — VM-IMPLICIT-VAR-READ-1（读位严格化+`int x;` 裸声明修复+clrNone/20 web 色补齐，真实策略扫描通过）/VM-ERR-CODE-COLLAPSE-1（typed BrokerRejectError 三段透传 lastError=native code）/RISK-DEDUP-KEY-1（Guard+Gate 双层 key 纳 account+magic+comment+真 type）/ACCOUNT-TRADE-ALLOWED-DEAD-1（connected 谓词）；各带 mutation RED→GREEN；明细 registry 行 40-43
- **ENV-TO-PG-1** ✅done — env 收口 PG，业务配置唯一真相=PG（Devin CLI 验收 2026-10-09，施工 d09f6795；独立 mutation×4+门禁探针实证；未部署——migration 282 随下批镜像）；明细 registry ENV-TO-PG-1 行

## 最近变更日志

> 完整历史见 `docs/audits/handover-audit-plan.md` + `docs/handoff/LOG.md`。

- 2026-09-17~09-19 **VM 批七项 ✅done + 簿记修正 + MQL-LOOP-4/LLM-CONFIG-1 翻正 + i18n 收官**——已滚出 `docs/handoff/LOG.md`（明细见该文件同日期段）。
- 2026-09-19 **LOWPRI-SWEEP-2 ✅done**（5c855630）— CQ-11 internal/ai 死簇删净（-289 行）+CQ-12 WorkspaceCenterColumn 313→242 lint 存量红消；Devin CLI 独立复审全绿；明细 registry。

> 2026-09-16 的 VM-HONESTY-3-REVIEW / VM-COMPILER-SEMANTICS-3 / VM-API-TRUTH-1 ✅done 明细已滚出至 LOG.md（09-16 段含验收记录）；registry 行保留。
> 2026-09-08 及更早的变更日志（FIX-2026-09-08-TEMP-RETRY/FIX-2026-09-08-BYOK-MODEL-PICKER/VM-TRADE-CONTEXT-1/2 ✅done、LIVE-ORDER-REENTRY-1-R4-REVIEW ✅done、VM-CACHE-INTEGRITY-1/2 ✅done、DATA-TRUTH-2b ✅done、三个 spec 落档、D-REVERT-SCOPE-DRIFT-001、D-REVERT-CLEANUP-001、治理结构重构、D-006/D-007、VM-CACHE-INTEGRITY-1/2 commit、LIVE-ORDER-REENTRY-1 R4 commit、第三/四批施工提示词落档、VM-COMPILER-SEMANTICS-1 + BT-FUNC-ENTRYPC-FWD ✅done、第四批施工提示词落档）已滚出至 `docs/handoff/LOG.md` + `docs/audits/handover-audit-plan.md`。

- 2026-10-09 **TASK env-to-pg 派单**（arb 同类收口平移，业主令）— 详单 docs/plan/2026-10-env-to-pg-consolidation.md：env 四档分类（引导件/构建期/业务旋钮→PG/秘密件→加密轨）+seed-once+Getenv门禁，Zcode 实施
