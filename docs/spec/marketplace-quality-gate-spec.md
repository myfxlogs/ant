# 施工 Spec：市场发布质量门——三层激活 + DEGRADED 硬阻断

> 市场价值链的下一个断裂点：质量门代码存在但三层全关（gates 表空 + 全部豁免 + 不检查 DEGRADED），导致无回测/假回测策略都能发布。本 spec 激活三层 + 加 DEGRADED 硬阻断。

## 现状（已查实）

- `marketplace/quality.go:98 ValidateBacktestQuality`：检查 sharpe/drawdown/trades/winrate——代码完整。
- `marketplace_quality_gates` 表：**空**（不存在或无数据）→ 阈值全零 → 全部通过。
- `marketplace_quality_waivers` 表：**6 条**（= 全部 6 个 published 策略）→ 全部豁免。
- 发布流程 `publish.go:182 Publish`：调 `ValidateBacktestQuality`，但上面三层全关 = 零拦截。
- **不检查 DEGRADED**：即使回测 status=DEGRADED（假成功），snapshot 里的 metrics 仍能通过质量门。

## 任务

### 第一层：配置质量门阈值（DB 初始化）

在 `backend/migrations/` 新增 migration（`.up.sql`），创建 `marketplace_quality_gates` 表（如果不存在）并插入合理默认值：

| 字段 | 默认值 | 理由 |
|---|---|---|
| enforce_snapshot | true | 必须有回测才能发布 |
| min_total_trades | 10 | 少于 10 笔无统计意义 |
| min_sharpe_ratio | -1.0 | 允许负夏普（差策略也能上，但用户能看到）|
| max_drawdown_pct | 0.80 | 超过 80% 回撤的策略风险极高 |
| min_win_rate | 0.0 | 不卡胜率（用户自己判断）|

**原则：质量门不是筛选"好策略"，是阻断"假数据/无数据"。** 默认宽松，只挡无回测和 DEGRADED。

### 第二层：DEGRADED 硬阻断（代码）

`quality.go ValidateBacktestQuality` 加 DEGRADED 检查：

```go
// 在 Unmarshal snapshot 之后、检查 metrics 之前，加：
// 查关联的 backtest_run status——如果 DEGRADED，硬阻断。
var runStatus string
err = s.pg.QueryRow(ctx,
    "SELECT status FROM backtest_runs WHERE strategy_id = $1 ORDER BY created_at DESC LIMIT 1",
    strategyID,
).Scan(&runStatus)
if err == nil && runStatus == "DEGRADED" {
    return []QualityViolation{{
        Metric: "backtest_status", Actual: "DEGRADED",
        Threshold: "SUCCEEDED (invariant checks must pass)",
    }}, nil
}
```

**DEGRADED 硬阻断不受 waiver 影响**——即使用户有豁免，DEGRADED 回测也不能发布（数据不可信，发布 = 欺诈）。

### 第三层：清理测试豁免（DB 清理）

在 migration 里清理 `marketplace_quality_waivers` 表的测试数据（6 条全删）。Waiver 应该是 Admin 审批后逐条添加，不是批量预置。

## 约束

- 后端 Go + SQL migration。
- 最小改动（quality.go 加一段 DEGRADED 检查 + migration 配置 gates + 清理 waivers）。
- 不改 Publish 主流程（只加质量门检查项）。
- `go build ./...` + `go test ./internal/marketplace/` 全绿。

## 硬性质量要求（不只"能跑"，必须最优）

1. 实现是最优解——DEGRADED 检查是硬阻断（不受 waiver 影响），因为假数据发布 = 欺诈，不可豁免。
2. 代码干净无冗余。
3. 无技术债。
4. 无违规（proto + decimal + 不用 nolint）。
5. 符合第一性原则——质量门默认宽松（不筛好策略），只挡假数据/无数据/DEGRADED。

## 验收

- 无 backtest snapshot 的策略 → 被质量门拦截（EnforceSnapshot=true）。
- DEGRADED 回测的策略 → 被硬阻断（即使有 waiver）。
- 正常 SUCCEEDED 回测 + 10+ trades → 通过质量门。
- `go build` + `go test` 全绿。
