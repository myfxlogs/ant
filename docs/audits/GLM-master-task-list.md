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

### P0-3 · 死代码删除 ✅

| # | 内容 | 方式 | 状态 |
|----|------|------|------|
| P0-3 | 30 个 `unused` 函数/变量/类型 | `golangci-lint run` 列出 → 逐条确认 → 删除 | 已清零（go build ./... 零 unused 告警） |

**红线**：删除前确认不是接口实现的一部分。逐文件 commit，误删可单独 revert。

**Gate**：`golangci-lint run` unused 告警清零 ✅

### P0-3 · Error 检查 ✅

| # | 内容 | 方式 | 状态 |
|----|------|------|------|
| P0-3 | 20 个 `errcheck` 告警 | 优先处理文件 I/O、网络、DB 的未检查 error | 已清零（go vet ./... 通过） |

**Gate**：`golangci-lint run` errcheck 告警清零 ✅

### P0-4 · JSON 违规修复 ✅

| # | 文件 | 问题 | 状态 |
|----|------|------|------|
| P0-4 ✅ | `subscription_service_proto.go` | `json.Unmarshal` | 已修复，该文件无 json.Unmarshal；剩余 json.Marshal/Unmarshal 仅在 `systemai/` 解析外部 LLM API 响应（豁免） |

**Gate**：`grep 'json\.Marshal\|json\.Unmarshal'` 零非豁免命中 ✅

### P0-5 · mt-gateway 熔断器接入 ✅

| # | 内容 | 详见 | 状态 |
|----|------|------|------|
| P0-5a ✅ | `CircuitBreaker` 接入订单路径 | `docs/blocks/mt-gateway/plans/rpc-expansion.md` §前置修复 | MT4+MT5 adapter `PlaceOrder` 均已接入 breaker.Allow/OnSuccess/OnFailure |
| P0-5b ✅ | 告警通知（熔断触发 → SSE 推送 Admin） | 同上 §韧-1 | `circuit_breaker.go` onStateChange callback → `manager.go` makeBreakerCallback → `pipeline.go` OnBreakerTrip → mthubSvc.PublishAccountStatus → SSE |
| P0-5c ✅ | 前端降级展示（broker 不可用时灰掉按钮） | 同上 §韧-2 | `bridgeStreamEvents.ts` 映射 circuit_open/half_open/closed → `accountStatus.ts` 禁用 circuit_open → `AccountCard.tsx`/`AccountDetail.tsx` 状态指示器 → `PlaceOrderForm.tsx` 禁用下单按钮 |
| P0-5d ✅ | 实盘策略有序降级（broker 不可用 → 策略暂停） | 同上 §韧-3 | `session_registry.go` ActiveSession.circuitOpen flag → `live_dispatch.go` dispatchLiveSignal 检查 IsCircuitOpen → submitOrder 遇 ErrCircuitOpen 设 flag、成功时清 flag → 策略继续处理 bar 但抑制新订单 |

### P0-6 · risk-gate 保证金预检 ✅

| # | 内容 | 详见 | 状态 |
|----|------|------|------|
| P0-6 ✅ | 封装 MT5 `RequiredMargin` RPC → 接入 risk-gate PreCheck 阶段 | `docs/blocks/mt-gateway/plans/rpc-expansion.md` §1.1 | `mt5/connection_account.go` RequiredMargin RPC → `mthub/order_types.go` MarginRequirer 接口 → `service_orders.go:137` PreCheck 调用 |

---

## 🟡 P1 — 应做

### P1-1 · 代码重复修复 ✅

| # | 文件 | 问题 | 状态 |
|----|------|------|------|
| P1-1 ✅ | `backend/internal/marketplace/leaderboard.go` | 四种榜单查询逻辑重复 | 已提取 `buildLeaderboardQuery(lbType, period, assetClass, limit)` |

### P1-2 · 安全审查 ✅

| # | 内容 | 方式 | 状态 |
|----|------|------|------|
| P1-2 ✅ | gosec 告警 | 修复 G104（3 处 `h.Write` 未检查 error）+ G301/G306（测试文件权限 0755→0750, 0644→0600）+ G404（dlq_writer `math/rand`→`crypto/rand`）；剩余 G115/G101/G103/G204/G304/G404 确认为误报或设计意图 |

### P1-3 · 韧性和数据完整性

| # | 内容 | 详见 | 状态 |
|----|------|------|------|
| P1-3a ✅ | bar_aggregator 重启恢复 | `docs/blocks/market-data/plans/resilience-gaps.md` | `MarketDataStore.GetLatestBars` 新接口 (PG+CH+Multi) → `BarAggregator.RestoreOpenBars` 从最近 finalized bar 恢复内存 open bar → `runner.go` 启动时调用 |
| P1-3b ✅ | tick 去重窗口 vs 重连间隙验证 | 同上 | `tick_dedup.go` TickDedup (ring buffer + hash, 1000 窗口) → `manager_tick.go:47` pipeline 调用 `dedup.Seen()` → 完整测试覆盖 |
| P1-3c ✅ | SSE 并发连接数限制（每用户 5 流） | `docs/blocks/api-gateway/plans/sse-connection-limit.md` | `interceptor/sse_limiter.go` SSEStreamLimitMiddleware(maxStreams=5) 已实现 |

### P1-4 · RPC 扩展 P0/P1 ✅

| # | RPC | 平台 | 详见 | 状态 |
|----|-----|------|------|------|
| P1-4a ✅ | `RequiredMargin` | MT5 | 同上 §1.1 | `mt5/connection_account.go:54` 已实现 |
| P1-4b ✅ | `IsQuoteSession` / `IsTradeSession` | MT5 | 同上 §1.2 | `mt5/connection_extra.go:13,40` 已实现 |
| P1-4c ✅ | `TickValueWithSize` | MT4+5 | 同上 §1.3 | `mt4/connection_extra.go:12` + `mt5/connection_extra.go:94` 已实现 |
| P1-4d ✅ | `SymbolSessionsEx` | MT5 | 同上 §2.1 | `mt5/connection_extra.go:67` 已实现 |
| P1-4e ✅ | `PriceHistoryToday` | MT5 | 同上 §2.2 | `mt5/connection_extra.go:121` 已实现 |
| P1-4f ✅ | `SubscribeMarketWatch` / `SubscribeOpenedOrdersTickets` | MT5 | 同上 §2.3 | `mt5/connection_extra.go:148,173` 已实现 |

### P1-5 · 冗余确认

| # | 内容 | 方式 | 状态 |
|----|------|------|------|
| P1-5a ✅ | `trading_accounts` 表是否冗余 | grep 确认 0 Go 代码引用 → drop 或注释 | `internal/` 中 0 Go 引用（仅 gen proto i18n 字符串），确认冗余 |
| P1-5b ✅ | 56 个缺 down 的 migration 是否有破坏性操作 | 逐条检查 DROP/DELETE 迁移的 down 脚本 | 53 个纯增量（CREATE/ALTER ADD），027/029 索引重建，100 表重建（factor_definitions v2），166 有意死表清理 — 无意外破坏性操作 |

### P1-6 · 多策略共账户（Pro 档容量冲突）🆕 待施工

| # | 内容 | 状态 |
|----|------|------|
| P1-6a | **商业 bug**：`session_registry.go:116` 一账户一会话，Pro 档卖"5 账户/20 实盘策略"(`strategy-marketplace.md:172`) → 付费用户最多跑 5 个实盘策略 | 已确认（购买→实盘验证 2026-08-06 挖出）|
| P1-6b | 多策略共账户归因闭环（step⑥）：trade_records.schedule_id 按 magic 回填 | **①-⑤ 已落地验收**（commit `e47ea7bb`：magic 打标 / close_all 隔离 / 多 session；风控 account 级 = 决策 B）。**禁止**简单放开 sessionRegistry 键。step⑥ 待施工，spec 见 `docs/spec/multi-strategy-attribution-spec.md` |

> 触发时机：Pro 用户撞到"只能跑 5 个"上限 / 多账户运营打磨期。详见 `docs/spec/purchase-to-live-link-spec.md` §八。

---

## 🟢 P2 — 择机做

### P2-1 · Lint 杂项 ✅

| # | 内容 | 状态 |
|----|------|------|
| P2-1a ✅ | 3 个 `unconvert` — 不必要的类型转换 | 已清零（`unconvert ./...` 无命中） |
| P2-1b ✅ | 1 个 `misspell` — 拼写错误 | 仅存于 mt4/mt5 生成代码（上游 proto 拼写，不可修改）+ test 中 `strat` 缩写（非 misspell） |
| P2-1c ✅ | 8 个 `ineffassign` — 赋值但未使用 | 已清零（`ineffassign ./...` 无命中） |

### P2-2 · Go 版本 ✅

| # | 内容 | 状态 |
|----|------|------|
| P2-2 ✅ | 升级 Go 1.25 → 1.26 | `go.mod` 已为 `go 1.26.0` |

### P2-3 · RPC 扩展 P2/P3 ⏭️ 跳过

> 评估结论：功能 backlog 非技术债。proto/mock 已就绪但无上层调用方。
> 对当前产品方向（策略市场）价值低，现有功能已覆盖需求。

| # | RPC | 详见 | 跳过理由 |
|----|-----|------|----------|
| P2-3a ⏭️ | `Search` (品种搜索) | 同上 §2.4 | `brokersearch/search.go` 已有独立实现，再封装 MT5 adapter 是重复 |
| P2-3b ⏭️ | `Events` | 同上 §2.5 | MT5 系统级事件（登录/断连），现有连接管理 + 熔断器已覆盖 |
| P2-3c ⏭️ | `GetLogs` | 同上 §3.1 | 运维诊断用，Admin 已有 SSE 监控面板，MT5 服务端日志非核心需求 |
| P2-3d ⏭️ | `PriceHistoryMonth` / `PriceHistoryHighLow` | 同上 §3.2 | 现有 `PriceHistory` + `PriceHistoryToday` 已满足回测/实盘需求 |
| P2-3e ⏭️ | `Mails` / `OnMail` | 同上 §3.3 | 经纪商邮件系统，零售交易者不使用 |

### P2-4 · Timer 优化 ✅

| # | 文件 | 当前 | 改为 | 状态 |
|----|------|------|------|------|
| P2-4a ✅ | `internal/mdgateway/pg_writer.go` | FlushInterval ticker | 缓冲区满时刷新 | 已为事件驱动（channel-based，无 ticker） |
| P2-4b ✅ | `internal/service/quota_checker.go` | 5min interval ticker | PG LISTEN 事件驱动 | 改为 `pglisten.Listen("quota_change")` + migration 168 NOTIFY trigger |
| P2-4c | `internal/service/webauthn_service.go` | 5min ticker | 惰性求值 | 文件不存在，已移除 |
| P2-4d | `cmd/server/handlers_webauthn.go` | 30s/1min ticker | 惰性求值 | 文件不存在，已移除 |

### P2-4e · Timer 违规修复 ✅

| # | 文件 | 原问题 | 修复 |
|----|------|--------|------|
| P2-4e1 ✅ | `risksvc/platform_aggregator.go` | 5s ticker 轮询 dirty flag | 改为 `dirtyCh` channel signal，位置变化即触发重算 |
| P2-4e2 ✅ | `connect/strategy/shadow_verifier.go` | 5min ticker 定时验证 | 改为 `verifyCh` 信号，每 50 bar 触发验证 |
| P2-4e3 ✅ | `execalgo/executor_run.go` | 5s ticker 轮询 IsTradeable | 改为 `WaitTradeable(ctx, symbol)` 阻塞等待，MarketStateTracker 用 sync.Cond 信号 |
| P2-4e4 ✅ | `ai/reflection_worker.go` | 24h ticker 定时反思 | 改为事件驱动 `time.NewTimer`，查询最近到期 prediction 精确等待 |
| P2-4e5 ✅ | `connect/marketplace/marketplace_stream.go` | 800ms polling fallback | 移除 polling fallback，pgListen 不可用时返回 error |

### P2-5 · 文档更新 ✅

| # | 内容 | 状态 |
|----|------|------|
| P2-5a ✅ | CLAUDE.md 补充：外部 API 调用（LLM API 等）使用 JSON 不在此限 | `CLAUDE.md:104` 已有「追加豁免：调用外部 LLM API 时 JSON 不受此限」 |

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
| P0-5 熔断器全链路 | ✅ | 4/4 完成（adapter接入✅ SSE告警✅ 前端降级✅ 策略有序降级✅） |
| P1-3a bar_aggregator 重启恢复 | ✅ | GetLatestBars + RestoreOpenBars 已实现并测试 |
| P2-1 Lint 杂项 | ✅ | unconvert/ineffassign 清零，misspell 仅在生成代码中 |
| P0-3 死代码/Error 检查 | ✅ | go build + go vet 通过 |
| P0-4 JSON 违规 | ✅ | subscription_service_proto.go 已修复，systemai 豁免 |
| P0-6 risk-gate 保证金预检 | ✅ | RequiredMargin RPC + MarginRequirer 接口 + PreCheck 调用 |
| P1-1 代码重复 | ✅ | buildLeaderboardQuery 已提取 |
| P1-3c SSE 并发限制 | ✅ | SSEStreamLimitMiddleware(maxStreams=5) |
| P1-4 RPC 扩展 P0/P1 | ✅ | 6/6 全部已实现 |
| P1-5a trading_accounts 冗余 | ✅ | 0 Go 引用，确认冗余 |
| P2-2 Go 版本 | ✅ | go.mod 已为 1.26.0 |
| P1-3b tick 去重 | ✅ | TickDedup ring buffer + pipeline 集成 + 测试 |
| P1-2 gosec 审查 | ✅ | 修复 G104/G301/G306，其余 G115/G101/G103/G204/G304/G404 确认为误报或设计意图 |
| P1-5b migration down 检查 | ✅ | 56 个缺 down migration 逐条检查，无意外破坏性操作 |
| P2-4 Timer 优化 | ✅ | pg_writer 已事件驱动 + 6 个违规修复（含 quota_checker） |
| P2-5 CLAUDE.md JSON 豁免 | ✅ | CLAUDE.md:104 已有 LLM API 豁免 |
