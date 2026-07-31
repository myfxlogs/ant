commit and push it, then deploy
prompt
playwright

1. 全面扫描代码，罗列所有逻辑漏洞、业务异常、安全风险、兼容问题；
2. 标注每处问题的风险等级、影响范围、触发场景、危害后果；
3. 修复时遵循原架构逻辑，仅处理问题，不改动无关代码和正常功能；
4. 逐行比对修改前后代码，输出版本差异明细；
5. 完成全链路回归测试，核验功能正常无新增Bug，输出审计报告；
6. 输出合规代码文件，附带修改日志和风险复盘文档

逐项审计（架构 / 最优性 / 第一性原则 / 代码整洁 / 技术债）开始
存在的lint需要修复，不能一直存在。

---

## 服务器资源配置说明

### 当前服务器规格
- **CPU**: 4 核
- **内存**: 8 GB (+ 4 GB swap)
- **磁盘**: 58 GB (已用 32 GB, 剩余 27 GB)

### 运行时资源消耗 (docker stats 实测)

| 容器 | 实际内存 | 内存上限 | CPU | 说明 |
|------|---------|---------|-----|------|
| backend | ~49 MB | 192 MB | ~3% | Go 单二进制, CGO tree-sitter |
| postgres | ~63 MB | 256 MB | ~0% | PG 18, 市场数据 + 用户数据 |
| umami | ~53 MB | 256 MB | ~0% | 网站分析, 可选 |
| nats | ~12 MB | 128 MB | ~5% | JetStream tick/bar 流 |
| redis | ~7 MB | 256 MB | ~0% | 最新报价缓存 (maxmemory 200mb) |
| frontend | ~2 MB | 256 MB | ~0% | Nginx 静态文件 |
| **合计** | **~186 MB** | **1.3 GB** | | |

### 磁盘占用 (docker system df 实测)

| 类型 | 大小 | 说明 |
|------|------|------|
| Docker 镜像 | 5.0 GB | 6 个镜像 (backend 529MB, umami 1.28GB, PG 409MB 等) |
| Build Cache | 2.5 GB | Go CGO 编译缓存, 每次 build 产生 2-3GB |
| Volumes | 377 MB | PG 数据 + Redis AOF + NATS JetStream |
| Containers | 136 MB | 运行时可写层 |
| **合计** | **~8 GB** | |

### 为什么需要 58 GB 磁盘

1. **Docker Build Cache (主要消耗)**: 每次 `docker compose build backend` 产生 2-3 GB cache (Go CGO 编译 tree-sitter 需 gcc + musl-dev + linux-headers)。**必须在每次 build 前执行 `docker builder prune -f`**，否则 3-4 次构建即吃满磁盘。
2. **Docker 镜像**: 6 个镜像共 5 GB, 其中 umami 1.28 GB (第三方, 无法缩减), PG 409 MB, backend 529 MB (含 Go runtime + tree-sitter .so)
3. **PostgreSQL 数据增长**: `md_bars` 表随行情持续增长 (8 周期 x 多品种 x 每秒 tick), 预估每月 ~200-500 MB
4. **NATS JetStream**: tick/bar 持久化数据, 取决于保留策略
5. **日志**: 每容器 50MB x 3 文件 = 150 MB/容器, 6 容器共 ~900 MB
6. **预留空间**: 系统更新, 临时文件, PG VACUUM, 备份文件

### 为什么需要 8 GB 内存

- 容器运行时实际仅用 ~186 MB, 上限合计 1.3 GB
- **Go 编译**: `docker compose build backend` 时 builder 容器需要 ~1-2 GB (CGO 编译 tree-sitter)
- **Node 构建**: `vite build` 需要 ~500 MB-1 GB
- **PostgreSQL**: 查询缓存、连接池 (MaxConns=25)、VACUUM 操作
- **操作系统**: 内核 + 文件系统缓存 ~1 GB
- **安全余量**: 防止 OOM kill 导致服务中断

### 资源优化建议

- **已优化**: ADR-0012 移除 ClickHouse, tick 不再持久化, 减少 ~1 GB 内存 + 大量磁盘
- **可优化**: umami 占 1.28 GB 镜像 + 53 MB 内存, 如不需要网站分析可移除
- **可优化**: Docker builder cache 自动清理: 设置 `docker builder prune --filter "until=24h"` cron
- **可优化**: PG `md_bars` 表分区 + 定期归档老数据