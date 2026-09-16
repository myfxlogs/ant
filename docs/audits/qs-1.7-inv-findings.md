# QS-1.7-INV findings — open outcomeUnknown 恢复可行性调研

> 施工产物（只读调研，零生产代码改动）。handoff：`docs/audits/builder-handoff-qs-1.7-inv.md`。
> 每条结论附 file:line 证据；repo 内无证据处显式标"未知"。go/no-go 决策留决策方。

## 结论速览

| Q | 结论 |
|---|------|
| Q1 | **否**——`ClientID` 在调用链任何一跳都不进 `req.Comment`，也不进任何其他 broker 可见字段；MT5 下发 `Comment` 恒为空串，MT4 adapter 连 `req.Comment` 都不透传。ClientID 不可回显。 |
| Q2 | 出站方向 repo 内**无任何** comment 截断/转义/长度处理；MT4/MT5 broker comment 长度限制在 repo/proto 注释中**无记录 → 未知**。 |
| Q3 | 回读管线存在且 `Comment` 逐跳映射进 `OrderRecord`/position proto；**回显保真度（broker 是否改写/附加后缀/清空）在 repo 内无记录 → 未知**；`trade_event_store.go` 注释证明 broker comment 内容不可信（偶发非 UTF-8）。 |
| Q4 | `pb.OrderSendRequest`（mt4+mt5）**无** `client_id`/`external_id` 类字段；唯一候选载体 = `comment`（MT4 adapter 当前不透传）+ `magic`/`expertID`（已被 schedule 级 magic 占用，非订单唯一）。 |

## Q1 — 逐跳追踪（ClientID 是否下发 broker）

链路：`submitOrder` → `MtHubService.PlaceOrder` → `submitToBroker` → `exec.PlaceOrder`（mt4/mt5 Gateway 直调）→ `pb.OrderSendRequest`。

| 跳 | 坐标 | ClientID 去向 | Comment 去向 |
|----|------|---------------|--------------|
| 1. 起点 | `live_dispatch.go:359-379` | `ClientID: strategyOrderClientID(...)` :366；`req.Comment` **从未赋值** | 零值 `""` |
| — 格式 | `live_helpers.go:37-39` | `start-<runID uuid>-<barOpenTime>-<signalType>`（约 55+ 字符） | — |
| 2. PlaceOrder | `service_orders.go:32,61-71,110-111,299` | 仅内部消费：OMS `IdempotencyKey`(:32)、`idem.CheckAndSet`(:110-111)、`idem.SetTicket`(:61-71)、`TradeEvent.ClientID`(:299) | 不读不写 |
| 3. submitToBroker | `service_orders.go:205-225` | `exec.PlaceOrder(ctx, req)` :212 原样透传 | 同上 |
| 4a. mt5 Gateway | `mt5/orders.go:39-47` | **不进 proto**——`pb.OrderSendRequest` 无该字段 | `Comment: &req.Comment` :45 → 恒 `""` |
| 4b. mt4 Gateway | `mt4/orders.go:61-68` | **不进 proto** | **`Comment` 字段存在（`mt4.pb.go:7804` field 9）但 adapter 根本未设置**——MT4 连空 comment 都不下发 |
| 5. proto | mt4 `mt4.pb.go:7786-7813` / mt5 `mt5.pb.go:12194+` | 无 client_id/external_id 字段 | mt4 `comment`(9)、mt5 `comment`(9, oneof) |

**旁路核对**（非策略路径）：`brokerExec.SubmitOrder`（`broker_registry.go:97-118`）构造 `oms.OrderRequest` 时 **ClientID 整体丢弃**（`oms.OrderRequest` 无该字段，`broker.go:12-23`），仅 `Comment: req.Comment` :111 透传；`brokerAdapter.Submit`(:34-44) 反向亦然。手动下单 RPC（`mthub_service.go:108`）`Comment`/`ClientID` 来自 proto 两个独立字段，仍无复制。execalgo 子单（`executor_run.go:71-77`）两字段均未设。

**结论**：全链路无一跳把 `ClientID` 复制进 `Comment` 或等价字段。当前形态下 ClientID **不可能**被 broker 回显。

## Q2 — comment 长度/改写约束（出站）

- 出站 comment 路径上**无任何**截断/转义/长度校验：`broker_registry.go:43,111` 与 `mt5/orders.go:45` 均为裸透传。`sanitizeUTF8`（`trade_event_store.go:29,107-109`）只作用于**入站**事件字段（Canonical/Broker/ClientID），与出站无关。
- `start-<uuid36>-<ts>-<signalType>` ≈ 55+ 字符：是否超 MT4/MT5 comment 限制被截断——repo 代码注释、pb 注释、docs 均**无记录 → 未知**（MT4 常见外部知识为 31 字符上限，但本仓无证据，不据此下结论）。

## Q3 — 回显保真度（入站）

回读管线存在，`Comment` 逐跳进 `OrderRecord`/position：

- MT4：`order_history.go:73`（OpenedOrders）、`:152`（OrderHistory）、`order_stream.go:178`（position update）、`profit.go:314,368`。
- MT5：`order_history.go:74,134`、`order_stream.go:155,178`、`profit.go:146`；`pb.Order.Comment`（mt5.pb.go:3886 区 field 26）+ 独立 `CloseComment`（field 16）。

**保真度证据（反向）**：
- `trade_event_store.go:104-106` 注释："MT4 gateways occasionally emit non-UTF-8 bytes in broker fields (comments, raw symbol names, etc.)"——broker comment 内容是 broker 控制的、可含任意字节。
- MT5 proto 将 `Comment` 与 `CloseComment` 分列（broker 可在平仓时写另一 comment）——佐证 broker 会改写/附加自己的 comment。
- broker 对 open-order comment 附加后缀（如 `[sl]`/`[tp]`）、改写或清空的行为：repo 内**无记录 → 未知**。

## Q4 — 其他 broker 可见字段

- mt4 `pb.OrderSendRequest`（`mt4.pb.go:7786-7813`）：id/symbol/operation/volume/price/slippage/stoploss/takeprofit/**comment**/magic/expiration/placedType——无 client_id/external_id。
- mt5 `pb.OrderSendRequest`（`mt5.pb.go:12194+`）：id/symbol/operation/volume/price/slippage/stoploss/takeprofit/**comment**/expertID/stopLimitPrice/expiration/expirationType/placedType——无 client_id/external_id。`Order.RequestId`（field 30）只存在于**回显**侧且无对应可写字段（broker 自分配）。
- `magic`(mt4)/`expertID`(mt5) 已被 `strategyMagic(scheduleID)`（`live_helpers.go:43-45`）占用且为 **schedule 级**——同一 schedule 所有订单同 magic，不能唯一标识单笔 open order。
- `expiration`/`placedType` 语义固定，不适合作恢复标识。

## 施工方可行性观察（go/no-go 留决策方）

1. **现状不可行**：ClientID 不经 Comment 回显（Q1 全链路无复制）；即使补上 `ClientID→Comment` 复制，MT4 adapter 当前不透传 `req.Comment`（`mt4/orders.go:61-68` 缺 `Comment` 设置，proto field 9 存在可用），MT5 恒发空串。
2. **即使接线后仍有两层未证**：comment 长度截断（Q2 未知）与 broker 回显改写/清空（Q3 未知，且已有"comment 内容不可信"的反例证据）。`Comment` 匹配做恢复标识需要前缀匹配容忍 broker 附加，或先实测 mtapi 层回显保真度。
3. **替代路径在 repo 内无解**：`magic`/`expertID` 非订单唯一；`RequestId` 只读。唯一候选载体就是 `comment`，需先改 MT4 adapter 透传 + 定义编码格式 + 实测回显保真度。
4. 提示词事实 d 一处偏差：称 mt4 同构透传 comment——实测 mt4 `PlaceOrder` 未设置 `Comment` 字段（仅 MT5 透传）。
