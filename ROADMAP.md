# Ant 项目路线图

> 最后更新: 2026-05-23

---

## M7: alfq 迁移

| 里程碑 | 状态 | 说明 |
|--------|------|------|
| M7.1 市场数据网关 (mdgateway) | 🚧 in_progress（部分已迁移） | 类型定义、Tick/Bar 本地化、CH 写入、质检、bar 聚合、spill replay 已完成。adapter/mt4、adapter/mt5 已桥接现有 mt4client/mt5client，backfill PG→CH 已实现。NATS publisher 为 no-op stub。 |
| M7.2 因子引擎 (factorsvc) | 🚧 in_progress（部分已迁移） | 因子计算引擎、DSL 编译器、窗口缓冲区、订阅分发、CH 写入已完成。`tenant_id` → `user_id` 全局替换完成。 |
| M7.3 因子 DSL (factor/dsl) | 🚧 in_progress（部分已迁移） | Go 端 DSL 引擎完整迁移（lexer/parser/compiler/operators）。Python 端 DSL (`research/`) 已就位。Go↔Python 对齐测试框架已搭建。 |
| M7.4 研究环境 (research) | 🚧 in_progress（部分已迁移） | Python DSL 引擎、回测框架桩、CLI、对齐测试数据就位。 |

---

## 已完成

| 里程碑 | 状态 |
|--------|------|
| — | — |

---

## 待办

| 项目 | 优先级 |
|------|--------|
| mdgateway adapter mt4/mt5 单元测试 | P1 |
| mdgateway chmigrate 单元测试 | P2 |
| NATS publisher 真实现 | P2 |
| mt4client/mt5client 公开 Conn()/SessionID() | P2 |
| factor_service 集成测试（CH 实际写入） | P2 |
