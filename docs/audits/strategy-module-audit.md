# 策略模块 — 全线审计报告

> **审计日期**：2026-07-07
> **审计对象**：策略模块（后端 `internal/connect/strategy/` 48 文件 + `service`/`repository` 相关 + 前端 `pages/strategy/`）
> **审计人**：Cascade
> **对照**：`rd.md` 声称「策略模块 — 审计完成，删 2 个死代码，路由/管线确认正常」
> **验证**：`go build ./...` ⚠️（当前因账号模块进行中的改动 `internal/connect/user/auth_token.go` 报 `encoding/json` unused，**与策略模块无关**）；`check-file-lines --strict` ✅ 0 errors

---

## 0. 审计结论概要

策略模块在**平台协议合规上表现良好**（无 REST、无 WebSocket、无轮询、`float64` 多用于统计而非价格）。
但审计发现 **1 整块死代码（含 1 个潜伏 BUG）**、**1 处价格精度反模式**，以及 **deepseek 的"审计完成"声明被证伪**——其实际工作仅 1 个 commit 删了 2 个前端文件，遗漏了本报告 §2.1 的整块死代码。

> **范围说明**：本模块体量为账号模块的约 10 倍。本次审计以「合规扫描 + 死代码/BUG 定向狩猎 + 管线/边界抽查」为主，未做逐行穷举与运行时行为测试。下列结论均有代码位置佐证。

---

## 1. deepseek 实际工作核实

`rd.md` 声称「审计完成，删 2 个死代码，路由/管线确认正常」。核实 git 记录：

- 唯一相关 commit：`9fbe4eb0 chore: delete dead code in strategy module`
  - 删除 `components/workspace/BacktestHistoryModal.tsx`（被 Drawer 取代）
  - 删除 `components/workspace/MiniPositionsTable.tsx`（无引用）
  - `useStrategyWorkspaceState`（282 行）保留（理由：低于硬红线，拆分收益<成本）

**结论**：删除的 2 个文件属实、判断正确。但「审计完成」被**夸大**——这是一次浅层清理，**未触及后端**，遗漏了 §2.1 的整块死代码写入路径。「路由/管线确认正常」缺乏可核查的依据。

---

## 2. 发现的问题

### 2.1 【死代码 + 潜伏 BUG】`strategy_execution_logs` 整条写入路径未接线

**死代码链**（均无 live 调用者，仅有定义 + 测试）：

| 符号 | 位置 |
|------|------|
| `NewStrategyExecutionLog` | `model/logs.go:99` |
| `LogService.LogExecution` | `service/log_service.go:30` |
| `LogService.UpdateExecution` | `service/log_service.go:34` |
| `LogRepository.CreateExecutionLog` | `repository/execution_log_repository.go:13` |
| `LogRepository.UpdateExecutionLog` | `repository/execution_log_repository.go:38` |

**证据**：全仓 `grep` `NewStrategyExecutionLog` / `.LogExecution(` / `.UpdateExecution(` / `CreateExecutionLog(` 在 live 代码中**零调用**（只有定义、`log_service` 转发、测试与 `DELETE` 清理）。

**连带影响（死表 / 潜在破功能）**：读取侧仍在用——
- RPC `LogService.GetExecutionLogs`（`connect/system/log_handler.go:151`）
- `schedule_health_repo` 的 schedule 统计（`GetScheduleStats` 等，读 `strategy_execution_logs`）

由于**从无代码写入该表**，这些读取恒返回空/零。需判定：执行日志是「待接线的功能」还是「已废弃」。

**潜伏 BUG（`execution_log_repository.go:35,52`）**：
```go
_, err := r.db.Exec(ctx, query, ...)
return fmt.Errorf("create execution log: %w", err)   // err==nil 时仍返回非 nil！
```
`CreateExecutionLog` / `UpdateExecutionLog` **无论成功与否都返回非 nil error**（`fmt.Errorf(..., nil)` 仍非 nil）。当前因写入路径为死代码而未爆发；一旦接线，每次调用都会误报失败。

**修复方向（二选一，需决策）**：
- **方案 A（接线）**：若执行日志是需要的功能，在 live 调度/派发路径（`live_dispatch.go` / `schedule_engine` 触发执行处）调用 `LogService.LogExecution` / `UpdateExecution`，并**同时修复** `fmt.Errorf` 恒错误 BUG（改为 `if err != nil { return fmt.Errorf(...) }; return nil`）。
- **方案 B（删除）**：若已废弃，删除上表整条写入链 + `GetExecutionLogs` RPC + 前端消费页，改用现存的 `schedule_run_logs`（`GetScheduleRunLogs`，已正常写入）作为唯一执行历史来源。

### 2.2 【精度反模式】回测品种信息 decimal→float64→string 往返

`connect/strategy/backtest_execution.go:115-138` —
```go
lotMin, _  := p.LotMin.Float64()    // decimal → float64（丢精度）
tickValue, _ := p.PointValue.Float64()
...
info.TickValue = strconv.FormatFloat(tickValue, 'f', -1, 64)  // float64 → string
```
`p.LotMin/LotMax/LotStep/LotSize/PointValue` 本是 `decimal.Decimal`，proto `SymbolInfo` 目标字段又是 **string**（本可无损）。中间却绕道 `float64`，对喂入回测 PnL/仓位计算的**合约规格与 tick 价值造成精度损失**，违反「价格计算禁用 float64」。

**修复**：直接 `p.LotMin.String()` / `p.PointValue.String()`，去掉 `float64` 中转。`point := math.Pow(10, -digits)` 亦建议用 `decimal` 幂或直接由 `Digits` 在下游构造。

### 2.3 【合规-注意】价格数据 JSON 序列化（当前落在 §2.1 死路径上）

`execution_log_repository.go:24,27`：
```go
klineData, _ = json.Marshal(log.KlineData)      // OHLC 价格 → JSON
strategyParams, _ = json.Marshal(log.StrategyParams)
```
将 K 线（含价格）序列化为 JSON 存 `kline_data` 列，触及「❌ JSON 持久化」与 float64 精度双重问题。当前因 §2.1 写入路径为死代码而**未实际执行**；若走方案 A 接线，必须先改造（价格用 proto/decimal，勿用 `json.Marshal`）。

---

## 3. 合规扫描（正面结论）

| 维度 | 结论 |
|------|------|
| REST 端点 | ✅ 策略后端无裸 HTTP handler（`connect/strategy` 无 `http.HandleFunc`/`http.Error`） |
| WebSocket | ✅ 无 |
| 前端轮询 | ✅ 无 `setInterval` 轮询；`setTimeout` 均为 SSE 重连退避（`LiveSchedulesTab` / `useLibrarySchedules`），符合 push-first |
| 后端 `time.Ticker` | ✅ 均为 `pgListen.Listen` 的「漏通知安全网」或调度器固有定时（`backtest_worker` / `strategy_experiment_handler` / `strategy_schedules` / `strategy_backtest_crud` / `backtest_execution`），push 为主、ticker 兜底，可接受 |
| `float64` | ✅ 大多用于统计（Spearman/Sharpe/评分）、日志、Prometheus buckets——非价格计算；例外见 §2.2 |
| AI 模块 `json.Marshal` | ✅ 用于外部 LLM API（OpenAI 协议强制 JSON）与解析 LLM 响应，属外部契约，合理 |

---

## 4. 架构观察（非缺陷，供参考）

- **策略参数以 JSON 字符串建模**：多个 proto 有 `parameters_json` string 字段（`EngineValidateResponse` / `ValidateStrategyResponse` / `ValidateStrategyExtendedResponse`），`code_assist_handler.go` 用 `json.Marshal` 生成。这是既有数据模型选择——参数本可用 proto `repeated` 消息表达。属长期架构债，非本次施工引入，**暂不列为必修项**，如后续重构参数体系可一并处理。
- **文件行数软警告**（🟡/🟢，不阻断）：`harness_template_live.go` 371、`strategy_execution_handler.go` 336、`schedule_engine.go` 315。均在硬红线内，语义内聚，暂不强制拆分。

---

## 5. 决策结论（需求方已拍板）

| 议题 | 决策 |
|------|------|
| 执行日志（§2.1） | **方案 B —— 删除整条死链**（写入+读取+RPC+前端 Tab），schedule 健康统计改指向已正常写入的 `schedule_run_logs` |
| 精度反模式（§2.2） | **改 `decimal.String()` 直通**（去掉 float64 中转） |

**方案 B 已核实的边界事实**（deepseek 据此实施）：
- 前端 `pages/logs/LogManagement.tsx:50` 存在「执行日志」Tab，调用 `logApi.getExecutionLogs` → 读恒空的 `strategy_execution_logs`。**方案 B 将移除该 Tab**（执行历史已由 "orders" Tab 的 `order_history` 与 schedule 日志覆盖，删除不丢功能）。
- `schedule_health_repo` 的统计（如 `GetScheduleStats`）读 `strategy_execution_logs`（恒空）→ **改指向 `schedule_run_logs`**（`log_repository.go:33` 确认为已写入来源）。
- 删除后 §2.1b 的 `fmt.Errorf` 恒错误 BUG 随函数一并消失，无需单独修。

**职责分工**：以下为交给 deepseek 的实现任务书；Cascade 负责设计与验收，不参与编码。

---

## 6. 实现任务书（deepseek 执行）

> 遵守 `AGENTS.md`：proto only、禁 REST/JSON 数据交换、`decimal.Decimal`、文件行数红线、部署走 `docker compose build backend`。
> 迁移新增 `.up.sql` 会随 build 自动执行，需成对提供 `.down.sql`。

### 任务 S-A：删除执行日志死链（方案 B）

**后端**
1. **删除** `repository/execution_log_repository.go` 整个文件（`CreateExecutionLog` / `UpdateExecutionLog` / `GetExecutionLogs` / `buildExecLogFilters`）。
2. `service/log_service.go` — 删除 `LogExecution` / `UpdateExecution` / `GetExecutionLogs` 三个方法。
3. `model/logs.go` — 删除 `StrategyExecutionLog` 结构体 + `NewStrategyExecutionLog`（先确认无其他引用）。
4. `connect/system/log_handler.go` — 删除 `GetExecutionLogs` handler（`:151` 一带）。
5. proto（`log.proto` / `log_execution.proto`）— 移除 `GetExecutionLogs` RPC 及 `GetExecutionLogsRequest/Response`、`ExecutionLog` message（或标 `deprecated`），重新生成 `gen/`。
6. `schedule_health_repo.go` — 把读 `strategy_execution_logs` 的统计改查 `schedule_run_logs`（字段对齐 status='success'/'failed'）；若与既有 schedule_run_logs 统计重复则直接删除该统计。
7. 新迁移 `191_drop_strategy_execution_logs.up.sql` / `.down.sql`：
   ```sql
   -- up
   DROP TABLE IF EXISTS strategy_execution_logs;
   -- down: 重建表结构（含 187 的 fk_exec_logs_account ON DELETE CASCADE）
   ```
   > 注：187 的 `fk_exec_logs_account` 随表消失，属预期。

**前端**
8. `pages/logs/LogManagement.tsx` — 删除 `'execution'` Tab 及其分支（`:50`）。
9. `client/log.ts` — 删除 `getExecutionLogs`。
10. i18n `logs.executionLogs`（i18n 豁免，可留可删，建议删以保持整洁）。

**验收标准 S-A**：
- 全仓 `grep` `strategy_execution_logs` / `GetExecutionLogs` / `CreateExecutionLog` / `NewStrategyExecutionLog` 无残留（迁移 down 除外）。
- schedule 健康统计改自 `schedule_run_logs`，数值非恒零（有真实运行时可见）。
- LogManagement 无空的「执行日志」Tab；"orders"/"connection"/"operation" Tab 不受影响。
- 191 有 up+down；`go build ./...` 与 `npx tsc --noEmit` 通过。

### 任务 S-B：回测品种规格精度直通（§2.2）

`connect/strategy/backtest_execution.go:115-138`：
- 删除 `p.LotMin.Float64()` / `LotMax` / `LotStep` / `LotSize` / `PointValue.Float64()` 等 float64 中转。
- 直接 `p.LotMin.String()` 等赋给 `SymbolInfo` 的 string 字段（保 decimal 精度）。
- `point`：改由 `decimal` 计算（如 `decimal.New(1, int32(-p.Digits))`）后 `.String()`，去掉 `math.Pow`+`float64`；或由下游据 `Digits` 构造，避免任何 float64。

**验收标准 S-B**：`backtest_execution.go` 无 `.Float64()` / `math.Pow` 价格中转；回测合约规格/tick 价值保 decimal 精度；`go build` 通过。

### 全局验收（deepseek 自检 + Cascade 复验）

```bash
go build ./...                                          # 通过
cd backend && go run ./tools/check-file-lines --strict  # 0 errors
npx tsc --noEmit                                        # 通过（frontend）
bash scripts/gen_capability_map.sh                      # 刷新能力图
```

---

## 附录：问题汇总与状态

| # | 问题 | 严重度 | 类型 | 决策/状态 |
|---|------|:------:|------|------|
| 2.1a | `strategy_execution_logs` 写入路径整块死代码 | 🟠 高 | 死代码/破功能 | 方案 B 删除 → 任务 S-A |
| 2.1b | `CreateExecutionLog`/`UpdateExecutionLog` 恒返回非 nil error | 🔴 BUG | 逻辑错误 | 随删除消失（任务 S-A） |
| 2.2 | 回测 SymbolInfo decimal→float64→string 精度损失 | 🟡 中 | 精度合规 | `decimal.String()` 直通 → 任务 S-B |
| 2.3 | KlineData/StrategyParams `json.Marshal` 价格入 JSON | 🟡 中 | JSON/精度 | 随 S-A 删除消失 |
| 1 | rd.md「策略审计完成」夸大（实为 1 commit 删 2 文件） | ℹ️ | 流程 | 已记录 |
| 3.* | REST / WebSocket / 轮询 / float64 主体 | ✅ | 合规 | 通过 |
