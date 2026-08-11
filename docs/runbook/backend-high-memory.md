# Runbook · BackendHighMemory

> 关联告警：`BackendHighMemory`（`process_resident_memory_bytes > 180MB`，持续 5m，severity=warning）

## 症状

`process_resident_memory_bytes{job="alphaforge-backend"} > 180 * 1024 * 1024` 持续 5 分钟。容器内存使用超过 180MB（限制 192MB），逼近 OOM。

## 影响

OOM kill 风险——内存达到 192MB 限制后容器被 kernel 杀死，服务中断。可能触发 `BackendDown` 告警。

## 诊断步骤

```bash
# 1. 确认内存使用趋势
docker stats --no-stream alphaforge-backend

# 2. 查看容器内存限制
docker inspect alphaforge-backend | grep -i memory

# 3. 查后端日志中的内存相关警告
docker logs alphaforge-backend --since 10m | grep -iE "memory|oom|alloc|gc"

# 4. 检查是否有大量并发 SSE 连接
curl -s http://localhost:8080/metrics | grep -iE 'sse.*conn|stream.*active|subscriber.*count'

# 5. 检查活跃策略 session 数量
curl -s http://localhost:8080/metrics | grep -iE 'strategy.*active|session.*count'
```

## 应急处置

1. **立即降险** → 重启 backend 释放内存：
   ```bash
   docker compose restart backend
   ```
   注意：重启会断开所有 SSE 连接，用户需刷新页面。
2. **SSE 连接堆积** → 检查是否有异常多的长连接：
   ```bash
   docker exec alphaforge-backend sh -c "ls /proc/1/fd | wc -l"
   ```
   若 fd 数量异常高，可能有 SSE 连接泄漏。
3. **bar 缓存窗口过大** → 检查 `maxContextBars` 配置，降低缓存窗口大小
4. **抓 heap profile** → 若 pprof 已暴露：
   ```bash
   curl -s http://localhost:8080/debug/pprof/heap > heap.prof
   go tool pprof heap.prof
   ```

## 常见根因

- **bar 缓存窗口过大**：每个活跃策略 session 维护 `maxContextBars` 历史 bar 缓存，并发 session 多时内存线性增长
- **SSE 连接泄漏**：客户端断开后服务端 goroutine 未退出，连接 fd 和缓冲区累积
- **Go runtime GC 不及时**：大量小对象分配后 GC 未触发，RSS 持续增长
- **PG LISTEN 连接占用**：每个 SSE stream 的 `pgListen.Listen` 占用一个 pool conn，大量并发 stream 时 PG 连接内存累积
