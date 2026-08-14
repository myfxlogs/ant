# 施工交接：Live 页 2-Tab 重设计（策略为中心）——批次 B（前端本体）

> **审计方定论 2026-08-15（2026-08-15 二次更新：前置全部就绪，本文件现为 §3-only 现状版）。** 用户以第一真实用户身份否决旧 3-tab（持仓不可见/日志埋弹窗/管理动作跨 tab）。改为 2-tab 策略为中心。
>
> **前置状态（全部 ✅，无需再做）**：
> - §0 MARGIN-GATE-2 ✅done（margin fail-closed + USD 公式，生产验证 0 拒单）
> - §2 批次 A ✅done（magic 全链：MT4 下单传 magic / PositionSnapshotItem.magic_number / `GetSchedulePositions` RPC + 对抗测试）
> - 双流字段全就绪：`StrategySchedule.is_running/active_run_id/signal_count` + `ActiveStrategy.pnl/last_signal_at/last_tick_at/schedule_id(=13)/strategy_name(=14)`
>
> **铁律**：对抗证明 + commit + 部署 + 回填 registry + 不自行宣告完成。

---

## §3 前端 2-Tab 本体（本批唯一任务）

### Tab1「我的策略」——双流 join 一张表

**数据源（全部已存在，勿新建流）**：
- `strategy-schedules.ts` 的 `watchSchedules` 流（低频）——配置态 + `isRunning/activeRunId/signalCount`（`libraryTypes.ts:20-22`）
- `LiveStrategyPage.tsx:85` 的 `watchActive` 流（高频）——`pnl/bid/ask/lastSignalAt/lastTickAt/lastError/scheduleId`
- **join 键**：`schedule.id ↔ active.scheduleId`（`ActiveStrategy.schedule_id=13` 已有；`activeRunId` 冗余可用但 scheduleId 是主键关联）

**列**（join 后一行 = 一个 schedule）：
`[策略名 | Symbol/TF | 账户 | 状态(绿●运行中 isRunning+active/黄○已启用未运行 isActive/灰○停用) | 报价(stale标记复用现 :153 逻辑) | 信号数+最后时间(复用 :165 灰/橙档) | PnL(复用 :172 绿红) | 错误(tooltip) | 操作(▶启用/⏹停用/⚡立即运行/✎编辑)]`

操作全按 schedule_id：`toggleSchedule`（strategy-schedules.ts）/ `onManualTrigger`（LiveSchedulesTab.tsx:122 现成，**含 #2 修复后的 params 传递**）/ 编辑跳现有表单。

**状态合并逻辑**：`isRunning`（schedule 流）为准显示运行状态；`activeRunId` 存在 → 从 active 流 join 指标列；**不存在 → 指标列全 "-"**（红队 #3：别崩）。active 流里有孤儿 run（无对应 schedule，如手动 paper 测试）→ 表底单独小节"临时运行"展示（别丢，用户在测的策略）。

### 行展开（点行，非弹窗）

Table `expandable.expandedRowRender`，四个内嵌区（tab 或纵向堆叠，选简单的）：
1. **持仓**：`strategyClient.getSchedulePositions(scheduleId)`（`strategy_pb.ts:127` 已生成 client）。刷新遵循 push-first：展开时拉一次 + `watchActive` 推送时刷新（active 事件带 pnl 变化即重拉），**勿 setInterval 轮询**（红队 #4）。
2. **最近信号**：现有 signals 数据源（watchSignals / 已有 signal log 数据），最近 20 条时间线。
3. **运行日志**：`logClient.getScheduleRunLogs`（`log.ts:86` 现成，带分页）。
4. **配置摘要**：symbol/timeframe/参数/schedule_type + 「编辑」跳现有编辑入口。

### Tab2「运行历史」
保留现有 Run History 表（`LiveStrategyPage.tsx:222` 块原样迁移）。**删除 Active Runs、Schedules 两个 tab**（它们的全部信息已被 Tab1 吸收）。

### 其他
- URL `?tab=` 参数兼容：`active|schedules` → 都映射到新 Tab1；`history` → Tab2。
- 空态 CTA「创建策略」（现 :245 Empty 组件迁移）。
- paper/live Tag 区分（cfg.Mode 或 schedule 上标注）。
- **LiveStrategyPage.tsx 拆分注意**：现 305 行（🟡 超标 22%），新表组件抽独立文件（如 `components/live/MyStrategiesTable.tsx` + `ScheduleExpandedRow.tsx`），主页面只留 tab 骨架 + 流管理。**拆后 check-file-lines --strict 必须 0 ERROR**。
- i18n：新 key 走 `t('strategy.live.*', { defaultValue })` + gen-missing + **translate-zh-cn + translate-llm 四 locale**（别只 gen 不翻，PIPE-F5 教训）。

---

## 验收标准（审计方将逐项核）

1. 双流 join 正确：运行中策略行有实时指标；未运行策略行指标 "-" 不崩。
2. 行展开四区数据真实（持仓含 magic 过滤后的本策略仓位——用生产 MT4 账户验证）。
3. 旧 3-tab 删除干净（无死代码/死 i18n key）。
4. 对抗证明：删 join 逻辑 → 运行中行指标全 "-"（RED）；删展开区持仓拉取 → 展开空（RED）。vitest 组件测试或最小渲染测试。
5. `?tab=schedules` 旧链接落到 Tab1 不 404。
6. strict 0 ERROR + tsc 0 err + npm build + vitest 全绿。

## 红队自审
- [ ] join 键用 scheduleId（非 activeRunId——它是派生值）。
- [ ] active 流孤儿 run（手动 paper 测试）别静默丢——"临时运行"小节。
- [ ] 展开持仓刷新非轮询（push-first）。
- [ ] schedule_id 为空的手动 run（无 schedule）在 join 时容错。
- [ ] i18n 四 locale 真翻译（非英文兜底）。
- [ ] 主页面文件不超线（新组件抽文件）。
- [ ] 完工回填 registry（新增 LIVE-REDESIGN 条目 → ✅done + 对抗证明）+ handover 变更日志。
