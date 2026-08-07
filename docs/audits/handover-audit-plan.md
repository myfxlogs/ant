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
| 4 | Agent循环 | frontend, api-gateway, agent-engine, mql-compiler, backtest-engine | 用户输入 → generate/revise → compile → backtest → 迭代 | 🟡半审（主干+主发现 AGT-2/AGT-1，细节待补）|
| 7 | 策略市场 | frontend, api-gateway, strategy-marketplace, backtest-engine, agent-engine | 发布/购买/冻结结算（购买→实盘断链已定位）| ✅ 已审 |
| — | account-mgmt | connect/{gateway,user} | MT 账户 CRUD/经纪商/用户（memory 称已完工最优，验真）| ⬜ |

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
