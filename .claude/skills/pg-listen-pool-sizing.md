---
description: Size and troubleshoot PostgreSQL connection pool with push-first LISTEN/NOTIFY
---

# PG Connection Pool & LISTEN/NOTIFY

## Default pool is too small

Original `pgxpool.New(ctx, dsn)` with no `MaxConns` defaulted to `max(4, NumCPU)`. On a 4-core host this is 4 connections.

Push-first refactor added 4 permanent PG LISTEN holders, each `pool.Acquire()` and never release:

- `normalizer_invalidator.go:40`
- `wiring.go:124` (backfiller)
- `strategy_experiment_worker.go:46`
- `backtest_worker.go:29`

With default pool size = 4, all connections are consumed at startup, so even `Login` and `/healthz` `pool.Ping` block forever on `Acquire()`.

## Fix

Set `DB_MAX_CONNS` (default 25) and configure pool:

```go
poolCfg, err := pgxpool.ParseConfig(dsn)
if err != nil { /* ... */ }
poolCfg.MaxConns = int32(cfg.DBMaxConns)
pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
```

## Per-stream LISTEN cost

Each SSE stream currently calls `pgListen.Listen`, acquiring another pool connection:

- `strategy_schedules.go:183`
- `python_strategy_backtest_crud.go:183`
- `strategy_experiment_handler.go:233`

At scale this exhausts the pool even with 25 connections.

## Recommended architecture

- Use a **dedicated LISTEN pool** for permanent listeners, separate from request pool.
- Or use **one shared listener per channel** and fan-out to subscribers in-process.

## Symptoms of exhaustion

- 524/504 on API calls
- `Login` or `/healthz` timeout
- Container marked unhealthy
- `/readyz` still works (no DB pool)
- `pool.Ping()` blocking with no timeout

## Diagnostics

```bash
curl -s http://localhost:8080/metrics | grep -E 'pgxpool_pool_total_conns|pgxpool_pool_acquired_conns'
```

See full runbook: `docs/runbook/pg-pool-exhausted.md`
