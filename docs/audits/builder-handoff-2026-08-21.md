施工 `LIVE-ORDER-REENTRY-1`，严格按 `docs/audits/tech-debt-registry.md` 与 `docs/audits/handover-audit-plan.md` 的最新返工要求执行，完成后回填三层文档并停在 `⚠️待Claude复审`，禁止部署/扩 scope。
施工 `LIVE-ORDER-REENTRY-1` 第4轮，严格按 `docs/audits/tech-debt-registry.md` 与 `docs/audits/handover-audit-plan.md` 的最新返工要求执行，完成后回填三层文档并停在 `⚠️待Claude复审`，禁止部署/扩 scope。

## 待施工：LIVE-MQL-ORDER-CONTEXT-1

施工 `LIVE-MQL-ORDER-CONTEXT-1`，严格按本节要求执行，完成后按 SOP 回填 registry/handover/AGENTS 并停在待复审，禁止修改异步执行 barrier、诊断 UI 或 FE-POSITIONS-524-PUSH，禁止部署/扩 scope。

范围：只修 broker order snapshot → LivePosition/LivePendingOrder → sdk.Position/sdk.PendingOrder → MQL OrdersTotal/OrderSelect/OrderMagicNumber 的完整字段语义。

要求：
1. LivePosition proto 补齐 symbol、magic_number、order_type、SL、TP、profit、swap、commission、comment、open_time。
2. 增加 LivePendingOrder；不得把 buy_limit、sell_stop 等挂单伪装成 market position。
3. Runner.UpdateLiveState 同时接收 positions 和 pendingOrders。
4. Harness broker 的 Positions(0) 只返回 market positions，Orders(0) 只返回 pending orders。
5. OrdersTotal 保持账户级语义：len(positions)+len(pendingOrders)。
6. OrderMagicNumber 返回 broker 原始 Magic。
7. 禁止默认按 schedule Magic 过滤 OrdersTotal。

对抗验收：
- buy/sell/buy_limit/sell_stop → Positions=2、Orders=2、OrdersTotal=4。
- magic=1699507621 必须端到端到达 OrderMagicNumber。
- 删除任一层 magic mapping，测试必须 RED。

## 待施工：LIVE-DIAG-TRUTH-1

施工 `LIVE-DIAG-TRUTH-1`，严格按本节要求执行，完成后按 SOP 回填 registry/handover/i18n/前端测试/build 并停在待复审，禁止修改交易执行语义、轮询或新增第二条 SSE，禁止部署/扩 scope。

范围：只修实盘诊断真实性。

诊断页必须区分：
- VM OrdersTotal（账户级、最后一次 VM 实际看到的值）。
- Broker account orders。
- Strategy magic orders。
- Pending broker orders。
- Execution in flight / outcome unknown。
- Schedule magic。
- financial source/captured_at/age/fresh。
- positions source/captured_at/age/fresh。
- 最近 broker ticket。
- 最近订单生命周期状态。

规则：
1. signal_generated 不能显示成成交。
2. 生命周期必须区分 signal_generated、order_submitting、order_submitted、order_confirmed/filled、order_rejected、order_outcome_unknown。
3. VM count 与 broker count 不一致，或 positions stale/outcome unknown 时，状态必须红色或 warning，不能绿色 active。
4. sessionDiag 的 runtime state 更新必须与 indicator ring 分离；RecordIndicators 在 values 为空时不得阻断 OrdersTotal 更新。
5. 前端只渲染后端计算结果，不自行推断权威状态。
6. 使用现有 ActiveStrategy/WatchActive SSE，不新建轮询或第二条 SSE。
7. mixed magic 测试：broker account=3、target magic=1、VM=0，UI 必须明确展示三者，不混为一个 OrdersTotal。
8. 100 次 watch 更新不得新增 RPC polling。

## LIVE-MQL-ORDER-CONTEXT-1 返工提示词

施工 `LIVE-MQL-ORDER-CONTEXT-1` 返工，严格按 `docs/audits/tech-debt-registry.md` 与 `docs/audits/builder-handoff-2026-08-21.md` 的最新复审要求执行，完成后回填 registry/handover/AGENTS 并停在 `⚠️待Claude复审`，禁止修改 `LIVE-ORDER-REENTRY-1`、诊断 UI、`FE-POSITIONS-524-PUSH`，禁止部署/扩 scope。

## LIVE-DIAG-TRUTH-1 施工提示词

施工 `LIVE-DIAG-TRUTH-1` 最后一轮测试补强，严格按 `docs/audits/tech-debt-registry.md` 最新审计复审要求执行，完成后回填 registry/handover/i18n/前端测试/build 并停在待独立审计复审，禁止修改交易执行语义、轮询或新增第二条 SSE，禁止部署/扩 scope。
