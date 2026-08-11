# Runbook · MtHubPlaceLatencyP99High

> 关联告警：`MtHubPlaceLatencyP99High`（`mthub_place_latency_seconds` p99 >2s，持续 5m，severity=warning）

## 症状

`histogram_quantile(0.99, rate(mthub_place_latency_seconds_bucket[5m])) > 2.0` 持续触发。订单从信号到落 broker 的端到端延迟 P99 超过 2 秒。

## 影响

滑点增大——策略信号价与实际成交价偏差扩大。实盘 vs 回测发散，策略表现劣化。高频策略影响尤甚。

## 诊断步骤

```bash
# 1. 确认是普遍涨还是单 broker
curl -s http://localhost:8080/metrics | grep 'mthub_place_latency_seconds_bucket' | awk -F'{' '{print $2}' | sort | uniq -c | sort -rn

# 2. 查 mtapi RPC 耗时分布
docker logs alphaforge-backend --since 10m | grep -iE "place.*order.*latency|mtapi.*duration|OrderSend.*took"

# 3. 查本机负载
docker stats --no-stream alphaforge-backend
# CPU / 内存 / 网络

# 4. 查 PG 连接池是否瓶颈（影响 Gate 检查速度）
curl -s http://localhost:8080/metrics | grep 'pgxpool_pool_total_conns'
```

## 应急处置

1. **mtapi.io 代理延迟** → 检查到 mtapi.io 的网络延迟：
   ```bash
   docker exec alphaforge-backend ping -c 10 mt4grpc.mtapi.io
   ```
   若延迟 >500ms，可能是网络路由问题或 mtapi.io 负载高。
2. **本机 CPU/内存瓶颈** → 扩容或减少并发策略数
3. **PG 连接池接近耗尽** → 参考 `pg-pool-exhausted.md` runbook
4. **单 broker 持续高延迟** → 该 broker 服务器慢，考虑切换到更快 broker 或降低策略频率

## 常见根因

- **mtapi.io 代理慢**：mtapi.io 是 MT4/MT5 的 gRPC 代理层，其延迟受上游 broker 服务器和网络路径影响
- **broker 服务器慢**：MT broker 在高波动时段（如 NFP 数据发布）处理延迟增大
- **本机资源瓶颈**：CPU 被大量并发策略占满，导致 `submitOrder` goroutine 调度延迟
- **PG 连接池竞争**：Gate 风控检查需要查 PG，连接池接近耗尽时排队等待
