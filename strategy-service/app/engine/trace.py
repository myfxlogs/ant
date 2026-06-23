"""Unified strategy execution trace (T4.1).

Captures the full event chain for observability and debugging:
  tick → bar → strategy callback → order intent → risk gate → broker receipt

Supports:
  - Per-run trace replay (trace replay viewer)
  - Backtest vs live drift comparison
  - Translation coverage + behavior diff reporting

Trace events are collected in-memory during a run and serialized to JSON
for persistence.  Each event carries a monotonic sequence number, timestamp,
and typed payload.
"""

from __future__ import annotations

import json
import time
from dataclasses import dataclass, field
from decimal import Decimal
from typing import Any, Dict, List, Optional


# ── Trace event types ──────────────────────────────────────────────────

@dataclass
class TraceEvent:
    """One event in the strategy execution trace."""
    seq: int
    ts_ms: int
    event: str  #: tick, bar, on_init, on_tick, on_bar, on_timer, on_trade, on_deinit,
                #  order_intent, risk_decision, fill, close, modify, cancel, reject
    payload: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> dict:
        return {
            "seq": self.seq,
            "ts_ms": self.ts_ms,
            "event": self.event,
            **self.payload,
        }


@dataclass
class StrategyTrace:
    """Complete trace of one strategy execution run."""
    strategy_name: str
    run_id: str
    mode: str  # "backtest", "paper", "live"
    symbol: str
    timeframe: str
    started_at_ms: int
    events: List[TraceEvent] = field(default_factory=list)
    metadata: Dict[str, Any] = field(default_factory=dict)

    _seq: int = field(default=0, init=False)

    def _next_seq(self) -> int:
        self._seq += 1
        return self._seq

    def add(self, event: str, ts_ms: Optional[int] = None, **payload) -> TraceEvent:
        """Add an event to the trace."""
        evt = TraceEvent(
            seq=self._next_seq(),
            ts_ms=ts_ms or int(time.time() * 1000),
            event=event,
            payload=payload,
        )
        self.events.append(evt)
        return evt

    # ── Convenience methods for common events ──────────────────────────

    def tick(self, ts_ms: int, bid: float, ask: float) -> TraceEvent:
        return self.add("tick", ts_ms, bid=str(bid), ask=str(ask))

    def bar_close(self, ts_ms: int, timeframe: str,
                  open_: float, high: float, low: float, close: float,
                  volume: float) -> TraceEvent:
        return self.add("bar", ts_ms, timeframe=timeframe,
                        open=str(open_), high=str(high), low=str(low),
                        close=str(close), volume=str(volume))

    def lifecycle(self, hook: str) -> TraceEvent:
        return self.add(hook)  # on_init, on_tick, on_bar, on_timer, on_trade, on_deinit

    def order_intent(self, symbol: str, side: str, order_type: str,
                     volume: str, price: str = "0",
                     sl: str = "0", tp: str = "0",
                     magic: str = "0", comment: str = "",
                     ticket: int = 0) -> TraceEvent:
        return self.add("order_intent",
                        symbol=symbol, side=side, type=order_type,
                        volume=volume, price=price, sl=sl, tp=tp,
                        magic=magic, comment=comment, ticket=ticket)

    def risk_decision(self, allowed: bool, reason: str = "",
                      rule_hit: str = "", adjusted_volume: str = "",
                      ticket: int = 0) -> TraceEvent:
        return self.add("risk_decision",
                        allowed=allowed, reason=reason,
                        rule_hit=rule_hit, adjusted_volume=adjusted_volume,
                        ticket=ticket)

    def fill(self, ticket: int, symbol: str, side: str,
             volume: str, price: str, commission: str = "0") -> TraceEvent:
        return self.add("fill",
                        ticket=ticket, symbol=symbol, side=side,
                        volume=volume, price=price, commission=commission)

    def close_position(self, ticket: int, volume: str,
                       price: str, pnl: str, reason: str = "signal") -> TraceEvent:
        return self.add("close",
                        ticket=ticket, volume=volume,
                        price=price, pnl=pnl, reason=reason)

    def modify_position(self, ticket: int, sl: str = "", tp: str = "") -> TraceEvent:
        return self.add("modify", ticket=ticket, sl=sl, tp=tp)

    def cancel_order(self, ticket: int) -> TraceEvent:
        return self.add("cancel", ticket=ticket)

    def reject(self, reason: str, ticket: int = 0) -> TraceEvent:
        return self.add("reject", reason=reason, ticket=ticket)

    # ── Serialization ──────────────────────────────────────────────────

    def to_dict(self) -> dict:
        return {
            "strategy_name": self.strategy_name,
            "run_id": self.run_id,
            "mode": self.mode,
            "symbol": self.symbol,
            "timeframe": self.timeframe,
            "started_at_ms": self.started_at_ms,
            "event_count": len(self.events),
            "metadata": self.metadata,
            "events": [e.to_dict() for e in self.events],
        }

    def to_json(self, indent: int = 2) -> str:
        return json.dumps(self.to_dict(), indent=indent, default=str)

    @classmethod
    def from_dict(cls, data: dict) -> "StrategyTrace":
        trace = cls(
            strategy_name=data.get("strategy_name", ""),
            run_id=data.get("run_id", ""),
            mode=data.get("mode", ""),
            symbol=data.get("symbol", ""),
            timeframe=data.get("timeframe", ""),
            started_at_ms=data.get("started_at_ms", 0),
            metadata=data.get("metadata", {}),
        )
        trace._seq = data.get("event_count", 0)
        for e in data.get("events", []):
            trace.events.append(TraceEvent(
                seq=e["seq"],
                ts_ms=e.get("ts_ms", 0),
                event=e["event"],
                payload={k: v for k, v in e.items() if k not in ("seq", "ts_ms", "event")},
            ))
        return trace

    @classmethod
    def from_json(cls, json_str: str) -> "StrategyTrace":
        return cls.from_dict(json.loads(json_str))


# ── Trace replay ───────────────────────────────────────────────────────


def replay_trace(trace: StrategyTrace) -> List[str]:
    """Replay a trace as human-readable log lines."""
    lines: List[str] = []
    lines.append(f"Trace Replay: {trace.strategy_name} ({trace.mode})")
    lines.append(f"  Run: {trace.run_id} | {trace.symbol} {trace.timeframe}")
    lines.append(f"  Events: {len(trace.events)}")
    lines.append("-" * 60)

    for evt in trace.events:
        ts = evt.ts_ms
        etype = evt.event
        p = evt.payload

        if etype == "tick":
            lines.append(f"  [{ts}] TICK bid={p.get('bid')} ask={p.get('ask')}")
        elif etype == "bar":
            lines.append(f"  [{ts}] BAR {p.get('timeframe')} O={p.get('open')} H={p.get('high')} L={p.get('low')} C={p.get('close')}")
        elif etype.startswith("on_"):
            lines.append(f"  [{ts}] LIFECYCLE {etype}")
        elif etype == "order_intent":
            lines.append(f"  [{ts}] INTENT {p.get('side')} {p.get('type')} {p.get('volume')} lots @ {p.get('price')} | magic={p.get('magic')} ticket={p.get('ticket')}")
        elif etype == "risk_decision":
            decision = "ALLOW" if p.get("allowed") else f"DENY({p.get('rule_hit')})"
            lines.append(f"  [{ts}] RISK {decision} | {p.get('reason', '')}")
        elif etype == "fill":
            lines.append(f"  [{ts}] FILL ticket={p.get('ticket')} {p.get('side')} {p.get('volume')} @ {p.get('price')} commission={p.get('commission')}")
        elif etype == "close":
            lines.append(f"  [{ts}] CLOSE ticket={p.get('ticket')} vol={p.get('volume')} @ {p.get('price')} PnL={p.get('pnl')}")
        elif etype == "modify":
            lines.append(f"  [{ts}] MODIFY ticket={p.get('ticket')} SL={p.get('sl')} TP={p.get('tp')}")
        elif etype == "cancel":
            lines.append(f"  [{ts}] CANCEL ticket={p.get('ticket')}")
        elif etype == "reject":
            lines.append(f"  [{ts}] REJECT {p.get('reason')}")
        else:
            lines.append(f"  [{ts}] {etype} {p}")

    lines.append("-" * 60)
    lines.append(f"End of trace — {len(trace.events)} events")
    return lines


# ── Drift comparison ───────────────────────────────────────────────────


@dataclass
class DriftReport:
    """Comparison of two traces (e.g., backtest vs live)."""
    baseline_name: str     # e.g., "backtest"
    compare_name: str      # e.g., "live"
    total_events_baseline: int
    total_events_compare: int
    matched_events: int
    deviations: List[Dict[str, Any]] = field(default_factory=list)
    summary: str = ""

    @property
    def drift_pct(self) -> float:
        if self.total_events_baseline == 0:
            return 0.0
        return (1.0 - self.matched_events / self.total_events_baseline) * 100

    def to_dict(self) -> dict:
        return {
            "baseline": self.baseline_name,
            "compare": self.compare_name,
            "events_baseline": self.total_events_baseline,
            "events_compare": self.total_events_compare,
            "matched": self.matched_events,
            "drift_pct": round(self.drift_pct, 1),
            "deviations": self.deviations,
            "summary": self.summary,
        }

    def to_json(self) -> str:
        return json.dumps(self.to_dict(), indent=2, default=str)


def compare_traces(baseline: StrategyTrace, compare: StrategyTrace) -> DriftReport:
    """Compare two traces and produce a drift report.

    Alignment: match events by (event_type, symbol, side) in order.
    Reports count mismatches, extra/missing events, and price/volume deltas.
    """
    deviations: List[Dict[str, Any]] = []
    matched = 0

    b_events = list(baseline.events)
    c_events = list(compare.events)

    # Match by sequence index for events of the same type.
    b_idx = 0
    c_idx = 0
    while b_idx < len(b_events) and c_idx < len(c_events):
        be = b_events[b_idx]
        ce = c_events[c_idx]

        if be.event == ce.event:
            # Same event type — check payload consistency.
            matched += 1
            if be.event in ("order_intent", "fill", "close"):
                _compare_monetary_fields(be, ce, deviations)
            b_idx += 1
            c_idx += 1
        elif be.event < ce.event:
            deviations.append({
                "type": "missing_in_compare",
                "event": be.event,
                "seq_baseline": be.seq,
                "detail": f"Event '{be.event}' present in baseline but missing in compare",
            })
            b_idx += 1
        else:
            deviations.append({
                "type": "extra_in_compare",
                "event": ce.event,
                "seq_compare": ce.seq,
                "detail": f"Event '{ce.event}' in compare but not in baseline",
            })
            c_idx += 1

    # Remaining unmatched events.
    while b_idx < len(b_events):
        deviations.append({
            "type": "missing_in_compare",
            "event": b_events[b_idx].event,
            "seq_baseline": b_events[b_idx].seq,
        })
        b_idx += 1
    while c_idx < len(c_events):
        deviations.append({
            "type": "extra_in_compare",
            "event": c_events[c_idx].event,
            "seq_compare": c_events[c_idx].seq,
        })
        c_idx += 1

    summary = _build_drift_summary(deviations, baseline.mode, compare.mode)
    return DriftReport(
        baseline_name=f"{baseline.strategy_name} ({baseline.mode})",
        compare_name=f"{compare.strategy_name} ({compare.mode})",
        total_events_baseline=len(baseline.events),
        total_events_compare=len(compare.events),
        matched_events=matched,
        deviations=deviations,
        summary=summary,
    )


def _compare_monetary_fields(be: TraceEvent, ce: TraceEvent,
                              deviations: List[Dict]) -> None:
    """Compare price/volume fields between two matched events."""
    for field in ("price", "volume", "pnl", "commission"):
        bv = be.payload.get(field)
        cv = ce.payload.get(field)
        if bv and cv and bv != cv:
            try:
                bd = Decimal(str(bv))
                cd = Decimal(str(cv))
                delta = abs(bd - cd)
                if delta > Decimal("0.0001"):
                    deviations.append({
                        "type": "value_delta",
                        "seq": be.seq,
                        "field": field,
                        "baseline": str(bv),
                        "compare": str(cv),
                        "delta": str(delta),
                    })
            except Exception:
                pass


def _build_drift_summary(deviations: List[Dict], baseline_mode: str,
                          compare_mode: str) -> str:
    if not deviations:
        return f"No drift detected between {baseline_mode} and {compare_mode}."
    n_delta = sum(1 for d in deviations if d["type"] == "value_delta")
    n_missing = sum(1 for d in deviations if d["type"] == "missing_in_compare")
    n_extra = sum(1 for d in deviations if d["type"] == "extra_in_compare")
    parts = []
    if n_delta:
        parts.append(f"{n_delta} value delta(s)")
    if n_missing:
        parts.append(f"{n_missing} event(s) missing in {compare_mode}")
    if n_extra:
        parts.append(f"{n_extra} extra event(s) in {compare_mode}")
    return f"Drift: {', '.join(parts)}."
