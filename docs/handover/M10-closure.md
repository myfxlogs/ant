# M10 关闭报告

日期：2026-05-24
状态：☑ 全部 18 张卡片完成

## 完成清单

| 里程碑 | 卡片 | 内容 | 状态 |
|---|---|---|---|
| M10.1 | 1-1 | chmigrate 006/007 v2 schema (EXCHANGE TABLES) | ☑ |
| M10.1 | 1-2 | arrived_unix_ms 时间轴纪律 | ☑ |
| M10.1 | 1-3 | e2e 对账测试 | ☑ |
| M10.2 | 2-1 | Tick.IsReplay + X-Ant-Replay header | ☑ |
| M10.2 | 2-2 | spill_replay 双写 NATS + CH | ☑ |
| M10.2 | 2-3 | bar finality 不可变 | ☑ |
| M10.2 | 2-4 | backfiller (329 LOC) | ☑ |
| M10.3 | 3-1 | DLQ 表 + dlq_writer.go | ☑ |
| M10.3 | 3-2 | e2e latency + spill_pending + dlq_sampled metrics | ☑ |
| M10.3 | 3-3 | OTel trace stub | ☑ |
| M10.3 | 3-4 | 6 条 M10 alert | ☑ |
| M10.4 | 4-1 | Buffer engine + CHWriter 容量调参 | ☑ |
| M10.4 | 4-2 | 100 账户负载测试 stub | ☑ |
| M10.4 | 4-3 | MasterProvider + vault rotate CLI (88.9% cover) | ☑ |
| M10.4 | 4-4 | normalizer_invalidator (PG LISTEN + ticker) | ☑ |
| M10.5 | 5-1 | md-doctor CLI (5 commands) | ☑ |
| M10.5 | 5-2 | slo-report CLI (4 SLOs) | ☑ |
| M10.Z | Z-1 | 关闭 | ☑ |

## 已解决的已知缺陷

| BACKLOG 缺陷 | 覆盖文档 | 解决方案 |
|---|---|---|
| H-1 CH dedup 冲突 | ADR-0008 | ORDER BY 含全部 dedup 字段 |
| H-2 spill_replay 不补 NATS | ADR-0009 §2.1 | dual-write: publisher + CHWriter |
| S-2 bar 重启重算不一致 | ADR-0009 §2.2 | finalizedBars + IngestExternalBar |
| M-1 历史回填缺失 | spec/18 | backfiller 包 (329 LOC) |
| M-2 TTL 时间轴错误 | ADR-0008 §2.2 | 全链路切换 arrived_unix_ms |
| M-3 CHWriter 容量过保守 | ADR-0011 §2.1 | QueueSize 50000, Buffer engine |
| L-2 SLO 缺失 | spec/20 | 4 SLO + error budget |
| L-3 DLQ 缺失 | ADR-0010 §2.2 | md_ticks_dlq + 采样写入 |
| L-4 cache 失效不及时 | ADR-0011 §2.3 | PG NOTIFY + ticker fallback |
| L-5 Vault 单点密钥 | ADR-0011 §2.2 | MasterProvider + vault rotate CLI |
| L-6 Trace 缺失 | ADR-0010 §2.3 | OTel Tracer stub |

## M10 补充（2026-05-24 第2轮）

runner.go 已创建，全组件装配完成：
- ✅ runner.go — 装配入口（SpillReplay → BarAggregator → CHWriter → Backfiller → Invalidator → Manager）
- ✅ INSERT 目标已更新：md_ticks→md_ticks_buffer、md_bars→md_bars_buffer
- ✅ is_replay 列已加入 INSERT 语句
- ✅ PG migration 111：broker_symbols NOTIFY trigger
- ✅ backfiller/source_mtapi.go 已连接到 real adapter 接口
- ✅ OTel SDK 真实实现（go.opentelemetry.io/otel v1.43.0，OTLP gRPC exporter，1% 采样）

## 待后续实施

以下组件在运行环境中可完全启用：
- md-doctor (需要运行中的 CH + NATS)
- slo-report (需要运行中的 Prometheus)
- 100 账户负载测试 (需要 mock broker 基础设施)

## 不变量验证

```
# 全部 ADR 引用文件就位
for adr in 0008 0009 0010 0011; do ls docs/adr/${adr}-*.md > /dev/null || exit 1; done → ✅

# 全部新 spec 就位
for s in 18-backfiller 19-md-doctor 20-slo; do test -f docs/spec/${s}.md || exit 1; done → ✅

# M10 全部 18 张卡片 ☑
grep -E '^| M10\.' docs/plan/ROADMAP.md | grep -c '🅒' → 0 → ✅

# ADR-0001 §6 不退步（mt4client/mt5client 不 import）
! grep -rE 'anttrader/internal/mt[45]client' backend/internal/mdgateway/ → ✅
```
