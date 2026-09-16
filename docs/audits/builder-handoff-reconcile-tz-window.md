# builder-handoff-reconcile-tz-window — RECONCILE-TZ-WINDOW-1 修复

> v1 @2026-09-16（registry RECONCILE-TZ-WINDOW-1，P2，生产实拍实锤）

## 立项背景

`reconciliation.go:159` `cutoff := Clk.Now().Add(-24*time.Hour)`——`Clk`=real clock（`clock.go:7`），后端容器 `TZ=Asia/Shanghai` → Go time.Local=CST。pgx 对 `timestamp`（非 timestamptz）参数按**本地 wall clock** 编码 → PG 侧 cutoff 值 = CST-24h，比真实 instant 多 +8h → `orders.created_at >= $2`（UTC wall clock 存储）有效窗口仅 ~16h。生产实证：账户 40a7655e ticket 387276098（created_at Sep15 17:06 UTC）自 16.1h 龄起每轮标 ghost + `repaired` 虚增。broker 窗口（`FetchOrderHistory` RPC 传 instant）是真 24h → 窗口不对称复现结构性假 ghost。

## 设计 SSOT

- 修法唯一正确解：**Go 侧 `.UTC()`**。`($2)::timestamptz` 不可行——参数值本身已偏移，cast 只改类型不改值。
- `reconciliation.go:139` `FetchOrderHistory(Clk.Now()...)` 不动——broker RPC 收真实 instant 无时区编码问题（施工方复核确认后可注明）。

## 约束与目标

1. S1：`cutoff := Clk.Now().UTC().Add(-24 * time.Hour)`（一处一行）。
2. S2：pin 测试——断言 reconcile ant 查询 cutoff 为 UTC location（或等价：提取 `reconcileCutoff()` helper 返回 `.UTC()` 值，测试断言 `Location()==time.UTC`；二选一，helper 优先利于测）。
3. S3：同模式排查——全仓 grep `timestamp` 列 vs Go time 参数比较（种子：marketplace/trial.go:85/92、analytics.go ~9 处、settlement.go:50、live_performance.go:247、platform_health_handler.go:54 等 ~58 命中）。**逐站核对列类型是 `timestamp` 还是 `timestamptz`**（`timestamptz` 无 skew）+ 参数来源是否非 UTC。交付 `docs/audits/tz-timestamp-sweep-2026-09.md`：站点/列类型/是否受影响/建议。**只修 reconcile 本站，其余站点报告列清单由决策方分批排期**——禁止顺手全改。
4. 串行，勿部署、勿 push、禁 `--no-verify`；commit 用 `ANT_ROLE=builder git commit` 前缀。

## 对抗证明（mutation）

- M1：删 `.UTC()` → S2 pin 测试 RED → restore → GREEN。
- 验证补强（可选但推荐）：单元测试注入 `Clk` fake clock 返回已知 CST 时刻，断言 cutoff UTC 值正确。

## 验收标准

- [ ] `reconciliation.go:159` 单行修复到位
- [ ] pin 测试存在且 mutation RED→GREEN
- [ ] sweep 报告落盘（列类型逐站判定，非罗列 grep 结果）
- [ ] `go build ./...`、`go test -count=1 ./internal/mthub/`、`go test -race -count=3 ./internal/mthub/`、vet/gofmt/check-lines 0 errors
- [ ] 完工按 D-012 自审 + D-014 末行 `[施工完成:RECONCILE-TZ-WINDOW-1] @<commit-hash>` 自报

## D-013 出件自审记录（决策方）

- 坐标回验：`reconciliation.go:139/:159` 实拍；`Clk` real clock `clock.go:7`；容器 TZ=Asia/Shanghai（`docker compose exec backend env` 实拍）、postgres TZ=UTC（`SHOW timezone`）；created_at UTC wall clock（行值=flag 时刻吻合）。
- 路径可达性：pgx timestamp 编码用 time.Time wall-clock 组件 → CST 编码实锤（16.1h 龄首次复 flag，恰越 16h 阈值而非 24h）。
- 修法排除：`::timestamptz` 改型不改值（参数值已偏）——排除；列改 timestamptz 需 migration+全写入方改造——过重，列为 sweep 报告的长期建议。
- 生命周期：`.UTC()` 只改 wall-clock 表示，instant 不变，broker 窗口语义不受影响。
- 冲突检查：与 DATA-TRUTH-1 S1 意图一致（恢复真 24h 对称窗口）；sweep 只报告不施工，防 scope 蔓延。
- **署名**：最终决策：Devin CLI（[角色:决策终] 激活）
