# 施工派工单 — POST-2 容量探针批（性能/容量基线·安全形态）

**日期**: 2026-09-19 · **设计方**: Devin CLI · **施工**: 施工方（ANT_ROLE=builder）
**立项背景**: POST-2「性能/容量压测（下单/回测/SSE）」自 2026-08-09 上线审计登记以来悬空。

## 设计实查结论（决策层）

**本机即生产**（docker compose 全栈在跑：backend/frontend/postgres/redis/nats/prometheus——实盘账户连着真经纪商）。**对生产压测不安全**：下单轴真路径走 mtapi→真经纪商（打不下手）；即便回测 CPU 密集负载也与实盘 dispatch 抢同一宿主。且单机压测数字本就不代表真实部署容量。

**但代码实查已给出容量模型大半答案**（free）：

| 轴 | 界 | 证据 |
|---|---|---|
| 回测并发 | **有界 3 worker** | `backtest_worker.go:23` `defaultWorkers=3` + `ClaimNextForWork` SKIP LOCKED 原子认领 + 60s lease + LISTEN/NOTIFY 唤醒（push-first） |
| 回测配额 | 有界 | `subscription.go:18` `MaxBacktestsDaily` per-user 日配额 |
| 登录 | 有界 | `main.go:101` `RateLimitLoginPerMinute` |
| 实盘策略数 | 有界 | `MaxLiveStrategies` 配额 |
| **SSE 扇出** | **无界** | 未见 per-user/per-global 连接上限——N tabs×M users 可堆积 stream goroutine+PG/NATS 订阅 |
| **下单** | 不可测于本机 | 真路径=mtapi→经纪商；paper engine（internal/paper）可测进程内路径 |

## 施工范围（探针批——in-process，零触生产容器）

### S1 回测吞吐探针（进程内，无 DB）

- `backend/tools/mql2go/vm_bench_test.go` 已有 VM 微基准。**新增并发退化探针**：`BenchmarkVMExec_Concurrency{N=1,4,8,16}`——N goroutine 并行跑同一编译产物（各自独立 VM 实例），测 per-run wall-time 与吞吐曲线，量化"3 worker 满占时第 4 个请求排队代价"+CPU 饱和拐点。
- 如实标注：测的是 VM 纯执行（不含 claim/lease/PG 往返——那部分是 worker 有界常数，非瓶颈轴）。

### S2 SSE 扇出探针（进程内）

- 新测试文件用 `httptest.NewServer` 起真实 handler（复用一个轻量 stream handler——`session_registry.go` 的 `Watch()` chan 扇出模式是 SSE 变体可参考）或最小 SSE handler stub：N∈{10,100,500} 并发 client 保持流，测每 client 内存增量/goroutine 数/事件延迟 p99。**产出扇出成本曲线**——回答"无界 SSE 多少连接开始退化"。
- 若 handler 依赖重不可进程内起 → 如实降级为"每 client goroutine+chan 成本"微基准并注明口径差异。

### S3 下单路径探针（paper engine 进程内）

- `internal/paper/engine.go`（10K，内存 paper broker）：benchmark 并发下单/撮合延迟分布。**如实注明**：测的是 paper 路径延迟基线，真经纪商路径（mtapi RTT）不在本探针范围——记入"待 staging"清单。

### S4 交付物：`docs/benchmarks/post2-capacity-baseline-2026-09.md`

- 三轴实测数字 + 容量模型表（有界/无界枚举上表扩写）+ **发现的硬缺口**（如 SSE 无界→登记新债）+ "待 staging"残余清单（mtapi 真路径、claim/DB 吞吐、多容器水平扩展）。

## 边界（不做）

- **禁对运行中容器/生产端口发任何负载**（docker ps 可见 alphaforge-* 在跑）。
- 不测 mtapi/真经纪商路径。
- 不改生产配置、不加 cap（发现缺口→登记，不在本批修）。
- 探针测试不跑 DB（本机 127.0.0.1:5432 不通——PG 在容器内网）。
- 不部署、不 push。

## 门禁

```bash
cd backend && go build ./... && go vet ./... && go test ./internal/paper/... ./tools/mql2go/... -count=1 && go test -bench=. -benchtime=1s -run='^$' ./tools/mql2go/ ./internal/paper/ 2>&1 | tail -20
cd backend && go run ./tools/check-file-lines --strict && git diff --check
```

- 探针测试用 `testing.B` 或普通 test+度量输出；数字写进 S4 文档。
- SSE 探针若降级须如实注明口径。
- 串行，完成报证据等复审。
