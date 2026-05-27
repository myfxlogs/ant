# Design Decisions — V3 审计 B 类待定项

> 日期：2026-05-27
> 来源：`docs/audit/2026-05-27-交付审查报告-V3-Sprint后增量.md` §1
> 状态：待设计决策，随后实施

---

## B-1 · PlatformAggregator 数据源

**问题**（V3-R-3）：`PlatformAggregator` 提供 `UpdatePosition()` 方法，但全仓库无非测试调用方。`pipeline.go` Stage 3 拿 `GetSnapshot()` 永远为空 → platform_limits 层被绕过。

### 需要的决策

**数据源选型**：每 5s tick 从哪获取全账户持仓？

| 方案 | 数据源 | 优点 | 缺点 |
|------|--------|------|------|
| **A** | PG `positions` 表直接查 | 简单，已有表 | positions 表当前只在 mdgateway callback 里 INSERT，无 UPDATE同步；可能读到脏数据 |
| **B** | mthub.Hub 遍历所有 session，调 `FetchOpenedOrders` | 实时，权威数据源 | 所有用户必须保持连接；N 用户 = N 次 mtapi 调用，延迟高 |
| **C** | mdgateway RunnerDeps 暴露 `OnPositionSnapshot` 回调 → 写内存 → PlatformAggr 从内存读 | 实时 + 低延迟 | mdgateway 需新增 exporter 接口；与 snapshotBroker 定位重叠 |

**推荐**：**方案 C**。现有 `snapshotBroker.Publish()` 已经在推持仓快照，加一个 `PositionSnapshot → PlatformAggregator.UpdatePosition` 的桥接即可，不跨服务。

### 实施

- 在 main.go 的 `snapshotBroker` 订阅路径加一个 goroutine：收到 snapshot → `platformAgg.UpdatePosition` + `Recalculate`
- 或在 mdgateway 的 `OnOrderUpdate` callback 里直接调 `platformAgg.UpdatePosition`

---

## B-2 · Margin Call NATS 订阅 + 通知链路

**问题**（V3-R-7）：`TradeEventOrderMarginCall` 常量存在但无人 publish；`emailNotifier.MarginCallAlert()` 存在但无人调用。用户爆仓收不到任何通知。

### 需要的决策

**A. 谁触发 MarginCall？**

| 方案 | 实现 | 优点 | 缺点 |
|------|------|------|------|
| **A1** | mdgateway `OnAccountProfit` callback 里检测 `margin_level < threshold`，调 `eventStore.Publish(MarginCall)` | 数据已在这里，无需额外查询 | 每个 tick 都会触发，需要去重 |
| **A2** | 独立 goroutine 每 30s 扫 PG `mt_accounts` 表 | 解耦，去重简单 | 多一次 PG 查询 |

**推荐 A1**，用 `lastMarginCallSent map[accountID]time.Time` 去重（5min 冷却）。

**B. 通知通道选型？**

| 通道 | 实现 | 优先级 |
|------|------|--------|
| 邮件 | `emailNotifier.MarginCallAlert()` 已存在 | P0 — 必须 |
| SSE 推送 | `mthubSvc.PublishAccountProfit` 里加 `margin_call: true` flag | P1 |
| 站内信 | 新表 `user_notifications` + ConnectRPC `ListNotifications` | P2 |

**C. Margin Call 触发阈值？**

目前 `emailNotifier.MarginCallAlert` 只收参数不判断。建议在 publish 端判断：
- `margin_level < 100%` → "预警"（SSE + 站内信）
- `margin_level < 50%` → "强平警告"（邮件 + SSE + 站内信）

阈值存 `config` 或 `risksvc.DefaultPlatformLimits`。

### 实施

1. mdgateway callback 加 margin_level 检测 + 去重 + `eventStore.Publish(MarginCall)`
2. main.go 起 subscriber goroutine 订阅 `mthub.events.*` → 解析 MarginCall → `emailNotifier.MarginCallAlert()`
3. SSE 端：`streamServer` 已有 account profit 流，加 `margin_call` boolean 字段

---

## B-3 · Python sandbox multiprocessing 重构

**问题**（V3-R-8）：当前 `sandbox.py` 用 `threading.Timer` + `PyThreadState_SetAsyncExc`，C 扩展内部循环 / C 阻塞调用时**完全无效**。计划要求 `multiprocessing + Process.terminate`。

### 需要的决策

**A. 隔离粒度？**

| 方案 | 实现 | 优点 | 缺点 |
|------|------|------|------|
| **A1** | 每次 `call()` fork 一个子进程，结果通过 `multiprocessing.Queue` 传回 | 彻底隔离，terminate 硬生效 | fork 开销 ~50ms/次；需复制 RestrictedEnv |
| **A2** | 预热 worker pool（`multiprocessing.Pool`），每个 worker 预加载 RestrictedEnv | 复用 fork 开销 | 需管理 pool 生命周期；worker 状态污染风险 |
| **A3** | 单独用 `concurrent.futures.ProcessPoolExecutor` | stdlib 标准，API 简单 | 与 A2 类似，但 executor 封装更薄 |

**推荐 A1**（fork-per-call），原因：
- 策略执行是低频操作（backtest 单次跑几百根 bar，但用户不会同时跑几十个）
- 状态隔离最安全
- 实现最简单

**B. 超时后的行为？**

| 行为 | 说明 |
|------|------|
| 只 terminate 子进程 | `p.terminate()` + `p.join(5)` |
| terminate + 记日志 | 记录 `source_sha256` + 超时时间 → 审计追踪 |
| terminate + 通知用户 | 返回 `SandboxTimeoutError`，前端展示 |

**C. 与现有 `asyncio.wait_for` 的关系？**

`routes/strategy.py` 已有一层 `asyncio.wait_for(timeout_seconds + 5)`。改为 multiprocessing 后，`Process.terminate` 是硬杀，外层 `asyncio.wait_for` 保留作为 defense-in-depth。

### 实施

- 删除 `threading.Timer` + `_raise_async` 逻辑
- `StrategySandbox.call()` 改为 `multiprocessing.Process(target=_call_in_subprocess, args=(self, ctx, result_queue))`
- 父进程 `p.join(timeout_ms/1000)` → timeout → `p.terminate()`
- 子进程结果通过 `multiprocessing.Queue` 传回

---

## B-4 · Trading 页图表集成

**问题**（UX-1）：Trading 页 4 个子组件缺少 K 线/深度图、Bid/Ask 实时显示、symbol 下拉选择。

### 需要的决策

**A. 图表库选型？**

| 方案 | 库 | 打包体积 | K 线支持 | 学习曲线 |
|------|----|----------|----------|----------|
| **A1** | lightweight-charts (TradingView) | ~200KB | 原生 | 低 |
| **A2** | ECharts | ~1MB | 需配置 | 中 |
| **A3** | Canvas 自绘 | 0 | 自主可控 | 高 |

**推荐 A1**（lightweight-charts），与 TradingView 同源，体积小，K 线/成交量/指标原生支持。

**B. Symbol 下拉数据源？**

`MarketService.ListSymbols` 已有，但返回全量。建议加搜索/筛选：
1. 关注列表优先（从 `user_watchlists` 表读）
2. 最近交易过的 symbol 置顶
3. 搜索框输入 canoncial/display_name 模糊匹配

**C. 下单面板与图表联动？**

| 交互 | 说明 |
|------|------|
| 图表点击 → 填入价格 | 点击 K 线某个位置 → Limit Order 价格自动填入 |
| Bid/Ask 线 | 图表右侧显示实时 Bid/Ask 水平线 |
| SL/TP 可视化 | 下单后在图表上画 SL/TP 虚线，可拖拽修改 |

### 实施

1. 新增 `frontend/src/components/chart/` 目录，封装 lightweight-charts
2. `PlaceOrderForm` 上方嵌入图表组件
3. symbol 输入框改为 `AutoComplete` 下拉搜索
4. 图表与表单通过 zustand store（或组件 state）双向绑定

---

## 决策汇总

| # | 主题 | 关键选择 | 推荐 |
|---|------|---------|------|
| B-1 | PlatformAggr 数据源 | PG / Hub / mdgateway callback | **mdgateway callback (C)** |
| B-2 | Margin Call 通知 | 触发端 / 通道 / 阈值 | **mdgateway detect (A1) / 邮件+SSE / 100%/50%** |
| B-3 | sandbox multiprocessing | fork-per-call / pool / executor | **fork-per-call (A1)** |
| B-4 | Trading 图表集成 | lightweight-charts / ECharts / Canvas | **lightweight-charts (A1)** |

---

# 🎯 用户决策（2026-05-27 澄清）

> 以下为对 AI 4 项推荐的最终裁定 + AI 未考虑的关键约束 + 必须新增的子任务。**实施时按本段执行，原推荐段保留作为讨论上下文**。

---

## B-1 决策 · 采纳方案 C，但需结构清晰化

**裁定**：✅ 采用方案 C（mdgateway 推 → 桥接 → platformAgg），但**不要复用 `snapshotBroker`**。

### 理由

`snapshotBroker.Publish()` 当前职责是给前端 SSE 推持仓快照，订阅端是 HTTP streaming handler。把"风控聚合"挂到同一个 broker 上会造成：
- **故障耦合**：前端 SSE 客户端慢/断开不能影响风控
- **数据形状冲突**：SSE 要 JSON 友好（含 symbol display name），风控要 canonical + 数值精度
- **测试困难**：risk 单元测试要 mock SSE broker

### 实施约束

1. **新增独立接口** `internal/mdgateway/position_exporter.go`：
```go
type PositionExporter interface {
    OnPositionUpdate(accountID string, pos PositionUpdate)
    OnAccountClose(accountID string)
}
```
2. mdgateway runner 维护 `exporters []PositionExporter` slice，每次 broker 推持仓快照时**同步广播给所有 exporters**（fan-out）。
3. `snapshotBroker` 和 `platformAggExporter` 都实现该接口；main.go 注册两个。
4. **关键遗漏**：账户**断连时必须调 `platformAgg.ClearAccount(accountID)`**，否则会用陈旧数据计算 NetExposure。原推荐未提。
5. **Recalculate 频率**：不要每次 UpdatePosition 都跑 Recalculate（O(N·M) 开销）。改成：
   - UpdatePosition 标记 `dirty=true`
   - 5s ticker goroutine 检查 dirty 后 Recalculate 并 atomically swap snapshot
   - pipeline Stage 3 从 atomic.Pointer 读最新 snapshot

### 新增子任务

- **B-1.1**：定义 `PositionExporter` 接口（不复用 snapshotBroker）
- **B-1.2**：mdgateway runner fan-out 多 exporter
- **B-1.3**：`platformAggExporter` 适配器实现 + 注册到 main.go
- **B-1.4**：5s Recalculate ticker + atomic.Pointer snapshot swap
- **B-1.5**：账户 disconnect/logout 时 `ClearAccount` 触发
- **B-1.6**：integration test 验证：A 用户连接 → 持仓出现在 snapshot；A disconnect → 5s 内消失

---

## B-2 决策 · 采纳 A1，但阈值 per-broker，且追加 Telegram

**裁定**：
- ✅ 触发端 **A1**（mdgateway OnAccountProfit callback）
- ⚠ 通道：**邮件 P0** + **SSE P0**（升级，不能 P1）+ **Telegram P0**（新增）+ 站内信 P1
- ❌ 阈值 **不要硬编码 100%/50%**，必须 per-broker 配置

### 理由

#### 阈值问题
MT4/MT5 不同 broker 的 `margin_call_level` 与 `stop_out_level` 差异巨大：
- Exness: margin_call=60%, stop_out=0%
- IC Markets: margin_call=100%, stop_out=50%  
- FBS: margin_call=40%, stop_out=20%

硬编码 100%/50% 会导致：
- Exness 用户在 70% margin level 时收到"强平警告"邮件 → 实际 broker 不强平 → **狼来了**
- IC Markets 用户在 110% margin level 时**收不到任何预警** → 实际下个 tick 就 margin call

#### 通道问题
原推荐把 Telegram 放到 V3-9（一周后）。但**散户交易者 80% 时间不在网页前**，爆仓邮件可能 24h 后才打开。Telegram 必须 P0 同步实现。

### 实施约束

1. **从 broker 拉取 margin levels**：
   - mtapi 的 `AccountInfoDouble(ACCOUNT_MARGIN_SO_CALL)` 和 `ACCOUNT_MARGIN_SO_SO` 返回这两个值
   - mdgateway 连接成功时拉一次，缓存到 `mt_accounts.broker_margin_call_pct` + `broker_stop_out_pct`（migration 新增 2 字段）
2. **三级阈值**（基于 broker 配置，不是硬编码）：
   - `margin_level <= broker_margin_call_pct × 1.5` → **预警**（SSE only）
   - `margin_level <= broker_margin_call_pct` → **警告**（SSE + Email + Telegram）
   - `margin_level <= broker_margin_call_pct × 0.7`（接近 stop_out）→ **危急**（SSE + Email + Telegram，去重冷却缩短至 1min）
3. **去重策略**：`map[accountID]map[level]time.Time`，每级独立冷却 5min；上升回到安全区时清除记录（下次跌回再次触发）
4. **Telegram 集成**：
   - 新表 `user_telegram_links(user_id, chat_id, verified_at, opt_in_alerts)`
   - 用户在前端绑定 Telegram（OAuth-like：bot 给 token，用户在 bot 里 `/bind <token>`）
   - notifier 新增 `TelegramNotifier`，main.go wire

### 新增子任务

- **B-2.1**：migration 新增 `mt_accounts.broker_margin_call_pct / broker_stop_out_pct`
- **B-2.2**：mdgateway 连接成功时拉取 margin levels 并写 PG
- **B-2.3**：OnAccountProfit callback 三级阈值检测 + 去重
- **B-2.4**：`notifier.TelegramNotifier` 实现 + bot 命令 (`/bind` `/status` `/pause_alerts`)
- **B-2.5**：用户 Telegram 绑定 UI（Account Settings 页）
- **B-2.6**：站内信表 + ConnectRPC `ListNotifications` / `MarkRead`（P1，但 schema 本次落地）

---

## B-3 决策 · A1 不可行，改为**双路径** + 显式 start_method

**裁定**：❌ **拒绝 fork-per-call**，改为：
- **Backtest 路径**：1 进程包整个 backtest run（粗粒度）
- **Live 路径**：worker pool with per-run isolation

### 理由

AI 推荐 fork-per-call 时假定"低频"。但实际：
- Backtest 1 年 H1 数据 = ~6000 根 bar = 6000 次 fork × 50ms = **5min 纯 fork 开销**（不算策略本身）
- 用户体验：点 "Backtest" 后等 5min 看进度条 → 弃用
- 实测：strategy-service 单进程 backtest 一年 H1 数据通常 < 30s

### 实施约束

#### 双路径设计

**Backtest 路径**（fork-per-RUN）：
```python
# routes/strategy.py:backtest
def run_backtest(req):
    parent_conn, child_conn = multiprocessing.Pipe()
    p = multiprocessing.Process(
        target=_backtest_in_subprocess,
        args=(req.code, req.bars, child_conn),
    )
    p.start()
    p.join(timeout=req.timeout_seconds or 1800)  # 默认 30min
    if p.is_alive():
        p.terminate(); p.join(5)
        if p.is_alive(): p.kill()
        raise SandboxTimeoutError("backtest exceeded timeout")
    return parent_conn.recv()
```

**Live 路径**（worker pool with per-run isolation）：
```python
# 每个 strategy_run_id 启动一次 worker process，整个 live run 复用
# call() 通过 Pipe IPC 调用 worker；超时用 worker 内部 signal.alarm
class LiveWorker:
    def __init__(self, source: str):
        self.proc = multiprocessing.Process(target=_live_worker_main, args=(source, ...))
    def call(self, ctx) -> dict | None:
        self.send_pipe.send(ctx)
        if not self.recv_pipe.poll(timeout=5.0):
            self.proc.terminate(); raise SandboxTimeoutError
        return self.recv_pipe.recv()
```

#### 平台兼容性约束（AI 未提）

⚠ **macOS Python ≥ 3.8 默认 `spawn`**，不是 fork。必须在模块顶部显式：
```python
import multiprocessing
if sys.platform != "win32":
    multiprocessing.set_start_method("fork", force=True)
```
否则 macOS 开发机 + Linux 生产行为不一致（spawn 会丢失全局状态如 RestrictedEnv 缓存，行为差异极大）。

#### RestrictedEnv 复制

`fork()` 是 copy-on-write，RestrictedEnv 实例不会立即复制。但**子进程内修改 RestrictedEnv** 不会同步回父进程（OK，正是隔离需求）。

需注意：`_get_bytecode` 缓存（`@lru_cache`）在 fork 后子进程**独立**，第一次调用会重新编译。如果 RestrictedEnv 编译耗时显著，应该在父进程**预编译一次**，把 bytecode 通过 Pipe 传给子进程而非传 source。

### 新增子任务

- **B-3.1**：`sandbox.py` 移除 threading.Timer + PyThreadState_SetAsyncExc，标 deprecated
- **B-3.2**：新建 `engine/backtest_sandbox.py`：fork-per-run，结果走 Pipe
- **B-3.3**：新建 `engine/live_sandbox.py`：长驻 worker process + Pipe IPC + 内部 signal.alarm 超时
- **B-3.4**：模块入口 `set_start_method('fork')` + Windows 兼容性回退（spawn + 显式状态注入）
- **B-3.5**：预编译 bytecode 父进程缓存，子进程接收编译产物
- **B-3.6**：单元测试覆盖：(a) `while True: pass` 超时；(b) `time.sleep(99999)` 超时；(c) C 扩展 `numpy.dot(big_matrix)` 超时；(d) 子进程 OOM 后父进程仍存活

---

## B-4 决策 · 采纳 A1，但分阶段交付 + 性能预算约束

**裁定**：✅ lightweight-charts，但分两阶段：
- **Phase 1**（本轮）：read-only 图表 + symbol AutoComplete + Bid/Ask 线
- **Phase 2**（V3-11 路线图）：图表点击填价 + SL/TP 可拖拽

### 理由

原推荐把"点击填价 + SL/TP 拖拽"放在同一批次。但：
- SL/TP 拖拽涉及双向数据流（图表↔表单）+ 防抖 + 撤销，约 2-3 工日
- 用户最痛点是"看图与下单要切两个页面"，而非"无法拖拽"
- 应该先用 Phase 1 验证用户接受度，再投资 Phase 2

### 实施约束

#### 性能预算（AI 未提）
- lightweight-charts 库本身 ~200KB，但**必须 dynamic import**（`React.lazy + Suspense`），只在 Trading 页加载，不进 main bundle
- 初始 K 线数据上限 5000 根（再多浏览器卡顿），用户拖拽到边界时增量加载
- WebSocket / SSE 推 tick 必须**节流到 100ms**，否则单 symbol 每秒数十 tick 直接卡前端

#### Symbol AutoComplete 数据源
- ✅ 关注列表优先（user_watchlists 表）—— 但**该表当前不存在**，需要新增 migration + RPC
- 当前 fallback：从 `MarketService.ListSymbols` 拉全量 + 客户端模糊匹配（symbol 数量通常 < 500，可接受）
- 后续：加 PG 全文索引 + server-side search

#### 移动端考虑
Trading 页本就是桌面优先（V3-17 才做移动响应式）。Phase 1 图表至少加 `min-width: 1280px` 媒体查询，< 1280px 隐藏图表，只显示表单（防止移动端布局崩溃）。

### 新增子任务

- **B-4.1**：`frontend/src/components/chart/PriceChart.tsx` — lightweight-charts 封装，dynamic import
- **B-4.2**：migration 新增 `user_watchlists(user_id, symbol_canonical, sort_order)` + RPC
- **B-4.3**：`SymbolAutoComplete.tsx` — 关注列表 + 最近交易 + 全文搜索
- **B-4.4**：Bid/Ask 水平线渲染（订阅现有 quote SSE 流）
- **B-4.5**：< 1280px 媒体查询隐藏图表
- **B-4.6**：tick 节流到 100ms（lodash throttle 或自实现）
- **B-4.7**：（Phase 2 推迟）点击图表填价 + SL/TP 拖拽

---

## 最终决策汇总（修订版）

| # | 主题 | 原推荐 | 用户裁定 | 关键修订 |
|---|------|--------|---------|---------|
| **B-1** | PlatformAggr 数据源 | mdgateway callback (C) | ✅ 采用，但**不复用 snapshotBroker**，独立 `PositionExporter` 接口；新增 `ClearAccount` on disconnect；5s atomic snapshot swap | 6 子任务 |
| **B-2** | Margin Call 通知 | A1 / 邮件+SSE / 100%-50% | ⚠ **阈值 per-broker 拉取，不硬编码**；**Telegram 升级 P0**；SSE 升级 P0；三级阈值（预警/警告/危急） | 6 子任务（含 Telegram bot） |
| **B-3** | sandbox 隔离 | fork-per-call (A1) | ❌ **拒绝 fork-per-call**（backtest 6000bar × 50ms = 5min 不可接受）；改 **backtest fork-per-RUN + live worker pool**；必须 `set_start_method('fork')` 显式 | 6 子任务 |
| **B-4** | Trading 图表 | lightweight-charts (A1) | ✅ 采用，但**Phase 1 仅 read-only + Bid/Ask 线**；SL/TP 拖拽延后；必须 dynamic import + tick 节流 100ms + < 1280px 隐藏 | 7 子任务（Phase 2 留 1 项） |

**新增子任务总数**：25 项；估时约 18-22 工日（B-1: 2d / B-2: 5d / B-3: 4d / B-4: 4d Phase1，另 3d Phase2 推迟）。

---

## 实施顺序建议

```
Week 1
├─ B-1.1~B-1.6 (PlatformAggr 真实数据源)        — 2d, 解锁 V3-R-3
├─ B-2.1~B-2.3 (broker 阈值 + 3 级检测)          — 2d, 解锁 V3-R-7 基础
└─ B-2.6 (站内信 schema/RPC)                     — 0.5d

Week 2
├─ B-2.4~B-2.5 (Telegram bot + UI 绑定)          — 2.5d
├─ B-3.1~B-3.4 (sandbox 双路径核心)              — 3d, 解锁 V3-R-8
└─ B-4.1~B-4.2 (图表组件骨架 + watchlists)      — 1.5d

Week 3
├─ B-3.5~B-3.6 (sandbox 性能优化 + 测试)         — 1.5d
├─ B-4.3~B-4.6 (AutoComplete + Bid/Ask 线 + 节流) — 2.5d
└─ buffer + integration test + handover           — 1d
```

总 ≈ 14-16 工日（一人全职 3 周）。

---

# 🔍 施工条件审查（2026-05-27 · 第二轮）

> 审查人：Claude Opus 4.7（实施侧）
> 方法：逐子任务检查代码依赖 + 现有架构兼容性
> 结论：**方向全部正确，但 4 个 B 项均有施工阻塞点。共发现 3 处矛盾、8 处遗漏。**

---

## B-1 审查 · 有 2 处遗漏，无矛盾

**裁定无矛盾**。`PositionExporter` 独立于 snapshotBroker 是正确的。

### 遗漏 1 · `PositionUpdate` 类型未定义

`OnPositionUpdate(accountID string, pos PositionUpdate)` 中 `PositionUpdate` 的类型未指定。两个选择：

| 选项 | 说明 |
|------|------|
| **复用** `mdtick.OrderUpdate` | 现成，但带大量前端 SSE 不需要的字段（Comment / Commission / Swap） |
| **新建** `PositionUpdate`（轻量） | 只含 `AccountID, Symbol, Volume, OpenPrice, CurrentPrice`，风控专用 |

**需要裁定**：选哪个？影响接口签名。

### 遗漏 2 · `ClearAccount` 触发时机未指定

账户断连时需调 `ClearAccount(accountID)`，但断连检测点未明确。当前 mdgateway 在 `adapter/mt4/connection.go` 和 `adapter/mt5/connection.go` 中是否有 disconnect hook？

**需要确认**：mtapi session 断开时的回调路径。

---

## B-2 审查 · 1 处矛盾 + 3 处遗漏

### 矛盾 1 · broker 阈值拉取失败时无 fallback

B-2.3（三级阈值检测）依赖 B-2.1 + B-2.2（migration 加字段 + mdgateway 拉取 broker 阈值）。但如果：

- mtapi 版本不支持 `ACCOUNT_MARGIN_SO_CALL`
- 网络超时取不到
- broker 返回 0（表示不适用）

三级阈值退化成什么？文档说"不要硬编码"，但 fallback 策略是必须的。

**建议 fallback**：取不到时用 `DefaultPlatformLimits.MarginCallPct` / `StopOutPct`（`risksvc/platform_limits.go` 中定义），即 100%/50%，并 log.Warn 标记使用默认值。

### 遗漏 3 · Telegram bot token 来源未定义

Bot token 从哪来？环境变量 `TELEGRAM_BOT_TOKEN`？admin 配置页存入 `platform_configs` 表？影响 `TelegramNotifier` 的初始化方式。

### 遗漏 4 · 已有 R-7 实现需要重构

当前 `main.go` 的 R-7 实现（commit `cb527c2`）使用了：
- 单级硬编码 100% 阈值
- `marginCallLastSent map[string]time.Time` 单级去重

升级到三级 + per-broker 阈值时，去重结构需改为 `map[accountID]map[level]time.Time`。

### 遗漏 5 · 估时偏低

6 个子任务含 Telegram bot 完整基础设施（新表 + bot 命令 `/bind` `/status` `/pause_alerts` + 前端绑定 UI + `TelegramNotifier` 实现），5d 偏低。实际估算：7-10d。

---

## B-3 审查 · 2 处矛盾 + 3 处遗漏（风险最高）

### 矛盾 1 · `set_start_method('fork', force=True)` 与 Python 运行时冲突

`force=True` 在 macOS 上有已知风险：
- Python 文档要求 `set_start_method` 在 `if __name__ == '__main__':` 保护下调用
- strategy-service 是 FastAPI + uvicorn worker（非 `__main__` 上下文）
- `force=True` 可能在 macOS 开发环境 crash

**建议**：不要在模块顶层 `force=fork`。改为在 `_backtest_in_subprocess` 函数入口处显式检查 `multiprocessing.get_start_method()`，如果非 fork 且非 Windows，用 `spawn` + 显式传 `RestrictedEnv`（pickle 序列化）。

### 矛盾 2 · LiveWorker `poll(timeout=5.0)` 与 sandbox `timeout_ms` 不统一

现有 `sandbox.py` 中 `timeout_ms` 默认 30000ms，Live 策略的 `call()` 应在毫秒级完成（signal 计算）。硬编码 5.0 秒对 live 路径太宽松，且与 `timeout_ms` 参数脱节。

**建议**：LiveWorker 的 poll timeout 使用 `self._timeout_ms / 1000.0`，与 sandbox 保持一致。

### 遗漏 6 · fork + asyncio 冲突

FastAPI 在 asyncio event loop 中运行。`os.fork()` 后子进程继承父进程的 event loop 状态（已关闭的 fd、signal handlers、pending callbacks），可能导致死锁或 `RuntimeError: Event loop is closed`。

**建议**：子进程入口函数第一行调 `asyncio.set_event_loop(asyncio.new_event_loop())` 重建 event loop。或使用 `multiprocessing.get_context('forkserver')` 隔离。

### 遗漏 7 · LiveWorker 内存泄漏

长驻 worker process 不断接收 `call()` 请求，策略执行上下文对象不清。Python GC 在子进程中可能不触发（引用循环 + 无显式 `gc.collect()`）。

**建议**：加 `_call_count` 计数器，每 100 次 call 自动 `p.terminate()` + 重新 spawn。

### 遗漏 8 · 双路径缺少统一抽象

BacktestSandbox（fork-per-run）和 LiveWorker（长驻）对外接口不同，`routes/strategy.py` 需要用 `if isinstance` 分支。长期维护成本高。

**建议**：定义 `BaseSandbox` ABC：
```python
class BaseSandbox(ABC):
    @abstractmethod
    def call(self, ctx: dict) -> Optional[dict]: ...
    @abstractmethod
    def shutdown(self) -> None: ...
```

---

## B-4 审查 · 0 处矛盾 + 2 处遗漏（风险最低，可先行开工）

**裁定无矛盾**。Phase 1 read-only + Phase 2 交互分阶段合理。

### 遗漏 9 · Bid/Ask 线数据源频率不足

当前 mdgateway 推的是 5 秒 OHLC bar，不是 tick-level quote。Bid/Ask 水平线在 5 秒粒度下更新频率太低，用户可能看到线条 5 秒跳一次，视觉效果不佳。

**建议**：Phase 1 先用 OHLC close 作为 Bid/Ask 近似值（5 秒刷新），注上 "实验性" 标签；Phase 2 再接入 tick-level quote 流。

### 遗漏 10 · 图表初始数据来源与延迟

首次加载 5000 根 K 线从 `MarketService.GetPriceHistory` 拉？一次 HTTP 请求拉 5000 根 bar 的 JSON 响应 ≈ 2-5MB（含 OHLC + timestamp），在慢网络下加载时间不可忽略。

**建议**：首屏仅拉最近 300 根 bar（~3 分钟数据），用户缩放/拖拽历史时用 `getRange` 增量加载。

---

## 施工条件汇总

| B 项 | 矛盾 | 遗漏 | 可开工？ |
|------|------|------|---------|
| B-1 | 0 | 2（类型未定义 / 断连点不明） | ⚠ 需补充 |
| B-2 | 1（fallback 缺失） | 3（token / 重构 / 估时） | ⚠ 需补充 |
| B-3 | 2（fork+macOS / 超时不统一） | 3（asyncio / 泄漏 / 抽象） | 🔴 风险最高，需第二轮澄清 |
| B-4 | 0 | 2（quote 频率 / 初始数据量） | ✅ Phase 1 可先行开工 |

---

## 给 Opus 4.7 的第二轮澄清请求

请对以下 11 项给出裁定（同意/拒绝/替代方案）：

| # | B 项 | 类型 | 问题 | 建议 |
|---|------|------|------|------|
| 1 | B-1 | 遗漏 | `PositionUpdate` 类型 | 选复用 mdtick 还是新建轻量类型？ |
| 2 | B-1 | 遗漏 | `ClearAccount` 触发时机 | 断连检测在 mt4/mt5 connection.go 哪个 hook？ |
| 3 | B-2 | 矛盾 | broker 阈值 fallback | 采用 `DefaultPlatformLimits` (100%/50%) + log.Warn? |
| 4 | B-2 | 遗漏 | Telegram bot token 来源 | env var 还是 DB 配置？ |
| 5 | B-3 | 矛盾 | `set_start_method('fork', force=True)` | 改为 spawn + 显式状态注入? |
| 6 | B-3 | 矛盾 | LiveWorker poll timeout | 改为 `self._timeout_ms / 1000.0`? |
| 7 | B-3 | 遗漏 | fork + asyncio 冲突 | 子进程重建 event loop? |
| 8 | B-3 | 遗漏 | LiveWorker 内存泄漏 | 每 100 次 call 自动重启? |
| 9 | B-3 | 遗漏 | 双路径缺统一抽象 | 定义 `BaseSandbox` ABC? |
| 10 | B-4 | 遗漏 | Bid/Ask 数据源频率不足 | Phase 1 用 OHLC close 近似 + "实验性"? |
| 11 | B-4 | 遗漏 | 图表初始数据量过大 | 首屏 300 根，增量加载历史? |

