# VM-LIVE-PARITY-1 实盘对账报告（2026-09-19 实测）

> 执行：Devin CLI（业主授权：xianhua.chan@gmail.com 名下未冻结账户可用）。**测试账户 `904d14e6`（MT4 Exness-Trial demo，余额 $77.8k，connected，零在跑策略）**。周六仅 crypto 可交易→符号 BTCUSDm。方式：ConnectRPC API（前端同路径，JWT 经 JWT_SECRET 自签——未用用户密码）。3 单市价单+2 拒绝用例，合计 P&L **-0.94 USD**（demo）。测试毕全平，888 遗留持仓未动。

## 判定总表

| 假设 | 判定 | 证据 |
|---|---|---|
| **P1 OrderSend 回显请求值** | **✗ 实锤（双轴）** | 请求 price=0 → broker 实际 fill **81262.24**；请求 volume=0.015 → broker 归一化成交 **0.02**。VM `OrderResult{Price: req.Price, Volume: req.Volume}`（broker.go:51-55）——实盘模式策略读返回值拿到的是请求回显非 broker 事实 |
| **P2 deviation 丢弃** | ⚠ 静态实锤/行为未测 | `PlaceOrderRequest` proto 无 deviation 字段、`executor.PlaceOrder` 签名无参——丢弃成立；broker 侧影响不可经 API 观测 |
| **P3 venue 硬编码** | ~ 部分一致 | stops_level：VM 说 0=broker 实测 0（贴边 SL 被接受 ✓）；freeze/trade_mode/exemode/order_mode 因 SymbolParams 瘦数据无法对账 |
| **P4 venue 回填完整性** | **✗ 实锤瘦数据** | 实测 BTCUSDm：point=0/stopLevel=0/lotStep=0/lotMax=0/tradeMode=0/spreadFloat=false——而 lotStep 真实=0.01（T5b 归一化证明存在）。**VM 实盘会把 Point=0 给策略**——按 point 算 SL 距离的策略拿零值。根源在 mtapi SymbolParams 上游不返回，非 adapter 丢字段（orders.go:205-229 全映射） |
| **P5 无归一化层** | ✓ 如实传播 | 系统直通 0.015→broker 静默归一化 0.02（与 P1 同源：VM 报 0.015 实持 0.02） |
| **P6 无前置校验** | ✓ 如实传播 | code=131 Invalid volume 如实上浮无假成功（T5a）；贴边 SL 被接受=stops=0 真值 |
| **P7 tick 行为** | ✓ 正常 | 测试窗口 tick 活跃（81262→81207），断连未观测到（窗口短） |

## 附带发现

1. **comment 字段在途丢失**：PARITY-T3/T5a/T5b 的 comment 到 broker 记录全空（magic 99001 正常透传）。888 遗留单 comment 显示 `\ufffd` 乱码——comment 编码链有 UTF-8 损耗（可能是 mtapi↔MT4 ANSI 边界）。
2. **PlaceOrderResponse 无 fill 信息**：只回 {ticket, status="submitted"}——API 层同样不回读实际成交（P1 同源，API 层至少不撒谎只给 ticket）。
3. T5c 验证了**全链 SL 执行**：81253.63 开+SL 81240→市价下穿→81238.16 `[sl]` 触发，滑点 -1.84，OrderHistory 如实记录。

## 新债登记（修复项）

| ID | 内容 |
|---|---|
| **VM-LIVE-PARITY-F1（P2）** | 实盘 OrderSend 结果回显请求值非 broker 事实：①`OrderResult.Price/Volume` 应回读实际 fill（OrderHistory/position 对账后填，或标注"requested"语义）；②deviation 形参丢弃应如实声明或接线。影响：策略用返回值算 SL/记账时拿错事实。**回测无此问题**（回测 fill=模型价=请求价一致）。 |
| **VM-LIVE-PARITY-F2（P3）** | SymbolParams 瘦数据链：mtapi 上游对 BTCUSDm 不返回 point/lotStep/stopLevel/maxLot/tradeMode→VM 实盘暴露 Point=0。修复方向：point 可由 digits 推导（10^-digits）作 fallback 并标注来源；lotStep/stopLevel 缺省时应如实表达"未知"而非 0——或经 SymbolInfoEx（mt4.pb.go:3487 有更全字段）补链。 |
| **VM-LIVE-PARITY-F3（P4）** | comment 字段在途丢失+UTF-8 损耗（PARITY-* 全空/888 单乱码）。查 adapter→mtapi→MT4 comment 编码链。 |

## 不需修的（实测排除）

- 拒绝路径假成功：无（131 如实上浮）
- stops_level 硬编码 0 与 broker 不符：该 broker 实为 0 ✓
- OMS/place gate/idempotency 实盘可用：全链工作
