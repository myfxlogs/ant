# 施工交接：账户/参数管线安全 + 正确性批（F1/F2/#2/F4/F5，审计方独立复核为真）

> **审计方定论 2026-08-14，🟦待施工（2026-08-15 追加 F5 Live 页 i18n + F2 签名修正）。** 来源：账户/参数管线全面审计（2 agent 只读）。**好消息：跨用户真实资金交易不可能**（每条下单链路 checkBoundAccount + validateAccountAccess + UserOwnsAccount，uid 来自认证不可伪造）。本批修 4 个审计揪出的安全/正确性缺口 + 1 个 i18n 补齐。
>
> **铁律**：`scripts/verify-adversarial.sh` 自验删行必红 + commit + 部署 + 回填 registry + 不自行宣告完成。

---

## F1：ListActiveStrategies/WatchActive account_id 过滤无归属校验（P2 IDOR 信息泄露）

- **诊断（审计方复核）**：`strategy_active_handlers.go:33`(ListActive) / `:411`(WatchActive) 当前端传 `account_id` 过滤器时，直接 `sessionRegistry.ListByAccount(accountID)`，**不校验该 account 归属当前 uid**。`ListByAccount`（session_registry.go:166）只按 `sess.AccountID==accountID` 匹配，不过滤 UserID。→ 用户 A 传 B 的 account_id → **看到 B 的 live session 列表**（RunId/UserId/Symbol/StrategyName）。只读（Stop/Get/WatchSignals 有 `sess.UserID!=uid→PermissionDenied`，不能操控），但授权缺陷成立。
- **修法**：account_id 过滤器前加归属校验——`if err := s.checkBoundAccount(ctx, uid, uuid.Parse(accountFilter)); err != nil { return ...PermissionDenied/NotFound }`；或丢弃 account 过滤、改 `ListByUser(uid)` 后内存筛选。推荐前者（保留按账户过滤能力）。
- **对抗证明**：mock uid=A + account_id=B 的账户 → 旧代码返回 B 的 session（RED，泄露）；新代码 PermissionDenied/NotFound（GREEN）。

## F2：broker 层 accountOwnerVerifier 从未装配（P2 纵深防御死代码）

- **诊断（审计方复核）**：`mthub/service.go:59` 字段 + `:130 SetAccountOwnerVerifier` + preTradeChecks(service_orders.go:97)/preCloseChecks/modify/delete 里的 `if s.accountOwnerVerifier != nil`——**`SetAccountOwnerVerifier` 在 cmd/server 零调用**（grep 实证），字段恒 nil → 四处归属检查**全是死代码**。当前不可直接利用（上游 checkBoundAccount/validateAccountAccess 兜住），但设计的"最后一道防线"形同虚设。
- **⚠️ 实际签名（审计方 2026-08-14 自审修正，handoff 初版签名写错）**：`type AccountOwnerVerifier func(ctx context.Context, userID, accountID string) (bool, error)`（service.go:127）——三参返回 `(bool, error)`，非 `(ctx, accountID) error`。preTradeChecks 里 `owns, err := s.accountOwnerVerifier(ctx, uid, req.AccountID)`，**uid 为空直接拒单**（service_orders.go:99-100 fail-closed）。
- **修法**：`handlers_pipeline.go` wireMthubServices 里注入：
  ```go
  mthubSvc.SetAccountOwnerVerifier(func(ctx context.Context, userID, accountID string) (bool, error) {
      var exists bool
      err := pool.QueryRow(ctx,
          `SELECT EXISTS(SELECT 1 FROM mt_accounts WHERE id=$1::uuid AND user_id=$2::uuid AND deleted_at IS NULL)`,
          accountID, userID).Scan(&exists)
      return exists, err
  })
  ```
- **✅ 装配安全性（审计方已验证，非猜测）**：uid 取 `usermgr.GetUserID(ctx)`（ctx values 经 `context.WithoutCancel` 保留）；live 下单链 uid 已写入——手动路径 handler 注入 + `live_dispatch.go:390` 显式 `WithValue(placeCtx, interceptor.UserIDKey, cfg.UserID)`；自动调度路径 `schedule_engine.go:352` `UserID: schedule.UserID.String()` → buildLiveRun → 同 :390 写入。**两条路径 uid 都在，装配不会误拒生产单**。CloseOrder/Modify/Delete 路径同理经 dispatch 带 uid。
- **对抗证明**：注入后，mock 一个不归属用户的 account 经 preTradeChecks → 旧（nil verifier）放行（RED）；新（verifier 拒绝）报错（GREEN）。另加一测：**uid 为空的 ctx**（如内部调用）过 preTradeChecks → 拒绝（fail-closed GREEN）——防未来内部路径静默放行。

## #2：手动触发（Run Now）丢失策略参数（P2 正确性）

- **诊断（审计方复核）**：`frontend/src/pages/strategy/components/workspace/LiveSchedulesTab.tsx:130` onManualTrigger 调 `strategyActiveApi.start({ accountId, strategyCode, symbol, timeframe, mode:'paper', strategyId })`——**没传 `params`**。后端 StartStrategy 读 `req.Msg.GetParams()` 得空 → **手动测试跑的是空参数，不是 schedule 配的参数**。eager 无参不影响；参数化策略（MACD 等）手动测的与自动调度行为不同。
- **修法**：onManualTrigger 里把 `row.parameters`（或 `parseParametersToForm(row.parameters)` + `buildParametersFromForm` 结果）序列化后传入 `start({..., params})`。proto `StartStrategyRequest.params=6` 已有。
- **对抗证明**：前端测——渲染 onManualTrigger 路径，断言 start() 调用的 params 非空且=配置值（GREEN）；删传参 → params 空（RED）。

## F4：paper 模式 cfg.AccountID 无归属校验（P3 跨用户模拟写）

- **诊断（审计方复核）**：`strategy_active_handlers.go:347 resolveModeAndAccount` paper 分支不覆盖 cfg.AccountID（保留前端值），且 `:268 checkBoundAccount` 被 `mode==modeLive` 短路、`live_runner.go:106 preflightLiveChecks` 只查 live → **paper 模式接受任意前端 account_id，无归属校验**。→ 用户 A 传 B 的 account_id → `paperEngine.PlacePaperOrder(ctx, accountID=B)` 篡改 B 的模拟余额/持仓。仅 paper_orders 表（无真钱、可恢复），但是跨用户写。
- **修法**：paper 模式也加归属校验——resolveModeAndAccount 或 preflight 里，`if cfg.AccountID != "" { checkBoundAccount(uid, cfg.AccountID) }`（不限 live）。或 cfg.AccountID 归一为用户所有账户。
- **对抗证明**：mock uid=A + paper + account=B → 旧代码放行写 B 的 paper（RED）；新代码拒绝（GREEN）。

---

## F5：Live 页 i18n 补齐（P2，2026-08-15 用户要求追加进本批）

**现状（审计方实证）**：live-ui-final 新增 UI 的 i18n key 全部只有 `defaultValue` 兜底，textproto 无翻译条目 → 非 en locale（zh-cn/zh-tw/ja/vi）用户看到英文。i18n 体系 = `proto/ant/v1/i18n/base_*.textproto`（snake_case 字段名，如 `strategy_live_signal_type`）→ 生成 `frontend/src/gen/ant/v1/i18n/base_keys.ts`（camelCase 映射 `strategy.live.xxx`）。

**缺失清单（审计方逐一核对）**：
1. `LiveStrategyPage.tsx` 已引用但 textproto 无条目（grep `strategy.live.` 共 30 key，`base_en.textproto` 仅 32 个旧 snake_case，新 UI key 缺）：`lastSignal`（Last Signal）/ `pnl`（PnL）等全部 38 处 `defaultValue` 兜底的 key（完整列表 grep `defaultValue` 该文件即得）
2. `ScheduleTable.tsx:153-154`：`strategy.schedules.status.running` / `strategy.schedules.status.idle` 无条目
3. `LiveStrategyPage.tsx:161` `stale` Tag **硬编码无 key**——先加 key（建议 `strategy.live.stale`）再替换硬编码

**修法**：
1. 5 个 locale textproto（base_en / base_zh-cn / base_zh-tw / base_ja / base_vi）同步加条目（同 key 同字段名 snake_case，仅译文不同）。PnL 保持 "PnL"（交易术语通用），stale 译"过期/停滞"类语义
2. 跑项目 i18n 生成命令（查 `scripts/` 下 i18n 生成脚本，或 `base_keys.ts` 头部注释标注的生成方式）重生 keys
3. `stale` 硬编码替换为 `t('strategy.live.stale')`

**验证**：`grep -c "strategy.live" 各 locale textproto` 数量一致（5 文件同 key 数）；`npm run build` 绿；手工切 zh-cn 看 Live 页新列头非英文。

**对抗**：删任一 locale 的一条新 key → 该 locale Live 页对应文案回退 defaultValue 英文（可用 vitest 断言或生成脚本的 key 对齐检查——若生成脚本有 strict 校验则天然 RED）。
- [ ] F1：account 过滤校验用 checkBoundAccount（复用 :223-232 现成链），NotFound 不泄露存在性。
- [ ] F2：verifier 签名 `(ctx, userID, accountID) (bool, error)`（service.go:127），别照旧 handoff 记忆写单 error；SQL 查 mt_accounts 带 deleted_at IS NULL；uid 空 = 拒（fail-closed）。
- [ ] #2：params 序列化与自动调度路径一致（buildParametersFromForm）。
- [ ] F4：paper 校验 fail-closed（不归属即拒）。
- [ ] F5：5 个 locale textproto 同 key 数（`grep -c` 对齐）；snake_case 字段名与生成 map 一致；stale 硬编码已替换；npm run build 绿。
- [ ] 全部 commit + 部署。

## 非本批（设计决策，告知不修）
- mode 硬编码 live（schedule 无 mode 字段）——设计如此。
- strategyCode 活引用（每次拉模板当前代码，非冻结）——产品决策点（AI 迭代 vs 冻结），后续讨论。
