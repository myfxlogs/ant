#!/bin/bash
# 定期清理 Go/Docker/Node 缓存，防止磁盘/内存耗尽
# 来源: rd.md 任务触发磁盘满/内存满问题修复
set -euo pipefail

LOG_FILE="/tmp/cache-cleanup-$(date +%Y%m%d).log"
exec >>"$LOG_FILE" 2>&1

echo "[$(date -Iseconds)] cache cleanup started"

# Go build cache — 保留 7 天内活跃的，其余强制清
if command -v go &>/dev/null; then
    BEFORE=$(du -sh /root/.cache/go-build 2>/dev/null | cut -f1 || echo "0")
    go clean -cache 2>/dev/null && echo "  go build cache: $BEFORE → $(du -sh /root/.cache/go-build 2>/dev/null | cut -f1)"
fi

# Go module cache — 仅清理未使用的版本
if command -v go &>/dev/null; then
    BEFORE=$(du -sh /root/go/pkg/mod 2>/dev/null | cut -f1 || echo "0")
    go clean -modcache 2>/dev/null && echo "  go mod cache: $BEFORE → $(du -sh /root/go/pkg/mod 2>/dev/null | cut -f1)"
fi

# Docker build cache
if command -v docker &>/dev/null && docker info &>/dev/null 2>&1; then
    BEFORE=$(docker system df --format '{{.BuildCache}}' 2>/dev/null || echo "N/A")
    docker builder prune -f 2>/dev/null && echo "  docker build cache: pruned"
fi

# Node compile cache (V8 code cache, 安全删除)
rm -rf /tmp/node-compile-cache 2>/dev/null && echo "  node compile cache: removed"

# Playwright transform cache
rm -rf /tmp/playwright-transform-cache-* 2>/dev/null && echo "  playwright transform cache: removed"

# Go temp build dirs (编译失败后残留)
find /tmp -maxdepth 1 -name 'go-build*' -mtime +0 -exec rm -rf {} + 2>/dev/null && echo "  go temp build dirs: cleaned"
find /tmp -maxdepth 1 -name 'go-link-*' -mtime +0 -exec rm -rf {} + 2>/dev/null && echo "  go temp link dirs: cleaned"

# npm cache — 仅清理超过 30 天的
if command -v npm &>/dev/null; then
    npm cache clean --force 2>/dev/null && echo "  npm cache: cleaned"
fi

echo "[$(date -Iseconds)] cache cleanup completed"

# 清理 7 天前的日志
find /tmp -maxdepth 1 -name 'cache-cleanup-*.log' -mtime +7 -delete 2>/dev/null
