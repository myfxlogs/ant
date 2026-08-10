# 施工方交接提示词（2026-08-10）

> **依据**：CLAUDE.md「Deployment (强制 — 禁止手动)」+ `tech-debt-registry.md` EXEC-PARAMS 验收行（✅ 权威 done，2026-08-10 审计方实测）+ `handover-audit-plan.md` 验收条目。
> **强制遵守** `docs/audits/builder-sop.md` 三铁律 + 验收 checklist（§200）：① one task = one scope（不扩大）；② 对抗证明（删关键一行→测试必红）；③ **红队自审**（完工后对着 §200 checklist + 验收 5 维过一遍，带着债提交=失败）；④ 不自行宣告 ✅，等审计方实测；⑤ 完工回填三层；⑥ REUSE preflight（cap.sh）。
> **节奏**：一次一个 task。提示词 1/2 已完成（部署 `567b87e2` + 补测 `e45ce7d4`，审计方已核对）；当前进行**提示词 3（FILL-SIM）**。完工回填 + 自审后再交付，不并行多任务。
> **📌 FILL-SIM 提示词独立成文（2026-08-10）**：提示词 3 已移出本文件，独立为 **`docs/audits/builder-handoff-fill-sim-2026-08-10.md`**（自包含，打开即开工，含用户评审 3 点闭环）。下方提示词 3 原文保留作历史，**施工以独立文件为准**。
> **当前状态**：EXEC-PARAMS（a1c88f33）已由审计方实测验收通过并部署上线。**FILL-SIM spec 定稿**（`spec-fill-rule-limit-simulation-mode.md`：Windsurf 复审 7 点修订 + **Claude Code 审计方 2026-08-10 复核 §8**：8 项根因全部实测属实；3 项修订——§1.3/§3.3 事实校正、§2.6 VM pending 前置升级、§2.4 测试补漏）。**§2.6 是必备前置**，先于 §2.1/§2.2 施工。工作区未提交：audit spec 修订（docs）+ handover/registry（docs），代码零未提交，migrations 空。

---

## 提示词 1【🔴 部署 · 唯一方式，禁止手动替代】EXEC-PARAMS 部署 + 冒烟验证

```
依据 CLAUDE.md「Deployment (强制 — 禁止手动)」段 + registry EXEC-PARAMS 验收行（先完整读）。

背景：EXEC-PARAMS（signal_timing/fill_rule/simulation_mode 端到端接线 + StrictMode 引擎 bug 修复，
commit a1c88f33）已审计方实测验收通过，部署解锁。本次把该提交上线：后端 docker compose + 前端 docker cp。

范围（one task = one scope）：仅部署 + 冒烟验证。不改任何代码（含顺手修复/格式化）、不补测试
（另有提示词 2）、不动 docker-compose.yml/迁移/环境变量/其他服务。

步骤：
1. 先读 CLAUDE.md「Deployment」段 + builder-sop.md + registry EXEC-PARAMS 验收行。
2. 部署前检查（必做，任一不过停下确认）：
   - git status：确认零未提交代码文件；先 commit 审计方 2 行验收 doc（handover + registry，
     message: docs: EXEC-PARAMS 审计方验收记录）——使 build 上下文与 HEAD 一致。
   - git status backend/migrations/ 必须为空（未提交 .up.sql 会随 Docker build 自动执行；有 WIP 先移走）。
   - git log -1 确认 HEAD 含 a1c88f33（或其后的文档 commit）。
   - 记录当前运行中 backend 镜像 tag（rtk proxy docker compose images backend 或 docker inspect），
     写入回滚预案。
3. 后端部署（唯一方式）：rtk proxy docker compose build backend && rtk proxy docker compose up -d backend。
   确认容器 Healthy（rtk proxy docker compose ps）。
4. 前端部署（唯一方式）：cd frontend && npm run build（产出 dist/）→
   rtk proxy docker cp frontend/dist/. alphaforge-frontend:/usr/share/nginx/html/ →
   rtk proxy docker exec alphaforge-frontend nginx -s reload。
   ❌ 禁止宿主机 go build → docker cp；❌ 禁止在运行中容器内 go build / apk add build-base。
5. 冒烟验证（部署后，逐项记录结果）：
   - curl 容器 /healthz 与 /readyz 均健康。
   - 【对抗证明·validate 闸上线】API 级：startBacktestRun 带 executionConfig{simulation_mode:"DATASET"}
     → 预期 InvalidArgument（422，报 not yet implemented）；executionConfig{fill_rule:"limit"} 同样拒绝。
     若旧容器还在跑，这两个请求会被静默接受——拒绝 = 新代码真实上线。
   - 边界（防闸太紧误杀）：带合法请求 executionConfig{signal_timing:"next_bar_open", fill_rule:"bar_close",
     simulation_mode:"KLINE_RANGE"} → 不得误拒（验证不走到 validate 的 market-data 检查也没关系，
     只要不是 InvalidArgument 即可）。
   - 后端日志：docker logs backend 无 panic；启动日志正常。
   - 前端：浏览器硬刷新（Ctrl+Shift+R）打开策略工作区 → 回测参数 modal 出现 Execution Assumptions
     三选择器（Simulation Mode / Signal Timing / Fill Rule，DATASET 与 limit 选项置灰）。
     若无法连接 MT 账户做全流程，降级为产物验证：grep -c "next_bar_open" frontend/dist/assets/*.js > 0
     （旧 dist 无此产物），并在回填中注明降级。
   - 若条件具备（有连接账户+品种数据）：跑一次回测，ExecutionAssumptions 面板显示用户选择的
     signal_timing（非硬编码 next_bar_open）——诚实性验证。
6. 回滚预案（冒烟任一失败即回滚，不现场修代码）：
   - 后端：docker compose up -d 步骤 2 记录的旧镜像 tag。
   - 前端：docker cp 回旧 dist（若无备份，checkout 旧 commit 重新 build 再 cp）。
   - 回滚后报告审计方，不自行诊断修复。

对抗证明（部署场景，缺 = 任务判失败）：
- validate 闸生效：部署后 API 拒绝 DATASET/limit（旧行为是静默接受）——这是部署任务的"关键行"。
- 响应诚实性：回测 ExecutionAssumptions 显示用户选择值，非硬编码 next_bar_open。
- 前端新 UI 上线：dist grep "next_bar_open" 命中（旧 dist 无此产物）。

红队自审（部署任务级 edge cases，任一不过回去处理，不带债交付）：
- 部署前工作区零未提交代码文件（只允许刚 commit 的 2 行 doc）；多一个未提交文件=停下确认，不静默 build。
- build 失败 / pull 超时：先 rtk proxy docker compose config 验证配置；重试≤3 次，仍失败进回滚预案。
- 容器起不来（NATS/PG 依赖问题）：看 rtk proxy docker compose ps 依赖服务状态；不擅自重启/改依赖
  容器配置（超范围），报告等指示。
- 前端 dist 与 HEAD 匹配：build 前确认 HEAD 含 a1c88f33，避免用旧 dist 部署。
- 浏览器缓存：用户看到旧 UI 不是部署失败——提示硬刷新；不为此改 nginx 配置（超范围）。
- 克制：整个部署过程零代码改动；发现任何新问题只记录报告，不顺手修；不动 compose 文件/迁移/环境变量。
- 可演进性：回滚预案可复用（记录镜像 tag 成习惯），不写死单次操作。

Gate：部署 + 冒烟命令见步骤 5；无生产代码改动（doc-only commit），不需 go build/go test。

完工回填（不做 = 任务判失败）：
1. tech-debt-registry.md EXEC-PARAMS 条目追加部署行：部署时间 / 镜像 tag / 冒烟结果
   （validate 闸实测 + ExecutionAssumptions 实测或降级说明）。不改 ✅ 状态（已权威）。
2. handover-audit-plan.md 变更日志加一行（EXEC-PARAMS 部署完成 + 冒烟结果）。
3. commit（doc-only；message: docs: EXEC-PARAMS deployed + smoke results）。
4. 不自宣告完成——状态标 ⚠️待 Claude 复审，等审计方核对。
```

---

## 提示词 2【🟢 低优可选 · 非阻断】EXEC-PARAMS 补 2 个非阻断单测

```
依据 tech-debt-registry.md EXEC-PARAMS 验收行「非阻断观察 a」：worker 层回退派生
（strictMode→signalTiming）与 validate 拒绝逻辑无单测。提示词 1 部署完成后做。

范围（one task = one scope）：仅测试——backend/internal/connect/strategy/ 下新建/追加
backtest_worker_vm_test.go + strategy_backtest_validate_test.go。不动生产代码。

步骤：
1. REUSE preflight（动工前）：bash scripts/cap.sh validate；cap.sh backtestParams；
   查同包已有测试的 server 构造模式（如 backtest_execution_test.go 的 StrategyExecutionServer 零字段
   构造 + mock BarSource），复用不重造。PR 描述给 REUSE:/NEW:。
2. Test 1 TestBuildBacktestConfig_SignalTimingFallback（backtest_worker_vm_test.go）：
   - params{strictMode:false, signalTiming:""} + 最小 repository.BacktestRun → cfg.SignalTiming=="same_bar_close"。
   - params{strictMode:true, signalTiming:""} → cfg.SignalTiming=="next_bar_open"。
   - params{strictMode:true, signalTiming:"same_bar_close"} → 显式值优先，=="same_bar_close"（不回归）。
   - 顺带断言 cfg.FillRule=="bar_close" / cfg.SimulationMode=="KLINE_RANGE" 空值默认。
3. Test 2 TestValidateBacktestRequest_RejectsUnimplemented（strategy_backtest_validate_test.go）：
   - executionConfig{fill_rule:"limit"} → connect.CodeInvalidArgument。
   - executionConfig{simulation_mode:"DATASET"} → CodeInvalidArgument。
   - executionConfig{fill_rule:"bar_close", simulation_mode:"KLINE_RANGE"} → 不误拒。
   - req.Msg.GetExecutionConfig()==nil → 不 panic，正常走后续（marketDataRepo nil 时提前 return nil
     的短路点可隔离本测试，无需 mock 数据）。
   - 大小写精确性：fill_rule:"LIMIT" 不误拒（按当前实现精确匹配语义，不做模糊宽容）。

对抗证明（缺 = 任务判失败）：
- 删 validate 中 limit 拒绝分支 → Test 2 必红。
- 删 buildBacktestConfig 中 signalTiming 派生行 → Test 1 必红。

红队自审（任务级 edge cases）：
- nil executionConfig / 空字符串值合法不拒绝 / 大小写边界（"LIMIT"≠"limit"）/ 显式值优先于派生 /
  测试数据确定性（固定 epoch，禁 time.Now）/ 不 mock 过度（真实 server struct 零字段即可）/
  克制（不动生产代码，不顺手重构）。
- 测试质量：覆盖"防闸太紧"（合法值不误拒）与"防闸太松"（非法值必拒）双向。

Gate：go build ./... + go test ./internal/connect/strategy/... + go run ./tools/check-file-lines --strict。

完工回填（不做 = 任务判失败）：
1. tech-debt-registry.md EXEC-PARAMS 条目追加补测行（测试名 + 对抗证明结果）。
2. handover-audit-plan.md 变更日志加一行。
3. commit（test + doc 一并）。
4. 不自宣告完成——标 ⚠️待 Claude 复审，等审计方实测。
```

---

## 提示词 3【🔴 FILL-SIM】fill_rule=limit + simulation_mode=OHLC_PATH 落地（含 §2.6 VM pending 前置）

```
依据 `docs/audits/spec-fill-rule-limit-simulation-mode.md`（先完整读：§1 问题 + §2 设计 + §2.6 前置 +
§3 最优性 + §4 清单 + §6 风险 + §7/§8 修订汇总）。审计方已复核定稿，本提示词只重申强制点，一切以 spec 为准。

背景：fill_rule=limit 与 simulation_mode=DATASET 当前在 validateBacktestRequest 被 API 拒绝
（EXEC-PARAMS 的诚实性闸）。本任务实现它们：limit = market order 转 pending；DATASET wire 值改名
OHLC_PATH = bar 内 OHLC 路径 SL/TP 顺序判定。**前置（§2.6）：VM pending order 可见性是必备前置**——
审计实测 builtinOrdersTotal/OrderSelect 只查 Positions、PositionModify 只搜 positions、OrderType 对
pending 报错——不先修，market→pending 会让所有 market order EA 的持仓管理静默错乱（违反 MQL 诚实性红线）。

范围（one task = one scope）：仅 FILL-SIM 一个任务，内部按 Phase A→E 顺序执行（每 Phase 独立对抗证明 +
测试，全部完成才交付）。❌ 不扩大：不拉 M1/tick 数据、不重建 tick 持久化、不动实盘下单路径
（runner/broker.go 不改）、不顺手重构其他代码。

【REUSE preflight 必做】动工前 bash scripts/cap.sh 换词查：checkSLTP / pending / OrderModify /
ordersTotal / whitelist。已有能力直接复用（checkPendingOrders 触发逻辑、PositionModify 结构、
exec_params_validation_test.go 现有测试模式）。PR 描述逐条给 REUSE:/NEW:。

──────── Phase A【§2.6 必备前置，先做】VM pending 可见性 ────────
文件：backend/tools/mql2go/vm_builtin_trade.go + backend/strategy/backtest/broker.go
1. builtinOrdersTotal：len(Positions(0)) + len(Orders(0))（MQL4 OrdersTotal 含 market+pending）。
   ⚠️ 红队：live 路径 brokerImpl.Positions 来自 executor.OpenedOrders——先核验 adapter
   （mdgateway/adapter/mt5/order_history.go 等）的 OpenedOrders 是否已含 pending；若含则 Positions+Orders
   双计。实测确认后二选一：a) 确认 disjoint（多数 MT5 语义 OpenedOrders=positions 仅）；b) 若 MT4 侧含，
   在测试中锁定 backtest（SimBroker positions/pending 天然 disjoint，broker.go:135-141 实证）+ live 侧
   记录已知边界不双计（写回 spec 或注释，不静默）。
2. builtinOrderSelect MODE_TRADES：positions 索引段之后追加 pending 段（SELECT_BY_POS 与 SELECT_BY_TICKET
   均需覆盖；MQL4 语义 order 池按位置索引含 pending）。
3. builtinOrderType：pending 订单按 OrderType 返回 OP_BUYLIMIT=2/OP_SELLLIMIT=3/OP_BUYSTOP=4/OP_SELLSTOP=5，
   非按 Side（sideToOrderType 只在 currentPos 为 position 时用）。
4. SimBroker.PositionModify：追加 pending 扫描段（SL/TP 可改）；builtinOrderModify 接住 args[1]（price，
   MQL4 OrderModify(ticket,price,sl,tp)），改价语义=改 pending.Price（回测侧）。
5. 缓存一致性：pending 快照与 cachedPositions 同生命周期（事件内多次调用一致）。
测试（mql2go 包 + backtest 包）：EA 下 OP_BUYLIMIT → OrdersTotal()==1；OrderSelect(0,MODE_TRADES) 成功
且 OrderType()==2；OrderModify 对 pending 成功（SL 变更后 OrderStopLoss() 反映）；pending 成交后
OrdersTotal() 回落；OrderDelete 对 pending 不回归（已支持）。
对抗证明（缺 = Phase A 判失败）：删 builtinOrdersTotal 的 +len(Orders) 行 → OrdersTotal 测试红；
删 PositionModify pending 段 → OrderModify 测试红。

──────── Phase B【§2.1 + §6.4】fill_rule=limit ────────
文件：backend/strategy/backtest/broker.go（+ engine.go fill 分支）
1. OrderSend：config.FillRule=="limit" && req.Type==OrderMarket → 转 OrderLimit。**顺序强制**：
   price=0→currentPrice 解析必须发生在转换后仍生效（spec §2.1 注：先解析价再转换，防挂单 Price=0
   永不触发）。保留 SL/TP/comment/magic 原样。
2. §6.4 决策①（默认执行，审计已认可）：commission 从 OrderSend 移至 checkPendingOrders fill 分支
   （成交时刻扣）+ 同点 margin 复检（不足 → 撤单记 RetNoMoney 并 log，不 append 进 positions）。
   ⚠️ 红队：这是行为变化——原生 pending（KLINE_RANGE 下）从不成交的单将不再扣 commission（修正
   而非回归）；已成交单在成交时刻扣（总额不变，时点变化）。现有测试若断言"下单即扣"须同步（KLINE_RANGE
   回归测试锁定新语义）。若实测发现改造成本不可控，可退回②（保持现状+文档声明），但必须在回填中说明
   理由，不能悄悄选。
   📌 范围已确认（2026-08-10 评审）：适用于全部 pending（含原生 OP_BUYLIMIT），非仅 limit 转换单——
   实盘语义对齐（真实 MT4 挂单不成仓不收 commission）。依据见 spec §6.4「范围确认」块；已核实无现有测试
   锁定"下单即扣"。
3. 不改 checkPendingOrders 触发逻辑本身（范围检查保持，§3.4 局限 1a 已接受）。
测试（§4.3 前 4 例）：TestFillRule_Limit_MarketOrderBecomesPending（same_bar_close 模式）/
TestFillRule_Limit_PendingFillsOnBarTouch / TestFillRule_Limit_NextBarOpen_FillsSameBarAtOpen（退化行为
锁定）/ TestFillRule_Limit_ExplicitPrice_WaitsForTouch。
对抗证明：删转换行 → pending 空 → TestFillRule_Limit_MarketOrderBecomesPending 红；删 fill 分支
commission 行 → commission 断言红。

──────── Phase C【§2.2】OHLC_PATH ────────
文件：backend/strategy/backtest/engine.go（或新文件 sltp_path.go——engine.go 已 495 行，checkSLTPPath
单独成文件优先，避免继续膨胀）+ 主循环 3 行切换（SimulationMode=="OHLC_PATH" 时替代 checkSLTP）。
1. 路径构建：阳线 O→H→L→C、阴线 O→L→H→C（Close==Open 归阳线，注释说明）。
2. 3 单调段区间包含检查（buy/sell 对称）；SL/TP 落不同段 → 先出现的段先触发；落同一段 → 距段起点
   近者先触发；成交价=触发价。
3. gap-at-open 保留：Open 已穿越 SL/TP → 成交价=Open（与现有 checkBuySLTP/checkSellSLTP 行为一致，
   spec §2.2 修订 4）。
4. checkPendingOrders 保持范围检查（§3.4 局限 1a，不扩范围）。
测试（§4.3 后 7 例）：Buy_TPBeforeSL_BullishBar / Buy_SLBeforeTP_BearishBar / Sell_TPBeforeSL_BearishBar /
SameSegment_NearerFirst / GapOpen_FillsAtOpen / NoHit / KlineRange_BehaviorUnchanged（回归：默认模式
逐位不变）。
对抗证明：删路径顺序判定改用 SL 优先 → TestOHLCPath_Buy_TPBeforeSL_BullishBar 红。

──────── Phase D【§2.3 + §2.4】wire 改名 + 白名单 ────────
文件：strategy_backtest_validate.go + exec_params_validation_test.go + proto/types 注释 + ExecutionAssumptions 注释
1. 白名单校验（spec §2.4）：fill_rule ∈ {"",bar_close,market,limit}、simulation_mode ∈
   {"",KLINE_RANGE,OHLC_PATH}、signal_timing ∈ {"",next_bar_open,same_bar_close}（顺带补齐）；
   非法值 → invalid_argument，错误信息列合法值；"DATASET" 报错含"已更名 OHLC_PATH"提示。
2. 测试翻转（原稿 + §8 审计补漏）：:118 RejectFillRuleLimit → 接受断言；:144 RejectSimulationModeDataset
   → 拒绝+改名提示断言；**:383 CaseSensitiveFillRule → 翻转（"LIMIT" 大写现在必须被拒——白名单顺带修正
   大小写 bug，测试同步改）；新增未知值拒绝测试。
3. proto/types/ExecutionAssumptions 注释 DATASET→OHLC_PATH（spec §2.3，wire 值直接改名，无旧快照零成本）。
对抗证明：删白名单 → "FOO" 静默走默认 → 未知值拒绝测试红。

──────── Phase E【§4.2】前端 ────────
文件：ExecutionAssumptionsSelectors.tsx（limit/DATASET 选项 enabled；DATASET value 改 "OHLC_PATH"）+
BacktestParamsModal.tsx / useBacktestRunner.ts / strategyRuntime.ts / useStrategyWorkspaceState.ts
（union 'DATASET'→'OHLC_PATH'）+ i18n（OHLC_PATH 显示名 "OHLC Path"/"K线路径模拟"；limit tooltip 说明
§2.5 退化行为——next_bar_open 下 Price=0 的 market order 转 limit 同 bar open 即成交；same_bar_close 下
可能永不成交）。
对抗证明：改回 'DATASET' union → tsc --noEmit 红。

【Gate（全部 Phase 完成才跑）】go build ./... + go test ./strategy/... + go test ./internal/connect/strategy/...
+ go test ./tools/mql2go/... + go run ./tools/check-file-lines --strict + cd frontend && npm run build
+ npx tsc --noEmit + npx vitest run。全部绿才回填。

【红队自审（任务级 edge cases，任一不过回去处理，不带债交付）】
- Phase A：live OpenedOrders 是否含 pending（双计风险，先核验再写死）；OrderSelect 索引越界不 panic；
  OrderType 对 pending 四型映射不串；缓存一致性（事件内多调用）；KLINE_RANGE 原生 pending 路径回归。
- Phase B：price=0→currentPrice 顺序（转换后仍生效）；next_bar_open 退化行为被测试锁定（同 bar open
  成交）非静默；SL/TP 随转换保留；commission/margin 时点变化对 KLINE_RANGE 现有测试的影响（先跑回归
  确认破坏面）。
- Phase C：doji（Close==Open 归阳线）；SL/TP 恰在段边界（区间包含用 <=/>= 与现有 checkSLTP 一致）；
  开仓于 bar 内（pending fill）的仓位同 bar SL/TP——局限 1a 接受，不扩范围；成交价恒为 SL/TP 价/Open
  （无 spread 混入，fill_rule=limit 非 market）。
- Phase D：空串合法（默认语义）；大小写敏感（"LIMIT"/"DATASET" 被拒）；错误信息可执行（列合法值）。
- Phase E：TS union 全链改齐（5 文件，缺一 tsc 红）；i18n 5 语言 key 同步；退化行为 tooltip 必须写
  （诚实性，spec §2.5）。
- 克制：不改实盘路径；不扩 pending 路径精度（§3.4 局限 1a/2 已接受）；engine.go 超行数时优先新文件。
- 测试数据确定性：固定 epoch，禁 time.Now()（spec 21 §10 Determinism Contract）。

完工回填（不做 = 任务判失败）：
1. tech-debt-registry.md FILL-SIM 条目（🟦open → ✅done 标日期）追加：真实根因/修复方式/对抗证明结果/
   测试结果；若实际根因与 spec 假设不同如实写明。
2. handover-audit-plan.md 变更日志加一行。
3. commit（代码 + 测试 + docs 一并，message 含 FILL-SIM）。
4. 不自宣告完成——标 ⚠️待 Claude 复审，等审计方实测（对抗证明会实测验证）。
```
