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

### ✅ M11 已重建完成 (2026-05-26 验收)

| ID | 内容 | 实际情况 |
|---|---|---|
| M8.1-2 | StrategyTemplateService 合并 platform ∪ user | ✅ `internal/service/strategy_template_service.go` 已实现，main.go 已注入 |
| M8.1-3 | FactorService/AIAgentService 类似重构 | ✅ `internal/service/` + `internal/connect/ai_handler.go` 已重建 |
| M8.1-4 | 删除 per-user seed 逻辑 | ✅ 新 seed 逻辑已在 strategy_svc 中 |
| M8.2-1 | marketplace proto | ✅ `proto/ant/v1/marketplace_service.proto` 已存在 |
| M8.2-2 | MarketplaceService 实现 | ✅ `internal/marketplace/service.go` PG-backed 实现，main.go 已注入 |
| M8.2-3 | CopyTradeService | ✅ 已合入 marketplace subscribe 流程 |
| M8.2-4 | 前端 marketplace 页面 | ✅ `frontend/src/pages/marketplace/Marketplace.tsx` 93 行，ConnectRPC client |
| M8.3-1 | admin JWT scope 独立 | ✅ `internal/interceptor/auth.go` JWT scope 校验 |
| M8.3-2 | admin 方法迁到 internal/admin/ | ✅ admin handlers 全部在 `internal/connect/admin_*_handler.go`，main.go 已注入 |
| M8.3-3 | 前端 admin 独立登录 | ✅ `frontend/src/pages/admin/` 8 页面 2152 行，含 Dashboard/UserManagement/TradingMonitor 等 |
| M8.4-5 | golangci-lint < 50 | ✅ 0 issues, clean exit (2026-05-26) |
| M9-X2 | 清理旧表 + 死代码 | ✅ `kline_repository.go` 已删除，PG 旧表已清理 |

## M8/M9 重建优先级

### P0 — ✅ 已完成 (2026-05-26)
- M9-X2: 清理 PG 中的旧表（kline_data, tick_data 等）✅
- M8.4-5: golangci-lint baseline 达标 ✅ 0 issues

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

## M10 新增待办（已晋升到 ROADMAP M10-BASE）

> **2026-05-25**：以下所有卡片已晋升到 `docs/plan/ROADMAP.md` M10-BASE Phase B/C/D/E/F。
> M11 金融架构设计（`docs/金融架构改造-M11路线图-2026-05-25.md`）的架构决策已融合到 M10-BASE 各卡片中。
> BACKLOG 中保留此节仅作历史参考；以后内容以 ROADMAP 为准。

### 晋升映射

| BACKLOG ID | ROADMAP ID | Phase |
|---|---|---|
| M10-ORDER-1~4 | M10-BASE-B1~B6 | Phase B (OMS + 幂等 + Event Ledger) |
| M10-RISK-1~3 | M10-BASE-C1~C6 | Phase C (风控引擎重构) |
| M10-BACKTEST-1~4 | M10-BASE-D4~D6 | Phase D (回测引擎 + costsvc) |
| M10-PAPER-1~3 | M10-BASE-D1~D3 | Phase D (回测引擎 + costsvc) |
| M10-AI-1~6 | M10-BASE-E1~E6 | Phase E (AI 策略质量门控) |
| M10-BAR-1~4 | M10-BASE-F1~F3 | Phase F (数据质量升级) |
| M10-SLO-1~3 + OBS-1~2 | M10-BASE-F4~F7 | Phase F (数据质量升级 + SRE) |
| M8.1-2/3/4, M8.2-1/2, M8.3-1/2 | M10-BASE 各 Phase | 按优先级分配 |

### 原卡片内容（历史参考）

<details>
<summary>点击展开原 BACKLOG M10 卡片</summary>

（内容已迁移，见 ROADMAP.md M10-BASE 节。此处省略。）
</details>
