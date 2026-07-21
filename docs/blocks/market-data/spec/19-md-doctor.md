# 19 · md-doctor 数据基础对账工具规范

> 路径：`backend/cmd/md-doctor/`
> 目标 LOC：≤ 400
> 关联 ADR：ADR-0008（去重对齐前提）、ADR-0009（finality + replay）

## 1. 目标

提供**单条命令**回答四个生产问题：

1. **NATS 与 CH 是否对账一致？**（端到端正确性）
2. **bar 是否连续？**（数据完整性）
3. **canonical 是否漂移？**（订阅活性）
4. **DLQ 中是否有需要排查的 parse_error 样本？**（数据健康）

## 2. CLI 设计

```
md-doctor [global flags] <command> [command flags]

Global Flags:
  --window duration    检查窗口（default 1h）
  --output text|json   输出格式（default text）
  --strict             任一不通过 → exit 1

Commands:
  reconcile            # NATS publish count vs CH row count（最近 window）
  bar-continuity       # md_bars 时间序列缺口扫描
  canonical-liveness   # broker_symbols 中 canonical 是否最近活跃
  dlq-tail             # md_ticks_dlq 最近 N 条 parse_error
  all                  # 跑全部并汇总
```

## 3. 子命令详细规约

### 3.1 `reconcile`

**目的**：发现 H-1（CH dedup 与应用层不一致）类问题。

```bash
md-doctor reconcile --window 10m
```

逻辑：
- 取 NATS `MD_EVENTS` stream 在 [now-window, now] 内 `md.tick.>` 消息计数（`nats stream info MD_EVENTS --json`）
- 取 CH `md_ticks` 同窗口（`arrived_unix_ms`）行数
- 计算差异比 = |nats - ch| / max(nats, ch)
- 输出表格 by (broker, canonical)：nats_count / ch_count / diff_pct

通过条件：差异比 < 0.1%。

### 3.2 `bar-continuity`

**目的**：发现 1m bar 时间缺口（broker 离线 / aggregator bug）。

```bash
md-doctor bar-continuity --window 24h --period 1m
```

逻辑：
- 对每 (broker, canonical, period='1m')：CH 查 `arrayJoin(arrayDistinct(...))` 得到所有 close_ts
- 计算相邻 close_ts 差值，> period * 1.5 即记一个 gap
- 排除已知节假日窗口（周末 = Sat 22:00 ~ Sun 22:00 UTC）

输出：
```
broker          canonical    gaps  total_gap_minutes  worst_gap
ICMarkets       EURUSD       3     12                 5min @ 2026-05-23T14:23:45Z
```

通过条件：所有 (broker, canonical) gap 总分钟数 < window 的 1%。

### 3.3 `canonical-liveness`

**目的**：发现订阅死亡（broker 推送停止但未告警）。

```bash
md-doctor canonical-liveness --window 1h
```

逻辑：
- PG `broker_symbols` 列出所有 canonical
- 检查 CH `md_ticks` 最近 window 内每个 (broker, canonical) 是否有 tick
- 排除：周末 + 已知低流动性品种（PG `broker_symbols.trade_mode IN (0,3)`）

输出："已配置但 N 小时无 tick"列表。

### 3.4 `dlq-tail`

```bash
md-doctor dlq-tail --reason parse_error --limit 50
```

直接 `SELECT * FROM md_ticks_dlq WHERE reason=? ORDER BY arrived_unix_ms DESC LIMIT ?`，pretty-print。便于工程师定位损坏数据样本。

### 3.5 `all`

按顺序跑前四个，输出 markdown 摘要。`--strict` 时任一不过 → exit 1。

可作为 cron / Kubernetes Job 每日运行，输出存档 `docs/handover/md-doctor-YYYYMMDD.md`。

## 4. 实现约束

```
backend/cmd/md-doctor/
├── main.go              ≤80   lines  cobra root + 命令注册
├── reconcile.go         ≤100  lines
├── bar_continuity.go    ≤100  lines
├── canonical_liveness.go ≤80  lines
├── dlq_tail.go          ≤40   lines
└── *_test.go
```

依赖：
- `github.com/spf13/cobra` 已在仓库
- `github.com/nats-io/nats.go` JetStream API
- `clickhouse.Conn` / `pgx.Pool` 复用 `internal/storage/`

## 5. 验收命令

```bash
# 1. 编译
cd backend && go build -o /tmp/md-doctor ./cmd/md-doctor/

# 2. 子命令存在
/tmp/md-doctor --help | grep -E 'reconcile|bar-continuity|canonical-liveness|dlq-tail|all' | wc -l | grep -q '^5$'

# 3. reconcile 在新鲜数据上 PASS
/tmp/md-doctor reconcile --window 10m --strict

# 4. all + json 输出可被 jq 解析
/tmp/md-doctor all --window 1h --output json | jq -e '.reconcile.passed and .bar_continuity.passed'

# 5. 单测
go test -cover ./cmd/md-doctor/...
```

## 6. 集成到 M10 关闭检查

`docs/runbook/m10-closure.md` 强制要求关闭前：
```bash
md-doctor all --window 24h --strict --output json > docs/handover/m10-doctor.json
```
全 PASS 是 M10.Z-1 卡片验收的硬条件。
