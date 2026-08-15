# Builder Handoff — SRD-1（策略运行诊断：评估计数器 + 指标捕获 + 诊断 tab）

> 批次名：SRD-1 ｜ 日期：2026-08-15 ｜ 审计方：Claude Code ｜ 施工方：Windsurf
> **设计依据：`docs/spec/strategy-runtime-diagnostics.md` v1.1**（自审修订后版本——施工以 spec 为准，本单只给任务切分与验收；spec §2/§3 的 v1.1 修正标记（R1-R6）是血泪教训，别按直觉改回去）。
> 涉及功能块：`strategy-runtime` + `mql-compiler`（VM builtins）+ `api-gateway`（proto/SSE）+ `frontend`。
> ⚠️ MACD 监控在进行中：后端部署会重启 run（调度自动接力+满窗种子），正常现象，勿以为搞坏了。

---

## 1. 任务分解

### Task S1 — L1 评估计数器（后端）

**捕获点**：`vm_live_handlers.go` 三个 handler（vmHandleBar/vmHandleTick/vmHandleTrade）——**VM 真实执行处**（spec §2 已修正：不是事件到达处）。session 尚未建立（nil early-return）不计。

**存储**：新文件 `session_diag.go`（session_registry.go 已 403 行 🟡，勿再加）：
```go
type sessionDiag struct {
    mu         sync.Mutex
    evalCount  int64
    tickCount  int64
    barCount   int64
    lastEvalAt int64  // unix ms
    windowBars int
    indicators map[string][]decimal  // S2 复用同一结构
    indicatorKeyOrder []string        // 展示序
}
```
ActiveSession 加 `diag *sessionDiag` 字段 + 方法 `RecordEval(kind)` / `RecordWindow(n)` / `SnapshotDiag()`（拷贝快照，防锁外泄）。**锁纪律**：diag.mu 细粒度，绝不复用 session 全局锁（热路径）。

**windowBars 来源**：`live_runner_events.go handleBar` 处 `len(*bars)`（服务端窗口长度，暖机可见性——用户看 53 vs 500 一目了然）。

### Task S2 — L2 指标捕获（VM builtin 层）

**记录点**：`vm_builtin_indicators.go` 各 builtin 返回处——首批覆盖：iMA / iMACD(main+signal) / iRSI / iATR / iBands(上中下)。新文件 `vm_builtin_diag.go`（红线预防）放记录 helper。

**键规范与热路径纪律（R2，违者成本表作废）**：
```go
// VM 字段：lastIndicators map[string]decimal + diagKeyCache map[uint64]string
// 键 = "iMACD[12,26,9].main" 样式；hash(参数序列) → 键 memoize——
// 参数组合对给定策略恒定，首次构建后零分配。
```
- 只记 shift==0 调用；同事件多次调用记最新值。
- VM 侧每事件照记（无负担）；**5s 写入节流放 server 侧**（vmHandleX 尾部读 `VMRunner.LastIndicators()` + `LastOrdersTotal()` → 判距上次写入 >5s 才写 sessionDiag）——tick 洪峰策略的环形写频率被压平。
- `VMRunner.LastOrdersTotal()`（R3：VM 内部值，从这走，不在 L1）。

### Task S3 — proto + SSE 传输（R1 修正版）

```proto
message StrategyDiagnostics {
  int64 eval_count = 1;  int64 tick_count = 2;  int64 bar_count = 3;
  int64 last_eval_at_unix_ms = 4;  int32 window_bars = 5;  int32 orders_total_seen = 6;
  repeated IndicatorSnapshot indicators = 7;
}
message IndicatorSnapshot { string key = 1; string value = 2; int64 at_unix_ms = 3; }
// ActiveStrategy 加 StrategyDiagnostics diagnostics = <下一个可用号>;
```
**传输（勿按直觉改成"随心跳带"——心跳是防闪空的空事件，R1 教训）**：`strategy_active_handlers.go` WatchActiveStrategies 的心跳 select 改为**节拍复用+内容分频**——每 3 个心跳（60s）发携带完整 strategies+diagnostics 的数据事件，其余心跳维持现状空事件。前端既有"跳过空事件"逻辑零改动。诊断快照在**发送时**才 SnapshotDiag()（不逐事件推）。

TS 侧 buf generate 同步（工具链具备）。

### Task S4 — 前端「诊断」tab

行展开加第五个 tab「诊断」（持仓/信号/日志/配置/诊断）——网格 CSS 用 nth-child(1..4) 锁前四，第五个自然顺排，**COL_PCT 与 <style> 生成逻辑零改动**（已验证通用）。内容：
1. **状态徽章**（前端推导，spec §2 状态机）：评估中(绿)/数据饥饿(橙，lastEvalAt>2×周期)/暖机(蓝，windowBars<100)/错误(红，既有 last_error)
2. 计数器行：评估总数 · tick:bar · 最后评估 · 窗口根数 · OrdersTotal；标注"本次运行"（R6：E2 重启归零是 run 语义）
3. 指标 sparkline：每键一条 64 点轻量 SVG（无现成组件则手写 polyline，勿引重库）
4. i18n 新 key ×5 locale；空参数策略显示"无参数"

## 2. 红队自审（逐项打勾）

- [ ] diag.mu 细粒度；SnapshotDiag 返回拷贝（浅拷 values）
- [ ] 热路径零分配：`go test -bench -benchmem` 跑 builtin 记录路径，memoize 后 allocs/op==0
- [ ] 环形写入频率 ≈ 时长/5s（tick 洪峰不随 tick 线性）——测试断言写次数
- [ ] indicators 键数 cap 32（防策略狂调不同参数组合撑爆）
- [ ] 心跳分频后空事件仍占 2/3（前端防闪空回归——既有测试别删）
- [ ] E2 重启 → diag 归零（run 语义，前端标注）——不试图跨 run 累积
- [ ] session_diag.go / vm_builtin_diag.go 新文件（不动 403 行的 session_registry.go 主体）
- [ ] 部署重启 MACD run 后 seeded=500 回归（SEED-GAP 不倒退）

## 3. 复用核对

| 项 | 结论 |
|---|---|
| 心跳机制 | REUSE: WatchActiveStrategies 既有 ticker（改分频，不新建通道） |
| 参数区渲染模式 | 参考 StrategyParamsSection 的表单模式（诊断区是新 UI，无直接复用） |
| 指标计算 | **零新计算**——builtin 返回处捕获（spec 第一原则） |
| 新能力 | NEW：sessionDiag / VM lastIndicators+memoize / proto 消息 / 诊断 tab |

## 4. 门禁

既有全套（go build/test + check-file-lines + tsc + vitest + build）+ `-benchmem` 热路径断言。部署：后端 compose + 前端 docker cp（唯一方式）。

## 5. 验收（审计方）

独立删行复测 + 生产实测：① MACD run 诊断 tab 显示评估中（绿）+ eval_count 增长 + window≈500；② 手动断开场景或等数据间隙 → 徽章转数据饥饿（橙）；③ iMACD 值与**固定 500 根窗口的独立计算**确定性对账（R4：禁用实时 PG 对账——闪断）；④ SSE 每 3 心跳一个数据事件、空心跳仍防闪空；⑤ tick 洪峰下 allocs/op==0。

## 6. 回填纪律

registry SRD 条目状态 + 真实根因/对抗结果；handover changelog；append-only；不自宣告 ✅。
