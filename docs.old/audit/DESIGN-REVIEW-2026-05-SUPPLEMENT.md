# ant 项目深度审查报告

**日期**：2026-05-23
**版本**：ROADMAP v2.0 Completed（M0.0–M6 全部 ✅）
**审查范围**：全仓（backend / frontend / strategy-service / docker / CI）

---

## 1. 架构设计

### 1.1 旧 RiskControl JSONB 未删除（🔴 高）

`strategies.risk_control` 列仍在 repository 中被 INSERT（`strategy_repository.go:51`），`GetRiskControl()`/`SetRiskControl()` 方法仍在使用。M3 建立的 `user_risk_profiles` 表几乎未被引用。两套风控体系并存，数据不一致风险。

**建议**：删除 `strategies.risk_control` 列，所有风控读路径改为 `user_risk_profiles`。

### 1.2 Duck typing 断裂（🟠 中）

`symbol.CanonicalEntry` 与 `symbolsync.SeedCanonicalSymbols` 的入参类型不兼容，`seed_canonical_symbols.go` 被迫绕过 repo 直接写 SQL。

**建议**：统一用 `symbol.CanonicalEntry` 作为 symbol 包唯一导出类型。

### 1.3 BrokerAdapter 接口实现不完整（🟠 中）

MT4/MT5 adapter 的 `Cancel`、`Modify`、`Query` 全部返回 `"not implemented"`。

**建议**：至少返回带中文报错的 `ErrUnsupported`。

### 1.4 MatchingEngine 双重实现绕行限流（🔴 高）

`strategy_executor.go:103` 直接调 `engine.OrderSend`，跳过了 `ExecutionGateway` 的限流/熔断/幂等保护。

**建议**：所有下单统一走 `ExecutionGateway.OrderSend`。

---

## 2. 代码质量与复杂度

### 2.1 超大文件（🔴 高）

| 维度 | 超标数 | 最大违规 |
|------|--------|----------|
| Go 文件 >300 行 | 52 | 722 行 `debate_v2_service.go` |
| TS 文件 >250 行 | 36 | 793 行 `DebatePageV2Steps.tsx` |

baseline.json 已冻结 281 处违规，但 M0.3 承诺的拆分未实际执行。

### 2.2 测试覆盖率极低（🔴 高）

| 包 | 覆盖率 |
|----|--------|
| `service` | 3.2% |
| `connect` | 0.9% |
| `ai` | 10.8% |
| `oms` | 37.5% |
| `errs` | 95.7% |

83 测试函数覆盖 121,469 行代码，整体 < 5%。

### 2.3 SQL 注入风险（🔴 高）

`log_repository.go` 中 5 处 `fmt.Sprintf` 拼接列名到 SQL；`admin_repo_accounts.go` 中 2 处。

```go
// log_repository.go:100
dataQuery := base + fmt.Sprintf(`SELECT * FROM merged ORDER BY created_at DESC LIMIT $3 OFFSET $4`)
```

**建议**：列名走白名单常量，动态排序用 `db.Rebind` + 参数化。

### 2.4 go vet / staticcheck

`go vet` 通过。`staticcheck` 未安装。golangci-lint 配置已创建（`.golangci.yml`）但 CI 中未执行。

---

## 3. 工程化

### 3.1 CI 中 golangci-lint 未运行（🔴 高）

`.github/workflows/ci.yml` backend-lint job 只执行 `go vet`，未调用 `golangci-lint run`。

**建议**：安装 golangci-lint 并执行 `golangci-lint run --new-from-rev=origin/main`。

### 3.2 ESLint 增量检查有缺陷（🟠 中）

CI 中 `eslint $FILES` 会对改动文件报 `max-lines` 规则错误（存量超标 814 行）。

**建议**：增量 lint 关闭 `max-lines`，新增大文件检查单独运行。

### 3.3 无测试覆盖率门禁（🟠 中）

**建议**：CI 加 `go test -coverprofile` + 门禁阈值（初期 15%）。

### 3.4 前端测试仅烟测（🟠 中）

0 个页面级测试，`smoke.test.tsx` 只测了 `<div>Hello, ant</div>`。

---

## 4. 用户体验（前端）

### 4.1 i18n 键严重不均衡（🔴 高）

| 语言 | 总键数 | 差异 |
|------|--------|------|
| zh-cn | ~1,700 | 基准 |
| en | ~1,100 | -600 |
| ja | ~200 | -1500 |
| zh-tw | ~300 | -1400 |
| vi | ~400 | -1300 |

日文/繁中用户会看到大量 fallback 英文甚至中文。

### 4.2 无全局错误边界（🟠 中）

每个页面自行管理 loading/error，API 失败时无统一重试。

### 4.3 SymbolDetection 组件无骨架屏（🟡 低）

---

## 5. API 设计

### 5.1 admin_repo 列名拼接（🔴 高）

`admin_repo_accounts.go` 使用 `fmt.Sprintf` 拼接 SELECT 列名。

**建议**：列名走白名单常量。

---

## 6. 数据模型与迁移

### 6.1 094 migration 有 LIMIT 1（🟡 低）

多策略用户只会迁移第一条 `risk_control` 记录。

### 6.2 无 schema version 表（🟡 低）

无法运行时确认 migration 状态。

### 6.3 缺少迁移循环测试（🟠 中）

无 `.up.sql → .down.sql → .up.sql` 循环验证。

---

## 7. 安全性

### 7.1 sandbox_scan 未挂载（🔴 高）

`strategy-service/app/sandbox_scan.py` 已编写，但 `main.py` 中执行用户代码前未调用 `scan_code()`。

### 7.2 无 API 限流（🟠 中）

`ExecutionGateway` 有并发控制但无全局 rate limit。

### 7.3 无 HTTP 安全头（🟡 低）

缺少 CSP / HSTS / X-Content-Type-Options。

---

## 8. 可观测性

### 8.1 trace_id interceptor 未集成（🔴 高）

`internal/interceptor/trace.go` 已编写但 `server.go` 中未挂载。

### 8.2 errs 包未被使用（🔴 高）

全仓 `MessageCN`/`MessageEN` 零调用。现有错误仍为裸字符串。

### 8.3 zap 日志缺 trace_id（🔴 高）

现有日志未注入 `trace_id` 字段。

### 8.4 无 Prometheus metrics 端点（🟠 中）

---

## 9. 文档与开发者体验

### 9.1 ROADMAP M6 名称与实际不符（🟡 低）

ROADMAP 中 M6 名为"anttrader 退役"，实际内容是部署工具链。

### 9.2 AGENT.md 阶段滞后（🟡 低）

仍显示"进行中 M0.3 复杂度与品质红线"。

### 9.3 缺少 API 文档（🟠 中）

proto 定义全但无文档注释。

---

## 10. 部署与运维

### 10.1 make deploy 等待不可靠（🟠 中）

`make deploy` sleep 5 秒，不等待 docker healthcheck 确认。

### 10.2 备份无还原文档（🟡 低）

有 `backup-db.sh` 无 restore 脚本。

### 10.3 无灰度切量方案（🟠 中）

只有全量 restart。

---

## 优先修复矩阵

| # | 问题 | 严重度 | 预估 |
|---|------|--------|------|
| 1 | trace interceptor 挂载 + 日志注入 trace_id | 🔴 | 3h |
| 2 | sandbox_scan 接入 strategy-service 执行路径 | 🔴 | 2h |
| 3 | golangci-lint 在 CI 中启用 | 🔴 | 1h |
| 4 | SQL 注入修复（log_repo + admin_repo） | 🔴 | 3h |
| 5 | risk_control JSONB 残留删除 | 🔴 | 2h |
| 6 | strategy_executor 统一走 ExecutionGateway | 🔴 | 3h |
| 7 | 前端 i18n en/ja/zh-tw 补齐 | 🔴 | 8h |
| 8 | errs 包替换现有裸字符串错误 | 🟠 | 6h |
| 9 | 测试覆盖率提升到 ≥15% | 🟠 | 20h |
| 10 | BrokerAdapter Cancel/Modify/Query 实现 | 🟠 | 4h |
| 11 | make deploy 改用 healthcheck 等待 | 🟠 | 1h |
| 12 | AGENT.md / ROADMAP 名称校正 | 🟡 | 0.5h |

**总计**：4 条高严重度安全/架空间隙 + 3 条架构未完成项 + 4 条中严重度工程债。
