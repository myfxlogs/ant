# 施工交接：Live 页 2-Tab 重设计（策略为中心）+ MARGIN-GATE-2 补全

> **审计方定论 2026-08-15（自审修正版）。** 用户以第一真实用户身份否决旧 3-tab（持仓不可见/日志埋弹窗/管理动作跨 tab）。改为 2-tab 策略为中心。**原口头方案经自审发现 2 个技术缺陷，本文件是修正版**。
>
> **前置**：MARGIN-GATE-2（本文 §0）先行或同批——margin 已修部分（overlay+杠杆），剩 2 处。

---

## §0 MARGIN-GATE-2（先做，小）：补 margin 修复的遗漏 2 处

已修：ContractSize overlay（service_orders.go SymbolParams 覆盖）+ SymbolLeverage（handlers_pipeline provider）。**遗漏**：
- **①** `risk/rules.go:38-43` contractSize() helper：state.ContractSize 为零时**静默默认 100000**——删默认；为零 → MarginPreCheck **fail-closed**（reason="contract size unknown for symbol X"，别静默当外汇）。
- **④** `risk/rules.go:325` 公式 `vol×price×CS÷lev`：**USD 本位币对（USD 开头：USDJPY/USDCAD/USDCHF）不应乘 price**（名义价值在 USD 本位币，price 是 USD→计价币汇率不是仓位大小）→ USDJPY 膨胀 ~150 倍全拒。改 `vol×CS×汇率÷lev`：symbol 以 "USD" 开头 → 汇率=1；否则（crypto/金属/EUR 本位）汇率=price。USD 账户假设，跨币对（EURGBP）近似写注释。
- **核实**：overlay 的 `SymbolParam.LotSize`（service_orders.go:171 / order_types.go:37）映射的是 proto `SymbolParams.ContractSize`（字段12）还是 min-lot？若语义不明 → 适配层加明确 `ContractSize` 字段映射字段12，**别用 LotSize**（若是 min-lot=0.01 → 保证金 $0.06 fail-open 裸奔）。
- **对抗**：USDJPY 0.01 lot → 旧 required≈$150,000 拒单 RED / 新 required≈$100（vol×100000÷100）GREEN；BTCUSDm 回归 ≈$6.33；fail-closed：mock 查不到 ContractSize → 拒单 reason 含 "contract size unknown"（删 fail-closed 分支→静默默认 RED）。

## §1 重设计架构（修正版，与口头版的关键差异）

**❌ 原方案缺陷 1**：把 pnl/bid/ask 塞进 WatchSchedules——错。schedule watch 只在调度变更/ticker 时推，**pnl/报价是每 tick 变的**，塞进去 = 表里价格永远 stale。
**✅ 修正**：**前端双流 join**——两个流都已存在且刚加好字段：
- `watchSchedules` 流：配置 + 运行状态（is_running/active_run_id/signal_count）——低频。
- `watchActive` 流：实时指标（pnl/bid/ask/last_signal_at/last_tick_at/last_error）——高频（tick 节流推送）。
- 前端按 `schedule_id ↔ active_run_id` join 成一张表。**后端批次 A 只需做 §2 持仓查询，不再动 WatchSchedules。**

**❌ 原方案缺陷 2**：per-strategy 持仓按 magic 过滤——但 `PositionSnapshotItem`（mthub/broker_types.go:28）**没有 magic_number 字段**，无法过滤。
**✅ 修正**：§2 必须先把 magic 打通进持仓数据，才能过滤。

## §2 批次 A（后端）

1. **magic 进持仓**：`mdtick.OrderUpdate` 捕获 magic（mtapi OnOrderUpdate 的 order 数据带 magic；adapter 侧确认字段映射）→ `PositionSnapshotItem` 加 `MagicNumber int64`。
2. **per-schedule 持仓查询 RPC**：`GetSchedulePositions(schedule_id)` → live：position snapshot 按 magic==strategyMagic(schedule_id) 过滤（paper：paperEngine 订单按 run 过滤）。返回 `[{ticket, symbol, side, volume, open_price, current_price, pnl, open_time}]`。**归属校验**：schedule 属当前 uid（复用 GetSchedule 的 user 过滤）。
3. proto 改动 `buf generate`。
- **对抗**：mock 两策略（magic A/B）各一仓 → GetSchedulePositions(A) 只回 A 的仓（B 的混入 RED）；删 magic 过滤 → 混入 RED。

## §3 批次 B（前端，A 完工验收后）

1. **Tab1「我的策略」**：一张表，双流 join（§1）：
   `[策略名 | Symbol/TF | 账户 | 状态(绿●运行中/黄○已启用未运行/灰○停用) | 报价(stale标记) | 信号+最后时间 | PnL(绿红) | 错误(tooltip) | 操作(▶/⏹/✎/⚡)]`
   操作全部按 schedule_id（toggle/stop/edit/runNow 的 RPC 均已有）。
2. **行展开**（点行，非弹窗）：四个内嵌区——持仓（§2 RPC，实时刷新）/ 最近信号（最近20，时间线）/ 运行日志（schedule_run_logs）/ 配置摘要+编辑。
3. **Tab2「运行历史」**：保留现表。**删** Active Runs、Schedules 两 tab。
4. 空态 CTA「创建策略」；paper/live Tag 区分。

## 红队自审
- [ ] §0 ④ 汇率规则只适用 USD 账户——非 USD 账户先 fail-closed 拒（别算错）。
- [ ] §2 magic 映射：MT4 order 的 magic 在 OnOrderUpdate 里是否真的推送？先抓一次真实 OrderUpdate 验证字段存在再写过滤。
- [ ] §3 join 键：active_run_id 为空（未运行）→ 表里该行指标列显示 "-"，别崩。
- [ ] 展开区持仓刷新别轮询过频（10s 或 SSE，遵循 push-first）。
- [ ] proto 改动 buf generate；全部 commit + 部署 + 回填 registry（**勿再只留 handoff 不回填**）。
