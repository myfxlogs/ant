# Builder Handoff — DEPLOY-PARAMS（上线弹窗补调参面板 + 补行为测试）

> 批次名：DEPLOY-PARAMS ｜ 日期：2026-08-15 ｜ 审计方：Claude Code ｜ 施工方：Windsurf
> 背景：审计定论「上线调度弹窗缺了主功能——创建时完全不能调参」（对标 MT4 拖 EA 到图表的对话框：主体就是参数表）。管线积木全部现成（EDIT-PARAMS-FIX 批刚建好），本批是拼装 + 两个补漏。
> 依赖事实（已核实）：`CreateScheduleRequest.parameters` proto 字段已存在（strategy_schedule_control.proto:15）、`CreateSchedule` handler 已存库（strategy_schedules.go:95）、`validateScheduleParams` 已存在但只接在 Update（:123）。**proto 零改动**。

---

## 1. 任务分解

### Task D1 — 上线弹窗参数区（P1，前端）

`frontend/src/pages/strategy/components/DeployScheduleModal.tsx`：

1. **照抄 EditParamsModal 的参数管线**（勿复制粘贴成死代码——若两处 >30 行相同，抽共享 hook `useStrategyParams(templateId)` 到 `components/live/` 旁，两 modal 共用；抽不抽由实际行数判断，宁抽勿拷）：
   - `loadStrategyParams(templateId)`：`strategyTemplateApi.get` → `codeAssistApi.validateExtended` → `extractedParams`（E1 的教训：**load effect 只依赖 `[open, templateId]`**，别把 extractedParams 放进加载 effect 的 deps——那是刚修掉的无限循环）
   - 参数值 state 初始化 = 各参数 `default`（上线场景没有存量值，全默认预填——MT4 语义）
   - `<StrategyParamsSection>` 渲染（现成组件）
2. `handleSubmit` 的 `createSchedule` 请求加 `parameters: strategyParamValues`。
3. 模态框宽度从默认调到 560（与 EditParamsModal 一致，参数表需要横向空间）；参数区放在触发方式之后、用同样的分隔样式。

**对抗（vitest，放 `src/test/deploy-params-antitest.test.tsx`——该目录已可正常入库）**：① 渲染弹窗（mock 模板+validateExtended 返回 2 个参数）→ 断言两个参数输入框出现且**默认值预填**；② 提交 → 断言 createSchedule 收到 `parameters` 含改过的值；③ 突变（删 parameters 请求字段）→ RED。注意 mock 策略与 `edit-params-antitest.test.tsx` 同款（复用 `mockClient.ts` 模式）。

### Task D2 — Create 路径补 E3 校验（P1，后端）

`backend/internal/connect/strategy/strategy_schedules.go` CreateSchedule：在构建 `ScheduleRow` **之前**（约 :88 前）加：

```go
if m.Parameters != nil && templateID != uuid.Nil {
    if err := s.validateScheduleParams(ctx, templateID, m.Parameters); err != nil {
        return nil, err
    }
}
```

**REUSE `validateScheduleParams`**（UPDATE 侧同款，零新代码）——创建时就拒未知键/类型错，与编辑一致。

**对抗（go test）**：CreateSchedule 带未知键 → 400（断言错误信息含键名）；带合法参数 → 创建成功且读回 parameters 一致。突变（删校验调用）→ RED。

### Task D3 — 补 EDIT-PARAMS 批记档的 2 个行为测试缺口（P2，后端）

上一批复审记档：「maybeRestartSchedule 零行为测试（测的是邻居）+ validateScheduleParams 全路径未测（唯一测试是自构自证）」。本批补上：

1. **`validateScheduleParams` 全路径**：构造可测服务器——`s.svc.GetTemplate` 需要真 svc？看依赖：若 svc 是具体类型难 mock，则抽校验核心为纯函数 `validateParamsAgainstSchema(declared []*antv1.ParameterEntry, params map[string]string) error`（`validateScheduleParams` 调它），纯函数直接测：未知键→400 含键名 / int 传"abc"→400 / 遗留键被剥离（断言 map 副本，不污染原 map——注意值语义）/ 合法集→nil。突变（删 unknown 分支）→ RED。
2. **`maybeRestartSchedule` 行为**：构造 `ScheduleEngine{activeRuns: map[id]*runHandle{...}}` + 捕获 Stop/Start——`StartSchedule` 会走 repo…若难注入，抽决策为纯函数 `shouldRestart(engine, id, substantiveChanged) bool` 不够（测不到动作）。替代：给 ScheduleEngine 的 runHandle 用真实 cancel（可断言 ctx 被 cancel）+ StartSchedule 失败路径只 warn（断言不 panic 不返回错误）。最低要求：**运行中+substantive=true → runHandle 的 cancel 被触发**（构造带 cancel 的 context 观察 Done）。突变（删 StopSchedule 调用）→ RED。

## 2. 红队自审

- [ ] D1：`useStrategyParams` 若抽取——两个 modal 的行为差异（Edit=存量值合并，Deploy=纯默认）放 hook 参数（`initialValues`），别在 hook 里塞分支魔法
- [ ] D1：symbol/timeframe 选择变化不影响参数区（参数只跟 templateId 走）
- [ ] D1：模板无参数（validateExtended 空）→ 参数区显示"无参数"文案而非空白
- [ ] D2：`m.Parameters` 为 nil（旧调用方不带参数）→ 跳过校验照常创建（向后兼容）
- [ ] D2：校验放 marshal cfgBytes 之后 uid 解析之后、构建 row 之前——顺序无副作用
- [ ] D3：`validateParamsAgainstSchema` 剥离遗留键时**不要原地改入参 map**（Update 流程后续还用这个 map 存库——原地删会双重生效，行为虽同但语义脏；返回剥离后副本）
- [ ] 门禁 + verify-adversarial + **前后端都部署**（会重启 MACD run，正常）

## 3. 复用核对

| 项 | 结论 |
|---|---|
| 参数提取+渲染管线 | **REUSE: EditParamsModal 的 loadStrategyParams + StrategyParamsSection**（建议抽 hook 共用） |
| Create 校验 | **REUSE: `validateScheduleParams`**（strategy_schedules.go:123 同款） |
| proto | **零改动**（`CreateScheduleRequest.parameters=6` 已存在，handler :95 已存库——只缺前端发送 + 校验） |
| 新能力 | NEW：`useStrategyParams` hook（若抽）+ `validateParamsAgainstSchema` 纯函数（D3 顺手产物） |

## 4. 门禁

```bash
cd backend && go build ./... && go test ./internal/connect/... -count=1 && go run ./tools/check-file-lines --strict
cd ../frontend && npx tsc --noEmit && npx vitest run && npm run build
```
部署：后端 compose build/up + 前端 docker cp + nginx reload（唯一合法方式）。

## 5. 验收（审计方）

独立删行复测 D1/D2/D3 对抗 + 生产实测：① 从策略卡片点「上线调度」→ 弹窗出现参数区、默认值预填（用 MACD Sample 模板：TakeProfit=50/Lots=0.1 等可见）→ 改 Lots=0.02 → 创建 → 调度启动且 DB `strategy_schedules.parameters` 含 Lots=0.02 → **首个信号 volume=0.02**（端到端：参数从弹窗到 VM 到订单）；② 创建时 API 直发未知键（或 vitest）→ 400；③ MACD 监控不受扰。

## 6. 回填纪律

registry「EDIT-PARAMS 审计」+「上线弹窗缺调参」相关条目状态更新；handover changelog 追加；append-only；不自宣告 ✅。
