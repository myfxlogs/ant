# 接手审计计划书（Handover Ground-Truth Audit）

> 新任全权负责人接手旧项目的"地面真相"审计。目标：把继承的、已多次证伪的"项目状态图景"，逐条替换为**自己对着代码验证过的事实**，作为后续一切建设/修复决策的地基。
>
> **原则**：块结构已验证（11 块目录全在）；7 管线是"待验证假设"（#7 已证实在接缝处断开）。审计管线 = 把假设变事实。

---

## 范围

7 管线 + account-mgmt（覆盖 11 功能块）。管线是穿越多块的代码路径，account-mgmt 是 CRUD 支撑块（不在管线内，单独轻审）。

| # | 管线 | 覆盖块 | 审计方法 | 状态 |
|---|------|--------|---------|------|
| 6 | 实盘调度 | frontend, api-gateway, strategy-runtime, market-data, mt-gateway | 策略收 bar → 出信号 → 风控 → OMS → MT 下单 | ✅ 已审（2026-08-07）|
| 1 | 行情引入 | mt-gateway, market-data | MT 连接 → 去重/质量/归一化 → NATS+PG | ✅ 已审（2026-08-07）|
| 2 | 策略执行 | market-data, strategy-runtime, risk-gate, oms, mt-gateway | bar → runner → 6门 → 16状态机 → 下单 | ✅ 已审（2026-08-07）|
| 3 | 订单对账 | mt-gateway, mthub, oms | 订单事件 → 幂等门/对账门 → 状态更新 → PnL | ✅ 已审（2026-08-07）|
| 5 | 回测 | frontend, api-gateway, backtest-engine | 参数 → VMRunner+SimBroker → PG → SSE（接 ADR-0028 防线）| ✅ 已审（2026-08-07）|
| 4 | Agent循环 | frontend, api-gateway, agent-engine, mql-compiler, backtest-engine | 用户输入 → generate/revise → compile → backtest → 迭代 | ✅ 已审（2026-08-07，bridge/credit/memory 补细节审完）|
| 7 | 策略市场 | frontend, api-gateway, strategy-marketplace, backtest-engine, agent-engine | 发布/购买/冻结结算（购买→实盘断链已定位）| ✅ 已审 |
| — | account-mgmt | connect/{gateway,user} | MT 账户 CRUD/经纪商/用户（memory 称已完工最优，验真）| ✅ 已审（2026-08-07，轻审无 bug）|

---

## 每条管线审什么（统一四问）

沿管线逐跳（入口数据 → 每跳变换 → 出口落地的状态），对着代码问：

1. **通不通**：端到端真能跑吗？数据真的从入口流到出口？（#7 已答：购买→实盘在 schedule→live 接缝断）
2. **边界对不对**：错误/空值/并发/重试/精度（Decimal）/幂等 —— 异常路径是否被吞？
3. **旧审计落位**：经过该管线的 🔴/🟡 审计发现，对着代码定 ✅已修/🟦真open。
4. **文档对账**：该块 living 文档（状态表/"已落地✅"/README/`CAPABILITIES.md`）声称与代码一致吗？不一致即修；被证伪的历史文档(ADR 正文/dated 审计)加 `verified YYYY-MM-DD` 注，**不重写**。

每条管线产：**真实状态 + 实证 gap 清单**。gap 进 `tech-debt-registry.md`。

---

## 文档对账原则（防范围爆炸 + 防毁决策史）

接手期文档已多次迭代、多处过时（本会话已实证 6 处漂移：ADR-0028 §7、acceptance Gate2、AUD-MK1 等）。对账必要，但分两类：

- **活文档**（断言当前状态：状态表、落地表、README、CAPABILITIES、registry、memory）→ **必须对齐**，漂移即危害。
- **历史记录**（ADR 正文论证、dated 审计报告、retrospective）→ **只加注、不重写**。改历史 = 伪造决策轨迹。ADR 的状态/落地表是活的（对齐），正文是历史（加注）。
- **豁免**：① 即将大改写的块（如购买→实盘），现在对齐它的文档是浪费，改完再说；② 零流量文档跳过。
- 对账是审计的**副产品**（人在代码里、手握事实，边际成本≈0），**不单开文档重写项目**。新写的 registry/memory/本计划即新的可信"当前状态层"。

---

## 顺序（按风险/依赖，非按编号）

`#6 补全` → `#1` → `#2` → `#3` → `#5 补全` → `#4` → `account-mgmt` → **写前瞻计划**

理由：#6 是收入命脉的执行半边且已读一半（闭环性价比最高）→ #1 是万物依赖的地基 → #2/#3 是钱路径（风控/对账，最该审）→ #5 信任基础 → #4 供给侧 → account-mgmt 验真 → 全部摸清后写前瞻。

---

## 节奏与产出

- **一条管线 = 一个工作单元**：代码级审计 → 结论进总账 → 一个 checkpoint 汇报（验证了什么/哪断/置信度/下一条）。
- **方向我定，范围你拦**：每条边界可改方向。
- **串行为主**：管线天然可并行（各自独立），但子 agent 因环境模型不可用挂过；先串行，agent 复活则并行加速。
- **不丢失**：结论落 `tech-debt-registry.md` + memory `open-items-registry.md` + 周一自续 cron（`66e2b149`）三层。

---

## 完成定义（Definition of Done）

7 管线 + account-mgmt 全部审完 → 每条有"真实状态 + gap" → 总账里所有 ❓待核 尽可能落 ✅/🟦 → **此时写前瞻计划**（建设/修复优先级，建在已验证地基上）。

---

## 变更日志

- 2026-08-06 建立。块结构已验证（11 目录全在）。#7 已审（购买→实盘断）；#6/#5 半审。下一步：#6 补全。
- 2026-08-07 #6 实盘调度补全审完。两跳通断已查实：
  - **market-data 推流**：MT → mdgateway(6 阶段 tick 管线 + BarAggregator) → `OnBar` 回调 → `mthubSvc.PublishBar` → barBroker(进程内 pub/sub) → `LiveSource.Subscribe` → runner barCh → `handleBar`。**通，全 push 无轮询。**
  - **mt-gateway 下单出口**：`handleBar` → VM → `dispatchFromBytes` → `dispatchLiveSignal` → `submitOrder`(异步 goroutine) → `mtHub.PlaceOrder` → `preTradeChecks`(killSwitch/guard/accountOwner/idem/reconcileGate/userLimiter) + `evaluatePlaceGate`(gate.Evaluate, D6-A) + `submitToBroker`(risksvc.PreCheck P0-6) → mt4/mt5 adapter → MT。**通。** accountStateProvider 已注入(`handlers_pipeline.go:65`)，gate 真跑非 fail-open。
  - **主发现 LIVE-1**：open bar 泄漏进策略执行流（`StartOpenBarTicker` 500ms 复用策略的 onBar 回调，runner 不过滤 `bar.Closed`）→ 重绘 + 窗口污染 + 实盘/回测发散。
  - 顺带改判：ARCH-1/LAUNCH-3 → ✅（GoExecutor `go run` 已移除，剩死 stub）；ARCH-2 双风控再次核验仍 open。
  - 全部落 `tech-debt-registry.md`（LIVE-1/LIVE-2 + DOC-4/DOC-5）。
  - 下一步：#1 行情引入（万物依赖的地基）。
- 2026-08-07 #1 行情引入审完。**管线稳健，无 🔴/🟡 正确性 bug**（符合 ADR-0012 PG 单存储 + M10-BASE 分层）。逐阶段：MT 连接 push-first（OnQuote gRPC stream，非轮询，重连已接）→ Normalizer（symbol 解析 + PG LISTEN 失效缓存）→ Quality（只硬 drop 不可能数据：bid>ask/非正/clock skew>5s，合法异常值只 tag，对标 LMAX）→ TickDedup（ring buffer + xxhash，排除 ArrivedUnixMs 让 live/replay 同 hash）→ BarAggregator（按**到达时间** `ArrivedUnixMs=Clk.Now()` 分桶，stream 内单调→rollover 正确，滞后 tick 误 finalize 仅 NTP 回跳极端情况）→ PgWriter（drop-based：channel 满/PG 故障 drop，crash 丢内存批次）→ Publisher（NATS best-effort，失败仅 log，PG 是真相源）。**安全网 = Backfiller**（启动扫描 + 6h cron + PG NOTIFY 新账户即触发 + 限速）覆盖所有 drop 路径。3 个特性（非 bug）：① 到达时间分桶使实盘/回测一致但偏离 broker 官方 bar；② PgWriter 无定时 flush（低流量部署 PG 可见性延迟，聚合流量+backfiller 缓解）；③ 过载 drop 由 backfiller 恢复。下一步：#2 策略执行（钱路径：bar→runner→6门→16状态机→下单）。
- 2026-08-07 #2 策略执行审完。**钱路径风险门+OMS 稳健，§1 三个 🔴 全部对账为 ✅。** Gate.Evaluate 组合正确：kill-switch 最先(R10)→autotrade(R11)→**LIVE 单 fail-closed**(`state==nil`/负净值即 block,T3.2b,在规则链前)→规则链首阻断即止(记 RuleHit/Reason/AdjustedVolume)。规则 nil-safe。OMS 16 态转移图(`oms_writer.go:60` isValidOMSTransition)完整正确,终态无出边,RECONCILING/REQUOTED/SLIPPAGE_REJECTED/MARGIN_CALL 有合理回路边。对账：**AUD-R1**✅(Gate fail-closed nil state 在前,到不了 DailyPnL)、**AUD-R2**✅(`contractSize()` 100000 fallback 已乘,~100x 少算不复现)、**AUD-MT1**✅(live 路径硬编码 Source=LIVE,Source-gating 有意,用户不可控无 IDOR)。ARCH-2 双风控仍 🟦(#6 已记)。残留 mild：parse-error→pass 的 fail-open(畸形 volume 跳过 margin/exposure 检查,broker 下游拒,低)。下一步：#3 订单对账（钱路径另一半：订单事件→幂等门/对账门→状态更新→PnL）。
- 2026-08-07 #3 订单对账审完。订单事件路径:MT OnOrderUpdate → `buildOnOrderUpdate`(`pipeline_callbacks.go:22`)发持仓快照(展示)+写**平仓** TradeRecord(仅 close)。对账：**AUD-W1-1**✅(SubscribeOrderUpdates 认证 userID+UserOwnsAccount+PermissionDenied+userID-keyed broker)、**AUD-W1-2**✅(OrderEventBroker 按 userID 隔离无跨户)、**TradeRecord 幂等**✅(DB `UNIQUE(account_id,ticket,close_time)` 兜底,重放撞约束不双计;小瑕疵:用 Create+retry 而非 ON CONFLICT→error 噪音)。**主发现 RECON-1(🟦open,偏重)**:`SyncAccountHistory` 是**空壳 stub**(只 log.Debug),`ReconciliationLoop` 检测 ghost/orphan 但**只 log.Debug 不修复**+1h 窗口 → disconnect 间隙漏单 ant 侧 trade_records 永久缺失 → 逐笔战绩少计(账户级 balance/equity 来自 profit 流仍对)。修复=实现 SyncAccountHistory 用 FetchOrderHistory upsert(ON CONFLICT)+Reconciliation 升 Warn/扩窗。下一步：#5 回测补全（接 ADR-0028 防线）。
- 2026-08-07 LIVE-1 当场修复：`live_runner.go` runLiveEventLoop bar 分支加 `if !bar.Closed { continue }` 守卫——open bar 不再进策略执行（实盘/回测语义对齐）。build+strategy 测试绿、file-lines 0🔴。CQ-9（K线图+因子死代码）另立待产品决策。审计继续 #1。
- 2026-08-07 **#5 回测审计审完**。**ADR-0028 核心防线全部 ✅ 落位且正确**：防线B 五恒等类（`backtest_invariants.go`：手数>0/资金守恒/价格>0/方向/时间序）+ assessRisk 闸门（`buildBacktestResponse` 防线B 前置，违反强制 IsReliable=false）+ 资金守恒等式与 SimBroker 记账自洽（balance=本金+ΣProfit−ΣComm−ΣSwap → 守恒检查有效）+ DEGRADED marketplace 硬阻断（`quality.go:checkDegradedStatus` 不可豁免）。管线端到端通：`StartBacktestRun`(验账户归属+quota) → worker(LISTEN backtest_pending+SKIP LOCKED, push-first) → VM 回测 → 防线B → DEGRADED → PG → `WatchBacktestRun` LISTEN backtest_status → SSE；取消走 LISTEN backtest_cancel。T5 确认回测引擎无偷看未来（`indicators_decimal.go:visibleBars` 截断 `bars[:barIdx+1]`）。**主发现 BT-6（🟦open 偏重）**：参数链 E2E 测试 `TestParamPipeline_FloatDefaultParam` flaky（`makeE2EBars` 用 `time.Now`，违反 Determinism Contract；`-count=1` 5 跑 = 3 PASS/2 FAIL；FAIL 时 `volume=0` = xianhua 精确复现）→ ADR §7"端到端测试已做"是**假闭环**，CI 零可靠守护；防线B `checkVolumeInvariant` 兜底标 DEGRADED（假成功不流出）但根因被 flaky 掩盖。另 BT-4（backtest_pending 无 NOTIFY，30s 延迟）/BT-5（DEGRADED 推送断链：notify 条件+watch 终态均漏 DEGRADED）/BT-1-3（SimBroker 简化：swap/equity/无滑点）/DOC-6-7。memory hash 笔误 30618f64→30668f64。下一步：#4 Agent循环。
- 2026-08-07 **#4 Agent循环半审**（主干+主发现）：管线✓通（`agent_loop` Think→Act→Observe → `generator_agent` → `write_strategy`(I1) → `CompilePythonWithCoverage` → `runVMBacktest` → LLM 迭代；credit PreHold/Settle + sessionQuota + memory + conversation 接上）。**主发现 AGT-2（🟡偏重）**：agent 回测 `backtest_helpers.go:buildBacktestResultProto` 完全绕过 ADR-0028 防线B（agent 包零引用 checkVolumeInvariant/assessRisk/DEGRADED），LLM 基于未校验 metrics 迭代。**AGT-1**：`lookahead_scanner` 旧 DSL 残留，对 Python/MQL 永远 skip（FEAT-2 偷看未来检测待做）。详见 registry AGT-1/AGT-2。待审细节：bridge/credit/memory。
- 2026-08-07 **#4 Agent循环补细节审完**（bridge/credit/memory 三层）：① **Bridge**：`bridge.go` TranslateWithRetry LLM→compile→coverage→backtest 重试循环功能正确，但 **BRIDGE-1（🟡）**：首次 LLM 失败时 `result==nil` → attempt 2 `buildBridgeRetryPrompt(result.PythonSource)` nil panic。**BRIDGE-2（🟢）**：bridge backtest `validateBacktest` 也绕过防线B（同 AGT-2），但结果只用于 semanticDiff 不喂 LLM，严重性低。② **Credit**：PreHold/Settle/ReleaseHold 三阶段功能正确，但 **CREDIT-1（🟡）**：`holds` 是 in-memory map，服务重启 mid-session → DB 冻结额度永久 orphaned。**CREDIT-2（🟢）**：CheckBalance fail-open on DB errors（可用性优先，特性）。③ **Memory**：三层（domain knowledge / user templates / experiences）功能正确，错误静默吞合理（memory 失败不阻断生成），fingerprint dedup 正确，无 bug。四处新 gap 入 registry（BRIDGE-1/BRIDGE-2/CREDIT-1/CREDIT-2）。**#4 审计状态：半审→审完**。下一步：account-mgmt 轻审。
- 2026-08-07 **account-mgmt 轻审完**：MT 账户 CRUD（Create/Update/Delete/List/Get）+ User CRUD 均干净。安全：所有账户查询 userID-scoped（`WHERE id=$1 AND user_id=$2`），无 IDOR；密码 `secrets.Client` 加密存储；DeleteAccount 需 broker 密码验证；CreateAccount 在 TX 外验证 MT 凭证（避免池耗尽）。User CRUD `deleted_at IS NULL` 过滤 + `IncrementTokenVersion` JWT 失效。memory 称"已完工最优"验真通过，无新 gap。
- 2026-08-07 **🔄 角色轮班**：Windsurf 转为**双角色 agent（审计+施工）**（`AGENTS.md`/`.windsurfrules` 已更新），Claude Code 轮休。三铁律保客观性（对抗证明 / 红队自审 / `⚠️待Claude复审`）。无损交接靠三层文档（registry/handover/memory）。**当前任务：写前瞻计划（DoD 收尾）+ AGT-2/RECON-1 施工**。Claude 回来读本文件 + registry + memory 即恢复，挑 `⚠️待Claude复审` 重验。
- 2026-08-07 **P0 三项全部完成+部署**：AGT-2（agent 回测接防线B）+ RECON-1（漏单修复）+ BRIDGE-1（bridge nil panic）。go build + go test 绿。Docker build + deploy 生产。
- 2026-08-07 **P1 七项全部完成+部署**：CREDIT-1（credit hold 持久化）+ AUD-U1/U3（禁用用户撤会话）+ AUD-A6/A7/A8（auth-session 三件）+ AUD-W4-1（SSRF 过滤）+ AUD-MK2（retry 双扣）+ AUD-C1（Deposit 事务）+ AUD-MT2（nil logger）。go build + go test 绿。Docker build + deploy 生产。
- 2026-08-07 **MEDIUM 全部核验完成**：19 项中 17 误报/已修，2 个已知 open（AGT-1=FEAT-2, BRIDGE-2 低）。零 ❓待核。
- 2026-08-07 **LOW + ARCH 全部核验完成**：LOW 4 项 ❓→3 误报+1 设计权衡；ARCH 2 项 ❓→1 已完成(ARCH-5)+1 误报(ARCH-6)。**Registry 零 ❓待核。**
- 2026-08-07 **ARCH-3 双模板表修复+部署**：`ScheduleEngine` 从 `AIStrategyTemplatesRepository`（读 `ai_strategy_templates`）改为 `TemplateCodeReader` 接口注入 `StrategySvc`（读 `strategy_templates`），用 `tpl.Code` 替代 `tpl.CodeSkeleton`。3 文件改动，go build + go test 绿，Docker deploy 生产。**这是购买→实盘核心路径 bug，修复后用户调度可正常执行。**
- 2026-08-07 **审计章节闭合**：Registry 状态总览 — HIGH 14/14 ✅，MEDIUM 17✅+2🟦，LOW 5✅+5🟦+2✅(ARCH)，ARCH 4✅+1🟦+1⚠️。零 ❓待核。剩余 🟦open 全部为已知特性/低风险/产品决策。**下一步方向：产品功能开发或 LAUNCH 缺口（E2E 测试/Prometheus）。**
- 2026-08-08 **FEAT-1 购买→实盘链路全部完成+部署**：7 任务全部 ✅（ARCH-3 取码源修正 + 事件型会话化 + 订阅授权闸 + 每 bar 授权复验 + 配额闸 + 集成测试 + ADR-0029）。Docker deploy 生产。
- 2026-08-09 **LAUNCH-1 Agent 策略质量基准测试套件完成**：20 策略用例（trend/mean_reversion/breakout/grid/multi_tf/oscillator），三项指标全部达标（编译 100%≥90%、回测 100%≥80%、Sharpe>0 58%≥50%）。CI 快速子集无 benchmark tag。发现并修复 3 个工具链 Bug：①`ctx.broker.close_all()` 未映射→新增 `CloseAll` builtin；②`ctx.positions()` 未映射→`PositionsTotal`；③市价单 `req.Price=0` 未填充当前市价→`SimBroker.OrderSend` 修复。go build + go test + file-lines 全绿。
- 2026-08-09 **ARCH-2 双风控引擎修复**：移除 `submitToBroker` 中的 `risksvc.PreCheck` 调用（含 `RequiredMargin` broker RPC + 4 项检查），将等效规则注册到 Gate（`MaxPositionCount`/`MaxLotSize`/`MarginPreCheck`）。现在 `PlaceOrder` 风控路径 = `preTradeChecks`→`evaluatePlaceGate`（单一 chokepoint）→`submitToBroker`（纯执行），符合 D6-A。3 个测试更新。go build + go test + file-lines 全绿。
- 2026-08-09 **LAUNCH-2 marketplace 资金链路集成测试**：15 个 `//go:build integration` 测试覆盖 purchase/settle/refund/subscribe 全路径。发现并修复 2 个退款生产 Bug：①`refund.go` SELECT 不存在的列 `idempotency_key`（应为 `idem_key`）；②退款用 publisher ID 查购买交易（应用 subscription 的 `idempotency_key`），导致退款功能完全不可用。全 15 测试 PASS。
- 2026-08-09 **Git 提交推送**：51 个未提交文件按逻辑分 6 个 commit 推送到 `audit-pipeline5-collab` 分支。工作区干净。下一步：快速清理批次（DOC-4/5 注释漂移 + CQ-3 json.Marshal 核验 + CQ-4 time.Ticker 核验）。
- 2026-08-09 **DOC-4/5 + CQ-3/4 快速清理完成**：DOC-4（runner.go OnBar 注释）、DOC-5（ExecuteLive 注释）各一行修复。CQ-3（json.Marshal 48 处全量核验 — 零违规，全豁免）、CQ-4（time.Ticker 21 处全量核验 — 零违规，全合法）。提交推送 `9c902cce`。
- 2026-08-09 **DOC-6/7 文档漂移修复完成**：DOC-6（spec 21 顶部加弃用标注，§2–§8 标为历史参考，关联 ADR 改为 ADR-0023）。DOC-7（ADR-0028 §7 端到端测试从 P0 缺口改为 ✅done，附 BT-6 修复说明）。
- 2026-08-09 **Registry 全量对账完成 — 零 ❓待核**：剩余 8 个 ❓待核项全部核验：CQ-1（死代码存量🟦）、CQ-2（前端死代码存量🟦）、CQ-5（eslint-disable 非硬违例🟦）、CQ-7（mt4/mt5 自动生成豁免✅）、CQ-8（check-file-lines 0🔴✅）、MIG-2（DDL 仍 AVAILABLE🟦）、DOC-1（=DOC-7✅）、DOC-2（全量对账完成✅）。合并到 main 推送。**Registry 62 项全核验，零 ❓待核，零上线阻塞。**
- 2026-08-09 **MIG-2 + CQ-6 修复**：MIG-2（migration 262：deposit address schema 修正，DEFAULT AVAILABLE→ASSIGNED + CHECK 约束 ASSIGNED/RETIRED）。CQ-6（15 个 i18n en 文件已全英文，删除 stale TODO 注释）。MODE_SIGNAL/builtinOrderType 内存记录的 bug 已在之前修复（constants.go:80-81 + vm_builtin_trade.go:222-231 确认）。
- 2026-08-09 **AGT-1/FEAT-2 偷看未来检测重写完成**：IR 级 `DetectLookahead(ir)` 替代旧 DSL regex scanner。① `interp/lookahead.go` — 遍历 IR OnBar/OnInit/Funcs，检测 series subscript（Close/Open/High/Low/Volume/Time）+ indicator call（iMA/iRSI 等 40+）的 shift 参数；负 shift = Fatal，非恒定可能负 = Warning；`evalConstInt` 编译期求值，`couldBeNegative` 检测 ExprVar/ExprUnary-/ExprBinary-；dedup by (func,shiftExpr)。② `AnalyzeCoverage` 末尾调 DetectLookahead → `CoverageResult.LookaheadViolations`。③ `DetectLookaheadFromSource` 轻量入口。④ gate pipeline `evalLookAhead` 先查 IR violations → fail，fallback DSL。⑤ `BuildPipelineInputFromRepo` 从 source recompile → 填 violations。⑥ CheckCode + SubmitStrategy 响应加 lookahead blind spots。12 unit + 2 gate pipeline tests 全绿。Registry AGT-1 🟦→✅。
- 2026-08-09 **FEAT-4 实盘战绩不可篡改完成**：trade_records 哈希链实现。① migration 263：`seq`（GENERATED ALWAYS AS IDENTITY）+ `prev_hash` + `entry_hash` + trigger 阻止 hash 字段被 UPDATE。② `trade_record_repository.go`：`Create`/`BatchCreate` 改用 `insertWithHashChain`（advisory lock → 读链尾 prev_hash → INSERT → 计算 entry_hash → UPDATE），复用 `wallet_repo.ledgerChainInsert` 模式。③ `computeTradeEntryHash` = SHA256(prev_hash‖seq‖account_id‖ticket‖symbol‖volume‖open_price‖close_price‖profit‖open_time_ms‖close_time_ms)。④ `VerifyChain` 方法遍历链验证 prev_hash 链接 + entry_hash 重算，返回 `[]ChainBreak`。⑤ `model.TradeRecord` 加 Seq/PrevHash/EntryHash + `ChainBreak` 结构。⑥ 5 unit tests 全绿（确定性/不同输入/空 prev_hash/bytesEqual/ChainBreak）。go build + check-file-lines 0 errors。Registry FEAT-4 🟦→✅。
- 2026-08-09 **FEAT-4 回测 vs 实盘对比完成**：backtest vs live divergence comparison 实现。① proto `live_backtest_divergence.proto`：`GetDivergenceReport`（unary）+ `WatchDivergenceReport`（SSE stream），`DivergenceReport` 含 backtest/live metrics + divergence scores + status enum（CONSISTENT/MINOR/MAJOR/INSUFFICIENT_DATA）。② `divergence_handler.go`：`computeReport` 取最新 SUCCEEDED/DEGRADED backtest run（`GetLatestCompletedByStrategyID`）+ 通过 `trade_records.schedule_id → strategy_schedules.template_id` 关联 live trade records（`GetByStrategyID`），计算 DivergenceMetrics（trade_count/wins/losses/net_pnl/win_rate/sharpe_ratio/avg_trade_pnl/period）。③ divergence 评分：pnl_divergence_pct / trade_count_divergence_pct / win_rate_divergence_pct / sharpe_divergence，阈值 <10% CONSISTENT / <30% MINOR / ≥30% MAJOR。④ `WatchDivergenceReport` push-first：pgListen `trade_record_sync` + 60s fallback ticker，复用 `WatchSchedules` pattern。⑤ migration 264：`trade_records` INSERT → `pg_notify('trade_record_sync')` 触发 SSE push。⑥ `handlers_strategy_runtime.go` 注册 `DivergenceServer`。⑦ 9 unit tests 全绿（backtest/live metrics + empty + sharpe + pctDivergence + countDivergence + assessDivergence）。go build + check-file-lines 0 errors。REUSE: `WatchSchedules` pgListen pattern + `BacktestRunTrade` struct。NEW: `live_backtest_divergence.proto` + `divergence_handler.go` + `GetLatestCompletedByStrategyID` + `GetByStrategyID`。
- 2026-08-09 **FEAT-4 walk-forward 验证完成**：walk-forward validation 实现。① proto `walk_forward.proto`：`GetWalkForwardReport`（unary）+ `WatchWalkForwardReport`（SSE stream），`WalkForwardReport` 含 IS/OOS metrics + degradation ratio + fold details + CPCV median Sharpe + status enum（PASS/FAIL/INSUFFICIENT_DATA）。② `walk_forward_handler.go`：`computeReport` 取最新 backtest run → `EquityCurveToDailyReturns` → `ai.WalkForward`（5-fold purged walk-forward，检查 overfitting Sharpe diff > 1.0 / maxDD > 30% / tradeCount < 30）+ `ai.CPCV`（6-group combinatorial purged CV，median OOS Sharpe）+ IS/OOS 70/30 split trades metrics（trade_count/net_pnl/win_rate/sharpe/maxDD/avg_trade_pnl/period）+ `DegradationRatio`（OOS/IS Sharpe）。③ `WatchWalkForwardReport` push-first：pgListen `backtest_status` + 60s fallback ticker，复用 `WatchSchedules` pattern。④ `handlers_strategy_runtime.go` 注册 `WalkForwardServer`。⑤ 9 unit tests 全绿（sharpe/maxDD/empty/single/basic/periodBounds/noReturns/splitBoundary）。go build + check-file-lines 0 errors。REUSE: `ai.WalkForward` + `ai.CPCV` + `ai.EquityCurveToDailyReturns` + `WatchSchedules` pgListen pattern + `BacktestRunTrade` struct。NEW: `walk_forward.proto` + `walk_forward_handler.go`。FEAT-4 全部子项完成。
- 2026-08-09 **LIVE-2 策略订单幂等修复完成**：`submitOrder` 之前不设 `ClientID` → `IdempotencyKey` 返回随机 UUID → 幂等门跳过。修复：① `strategyOrderClientID(runID, barOpenTime, signalType)` 生成确定性 ClientID = `strat-{runID}-{barOpenTime}-{signalType}`。② `dispatchLiveSignal` 传 `bar.OpenTime` → `dispatchMarketOrder`/`dispatchPendingOrder` → `submitOrder`（新增 `barOpenTime int64` 参数）。③ `PlaceOrder` 的 `idem.CheckAndSet` 现在能正确去重重复信号（bar replay / VM retry / network glitch）。5 unit tests 全绿（deterministic/differentRunID/differentBarTime/differentSignalType/format）。go build + check-file-lines 0 errors。REUSE: `mthub.IdempotencyGuard.CheckAndSet` + `IdempotencyKey`。NEW: `strategyOrderClientID` helper。Registry LIVE-2 🟦→✅。
- 2026-08-10 **FEAT-2 统计类提示 + 根治报告 + Admin健康中心完成**：ADR-0028 §4.2 + §5.2-5.3 剩余 P2 项实现。① `strategy/backtest/statistical_hints.go` — 4 个统计类检查（`CheckZeroEquityVariance`/`CheckMonotonicPositionGrowth`/`CheckAllSameDirectionSameVolume`/`CheckAbnormalTradeFrequency`），severity="提示" category="statistical"，不阻断 IsReliable；`CheckStatisticalHints` 聚合入口。集成到 `buildBacktestResponse`（`backtest_worker_vm.go:342`）。14 unit tests 全绿。② `platform_health.proto` — `PlatformHealthService`：`GetRootCauseReport`（unary，按 signature 聚类 + 影响面 + 复发率）+ `WatchHealthAlerts`（SSE stream，push-first 监听 `backtest_status` NOTIFY）。③ `platform_health_handler.go` — `GetRootCauseReport` 查 `backtest_failure_signatures` 表聚合 + 前后周期对比 recurrence rate；`WatchHealthAlerts` pgListen + `parseBacktestNotification` 提取 DEGRADED/FAILED 告警。④ migration 264 `backtest_failure_signatures` 表（run_id/strategy_id/signature/severity/category/description + 4 索引）。⑤ `backtest_persistence.go` DEGRADED 时 `persistFailureSignatures` 自动提取 BlindSpots 写入签名表。⑥ `handlers_admin.go` 注册 `PlatformHealthServer`。⑦ 5 handler tests 全绿。go build + check-file-lines 0 errors。REUSE: `backtest.CheckStatisticalHints` + pgListen pattern（同 `WatchSchedules`/`WatchDivergenceReport`）。NEW: `statistical_hints.go` + `platform_health.proto` + `platform_health_handler.go` + migration 264。Registry FEAT-2 统计类提示+根治报告标完成。
- 2026-08-10 **FEAT-2 防线A 解析后校验完成**：ADR-0028 §4.1 Defense A 实现。① `interp/defense_a.go` — `ValidateDefenseA(ir)` 检查 3 项编译期规则：参数名合法性（非 MQL 关键字/类型名/内置变量名 `mqlKeywordsAndTypes`）、参数名唯一（无重名）、入口存在（OnTick/OnBar/OnTimer/start 至少一个被映射）。违反 → `DefenseAViolation{Rule, Identifier, Severity=Fatal, Message}`。② `AnalyzeCoverage` 末尾调 `ValidateDefenseA` → `CoverageResult.DefenseAViolations`。③ `VMRunner` 携带 `defenseAViolations`（`CompileMQLWithCoverage`/`CompilePythonWithCoverage` 填充，`InjectDefenseAViolations` 用于 cache 恢复）。④ `buildBacktestResponse` 注入 Defense A blind spots + `IsReliable=false`（fatal 违规降级回测结果）。⑤ `code_check_handler.go` + `strategy_import_handler.go` 实时编辑器/导入时反馈 Defense A blind spots。⑥ 11 unit tests 全绿（no_entry_point/OnTick/OnBar/start/keyword/builtinVar/duplicate/valid/multiple/format_empty/format_nonempty）。go build + check-file-lines 0 errors。REUSE: `AnalyzeCoverage` pattern + `VMRunner` inject pattern。NEW: `defense_a.go` + `DefenseAViolation` type。Registry FEAT-2 防线A标完成。剩余 FEAT-2 子项：判决型误报(不做)、用户侧恢复前端、golden trace(成熟期)。

---

## 前瞻计划（审计后建设/修复优先级）

> 7 管线 + account-mgmt 全部审完。以下按"正确性 > 安全 > 功能 > 体验"排序，结合"钱路径 > 用户路径 > 内部路径"权重。

### P0 — 立即施工（DoD 收尾，本会话）✅ 全部完成

| # | ID | 项 | 估时 | 备注 |
|---|-----|----|------|------|
| 1 | AGT-2 | agent 回测接防线B | 中 | ✅ 共享 `strategy/backtest/invariants.go` + proto 加 `is_reliable`/`invariant_blind_spots` + `buildBacktestResultProto` 调 `ValidateInvariants` + LLM 反馈 |
| 2 | RECON-1 | 漏单修复 | 中 | ✅ `SyncAccountHistory` 实现增量 upsert + Reconciliation 升 Warn/24h 窗口 |
| 3 | BRIDGE-1 | bridge.go nil panic | 小 | ✅ `bridge.go:82` 加 `result == nil` 守卫 |

### P1 — 高优修复（正确性/安全）✅ 全部完成+部署（2026-08-07）

| # | ID | 项 | 估时 | 备注 |
|---|-----|----|------|------|
| 4 | CREDIT-1 | credit hold 持久化 | 中 | ✅ migration 261 + session_id + GetStaleHolds + RestoreHolds 启动恢复 |
| 5 | AUD-U1/U3 | 禁用用户不撤会话 | 小 | ✅ DisableUser 调 IncrementTokenVersion + session revoke |
| 6 | AUD-A6/A7/A8 | auth-session 三件 | 小-中 | ✅ token_version 校验 + JWT 签发错误处理 + reset token 原子化 |
| 7 | AUD-W4-1 | AI provider SSRF | 小 | ✅ base_url 私有 IP 过滤 |
| 8 | AUD-MK2 | retryFailedReversals 双扣 | 小 | ✅ idem key 固定不随重试变 |
| 9 | AUD-C1 | Deposit 记录事务外 | 小 | ✅ 移入主事务 |
| 10 | AUD-MT2 | PlaceOrder nil logger panic | 小 | ✅ nil 检查或注入 |

### P2 — 中优（功能补全/安全加固）✅ 全部核验完成（2026-08-07）

| # | ID | 项 | 估时 | 备注 |
|---|-----|----|------|------|
| 11 | AGT-1 | 偷看未来检测重写 | 中-大 | ✅done — IR 级 `DetectLookahead` 替代旧 DSL scanner。`interp/lookahead.go` 检测 series subscript + indicator shift（40+ 函数）；集成到 gate pipeline / CheckCode / SubmitStrategy；12 unit + 2 gate pipeline tests 全绿 |
| 12 | AUD-W3-2 | NATS 无认证 | 中 | ✅done 误报 — `client.go` 支持 CredsFile 认证，是部署配置项 |
| 13 | AUD-W1-4 | StartStrategy 回退用户 accountID | 小 | ✅done 误报 — `resolveModeAndAccount` live 模式服务端覆盖用户 accountID |
| 14 | AUD-N1 | SendNotification IDOR | 小 | ✅done 误报 — admin 校验已存在 |
| 15 | AUD-SB1 | subscribeFree 静默吞 plan 错 | 小 | ✅done 误报 — 错误正确传播 |
| 16 | AUD-WS1/WS2 | wallet settlement 两件 | 中 | ✅done 误报 — freezeOp 有 undo + saveBundleAndLinkWithdrawal 原子 |
| — | AUD-W3-1 | migration 未包事务 | 小 | ✅done — `docker-entrypoint.sh` 用 BEGIN/COMMIT 包事务 |
| — | AUD-SW1 | BroadcastingBundle 23h 硬编码 | 小 | ✅done 误报 — 23h 是 fallback，可配置 |
| — | AUD-U2 | UpdateUser status 字符串校验 | 小 | ✅done 误报 — validStatus 已校验 |

### P3 — 低优（特性/简化/文档）— 全部核验完成（2026-08-07）

| # | ID | 项 | 备注 |
|---|-----|----|------|
| 17 | BRIDGE-2 | bridge backtest 接防线B | ✅done — `validateBacktest` 闭包增加 `ValidateInvariants` 全 5 项检查 |
| 18 | CREDIT-2 | CheckBalance fail-open | ✅done — fail-open→fail-closed，DB 错误时返回 error 阻止访问 |
| 19 | BT-1/2/3 | SimBroker swap/equity/slippage | ✅done — swap 记账 refactor + equity 定义统一 + spread 加价 |
| 20 | LIVE-2 | 策略订单幂等 | 🟦open — 已知特性 |
| 21 | AUD-A1-4 | connect/strategy 包拆分 | ✅done — 无超标文件 |
| 22 | AUD-W1-3 | session token INFO 级日志 | ✅done 误报 — 只 log userID/error，不 log token |
| 23 | AUD-WS3 | Admin AdjustBalance txType 错分 | ✅done 误报 — txType="adjustment" 正确 |
| 24 | AUD-W2-1 | access_token URL query 传递 | 🟦open 低 — SSE 标准方案，TTL=15min 缓解 |
| 25 | AUD-F3-1 | OMS 事件广播架构一致性 | ✅done 误报 — OrderEventBroker 按 userID 隔离 |
| 26 | ARCH-5 | kline_data 表删除 | ✅done — migration 166 已 DROP |
| 27 | ARCH-6 | RecognizerRegistry 死注册 | ✅done 误报 — 代码中不存在 |
| 28 | ARCH-3 | 双模板表修复 | ✅done — ScheduleEngine 改用 StrategySvc.GetTemplate |
| 29 | ARCH-2 | 双风控引擎 | ✅done — 移除 submitToBroker 中的 risksvc.PreCheck，等效规则注册到 Gate（D6-A 单一 chokepoint）|
| 30 | FEAT-1 | 购买→实盘链路（任务2-5） | ✅done — 事件型会话化+授权闸+配额闸+每bar复验+集成测试+ADR-0029+已部署 |
| 31 | FEAT-4 | 实盘战绩不可篡改（append-only + 哈希链） | ✅done — migration 263 + insertWithHashChain + VerifyChain + computeTradeEntryHash + 5 tests |

### 执行原则

- **每项施工前**：读 registry 条目 + `git log/blame` + `cap.sh` 查复用
- **每项完工后**：registry 🟦→✅ + 对抗证明 + 红队自审 + handover-plan 变更日志
- **关键/架构决策**：标 `⚠️待Claude复审`
- **P0 三项为本会话目标**，P1+ 留给后续会话/外部 agent
