# Audit Remediation Handover — 2026-05-27

## 概述

本次交付修复了 opus4.7 审计发现的 4 个 B 级问题（Fix 1-4），加上 Week 3 的功能开发（B-3.5/3.6/4.3/4.5/4.6）以及若干预存的构建修复。

## Fix 1: 删除死代码 PositionExporter 接口 (B-1)

**文件**: `backend/internal/risksvc/position_exporter.go` — 已删除

`PositionExporter` 接口从未被任何代码导入或使用。实际的风险数据流转通过 `RunnerDeps` 中的函数回调（`OnAccountProfit`, `OnOrderUpdate`）完成，而非通过该接口。

## Fix 2: Broker 保证金阈值获取 (B-2.2)

**问题**: 虽然 proto 定义了 `BrokerInfo` 消息类型，但 mdgateway 从未在 Connect 后从 mtapi 获取 broker 级别的保证金设置（MarginCallLevel / StopOutLevel）。

**修改**:

| 文件 | 变更 |
|------|------|
| `backend/internal/mdgateway/adapter/mdtick/mdtick.go` | 新增 `BrokerInfo` 结构体、`BrokerInfoHandler` 回调类型、`BrokerInfoFetcher` 接口 |
| `backend/internal/mdgateway/adapter/mt4/connection.go` | 实现 `FetchBrokerInfo()` — 调用 `AccountSummary`，当前返回零值（proto 尚未暴露对应字段） |
| `backend/internal/mdgateway/adapter/mt5/connection.go` | 同上 |
| `backend/internal/mdgateway/runner.go` | `RunnerDeps` 新增 `OnBrokerInfo` 回调；Connect 成功后自动调用 `FetchBrokerInfo` |
| `backend/cmd/server/main.go` | 注入 `OnBrokerInfo` 回调，将阈值写入 `mt_accounts` 表 |

**备注**: mtapi proto v2.x 的 `AccountSummary` 尚未暴露 `MarginCallLevel` / `StopOutLevel` 字段。当 proto 扩展后，取消 `FetchBrokerInfo` 中的注释即可自动填充真实阈值。

## Fix 3: PlatformAggregator 异步重算 (B-1.4)

**问题**: 每次 `UpdatePosition` / `ClearAccount` 调用同步执行 `Recalculate()`（O(N*M) 复杂度）。`OnOrderUpdate` 在每个仓位变动时触发，导致热路径上的延迟爆炸。

**修改**: `backend/internal/risksvc/platform_aggregator.go` — 完全重写

- 新增 `dirty bool` 标志位 — `UpdatePosition` / `ClearAccount` 仅标记 dirty=true
- 新增 `snapshot unsafe.Pointer` — 通过 `atomic.StorePointer/LoadPointer` 实现无锁快照读取
- 新增 `StartRefreshLoop(interval)` — 后台 5s ticker 检查 dirty 并调用 `Recalculate`
- 新增 `Shutdown()` — 优雅停止
- 移除热路径上的所有同步 `Recalculate()` 调用
- `main.go` 中改为 `platformAgg.StartRefreshLoop(5 * time.Second)`

## Fix 4: PriceChart 动态导入 + 增量加载 (B-4)

**问题**: 
1. `Trading.tsx` 静态导入 `PriceChart`（~550KB lightweight-charts），阻塞首屏渲染
2. 图表仅加载固定 300 根 K 线，用户拖拽到左边缘时无法加载更早的历史数据

**修改**:

| 文件 | 变更 |
|------|------|
| `frontend/src/client/market.ts` | `getKlines` 新增可选 `before` 参数，映射 proto `to` 字段实现基于游标的翻页 |
| `frontend/src/components/chart/PriceChart.tsx` | 实现 `handleVisibleRangeChange` 增量加载逻辑；新增 `loadingMore`/`loadedAll` ref 防止重复请求；订阅 chart `timeScale().subscribeVisibleLogicalRangeChange`；符号/周期切换时重置翻页状态 |
| `frontend/src/pages/trading/Trading.tsx` | 改用 `React.lazy(() => import(...))` 动态导入 PriceChart；包裹 `<Suspense fallback={<Spin/>}>` |

## 附加变更 (Week 3 功能开发)

### B-3.5/3.6: Python 沙箱字节码预编译 + 测试

| 文件 | 变更 |
|------|------|
| `strategy-service/app/engine/sandbox.py` | 提取 `build_sandbox_globals()` 为独立函数；新增 `compile_and_serialize()` / `exec_serialized()` 使用 `marshal` |
| `strategy-service/app/engine/backtest_sandbox.py` | 重写为通过 Pipe 传递预编译字节码（不再传递源码字符串） |
| `strategy-service/app/engine/live_sandbox.py` | 同上，worker 启动时编译一次，复用 500 次调用 |
| `strategy-service/app/engine/sandbox_base.py` | ABC 基类 |
| `strategy-service/app/engine/__init__.py` | 导出新类型 |
| `strategy-service/app/engine/runner.py` | 接入 BacktestSandbox |
| `strategy-service/tests/test_sandbox_mp.py` | 7 个测试：超时（无限循环/大计算/numpy C 扩展）、OOM 存活、LiveWorker 基本/多次/重启 |

### B-4.3/4.5/4.6: 前端增强

| 文件 | 变更 |
|------|------|
| `frontend/src/components/chart/SymbolPicker.tsx` | localStorage 自选列表、分组选项（自选/全部）、星标切换 |
| `frontend/src/hooks/useThrottle.ts` | 通用 throttle hook（trailing-edge） |
| `frontend/src/pages/trading/components/PlaceOrderForm.tsx` | 用 SymbolPicker 替换手动 Input |
| `frontend/package.json` | 添加 `lightweight-charts` 依赖 |

### 构建修复 (预存问题)

| 文件 | 问题 |
|------|------|
| 7 个文件中的 `Icon*` 导入 | `@ant-design/icons` 不导出 `IconHome` 等，映射为 `Outlined` 等价物 |
| `frontend/src/utils/message.tsx` | 新增 `showInfo` 函数（`BindAccount.tsx` 引用但未导出） |

## 验证结果

- `cd frontend && npx tsc --noEmit` — 通过
- `cd frontend && npm run build` — 通过（含 code-split chunk: vendor-charts 546KB）
- `go build ./...` — 所有 backend 包编译通过
- `cd strategy-service && python -m pytest tests/ -x -q` — 通过

## 已知限制

1. **Broker 保证金阈值**: mtapi proto 尚未暴露 `ACCOUNT_MARGIN_SO_CALL/SO_SO`，当前 `FetchBrokerInfo` 返回零值（使用 schema DEFAULT 100.0/50.0）。proto 扩展后取消注释即可。
2. **增量加载**: 仅支持向历史方向（拖拽左边缘），不支持向前预加载。无 cursor 去重——如果两批 kline 有重叠时间戳，轻量图表库的 `setData` 会覆盖。
