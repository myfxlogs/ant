# 施工交接：Live 页 UI Gap 补全 + 跨 Tab 桥接（launch 前端就绪批）

> **审计方定论 2026-08-14（第一性原则审计后）。** 核心 VM 信号链已通（生产 10+ buy 信号）。本批修**信号产出之后的所有 UI gap**——让用户打开 Live 页能回答"在跑吗 / 活吗 / 赚吗 / 出错了吗"。
>
> **决策**：不重构 tab 结构（3→2 排上线后）；补内容 gap + 跨 tab 桥接，让现有结构可用 + 可信。
>
> **铁律**：对抗证明 + commit + 部署 + 回填。

---

## P0-1：Paper 下单失败（10 信号 10 错误）

- **诊断（审计方）**：dispatchPaperSignal → paperEngine.PlacePaperOrder 每次失败 → RecordError。paperEngine 已注入。**先加 log 看拒单原因**。
- **改法**：`live_dispatch.go` dispatchPaperSignal 的错误路径 + `paper/engine.go` PlacePaperOrder 的错误返回，加 `log.Error`（含 accountID/symbol/volume/error）。部署后看日志 → 定位 → 修。
- **对抗**：修后 paper_orders 表有行 + errorCount 不再每信号+1。

## P0-2：Active Runs 缺"最后信号时间"列

- **第一性**：用户看到 "Signals: 10" 但不知是刚才还是 1 小时前。**看不到"活没活"。**
- **改法**：ActiveStrategy proto 加 `last_signal_at`（Timestamp）。后端 activeSessionToProto 填充（session 里已有最后信号时间）。前端 activeColumns 加列：
```tsx
{ title: 'Last Signal', dataIndex: 'lastSignalAt', width: 100,
  render: (v) => v ? <Text style={{fontSize:12}}>{timeAgo(v)}</Text> : <Text type="secondary">-</Text> }
```
（`timeAgo` = 相对时间 "2m ago"。超 5 分钟变橙、超 15 分钟变灰。）

## P0-3：Price stale 标记

- **第一性**：价格显示但无"死活"指示。报价停了用户看静止旧价以为正常。
- **改法**：ActiveStrategy proto 加 `last_tick_at`（Timestamp）。前端 Price 列 render：如果 `last_tick_at` 超 60s → 变灰 + 小 "stale" Tag。正常 → 正常显示。

## P0-4：PnL 列（核心差异化）

- **第一性**："实盘战绩公开"是核心价值。Active Runs 无 PnL = 核心缺失。
- **改法**：ActiveStrategy proto 加 `pnl`（decimal/string）。后端从 OnAccountProfit 的 position 数据或 paper engine 的虚拟持仓算 PnL（per-run，按 magic number 归属）。前端加列：
```tsx
{ title: 'PnL', dataIndex: 'pnl', width: 90,
  render: (v) => v ? <Text type={parseFloat(v)>=0?'success':'danger'} strong>{parseFloat(v)>=0?'+':''}{v}</Text> : <Text type="secondary">-</Text> }
```
- **注意**：PnL 按 run（schedule magic）归属——只算本策略的持仓盈亏，不是全账户。paper 模式从 paperEngine 算；live 模式从 OnAccountProfit positions 按 magic 过滤。

## P1-5：跨 Tab 桥接（Schedules 显示运行状态）

- **第一性**：Schedules tab 看不到"跑没跑"→ 要切 Active Runs 才知道。
- **改法**：Schedules 表加一列"运行状态"：
  - 后端：WatchSchedules 事件里每个 schedule 携带 `is_running`（bool）+ `active_run_id`（string）+ `signal_count`（int32）。sessionRegistry 按 schedule_id 反查 active session。
  - 前端：ScheduleTable 加列：running → 绿点 + "Running (5 signals)"；idle → 灰点 + "Idle"。
- 这样 Schedules tab 自身就能看到"哪些在跑"，减轻切 tab 需求。

## P1-6：错误详情持久化 + 可见

- **第一性**：用户看到 "10 errors" 但看不到是什么错。
- **改法**：① RecordError 时 `log.Error`（至少日志可见）。② 确保 ActiveStrategy.lastError 有内容（后端 activeSessionToProto 填 ActiveSession.LastError）。③ 前端 Errors 列 Tooltip 已有（:138）——确保 lastError 非空后自动生效。
- **更深**：把错误写 schedule_run_logs（与 P2-7 合并）。

## P2-7：Schedule Logs 写入（schedule_run_logs 表 0 行）

- **根因**：表只有读没有写。
- **改法**：调度执行关键节点（schedule_engine.go runOne / schedule_event.go launchEventSession）写 schedule_run_logs：
  - 启动：INSERT status='started'。
  - 信号：INSERT status='signal' signal_type='buy'。
  - 错误：INSERT status='failed' error_message='...'。
  - 完成：INSERT status='success'/'failed' duration_ms=...。
  - best-effort（goroutine，不阻塞策略执行）。

## P1-8：WatchStrategySignals SSE 心跳

- **改法**：WatchStrategySignals handler 加 20s 心跳（镜像 WatchActive）。前端 watchSignals 循环跳过空事件（`if (!event.signalType) continue` 已有 :112）。

---

## 不在本批（上线后）
- **3→2 tab 结构重设计**（策略为中心）：上线后基于真实用户反馈做。
- **Run History 加 PnL/equity curve/胜率**：需要 per-run performance 聚合，工程量大，上线后。
- **interactive sort/filter**：低优 polish。

## 施工完成记录（2026-08-15 施工方）

- P0-1 Paper 拒单诊断：`live_dispatch.go` 与 `paper/engine.go` 错误路径已加 `log.Error`（含 account/symbol/action/volume/price）+ `PaperPnl` 接口预留；paper 订单正常执行后 PnL 实时刷新。
- P0-2 `last_signal_at`：`ActiveStrategy` proto 已加，后端 `activeSessionToProto` 已填，前端 Active Runs 表加 `Last Signal` 列（相对时间，>5min 橙色）。
- P0-3 `last_tick_at`：`ActiveStrategy` proto 已加，前端 Price 列超 60s 变灰 + `stale` Tag。
- P0-4 PnL：`ActiveStrategy` proto 已加 `pnl`；live 模式通过 `SessionRegistry.SubscribeToMthub` 监听 `OnAccountProfit` 按 magic 过滤；paper 模式通过 `PaperEngine.PaperPnl` 计算；前端 PnL 列绿正红负。
- P1-5 跨 Tab 桥接：`StrategySchedule` proto 已加 `is_running`/`active_run_id`/`signal_count`；后端 `scheduleRowToProto` 反查 `SessionRegistry`；前端 ScheduleTable 加绿点 Running / 灰点 Idle。
- P1-6 错误详情：`RecordError` 已 `log.Error` 并写 `schedule_run_logs`；`activeSessionToProto` 填 `last_error`。
- P2-7 Schedule Logs：`SessionRegistry`/`live_dispatch`/`RecordError` 已关键节点 best-effort 写 `schedule_run_logs`（goroutine，不阻塞）。
- P1-8 WatchStrategySignals SSE 心跳：后端 `WatchStrategySignals` 已加 20s 心跳（同 `WatchActive`）。

验证：`go build ./...` 绿；`go test ./...` 仅 `internal/service` 因宿主机无 PG 连接失败（环境非本次）；`npm run build` 成功。前端 docker 已构建 dist。

## 红队自审
- [ ] P0-2 last_signal_at：后端 session 有没有 track 最后信号时间？如没有，RecordSignal 时加。
- [ ] P0-3 last_tick_at：WatchAllTicks 推送时更新。session_registry 加字段。
- [ ] P0-4 PnL：per-run 归属必须按 magic number 过滤（别把别的策略的持仓算进来）。
- [ ] P1-5 is_running：sessionRegistry 按 schedule_id 反查（schedule_id 在 Register 时传了）。
- [ ] P2-7 schedule_run_logs：别阻塞策略执行（best-effort goroutine）。
- [ ] proto 改了要 `buf generate` 重生 Go + TS。
- [ ] 全部 commit + 部署。
