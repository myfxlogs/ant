# P0 施工 Spec 04：永久防线第 3 层——阻止冒充成功（status 降级 + 前端呈现）

> ADR-0028 §5 永久防线"修正 bug 影响"的闭环。前两层（检测+标记 IsReliable/BlindSpot）已落地，但 status 仍 SUCCEEDED、前端没展示 BlindSpot → 错误结果仍冒充成功骗用户。本任务补第 3 层：**IsReliable=false / invariant 违反时，status 降级 + 前端醒目展示**，让"假成功→诚实失败"真正生效。

## 现状（必读）

- `backend/internal/connect/strategy/status_constants.go`：`StatusSucceeded="SUCCEEDED"` 等（line 8-14）。
- `backend/internal/connect/strategy/backtest_persistence.go:63-64`：`saveBacktestResult` 在 result.Success=true 时设 `StatusSucceeded`。
- `buildBacktestResponse` 已设 `resp.Risk.IsReliable`（防线 B 违反时 false）+ `resp.BlindSpots`（invariant 类：zero_volume_trade / capital_not_conserved / non_positive_price / invalid_side / time_order_violation）。
- 前端：回测结果页**没有**展示 BlindSpot（grep 只命中 i18n 的导入分析 blindSpot，回测结果组件无）。

## 任务

### 后端：status 降级（status_constants + saveBacktestResult）

1. **`status_constants.go`** 加：`StatusDegraded = "DEGRADED"`（注释：执行成功但防线 B 判定结果不可信——介于 SUCCEEDED 与 FAILED 之间）。
2. **`backtest_persistence.go` `saveBacktestResult`**：在设 status 前，检查 `result.BlindSpots` 是否含 **invariant 类**（Id ∈ {zero_volume_trade, capital_not_conserved, non_positive_price, invalid_side, time_order_violation}）。**有则 status=`StatusDegraded`，无则 `StatusSucceeded`**（原逻辑）。
   - 即：line 64 的 `StatusSucceeded` 改为根据 invariant BlindSpot 决定 SUCCEEDED 还是 DEGRADED。
   - 抽个小 helper `hasInvariantBlindSpot(resp) bool` 复用。
3. `BacktestRunsTotal` metric（line 63）用对应 status（DEGRADED 单独计数）。

### 前端：回测结果页醒目呈现 DEGRADED + BlindSpot

4. **回测结果组件**（自行定位：展示回测 status/metrics 的组件，可能在 `frontend/src/components/backtest/` 或 `pages/strategy/...workspace`）：status=`DEGRADED` 时**醒目**展示（红色/警告色 + 文案"回测结果不可信"），不能显示成"成功"。
5. **BlindSpot 展示**：回测结果的 `BlindSpots`（invariant 类）醒目列出（每条 Id + Description），让用户看到"为什么不可信"（如"存在手数<=0的交易"）。
6. i18n：DEGRADED 状态文案 + invariant BlindSpot 描述的中英日越繁中（现有 i18n 体系）。

## 测试（必须带）

- 后端：`saveBacktestResult` 设 status 的单测——invariant BlindSpot 有→DEGRADED，无→SUCCEEDED；`hasInvariantBlindSpot` 单测。
- 前端：DEGRADED 状态渲染 + BlindSpot 列表渲染（组件测试 or 手动验证）。

## 约束

- 后端：最小改动（status_constants 加常量 + saveBacktestResult 改 status 决定 + helper）。不改 invariant 检测逻辑（防线 B 已有）。
- 前端：最小改动（DEGRADED 呈现 + BlindSpot 列表），不重构回测结果页。
- `go build ./...` + `go test ./internal/connect/strategy/` 全绿；前端 `npm run build` 过。
- 对抗验收（不预先公布）。

## 验收

- 后端：有 invariant BlindSpot 的回测 → status=DEGRADED（不再 SUCCEEDED）；无 → SUCCEEDED。
- 前端：DEGRADED 醒目（不冒充成功）+ BlindSpot 列出原因。
- 这是永久防线"修正 bug 影响"的闭环——错误结果不再骗用户。
