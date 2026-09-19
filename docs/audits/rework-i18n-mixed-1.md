# I18N-MIXED-1 返修规格（复审修订记录 2026-09-19 R3-R6）

> 施工 9a641ea2 核心面合格（DICT+209 / 脚本泛化 / --locale / pre-commit 门禁 / Class B 清零），
> 但独立复审实证一项**功能性回退** + 一项白名单越界 + 一项孤儿 key。按下列 R3-R6 返修。
> 已译文本零改动，全部改动为 SSOT 补全 + 检查器豁免收窄 + 回归守卫。

## 背景实证

`npx tsx scripts/i18n-build.ts` 重生使 5 个 locale 的 base.ts 各丢 **43 个 leaf key**：

- `strategy.live.diag.*` x38 — DiagnosticsTab.tsx 34 处 t() 调用（含动态模板 lifecycle./execState./source.）
- `logs.signalType.*` x5 — logColumns.tsx:33 `t('logs.signalType.'+type)`

根因：这些 key 历史上是**生成文件私货**（从未进 textproto/map.json SSOT），重生成按 SSOT 产出即丢。
ja/zh-tw/vi/zh-cn 丢的是**真译文**（注文の真実 / 訂單真相 / 买入 / 買い / MUA）——跨 locale 回退。
en 部分有 defaultValue 兜底，但 diag 多数无 → 渲染原始 key 串。

## R3 — SSOT 吸收 43 keys（主返修项）

### R3a base section：strategy.live.diag.* x38

`base_map.json` fields 追加 38 条映射（key → 完整路径）：

```
"strategy_live_diag_state_warning": "strategy.live.diag.state.warning"
"strategy_live_diag_order_truth": "strategy.live.diag.orderTruth"
"strategy_live_diag_vm_orders_total": "strategy.live.diag.vmOrdersTotal"
"strategy_live_diag_broker_account_orders": "strategy.live.diag.brokerAccountOrders"
"strategy_live_diag_strategy_magic_orders": "strategy.live.diag.strategyMagicOrders"
"strategy_live_diag_pending_broker_orders": "strategy.live.diag.pendingBrokerOrders"
"strategy_live_diag_schedule_magic": "strategy.live.diag.scheduleMagic"
"strategy_live_diag_last_broker_ticket": "strategy.live.diag.lastBrokerTicket"
"strategy_live_diag_vm_broker_mismatch": "strategy.live.diag.vmBrokerMismatch"
"strategy_live_diag_execution": "strategy.live.diag.execution"
"strategy_live_diag_execution_state": "strategy.live.diag.executionState"
"strategy_live_diag_order_lifecycle": "strategy.live.diag.orderLifecycle"
"strategy_live_diag_freshness": "strategy.live.diag.freshness"
"strategy_live_diag_financial_source": "strategy.live.diag.financialSource"
"strategy_live_diag_financial_age": "strategy.live.diag.financialAge"
"strategy_live_diag_financial_fresh": "strategy.live.diag.financialFresh"
"strategy_live_diag_positions_source": "strategy.live.diag.positionsSource"
"strategy_live_diag_positions_age": "strategy.live.diag.positionsAge"
"strategy_live_diag_positions_fresh": "strategy.live.diag.positionsFresh"
"strategy_live_diag_fresh": "strategy.live.diag.fresh"
"strategy_live_diag_stale": "strategy.live.diag.stale"
"strategy_live_diag_na": "strategy.live.diag.na"
"strategy_live_diag_lifecycle_signal_generated": "strategy.live.diag.lifecycle.signal_generated"
"strategy_live_diag_lifecycle_order_submitting": "strategy.live.diag.lifecycle.order_submitting"
"strategy_live_diag_lifecycle_order_submitted": "strategy.live.diag.lifecycle.order_submitted"
"strategy_live_diag_lifecycle_order_confirmed": "strategy.live.diag.lifecycle.order_confirmed"
"strategy_live_diag_lifecycle_order_rejected": "strategy.live.diag.lifecycle.order_rejected"
"strategy_live_diag_lifecycle_order_outcome_unknown": "strategy.live.diag.lifecycle.order_outcome_unknown"
"strategy_live_diag_exec_state_idle": "strategy.live.diag.execState.idle"
"strategy_live_diag_exec_state_submitting": "strategy.live.diag.execState.submitting"
"strategy_live_diag_exec_state_accepted_unconfirmed": "strategy.live.diag.execState.accepted_unconfirmed"
"strategy_live_diag_exec_state_confirmed": "strategy.live.diag.execState.confirmed"
"strategy_live_diag_exec_state_deterministic_rejected": "strategy.live.diag.execState.deterministic_rejected"
"strategy_live_diag_exec_state_outcome_unknown": "strategy.live.diag.execState.outcome_unknown"
"strategy_live_diag_source_account_summary": "strategy.live.diag.source.account_summary"
"strategy_live_diag_source_profit_stream": "strategy.live.diag.source.profit_stream"
"strategy_live_diag_source_order_update": "strategy.live.diag.source.order_update"
"strategy_live_diag_source_position_snapshot": "strategy.live.diag.source.position_snapshot"
```

`base_{en,zh-cn,zh-tw,ja,vi}.textproto` 各追加 38 条值（逐 locale 见附录 A——
9a641ea2~1 历史值逐字恢复；zh-cn/zh-tw/ja/vi 全是真译文，en 为原文）。

### R3b logs section：logs.signalType.* x5

`logs_map.json` fields 追加 5 条（相对 logs 前缀路径）：

```
"signal_type_buy": "signalType.buy"
"signal_type_sell": "signalType.sell"
"signal_type_close": "signalType.close"
"signal_type_hold": "signalType.hold"
"signal_type_modify": "signalType.modify"
```

`logs_{en,zh-cn,zh-tw,ja,vi}.textproto` 各追加 5 条值（附录 A 末尾 5 条）。

### R3c 重生成 + 全量校验

```
npx tsx scripts/i18n-build.ts
# 对 5 locale 各断言 43 路径全部存在（附录 B keydiff 校验）
```

## R4 — 5 条用户面漏译补齐（检查器豁免收窄）

以下 5 个 en 值是**用户面文案**（workspace 引导 / 回测诊断建议 / 确认按钮），
不属于白名单技术词，9a641ea2 用 stopgap EXEMPT_PATTERNS 豁免——违反本债目标。

DICT 追加（exact-match；两处 tour 串在 base + strategy_workspace 两文件对均出现，
一条 DICT 全覆盖）：

```
'Ask AI to generate, optimize, or debug your strategy. Applied code appears in the editor instantly.': '让 AI 生成、优化或调试您的策略。应用的代码会即时出现在编辑器中。',
'Run backtests with configurable parameters. View equity curve, trade statistics, and risk metrics.': '使用可配置参数运行回测。查看净值曲线、交易统计和风险指标。',
'iCustom (custom indicator) is not supported — replace with a built-in indicator (iMA/iRSI/iMACD etc.) or implement the logic manually': '不支持 iCustom（自定义指标）——请改用内置指标（iMA/iRSI/iMACD 等）或手动实现该逻辑',
'DLL imports are not supported — remove external DLL calls and use built-in MQL functions': '不支持 DLL 导入——请移除外部 DLL 调用，改用内置 MQL 函数',
'Acknowledge as intentional — hide this warning': '确认为有意为之——隐藏此警告',
```

同时**删除** i18n-check.ts 中对应 5 条 EXEMPT_PATTERNS
（iCustom / DLL imports / Acknowledge / Ask AI / Run backtests）——
翻译落地后豁免失去必要，留着会掩盖未来同类回归。
保留：`/^Example:\n/`（AI prompt 内部串，附录 C 合法）及其余技术词白名单。

跑 `npx tsx scripts/i18n-translate-zh-cn.ts` + `i18n-build.ts` 落地。

## R5 — close_position 孤儿三 key 清零 3 warnings

`strategy_live_close_position_{title,desc,confirm}` 仅存在于 zh-cn textproto（en 无对应），
且 `strategy.live.closePosition` 路径**零消费方**（现存 `trading.closePosition.*` 是另一路径，已核）。
判定死 key/drift：删除 `base_zh-cn.textproto` 该三行 → strict warnings 3→0。
若复核发现消费方存在 → 改道：en textproto + map 补齐对应 key。

## R6 — 回归守卫测试（对抗证明载体）

新建 `frontend/src/test/i18n-key-presence.test.ts`（vitest；先例 schedule-log-message.test.tsx）：

- import 5 locale 的合并资源树（或各 base.ts + logs.ts 合并）
- 断言附录 A 43 个恢复路径 x 5 locale 全部解析为非空字符串
- 断言 `strategy_gen.execFeedback.placeholder` zh-cn 存在
- 断言 5 条 R4 译文值在 zh-cn 资源中生效

判别性：从 textproto 删任一 43-key → 重生成 → 测试 RED（即 R3 对抗证明）。

## 门禁

- `npx tsx scripts/i18n-check.ts --strict --locale zh-cn` → **0 errors 0 warnings**
- `cd frontend && npx tsc --noEmit` 零错
- 新增 vitest 全绿；既有前端测试不回归
- `git diff --check` 净
- 附录 B keydiff 对 5 locale 输出 0 lost

勿部署，完成报证据等复审。

## 附录 A — 43 keys x 5 locales 历史值（9a641ea2~1 逐字）

key → textproto key 规则：`strategy.live.diag.<leaf>` → `strategy_live_diag_<snake(leaf)>`（嵌套 lifecycle./execState./source./state. 的子键按 `strategy_live_diag_<group>_<snake(sub)>`）；`logs.signalType.<leaf>` → `signal_type_<snake(leaf)>`，进 logs section（`logs_map.json` 用相对路径 `signalType.<leaf>`）。

```json
{
 "en": {
  "strategy.live.diag.state.warning": "Warning",
  "strategy.live.diag.orderTruth": "Order Truth",
  "strategy.live.diag.vmOrdersTotal": "VM OrdersTotal",
  "strategy.live.diag.brokerAccountOrders": "Broker Account Orders",
  "strategy.live.diag.strategyMagicOrders": "Strategy Magic Orders",
  "strategy.live.diag.pendingBrokerOrders": "Pending Broker Orders",
  "strategy.live.diag.scheduleMagic": "Schedule Magic",
  "strategy.live.diag.lastBrokerTicket": "Last Broker Ticket",
  "strategy.live.diag.vmBrokerMismatch": "VM count differs from broker count",
  "strategy.live.diag.execution": "Execution",
  "strategy.live.diag.executionState": "Execution State",
  "strategy.live.diag.orderLifecycle": "Order Lifecycle",
  "strategy.live.diag.freshness": "Freshness",
  "strategy.live.diag.financialSource": "Financial Source",
  "strategy.live.diag.financialAge": "Financial Age",
  "strategy.live.diag.financialFresh": "Financial Fresh",
  "strategy.live.diag.positionsSource": "Positions Source",
  "strategy.live.diag.positionsAge": "Positions Age",
  "strategy.live.diag.positionsFresh": "Positions Fresh",
  "strategy.live.diag.fresh": "Fresh",
  "strategy.live.diag.stale": "Stale",
  "strategy.live.diag.na": "N/A",
  "strategy.live.diag.lifecycle.signal_generated": "Signal Generated",
  "strategy.live.diag.lifecycle.order_submitting": "Order Submitting",
  "strategy.live.diag.lifecycle.order_submitted": "Order Submitted",
  "strategy.live.diag.lifecycle.order_confirmed": "Order Confirmed",
  "strategy.live.diag.lifecycle.order_rejected": "Order Rejected",
  "strategy.live.diag.lifecycle.order_outcome_unknown": "Outcome Unknown",
  "strategy.live.diag.execState.idle": "Idle",
  "strategy.live.diag.execState.submitting": "Submitting",
  "strategy.live.diag.execState.accepted_unconfirmed": "Accepted (Unconfirmed)",
  "strategy.live.diag.execState.confirmed": "Confirmed",
  "strategy.live.diag.execState.deterministic_rejected": "Rejected",
  "strategy.live.diag.execState.outcome_unknown": "Unknown",
  "strategy.live.diag.source.account_summary": "Account Summary",
  "strategy.live.diag.source.profit_stream": "Profit Stream",
  "strategy.live.diag.source.order_update": "Order Update",
  "strategy.live.diag.source.position_snapshot": "Position Snapshot",
  "logs.signalType.buy": "BUY",
  "logs.signalType.sell": "SELL",
  "logs.signalType.close": "CLOSE",
  "logs.signalType.hold": "HOLD",
  "logs.signalType.modify": "MODIFY"
 },
 "zh-cn": {
  "strategy.live.diag.state.warning": "警告",
  "strategy.live.diag.orderTruth": "订单真相",
  "strategy.live.diag.vmOrdersTotal": "VM订单总数",
  "strategy.live.diag.brokerAccountOrders": "经纪商账户订单",
  "strategy.live.diag.strategyMagicOrders": "策略Magic订单",
  "strategy.live.diag.pendingBrokerOrders": "经纪商挂单",
  "strategy.live.diag.scheduleMagic": "调度Magic",
  "strategy.live.diag.lastBrokerTicket": "最近经纪商订单号",
  "strategy.live.diag.vmBrokerMismatch": "VM订单数与经纪商不一致",
  "strategy.live.diag.execution": "执行状态",
  "strategy.live.diag.executionState": "执行状态",
  "strategy.live.diag.orderLifecycle": "订单生命周期",
  "strategy.live.diag.freshness": "数据新鲜度",
  "strategy.live.diag.financialSource": "金融数据来源",
  "strategy.live.diag.financialAge": "金融数据年龄",
  "strategy.live.diag.financialFresh": "金融数据新鲜",
  "strategy.live.diag.positionsSource": "持仓数据来源",
  "strategy.live.diag.positionsAge": "持仓数据年龄",
  "strategy.live.diag.positionsFresh": "持仓数据新鲜",
  "strategy.live.diag.fresh": "新鲜",
  "strategy.live.diag.stale": "过期",
  "strategy.live.diag.na": "无",
  "strategy.live.diag.lifecycle.signal_generated": "信号已生成",
  "strategy.live.diag.lifecycle.order_submitting": "订单提交中",
  "strategy.live.diag.lifecycle.order_submitted": "订单已提交",
  "strategy.live.diag.lifecycle.order_confirmed": "订单已确认",
  "strategy.live.diag.lifecycle.order_rejected": "订单已拒绝",
  "strategy.live.diag.lifecycle.order_outcome_unknown": "订单结果未知",
  "strategy.live.diag.execState.idle": "空闲",
  "strategy.live.diag.execState.submitting": "提交中",
  "strategy.live.diag.execState.accepted_unconfirmed": "已接受（未确认）",
  "strategy.live.diag.execState.confirmed": "已确认",
  "strategy.live.diag.execState.deterministic_rejected": "已拒绝",
  "strategy.live.diag.execState.outcome_unknown": "未知",
  "strategy.live.diag.source.account_summary": "账户摘要",
  "strategy.live.diag.source.profit_stream": "利润流",
  "strategy.live.diag.source.order_update": "订单更新",
  "strategy.live.diag.source.position_snapshot": "持仓快照",
  "logs.signalType.buy": "买入",
  "logs.signalType.sell": "卖出",
  "logs.signalType.close": "平仓",
  "logs.signalType.hold": "持有",
  "logs.signalType.modify": "修改"
 },
 "zh-tw": {
  "strategy.live.diag.state.warning": "警告",
  "strategy.live.diag.orderTruth": "訂單真相",
  "strategy.live.diag.vmOrdersTotal": "VM訂單總數",
  "strategy.live.diag.brokerAccountOrders": "券商帳戶訂單",
  "strategy.live.diag.strategyMagicOrders": "策略Magic訂單",
  "strategy.live.diag.pendingBrokerOrders": "待處理券商訂單",
  "strategy.live.diag.scheduleMagic": "排程Magic",
  "strategy.live.diag.lastBrokerTicket": "最後券商票號",
  "strategy.live.diag.vmBrokerMismatch": "VM數量與券商數量不一致",
  "strategy.live.diag.execution": "執行",
  "strategy.live.diag.executionState": "執行狀態",
  "strategy.live.diag.orderLifecycle": "訂單生命週期",
  "strategy.live.diag.freshness": "新鮮度",
  "strategy.live.diag.financialSource": "金融來源",
  "strategy.live.diag.financialAge": "金融年齡",
  "strategy.live.diag.financialFresh": "金融新鮮度",
  "strategy.live.diag.positionsSource": "持倉來源",
  "strategy.live.diag.positionsAge": "持倉年齡",
  "strategy.live.diag.positionsFresh": "持倉新鮮度",
  "strategy.live.diag.fresh": "新鮮",
  "strategy.live.diag.stale": "滯後",
  "strategy.live.diag.na": "無",
  "strategy.live.diag.lifecycle.signal_generated": "訊號已產生",
  "strategy.live.diag.lifecycle.order_submitting": "訂單提交中",
  "strategy.live.diag.lifecycle.order_submitted": "訂單已提交",
  "strategy.live.diag.lifecycle.order_confirmed": "訂單已確認",
  "strategy.live.diag.lifecycle.order_rejected": "訂單已拒絕",
  "strategy.live.diag.lifecycle.order_outcome_unknown": "結果未知",
  "strategy.live.diag.execState.idle": "閒置",
  "strategy.live.diag.execState.submitting": "提交中",
  "strategy.live.diag.execState.accepted_unconfirmed": "已接受（未確認）",
  "strategy.live.diag.execState.confirmed": "已確認",
  "strategy.live.diag.execState.deterministic_rejected": "已拒絕",
  "strategy.live.diag.execState.outcome_unknown": "未知",
  "strategy.live.diag.source.account_summary": "帳戶摘要",
  "strategy.live.diag.source.profit_stream": "利潤流",
  "strategy.live.diag.source.order_update": "訂單更新",
  "strategy.live.diag.source.position_snapshot": "持倉快照",
  "logs.signalType.buy": "買入",
  "logs.signalType.sell": "賣出",
  "logs.signalType.close": "平倉",
  "logs.signalType.hold": "持有",
  "logs.signalType.modify": "修改"
 },
 "ja": {
  "strategy.live.diag.state.warning": "警告",
  "strategy.live.diag.orderTruth": "注文の真実",
  "strategy.live.diag.vmOrdersTotal": "VM注文総数",
  "strategy.live.diag.brokerAccountOrders": "ブローカーアカウント注文",
  "strategy.live.diag.strategyMagicOrders": "ストラテジーマジック注文",
  "strategy.live.diag.pendingBrokerOrders": "保留中ブローカー注文",
  "strategy.live.diag.scheduleMagic": "スケジュールマジック",
  "strategy.live.diag.lastBrokerTicket": "最終ブローカーチケット",
  "strategy.live.diag.vmBrokerMismatch": "VM数とブローカー数が異なります",
  "strategy.live.diag.execution": "実行",
  "strategy.live.diag.executionState": "実行状態",
  "strategy.live.diag.orderLifecycle": "注文ライフサイクル",
  "strategy.live.diag.freshness": "鮮度",
  "strategy.live.diag.financialSource": "金融ソース",
  "strategy.live.diag.financialAge": "金融経過時間",
  "strategy.live.diag.financialFresh": "金融鮮度",
  "strategy.live.diag.positionsSource": "ポジションソース",
  "strategy.live.diag.positionsAge": "ポジション経過時間",
  "strategy.live.diag.positionsFresh": "ポジション鮮度",
  "strategy.live.diag.fresh": "新鮮",
  "strategy.live.diag.stale": "遅延",
  "strategy.live.diag.na": "N/A",
  "strategy.live.diag.lifecycle.signal_generated": "シグナル生成",
  "strategy.live.diag.lifecycle.order_submitting": "注文送信中",
  "strategy.live.diag.lifecycle.order_submitted": "注文送信済み",
  "strategy.live.diag.lifecycle.order_confirmed": "注文確認済み",
  "strategy.live.diag.lifecycle.order_rejected": "注文拒否",
  "strategy.live.diag.lifecycle.order_outcome_unknown": "結果不明",
  "strategy.live.diag.execState.idle": "アイドル",
  "strategy.live.diag.execState.submitting": "送信中",
  "strategy.live.diag.execState.accepted_unconfirmed": "承認（未確認）",
  "strategy.live.diag.execState.confirmed": "確認済み",
  "strategy.live.diag.execState.deterministic_rejected": "拒否",
  "strategy.live.diag.execState.outcome_unknown": "不明",
  "strategy.live.diag.source.account_summary": "口座サマリー",
  "strategy.live.diag.source.profit_stream": "プロフィットストリーム",
  "strategy.live.diag.source.order_update": "注文更新",
  "strategy.live.diag.source.position_snapshot": "ポジションスナップショット",
  "logs.signalType.buy": "買い",
  "logs.signalType.sell": "売り",
  "logs.signalType.close": "決済",
  "logs.signalType.hold": "ホールド",
  "logs.signalType.modify": "修正"
 },
 "vi": {
  "strategy.live.diag.state.warning": "Cảnh báo",
  "strategy.live.diag.orderTruth": "Thực tế Lệnh",
  "strategy.live.diag.vmOrdersTotal": "Tổng lệnh VM",
  "strategy.live.diag.brokerAccountOrders": "Lệnh tài khoản broker",
  "strategy.live.diag.strategyMagicOrders": "Lệnh Magic chiến lược",
  "strategy.live.diag.pendingBrokerOrders": "Lệnh broker chờ xử lý",
  "strategy.live.diag.scheduleMagic": "Magic lịch trình",
  "strategy.live.diag.lastBrokerTicket": "Ticket broker cuối",
  "strategy.live.diag.vmBrokerMismatch": "Số lượng VM khác broker",
  "strategy.live.diag.execution": "Thực thi",
  "strategy.live.diag.executionState": "Trạng thái thực thi",
  "strategy.live.diag.orderLifecycle": "Vòng đời lệnh",
  "strategy.live.diag.freshness": "Độ tươi",
  "strategy.live.diag.financialSource": "Nguồn tài chính",
  "strategy.live.diag.financialAge": "Tuổi tài chính",
  "strategy.live.diag.financialFresh": "Tươi tài chính",
  "strategy.live.diag.positionsSource": "Nguồn vị thế",
  "strategy.live.diag.positionsAge": "Tuổi vị thế",
  "strategy.live.diag.positionsFresh": "Tươi vị thế",
  "strategy.live.diag.fresh": "Tươi",
  "strategy.live.diag.stale": "Lỗi thời",
  "strategy.live.diag.na": "N/A",
  "strategy.live.diag.lifecycle.signal_generated": "Tín hiệu đã tạo",
  "strategy.live.diag.lifecycle.order_submitting": "Đang gửi lệnh",
  "strategy.live.diag.lifecycle.order_submitted": "Lệnh đã gửi",
  "strategy.live.diag.lifecycle.order_confirmed": "Lệnh đã xác nhận",
  "strategy.live.diag.lifecycle.order_rejected": "Lệnh bị từ chối",
  "strategy.live.diag.lifecycle.order_outcome_unknown": "Kết quả không rõ",
  "strategy.live.diag.execState.idle": "Chờ",
  "strategy.live.diag.execState.submitting": "Đang gửi",
  "strategy.live.diag.execState.accepted_unconfirmed": "Đã nhận (chưa xác nhận)",
  "strategy.live.diag.execState.confirmed": "Đã xác nhận",
  "strategy.live.diag.execState.deterministic_rejected": "Bị từ chối",
  "strategy.live.diag.execState.outcome_unknown": "Không rõ",
  "strategy.live.diag.source.account_summary": "Tóm tắt tài khoản",
  "strategy.live.diag.source.profit_stream": "Luồng lợi nhuận",
  "strategy.live.diag.source.order_update": "Cập nhật lệnh",
  "strategy.live.diag.source.position_snapshot": "Ảnh chụp vị thế",
  "logs.signalType.buy": "MUA",
  "logs.signalType.sell": "BÁN",
  "logs.signalType.close": "ĐÓNG",
  "logs.signalType.hold": "GIỮ",
  "logs.signalType.modify": "SỬA"
 }
}
```

## 附录 B — keydiff 校验要点

对每个 locale 的 base.ts + logs.ts（或合并资源）断言附录 A 全部 43 路径存在。
复审方参考实现：展开 git show 9a641ea2~1 与各 locale 现文件，flatten 后差集比对。
