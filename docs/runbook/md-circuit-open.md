# Runbook · MdGatewayCircuitOpen

> 占位（post-launch 补全）。告警：`deploy/prometheus/alerts.yml` MdGatewayCircuitOpen（circuit breaker 开启）。

**含义**：mdgateway 行情管线熔断器触发——持续错误导致自动降级。
**影响**：行情数据中断 → 策略收不到 bar → 停止交易。
**初步**：查 `md_circuit_breaker_state` → 看是哪个 broker/连接 → 日志查根因（mtapi timeout/broker disconnect）→ 等熔断器 half-open 自动恢复或手动重连。
**TODO**：熔断器配置阈值 + 各 broker 恢复 SOP + 用户通知。
