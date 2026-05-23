# Ant 项目路线图

> 最后更新: 2026-05-23

---

## M7: alfq 迁移 — ✅ 完成

| 里程碑 | 状态 | 交付 |
|--------|------|------|
| M7.1 市场数据网关 (mdgateway) | ✅ 已完成 | 类型本地化、CH 写入、质检、bar 聚合、spill replay、adapter/mt4、adapter/mt5、backfill。NATS JetStream publisher + 熔断器 + SpillWriter。 |
| M7.2 因子引擎 (factorsvc) | ✅ 已完成 | 因子计算引擎、DSL 编译器、窗口缓冲区、订阅分发、CH 写入。`tenant_id` → `user_id` 全局替换。 |
| M7.3 因子 DSL (factor/dsl) | ✅ 已完成 | Go 端 DSL 引擎（lexer/parser/compiler/14 operators）。Python 端 DSL 就位。Go↔Python 对齐测试。 |
| M7.4 研究环境 (research) | ✅ 已完成 | Python DSL 引擎、回测框架桩、CLI、对齐测试数据。 |
| M7.8 装配与收尾 | ✅ 已完成 | runner.go、启动钩子、proto 9 RPC、ConnectRPC handler、CH config、TTL、ADR-0014、4 端点切流标记、kline_service Deprecated、3 测试文件 + quality labels。 |

### M7.8 提交记录

| 卡片 | SHA |
|------|-----|
| M7.8-1/2/5/6 — runner + hook + config + TTL + ADR-0014 | `8de43c6` |
| M7.8-3/4 — proto 9 RPC + ConnectRPC handler | `2e9e250` |
| M7.8-7/8/9 — 4 endpoint 切流 + kline_service Deprecated | `8c30039` |
| M7.8-10~15 — tests + NATS TODO + quality labels + backfill MT5 | `66a89f7` |

### 架构升级（重构 + 缺陷修复）

| 提交 | SHA |
|------|-----|
| 架构升级: NATS/熔断/Spill/Prometheus/OrderExecutor | `42921c3` |
| 重构: 消除 MT 代码冗余 | `c0a5984` / `e719ff8` |
| 修复: 5 个缺陷（数据竞争/死指标/nil防护/gauge漂移/静默失败） | `19925f2` |

---

## 已完成

| 里程碑 | 状态 |
|--------|------|
| M0.1 工程基线 | ✅ |
| M0.2 文档与 ADR | ✅ |
| M0.3 复杂度与品质 | ✅ |
| M1 canonical symbol | ✅ |
| M2 OMS 状态机 | ✅ |
| M3 风控引擎 | ✅ |
| M4 AI → canonical | ✅ |
| M5 策略市场 | ✅ |
| M6 部署上线 | ✅ |
| M7 量化基础设施 | ✅ |

---

## 待办

| 项目 | 优先级 | 说明 |
|------|--------|------|
| `make proto` 生成 ConnectRPC stub | P0 | proto/mthub.proto 需要 buf generate |
| 重建镜像 + 重启容器 | P0 | 代码变更需重新构建生效 |
| CH 查询路径实际切换 | P1 | 4 个 connect endpoint 的 TODO → 实现 |
| NATS server 部署 | P1 | publisher 已实现，需 NATS 运行时 |
| adapter/mt4 mt5 集成测试 | P1 | 需真实 MT 连接 |
| Prometheus `/metrics` endpoint | P2 | 指标已注册，需暴露 HTTP handler |
| kline_service 下线 | M8 | 7 个文件已标记 Deprecated |
