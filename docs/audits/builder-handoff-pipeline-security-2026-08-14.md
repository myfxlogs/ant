# 施工交接：账户/参数管线安全 + 正确性批（F1/F2/#2/F4，审计方独立复核为真）

> **审计方定论 2026-08-14，🟦待施工。** 来源：账户/参数管线全面审计（2 agent 只读）。**好消息：跨用户真实资金交易不可能**（每条下单链路 checkBoundAccount + validateAccountAccess + UserOwnsAccount，uid 来自认证不可伪造）。本批修 4 个审计揪出的安全/正确性缺口。
>
> **铁律**：`scripts/verify-adversarial.sh` 自验删行必红 + commit + 部署 + 回填 registry + 不自行宣告完成。

---

## F1：ListActiveStrategies/WatchActive account_id 过滤无归属校验（P2 IDOR 信息泄露）

- **诊断（审计方复核）**：`strategy_active_handlers.go:33`(ListActive) / `:411`(WatchActive) 当前端传 `account_id` 过滤器时，直接 `sessionRegistry.ListByAccount(accountID)`，**不校验该 account 归属当前 uid**。`ListByAccount`（session_registry.go:166）只按 `sess.AccountID==accountID` 匹配，不过滤 UserID。→ 用户 A 传 B 的 account_id → **看到 B 的 live session 列表**（RunId/UserId/Symbol/StrategyName）。只读（Stop/Get/WatchSignals 有 `sess.UserID!=uid→PermissionDenied`，不能操控），但授权缺陷成立。
- **修法**：account_id 过滤器前加归属校验——`if err := s.checkBoundAccount(ctx, uid, uuid.Parse(accountFilter)); err != nil { return ...PermissionDenied/NotFound }`；或丢弃 account 过滤、改 `ListByUser(uid)` 后内存筛选。推荐前者（保留按账户过滤能力）。
- **对抗证明**：mock uid=A + account_id=B 的账户 → 旧代码返回 B 的 session（RED，泄露）；新代码 PermissionDenied/NotFound（GREEN）。

## F2：broker 层 accountOwnerVerifier 从未装配（P2 纵深防御死代码）

- **诊断（审计方复核）**：`mthub/service.go:59` 字段 + `:125 SetAccountOwnerVerifier` + preTradeChecks(service_orders.go:97)/preCloseChecks/modify/delete 里的 `if s.accountOwnerVerifier != nil`——**`SetAccountOwnerVerifier` 在 cmd/server 零调用**（grep 实证），字段恒 nil → 四处归属检查**全是死代码**。当前不可直接利用（上游 checkBoundAccount/validateAccountAccess 兜住），但设计的"最后一道防线"形同虚设。
- **修法**：`handlers_pipeline.go` wireMthubServices 里注入：`mthubSvc.SetAccountOwnerVerifier(func(ctx, accountID) error { /* UserOwnsAccount 包装：DB WHERE id AND user_id */ })`。需把 uid 从 ctx 取（interceptor.GetUserID）。
- **对抗证明**：注入后，mock 一个不归属用户的 account 经 preTradeChecks → 旧（nil verifier）放行（RED）；新（verifier 拒绝）报错（GREEN）。

## #2：手动触发（Run Now）丢失策略参数（P2 正确性）

- **诊断（审计方复核）**：`frontend/src/pages/strategy/components/workspace/LiveSchedulesTab.tsx:130` onManualTrigger 调 `strategyActiveApi.start({ accountId, strategyCode, symbol, timeframe, mode:'paper', strategyId })`——**没传 `params`**。后端 StartStrategy 读 `req.Msg.GetParams()` 得空 → **手动测试跑的是空参数，不是 schedule 配的参数**。eager 无参不影响；参数化策略（MACD 等）手动测的与自动调度行为不同。
- **修法**：onManualTrigger 里把 `row.parameters`（或 `parseParametersToForm(row.parameters)` + `buildParametersFromForm` 结果）序列化后传入 `start({..., params})`。proto `StartStrategyRequest.params=6` 已有。
- **对抗证明**：前端测——渲染 onManualTrigger 路径，断言 start() 调用的 params 非空且=配置值（GREEN）；删传参 → params 空（RED）。

## F4：paper 模式 cfg.AccountID 无归属校验（P3 跨用户模拟写）

- **诊断（审计方复核）**：`strategy_active_handlers.go:347 resolveModeAndAccount` paper 分支不覆盖 cfg.AccountID（保留前端值），且 `:268 checkBoundAccount` 被 `mode==modeLive` 短路、`live_runner.go:106 preflightLiveChecks` 只查 live → **paper 模式接受任意前端 account_id，无归属校验**。→ 用户 A 传 B 的 account_id → `paperEngine.PlacePaperOrder(ctx, accountID=B)` 篡改 B 的模拟余额/持仓。仅 paper_orders 表（无真钱、可恢复），但是跨用户写。
- **修法**：paper 模式也加归属校验——resolveModeAndAccount 或 preflight 里，`if cfg.AccountID != "" { checkBoundAccount(uid, cfg.AccountID) }`（不限 live）。或 cfg.AccountID 归一为用户所有账户。
- **对抗证明**：mock uid=A + paper + account=B → 旧代码放行写 B 的 paper（RED）；新代码拒绝（GREEN）。

---

## 红队自审
- [ ] F1：account 过滤校验用 checkBoundAccount（复用），NotFound 不泄露存在性。
- [ ] F2：verifier 从 ctx 取 uid（认证），别从前端。
- [ ] #2：params 序列化与自动调度路径一致（buildParametersFromForm）。
- [ ] F4：paper 校验 fail-closed（不归属即拒）。
- [ ] 全部 commit + 部署。

## 非本批（设计决策，告知不修）
- mode 硬编码 live（schedule 无 mode 字段）——设计如此。
- strategyCode 活引用（每次拉模板当前代码，非冻结）——产品决策点（AI 迭代 vs 冻结），后续讨论。
