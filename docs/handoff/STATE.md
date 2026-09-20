# STATE — 当前状态 + 交接负载（T0）

> **轻量交接负载**。技术债务明细在 `docs/audits/tech-debt-registry.md`，本文件只放当前活跃条目指针。
> 收工必更新本文件（pre-commit 强制）。≤ 20KB。

## 交接负载

- **现状**: **VM-API-TRUTH-1 整债 ✅done 收官**（46 API 重分类，明细 registry）。VM-ARRAY-OOB-FAILCLOSED-1 ✅done（bdb3733f）。VM-ENUM-NUMBERING-1 ✅done（477e8273+bfb42ea3）。**VM-GLOBAL-ARRAY-DECL-1 ✅done**（da902af6）。**ORDERSEND-NILBROKER ✅done**（2739f100）/**TRADE-BUILTIN-ERR-SWALLOW ✅done**（6eae8160）/**VM-FUNC-FATAL-DELAY ✅done**（de6f672c+6ef18536）/**TEST-WAITSTATE-ACQUIRE-BCAST ✅done**（53e886e9）/**SNAPSHOT-SLICE-ALIAS-1 ✅done**（40148ede）/**PY-DECIMAL-CTOR-1 ✅done**（e928722c）。**PY-SCOPE-KNOWN-1 ✅done**（373ca8d6）。**TZ-PAIRED-CST-COLS-1 ✅done**（aed6ff70）。**LIVE-ACCOUNT-FIELDS-1 ✅done**（45767c9f）。**ACCOUNT-MARGIN-LEVEL-PCT-1 ✅done**（ac509e20，Devin CLI 验收 2026-09-19）。**DATA-TRUTH-3 ✅done**（de1d0975）。**MT5-ACCMETHOD-ADAPTER-1 ✅done**（4578cb4e，Devin CLI 验收 2026-09-19）。**MQL-LOOP-4 ✅done**（条目漂移翻正+弱 pin 补强）。**LLM-CONFIG-1 ✅done**（条目漂移翻正，43f1e20a 已修复）。**MQL-COMPILER-LOCAL-ARRAYS ✅done**（5c947ce3，Devin CLI 验收 2026-09-19——局部数组端到端+ArrayResize 槽回写+initializer 拒收，mutation×4 实证）。 **VM-LIVE-PARITY-F1/F2/F3 ✅done**——d39afc62+审计侧修补，M1/M2/M3 mutation 实证。**VM-LIVE-VENUE-1 ✅done**（dfd9eccd，Devin CLI 复审验收——M1/M2/M3 独立 mutation 全 RED→恢复 GREEN；mt5/orders.go 450 贴线裁决存量债另立 CODE-SIZE-MT5-ORDERS-1 open）。**backend 已部署 healthy**——Dockerfile stale `COPY configs`+entrypoint `-config` 残留修复（8fdc5ff5 删 configs 后遗留）；go-build cache 18G 清盘；migration 278 部署实锤两处盲区修复（hash 触发器 session_replication_role 旁路+879 重复行 NOT EXISTS 守卫→UPDATE 10623，留残 TRADE-RECORDS-DUP-1）。
- **方向校验**: ✅ 与 AGENTS.md §1 一致（策略市场平台）。
- **施工表**:

| 子任务 | 状态 | 锚点 |
|--------|------|------|
| VERIFY-CHAIN-SEMANTIC-1 交易台账 hash 链端到端语义修正（union 全局验链+写侧 hash 输入规范化+双编码重算） | ⚠️待独立复审 | 施工完成（S1 ChainBreak+AccountID；S2 insertWithHashChain RETURNING 扩列 hash 覆盖列范式表示；S3 VerifyGlobalChain union 双侧 seq 走查+双编码重算+unhashed 披露+42P01 回退，VerifyChain 保留签名全局过滤，删 dedup 豁免，验链子系统拆 trade_record_verify.go；S4 T1-T9+dedup_verify 改写归并）。副本实测 chain_break=0 硬断言过/hash_mismatch=4,599（dump vintage 差：副本 log=3,903 vs 产库 3,920，17 行×31.96% 失配率精确对账；设计基线 4,605 保留断言待复审方产库克隆裁决）；M1-M5 归复审方。待 Devin CLI 复审 |
| TRADE-RECORDS-DUP-1 trade_records 去重（879 对 CST/UTC+3,024 epoch 幻影+160 BALANCE 混入；删行归档+写入止血+VerifyChain 豁免） | ✅done | Devin CLI 独立复审验收 2026-09-19；施工 7fce4558；产数据克隆实证（ant_dedup_review：DELETE 879/3,024 精确、log=3,903、down 后全表逐字节相等）+独立 mutation×3 全恢复 GREEN。**已部署**——产库实清 epoch=0/log=3,903；**复审抓出第三出血口**：reconciliation 幽灵单 ImportBrokerOrder `!IsZero()` 守卫对 epoch 失效（Unix(0,0)≠Go 零值），部署后仍回潮 17 行→审计侧补救 SyncableClosedTrade 加 `Unix()<=0`+第三站点接单源守卫+pin 锁（M-D mutation RED→GREEN），二次部署止血中 |
| VM-LIVE-PARITY-F1/F2/F3 实盘对账修复（F1 OrderResult 回显请求值/F2 SymbolParams 瘦数据链/F3 comment 丢链） | ✅done | Devin CLI 独立复审验收 2026-09-19；施工 d39afc62+审计侧 S5 修补（brokerImpl.SymbolInfo harness 7 字段死存储补齐+HarnessFullFacts pin）；M1/M2/M3 独立 mutation 复验 RED→GREEN；残余 R1-R4 登记 |
| VM-LIVE-VENUE-1 venue 常量真值化（F 系残余 R2） | ✅done | Devin CLI 验收 2026-09-19，dfd9eccd；M1/M2/M3 独立 mutation 实证；已部署+RPC 边界复验 tradeMode=2；明细 registry 行 31 |

- **阻塞/待决策**: TRON-SECURITY-1 业主暂缓（不做）。R1（signal-mode sentinel）/R4（CloseBy dispatch）裁决：R1 属架构语义变更单独立项评审，R4 为缺功能待需求驱动。TRADE-RECORDS-DUP-1 ✅done 已部署——产库 epoch=0/log=3,920（879 CST 侧+3,024 迁移期幻影+17 部署窗口期回潮残余全归档清除）；复审抓出第三出血口（reconciliation ghost→ImportBrokerOrder IsZero 对 epoch 失效）已止血，同一 ghost ticket 393912977 修复后实测零幻影写入。VERIFY-CHAIN-SEMANTIC-1 施工完毕（⚠️待独立复审，勿部署；副本实测与 4,605 基线的 6 行差=克隆 vintage 精确对账）。副本侧观察：trade_records 无 user_id FK（150 意图未落实仅 NOT NULL——另债观察）。
- **下一步**: **TRADE-RECORDS-DUP-1 ✅done 已部署+产线止血实证**——migration 281 产库应用（879 CST 侧+3,024 epoch 删行归档 log=3,920）、三写入路径守卫全闭（两 sync 站+ImportBrokerOrder epoch 语义修复）、VerifyChain deleted_link 豁免上线；同一 ghost ticket 393912977 修复前后对照实证零幻影写入。此后：VERIFY-CHAIN-SEMANTIC-1 独立复审（施工已完，M1-M5 mutation 由复审方执行）/R1（signal-mode sentinel 架构评审）/暂缓系（VM-LIVE-MTF-1/TRON-SECURITY-1/FEAT-3）。此前：**demo `904d14e6` F1/F2/F3 有界复验 ✅done**（BTCUSDm 1 单 comment 落 broker+SymbolParams 非零+fill 81227.46 事实回读，-0.09 USD 已全平）。此前：**backend 部署完成 healthy**（F1/F2/F3+G-POST2+边界批全上线）；**VM-LIVE-PARITY-1 实盘对账完成**（vm-live-parity-results-2026-09.md——Exness-Trial demo BTCUSDm 实测：P1 回显实锤双轴/P4 瘦数据实锤/拒绝 fail-closed 实证/SL 全链实证；新债 F1 回显/P2 F2 瘦数据/P4 F3 comment 编码登记 open）。G-POST2-1/2 ✅done（295d93f8）。POST-2 探针批 ✅done（2408d34c——容量基线落档 docs/benchmarks/post2-capacity-baseline-2026-09.md）。LOWPRI-SWEEP-1/2 ✅done。i18n/VM 债系收官。剩余暂缓/低优：VM-LIVE-MTF-1/TRON-SECURITY-1/FEAT-3。
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
- **VM-LIVE-PARITY-F1/F2/F3** ✅done — 实盘对账修复（Devin CLI 验收 2026-09-19，d39afc62+审计侧修补，M1/M2/M3 mutation 实证）；明细 registry 行 28-30
- **VM-LIVE-VENUE-1** ✅done — venue 常量真值化 -1=unknown 哨兵全链+TRADEALLOWED 双轴（Devin CLI 验收 2026-09-19，dfd9eccd，M1/M2/M3 独立 mutation；**已部署**+demo RPC 边界复验 tradeMode=2 真值）；明细 registry 行 31
- **CODE-SIZE-MT5-ORDERS-1** ✅done — mt5/orders.go 450→348 拆分（Devin CLI 直接施工+验收 2026-09-19：符号元数据三函数 verbatim move→symbol_params.go，diff 逐字节一致=零行为变更）；明细 registry 行 32
- **TRADE-RECORDS-DUP-1** ✅done — 去重+写入止血+VerifyChain 豁免（Devin CLI 验收 2026-09-19，7fce4558，产数据克隆实证+mutation×3）；**已部署**（91ba089d——migration 281 产库应用+第三出血口 ImportBrokerOrder epoch 语义修复+残余 17 行归档清除，产库 epoch=0/log=3,920）；明细 registry 行 33
- **VERIFY-CHAIN-SEMANTIC-1** 🟦open-待施工 — 设计 v2+派工单落档（对抗审计扩面：根因 B 浮点编码——写侧 hash 覆盖 NewFromFloat 全精度串不可重建，union 实测 9,804 可重算/4,605 不可确认；方案=union 验链+写侧 RETURNING::text 规范化+双编码重算）；明细 registry 行 34

## 最近变更日志

> 完整历史见 `docs/audits/handover-audit-plan.md` + `docs/handoff/LOG.md`。

- 2026-09-17~09-19 **VM 批七项 ✅done + 簿记修正 + MQL-LOOP-4/LLM-CONFIG-1 翻正 + i18n 收官**——已滚出 `docs/handoff/LOG.md`（明细见该文件同日期段）。
- 2026-09-19 **LOWPRI-SWEEP-2 ✅done**（5c855630——CQ-11 死簇删净复核门零命中；CQ-12 242 行 lint 转绿自 0d52f0a6 起存量红消除） — CQ-11 internal/ai 传递性死簇删除（backend -289 行：两整文件+strategy_prompt.go 死成员）+ CQ-12 WorkspaceCenterColumn 拆修（313→242 行+4 新文件）；Devin CLI 独立复审全绿：复核门重跑零有效命中+build/vet/test 绿+eslint src 零输出+lint exit 0[存量红消]+tsc 0+vitest 217 绿+check-lines 0 errors+diff --check 净；明细 registry。

> 2026-09-08 及更早的变更日志（FIX-2026-09-08-TEMP-RETRY/FIX-2026-09-08-BYOK-MODEL-PICKER/VM-TRADE-CONTEXT-1/2 ✅done、LIVE-ORDER-REENTRY-1-R4-REVIEW ✅done、VM-CACHE-INTEGRITY-1/2 ✅done、DATA-TRUTH-2b ✅done、三个 spec 落档、D-REVERT-SCOPE-DRIFT-001、D-REVERT-CLEANUP-001、治理结构重构、D-006/D-007、VM-CACHE-INTEGRITY-1/2 commit、LIVE-ORDER-REENTRY-1 R4 commit、第三/四批施工提示词落档、VM-COMPILER-SEMANTICS-1 + BT-FUNC-ENTRYPC-FWD ✅done、第四批施工提示词落档）已滚出至 `docs/handoff/LOG.md` + `docs/audits/handover-audit-plan.md`。
