# 服务器资源限制与缓存清理

> 服务器：4 核 / 8 GB / 58 GB。Go 工具链（lint/test/build）默认吃满内存 → OOM 死机。以下限流已应用（2026-08-01）。所有在此服务器跑全量构建的 AI 工具/人都适用。

## 问题

全量 `golangci-lint run ./...` / `go test -race ./...` / `docker compose build` 时，Go 编译缓存无上限增长（~10 GB）+ Go 工具链无内存限制 → swap thrashing → 服务器死机。

## 已应用的限流

### 环境变量（`/root/.bashrc`）

- `export GOMEMLIMIT=4096MiB` — Go 堆软上限 4 GB，超限时 GC 更激进
- `alias golangci-lint='golangci-lint --concurrency 2'` — lint 限 2 并发（默认 = CPU 核数，吃光内存）
- `alias gotest='go test -p 2 -count=1'` — 限制测试并行度
- `alias gotestshort='go test -short -p 2 -count=1'`

### 清理脚本 `scripts/cleanup-caches.sh`

go build cache + go mod cache + docker builder prune + `/tmp/go-build*` / `/tmp/go-link-*` / node compile cache 残留。

### 系统 crontab

`37 3 * * 0 /opt/ant/scripts/cleanup-caches.sh` — 每周日凌晨 3:37 自动清理。

## 注意

- `GOMEMLIMIT=4096MiB` 是**软限制**，极端场景 Go 仍可能短暂超过，但立即触发 GC。
- `golangci-lint --concurrency 2` alias 仅对**交互式 shell** 生效；CI 中需显式传参。
- 容器内存限制合计 ~1.5 GB（backend 192M / frontend 256M / PG 256M …），**Docker build 不受容器限制**（在宿主机跑）。
- 仍 OOM → 进一步降 `GOMEMLIMIT` 至 `3072MiB` 或设 `GOGC=50`。

## 相关

- `rd.md` — 质量验收标准文档（全量 lint/test/build 触发源）
- `/root/.bashrc` — Go 环境变量
- `scripts/cleanup-caches.sh` — 定期清理脚本
