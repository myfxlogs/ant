# 31 · Risk Gate Protocol

> **关联 ADR**: ADR-0020 (EA 完全替代)
> **关联 proto**: `proto/ant/v1/risk_gate.proto`
> **关联 spec**: `docs/spec/23-risk-management.md` (existing risk engine, pre-dates risk gate)
> **状态**: Frozen (Phase 0 contract)

## 1. Purpose

The risk gate is the **single, non-bypassable money-safety boundary** that intercepts every `OrderIntent` — from both SimBroker and LiveBroker — before it reaches any broker. It decouples money safety from strategy correctness: even if a strategy goes rogue, bad orders never reach the market.

This document defines the protocol (proto messages) and the judgment rules with their default thresholds. The Go implementation lives in `backend/internal/risk/gate.go` (T3.2).

## 2. Architecture

```
Strategy SDK (Go)
       │
       ▼  broker.order_send(request)
  ┌─────────────┐
  │  Broker impl │── SimBroker  (backtest/paper)
  │  (Go)        │── LiveBroker (real)
  └──────┬──────┘
         │  Build OrderIntent
         ▼
  ┌──────────────────────┐
  │  Risk Gate (Go)       │  ← single choke point
  │  Evaluate all rules   │
  │  Return RiskDecision  │
  └──────┬───────────────┘
         │
    ┌────┴────┐
    │ ALLOW   │ DENY (or adjust)
    ▼         ▼
  broker ──► audit log + metric + reject signal
```

**Invariant**: There is NO code path that places an order without passing through the risk gate. The gate is called synchronously; a `RiskDecision{allow: false}` aborts the order immediately.

## 3. Proto Messages

### 3.1 OrderIntent

| Field | Type | Description |
|-------|------|-------------|
| `user_id` | string | Owning user |
| `account_id` | string | Target trading account |
| `symbol` | string | Broker symbol (canonical, no suffix stripping) |
| `side` | string | `"buy"` or `"sell"` |
| `volume` | string | Decimal string, e.g. `"0.10"` |
| `type` | string | `"market"`, `"limit"`, `"stop"`, `"stop_limit"` |
| `price` | string | Decimal string; `"0"` for market orders |
| `sl` | string | Stop-loss price, decimal string; `"0"` = none |
| `tp` | string | Take-profit price, decimal string; `"0"` = none |
| `magic` | int64 | EA magic number for position identification |
| `source` | enum | `ORDER_INTENT_SOURCE_SIM` or `ORDER_INTENT_SOURCE_LIVE` |
| `comment` | string | User-defined order comment |
| `created_at_unix_ms` | int64 | Intent creation timestamp (UTC, ms) |

All monetary/quantity fields (`volume`, `price`, `sl`, `tp`) use **decimal strings** to comply with the project-wide ban on `float64` for prices (CLAUDE.md).

### 3.2 RiskDecision

| Field | Type | Description |
|-------|------|-------------|
| `allow` | bool | `true` = pass; `false` = block |
| `reason` | string | Human-readable explanation (for audit log and user feedback) |
| `adjusted_volume` | string? | Optional gate-proposed volume when blocking due to size |
| `rule_hit` | string | Name of the rule that triggered the block (empty if allowed) |

### 3.3 OrderIntentResult

Wraps `OrderIntent` + `RiskDecision` + `evaluated_at_unix_ms` for full audit trailing. Every intent-decision pair is persisted.

## 4. Risk Rules (11 Rules)

Rules are evaluated in order. The **first BLOCK stops the pipeline** — subsequent rules are skipped.

### R1 · Max Lot Size

| Property | Value |
|----------|-------|
| **Input** | `volume`, account config `max_lot_size` |
| **Logic** | `volume > max_lot_size` → BLOCK |
| **Default** | `max_lot_size = 10.0` (configurable per account) |

### R2 · Max Position Count

| Property | Value |
|----------|-------|
| **Input** | current open positions count, account config `max_positions` |
| **Logic** | `open_positions + 1 > max_positions` → BLOCK |
| **Default** | `max_positions = 20` (configurable per account) |

### R3 · Max Exposure

| Property | Value |
|----------|-------|
| **Input** | current total exposure (sum of position notional values), new order notional, account config `max_exposure` |
| **Logic** | `current_exposure + new_notional > max_exposure` → BLOCK |
| **Default** | `max_exposure = balance × leverage / 2` (50% buying power; configurable) |

### R4 · Daily Loss / Drawdown Circuit Breaker

| Property | Value |
|----------|-------|
| **Input** | `daily_realized_pnl`, `current_equity`, `peak_equity` |
| **Logic (daily loss)** | `daily_realized_pnl < -max_daily_loss` → BLOCK |
| **Logic (drawdown)** | `(peak_equity - current_equity) / peak_equity > max_drawdown_pct` → BLOCK |
| **Default** | `max_daily_loss = 0` (disabled), `max_drawdown_pct = 0.30` (30%; configurable) |

### R5 · Symbol Whitelist

| Property | Value |
|----------|-------|
| **Input** | `symbol`, account config `symbol_whitelist` |
| **Logic** | `symbol_whitelist` is non-empty AND `symbol` not in whitelist → BLOCK |
| **Default** | empty list = all symbols allowed |

### R6 · Leverage Cap

| Property | Value |
|----------|-------|
| **Input** | `symbol` leverage, account config `max_leverage` |
| **Logic** | `symbol_leverage > max_leverage` → BLOCK |
| **Default** | `max_leverage = 500` (configurable) |

### R7 · Order Frequency Limit

| Property | Value |
|----------|-------|
| **Input** | order count in sliding window (per account) |
| **Logic** | `order_count_in_window >= max_orders_per_window` → BLOCK |
| **Default** | `max_orders_per_window = 60`, `window = 60s` (1 order/sec avg; configurable) |

### R8 · Duplicate Order Protection

| Property | Value |
|----------|-------|
| **Input** | `symbol`, `side`, `volume`, `type`, `price` of last N orders |
| **Logic** | an identical intent was submitted within `dedup_window_ms` → BLOCK |
| **Default** | `dedup_window_ms = 5000` (5s; configurable) |

### R9 · Margin Pre-Check

| Property | Value |
|----------|-------|
| **Input** | `required_margin` (volume × price × contract_size / leverage), `free_margin` |
| **Logic** | `(used_margin + required_margin) / equity > max_margin_ratio` → BLOCK |
| **Default** | `max_margin_ratio = 0.80` (80%; configurable) |

### R10 · Global Kill-Switch

| Property | Value |
|----------|-------|
| **Input** | global kill-switch state (admin toggle) |
| **Logic** | kill-switch is ON → BLOCK **all** live orders (sim orders unaffected) |
| **Default** | OFF (kill-switch inactive) |

### R11 · Per-User Autotrade Switch

| Property | Value |
|----------|-------|
| **Input** | `auto_trade_enabled` flag from `auto_trading_settings.proto` (per schedule/user) |
| **Logic** | `auto_trade_enabled == false` AND `source == LIVE` → BLOCK |
| **Default** | follows existing `auto_trade_enabled` setting |

> **Note**: R11 is a **coarse-grained** user-level switch (existing). The risk gate reads it as one of its rules rather than maintaining a separate flag. It acts as the final backstop: if autotrade is off, no live order leaves the gate regardless of other rule outcomes.

## 5. Rule Evaluation Order

```
R1 → R2 → R3 → R4 → R5 → R6 → R7 → R8 → R9 → R10 → R11

First BLOCK stops the pipeline.
```

The order is deliberate:
- **R1–R3** (size/count/exposure) are cheapest to evaluate and catch the most common issues.
- **R4** (loss/drawdown) is the money-safety circuit breaker.
- **R5–R6** (whitelist/leverage) are config-driven filters.
- **R7–R8** (frequency/duplicate) are behavioral guards.
- **R9** (margin) is the most expensive check (requires account state).
- **R10–R11** (kill-switch/autotrade) are the final coarse-grained gates.

## 6. Audit Trail

Every `OrderIntentResult` is persisted with:
- Full `OrderIntent` (all fields)
- Full `RiskDecision` (allow/reason/rule_hit)
- `evaluated_at_unix_ms` timestamp

This enables per-strategy-run trace replay and drift analysis (T4.1).

## 7. Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `risk_gate_intent_total{source,decision}` | Counter | Intents evaluated by source (sim/live) and outcome (allow/deny) |
| `risk_gate_deny_total{rule}` | Counter | Denials broken down by rule name |
| `risk_gate_latency_ms` | Histogram | Gate evaluation latency |

## 8. Integration Points

| Point | Description |
|-------|-------------|
| **SimBroker** | Calls gate before every simulated fill; source = SIM |
| **LiveBroker** | Calls gate before every MT gateway RPC (`OrderSend`, `PositionClose`, etc.); source = LIVE |
| **OMS / Signal Router** | Existing signal path — will be retired per D7; risk gate replaces its pre-check role |
| **Admin API** | Kill-switch and rule thresholds are configurable via admin endpoints (T3.2) |

## 9. Verification

```bash
# Proto compiles
buf lint proto/ant/v1/risk_gate.proto && buf generate proto/ant/v1/risk_gate.proto

# Generated Go types import correctly
go build ./backend/gen/proto/ant/v1/...

# Spec completeness: all 11 rules have input + default threshold documented
grep -c "^### R" docs/spec/31-risk-gate.md  # must be 11
```
