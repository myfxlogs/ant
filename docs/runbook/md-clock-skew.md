# Runbook · MdGatewayClockSkew

> 占位。告警：MdGatewayClockSkew（时钟偏差 >5s）。

**含义**：服务器时钟与 broker/MT 时钟偏差过大。
**影响**：bar 时间戳不准 → bar 归属错误 → 回测/实盘发散。
**初步**：`chronyc tracking` 或 `timedatectl` 查 NTP 同步状态 → 确认是否 NTP 服务挂了 → 修复 NTP 同步。
**TODO**：NTP 监控 + 时钟偏差自动告警阈值 + bar 聚合容差调优。
