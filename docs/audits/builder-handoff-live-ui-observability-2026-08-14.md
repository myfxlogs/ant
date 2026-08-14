# 施工交接：Live UI 可观测性 + Paper 下单修复（一批收尾）

> **审计方定论 2026-08-14。** 核心 VM 信号链已通（生产 10+ buy 信号）。本批修**信号产出之后的 4 个 gap**：paper 下单失败 + 行跳动 + error 不可见 + Signal Log/Schedule Logs 空白。
>
> **铁律**：对抗证明 + commit + 部署 + 回填。

---

## P0-1：Paper 下单每次失败（10 信号 10 错误）

- **症状**：run 249b5278 产出 10 条 buy 信号（status=executed），但 errorCount=10（每信号 1 错误）。dispatchPaperSignal → paperEngine.PlacePaperOrder 失败 → RecordError。paperEngine 已注入（handlers_strategy.go:97）。
- **诊断方向**：读 `dispatchPaperSignal`（live_dispatch.go:281）+ paperEngine.PlacePaperOrder（paper/engine.go）。看 PlacePaperOrder 拒单原因（volume/symbol/account 校验？DB 写入失败？）。**关键**：RecordError 的错误内容没打日志——先加 log（把 paper 下单的错误内容 log.Error 出来），部署后看具体拒单原因，再修。
- **修法**：待定位（先加错误日志看原因，再修）。
- **对抗证明**：修后 paper 下单成功（paper_orders 表有行）+ errorCount 不再增长。

## P1-2：Active Runs 行跳动（Go map 随机序）

- **根因**：sessionRegistry.ListByUser/ListByAccount 遍历 Go map（`map[uuid.UUID]*ActiveSession`）→ 每次顺序随机 → SSE 推送列表顺序变 → 前端 Table 行跳。
- **修法（前端一行）**：`LiveStrategyPage.tsx` activeColumns 的 Table dataSource 加 sort：
```tsx
dataSource={[...activeStrategies].sort((a, b) => a.runId.localeCompare(b.runId))}
```
（或后端 handler sort 后转 proto——前端更快。）

## P1-3：error 详情不可见

- **根因**：ActiveSession.RecordError 只在内存存 lastError + errorCount++，**没持久化、没在 UI 显示具体错误内容**。用户看到"10 errors"但看不到是什么错。
- **修法**：① RecordError 时同时 log.Error（至少日志可见）；② ActiveStrategy proto 加 `last_error` 字段（已有？strategy_active_handlers.go:175 附近 `LastError`）→ 前端 Active Runs error 列加 Tooltip 显示 last_error。③ 理想：把错误写 schedule_run_logs（与 P2-4 合并）。
- **最小修**：确认 ActiveStrategy proto 有 last_error → 前端显示；RecordError 加 log。

## P2-4：Schedule Logs 空白（schedule_run_logs 表 0 行）

- **根因**：`schedule_run_logs` 表**从没有代码写入**（只有读：GetScheduleRunLogs / schedule_health_repo）。ScheduleLogsModal 读空表 → 空白。
- **修法**：调度执行时写日志——在 `runOne`（schedule_engine.go:385）或 `launchEventSession`（schedule_event.go:52）的关键节点（启动/信号/错误/完成）INSERT schedule_run_logs（status + signal_type + duration_ms + error_message）。这样 Schedule Logs 页有内容 + error 详情可见。

## P1-5：WatchStrategySignals SSE 524（Signal Log 空白）

- **根因**：WatchStrategySignals SSE 无心跳（同 LIVE-SSE-HEARTBEAT 修的 WatchActive，但这条流没修）→ 代理超时 524 → Signal Log 空白。
- **修法**：WatchStrategySignals handler 加心跳 ticker（镜像 WatchActive 的 20s 心跳）+ 前端跳过空事件。
- **位置**：`strategy_active_handlers.go` WatchStrategySignals handler。

---

## 红队自审
- [ ] P0-1：先加错误日志定位 paper 拒单原因，**别盲改**（看到具体错误再修）。
- [ ] P1-2：sort 用稳定字段（runId），别用会变的（如 signalCount）。
- [ ] P1-3：确认 proto ActiveStrategy 有 last_error 字段（可能已有，只需前端显示）。
- [ ] P2-4：schedule_run_logs 写入别阻塞策略执行（异步或 best-effort）。
- [ ] P1-5：心跳超时可注入测试。
- [ ] 全部 commit + 部署。
