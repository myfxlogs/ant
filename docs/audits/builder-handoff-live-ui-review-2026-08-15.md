# 施工交接：live-ui-final 复审批（ARCH-4-MT4-MAGIC + 红测试修复 + 拆分收尾）

> **审计方（Claude Code）2026-08-15。** 复审 2198143e（live-ui-final）后定论。本批 3 项，全部有精确根因 + 对抗测试要求。one task = one scope，不扩范围。

---

## 🔴 Task 1：MT4 下单传 magic（ARCH-4-MT4-MAGIC，P1）

**根因**：`backend/internal/mdgateway/adapter/mt4/orders.go:60-66` `PlaceOrder` 构造 `pb.OrderSendRequest` 时**未填 `Magic: req.Magic`**。mt4.proto 有 `int32 magic = 10` 字段；mt5 正确传（`mt5/orders.go:45` `ExpertID: pInt64(int64(req.Magic))`）。生产账户是 MT4 → 持仓回读 magic 恒 0 → close-all 静默全 skip / GetSchedulePositions 恒空 / live-ui-final P0-4 PnL 恒 "-"。

**改法（1 行）**：mt4 `PlaceOrder` 的 OrderSendRequest 加 `Magic: req.Magic`。

**对抗测试（必做）**：
- `backend/internal/mdgateway/adapter/mt4/mock_test.go` mockTradingClient 加 `lastOrderSend *pb.OrderSendRequest` 捕获字段，`OrderSend` 里记录 `m.lastOrderSend = in`。
- `mt4_test.go` 的 `TestPlaceOrder_Success`（:831）：PlaceOrder 时传 `Magic: 12345`，断言 `exec.lastOrderSend.Magic == 12345`。
- **删 `Magic: req.Magic` 该行 → 测试必须 RED**（assert 失败）。用 `scripts/verify-adversarial.sh` 验证。

**不动**：mt5（已正确）、mthub（ORDERS-MAGIC 已修已部署，别动）。

---

## 🔴 Task 2：红测试修复（DEPLOY-LIVE-1-COVERAGE-RED）

**根因**：`backend/internal/connect/strategy/deploy_live_test.go` 的 `mockOrderExecutor.FetchSymbolParams` 返回 `nil, nil` → `MtHubService.evaluatePlaceGate` 的 `CachedSymbolParam` 报 "symbol EURUSD not found" → **fail-closed 拒单**（service_orders.go:171，05859858f margin refactor 引入，生产行为正确）→ executor 从未调用 → `TestDeployLive1_LivePathNilBarNoPanic` 2s 超时红。

**改法（1 处 mock）**：照抄 `backend/internal/mthub/limiter_estimator_test.go:155-161` 的 mockExecutor 模式：

```go
func (m *mockOrderExecutor) FetchSymbolParams(_ context.Context, canonicals []string) ([]*mthub.SymbolParam, error) {
	if len(canonicals) == 0 {
		return nil, nil
	}
	return []*mthub.SymbolParam{{Canonical: canonicals[0], ContractSize: decimal.NewFromInt(100000)}}, nil
}
```

**验证**：`go test -count=1 -run 'TestDeployLive1' -timeout 60s ./internal/connect/strategy/` 全绿（6 个测试）。
**对抗验证**：把 mock 改回 `nil, nil` → `TestDeployLive1_LivePathNilBarNoPanic` 必须 RED（2s 超时）。

---

## Task 4：前端 live-ui-final 复审批 🟡×3（`frontend/src/pages/strategy/LiveStrategyPage.tsx`）

1. **:166-167 空 lastSignalAt 被标橙**：`secondsSince(null)` 返回 Infinity → `Infinity > 300` 为真 → 从未出信号的策略 "-" 变橙色。改：lastSignalAt 缺失时颜色用正常（不加 warning）。
2. **:167 缺 >15min 灰档**（spec 要求）：`s > 300` 橙 → 应 `s > 900` 灰（secondary）。
3. **:86 心跳跳过误伤真空列表**（5fbebacc 引入，影响本批心跳机制）：`!event.strategies?.length` 把心跳 `{}` 和真实空列表 `{strategies: []}` 一起跳过 → 用户无 active 策略时 loading 永转、停掉最后一个策略后旧行残留。**⚠️ 前端无法区分**（protobuf-es 对 absent repeated 字段初始化为 `[]`，心跳 `{}` 反序列化后 `strategies === []` 与真实空列表相同；`=== undefined` 永远 false）→ **必须后端加 marker**：
   - `proto/ant/v1/strategy_runtime.proto:540` `WatchActiveStrategiesEvent` 加 `bool heartbeat = 2;`
   - `buf generate`（Go + TS 同步重生）
   - `strategy_active_handlers.go:441` 心跳改发 `&antv1.WatchActiveStrategiesEvent{Heartbeat: true}`
   - 前端 `:86` 改 `if (event.heartbeat) continue;` —— 心跳跳过、真实空列表正常 `setActiveStrategies([])` + `setLoading(false)`
4. （顺带 :154）有 bid/ask 但 lastTickAt 缺失时误挂 stale Tag：加 `record.lastTickAt` 存在性守卫。

**对抗**：① 后端 `TestWatchActiveStrategies_Heartbeat`（已存在）改为断言心跳事件 `Heartbeat == true`（删 `Heartbeat: true` → RED，事件不可区分）；② 前端（vitest 或手工）：心跳事件不触发 setLoading/清行（GREEN）、`{strategies: []}` 触发加载完成清行（GREEN）。

---

## Task 3：审计方拆分收尾（已编译验证，只需确认+提交）

审计方把 `strategy_schedules.go`（468 行）拆为：
- `strategy_schedules.go`（299 行，已删 GetSchedulePositions/proto converters/import 清理）
- `strategy_schedule_positions.go`（新文件，192 行：GetSchedulePositions + paperSchedulePositions + buildScheduleProto + scheduleParamsToProto + stringListToProto）

已验：`go build ./...` + `go vet ./internal/connect/strategy/` 绿；check-file-lines 0 errors；strategy 包仅 Task 2 的测试红。**你只需**：确认拆完后 diff 与原文件逻辑一致（`git diff HEAD -- backend/internal/connect/strategy/strategy_schedules.go` + 新文件对照原 468 行版本），如有遗漏补回。

---

## 提交 + 部署 + 回填（全部必做）

1. 4 个改动（mt4 orders.go + deploy_live_test.go + 拆分确认 + 前端 🟡×3）+ 本批对抗测试 → 一个 commit，message 格式 `fix(ARCH-4-MT4-MAGIC): mt4 下单传 magic + 红测试修复 + 拆分收尾 + live-ui 前端 🟡`
2. 门禁：`go build ./...`、`go test ./...`（除 internal/service 宿主机无 PG 的既有失败外必须全绿）、`go run ./tools/check-file-lines --strict`（0 🔴）、前端 `npm run build`（tsc 0 err）
3. 部署：后端 `docker compose build backend && docker compose up -d backend` + 前端 `docker cp frontend/dist/. alphaforge-frontend:/usr/share/nginx/html/ && docker exec alphaforge-frontend nginx -s reload`（唯一方式）
4. 部署后回填 registry（只改状态列+追加，不删不改审计方事实）：
   - `ARCH-4-MT4-MAGIC` → ✅done（标日期 + 对抗证明结果）
   - `DEPLOY-LIVE-1-COVERAGE-RED` → ✅done（同上）
   - `tech-debt-registry.md` 变更日志 + `handover-audit-plan.md` 变更日志 各加一行
5. 部署后实测回填：MT4 下单后日志/持仓 magic 非 0（`GetSchedulePositions` 有数据）、PnL 列有值

## 红队自审清单（提交前逐项自查）

- [ ] mt4 Magic 传参后，MT4 客户端持仓显示 magic number 与我们策略 magic 一致（strategyMagic(scheduleID) 确定性，可在日志对）
- [ ] mockTradingClient 捕获的 lastOrderSend 是同一个请求对象（非深拷贝，断言值即请求值）
- [ ] 红测试修复没动生产代码（只动 deploy_live_test.go mock）
- [ ] 拆分文件没丢 2198143e 的 P1-5 富化逻辑（buildScheduleProto 的 reg.GetByScheduleID → IsRunning/ActiveRunId/SignalCount）
- [ ] Task 4 #3：proto 加 heartbeat 字段后 **buf generate 必须执行**（Go `gen/proto` + TS `frontend/src/gen`），否则后端 `Heartbeat` 字段编译不过 / 前端类型缺失
- [ ] Task 4 #3：旧心跳测试 `TestWatchActiveStrategies_Heartbeat` 同步更新为断言 `Heartbeat==true`（别只改生产代码不改测试）
- [ ] 全部 commit（不留工作树），无 --no-verify
