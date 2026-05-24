# BACKLOG（v2）

> 已知缺陷、待办、未立项的工作项。卡片成熟后晋升到 `ROADMAP.md`。

## 真实完成状态（2026-05-24 重新验收）

### ✅ 真正完成

| ID | 内容 | 证据 |
|---|---|---|
| M8.0-1 | 9 张 C2C 平台表 | PG 中存在，migration 110 已执行 |
| M8.0-2 | ETL strategies → platform_strategies | 10 条去重策略已迁入 |
| M8.0-3 | ETL factors | 源数据为空，跳过，合理 |
| M8.0-4 | ETL ai_agents | 0 条，源表为空 |
| M8.0-5 | ETL admin → admins | 1 条 super_admin 已迁入 |
| M8.0-6 | sqlc platform queries | `platform_queries.sql` 存在 |
| M8.1-1 | PlatformScope 接口 | `internal/platform/scope.go` 编译通过 |
| M8.4-1 | service 文件 ≤ 400 行 | 全部 < 200 行 ✅ |
| M9-X1 | mt4client/mt5client → legacy/ | 0 处 import ✅ |
| M9-X2 | kline_service 删除 | 已删除 ✅ |
| M9-X4 | docs.old 删除 | 已删除 ✅ |

### ❌ 纸面完成——需重建

| ID | 内容 | 实际情况 |
|---|---|---|
| M8.1-2 | StrategyTemplateService 合并 platform ∪ user | service 层被清理，需在 v2 server 中重建 |
| M8.1-3 | FactorService/AIAgentService 类似重构 | 同上 |
| M8.1-4 | 删除 per-user seed 逻辑 | 旧 seed 文件已删，但无新 seed 逻辑 |
| M8.2-1 | marketplace proto | proto 文件写了吗？需检查 `proto/ant/v1/marketplace_service.proto` |
| M8.2-2 | MarketplaceService 实现 | 被删除，cmd/server 不包含 |
| M8.2-3 | CopyTradeService | 同上 |
| M8.2-4 | 前端 marketplace 页面 | 前端已全部清空 |
| M8.3-1 | admin JWT scope 独立 | 被删除 |
| M8.3-2 | admin 方法迁到 internal/admin/ | 目录可能不存在 |
| M8.3-3 | 前端 admin 独立登录 | 前端已清空 |
| M8.4-2 | sqlc 覆盖 ≥ 80% | 未验证 |
| M8.4-3 | errs 包替换裸字符串 | 未验证 |
| M8.4-4 | trace_id 全链路 | 未实施 |
| M8.4-5 | golangci-lint < 50 | CI 可能仍报错 |
| M8.4-6 | SQL 注入修复 | 未验证 |
| M8.4-7 | risk_control JSONB 删除 | 未实施 |
| M8.4-8 | i18n 补齐 | i18n 翻译文件存在，但页面已空 |
| M8.4-9 | sandbox_scan 接入 | 未实施 |
| M9-X2 | DROP TABLE kline_data | PG 中旧表可能仍存在 |
| M9-X3 | 删除 kline_service 文件 | 已删除 ✅ |

## M8/M9 重建优先级

### P0 — 阻塞 CI/CD
- M9-X2: 清理 PG 中的旧表（kline_data, tick_data 等）
- M8.4-5: golangci-lint baseline 达标

### P1 — 后端可运行
- M8.1-2/3/4: 在 v2 cmd/server 中重建 service 层
- M8.2-1/2: marketplace proto + service
- M8.3-1/2: admin auth + service

### P2 — 前端可用
- M8.2-4: 前端 marketplace 页面（从 gen/ant/v1/ 直接 import ConnectRPC client）
- M8.3-3: 前端 admin 页面

## 暗坑追踪

| Quirk | 状态 |
|---|---|
| Q-001 ~ Q-015 | 已在 `docs/spec/16-mtapi-quirks-register.md`，adapter 代码中 QUIRK 注释完整 |
