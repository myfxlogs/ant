# Live Strategy User-Facing Roadmap

## Background

Engine layer is complete: WASM strategy compilation + execution (Go compiled path + MQL interpreter path), real-time data streaming (bar/tick/trade), signal dispatch to broker/paper, session lifecycle management, risk gate integration.

This roadmap covers the remaining work to make live strategies usable by end users.

## Phases

### Phase 1: Signal Persistence (Foundation)

**Goal**: Strategy signals are written to DB, queryable via API.

**Why first**: Without persisted signals, monitoring API has no data, frontend has nothing to display, users cannot review what their strategies did.

**Tasks**:
- [ ] Design `strategy_signals` table schema (signal_id, strategy_run_id, account_id, symbol, signal_type, volume, price, sl, tp, ticket, pnl, created_at)
- [ ] Design `strategy_runs` table schema (run_id, user_id, account_id, strategy_code, symbol, timeframe, mode, status, started_at, stopped_at, error, total_signals)
- [ ] Create repository layer for both tables
- [ ] Hook `dispatchLiveSignal` to persist every signal before routing to broker
- [ ] Hook `RunLiveStrategy` start/stop to create/update `strategy_runs` records
- [ ] Add ConnectRPC: `ListStrategySignals`, `ListStrategyRuns`, `GetStrategyRun`
- [ ] Unit tests for persistence + retrieval

**Files to create/modify**:
- `internal/repository/strategy_signal_repo.go` (new)
- `internal/repository/strategy_run_repo.go` (new)
- `internal/connect/strategy/live_dispatch.go` (add persistence hook)
- `internal/connect/strategy/live_runner.go` (add run start/stop hooks)
- `internal/connect/strategy/strategy_execution_handler.go` (add query RPCs)
- `cmd/server/migrate/` (new migration SQL)

### Phase 2: Strategy Status Monitoring API

**Goal**: Query running strategies, health status, real-time PnL, error info.

**Why second**: Depends on Phase 1 for signal history. Provides the data layer for frontend.

**Tasks**:
- [ ] In-memory session registry: track active `LiveSession` instances with metadata (run_id, account, symbol, started_at, last_signal_at, error_count)
- [ ] Add ConnectRPC: `ListActiveStrategies`, `GetActiveStrategy`, `StopStrategy`
- [ ] Wire `RunLiveStrategy` to register/deregister in session registry
- [ ] Expose session health: WASM alive, last bar processed, stderr tail
- [ ] SSE stream: `WatchStrategySignals` (real-time signal push to frontend)
- [ ] Integration tests for registry lifecycle

**Files to create/modify**:
- `internal/connect/strategy/session_registry.go` (new)
- `internal/connect/strategy/live_runner.go` (register/deregister hooks)
- `internal/connect/strategy/strategy_execution_handler.go` (add monitoring RPCs)

### Phase 3: Frontend Strategy Management Page

**Goal**: Users can manage and monitor strategies from the UI.

**Why third**: Depends on Phase 1 + 2 APIs being ready.

**Tasks**:
- [ ] Strategy list page: show all user strategies with status badges (running/stopped/error)
- [ ] Strategy detail page: code editor, parameters, start/stop controls
- [ ] Signal log panel: real-time SSE stream of signals (from `WatchStrategySignals`)
- [ ] Run history tab: past runs with PnL, signal count, duration
- [ ] Start dialog: select account, symbol, timeframe, mode (paper/live), parameters
- [ ] Error display: stderr tail, last error, retry button

**Files to create/modify**:
- Frontend project (React + Ant Design + ConnectRPC client)

### Phase 4: Live API Independence (Low Priority)

**Goal**: Separate live strategy control from paper trading service.

**Why last**: Functional gap is minimal — paper handler already supports live mode via `cfg.Mode`. This is a naming/organization cleanup.

**Tasks**:
- [ ] Define `StrategyControlService` proto: `StartStrategy`, `StopStrategy`, `ListActiveStrategies`
- [ ] Migrate `StartPaperStrategy`/`StopPaperStrategy` callers to new service
- [ ] Deprecate strategy control methods on `PaperTradingService`

## Dependency Graph

```
Phase 1 (Signal Persistence)
    ↓
Phase 2 (Monitoring API)
    ↓
Phase 3 (Frontend)

Phase 4 (API Independence) — can be done anytime, low priority
```

## Current Status

- **Phase 1**: ✅ Implemented — migration, repository, dispatch hooks, RPCs
- **Phase 2**: Not started
- **Phase 3**: Not started
- **Phase 4**: Not started
