# Runbook · PGPoolExhausted

> 占位（post-launch 补全）。关联告警：`PGPoolExhausted`（`pgxpool` 活跃连接接近上限）。

## 含义
PG 连接池耗尽——长事务/连接泄漏/并发暴涨导致无可用连接。

## 影响
新请求阻塞、超时、级联故障（DB 是真相源）。

## 初步处置
1. `SELECT * FROM pg_stat_activity` 看长事务/空闲连接。
2. `docker logs alphaforge-backend | grep -i "pool\|conn"` 看连接获取失败。
3. 定位泄漏点（未关 tx / 慢查询）；临时调大池或重启 backend 释放。

## TODO（post-launch）
连接池监控面板、慢查询识别、泄漏检测、容量规划。
