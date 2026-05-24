# BACKLOG（v2）

> 已知缺陷、待办、未立项的工作项。卡片成熟后晋升到 `ROADMAP.md`。

## M8 · 业务层架构订正（C2C 平台 + 用户分层）

> **前置必读**：`docs/adr/0006-platform-shared-vs-user-private.md`
> **目标**：把 ant 从 user-as-tenant（B2B SaaS 错位）订正为"平台共享层 + 用户消费层"（C2C），让 marketplace / 跟单 / admin 鉴权成立。
> **不变量参照**：`02-overview.md` §8 #11 #12 #13

### M8.0 · 数据模型迁移（基础）

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M8.0-1 | ☑ PG migrations：新建 9 张 C2C 平台表 |
| M8.0-2 | ☑ ETL strategies 10 deduplicated → platform_strategies |
| M8.0-3 | ☑ ETL factors (source empty, skipped) |
| M8.0-4 | ☑ ETL ai_agents (deferred to M8.1) |
| M8.0-5 | ☑ ETL admin: super_admin → admins table |
| M8.0-6 | sqlc：为 9 张新表生成 queries | `backend/internal/repository/queries/{platform_*,admins,user_subscriptions,...}.sql` + 生成代码 | `cd backend && make sqlc && go build ./internal/repository/...` |

### M8.1 · PlatformScope + service 重构

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M8.1-1 | 新建 `internal/platform/scope.go`：`PlatformScope` 接口 + 默认实现（始终返回 `'ant'`）| 同左 + `scope_test.go` | `cd backend && go test ./internal/platform/...` |
| M8.1-2 | `StrategyTemplateService` 重构：`List(ctx, userID)` 返回 `platform_strategies ∪ user_strategies`；详情读取按 ID 路由到对应表 | `backend/internal/service/strategy_template_service.go` | 单测：返 list 含 platform_* 行 + user_* 行；handler 不变 |
| M8.1-3 | `FactorService`、`AIAgentService` 类似重构：列表 = platform ∪ user_*；user 修改 platform 项 → 写到 `user_*_overrides` 表（不污染 platform） | 对应 service | 单测：override 后 user List 看到的是 override 值；其他 user 看到的仍是 platform 原值 |
| M8.1-4 | 删除 `seed_default_templates.go` per-user 复制逻辑；改为单次 seed 到 `platform_strategies` | `backend/cmd/seed_strategy_templates/main.go` `backend/internal/server/seed_default_templates.go` | grep 无 `for _, userID := range users` seed 模式 |

### M8.2 · marketplace / 跟单（C2C 核心）

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M8.2-1 | proto：`MarketplaceService` RPC（PublishStrategy / Subscribe / Unsubscribe / ListPublished / ListSubscriptions）| `proto/ant/v1/marketplace_service.proto` | `make proto` |
| M8.2-2 | service：`MarketplaceService` 实现 + handler | `backend/internal/marketplace/{service.go,publisher.go,subscriber.go}` + handler | grpcurl 调通 PublishStrategy → ListPublished 看到 |
| M8.2-3 | service：`CopyTradeService`：订阅 NATS `oms.events.>` 中 publisher 用户的 fill 事件 → 按 ratio 在 subscriber MT 账户发对等单 | `backend/internal/copytrade/service.go` | E2E：A 发布策略 + 真实下单 → B 订阅 → B 账户出现等比例镜像单 |
| M8.2-4 | 前端 marketplace 页：列表 / 详情 / 订阅 / 我的发布 | `frontend/src/pages/marketplace/` | `pnpm tsc --noEmit` + 手测 |

### M8.3 · admin 鉴权独立

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| M8.3-1 | proto：`AdminService` 独立服务 + JWT scope `platform:admin` | `proto/ant/v1/admin_service.proto` + auth middleware | scope 校验单测通过 |
| M8.3-2 | 把当前散在 `service/admin_*.go` 的方法迁到 `internal/admin/`，**禁止**复用 user JWT | `backend/internal/admin/` | `! grep -rE 'users\.role.*admin' backend/internal/` |
| M8.3-3 | 前端 admin 页用独立登录路径 `/admin/login`；user JWT 不能进 admin 页 | `frontend/src/pages/admin/` | 手测 + 单测 |

### M8.4 · 旧债清理（原 M8-X1..X9 中仍有价值的）

| ID | 内容 | 估时 |
|---|---|---|
| M8.4-1 | service 拆分：每个文件 ≤ 400 行（M8.0-M8.3 改造后顺手做）| 5d |
| M8.4-2 | sqlc 覆盖业务表 ≥ 80% | 8d |
| M8.4-3 | errs 包替换所有裸字符串错误（spec/17 §2.3 lint 强约束） | 6d |
| M8.4-4 | trace_id 全链路（含 SSE）| 3d |
| M8.4-5 | golangci-lint baseline 削减到 < 50 | 4d |
| M8.4-6 | SQL 注入修复（log_repo + admin_repo）| 3d |
| M8.4-7 | risk_control JSONB 残留删除 | 2d |
| M8.4-8 | i18n en/ja/zh-tw 前端补齐 | 8d |
| M8.4-9 | sandbox_scan 接入 strategy-service 沙箱（M5 残留）| 2d |

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

## 设计复查（2026-05-23 第二轮）— 行业标杆决策已落地

> 全部已按行业最佳实践决策并落实到 spec/ROADMAP/ADR；本表保留作历史档案与决策追溯依据。

| ID | 类别 | 决策 | 行业参照 | 落地位置 |
|---|---|---|---|---|
| RV-C1 | 作用域 | **改 per-broker endpoint**；认证错误走单独 `auth_failed` 路径不入 breaker | Netflix Hystrix / resilience4j 故障隔离粒度 = 网络资源 | spec/11 §11；ADR-0005 §2.1 |
| RV-C2 | 哲学 | **行情"少不丢"**：仅 hard-reject (bid≥ask、非正、parse 失败)；5σ/gap/skew 改 metric 不丢 | LMAX / Bloomberg B-PIPE / Refinitiv Elektron | spec/11 §5.1-5.2 |
| RV-C3 | 取证 | **保留 100-window dedup**：成本 ~50ns/tick，false-positive ≈ 0；anttrader 历史已验证有 broker 重发 | TCP dup-ACK 检测同理 | spec/11 §6（保留原设计）|
| RV-C4 | 哲学 | **删除 `/livez/account/{id}`**；改 Prom Gauge `mt_account_connected` + Grafana alert | k8s liveness 仅进程级哲学 | 03-data-flow §5.1；spec/15 |
| RV-C5 | 简化 | **保留 size + age 双触发**：~10 行代码，运维好处明确（防 fsync 拖累、replay 顺序）| logrotate / journald / loki 标准 | spec/11 §10.1（保留原设计）|
| RV-C6 | 完整性 | **新增 M7.6-7 卡**：tests/e2e/telemetry_test.go 校验 metric 白名单全部暴露 | Prometheus community 推荐 | ROADMAP M7.6-7 |
| RV-B5 | 安全 | **新 spec/17 + M7.0-9 卡**：AES-256-GCM + HKDF + 版本化 KEK；接口形态可平滑迁 Vault transit | HashiCorp Vault transit / AWS KMS envelope | spec/17 §1；ROADMAP M7.0-9 |
| RV-B6 | 错误体系 | **复用现有 `backend/internal/errs/`** + 补充 `ToConnectError()` 映射 + lint 强约束 | Go std errors + Connect.Error map | spec/17 §2 |
| RV-B7 | 文档同步 | **M7.0-7 验收附带**更新 spec/10 §5 的 import 示例为 v2 路径 | 单点修复 | ROADMAP M7.0-7 |

## 文档债

- [ ] `docs/spec/dep-allowlist.md` 依赖白名单（M8 立）
- [ ] `docs/spec/20-factorsvc.md` factorsvc 详细规范（M7.3 完成后回填）
- [ ] `docs/spec/21-quantengine.md` quantengine 详细规范（M7.4 完成后回填）
- [ ] `docs/spec/22-oms.md` 订单状态机（M8 立）
- [ ] `docs/runbook/incidents-general.md` 通用故障手册（M8）
