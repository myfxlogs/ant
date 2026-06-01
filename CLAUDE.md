# Project "ant" — Mandatory Constraints

These constraints are enforced at implementation time. Violation = fix before commit.

## File & Function Size (NON-NEGOTIABLE)

| Language | Max File | Max Function |
|----------|----------|--------------|
| Go       | 300 行   | 50 行        |
| TypeScript | 250 行 | 50 行        |

- **Before adding code to a near-limit file: SPLIT THE FILE FIRST.**
- Generated code (`gen/`) and tests exempt (50% overage allowed).
- Check: `make check-size`

## Prohibited (Zero Tolerance)

- ❌ REST endpoints (except healthz/readyz/livez/metrics)
- ❌ WebSocket
- ❌ float64 in price calculations (use `decimal.Decimal` in Go)
- ❌ Cross-scope changes (one task = one scope)
- ❌ Hardcoded secrets / `.env` in repo
- ❌ `//nolint`, `# noqa`, `// @ts-ignore`

## Platform Protocol

- External API: **ConnectRPC + SSE ONLY**
- Internal: in-process function calls OR NATS JetStream
- MT access: mtapi gRPC ONLY (via `adapter/mt4/` and `adapter/mt5/`)
- MT4 and MT5 adapters MUST NOT share code (except `adapter/mdtick/` shared DTO)

## Push-First Architecture

- **gRPC streaming + SSE is the default.** Prefer server-push over client-pull in every scenario.
- ❌ Polling / cron / `setInterval` / `time.Ticker` — ONLY when the data source has no push capability AND the data is not latency-sensitive
- ❌ Never poll when a streaming equivalent exists (e.g. MT5 `OnQuote` stream over polling `GetQuote`, SSE `bar_update` over polling `PriceHistory`)
- ✅ If adding a new data feed, ask first: "Can this be a stream?" If yes, make it a stream

## Data Precision

- Prices: `NUMERIC(20,8)` PG / `Decimal(18,6)` CH / `decimal.Decimal` Go
- Time: UTC, millisecond precision (`int64 ts_unix_ms`)
- Symbol: raw broker symbol = canonical (no suffix stripping)

## Before Commit

```bash
make check-size   # file/function size compliance
go build ./...    # must pass
```

Full constraint details: see `/root/.claude/projects/-opt-ant/memory/`
