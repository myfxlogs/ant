# 施工交接：Tier 1 Launch 批（3 项，已审计方独立复核为真）

> **审计方定论 2026-08-13，🟦待施工。** 每项诊断已经审计方独立读码复核（非照 agent 结论）。来源 `docs/audits/launch-readiness-2026-08-13.md` + 止血扫描。
>
> **铁律**：每项用 `scripts/verify-adversarial.sh` 自验"删行必红" + 回填 registry 🟦→✅ + commit（勿只部署不提交）+ 不自行宣告完成。部署 `docker compose build backend && docker compose up -d backend`（后端）/ `docker cp frontend/dist/. alphaforge-frontend:/usr/share/nginx/html/ && docker exec alphaforge-frontend nginx -s reload`（前端）。

---

## LAUNCH-G1：公开策略详情页未登录 401（发现→转化漏斗断）

- **诊断（审计方已复核）**：`AppRoutes.tsx:188` `<Route path="/strategy/:strategyId" element={wrap(<StrategySharePage />)} />`——`wrap`（:78）是 layout **非 auth guard**，路由公开可达。但 `StrategySharePage.tsx:7,23` 用 **authed `marketplaceClient.getStrategyPublicInfo`** → 未登录访客 API 401 → 公开策略详情空白。这是公开发现→购买的漏斗入口。
- **修法**：`StrategySharePage` 改用**公开 client**调 `getStrategyPublicInfo`（参考 `/share/:token` 已用的 `sharePublicClient` 模式）。确认 `getStrategyPublicInfo` 是 public RPC（名字带 Public，proto 里应无 auth interceptor）。
- **对抗证明**：删 public client 接线（改回 authed）→ 未登录态下组件渲染触发 401（RED）；public client → 渲染成功（GREEN）。

## MDGATEWAY-1：FetchAccountInfo 吞响应错误 → 误判"只读账户"

- **诊断（审计方已复核）**：`backend/internal/mdgateway/adapter/mt4/connection_account.go:32` FetchAccountInfo 调 `AccountSummary`，:33 查 gRPC err，:36 查 `resp.GetResult()==nil`，**全程不查 `resp.GetError()`**。mtapi 返回 gRPC-OK + body error 时 → result nil → 落 :37-41 IsInvestor 兜底 → **返回 `IsInvestor: true`**。会话过期/权限错被当成"只读账户"→ 实盘被错误阻断 + 误诊。MT5 侧 `connection_account.go:31` 同构同 bug。
- **修法**：两处加 `if e := resp.GetError(); e != nil && e.GetCode() != 0 { return nil, fmt.Errorf("mt4/5 AccountSummary: code=%d msg=%s", e.GetCode(), e.GetMessage()) }`（复用 `connection_extra.go:32` 模式）。fail-closed：app error 返回 error，不降级为 investor。
- **对抗证明**：mock AccountSummary 返回 gRPC nil-error + body Error → 旧代码返回 IsInvestor:true（RED，误判只读）；新代码返回 error（GREEN）。

## MDGATEWAY-3：订单事件流无死检测（成交/止损事件静默停达）

- **诊断（审计方已复核）**：`backend/internal/mdgateway/adapter/mt4/order_stream.go:71` orderUpdateRecvLoop 内层 `for { stream.Recv(); if err { break } }`——**无 no-data 超时**（quote/profit 流已修有 90s，这条对称兄弟流漏了）。broker 停推订单事件时流静默阻塞，成交/止损/止盈/平仓事件停止到达、无重连。MT5 侧同构。
- **修法**：仿 `quotes.go:164` quote 流的 `select{ recv/err/case <-time.After(quoteTimeout) }` 模式，给 orderUpdateRecvLoop 加 no-data 超时 + `handleStreamError` 重连。`quoteTimeout` 字段可复用（或独立 orderTimeout，可注入测试）。
- **对抗证明**：mock Recv 阻塞 → 超时触发 reconnecting（GREEN）；删 time.After 分支 → 永不重连（RED）。`scripts/verify-adversarial.sh` 可验。

---

## 不在本批（已复核后排除/降级）

- **TRUST-1（demo 标注）降级为上线后抛光**：审计方复核发现 `mt_accounts.account_type` **12 个账户全 'unknown'**（字段在但从未赋值）→ "标注模拟"缺数据源。需先在账户连接时判定 demo/real（mtapi AccountSummary flag 或 server 名推断）才能传播+标注。用户决策"包容即可"→ 不阻断核心环，上线后再做。**不动本批。**
- **MDGATEWAY-2/4**（HealthCheck 吞错、hub 订单 goroutine 无重连）：同区域，agent 报告**审计方尚未独立复核**。若施工方在改 MDGATEWAY-1/3 时顺手遇到，可一并核对修，但**修前先确认诊断属实**（别照 agent 结论盲改）。

## 红队自审
- [ ] LAUNCH-G1：确认 getStrategyPublicInfo 走 public transport 后不丢鉴权要求（它本就是 public RPC）；/share/:token 的 sharePublicClient 是现成参照。
- [ ] MDGATEWAY-1：app error 必须 fail-closed（返回 error），绝不降级返回 IsInvestor:true。
- [ ] MDGATEWAY-3：超时可注入测试（勿真等 90s）；MT5 同改。
- [ ] 全部 commit（上批已 commit，保持）。
