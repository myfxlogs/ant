# 施工交接：LIVE-REDESIGN-2TAB 返工 —— UI-4 对抗测试补强（唯一返工项）

> **审计方 2026-08-15**：LIVE-REDESIGN-2TAB 批次（`acd5fff1`）验收 9 通过 / 1 返工。返工 = **UI-4 双流 join 对抗测试无效**。本文件是唯一任务，勿扩大范围。

## 问题（审计方实证）

`frontend/src/test/live-ui-antitest.test.tsx:104-130`（UI-4）自建 Map 自测自：

```ts
// 测试里是这一行（= 组件内联逻辑的拷贝）：
const joined = schedules.map(s => ({ ...s, active: activeBySchedule.get(s.id) }))
```

真逻辑在 `LiveStrategyPage.tsx:107-113`（`joinedRows` useMemo 内联）。**删真代码 → 测试仍绿**（「测试测拷贝不测真代码」，POST-1 同模式）。另外 `:115-118` 的 orphan 逻辑同样内联无测试。

## 修复方案（唯一正确解：抽纯函数，组件与测试同源）

1. **新建 `frontend/src/pages/strategy/components/live/joinLiveData.ts`**，导出两个纯函数（类型从 `LiveStrategyPage` 现有 import 借用 `ActiveStrategy` / `ScheduleRow`）：

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

2. **`LiveStrategyPage.tsx`**：
   - `:107-113` 改为 `const joinedRows = useMemo(() => joinSchedulesWithActive(schedules, activeStrategies), [schedules, activeStrategies]);`
   - `:115-118` 改为 `const orphanRuns = useMemo(() => findOrphanRuns(activeStrategies, schedules), [activeStrategies, schedules]);`
   - 删除文件内本地 `JoinedRow` 接口定义（若有），改 import 自新模块；检查 `:19-20` 的 `isLogButtonDisabled`/`isHealthButtonDisabled` **保留不动**（UI-3 已验收有效）。
   - ⚠️ 注意：若 `JoinedRow` 被 `MyStrategiesTable.tsx`/`ScheduleExpandedRow.tsx` import，不要动那两个文件的 import 路径——要么让它们继续 import 自 `LiveStrategyPage`（若其导出了该类型），要么统一改到新模块。**最小改动：新模块定义 `JoinedRow`，页面与表组件统一 import 新模块**（页面原来怎么导出就保持怎么导出，避免破坏其他 import）。

3. **`test/live-ui-antitest.test.tsx` UI-4**（`describe('UI-4'...)` 整块重写）：改为 import 真函数：

```ts
import { joinSchedulesWithActive, findOrphanRuns } from '@/pages/strategy/components/live/joinLiveData'

describe('UI-4: dual-stream join (real function)', () => {
  it('join: no matching active → active undefined', () => {
    const joined = joinSchedulesWithActive([{ id: 's1', ... }], []);
    expect(joined[0].active).toBeUndefined();
  })
  it('join: scheduleId match → active attached', () => {
    const joined = joinSchedulesWithActive([{ id: 's1', ... }], [{ scheduleId: 's1', pnl: '100.50', runId: 'r1', ... }]);
    expect(joined[0].active?.pnl).toBe('100.50');
  })
  it('join: active with empty scheduleId ignored', () => {
    const joined = joinSchedulesWithActive([{ id: 's1', ... }], [{ scheduleId: '', pnl: '1', ... }]);
    expect(joined[0].active).toBeUndefined();
  })
  it('orphan: active with no scheduleId is orphan', () => {
    const orphans = findOrphanRuns([{ scheduleId: '', ... }, { scheduleId: 's1', ... }], [{ id: 's1', ... }]);
    expect(orphans).toHaveLength(1);
    expect(orphans[0].scheduleId).toBe('');
  })
  it('orphan: active with unknown scheduleId is orphan', () => {
    const orphans = findOrphanRuns([{ scheduleId: 'ghost', ... }], [{ id: 's1', ... }]);
    expect(orphans).toHaveLength(1);
  })
})
```

（mock 对象字段以 `ActiveStrategy` 真实字段为准，`scheduleId`/`pnl`/`runId` 必须有，其他可选。）

## 对抗证明（必做，删行必红）

⚠️ **陷阱提醒（审计方实测推演，勿踩）**：`if (a.scheduleId)` 守卫删掉**不会让 join 测试红**——守卫只阻止空 scheduleId 进 map 的 `''` 键，而行 id 非空，`''` 键永不被 lookup。所以：

1. **join 对抗**：把 `joinSchedulesWithActive` 函数体中和为「不建 map、直接返回无 active」（即删掉整个 `for` 循环 + map 构造，返回 `schedules.map(s => ({ ...s }))`）→ **UI-4「match → active attached」用例必 RED**（join 断了，指标全 undefined）。恢复后绿。
2. **orphan 对抗**：删除 `findOrphanRuns` 中 `!a.scheduleId ||` → **「orphan: empty scheduleId」用例必 RED**（空 scheduleId 不再判孤儿，orphans 长度 0）。恢复后绿。
3. 两个删行验证都 RED 后恢复原代码，全量 vitest 绿。

> 备注：empty-scheduleId join 用例（第 3 个）保留——它钉住防御语义（空 scheduleId 的 active 不挂任何行），虽当前行为不可观察，但防未来 refactor 改键语义。

## 门禁（全绿才算完工）

- `cd frontend && npx tsc --noEmit` 0 err
- `cd frontend && npx vitest run` 全绿
- `cd frontend && npm run build` 成功
- `cd backend && go run ./tools/check-file-lines --strict` 0 ERROR
- 无需动后端（纯前端返工），无需重新部署（前端 docker cp + nginx reload 一次即可，或随下批一起）

## 红队自审（施工方自查，逐项打勾）

- [ ] 抽函数是**纯移动**：join/orphan 语义与 `LiveStrategyPage.tsx:107-118` 逐字一致，无"顺手优化"（如改 filter 条件、加排序）。
- [ ] `JoinedRow` 类型只有一个定义源（新模块），页面/表组件 import 路径不重复定义、不破坏现有 import。
- [ ] 对抗证明的"删守卫不红"陷阱已避开（empty scheduleId 用例存在）。
- [ ] UI-2/UI-3/UI-1 测试未动（ScheduleTable 渲染测试 / isLogButtonDisabled / getEnableNavigateTarget 已验收有效）。
- [ ] `isLogButtonDisabled`/`isHealthButtonDisabled`（LiveStrategyPage.tsx:19-20）原样保留，UI-3 引用不破坏。
- [ ] 完工回填 registry（LIVE-REDESIGN-2TAB 条目状态 🟦返工中 → 追加你的完成记录，不删审计方事实陈述）+ handover 变更日志一行。
- [ ] 不自行宣告完成——等审计方独立删行复测 + 门禁复跑后，才由审计方权威标 ✅done。

## 验收标准（审计方将逐项核）

1. `joinLiveData.ts` 与 :107-118 逻辑逐字一致（git diff 仅移动）。
2. 审计方独立删行复测：删守卫行 → UI-4 红；恢复 → 绿（断言级）。
3. tsc / vitest / build / strict 全绿复跑。
4. 前端部署后 Live 页 join 行为与现网一致（Tab1 运行中行有指标、未运行行 "-"、临时运行小节不丢）。
