# Builder Handoff — EDIT-PARAMS-FIX（编辑参数功能修复批）

> 批次名：EDIT-PARAMS-FIX ｜ 日期：2026-08-15 ｜ 审计方：Claude Code ｜ 施工方：Windsurf
> 审计依据：registry 变更日志「EDIT-PARAMS 审计」（3 处第一性违背 + 1 前端 🔴 + 2 🟡），全链路证据在案。
> 涉及功能块：`frontend`（EditParamsModal）+ `strategy-runtime`（UpdateSchedule/引擎重启）。
> ⚠️ 本批与 MACD 监控共存：后端部署会重启 run（调度自动接力，正常现象）。

---

## 0. 背景与根因（一段读懂）

链路：`EditParamsModal` → `codeAssist.validateExtended` 提取策略 `input` 声明（✅ 权威源正确）→ `UpdateSchedule`（**零校验**）→ `parameters` bytea → VM `injectParams`（**OnInit 一次性注入**，只认声明过的键，`interp_runner.go:371-375` 未声明键静默跳过）。三个断裂：① 运行中编辑静默无效（`engine.Notify` 只重算定时器，`schedule_engine.go:139`，运行中会话不重启不提示）；② 后端不校验（schema 就在代码里却不查）；③ 五个"风控参数"全代码库零消费方（安慰剂 UI）。外加前端 useEffect 无限循环 🔴。

## 1. 任务分解

### Task E1 — 🔴 EditParamsModal 无限循环（P0，前端）

`frontend/src/pages/strategy/components/live/EditParamsModal.tsx:50-72`：useEffect deps 含 `extractedParams`，effect 内 `loadStrategyParams` → `setExtractedParams(新数组)` → 再触发 → **循环**。两个伤害：模态框打开期间持续 spam `validateExtended`（跑 MQL 提取）+ 每轮 `setStrategyParamValues(vals)` **覆盖用户输入中的值**。

**修法**：拆成两个 effect——
- 加载 effect：deps `[open, schedule?.id]`，只在开窗时 fetch 一次；
- 合并 effect：deps `[extractedParams]`，用函数式更新 `setStrategyParamValues(prev => 仅对 prev 中不存在的键补 default)`（保留用户已改的值，不整体覆盖）。

**对抗（vitest）**：mock `strategyTemplateApi.get` + `codeAssistApi.validateExtended` 带调用计数 → 渲染 modal → 等待稳定 → 断言 `validateExtended` 恰好调用 1 次；用户先改一个参数值再等异步完成 → 断言该值保留。突变（还原旧 deps）→ 调用数 >1 / 值被覆盖 → RED。

### Task E2 — 🔴 运行中编辑静默无效 → 自动重启生效（P1，后端）

**修法**（`backend/internal/connect/strategy/strategy_schedules.go` UpdateSchedule）：保存成功后，若 `s.engine != nil && s.engine.isRunning(id)` **且本次变更了 Parameters/Symbol/Timeframe/AccountID 任一** → 自动重启会话：
```go
s.engine.StopSchedule(id)                       // 现成，schedule_engine.go:423
if err := s.engine.StartSchedule(ctx, id); err != nil { /* log.Warn，不 fail 整个更新 */ }
```
（`StartSchedule` 现成，`schedule_event.go:17`；ToggleSchedule 已用同一对，照抄模式。）重启失败只 warn——参数已持久化，下次启动自然生效，不让更新失败。

**前端提示**（EditParamsModal handleOk 成功后）：若 `schedule.active` 存在（行在运行）→ toast「已保存，策略已重启生效」；否则现有 Updated toast。

**对抗（go test）**：mock engine（或真 engine + 假 repo）——运行中调度 + 变更 Parameters → 断言 Stop+Start 均被调用；突变（删重启调用）→ RED。未运行调度 → 断言不调用 Start。变更仅 name → 断言不重启（避免无谓重启）。

### Task E3 — 🟡 参数按模板 schema 校验（P1，后端）

**修法**（UpdateSchedule，`m.Parameters != nil` 分支）：
1. 取模板代码：`schedule.templateId` → `strategy_templates.code`（repo 现成查询）。
2. **REUSE `extractParams`**（`internal/connect/ai/code_assist_handler.go:275`，需导出为 `ExtractParams` 或移到共享包——禁止复制正则）。
3. 校验传入键值：未声明键 → `CodeInvalidArgument`（错误信息列出未知键名——拼错键保护是本任务核心）；声明键 → 按 `type` 校验可解析（int/long→整数、double→小数、bool→true/false、string→任意）。
4. **遗留豁免**：五个历史死键（defaultVolume/maxPositions/stopLossPriceOffset/takeProfitPriceOffset/maxDrawdownPct）**静默剥离**（存量 schedule 里带着，不拒收不保留——旧数据读回时同样无害被 VM 忽略）。
5. 模板取不到（已删除的 template）→ 跳过校验（降级放行，log.Warn）——不能因模板亡而锁死编辑。

**对抗**：未声明键 → 400（突变：删校验 → 通过 → RED）；int 声明传 "abc" → 400；合法参数集 → 通过；含 5 个遗留键 → 通过且剥离（读回 schedule 确认不存在）。

### Task E4 — 🟡 五个安慰剂字段删除（P2，前端）

`EditParamsModal.tsx` 删除「Risk Parameters」整个区块（defaultVolume/maxPositions/SL·TP offset/maxDrawdownPct 五个 Form.Item，:145-164）+ `buildParametersFromForm` 不再合并它们（handleOk 只发 `parameters: strategyParamValues`，经 E3 会被剥离双保险）。i18n key 留着不删（无害，另批清理）。

**对抗（vitest）**：渲染 modal → 断言查询不到「默认手数/最大持仓」表单项；突变（还原区块）→ 查询到 → RED。

### Task E5 — 🟡 入口范围去重（P2，前端）

「编辑参数」modal 删掉 name/symbol/timeframe/account 四个字段（:124-143——它们属于「编辑」入口的职责，两入口重复）→ modal 变纯参数编辑：标题带策略名（只读展示），body 只有策略 input 参数区。handleOk 只发 `{id, parameters}`。若检查发现「编辑」（onEdit）入口不覆盖这些字段的编辑，则**保留在编辑参数里并反转本任务**（以 onEdit 实际能力为准，报告说明）。

**对抗**：渲染 modal → 无「品种/周期」表单项；有策略 input 参数项。

## 2. 红队自审（交付前逐项打勾）

- [ ] E1：`schedule?.id` 而非对象引用作 dep（行数据每秒变，对象引用会频繁触发）；模态框重开同一 schedule 不重复 fetch 之外的副作用
- [ ] E1：extractedParams 加载失败（模板无代码/校验失败）→ 空参数区 + 不报错弹窗（现状 catch 静默，保持）
- [ ] E2：重启与 Notify 顺序（先存库→再重启）；`isRunning` 与 Stop 之间的竞态（engine 内部有锁，确认 StopSchedule 对不存在 handle 安全）；**别在 timer 型调度上调用 StartSchedule 起流式会话**（StartSchedule 内部已分支，确认即可）
- [ ] E3：templateId 为空（legacy）→ 跳过校验；proto Parameters nil（未传）→ 不校验不动现有值（现状语义保持）
- [ ] E3：校验错误信息含具体键名（用户拼错时能自愈）
- [ ] E4/E5：`parseParametersToForm`/`buildParametersFromForm` 若别处复用（创建调度向导？）→ 只改本 modal 路径，grep 确认调用方
- [ ] 门禁 + verify-adversarial.sh 自检 + 前后端部署（compose build backend + docker cp frontend）

## 3. 复用核对

| 项 | 结论 |
|---|---|
| 会话重启 | **REUSE: `StopSchedule`（engine:423）+ `StartSchedule`（schedule_event.go:17）**，ToggleSchedule 同款模式 |
| 参数提取 | **REUSE: `extractParams`（code_assist_handler.go:275，导出后用）** |
| input 注入语义 | 既有 `injectParams`（interp_runner.go:366）不动——本批只修"到达 VM 之前"的链路 |
| 新能力 | NEW：UpdateSchedule 校验分支 + 前端两处 effect 重构（无现成） |

## 4. 门禁

```bash
cd backend && go build ./... && go test ./internal/connect/... -count=1
go run ./tools/check-file-lines --strict
cd ../frontend && npx tsc --noEmit && npx vitest run && npm run build
bash scripts/verify-adversarial.sh（go 侧三项自检）
```
部署：`docker compose build backend && docker compose up -d backend` + `docker cp dist/. alphaforge-frontend:/usr/share/nginx/html/ && docker exec alphaforge-frontend nginx -s reload`。部署会重启 MACD run——正常（调度自动接力重种子）。

## 5. 验收（审计方）

独立删行复测 5 项对抗 + 部署后实测：① 运行中的 599ddaa5 改参数（如 TakeProfit 50→60）→ 保存 → 日志出现旧 run stopped + 新 run seeded → toast「已重启生效」→ 新 run OnInit 用新参数（signal 或日志佐证）；② 拼错键名 → 400 带键名；③ modal 打开 30s → validateExtended 调用数恒定（Network 面板）；④ 安慰剂字段消失；⑤ MACD 监控不受扰（新 run 接力 + bar age 正常）。

## 6. 回填纪律

registry「EDIT-PARAMS 审计」条目状态更新 + 真实根因/对抗结果；handover changelog 追加一行；append-only；不自宣告 ✅。
