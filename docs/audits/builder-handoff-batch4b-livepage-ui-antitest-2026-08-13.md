# 施工交接批次 4b：Live 页 UI 对抗测试补强（审计方验收遗留返工单）

> **🆕 2026-08-13 已并入批次 5**：本文件 UI 三项任务已并入 `builder-handoff-batch5-antitest-2026-08-13.md`（含 LIVE-4 + LIVE-7b + UI 三项统一补强）。施工方请以 batch5 为准。本文件保留作历史痕迹，勿单独施工。
>
> 审计方（Claude Code）2026-08-13 批次4 验收后发出。**one task = one scope**：只做本文件列的三项 UI 对抗测试 + 一个低优清理。**不重新审计、不自由发挥、不扩大范围。**
> 背景：批次4 Live 页 UI 实现已验收 ✅（tab 序 / strategyName 列 / 日志+健康按钮 / Enable→tab1 联动 / 状态诚实 / healthId 联动，代码核对全符合），但 **spec 必做的 3 项对抗测试零存在**（`LiveSchedulesTab`/`ScheduleTable`/`LiveStrategyPage` 无任何 test 文件，`tests/e2e/deploy-schedule.spec.ts` 无新断言）。按铁律：**删了还绿 = 测试无效 = 未完成**——实现合格但对抗缺失 = 本批次任务未完成。

---

## 一、任务清单

| # | 场景（spec 对抗 #3/#4） | 红（删行/修复前） | 绿（修复后） | 位置 |
|---|--------------------------|------------------|-------------|------|
| 1 | Enable 成功自动跳 tab1（`LiveSchedulesTab.tsx` `onToggleActive` 内 `if (next) navigate('/strategy/live?tab=active')`） | 删联动行 → 不跳转 | 自动跳运行监视 | `LiveSchedulesTab.tsx:196-197` |
| 2 | last_error 红色错误显示（`ScheduleTable.tsx:153-157`） | 删渲染行 → 错误不可见 | 红色 ⚠ + Tooltip 显示 | `ScheduleTable.tsx:153-157` |
| 3 | 日志/健康按钮 scheduleId 空时 disabled（`LiveStrategyPage.tsx` activeColumns） | 断言失败 | 空→disabled / 非空→enabled | `LiveStrategyPage.tsx:120-127` |
| 4（低优） | healthId modal 关闭后清理 URL（红队自审残留，刷新重现） | — | 关闭 modal 清 `healthId` 参数 | `LiveSchedulesTab.tsx` health effect |

## 二、对抗证明（必做，审计方将独立删行复测）

**每项测试必须是"行为级"断言（删关键行 → RED），不是拷贝测试：**

1. **Enable→tab1**：组件测试渲染 `LiveSchedulesTab`（mock `react-router-dom` 的 `useNavigate` + mock `strategyScheduleV2Api.toggle` 成功 + 其余依赖 hook 按现有 mock 模式）→ 点击行 Switch（启用）→ 断言 navigate 被调用且 path = `/strategy/live?tab=active`。**删 `if (next) navigate(...)` → 测试 RED。** 若依赖链 mock 过深，允许抽 `onToggleActive` 可测薄函数（行为不变），但**必须走真实渲染路径**，禁手写 div 冒充组件。
2. **last_error 红显**：组件测试渲染 `ScheduleTable`（`schedules=[{...lastError:'oops'}]`）→ 断言页面出现 ⚠ 前缀 + `oops` 文本（或 `row.lastError.slice(0,40)` 断言）且为 danger 样式（`type="danger"` 断言）。**删 `{row?.lastError && (...)}` 渲染块 → RED。**
3. **disabled 断言**：渲染 `LiveStrategyPage` 或抽 activeColumns 为可测函数（行为不变）→ 构造 `{scheduleId:''}` 与 `{scheduleId:'abc'}` 两条记录 → 断言空 scheduleId 行日志/健康按钮 `disabled`，非空行 `enabled`。**删 `disabled={!record.scheduleId}` → RED。**
4. **healthId URL 清理**：关闭 HealthModal → 断言 URL 不再含 `healthId` 参数（可用 navigate mock 断言）。**删清理行 → RED**（此项低优，可与 1-3 一起做）。

**门禁**：tsc 0err / vitest 全量过（存量 144 + 新增）/ npm build / go build（未动后端也应绿）。

## 三、红队自审（逐条给出结论）

- [ ] 测试是否走**真实组件渲染**（`render(<ScheduleTable/>)`），非手写 div 拷贝
- [ ] 删行实验是否每项**真实执行过**（记录红→还原→绿输出），非只写断言
- [ ] mock 是否最小化：只 mock 被测组件实际依赖（useNavigate / api client / i18n / 数据 hooks），不 mock 被测逻辑本身
- [ ] 现有测试是否受影响（vitest 存量 144 全绿）
- [ ] 是否意外改动实现代码（实现已验收，除任务 4 的 URL 清理外**禁止改实现**；抽函数例外需行为不变 + 注释说明动因）
- [ ] 门禁全绿实测后记录输出
- [ ] 提交核对：registry/handover 变更日志**只追加不删**（pre-commit 钩子拦删，禁 `--no-verify`）；**⚠️ 上批教训：不得把审计方记录行替换为自身声明**——追加新行，不改既有行

## 四、回填（不做 = 任务判失败）

1. `docs/audits/tech-debt-registry.md`：批次4b 补强行追加（每项测试 + 删行红绿输出 + commit）。**只追加，不改审计方验收行。**
2. `docs/audits/handover-audit-plan.md` 变更日志加一行。
3. 完成后报告：每项测试文件/断言 + 删行红绿记录 + 门禁输出 + 回填位置。**不自行宣告完成**，等审计方独立删行复测后 ✅ 才权威。

## 五、沟通

- 完成后一句话报告：`批次4b 完成：T1/T2/T3(+T4) 测试文件 + 删行红绿记录 + 回填位置`。
- 聊天一句话（定位+铁律）："4b 补强：3 项 UI 对抗测试必走真实渲染 + 删行必红实测；只追加不改审计方行。"
