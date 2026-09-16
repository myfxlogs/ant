# builder-handoff-tz-sweep-affected — TZ-SWEEP-AFFECTED-1 修复

> v1 @2026-09-16（registry TZ-SWEEP-AFFECTED-1，P2，RECONCILE-TZ-WINDOW-1 sweep §2 批次）

## 立项背景

`marketplace/analytics.go:45` `since := time.Now().Add(-interval)`（CST wall clock）驱动 :54/:66/:79/:91/:101/:126/:137/:179/:182 共 ~9 站统计查询，列（wallet_transactions/user_subscriptions/marketplace_strategies.created_at）均为 UTC DEFAULT 写入 → 周期统计窗口起点偏 +8h，少计 8h 内交易。同型：`repository/backtest_run_worker.go:144` `CountRecentStartsByUser(userID, since)` 参数无约束（当前无调用方，休眠 API）。

## 约束与目标

1. S1：`analytics.go` 提取 `analyticsSince(interval time.Duration) time.Time` helper 返回 `time.Now().UTC().Add(-interval)`（复用 RECONCILE-TZ-WINDOW-1 的 `reconcileCutoff()` 模式），:45 单点替换——一处改全站愈。
2. S2：`backtest_run_worker.go` `CountRecentStartsByUser` 函数体首行 `since = since.UTC()` 归一化（比注释强——调用方传任何 zone 都正确）+ 注释注明 timestamp 列语义。
3. S3：pin 测试 `TestAnalyticsSince_IsUTC`（location==UTC + 可仿 reconcile 用固定 zone 断言钟面值）。
4. 串行，勿部署、勿 push、禁 `--no-verify`；commit 用 `ANT_ROLE=builder git commit` 前缀。
5. 完工按 D-012 自审 + D-014 末行 `[施工完成:TZ-SWEEP-AFFECTED-1] @<commit-hash>` 自报。

## 边界/不做

- 不动 refund.go:184/242/247、settlement.go:52/267/305、purchase.go:100 等 INSERT 侧 `time.Now()` 写入——属 TZ-MIXED-ENCODING-1（写入侧混合编码），决策方已补记这些站点进该债的 spike 清单。
- 不动 Go 内存侧比较（coupon.go:37/decay_monitor/publish 缓存——instant 比较无 skew）。
- 不动列类型/不写 migration。

## 对抗证明

- M1：删 `.UTC()`（analyticsSince 内）→ S3 pin 测试 RED → restore → GREEN。
- M2：`CountRecentStartsByUser` 删 `since.UTC()` 归一化 → 若有 pin 测试则 RED（如归一化不可测，commit message 注明覆盖等价理由）。

## 验收标准

- [ ] analyticsSince helper + :45 替换；worker 参数归一化
- [ ] pin 测试 + mutation RED→GREEN
- [ ] build/test/vet/gofmt/check-lines 0 errors；`go test -race -count=3 ./internal/marketplace/ ./internal/repository/`

## D-013 出件自审记录（决策方）

- 坐标回验：`analytics.go:45` `since := time.Now().Add(-interval)` 实拍（驱动 :54/:66/:79/:91/:101/:126/:137/:179/:182）；`backtest_run_worker.go:138-147` `CountRecentStartsByUser` 实拍（`created_at >= $2`）；schema `TIMESTAMP DEFAULT CURRENT_TIMESTAMP`（001_init.up.sql）=UTC 写入。
- 可达性：analytics 9 站全由 :45 单点 since 驱动——单点修法覆盖完整。
- 生命周期：`.UTC()` 同 instant 换钟面，统计窗口恢复真 24h/7d/30d 语义。
- 边界核实：Go 内存 instant 比较（coupon/decay/publish）无 skew 实拍确认；refund/settlement/purchase INSERT 侧 CST 写入属 MIXED 债已另行补记。
- **署名**：最终决策：Devin CLI（[角色:决策终] 激活）
