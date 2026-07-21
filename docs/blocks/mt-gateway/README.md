# mt-gateway — MT4/MT5 网关

> 连接 MT4/MT5 交易服务器，提供下单、持仓、行情、账户事件的统一接口。

## 代码位置

```
backend/mt{4,5}/                                ← proto stubs（mtapi.io 生成，勿手动改）
backend/internal/mdgateway/adapter/mt{4,5}/      ← adapter 封装（平台特定逻辑）
backend/internal/mdgateway/adapter/mdtick/        ← 共享 DTO（Tick/Bar/OrderUpdate/AccountConfig）
backend/internal/mthub/                           ← 中枢（幂等门、对账门、OMS、状态缓存）
```

## 架构

三层：proto stub → adapter → mthub。三层边界干净，mthub 不引用 MT4/MT5 具体类型。

**MT4/MT5 adapter 代码不共享**——两个平台协议差异巨大（订单类型枚举不同、字段类型不同、子消息结构不同、错误码体系不同），共享会引入平台判断分支，比独立维护更危险。详见 `docs/spec/16-mtapi-quirks-register.md`。

## 依赖 / 被依赖

```
market-data(行情推送) → mt-gateway
account-mgmt(账户凭据) → mt-gateway
mt-gateway → risk-gate(下单前风控) → oms(状态机) → mt-gateway(执行)
```

## 关键设计

- 重连：指数退避（1s → 5min max），每账户独立 gRPC 连接
- 订单幂等：三层（PG advisory lock + Redis SETNX 96h + broker magic）
- 熔断器：已实现但未接入（`circuit_breaker.go`），需修复
- MT4 RPC 49 个 / MT5 RPC 75 个，当前封装率 ~53%，核心交易/行情管线完整

## 施工计划

- [plans/rpc-expansion.md](plans/rpc-expansion.md) — 10 个新 RPC 封装 + 熔断器接入

## 关联文档

- [spec/10-mt-adapter.md](spec/10-mt-adapter.md)
- [spec/16-mtapi-quirks-register.md](spec/16-mtapi-quirks-register.md)
