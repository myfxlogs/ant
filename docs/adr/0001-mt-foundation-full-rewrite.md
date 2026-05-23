# ADR-0001 · MT 基础完全重写（路线 B）

- **状态**：Accepted
- **日期**：2026-05-23
- **关联 spec**：`docs/architecture/02-overview.md`、`docs/spec/10-16`

## 1. 背景

ant 项目原架构（v1）将 MT 接入定位为"K 线源 + 下单通道"，导致以下问题：

| 维度 | 现状（v1） | 量化 |
|---|---|---|
| MT 层 LOC | mt4client/mt5client/service/kline_service/connect/market_service 累计 | ~4500 行 |
| 数据存储 | PostgreSQL `kline_data` 单源（TIMESTAMP + UNIQUE） | 不支持 tick / 100M+ 行查询慢 |
| 多 broker | 同 symbol 不同后缀直接入库（`BTCUSDm`/`BTCUSD.pro`） | canonical 不一致 |
| 数据质量 | 无 quality 检查 | 脏 tick 直接污染下游 |
| 故障恢复 | 无 spill / 无熔断 | broker 抖动即丢数据 |
| 单一路径 | 业务代码直 import `internal/mt4client` | 上层耦合到协议细节 |

参考 alfq 项目（同 owner，6 天迭代 129 commits）：
- mtapi 关键暗坑已修复并文档化（OnQuote.Time、TradeMode、cross-broker symbol 等）
- 行情链路分层：adapter → mdgateway → factorsvc → quantengine → oms
- ClickHouse 作为时序，PG 作为业务

ant 项目当前**未上线、无外部用户**，是地基重写的最佳时机。

## 2. 决策

**采用路线 B：地基完全重做，业务渐进重构**。

具体范围：
- 🔥 **完全重做**（≤ 600 行替代 ~4500 行）：MT 接入（adapter）、行情网关（mdgateway）、ClickHouse 时序、因子 DSL、quantengine、订单 hub（mthub）
- 🔄 **重构**：`connect/` 切到新地基；`kline_service*` 删除；`broker/` 重写为薄壳
- ✅ **保留**：AI 助手、策略市场、admin、auth、worker、frontend、user/tenant 业务表
- 🗑️ **重设计 schema 子集**：`kline_data` `tick_data` 等行情表迁到 ClickHouse；65 张业务表保留

**不采用路线 A（完全重写）**：会丢失 AI/marketplace/53 页前端等已完成产品功能，且引入"第二系统效应"的高风险技术债。

## 3. 备选方案

| 方案 | 优点 | 缺点 | 否决理由 |
|---|---|---|---|
| A · 完全重写（含业务）| 架构最纯净 | 4-6 月停滞、丢失 53 页前端、ETL 风险、第二系统效应 | 投入产出比差 |
| **B · 地基重做 + 业务渐进**（采纳）| 地基达 alfq 水平 + 保留业务 + 风险可控 | 边界期需小心切流 | 最优 ROI |
| C · M7 + M7.8 + M9 渐进 | 风险最低 | 永久背 ~4500 行 mt4client 包袱 | 长期债太大 |

## 4. 后果

### 正面
- MT 接入总 LOC 从 ~4500 → ≤ 1000（可量化目标）
- 数据精度从 "PG TIMESTAMP" 提升到 "CH UInt64 ms"
- 增加：CircuitBreaker（局部超越 alfq）、Spill 旋转、Tick dedup、quirks register
- 业务功能（AI / market / admin）零损失
- 业务代码 0 处直调 mt4client/mt5client

### 负面
- M7-rewrite 期间地基/业务边界存在过渡期混合（约 5-8 个 service 文件需切流）
- 老 `kline_data` 表保留至 M9 才删除（占用 ~30 天技术债期）
- 业务表风格不统一（v1 部分 sqlc / 部分裸 SQL）需 M8/M9 渐进治理

### 中性
- 文档全部重写为 v2（docs.old/ 归档保留）
- AGENT.md 改为 AI 单一执行约束源

## 5. 实施约束

### 5.1 LOC 硬指标（B 路线长期硬指标）

| 指标 | M7 完成 | M8 完成 | M9 完成 |
|---|---|---|---|
| MT 接入总 LOC（非测试，与 AGENT.md §5 一致） | ≤ 1500 | ≤ 1200 | ≤ 1000 |
| 业务代码 grep 直调 mt4client/mt5client | 0 | 0 | 0（包已删除）|
| service/ 包文件 ≤ 400 行 | — | 100% | 100% |
| sqlc 覆盖率 | — | ≥ 80% | ≥ 95% |
| 生产路径 Python grep | 0 | 0 | 0 |

### 5.2 不变量

1. canonical 在 adapter 出口完成
2. 生产路径零 Python
3. 价格全程 `decimal.Decimal`，禁 float64
4. 时间统一 UTC 毫秒
5. 每个 Tick 必经 Quality.Check
6. CH 写入不阻塞订阅链路
7. 每张 ☑ 卡片必有 commit + 验收 log

### 5.3 引用关系

- 实施计划：`docs/plan/ROADMAP.md` §M7-rewrite
- 模块规范：`docs/spec/10-16`
- 验收手册：`docs/runbook/mt-incidents.md`
- AI 执行约束：`AGENT.md`

## 6. 验证方式

里程碑关闭时执行：

```bash
# (1) MT 接入 LOC
LOC=$(find backend/internal/mdgateway backend/internal/mthub \
            backend/internal/mdgateway/adapter \
       -name "*.go" -not -name "*_test.go" \
       | xargs wc -l | tail -1 | awk '{print $1}')
case "$MILESTONE" in
  M7) test "$LOC" -le 1500 ;;
  M8) test "$LOC" -le 1200 ;;
  M9) test "$LOC" -le 1000 ;;
esac

# (2) 业务代码 0 处直调（service/ 是 deprecation 区，豁免至 M9）
! grep -rE 'anttrader/internal/(mt4|mt5)client' backend/internal/{ai,marketplace,oms,risk,connect,quantengine,factorsvc,mthub}/

# (3) 生产路径零 Python
! grep -rE 'exec\(|eval\(|subprocess|sandbox' backend/internal/{quantengine,oms}/

# (4) ClickHouse 实写
docker exec ant-clickhouse clickhouse-client --query \
  "SELECT count() FROM md_ticks WHERE arrived_unix_ms > now64()*1000 - 60000" \
  | awk '$1<10{exit 1}'
```

任一断言失败 → milestone 不许 ☑。
