# Runbook · BackendHighMemory

> 占位（post-launch 补全）。关联告警：`BackendHighMemory`（容器内存接近上限）。

## 含义
backend 内存占用过高，逼近 OOM。

## 影响
OOM kill 风险 → 服务重启/中断。

## 初步处置
1. `docker stats` 看内存趋势；`go tool pprof`（若暴露）抓 heap。
2. 常见嫌疑：bar 缓存窗口（maxContextBars）、活跃 strategy session 内存、SSE 连接堆积。
3. 临时：重启 backend 释放；根因：缩缓存窗口 / 修泄漏。

## TODO（post-launch）
内存基线、pprof 接入、各缓存上限调优、OOM 告警阈值。
