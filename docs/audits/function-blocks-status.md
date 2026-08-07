# 功能块实现状态总览

> **⚠️ 这是 2026-07-20 的实现快照（建立时），部分已过时。** 当前权威状态以：
> - [`handover-audit-plan.md`](handover-audit-plan.md) — 管线审计进度（逐管线代码级核验，最新事实）
> - [`tech-debt-registry.md`](tech-debt-registry.md) — 已核验 gap 清单
>
> 为准。本表仅作「功能块规模/范围」的快速基线参考，不作「是否无 bug」的定论。

## 11 功能块（2026-07-20 基线）

| # | 块名 | 实现状态 | 规模/范围 |
|---|------|---------|----------|
| 1 | `mt-gateway` | ✅ 完整 | MT4/MT5 双网关 via mtapi.io，连接/认证/下单/持仓/报价流/订单事件全闭环。`backend/mt{4,5}/` + `backend/internal/mthub/`（35+ 文件） |
| 2 | `strategy-runtime` | ✅ | `backend/strategy/{sdk,runner,backtest,indicators}/`。Strategy 接口 + LiveRunner + 50+ 技术指标库 |
| 3 | `mql-compiler` | ✅ | `backend/tools/mql2go/`（55+ 文件）。tree-sitter → IR → Bytecode → VM。277+ 内置函数 |
| 4 | `agent-engine` | ✅ | `backend/internal/{agent,ai}/`（87+ 文件）。Agent 循环 + 盲区桥接 + 三层记忆 + 蒙特卡洛/walkforward/优化器 |
| 5 | `backtest-engine` | ✅ | `backend/internal/backtest/` `backend/strategy/backtest/`。SimBroker + 撮合 + 净值曲线 + 多品种 |
| 6 | `risk-gate` | ✅ | `backend/internal/{risk,risksvc,paper,oms}/`。6 门管线 + 11+ 规则 + 16 状态机 OMS |
| 7 | `account-mgmt` | ✅（待复审） | `backend/internal/connect/{gateway,user}/`。MT 账户 CRUD + 认证 + WebAuthn + 钱包 |
| 8 | `market-data` | ✅ | `backend/internal/{mdgateway,source,symbol}/`（60+ 文件）。tick 去重 + 回填 + PG/CH/NATS |
| 9 | `frontend` | ✅ | `frontend/src/pages/`（17 页面目录）。工作区/管理后台/市场/交易/钱包 |
| 10 | `api-gateway` | ✅ | 52 gRPC 服务 / 335 RPC / 14 Connect handler 目录 / 6 SSE 流 |
| 11 | `strategy-marketplace` | ✅ | `backend/internal/marketplace/`（15 文件）+ `connect/marketplace/`。发布/购买/实盘跟踪/批量生成 |

基础设施：Vault 加密、HD 钱包冷签名、Tron 链上监控、对账引擎、85+ repository、35+ service、220 migration、8 CLI 命令。

## 待复审的旧断言

- **account-mgmt「已完工、架构最优、无 BUG」**（2026-07 断言）：该块在管线审计中标 ⬜未审（account-mgmt 不在 7 管线内，单独轻审待做）。"无 BUG" 为旧断言，复审前不作为定论。
- **各块「✅ 完整」**：仅表示 2026-07-20 时功能闭环，不等于无 bug。管线审计（#1~#7）已在多个"✅ 完整"块中发现真实 gap（见 tech-debt-registry.md：LIVE-1/RECON-1/BT-6 等）。
