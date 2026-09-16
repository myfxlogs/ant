# tz-timestamp-sweep-2026-09 — `timestamp` 列 vs Go time 参数全仓排查

> RECONCILE-TZ-WINDOW-1 S3 交付。只查不改：本站修复限 `reconciliation.go`，其余站点列清单由决策方分批排期。
> 环境前提：后端容器 `TZ=Asia/Shanghai`（Go time.Local=CST），postgres `SHOW timezone`=UTC。

## 0. 判定模型

pgx 对 `timestamp`（OID 1114，非 timestamptz）参数按 **wall-clock 分量**编码——`time.Now()`（CST Local）参数编码为 CST 钟面，`time.Now().UTC()` 编码为 UTC 钟面。列内存储值同理 = 写入方的编码钟面。因此一条时间比较的正确性 = **写入编码与参数编码是否一致**，与"哪侧是 UTC"无关：

| 写入编码 | 参数编码 | 结果 |
|----------|----------|------|
| PG `now()`/DEFAULT/`.UTC()` 参数 → UTC 钟面 | `.UTC()` 参数 / SQL `now()` | ✅ 一致 |
| 同上 UTC 钟面 | `time.Now()`（CST） | ❌ 参数偏 +8h，窗口缩短 8h |
| `time.Now()`/`time.Unix()`（Local→CST 钟面） | CST 参数 | ✅ 一致（自洽） |
| 同上 CST 钟面 | `.UTC()` 参数 / SQL `now()` | ❌ 反向偏 8h，窗口变宽/判定错位 |
| **混合写入（同列 CST+UTC 并存）** | 任何单侧编码 | ❌ 对一半行错 |

timestamptz 列按 instant 规范化，全部不受影响（`order_history` 表 open_time/close_time，`012_enhanced_logs`）。

## 1. 本站（已修）

| 站点 | 列 | 写入编码 | 原参数编码 | 判定 |
|------|-----|----------|------------|------|
| `mthub/reconciliation.go:161` | `orders.created_at` | PG `DEFAULT now()` → **UTC** | ~~CST~~ → 已改 `.UTC()` | ✅ FIXED |
| `mthub/reconciliation.go:163` | `trade_records.close_time` | **混合**（见 §3） | 同上 | ⚠️ UTC 分支已对齐；CST 写入行窗口 24→32h，仅多收旧 CLOSED 行→orphan warn 噪声（UNION 分支 state 恒 'CLOSED'，不触发 repair，无害残留待 §3 根治） |

broker 侧 `FetchOrderHistory(Clk.Now()...)`（:139）不动——RPC 传真实 instant，无时区编码问题（复核确认）。

## 2. AFFECTED——UTC 编码列 vs 非 UTC 参数（同型 bug，待排期）

| 站点 | 列（写入编码） | 参数 | 影响 |
|------|----------------|------|------|
| `marketplace/analytics.go:45` → :54/:66/:79/:91/:137 | `wallet_transactions.created_at`（INSERT 省列→DEFAULT **UTC**） | `since := time.Now()`（CST，缺 `.UTC()`） | GMV/手续费/结算/购买额/退款统计窗口起点偏 +8h，周期性少计 8h 内交易 |
| `marketplace/analytics.go:101` | `user_subscriptions.created_at`（DEFAULT **UTC**） | 同上 `since` | 新增订阅计数同偏 |
| `marketplace/analytics.go:126` | `marketplace_strategies.created_at`（`now()` **UTC**） | 同上 `since` | 新上架策略计数同偏 |
| `marketplace/analytics.go:179/:182` | `wt.created_at`（wallet_transactions，**UTC**） | 同上 `since` | 收入明细同偏 |
| `repository/backtest_run_worker.go:144` | `backtest_runs.created_at`（`CURRENT_TIMESTAMP` **UTC**） | `since` 参数（调用方未定） | 当前**无调用方**（休眠 API）；启用前须确认参数 UTC |
| `service_subscription.go:196/:385` + `notification_triggers.go:26/:168` | `user_subscriptions.expires_at` | SQL `now()`（UTC） | 列**混合编码**（§3）→ 对 CST 写入的订阅行，到期判定晚 8h（超期仍 active）；对 `now()+INTERVAL` 写入的续费行正确 |

修法建议（分批）：analytics.go 单点 `since := time.Now().UTC()`（一处改全站愈）；`backtest_run_worker` 加调用方约束注释或参数归一化。

## 3. MIXED——混合编码列（根因级，建议独立立项）

| 列 | CST 写入路径 | UTC 写入路径 | 后果 |
|----|--------------|--------------|------|
| `trade_records.close_time` / `open_time` | 实时流 `buildClosedTradeRecord`：`time.Unix(...)`（Local）`cmd/server/pipeline_callbacks.go:162` | 历史/同步：`GetCloseTime().AsTime()`（UTC）`adapter/mt4/order_history.go:67` + `account_sync_service.go` + `ImportBrokerOrder` | 同列两编码并存 → 任何单侧参数只对一半行正确：analytics CST 参数对 live 行对、对 import 行错（窗口 +8h）；`live_performance.go:247` UTC 日界桶对 import 行对、对 live 行错（日界偏 -8h，跨日 PnL 错归） |
| `user_subscriptions.expires_at` | 首次购买 `time.Now().Add(30d)`（`purchase.go:101`） | 续费 `now()+INTERVAL '30 days'`（`service_subscription.go:297`） | 同上：SQL `now()` 比较对 CST 行晚判 8h 到期 |

修法方向（决策方裁定）：统一写入方为 `.UTC()`（`time.Unix(t,0).UTC()`、购买路径 `.UTC()`）+ 存量数据识别/回填；或列改 timestamptz（migration + 全写入方改造，过重——与 D-013 修法排除记录一致）。

## 4. CONSISTENT——自洽（无需动）

| 站点 | 列（编码） | 参数（编码） | 说明 |
|------|------------|--------------|------|
| `trial.go:85/:92` | `marketplace_trials.expires_at`（`time.Now().UTC()` 写入） | `time.Now().UTC()` | ✅ |
| `live_performance.go:209/:247` | `date`（DATE 型，UTC truncate 写入） | `time.Now().UTC()` | ✅ DATE 无钟面分量 |
| `platform_health_handler.go:54/:65/:110/:148/:161/:165` | `backtest_runs.created_at`（UTC） | `time.Now().UTC()` | ✅ |
| 调度器 `schedule_read_repo.go:128/:154` | `strategy_schedules.next_run_at`（CST 写入） | `e.nowTime()`→`time.Now()`（CST） | ✅ 同侧自洽 |
| `analytics_repository_*` close_time 区间（pnl/daily/equity/metrics/trades/account_stats/user_stats ~15 站） | `trade_records.close_time`（混合） | `time.Now()` CST（analytics_handler/attribution/rolling） | ⚠️ 对 live 行自洽、对 import 行偏——挂 §3 根治 |
| `trade_record_repository.go:85`、`analytics_repository_trades.go` 各站 | 同上 | `time.Now()` CST（analytics_handler:210） | 同上 |
| `trade_log_repository.go:116`、`analytics_repository_trades.go:83` | `trade_logs.created_at`（`time.Now()` CST 写入） | CST 参数 | ✅ 自洽（但见 §5 写入类） |

## 5. 写入侧同类缺陷（class B，非比较站点）

约 20+ 处 `SET xxx_at = $N` / INSERT 参数用 `time.Now()`（Local CST）写 `timestamp` 列 → 存储值 = 真实 instant +8h；pgx 解码 `timestamp` 按 UTC 钟面还原 → **读回 instant 系统性 +8h**（API/proto Timestamp 序列化、UI 显示受影响）。站点（抽样）：`schedule_write_repo.go`（next_run_at/last_run_at/updated_at×6）、`template_svc*.go` updated_at、`auto_trading_*.go` updated_at/daily_loss_used、`ai_conversation/ai_workflow` updated_at、`user_repo.go:168` last_login_at、`paper_repo.go:148` closed_at、`signal_svc.go:70` executed_at、`strategy_asset_repository.go:178` last_sync_check_at、`trade_log/auto_trading/connection_log/operation_log` created_at 参数写入。**建议**：写入侧统一 `.UTC()` 规范化（与 §3 同批）；`updated_at` 触发器表（users/mt_accounts/positions）已由 PG `CURRENT_TIMESTAMP` 写 UTC，无此问题。

## 6. VERIFY——需客户端语义复核（未定论）

| 站点 | 说明 |
|------|------|
| `connection_log_repository.go:54-55`、`operation_log_repository.go:66-67`、`auto_trading_settings.go:132-133` | `created_at`（CST 参数写入）vs `StartDate/EndDate` **string** 参数——PG 按字面解析，正确性取决于前端所传日期语义（本地日界 vs UTC 日界），边界可能 ±1 天 |
| `order_history_repository.go:81/:86` | `order_history.open_time` 是 **timestamptz** → 参数 string 按 instant 解析，无本类问题（仅列出以闭环） |
| `orders.open_time`（001:74 TIMESTAMP NOT NULL） | INSERT 路径均未显式写 open_time，取值来源待查（疑似 DEFAULT/后补写），编码待确认 |
| `analytics_handler.go:282` `time.Now().Format("2006-01-02")` | Go 侧字符串日界（CST 本地日）与 curve Date（源于 close_time）比较——非 pgx 编码类，记为同类时区语义待核 |
| `trade_records.close_time` 读侧显示链路 | proto Timestamp 序列化对 CST 写入行 +8h——与 §5 同根因，§3 根治后消 |

## 7. 汇总

| 类 | 站点数 | 处置 |
|----|--------|------|
| FIXED | 1（reconcile，本单） | 已修+pin 测试 |
| AFFECTED（UTC 列 vs CST 参数） | analytics.go ×9 + 休眠 API ×1 | 建议一批修（单点 `.UTC()`） |
| MIXED 列 | close_time/open_time、user_subscriptions.expires_at | 建议独立立项（写入归一化+回填裁定） |
| CONSISTENT | ~25 站 | 不动 |
| 写入侧 class B | ~20+ 站 | 建议与 MIXED 同批规范化 |
| VERIFY | 6 站 | 决策方裁定后再排 |

**声明**：本报告为排查清单，不构成任何站点的施工授权；除 reconcile 本站外零代码改动。
