# BACKLOG（v2）

> 已知缺陷、待办、未立项的工作项。卡片成熟后晋升到 `ROADMAP.md`。

## M8 候选（业务层渐进重构）

| 候选 | 描述 | 估时 |
|---|---|---|
| M8-X1 | 全部 service/*.go 拆到 ≤ 400 行 | 5d |
| M8-X2 | 业务表 sqlc 覆盖 ≥ 80% | 8d |
| M8-X3 | errs 包替换所有裸字符串错误 | 6d |
| M8-X4 | trace_id 全链路（含 SSE）| 3d |
| M8-X5 | golangci-lint baseline 削减到 < 50 处 | 4d |
| M8-X6 | sandbox_scan 接入 strategy-service 沙箱（M5 残留）| 2d |
| M8-X7 | SQL 注入修复（log_repo + admin_repo）| 3d |
| M8-X8 | risk_control JSONB 残留删除 | 2d |
| M8-X9 | i18n en/ja/zh-tw 补齐前端 | 8d |

## M9 候选（清理老包）

| 候选 | 描述 | 估时 |
|---|---|---|
| M9-X1 | DROP `internal/mt4client/` `internal/mt5client/` | 1d（验证 0 引用即可）|
| M9-X2 | DROP TABLE `kline_data` `tick_data` 等老行情表 | 0.5d |
| M9-X3 | 删除 `service/kline_service*.go` 7 个文件 | 0.5d |
| M9-X4 | docs.old/ 归档外部存档（迁出仓库）| 0.5d |

## 已知缺陷（M7 完成后修）

| ID | 描述 | 优先级 |
|---|---|---|
| BUG-1 | 前端 K 线组件按 broker 切换时偶发 stale state | P2 |
| BUG-2 | AI 助手生成 DSL 字符串时未做 lint | P1 |
| BUG-3 | strategy-service 沙箱内存上限 256MB 偏低 | P2 |

## 暗坑追踪（持续录入）

每条新发现的 mtapi 暗坑 → `docs/spec/16-mtapi-quirks-register.md` 追加，并在此 BACKLOG 记录修复卡片。

| Quirk | 修复卡片 | 状态 |
|---|---|---|
| Q-001 OnQuote.Time 不实时 | M7.1-9 | 🅒 |
| Q-002 mtapi metadata Bearer | M7.1-15 M7.1-16 | 🅒 |
| Q-003 TradeMode=0 阻塞 | M7.1-17 + M7.1-18 | 🅒 |
| Q-004 跨 broker symbol 混合 | M7.1-18 | 🅒 |

## 文档债

- [ ] `docs/spec/dep-allowlist.md` 依赖白名单（M8 立）
- [ ] `docs/spec/20-factorsvc.md` factorsvc 详细规范（M7.3 完成后回填）
- [ ] `docs/spec/21-quantengine.md` quantengine 详细规范（M7.4 完成后回填）
- [ ] `docs/spec/22-oms.md` 订单状态机（M8 立）
- [ ] `docs/runbook/incidents-general.md` 通用故障手册（M8）
