# 施工交接：DEPLOY-LIVE 实盘部署管线 P1 修复（2026-08-12）

> 审计方（Claude Code）2026-08-12 实盘部署管线审计完成，三层文档已回填（registry DEPLOY-LIVE 条目 / handover 变更日志 / memory open-items-registry）。
> 施工方（Windsurf）任务：修复 **DEPLOY-LIVE-1 + DEPLOY-LIVE-2**（两个 P1）。**one task = one scope：只做本文档范围内的事。**
> 并行任务：CREATE-SCHEDULE-200EMPTY 交接（`builder-handoff-create-schedule-200empty-2026-08-12.md`）含 DEPLOY-LIVE-3（applyAccountSwitch 同源修复 + 补切账户测试），本文档不重复。

---

## 一、症状与根因（已定论，勿重新排查）

### DEPLOY-LIVE-1：tick/trade 信号 → nil bar panic → 后端进程崩溃（P1）

**触发链（完整，对抗已验证）**：

1. 策略实现 `OnTick`（`detectExecModels` 探测 `HasOnTick` → `subscribeTickUpdates` 订阅 tick 流）
2. MQL EA 在 OnTick 里 `OrderSend()`（标准模式；`strategy/runner/runner.go:94-106` `ts.OnTick` 直接返回 Signal → `vmSignalToProto` → `"buy"`/`"sell"`）
3. `handleTick`（`live_runner_events.go:151`）调 `s.dispatchFromBytes(ctx, cfg, nil, respBytes, activeSess)` —— **bar 传 nil**（handleTrade 同，:181）
4. `dispatchLiveSignal`（`live_dispatch.go:54-63`）`case "buy", "sell"` → `s.dispatchMarketOrder(ctx, cfg, bar.OpenTime, ...)` —— **nil 解引用，panic 发生在调用线程**
5. panic 沿 event loop → `RunLiveStrategy` → `runOne` → ScheduleEngine goroutine 传播，**全链无 recover → 后端进程崩溃，所有账户全部调度停摆**

**修复方向（2 选 1，推荐 A）**：

- **A（最小）**：`dispatchMarketOrder`/`dispatchPendingOrder` 签名改收 `barOpenTime int64`（调用处 `if bar != nil { bar.OpenTime } else { 0 }`），或在 `dispatchLiveSignal` 里提取 `barOpenTime := int64(0); if bar != nil { barOpenTime = bar.OpenTime }` 后传入。
- **B（彻底）**：`dispatchFromBytes` 增加 nil-bar 分支，tick/trade 信号走独立路径。

**⚠️ 附带问题（必须一并处理，否则幂等失效）**：`submitOrder` 里 `ClientID: strategyOrderClientID(cfg.RunID, barOpenTime, sig.GetSignalType())`——tick 单的 `barOpenTime=0` → **同 run 同类型 tick 单共用同一 ClientID → 幂等守卫吞掉后续单**。修复：为 tick/trade 信号引入单调计数或时间戳维度（如 `strategyOrderClientID` 增加 per-run 计数器，或在 cfg 里加 `tickSeq` 原子计数器）。

### DEPLOY-LIVE-2：MT4 `mt4Op` default → `Op_Buy`：stop_limit 信号变市价买入错单（P1）

**位置**：`backend/internal/mdgateway/adapter/mt4/orders.go:17-34`：

```go
func mt4Op(side mthub.Side, ot mthub.OrderType) pb.Op {
	switch {
	case side == mthub.SideBuy && ot == mthub.OrderMarket:  return pb.Op_Op_Buy
	case side == mthub.SideSell && ot == mthub.OrderMarket: return pb.Op_Op_Sell
	case side == mthub.SideBuy && ot == mthub.OrderLimit:   return pb.Op_Op_BuyLimit
	case side == mthub.SideSell && ot == mthub.OrderLimit:  return pb.Op_Op_SellLimit
	case side == mthub.SideBuy && ot == mthub.OrderStop:    return pb.Op_Op_BuyStop
	case side == mthub.SideSell && ot == mthub.OrderStop:   return pb.Op_Op_SellStop
	default:
		return pb.Op_Op_Buy   // ← BUG：buy_stop_limit/sell_stop_limit 落这里 → 市价买入！
	}
}
```

**触发**：策略（MQL5/Python，SDK 支持 stop_limit）绑定 **MT4 账户** → `dispatchPendingOrder` 生成 `mthub.OrderStopLimit` → `mt4Op` 落 default → **Op_Buy 市价买入**。MT5 adapter（`mt5/orders.go:106-109`）有正确 `BuyStopLimit/SellStopLimit` case。mthub 层（`service_orders.go`）**无平台类型预校验**。

**修复方向**：
1. `mt4Op` 增加 stop_limit case 或 default 返回 `(pb.Op, error)`——**推荐**：改签名返回 error，`PlaceOrder` 对未知组合返回 `fmt.Errorf("mt4 unsupported order type: %s/%s", side, orderType)`，**绝不静默降级为 Buy**。
2. 可选加固：mthub `submitToBroker` 按 executor 平台预校验（`OrderStopLimit` 仅允许 MT5）。

---

## 二、修复步骤（按序执行）

1. **DEPLOY-LIVE-1**：改 `live_dispatch.go` 的 `dispatchMarketOrder`/`dispatchPendingOrder` 签名为 `barOpenTime int64`；`dispatchLiveSignal` 提取 nil-safe barOpenTime；`submitOrder` 的 ClientID 增加 tick 维度（计数器或时间戳）。
2. **DEPLOY-LIVE-2**：改 `mt4/orders.go` `mt4Op` 返回 `(pb.Op, error)`，default 返回 error；`PlaceOrder` 传播 error（含 breaker 处理）。**MT5 adapter 不改**（已正确）。
3. **门禁**：`cd backend && go build ./...` + `go test ./internal/connect/strategy/... ./internal/mdgateway/adapter/mt4/... ./internal/mthub/...`（全量跑不动就跑相关包，如实记录范围）。
4. **重建部署**（唯一合法方式，禁止宿主机 go build → docker cp）：
   ```
   docker compose build backend && docker compose up -d backend
   ```
5. **部署后冒烟**：现有 e2e 凭据登录 → 任一合法 CreateSchedule + Enable → 确认 schedule 记录存在、容器日志无 panic。**tick 触发场景**（OnTick 发单的策略）若环境可测则实测，不可测如实记录。

## 三、验收标准

| # | 标准 | 验证方式 |
|---|------|---------|
| 1 | tick 信号含 buy/sell → 不再 panic（下单或正常拒绝，绝不崩进程）| 单测 + 代码核对 |
| 2 | `mt4Op(buy, stop_limit)` → error，非 Op_Buy | 单测断言 |
| 3 | MT5 stop_limit 仍正常（回归不破）| `mt5/orders.go` 单测保持绿 |
| 4 | bar 信号路径回归：`bar.OpenTime` 仍进 ClientID | 现有 live_runner 测试保持绿 |
| 5 | 部署后容器重启 + 无 panic 日志 | docker ps + docker logs |
| 6 | 附带修复：tick 单 ClientID 不碰撞（同 run 连续 tick 单不被幂等吞）| 单测/代码核对 |

## 四、对抗证明（必做，删了不红 = 未完成）

1. **DEPLOY-LIVE-1**：写单测——tick 信号（`SignalType: "buy"`）+ nil bar 走 `dispatchLiveSignal` → 修复前 panic（红，可先跑证明）；修复后 → 走 `submitOrder`（绿）。**实测记录红/绿各一行**。
2. **DEPLOY-LIVE-2**：单测 `mt4Op(sideBuy, OrderStopLimit)` → 修复前 `Op_Buy`（红）；修复后 error（绿）。
3. **附带（幂等）**：单测 `strategyOrderClientID` 在两次 tick 信号下不同（绿）；若实现为计数器，验证回绕/并发安全。

## 五、红队自审（任务级 edge cases，必须逐条给出结论）

- [ ] `dispatchCloseOrder`/`dispatchModifyOrder`/`dispatchCancelOrder` 不碰 `bar.OpenTime`（不受改动影响）——核对
- [ ] `dispatchCloseAll` 的 goroutine 内 recover 已有——新改动不要破坏
- [ ] tick 计数器并发安全（event loop 单 goroutine 消费 tickCh，但 `SubmitOrder` 的 go routine 不读写计数器——确认）
- [ ] MT4 `PlaceOrder` 的 breaker：error 路径是否触发 `OnFailure`——保持与现有 error 分支一致
- [ ] 部署时是否有未提交 migration？（`git status backend/migrations/`）
- [ ] 提交内容核对：registry/handover 变更日志只追加不删（pre-commit 钩子拦删，被拦 = 改好文档再提交，禁 `--no-verify`）

## 六、回填（不做 = 任务判失败）

1. `docs/audits/tech-debt-registry.md`：DEPLOY-LIVE-1/2 条目 `🟦open → ✅done`（标日期）+ 追加真实修复记录（commit、测试输出、对抗红绿各一行）。若真实根因与审计方不同，如实写明。
2. `docs/audits/handover-audit-plan.md` 变更日志加一行。
3. 本文件完成后移到 `docs/audits/archive/` 或标注完成（审计方验收后处理）。

## 七、沟通

- 完成后在共享会话（或交接文件）报告：修复动作 + 测试输出 + 对抗证明红绿记录 + 回填位置。**不自行宣告完成**，等审计方核对状态 + 实测后 ✅ 才权威。
