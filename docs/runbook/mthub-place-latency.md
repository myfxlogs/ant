# Runbook · MtHubPlaceLatencyP99High

> 占位（post-launch 补全）。关联告警：`MtHubPlaceLatencyP99High`（`mthub_place_latency_seconds` p99 >2s，5m）。

## 含义
从策略信号到订单落 broker 的延迟 P99 过高。

## 影响
滑点增大、实盘成交价偏离信号价 → 实盘 vs 回测发散、策略表现劣化。

## 初步处置
1. Grafana 看 latency 是普遍涨还是单 broker。
2. 查 mtapi.io 代理延迟（`docker logs` 看 mtapi RPC 耗时）。
3. 查本机负载（CPU/连接池）是否瓶颈。

## TODO（post-launch）
分阶段延迟拆解（Gate/OMS/mtapi）、SLO 阈值、降级（kill switch）触发条件。
