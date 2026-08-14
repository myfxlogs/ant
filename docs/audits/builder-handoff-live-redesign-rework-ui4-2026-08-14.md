# 施工交接：LIVE-REDESIGN-2TAB 返工 —— 3 项（UI-4 对抗补强 / i18n 漏 4 key / 死代码函数）

> **审计方 2026-08-15**：LIVE-REDESIGN-2TAB 批次（`acd5fff1`）验收 9 通过 / **3 返工**（原 1 项，审计方自我审计全量复核抓出 2 项新增）。本文件是唯一任务，勿扩大范围。

---

## 返工 ① UI-4 双流 join 对抗测试无效

### 问题（审计方实证）

`frontend/src/test/live-ui-antitest.test.tsx:104-130`（UI-4）自建 Map 自测自：

```ts
// 测试里是这一行（= 组件内联逻辑的拷贝）：
const joined = schedules.map(s => ({ ...s, active: activeBySchedule.get(s.id) }))
```

真逻辑在 `LiveStrategyPage.tsx:107-113`（`joinedRows` useMemo 内联）。**删真代码 → 测试仍绿**（「测试测拷贝不测真代码」，POST-1 同模式）。另外 `:115-118` 的 orphan 逻辑同样内联无测试。

### 修复（唯一正确解：抽纯函数，组件与测试同源）

1. **新建 `frontend/src/pages/strategy/components/live/joinLiveData.ts`**：

```ts
export interface JoinedRow extends ScheduleRow { active?: ActiveStrategy }

export function joinSchedulesWithActive(schedules: ScheduleRow[], activeStrategies: ActiveStrategy[]): JoinedRow[] {
  const activeBySchedule = new Map<string, ActiveStrategy>();
  for (const a of activeStrategies) {
    if (a.scheduleId) activeBySchedule.set(a.scheduleId, a);
  }
  return schedules.map(s => ({ ...s, active: activeBySchedule.get(s.id) }));
}

export function findOrphanRuns(activeStrategies: ActiveStrategy[], schedules: ScheduleRow[]): ActiveStrategy[] {
  const scheduleIds = new Set(schedules.map(s => s.id));
  return activeStrategies.filter(a => !a.scheduleId || !scheduleIds.has(a.scheduleId));
}
```

2. **`LiveStrategyPage.tsx`** `:107-118` 改为 useMemo 包两个函数调用；`JoinedRow` 类型只留一个定义源（新模块），页面/表组件统一 import。
3. **`test/live-ui-antitest.test.tsx` UI-4 整块重写**为 import 真函数，用例：
   - 无匹配 active → `active` undefined
   - scheduleId 匹配 → `active.pnl` 可读（mock 含 `scheduleId`/`pnl`/`runId`，其余字段按 `ActiveStrategy` 真实类型可选补）
   - **空 scheduleId active 不挂任何行**（钉防御语义）
   - orphan：无 scheduleId 的 active 是孤儿 / 未知 scheduleId 的 active 是孤儿
4. UI-2（ScheduleTable 渲染）/UI-3（见返工 ③）/UI-1（getEnableNavigateTarget）**不动**。

### 对抗证明（⚠️ 陷阱已推演，勿踩）

`if (a.scheduleId)` 守卫删掉**不会红**（`''` 键永不被 lookup，行 id 非空）。正确对抗：
1. **中和 join 函数体**（删 for 循环 + map 构造，返回 `schedules.map(s => ({ ...s }))`）→ 「match → active attached」用例 RED → 恢复绿。
2. **删 `findOrphanRuns` 的 `!a.scheduleId ||`** → 「empty scheduleId 孤儿」用例 RED → 恢复绿。
3. 全量 vitest 绿。

---

## 返工 ② i18n 漏 4 key（gen-missing 未生效，PIPE-F5 同款复发）

### 问题（审计方实证）

本批新文件用了 4 个 `strategy.schedules.*` key，**textproto + base.ts 全 4 locale 完全不存在**（en 也没有）→ runtime 英文兜底：

| key | 位置 | 当前显示 |
|---|---|---|
| `strategy.schedules.status.disabled` | MyStrategiesTable.tsx:98 状态三色「Disabled」 | 英文兜底 |
| `strategy.schedules.actions.runNow` | MyStrategiesTable.tsx:176 Run Now tooltip | 英文兜底 |
| `strategy.schedules.deleteConfirm.title` | MyStrategiesTable.tsx:198 删除确认 | 英文兜底 |
| `strategy.schedules.table.schedule` | ScheduleExpandedRow.tsx:154 配置区「Schedule」 | 英文兜底 |

（对照：`strategy.schedules.status.running/idle/enabled` 旧 key 已在 textproto ✓——本批新引入的 4 个没走 gen-missing。）

### 修复

1. `cd frontend && npx tsx scripts/i18n-gen-missing.ts` 生成缺失 key（含以上 4 个）。
2. `npx tsx scripts/i18n-translate-zh-cn.ts`（zh-cn 字典）+ `i18n-translate-llm.ts`（zh-tw/ja/vi，**别只 gen 不翻**）。
3. `npx tsx scripts/i18n-build.ts` 重生 resources；确认 `frontend/src/i18n/resources/*/base.ts` 4 locale 都有 4 个 key 且**非英文值**（抽查 `禁用`/`立即运行`/`删除此计划？`/`计划`）。

### 对抗证明

删掉 base.ts 中任一新 key（或 textproto 中对应行）→ 该 locale 该文案回英文 → 用最小渲染测试断言中文文案存在（或人工抽查产物 diff）。至少：提交 diff 中 `base_zh-cn.textproto` 必须含 4 个新 key 行。

---

## 返工 ③ 死代码函数 isLogButtonDisabled / isHealthButtonDisabled

### 问题（审计方实证）

`LiveStrategyPage.tsx:19-20` 两个函数（`24828c9c` 为对抗测试引入）**生产零调用**：2-tab 重设计删了调用点但保留导出，antitest UI-3（:71-83）引用使其"看起来活着"——验收项 3「无死代码」不过。且按钮 disabled 逻辑在 MyStrategiesTable 是内联 `disabled={!row.id}`（:184 log / :189 health），两处表达同一语义但各自维护。

### 修复（最优解：函数移共享模块 + 真调用点，比删除更优）

1. 两个函数移到 `components/live/`（如 `MyStrategiesTable.tsx` 同目录 `disabledGuards.ts` 或并入 `joinLiveData.ts` 同模块风格），**保留原签名与语义**（`!scheduleId`）。
2. `MyStrategiesTable.tsx` actions 列 `:184`（log 按钮）与 `:189`（health 按钮）的 `disabled={!row.id}` 改为调用这两个函数。delete 按钮（:200）内联保留（无对应函数）。
3. `LiveStrategyPage.tsx` 不再定义/导出这两个函数（避免双定义源）；antitest UI-3 的 import 路径改到新模块，断言不变。

### 对抗证明

删除新模块中 `isLogButtonDisabled` 函数体（返回 `false`）→ UI-3「disabled when empty」用例 RED → 恢复绿。

---

## 门禁（全绿才算完工）

- `cd frontend && npx tsc --noEmit` 0 err
- `cd frontend && npx vitest run` 全绿
- `cd frontend && npm run build` 成功
- `cd backend && go run ./tools/check-file-lines --strict` 0 ERROR（若新文件超 250 行则拆）
- 纯前端返工，无需动后端；前端部署一次（docker cp + nginx reload）或随下批一起

## 红队自审（施工方自查，逐项打勾）

- [ ] 返工①：抽函数是**纯移动**，join/orphan 语义与 :107-118 逐字一致，无顺手优化（改 filter 条件/加排序等）。
- [ ] 返工①：对抗证明避开「删守卫不红」陷阱——用的是中和 join 函数体 + 删 orphan 守卫，两个都实测 RED。
- [ ] 返工②：gen-missing 跑完**必须**确认 4 个新 key 落 textproto（不是只生成在内存）；翻译后抽查 4 locale 非英文值；`npm run build` 后运行时资源含 key。
- [ ] 返工③：函数语义未变（`!scheduleId`），只搬位置 + 加真调用点；`LiveStrategyPage.tsx` 删除导出后全库无残留 import（grep `isLogButtonDisabled|isHealthButtonDisabled` 只剩新模块 + 测试）。
- [ ] UI-2/UI-1 测试未动；UI-3 断言不变（只改 import 路径）。
- [ ] 完工回填 registry（LIVE-REDESIGN-2TAB 条目 3 项各回填完成记录，不删审计方事实陈述）+ handover 变更日志一行。
- [ ] 不自行宣告完成——等审计方独立删行复测 + 门禁复跑后，才由审计方权威标 ✅done。

## 验收标准（审计方将逐项核）

1. 返工①：`joinLiveData.ts` 与 :107-118 逻辑逐字一致（git diff 仅移动）；审计方独立删行复测 2/2 RED。
2. 返工②：4 locale textproto + base.ts 都有 4 个新 key 真翻译；审计方全量 diff 复跑 0 缺失（live + schedules + common 三前缀）。
3. 返工③：`grep -rn "isLogButtonDisabled" frontend/src --include=*.tsx` 只剩新模块 + 测试；UI-3 独立删函数体复测 RED。
4. tsc / vitest / build / strict 全绿复跑。
5. 前端部署后 Live 页：Tab1 状态「停用」/「立即运行」tooltip/删除确认/配置区「计划」显示本地化文案（非英文）。
