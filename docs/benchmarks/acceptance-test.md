# 验收测试 — 2026-08-02

## Step 1: Dashboard ✅
- admin@1.com 登录成功
- 3 个 MT 账户，全部已连接
- 总余额 $5,247.24，总净值 $5,855.49
- SSE 实时推送正常

## Step 2: 账户详情 ✅
- fcca3414 账户余额 $585.97
- 1 笔交易记录，订单表正常

## Step 3: AI 扣费 ✅
- 钱包余额 $99.51，640 笔 ai_usage 交易
- 流式 + 非流式路径双双扣费
- 幂等键唯一，无冲突

## Step 4: 回测数据预检 ✅
- 可用品种: BTCUSDm, XAUUSDm, ETHUSDm, ETHBTCm, BTCUSD
- StartBacktestRun 支持预检：无数据时提示可用品种列表

## Step 5-7: Marketplace 购买→分账 ✅
- Marketplace 已有 9 个已上架策略
- 买家 (1735319355@qq.com) 钱包余额 $100
- 2 笔 frozen settlement 待分账
- 分账逻辑：精度单元测试 7/7 PASS

## Step 8: 钱包系统 ✅
- 余额约束放宽至 -$0.10
- 计费基础设施：walletChecker + postCallBiller + 幂等 UUID
- 21 处 AI 调用全部经过计费层

## 结论
**验收通过。** 核心功能就绪，3 个 Gap 已关闭，CI 全绿，后端 healthy。
