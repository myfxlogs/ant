# 施工交接：下一批（LAUNCH-G2 用户验收 + MDGATEWAY-2/4 可靠性，已审计方独立复核为真）

> **审计方定论 2026-08-13，🟦待施工。** 每项诊断已独立复核（非 agent 结论）。优先级：**LAUNCH-G2 是"基础用户能否走通"的关键**（用户验收目标），MDGATEWAY-2/4 是同区域可靠性、顺手做。
>
> **铁律**：`scripts/verify-adversarial.sh` 自验删行必红 + commit + **部署**（backend `docker compose build/up`；frontend `docker cp + nginx reload`）+ 回填 registry + 不自行宣告完成。
>
> **⚠️ 先确认 Tier 1 已部署**：binary/dist 时间早于 Tier 1 commit（16:09 UTC），可能是工作树 build 已含、也可能没部署。本批动工前先 `docker compose build backend && up` + 前端 cp/reload 确保 Tier 1（LAUNCH-G1/MDGATEWAY-1/3）真上线。

---

## LAUNCH-G2：购买后 + 空账户态无引导 → 新买家卡死（用户验收关键）

- **诊断（审计方已复核）**：`DeployScheduleModal.tsx:121` 空账户态只有 `notFoundContent={t(SCHEDULE_LAUNCH_NO_ACCOUNT_BODY_KEY)}`（文案），**无"去绑定 MT 账户"按钮/链接**。新买家（极可能还没绑账户）走到部署弹窗 → 空下拉、无出口 → 卡死。这是"基础用户走通 happy path"最实际的卡点。
- **修法**：
  1. `DeployScheduleModal` 空账户态：文案下加 `<Button onClick={() => navigate('/accounts/bind')}>` "去绑定 MT 账户"（route 已存在 BindAccount.tsx）。
  2. 购买成功后（`useMarketplace` 的 purchase handler）：`message.success` 后加引导 CTA——"去部署实盘"跳 `/strategy/live?tab=schedules` 或打开 DeployScheduleModal（参考 LiveSchedulesTab 的 getEnableNavigateTarget 模式）。
- **对抗证明**：渲染 DeployScheduleModal（accounts=[] 空态）→ 断言"去绑定"按钮存在 + 点击导航 `/accounts/bind`（GREEN）；删按钮 → getByRole 找不到（RED）。前端测试用 `@testing-library/react`。

## MDGATEWAY-2：HealthCheck 吞 Ping 响应错误（死会话判健康）

- **诊断（审计方已复核）**：`mt4/connection_account.go:62` HealthCheck `_, err := client.Ping(...)`（:78 行附近）**丢弃响应**，只查 gRPC err。mtapi Ping 返回 body error 时被吞 → HealthCheck 返回 nil（健康）→ 死会话/过期会话判健康 → 熔断不触发、不重连。MT5 同构。
- **修法**：`resp, err := client.Ping(...)`；加 `if e := resp.GetError(); e != nil && e.GetCode() != 0 { return fmt.Errorf("ping: code=%d msg=%s", ...) }`（复用 connection_extra.go:32 模式）。**先确认 `pb.PingReply` 有 Error 字段**（mt4 proto 核对；多数 mtapi Reply 都有）。
- **对抗证明**：mock Ping 返回 gRPC nil-error + body Error → 旧代码返回 nil（健康，RED）；新代码返回 error（GREEN）。

## MDGATEWAY-4：hub 订单事件 goroutine 无重连（Recv 错即永久死）

- **诊断（审计方已复核）**：`mt4/orders.go:304` SubscribeOrderEvents 的 `go func()`：Recv 出错即 `return`（:319）→ goroutine **永久退出，无外层重连**。mthub 订单事件馈线静默死亡，无重试。MT5 同构。
- **修法**：包外层重连循环（镜像 `orderUpdateRecvLoop` 的 `for { ensureConnected; ... if err { handleStreamError; continue } }` 模式），加 no-data 超时（复用 orderUpdateTimeout）。**别让单次 Recv 错误终结 goroutine**。
- **对抗证明**：mock Recv 返回 error → 旧代码 goroutine 退出无重连（RED，断言重连计数=0）；新代码重连（GREEN，重连计数>0）。

---

## 不在本批
- **TRUST-1**（demo 标注）：降级，需先做 demo/real 判定数据源（account_type 全 unknown），上线后。审计方在查 mtapi AccountSummary 是否返回 demo 标志。
- **TRUST-2**（OOS/阈值收紧）：需业务决策（阈值数值），待用户定。
- **EXEC-2 真实 fill 验证**：被动等任意策略产生成交，审计方从 orders.state 看 FILLED 即闭环。

## 红队自审
- [ ] LAUNCH-G2：`/accounts/bind` route 存在（BindAccount.tsx ✓）；空态 CTA 勿遮挡正常下拉。
- [ ] MDGATEWAY-2：先核 PingReply.Error 字段存在再写；fail-closed。
- [ ] MDGATEWAY-4：重连退避别风暴（复用 backoff/maxBackoff 模式）；MT5 同改。
- [ ] 全部 commit + 部署。
