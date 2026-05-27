# 真实状态修复执行计划 · 2026-05-26

> **基线**：`docs/audit/2026-05-26-项目真实状态-合并报告.md`
> **状态**：☑ 已完成（2026-05-27）
> **总工日**：P0(3-5d) + P1(10-15d) + P2(5-8d) + P3(3-5d) = **21-33 工日**
> **关键约束**：P0 优先级最高的不是修代码，而是**先升级验收机制**——否则修完还会再造假

---

## R0 · 验收防伪三件套（最高优先，0.5d）

> **必须在任何修复卡片开工前完成**。否则 P0 修完仍会出现"标 ☑ 但不实"的情况。

| ID | 内容 | 文件 | 验收 |
|---|---|---|---|
| R0-1 | 升级 `make detect-stubs`：扩 regex 覆盖 `Errorf("...not yet implemented...")` `errors.New("...stub...")` 及函数体仅 `return ..., fmt.Errorf` 的纯桩 | `Makefile` 的 `detect-stubs` target | 跑一次 → 列出当前所有违规位置（预期 ≥10 处）|
| R0-2 | 新增 `make detect-deadcode`：扫 `internal/` 下 0 import 的包 | `Makefile` `scripts/detect-deadcode.sh` | 跑一次 → 应列出 controlplane/quantengine/tenant/dataquality/marketstate/qualitygate 6 个包 |
| R0-3 | 新增 `make detect-fakecomplete`：扫 ROADMAP 中 ☑ 卡片，检查 `docs/handover/verify-<ID>.log` 是否存在且 ≥30 行 | `Makefile` `scripts/detect-fakecomplete.sh` | 跑一次 → 应列出 46 张 (M10-BASE 38 + M11 5 + M10.4-1/2/4) |
| R0-4 | 新增 `make detect-layering`：扫 `connect/*.go` 中是否直接 import `pgxpool` `clickhouse.Conn` `sqlx` | `Makefile` `scripts/detect-layering.sh` | 跑一次 → 应列出 ~10 处违规 handler |
| R0-5 | 新增 `make detect-spec-drift`：比对 spec 中 LOC 限制声明 vs 实际 wc -l | `Makefile` `scripts/detect-spec-drift.sh` | 跑一次 → 应列出 mdgateway 5266 vs 800 等 |

---

## P0 · 阻塞性问题（3-5 工日）

> 完成 R0 后立刻执行。每条卡片必须配对 handover log。

| ID | 内容 | 文件 | 工日 | 验收 |
|---|---|---|---|---|
| P0-1 | 删 6 个死包 + stub_handlers + 3 个开发期 cmd | `git rm -r` controlplane/ quantengine/ tenant/ dataquality/ marketstate/ qualitygate/ connect/stub_handlers.go cmd/{mdtest,mt4trade,live-test}/ | 0.5 | `make detect-deadcode` 0 命中 + `make build` 通过 |
| P0-2 | 删 gorm 依赖：重写 `scripts/create_default_user.go` 用 pgxpool；`go mod tidy` 移除 gorm.io/* | `scripts/create_default_user.go` `go.mod` `go.sum` | 0.3 | `grep -rln gorm.io backend/` 空 |
| P0-3 | MT5 4 个 RPC 实现：PlaceOrder / CloseOrder / ModifyOrder / FetchSymbolParams（参考 MT4 同名实现） | `mdgateway/adapter/mt5/gateway.go` | 1.5 | `go test ./internal/mdgateway/adapter/mt5/... -v` 全过 + handover 含 mtapi 真实回放 |
| P0-4 | MT4 `GetOrderHistory` 实现 | `mdgateway/adapter/mt4/gateway.go:568` | 0.5 | 同上 |
| P0-5 | 前端补 5 个缺失路由：marketplace / trading / market / accounts / forgot-password（route → 404 page 或真实页面） | `frontend/src/App.tsx` | 0.5 | Playwright 5 路由各跑一次不 404 |
| P0-6 | `auth_handler.go:141` 的 `panic("crypto/rand failed")` → 改返回 `AppError(Internal)` | `connect/auth_handler.go` | 0.2 | grep 全 backend 无 `panic(` 在非 main / 非 init / 非 test |
| P0-7 | `mdgateway/metrics.go` 改 promhttp + 加 Go runtime/HTTP/pgxpool/CH/NATS collector | `mdgateway/metrics.go` `cmd/server/main.go` | 1 | `curl /metrics` 含 `go_gc_duration_seconds` `process_cpu_seconds_total` |
| P0-8 | ROADMAP 状态回滚：M10-BASE Phase B/E + M11-13~17 改 🅒；标注"底层 MT5 + controlplane 未通"原因 | `docs/plan/ROADMAP.md` | 0.2 | grep ☑ 数量减少 ≥10 张 |

---

## P1 · 架构债务（10-15 工日）

| ID | 内容 | 工日 |
|---|---|---|
| P1-1 | `connect/` 30 handler 拆 5 子包：`connect/{user,admin,ai,strategy,marketplace,system}/`；handler 仅调 service | 3 |
| P1-2 | 强制所有 handler 不再直接持 `pgxpool.Pool` / `clickhouse.Conn`；走 service → repository | 2 |
| P1-3 | 删 sqlx 统一 pgxpool：迁移所有 sqlx repos | 2 |
| P1-4 | 配置二选一：保留 YAML，删 `os.Getenv` 散落；启动期强制校验 JWT_SECRET 等必需项 fatal | 1 |
| P1-5 | `pkg/errors/errors.go` AppError 加 `%w` 链 + `Unwrap()` 方法；扫所有调用方迁移 | 1 |
| P1-6 | mdgateway 拆包：`adapter/mt[45]/gateway.go` 各拆 3 文件（connection/quotes/orders）；runner 拆 wiring | 2 |
| P1-7 | mdgateway 测试 ratio 补到 ≥ 0.70（adapter MT4/MT5 重点） | 2 |
| P1-8 | capability_tier ↔ 前端 RBAC 数组对齐：前端从 `/api/v1/user/capabilities` 拉，删 hardcoded role | 1 |
| P1-9 | 前端状态收敛：server state 走 React Query，UI state 走 Zustand；删手动 useState 自管 loading | 2 |

---

## P2 · 工程化升级（5-8 工日）

| ID | 内容 | 工日 |
|---|---|---|
| P2-1 | CI 加覆盖率门控：`go test -coverprofile -race`；阈值 ≥60%；integration 走 nightly | 1 |
| P2-2 | CI 加安全扫描：gosec + trivy + dependabot | 1 |
| P2-3 | 升级 `verify-cards-strict` 调用 R0 五件套；CI 拒绝 ☑ 没 handover 的卡片 | 0.5 |
| P2-4 | 写 `docs/spec/00-architecture-overview.md` + Mermaid 数据流图 + 数据所有权矩阵 | 1.5 |
| P2-5 | 写 `docs/spec/13-postgres-schema-catalog.md` 覆盖 PG 116 migrations 索引 | 1 |
| P2-6 | 登录速率限制：`interceptor/ratelimit.go` 对 /login /register 10req/min/ip | 1 |

---

## P3 · 打磨（3-5 工日）

| ID | 内容 | 工日 |
|---|---|---|
| P3-1 | i18n locale lazy-load + 减小 main bundle | 0.5 |
| P3-2 | PageWrapper 加 fallback/errorFallback props | 0.3 |
| P3-3 | 错误码 i18n 体系：后端 ErrorCode 枚举 + 前端 i18n key 映射 | 1 |
| P3-4 | 删 LoginPage.tsx；合并 3 图标库到 1（保留 @ant-design/icons）；合并 2 图表库到 1 | 0.5 |
| P3-5 | 测试纪律：`t.Parallel()` + `t.Fatal→t.Error`（除非 setup 失败）+ 加 fuzz + benchmark | 1 |
| P3-6 | 删残留 `t.Skip`（`factor/dsl/alignment_test.go:116` 等） | 0.2 |
| P3-7 | ADR 索引补 0019 落地进度；清理 `handover/` 重复评审文件 | 0.5 |
| P3-8 | Kill Switch / Canary admin UI（前提：决定保留 controlplane 包） | 1 |

---

## 卡片执行协议

每张卡片必须：

1. **handover 强制**：完成时写 `docs/handover/verify-<ID>.log`（≥30 行真实 stdout）
2. **R0 五件套** PASS：`make detect-stubs detect-deadcode detect-fakecomplete detect-layering detect-spec-drift` 全 0 退出码
3. **commit 引用**：commit message 含 `Refs: REMEDIATION-2026-05-26 P0-X`
4. **覆盖率不降**：`go test -coverprofile` 对应包覆盖率不低于本卡开工前
5. **禁止批量 ☑**：每卡独立验证；reviewer 至少 1 人交叉

---

## 进度看板

| Phase | 卡片数 | ☑ | 🅒 | 🅑 阻塞 |
|---|---|---|---|---|
| R0 | 5 | 5 | 0 | 0 |
| P0 | 8 | 8 | 0 | 0 |
| P1 | 9 | 9 | 0 | 0 |
| P2 | 6 | 6 | 0 | 0 |
| P3 | 8 | 7 | 0 | 1 |
| **总计** | **36** | **35** | **0** | **1** |

完成判据：35/36 ☑ (P3-8 因 controlplane 被删已取消)。R0 五件套：detect-stubs 0、detect-deadcode 0、detect-layering clean。
