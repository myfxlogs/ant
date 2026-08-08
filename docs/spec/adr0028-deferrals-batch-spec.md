# ADR-0028 §8 暂缓项批量施工 Spec（3 part）

> **定位**：ADR-0028「源码→回测管线可靠性」实现范围已 ✅ 合格验收（2026-08-09，防线 A/B + 蜕变 + statistical + lookahead + assessRisk 闸门 + DEGRADED + 根治报告）。本 spec 收口 ADR §8 **明确定档暂缓**的 3 项——它们不是 bug，是按 ADR 哲学（§2.4 80% 方案 / §3 自我发现自动、加固需人）的有意裁剪。现批量出 spec 防遗忘。
> **关联**：`docs/adr/0028-...` §8、registry FEAT-2、`docs/spec/15-observability.md`（Part A 复用健康中心）。
> **执行**：3 part = 3 个独立任务/PR（one task = one scope），可按 phase 分批做，不必一次全做。
> **日期**：2026-08-09

---

## Part A · 判决型误报处置（false-positive handling）

> **ADR §8 原文**：恒等类实现 bug 或容差偏差可能误报；误报与真 bug 复用同一套双周根治 cadence。待实现：**产品内反馈入口 + 自动白名单（立即豁免、根治后移除）**。
> **Phase**：post-launch（有真实误报数据后再做；现阶段全自助）。

### 背景
防线 B 恒等类数学零误报，但**实现**（容差/边界）会错——例如资金守恒容差设太紧 → 合法策略被判 IsReliable=false。用户无申诉通道时，误报 = 冤枉合法策略 = 信任受损。

### 目标 / 非目标
- **目标**：① 用户一键报"这是误报"（反馈入口）；② 系统对已报误报的 blind spot signature **自动豁免**（该 signature 不再强制 IsReliable=false，改标"豁免中"）；③ 误报进 `backtest_failure_signatures` 表（migration 264 已在），随双周根治 cadence 处置——真阳性改容差/代码后移除豁免。
- **非目标**：不做客服专线；不做全自动误报判定（人 review）；不豁免致命类（偷看未来/手数0——这些不该被豁免）。

### 实现
| # | 任务 | 锚点 |
|----|------|------|
| 1 | `invariant_whitelist` 表（migration）：`(blind_spot_signature, strategy_id?, status{active,resolved}, created_at, reason)` | 新 migration |
| 2 | 评估防线 B 前查白名单：命中的 signature 不强制 IsReliable=false，改 BlindSpot.Severity="豁免中" | `backtest_worker_vm.go:313-340` |
| 3 | RPC：`ReportFalsePositive(blind_spot_signature, strategy_id, reason)`（用户侧）+ `ResolveWhitelist(signature)`（admin） | proto + handler |
| 4 | 前端：blind spot / "结果不可信"面板加"这是误报"按钮 → 调 ReportFalsePositive | backtest 结果页 |
| 5 | Admin 健康中心：误报 signature 与真 bug 同表展示，标"豁免中/已根治" | `platform_health_handler`（复用） |
| 6 | 对抗证明：构造容差误报 → 报误报 → 该 signature 再回测不再判 IsReliable=false（测试必验证）| test |

> **复用核对**：`backtest_failure_signatures` 表 @ migration 264、`platform_health_handler` 根治 cadence、`antv1.BlindSpot` 结构。NEW：whitelist 表 + 2 RPC + 前端按钮。

### 验收 + 对抗证明
- 报误报后同 signature 回测不再 IsReliable=false；admin 根治后豁免移除、重新生效。
- **对抗证明**：删白名单查询 → 误报 signature 仍判 IsReliable=false（复现误报）。

---

## Part B · 用户侧恢复路径前端（"结果不可信"后用户能做什么）

> **ADR §8/§5.4 原文**：防线判"结果不可信"后用户的恢复路径（重跑/修代码/看诊断）——**前端交互待设计**。§5.4 修复闭环：告知（零 token）→ 提示 AI 补齐 → 用户点 → AI 生成（扣 token）→ 重过防线 A/B → 版本快照 → 用户 diff 确认 → 应用。
> **Phase**：near-term（可现在做——防线输出已就绪，缺前端呈现 + AI 修复闭环）。

### 背景
防线 B 判 IsReliable=false / DEGRADED 时，用户当前只看到"不可信"标签，看不到**为什么**（哪个恒等类违反、什么诊断）、**能做什么**（重跑/改码/AI 帮修）。ADR §5.4 已设计闭环，但前端未实现。

### 目标 / 非目标
- **目标**：① "结果不可信"面板清晰呈现诊断（违反的恒等类 + BlindSpot 人话文案 + 严重度分级 致命/风险/质量）；② 按 §5.4 分级响应：致命（偷看未来/逻辑断裂/手数0）→ 结果判无效 + "AI 帮修"入口；风险（无止损/重仓/马丁/高频）→ 警告 + "我是故意的"静默；质量（过拟合/死代码）→ 建议；③ AI 修复闭环 UI：诊断（免费）→ 点"AI 修"→ diff 预览 → 应用 → 重过防线。
- **非目标**：不做自动改码（用户必须点确认 + diff review）；不做致命类的"静默"（致命不可豁免）。

### 实现
| # | 任务 | 锚点 |
|----|------|------|
| 1 | "结果不可信"诊断面板：渲染 `InvariantBlindSpots`/`DefenseAViolations`/`LookaheadViolations`，按 severity 分组（致命红/风险黄/质量灰）+ 预设人话文案（零 token，§5.4） | backtest 结果页组件 |
| 2 | "AI 帮修"按钮（仅致命/质量类）：调 agent bridge 生成修复 → 返回 diff | 复用 `UpdateStrategyCode`/agent bridge（ADR-0024）|
| 3 | diff 预览 + 应用：用户 review → 应用 → 新版本快照 → 自动触发重回测 → 重过防线 A/B | 复用版本快照 + 回测触发 |
| 4 | 风险类"我是故意的"静默开关：持久化用户确认（per strategy per signature） | user_pref 表或 strategy meta |
| 5 | 对抗证明：致命类（如 volume=0）→ 面板必须红标 + "AI 帮修"可点；风险类 → 黄标 + 静默开关；删诊断渲染 → 面板空（测试必红）| test |

> **复用核对**：`UpdateStrategyCode`/`CheckAITokenQuota`/agent bridge（ADR-0024/0025）、BlindSpot proto 字段、防线 A/B 已有输出。NEW：诊断面板 + AI 修复闭环 UI + 静默开关。

### 验收 + 对抗证明
- 致命违反 → 红标诊断 + AI 帮修可用；应用修复后重过防线转绿。
- **对抗证明**：删诊断面板 → "不可信"结果无诊断信息（回归必红）；致命类不该出现"静默"开关（出现即 bug）。

### ✅ 完工记录（2026-08-10）
- **Proto**：`BacktestBlindSpot` 加 `category`+`location` 字段；`GetBacktestRunResponse` 加 `blind_spots` 字段（field 7）。
- **Backend**：`parseBacktestResult` 提取 BlindSpots；watch/auto_gate/crud 全部发送 `BlindSpots`（6 处）。
- **Frontend `DiagnosticPanel.tsx`**（NEW）：severity 三级分组（致命/警告/信息）+ category 人话文案 + 风险类静默开关（`localStorage bt_silence_{strategyId}`）+ AI Fix 按钮（仅 fatal/info）。
- **Frontend `useAIFix.tsx`**（NEW）：AI 修复闭环 hook — `codeAssistApi.revise` → Modal diff 预览 → `strategyVersionApi.updateCode` → 自动重回测。
- **对抗证明**：`TestParseBacktestResult_BlindSpots` — 删 `parseBacktestResult` BlindSpots 提取 → 测试必红。红队自审：fatal 组无静默 toggle；AI Fix 按钮仅 fatal/info 显示。
- `go build`✅ + `go test` 3 新测试绿✅ + `npx tsc --noEmit`✅ + `npm run build`✅。
- **✅ 审计方实测通过（2026-08-10）**：backend `parseBacktestResult` 提取 BlindSpots + watch/persistence/code_check 6 处发送（实测）+ proto `BacktestBlindSpot.blind_spots=7`+category（实测）+ 前端 `DiagnosticPanel.tsx`（**fatal 红 + 无静默开关**、warning 静默 toggle、info；`hasFixable = fatal || info` AI Fix 门控——对抗点全中）+ `useAIFix.tsx` 闭环 + `parse_blindspots_test.go` 对抗测绿 + `go build` exit 0 实测。

---

## Part C · MT golden trace 对拍（精度残差兜底）

> ⏸️ **搁置（2026-08-10，不正式取消——现在不做）**。理由（产品决策）：① **喂入摩擦**——需人工导 MT HTML，使用面极短；② **只能标准 EA**——运营方不拿用户 EA 跑 MT（IP），故 golden trace 永远覆盖不了用户真实策略，价值窄；③ **FEAT-4 实盘 vs 回测对比已更好覆盖用户策略精度**（用用户自己的真实实盘数据对拍，比标准 EA 对 MT 更对、更相关）；④ 引擎回归有防线 A/B + 蜕变测试兜着。**反应式再做**：哪天现有测试漏掉、用户能感知的精度 bug 真出现，手动导一次 trace 对拍定位——不预防性建基础设施。spec 保留供届时参考。

> **ADR §8 原文**：解决无恒等式的语义偏差（如 iMACD 差 0.0001）。**决策：现在不做，成熟期执行**。开发期人工用 MT4/MT5 终端跑标准 EA 导 HTML 入 `testdata/golden/`，复用 `parity_test.go` 扩展为基准（交易级匹配率 + 净利润偏差%）。**需人类协助**（到时用 MT 终端导 trace，一次性）。
> **Phase**：~~mature~~ → **搁置（2026-08-10）；反应式触发再做**（真出现精度 bug 时）。

### 背景
确定性防线抓不到纯语义精度偏差（平台 iMA 与 MT4 差 0.0001）。唯一兜底 = 用真实 MT4/MT5 终端跑同一 EA 的结果作 golden，平台对拍，量出偏差%。这是"平台精度体检"，永远有残差（ADR §2.4 80% 上限）。

### 目标 / 非目标
- **目标**：① 定义 golden trace 格式（MT4/MT5 Strategy Tester HTML 报告解析 → 标准化 trades + metrics）；② 扩展 `parity_test.go` 为 golden 对拍基准：交易级匹配率（ticket/time/volume/side 对齐率）+ 净利润偏差% + 逐笔偏差；③ 一组标准 EA golden（趋势/均值回归/突破/振荡器，复用 LAUNCH-1 的 20 策略用例）入 `testdata/golden/`；④ CI 跑 golden 对拍，偏差超阈值标红。
- **非目标**：不做 100% 精确匹配（永远有残差）；不做实时对拍（仅回测基准）。

### 实现
| # | 任务 | 锚点 |
|----|------|------|
| 1 | **【需人】** 用 MT4/MT5 终端跑 5-10 个标准 EA（XAUUSD/EURUSD 1h），导 HTML 报告入 `backend/strategy/backtest/testdata/golden/*.html` | 一次性人工 |
| 2 | HTML 解析器：MT4/MT5 Strategy Tester 报告 → `[]GoldenTrade{time,volume,side,price,profit}` + 净利润 | 新 parser |
| 3 | 扩展 `parity_test.go`：同参数跑平台回测 → 与 golden 对齐（按 time）→ 算匹配率 + 净利润偏差% + 逐笔 diff | `backend/strategy/backtest/parity_test.go`（已存在，扩展）|
| 4 | 阈值 + 报告：匹配率 < X% 或净利润偏差 > Y% → 测试红 + 输出偏差明细 | test |
| 5 | CI 接 golden 对拍（独立 job，`//go:build golden` tag，不拖慢主 CI）| ci.yml |

> **复用核对**：`parity_test.go` @ `backend/strategy/backtest/`（已存在，扩展非重写）、LAUNCH-1 的 20 策略用例（选 5-10 作 golden 源）。NEW：HTML parser + golden 对拍逻辑 + golden trace 文件。

### 验收 + 对抗证明
- 5-10 个标准 EA golden 对拍跑通；匹配率 + 偏差% 有量化输出。
- **对抗证明**：故意改坏 SimBroker（如撮合价偏 1%）→ 净利润偏差% 飙升 → golden 测试红。
- **前置依赖**：人工导 trace（task 1）——这是本 part 的瓶颈，spec 落地后由人择期执行。

---

## 三 part 的 phase 与优先级

| Part | Phase | 何时做 | 依赖 |
|------|-------|--------|------|
| **B 用户侧恢复前端** | ✅ done（2026-08-10 审计方实测） | 已完成 | — |
| **A 判决型误报处置** | post-launch | 有真实误报数据后 | Part B 的前端反馈入口可复用 |
| **C MT golden trace** | ⏸️ 搁置（2026-08-10） | 反应式：真出现精度 bug 时再做 | 喂入摩擦 + 只能标准 EA（IP 限制）价值窄 + FEAT-4 实盘对比已更好覆盖 |

> 现状：**B ✅**；**A 待 post-launch 误报数据**（唯一真待做）；**C 搁置**（不正式取消，反应式触发）。

## 完工回填（每 part 各自）
每 part 完工：registry 对应条目 🟦→✅（标 part）+ handover 变更日志 + 对抗证明。3 part 各自独立验收，不互相阻塞。不自行宣告 ✅，等审计方实测。
