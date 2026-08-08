# Runbook · MdGatewayNormalizerFallback

> 占位。告警：MdGatewayNormalizerFallback（normalizer 退化到 fallback 模式）。

**含义**：symbol normalizer 的 PG LISTEN 缓存失效，退化到全量扫描或硬编码映射。
**影响**：symbol 解析可能不准/慢 → 错品种下单风险。
**初步**：查 PG LISTEN 连接状态 → normalizer 日志 → 确认缓存失效原因（PG 重启/LISTEN 断开）→ 触发缓存重建。
**TODO**：normalizer 缓存重建 SOP + LISTEN 健康监控 + fallback 降级阈值。
