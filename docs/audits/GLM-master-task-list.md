# GLM 施工总清单

> 来源：四层地基审计 + strategy-marketplace Phase 1-5
> 按优先级排序，含文件路径和验收标准。
>
> **经营方向**（详见 `docs/roadmaps/business-direction.md`）：
> - 产品 = 策略市场，对标 MQL5 Market。核心差异：代码不出平台、实盘战绩公开、AI 迭代。
> - 服务 MQL 开发者（供给侧）+ 零售交易者（需求侧）。不服务量化机构。
> - Year 1 聚焦 MT 生态，IR 层保持语言中性（为加密市场留后路）。
> - 所有施工决策对齐一个目标：**marketplace 上线 → 产生第一笔收入。**

---

## 🔴 P0 — 阻塞项

### P0-1 · 文件超红线拆分

| # | 文件 | 行数 | 拆为 |
|----|------|------|------|
| P0-1a | `backend/cmd/server/handlers.go` | 462 | ✅ `handlers_marketplace.go` + `handlers_user.go` 已拆分 |
| P0-1b | `backend/internal/service/account_service.go` | 452 | ✅ `account_crud.go` + `account_sync_service.go` 已拆分 |
| P0-1c | `frontend/src/.../AutoGeneratePanel.tsx` | 433 | ✅ 已拆为 `AutoGeneratePanel.tsx` + `AutoGenerateProgress.tsx` + `AutoGenerateResult.tsx` |
| P0-1d | `backend/internal/connect/user/deposit_handler.go` | 406 | ✅ 已拆为 `deposit_handler.go` (240) + `sweep_handler.go` (186) |

**Gate**：`go run ./tools/check-file-lines --strict` 零 ERROR ✅ Phase 1-4 文件拆分全部完成

### P0-2 · 忘记密码前端（阻塞上线）✅

| # | 内容 | 详见 |
|----|------|------|
| P0-2a ✅ | `frontend/src/pages/auth/ForgotPassword.tsx` | 邮箱/MT验证/管理员三选一 Tab |
| P0-2b ✅ | `frontend/src/pages/auth/ResetPassword.tsx` | 输入新密码 → 调 ResetPassword RPC |
| P0-2c ✅ | MT 凭据验证（VerifyMTIdentity RPC + handler + rate limit） | `docs/blocks/account-mgmt/plans/mt-password-reset.md` |

### P0-3 · 死代码删除

| # | 内容 | 方式 |
|----|------|------|
| P0-2 | 30 个 `unused` 函数/变量/类型 | `golangci-lint run` 列出 → 逐条确认 → 删除 |

**红线**：删除前确认不是接口实现的一部分。逐文件 commit，误删可单独 revert。

**Gate**：`golangci-lint run` unused 告警清零

### P0-3 · Error 检查

| # | 内容 | 方式 |
|----|------|------|
| P0-3 | 20 个 `errcheck` 告警 | 优先处理文件 I/O、网络、DB 的未检查 error |

**Gate**：`golangci-lint run` errcheck 告警清零

### P0-4 · JSON 违规修复

| # | 文件 | 问题 | 方式 |
|----|------|------|------|
| P0-4 | `backend/internal/service/subscription_service_proto.go:187` | `json.Unmarshal` | 改用 proto message |

**Gate**：`grep 'json\.Marshal\|json\.Unmarshal'` 零非豁免命中

### P0-5 · mt-gateway 熔断器接入

| # | 内容 | 详见 |
|----|------|------|
| P0-5a | `CircuitBreaker` 接入订单路径 | `docs/blocks/mt-gateway/plans/rpc-expansion.md` §前置修复 |
| P0-5b | 告警通知（熔断触发 → SSE 推送 Admin） | 同上 §韧-1 |
| P0-5c | 前端降级展示（broker 不可用时灰掉按钮） | 同上 §韧-2 |
| P0-5d | 实盘策略有序降级（broker 不可用 → 策略暂停） | 同上 §韧-3 |

### P0-6 · risk-gate 保证金预检

| # | 内容 | 详见 |
|----|------|------|
| P0-6 | 封装 MT5 `RequiredMargin` RPC → 接入 risk-gate PreCheck 阶段 | `docs/blocks/mt-gateway/plans/rpc-expansion.md` §1.1 |

---

## 🟡 P1 — 应做

### P1-1 · 代码重复修复

| # | 文件 | 问题 | 方式 |
|----|------|------|------|
| P1-1 | `backend/internal/marketplace/leaderboard.go` | 四种榜单查询逻辑重复 | 提取 `buildLeaderboardQuery(type, period)` |

### P1-2 · 安全审查

| # | 内容 | 方式 |
|----|------|------|
| P1-2 | 5 个 `gosec` 告警 | 逐条审查，确认误报或修复 |

### P1-3 · 韧性和数据完整性

| # | 内容 | 详见 |
|----|------|------|
| P1-3a | bar_aggregator 重启恢复 | `docs/blocks/market-data/plans/resilience-gaps.md` |
| P1-3b | tick 去重窗口 vs 重连间隙验证 | 同上 |
| P1-3c | SSE 并发连接数限制（每用户 5 流） | `docs/blocks/api-gateway/plans/sse-connection-limit.md` |

### P1-4 · RPC 扩展 P0/P1

| # | RPC | 平台 | 详见 |
|----|-----|------|------|
| P1-4a | `RequiredMargin` | MT5 | `docs/blocks/mt-gateway/plans/rpc-expansion.md` §1.1 |
| P1-4b | `IsQuoteSession` / `IsTradeSession` | MT5 | 同上 §1.2 |
| P1-4c | `TickValueWithSize` | MT4+5 | 同上 §1.3 |
| P1-4d | `SymbolSessionsEx` | MT5 | 同上 §2.1 |
| P1-4e | `PriceHistoryToday` | MT5 | 同上 §2.2 |
| P1-4f | `SubscribeMarketWatch` / `SubscribeOpenedOrdersTickets` | MT5 | 同上 §2.3 |

### P1-5 · 冗余确认

| # | 内容 | 方式 |
|----|------|------|
| P1-5a | `trading_accounts` 表是否冗余 | grep 确认 0 Go 代码引用 → drop 或注释 |
| P1-5b | 55 个缺 down 的 migration 是否有破坏性操作 | 逐条检查 DROP/DELETE 迁移的 down 脚本 |

---

## 🟢 P2 — 择机做

### P2-1 · Lint 杂项

| # | 内容 |
|----|------|
| P2-1a | 3 个 `unconvert` — 不必要的类型转换 |
| P2-1b | 1 个 `misspell` — 拼写错误 |
| P2-1c | 8 个 `ineffassign` — 赋值但未使用 |

### P2-2 · Go 版本

| # | 内容 |
|----|------|
| P2-2 | 升级 Go 1.25 → 1.26 |

### P2-3 · RPC 扩展 P2/P3

| # | RPC | 详见 |
|----|-----|------|
| P2-3a | `Search` (品种搜索) | `docs/blocks/mt-gateway/plans/rpc-expansion.md` §2.4 |
| P2-3b | `Events` | 同上 §2.5 |
| P2-3c | `GetLogs` | 同上 §3.1 |
| P2-3d | `PriceHistoryMonth` / `PriceHistoryHighLow` | 同上 §3.2 |
| P2-3e | `Mails` / `OnMail` | 同上 §3.3 |

### P2-4 · Timer 优化（择机）

| # | 文件 | 当前 | 改为 |
|----|------|------|------|
| P2-4a | `internal/mdgateway/pg_writer.go` | FlushInterval ticker | 缓冲区满时刷新 |
| P2-4b | `internal/service/quota_checker.go` | interval ticker | 惰性求值 |
| P2-4c | `internal/service/webauthn_service.go` | 5min ticker | 惰性求值 |
| P2-4d | `cmd/server/handlers_webauthn.go` | 30s/1min ticker | 惰性求值 |

### P2-5 · 文档更新

| # | 内容 |
|----|------|
| P2-5a | CLAUDE.md 补充：外部 API 调用（LLM API 等）使用 JSON 不在此限 |

---

## 📋 施工约束（全局）

- 每个任务做完跑对应的 Gate，通过再进下一个
- 禁止 `//nolint`、`# noqa`、`// @ts-ignore`
- 禁止新增 timer/ticker/polling
- 禁止 JSON 序列化（外部 API 除外）
- 价格/金额禁止 float64
- 文件大小：Go ≤300 行（软性）/ ≤450 行（硬性红线），TS ≤250/375

---

## 施工进度

| Phase | 状态 | 备注 |
|-------|------|------|
| 施工前清理（下线 copytrade） | ✅ | migration 212 已执行 |
| Phase 1 · 信任基础设施 | ✅ | 实盘/质量门槛/验证/风险声明 + walkforward 升级 |
| Phase 2 · AI 策略供给 | ✅ | 批量生成 + 参数模板 + 提供者面板 |
| Phase 3 · 增长引擎 | ✅ | 排行榜/试用/对比/通知/分享 |
| Phase 4 · 平台运营 | ✅ | Admin 管理/退款/收入/优惠券 |
| Phase 5 · AI 迭代 + 增值 | ✅ | 衰减检测/策略优化/捆绑包/阶梯费率 已实现并跑通 |
| SEO S1-S4 | ✅ | Seo.tsx + 关键词落地页 + JSON-LD + sitemap |
| 用户系统加固 | ✅ | 5/5 完成（密码强度✅ Token撤销✅ 密码找回✅ MT验证✅ 前端页面✅） |
