# 地基审计 · mt-gateway

> 审计框架：`docs/audits/foundation-audit-framework.md`

## 1. 核心设计替代方案

**当前**：三层——proto stub → adapter（MT4/5 独立）→ mthub 中枢。适配器各自封装 gRPC，mthub 通过 `OrderExecutor` 接口统一访问。

**替代方案 A**：直接让 mthub 调用 mtapi.io RPC，干掉 adapter 层。

不选。adapter 层做两件事：重连循环和平台差异封装。重连逻辑（指数退避,取消上下文,重订阅）如果混入 mthub，mthub 的代码会膨胀 2-3 倍且被平台特定逻辑污染。分三层是对的。

**替代方案 B**：MT4/5 adapter 共享代码。

不选。深入审计已确认两个平台的 proto 差异是语义级（字段类型不同、子消息结构不同、错误码不同），共享代码 = 满屏 `if platform == "mt4"`。当前"禁止共享"约束是正确的。

**结论**：✅ 三层架构是最优解。不需要改。

## 2. 上下游契约

**上游**：account-mgmt 提供账户凭据 → runner_gateway.go 装配 Gateway 实例 → `Connect()`。契约是 `mdtick.AccountConfig` 结构体。清晰。

**下游**：
- `risk-gate/oms` 通过 `mthub.OrderExecutor` 接口下单。接口干净——mthub 不暴露 MT4/5 具体类型。
- `market-data` 通过 adapter 的 `recvLoop` 接收报价流，handler 回调写入 NATS。报价数据的类型转换在 adapter → mdtick 边界完成。清晰。

**隐患**：`connection_account.go` 暴露了 `MT4Client()`/`MT5Client()` 方法，绕过 mthub 直接返回原始 gRPC stub。如果有代码直接调用这些方法，就绕开了 mthub 的幂等门和风控管线。目前只在测试代码中使用——需要加 golangci-lint 规则禁止生产代码调用这两个方法。

## 3. 已知架构债务

| 债务 | 严重度 | 方案 |
|------|--------|------|
| 熔断器未接入 | 🔴 | 已写入 plans/rpc-expansion.md |
| 无告警/降级/策略暂停 | 🔴 | 韧-1/2/3，已写入 plans/rpc-expansion.md |
| `MT4Client()`/`MT5Client()` 暴露 | 🟡 | 加 lint 规则禁止生产代码调用 |
| 10 个 RPC 未封装 | 🟡 | P0-P3，已写入 plans/rpc-expansion.md |
| 每账户独立 gRPC 连接，无连接复用 | 🟢 | 当前体量不需要。同一 broker 多账户的场景出现后再优化 |

## 4. 总评

架构是最优的。问题不在设计——在实现完整性。熔断器接线、韧性增强、RPC 补齐做完后，这个块就是完整且有韧性的。优先级：熔断器接入 > 告警/降级 > RPC 扩展。
