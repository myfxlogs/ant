# 18 · 历史回填器（Backfiller）规范

> 路径：`backend/internal/mdgateway/backfiller/`
> 目标 LOC：≤ 350（非测试）
> 关联 ADR：ADR-0009
> 上游：mtapi `GetPriceHistory` RPC；下游：`bar_aggregator` finality → CH `md_bars` + NATS `md.bar.>`

## 1. 目标

补齐三类数据缺口：
1. **新订阅 symbol**：账户首次订阅某 canonical → 回填 30 天 1m + 90 天 1h + 365 天 1d
2. **broker 离线窗口**：mdgateway 启动时检测 CH `md_bars` 与 now 的 gap，补齐
3. **新接入账户**：账户 `is_active` 从 false → true → 回填该账户全部已订阅 canonical

## 2. 文件清单

```
backend/internal/mdgateway/backfiller/
├── backfiller.go        ≤200 lines  调度 + 限速
├── source_mtapi.go      ≤80  lines  GetPriceHistory 适配
├── target.go            ≤80  lines  写入 bar_aggregator（带 IsReplay=true）
├── metrics.go           ≤40  lines
└── *_test.go
```

## 3. 接口

```go
package backfiller

type Source interface {
    // FetchBars 取 (broker, canonical, period) 在 [from, to] 区间的历史 bar
    // 返回按 close_ts_unix_ms 升序
    FetchBars(ctx context.Context, req FetchReq) ([]*mdtick.Bar, error)
}

type FetchReq struct {
    AccountID string
    Broker    string
    Canonical string
    SymbolRaw string         // 用于 mtapi 调用
    Period    string         // "1m" | "1h" | "1d"
    From, To  int64          // unix ms
    Limit     int            // 默认 5000
}

type Target interface {
    // IngestBar 走 bar_aggregator finality 检查（ADR-0009 §2.2）
    IngestBar(ctx context.Context, bar *mdtick.Bar) error
}

type Backfiller struct {
    src     Source
    tgt     Target
    pg      *pgxpool.Pool
    ch      clickhouse.Conn
    log     *zap.Logger
    limiter *rate.Limiter  // 6 req/min/account
    metrics *Metrics
}

// Run 启动一次完整扫描；ctx.Done 时退出。
// 调用频率：runner 启动时 1 次 + 每 6h 1 次（cron-like）
func (b *Backfiller) Run(ctx context.Context) error

// BackfillAccount 处理单账户全部 canonical（新接入触发）
func (b *Backfiller) BackfillAccount(ctx context.Context, accountID string) error

// BackfillSymbol 处理单 (account, canonical) 全周期（新订阅触发）
func (b *Backfiller) BackfillSymbol(ctx context.Context, accountID, canonical string) error
```

## 4. 算法

```go
func (b *Backfiller) Run(ctx context.Context) error {
    accounts := b.loadActiveAccounts(ctx)
    for _, acc := range accounts {
        for _, canon := range acc.Symbols {
            for _, period := range []string{"1m", "1h", "1d"} {
                from := b.queryMaxCloseTs(acc.Broker, canon, period)
                if from == 0 {
                    from = b.defaultFrom(period)  // 30d / 90d / 365d ago
                }
                to := time.Now().UnixMilli()
                if to-from < periodMs(period)*2 {
                    continue  // gap < 2 个周期，不值得调用
                }
                b.backfillRange(ctx, acc, canon, period, from, to)
            }
        }
    }
    return nil
}

func (b *Backfiller) backfillRange(ctx, acc, canon, period, from, to) {
    for from < to {
        b.limiter.Wait(ctx)  // 限速
        bars, err := b.src.FetchBars(ctx, FetchReq{...})
        if err != nil { /* metric + retry exponential */ continue }
        for _, bar := range bars {
            bar.IsReplay = true
            b.tgt.IngestBar(ctx, bar)  // finality 检查在内
        }
        if len(bars) == 0 { break }
        from = bars[len(bars)-1].CloseTsUnixMs + 1
    }
}
```

## 5. 默认回填窗口

| Period | 默认 lookback | 理由 |
|---|---|---|
| 1m | 30 天 | 短期策略需要分钟级；> 30d mtapi 通常无数据 |
| 1h | 90 天 | 中期策略 |
| 1d | 365 天 | 长期回测 |
| 5m / 15m / 4h | 不回填 | 由 1m 数据离线计算（factorsvc 派生）|

## 6. 限速

`golang.org/x/time/rate.NewLimiter(rate.Every(10*time.Second), 1)` per account：
- 6 次/分钟/账户（mtapi 软上限）
- 全局总限速 60 req/s（保守）

## 7. metrics

```
md_backfill_started_total{account_id, broker, canonical, period}      Counter
md_backfill_bars_ingested_total{broker, canonical, period}            Counter
md_backfill_bars_skipped_finalized_total                              Counter  (与 bar_aggregator 的 metric 一致)
md_backfill_errors_total{kind}                                        Counter  kind ∈ {fetch, ingest, ratelimit}
md_backfill_duration_seconds{period}                                  Histogram
md_backfill_lag_seconds                                               Gauge   max(now - max_close_ts) 跨所有 (broker,canonical,period)
```

## 8. 与 bar_aggregator 的协作（finality）

- backfiller `IngestBar` 内部调用 `bar_aggregator.IngestExternalBar(bar, IsReplay=true)`
- aggregator 检查 `finalizedBars[key]`：若 `bar.close_ts <= finalized` → 丢弃 + `md_bar_skipped_finalized_total++`
- 否则：写 CH（INSERT 走 `md_bars_buffer`）+ NATS PublishBar（带 X-Ant-Replay header）+ 更新 `finalizedBars[key] = bar.close_ts`

## 9. 触发时机

| 触发 | 调用 | 频率 |
|---|---|---|
| mdgateway 启动 | `Run(ctx)` | 1 次 |
| 周期 cron | `Run(ctx)` | 6h |
| 账户 `is_active` 0→1 | `BackfillAccount` | 即时 |
| `canonical_subscribed_symbols` 新增 | `BackfillSymbol` | 即时（PG NOTIFY） |

后两种由 `runner.go` 监听 `mt_accounts`/`broker_symbols` 变更触发（PG `LISTEN` 同 ADR-0011 §2.3 机制）。

## 10. 验收命令

```bash
# 1. 编译 + 测试
( cd backend && go build ./internal/mdgateway/backfiller/... \
                && go test -race -cover ./internal/mdgateway/backfiller/... )

# 2. LOC
LOC=$(find backend/internal/mdgateway/backfiller -name "*.go" -not -name "*_test.go" \
       | xargs wc -l | tail -1 | awk '{print $1}')
test "$LOC" -le 350

# 3. 端到端：制造 1h gap → backfiller 补齐
go test -tags=integration ./internal/mdgateway/backfiller/ -run TestBackfillGap -timeout 5m
# 测试逻辑：
#   - 启动 mock mtapi 提供 1h 历史 bar
#   - CH 故意空 md_bars
#   - Run() 后断言 CH md_bars 行数 == 60（1m × 60）
#   - 同 close_ts 再次 Run() → 0 行新增（finality）

# 4. 限速生效
go test -tags=integration ./internal/mdgateway/backfiller/ -run TestRateLimit -v
# 测试：100 账户并发 → 60s 内总调用 ≤ 600

# 5. metric 暴露
curl -s localhost:8080/metrics | grep -c '^md_backfill_' | grep -qE '^[5-9]$|^[1-9][0-9]$'
```
