# VM 实盘 parity 测试协议（VM-LIVE-PARITY-1 设计实查）

**日期**: 2026-09-19 · **设计方**: Devin CLI · **状态**: 待账号（demo 优先；凭据走 secrets，禁聊天）

## 立项背景

VM 债系全部独立验收后，代码层 fail-closed/API 真实性已证。剩余未证轴=**真实经纪商行为 parity**——以下假设只有实盘（或 demo）能证伪。

## A. 已证（不需重测）

- OrderSend/Modify/Close 错误路径 fail-closed（ORDERSEND-NILBROKER/TRADE-BUILTIN-ERR-SWALLOW mutation 实证）
- coverage fatal 门控三路收敛（MQL-LOOP-4）
- mthub PlaceOrder 全闸：preTradeChecks→OMS 状态机→place gate→idempotency→cost estimate（service_orders.go:21-58）

## B. 假设清单（实盘证伪点）

| # | 假设（代码坐标） | 性质 | 测试 |
|---|---|---|---|
| P1 | `OrderResult{Price: req.Price, Volume: req.Volume}` 回显**请求值**（strategy/runner/broker.go:51-55）——executor 只回 ticket，实际成交价/量从不回读 | 真实成交滑点时 VM 结果偏离 broker 事实 | T3 校验 |
| P2 | `deviation`（OrderSend arg4）取进 req 但 executor.PlaceOrder 签名无该参（broker.go:40-46）——滑点保护静默丢弃 | mtapi 若不支持则无影响；若支持则 VM 漏保护 | T4 |
| P3 | 符号常量硬编码 venue 值：`FREEZE_LEVEL=0`/`TRADE_MODE=FULL`/`EXEMODE=MARKET`/`ORDER_MODE=63`/`SPREAD_FLOAT=0`/`ORDER_GTC_MODE=0`（vm_builtin_account.go:180-194） | 真 broker freeze>0/执行模式不同→策略拿错事实 | T1 |
| P4 | `StopsLevel/Digits/Point/ContractSize/TickValue/TickSize/MinLot/LotStep` 实盘经 venue 回填（live_context.go:398-433）——设计正确，但回填覆盖度未在真 broker 验过 | 回填缺失字段→VM 拿到零值 | T1 |
| P5 | 无 min lot/lot step 归一化层可见（OrderSend volume 直通 executor）——broker 侧拒绝才是真相源 | 归一化缺失→真拒绝 vs VM 预期 | T5 |
| P6 | freeze/stops level 前置校验未见（OrderSend 不查 stops level 直发） | 贴边 SL/TP→broker 拒绝，VM 是否正确传播 | T5 |
| P7 | tick 流断连/跳空/点差放大时 OnTick 行为 | 实盘观测 | T6 |
| P8 | VM-LIVE-MTF-1（实盘 MTF）——已知暂缓，不在本协议范围 | 需求驱动 | 跳过 |

## C. 测试协议（有界）

**前置**：demo 账户（或 0.01 手可承受实盘）+专门测试用户+该账户仅挂测试。符号白名单 EURUSD（高流动、5 位小数）。

| 测试 | 动作 | 可证伪断言 |
|---|---|---|
| T1 符号属性对账 | VM 侧 `MarketInfo`/`SymbolInfo*` 全字段 vs mtapi `SymbolInfo` RPC 原生值逐字段对账（stops/freeze/minlot/lotstep/tickvalue/spread_float） | 硬编码项（P3）与 broker 真值不符→差异清单=修复项 |
| T2 账户字段对账 | `AccountInfo*` 全字段 vs mtapi 账户摘要逐字段 | 零值/假值字段清单 |
| T3 成交回报对账 | 0.01 手市价开→立即查 broker 侧 position/deal 实际 fill 价/量 vs `OrderSend` 返回值 | `RetCode=Done` 但 Price 是请求回显≠fill→P1 实锤 |
| T4 deviation 语义 | OrderSend deviation=0 vs deviation=999 各一单，对比行为/成交 | 无差异=deviation 被丢（P2）或有差异=被传递 |
| T5 拒绝路径 | ①volume 低于 MinLot（如 0.001）②volume 违 LotStep（如 0.015 步进 0.01）③SL 贴边（<StopsLevel）④freeze 区 modify | 每例断言：VM 错误如实传播（RetCode≠Done 或 error 上浮），**禁假成功**；错误码映射合理性 |
| T6 tick 观测 | 10min 实盘 tick 流：OnTick 频率/断连行为/点差真实值 vs MarketInfo(MODE_SPREAD) | 断连重连语义+spread 真实值与常量差异 |
| T7 收尾 | 全平+对账（broker 持仓=0，VM 侧 OrdersTotal=0） | 无残留仓位/幻影持仓 |

**硬边界**：max 5 单、仅市价单、仅 EURUSD、0.01 手、T7 自动全平兜底、时限 30min。**禁**：挂单/加仓/马丁/非白名单符号/人工凭据进 chat。

## D. 凭据交付

- 密码已在聊天暴露→**先重置**；新凭据经 upload-secrets 通道。
- 还需：**broker 服务器名 + MT4/MT5 + demo/live** 三要素（mtapi 连接必需）。
- 测试走现有 live 路径（启动一个实盘策略跑测试策略代码），不依赖未部署改动。
