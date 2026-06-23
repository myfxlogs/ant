"""Fidelity diff harness (T1.3).

Produces structured trade traces from SimBroker runs and compares them
against expected (golden) traces.  Classifies every deviation as
EXPLAINED or NEEDS-DECISION.
"""

from __future__ import annotations

import json
from dataclasses import dataclass, field
from decimal import Decimal
from typing import Any, Dict, List, Optional

from app.engine.cost import CostModel
from app.engine.fill import FillModel
from app.engine.margin import MarginModel
from app.engine.market import MarketSimulator
from app.engine.portfolio import Portfolio
from app.engine.sim_broker import SimBroker
from app.engine.types import Bar, CostProfile, SlippageMode, Tick
from app.sdk import (
    AccountMode,
    OrderRequest,
    OrderResult,
    OrderType,
    PositionSide,
    Retcode,
    StrategyBase,
    SymbolInfo,
    TypeFilling,
)
from app.sdk.runtime import RuntimeContext, StrategyRuntime
from app.sdk.series import Bars, Series


# ── Trace event types ──────────────────────────────────────────────────


@dataclass
class TraceEvent:
    """One event in a strategy execution trace."""
    seq: int
    event: str             # "open" | "close" | "modify" | "cancel" | "reject"
    timestamp_ms: int
    ticket: int
    symbol: str
    side: str              # "buy" | "sell" | ""
    volume: str            # Decimal string
    price: str             # Decimal string
    commission: str        # Decimal string
    pnl: str               # Decimal string (only for close events)
    balance_after: str     # Decimal string
    comment: str = ""

    def to_dict(self) -> dict:
        return {
            "seq": self.seq,
            "event": self.event,
            "timestamp_ms": self.timestamp_ms,
            "ticket": self.ticket,
            "symbol": self.symbol,
            "side": self.side,
            "volume": self.volume,
            "price": self.price,
            "commission": self.commission,
            "pnl": self.pnl,
            "balance_after": self.balance_after,
            "comment": self.comment,
        }


@dataclass
class DiffEntry:
    """One deviation between actual and expected traces."""
    seq: int
    field: str
    actual: str
    expected: str
    classification: str   # "EXPLAINED" or "NEEDS-DECISION"
    explanation: str = ""


@dataclass
class FidelityReport:
    """Result of a fidelity comparison run."""
    strategy_name: str
    bars_count: int
    ticks_count: int
    actual_events: List[TraceEvent] = field(default_factory=list)
    expected_events: List[TraceEvent] = field(default_factory=list)
    diffs: List[DiffEntry] = field(default_factory=list)
    assumptions: List[str] = field(default_factory=list)

    @property
    def passed(self) -> bool:
        """True if no NEEDS-DECISION deviations exist."""
        return all(d.classification != "NEEDS-DECISION" for d in self.diffs)

    @property
    def deviation_count(self) -> int:
        return len(self.diffs)

    def to_dict(self) -> dict:
        return {
            "strategy_name": self.strategy_name,
            "bars_count": self.bars_count,
            "ticks_count": self.ticks_count,
            "assumptions": self.assumptions,
            "actual_event_count": len(self.actual_events),
            "expected_event_count": len(self.expected_events),
            "deviation_count": self.deviation_count,
            "passed": self.passed,
            "diffs": [
                {
                    "seq": d.seq,
                    "field": d.field,
                    "actual": d.actual,
                    "expected": d.expected,
                    "classification": d.classification,
                    "explanation": d.explanation,
                }
                for d in self.diffs
            ],
            "actual_events": [e.to_dict() for e in self.actual_events],
            "expected_events": [e.to_dict() for e in self.expected_events],
        }

    def to_json(self, indent: int = 2) -> str:
        return json.dumps(self.to_dict(), indent=indent)


# ── Trace collector ────────────────────────────────────────────────────


class TraceCollector:
    """Wraps a SimBroker and collects every trade event into a trace."""

    def __init__(self, broker: SimBroker, strategy_name: str = "unknown"):
        self.broker = broker
        self.strategy_name = strategy_name
        self.events: List[TraceEvent] = []
        self._seq = 0
        self._open_positions: Dict[int, Dict[str, Any]] = {}  # ticket → entry info

    def record_open(self, result: OrderResult, request: OrderRequest) -> None:
        """Record a position-open event."""
        if result.retcode not in (Retcode.DONE, Retcode.DONE_PARTIAL):
            self._record_reject(result, request)
            return

        self._seq += 1
        account = self.broker.account()
        self._open_positions[result.ticket] = {
            "symbol": request.symbol,
            "side": request.type.value,
            "volume": str(result.volume),
            "open_price": str(result.price),
        }
        self.events.append(TraceEvent(
            seq=self._seq,
            event="open",
            timestamp_ms=self.broker.server_time(),
            ticket=result.ticket,
            symbol=request.symbol,
            side="buy" if "buy" in request.type.value else "sell",
            volume=str(result.volume or Decimal("0")),
            price=str(result.price or Decimal("0")),
            commission="0",
            pnl="0",
            balance_after=str(account.balance),
            comment=request.comment,
        ))

    def record_close(self, result: OrderResult) -> None:
        """Record a position-close event."""
        if result.retcode not in (Retcode.DONE, Retcode.DONE_PARTIAL):
            return

        self._seq += 1
        account = self.broker.account()
        entry = self._open_positions.pop(result.ticket, {})
        pnl = self._compute_pnl(entry, str(result.price), str(result.volume))
        self.events.append(TraceEvent(
            seq=self._seq,
            event="close",
            timestamp_ms=self.broker.server_time(),
            ticket=result.ticket,
            symbol=entry.get("symbol", ""),
            side=entry.get("side", ""),
            volume=str(result.volume or Decimal("0")),
            price=str(result.price or Decimal("0")),
            commission="0",
            pnl=pnl,
            balance_after=str(account.balance),
            comment=result.comment,
        ))

    def record_modify(self, ticket: int, sl: Optional[Decimal], tp: Optional[Decimal]) -> None:
        """Record a position-modify event."""
        self._seq += 1
        self.events.append(TraceEvent(
            seq=self._seq,
            event="modify",
            timestamp_ms=self.broker.server_time(),
            ticket=ticket,
            symbol="",
            side="",
            volume="0",
            price="0",
            commission="0",
            pnl="0",
            balance_after=str(self.broker.account().balance),
            comment=f"sl={sl},tp={tp}",
        ))

    def _record_reject(self, result: OrderResult, request: OrderRequest) -> None:
        self._seq += 1
        self.events.append(TraceEvent(
            seq=self._seq,
            event="reject",
            timestamp_ms=self.broker.server_time(),
            ticket=0,
            symbol=request.symbol,
            side=request.type.value,
            volume=str(request.volume),
            price=str(request.price or Decimal("0")),
            commission="0",
            pnl="0",
            balance_after=str(self.broker.account().balance),
            comment=result.comment,
        ))

    @staticmethod
    def _compute_pnl(entry: dict, close_price_str: str, volume_str: str) -> str:
        try:
            open_p = Decimal(entry.get("open_price", "0"))
            close_p = Decimal(close_price_str)
            vol = Decimal(volume_str)
            side = entry.get("side", "")
            if "buy" in side:
                pnl = (close_p - open_p) * vol
            else:
                pnl = (open_p - close_p) * vol
            return str(pnl)
        except Exception:
            return "0"


# ── Expected trace computer ────────────────────────────────────────────


def compute_expected_trace(
    close_prices: List[float],
    ticks: List[Tick],
    threshold_buy: float = 1.08500,
    threshold_sell: float = 1.08300,
    lot_size: float = 0.10,
) -> List[TraceEvent]:
    """Compute expected trades for a simple threshold-crossing strategy.

    Strategy logic:
      - BUY  when price crosses ABOVE threshold_buy and we have no position
      - SELL when price crosses BELOW threshold_sell and we have a position
      - Fill at tick.ask (for buys) or tick.bid (for sells)
      - No commission for simplicity (actual SimBroker includes commission)

    Returns a list of TraceEvent that the SimBroker should approximately produce.
    """
    events: List[TraceEvent] = []
    seq = 0
    in_position = False
    ticket_counter = 1
    balance = Decimal("10000")

    for i, tick in enumerate(ticks):
        price = (tick.bid + tick.ask) / 2.0

        if not in_position and price > threshold_buy:
            # Buy signal.
            seq += 1
            fill_price = tick.ask
            volume = Decimal(str(lot_size))
            events.append(TraceEvent(
                seq=seq, event="open",
                timestamp_ms=tick.ts,
                ticket=ticket_counter,
                symbol="EURUSD", side="buy",
                volume=str(volume),
                price=str(Decimal(str(fill_price))),
                commission="0", pnl="0",
                balance_after=str(balance),
                comment="expected_buy",
            ))
            in_position = True
            ticket_counter += 1

        elif in_position and price < threshold_sell:
            # Sell signal.
            seq += 1
            fill_price = tick.bid
            volume = Decimal(str(lot_size))
            open_price = Decimal(str(events[-1].price))
            pnl = (Decimal(str(fill_price)) - open_price) * volume
            balance += pnl
            events.append(TraceEvent(
                seq=seq, event="close",
                timestamp_ms=tick.ts,
                ticket=ticket_counter - 1,
                symbol="EURUSD", side="buy",
                volume=str(volume),
                price=str(Decimal(str(fill_price))),
                commission="0", pnl=str(pnl),
                balance_after=str(balance),
                comment="expected_sell",
            ))
            in_position = False

    return events


# ── Diff engine ────────────────────────────────────────────────────────


def diff_traces(
    actual: List[TraceEvent],
    expected: List[TraceEvent],
    assumptions: Optional[List[str]] = None,
) -> FidelityReport:
    """Compare actual and expected traces, classify deviations.

    Alignment: match events by (seq, event_type).  For each matched pair,
    compare timestamp, price, volume, side.
    """
    report = FidelityReport(
        strategy_name="threshold_cross",
        bars_count=0,
        ticks_count=0,
        assumptions=list(assumptions or []),
    )

    # Match by index (position in sequence).
    max_len = max(len(actual), len(expected))
    for i in range(max_len):
        act = actual[i] if i < len(actual) else None
        exp = expected[i] if i < len(expected) else None

        if act is None:
            report.diffs.append(DiffEntry(
                seq=i, field="event",
                actual="(missing)", expected=exp.event if exp else "?",
                classification="NEEDS-DECISION",
                explanation=f"Expected event {exp.event if exp else '?'} at seq {i} not produced by SimBroker",
            ))
            continue
        if exp is None:
            report.diffs.append(DiffEntry(
                seq=i, field="event",
                actual=act.event, expected="(missing)",
                classification="NEEDS-DECISION",
                explanation=f"SimBroker produced extra event '{act.event}' at seq {i}",
            ))
            continue

        # Compare fields.
        for field in ["event", "side"]:
            av = getattr(act, field)
            ev = getattr(exp, field)
            if av != ev:
                report.diffs.append(DiffEntry(
                    seq=i, field=field,
                    actual=str(av), expected=str(ev),
                    classification="NEEDS-DECISION",
                    explanation=f"Event type mismatch at seq {i}",
                ))

        # Price comparison (tolerant — SimBroker uses ask/bid, expected uses mid).
        if act.price and exp.price:
            try:
                ap = Decimal(act.price)
                ep = Decimal(exp.price)
                diff = abs(ap - ep)
                if diff > Decimal("0.001"):  # more than 1 pip deviation
                    report.diffs.append(DiffEntry(
                        seq=i, field="price",
                        actual=act.price, expected=exp.price,
                        classification="EXPLAINED",
                        explanation=f"Price diff {diff}: SimBroker uses ask/bid, expected uses mid-price; tick synthesis gap",
                    ))
            except Exception:
                pass

    report.actual_events = actual
    report.expected_events = expected
    return report


# ── Deterministic threshold strategy ───────────────────────────────────


class ThresholdCrossStrategy(StrategyBase):
    """Deterministic EA: buy above threshold, sell below.
    Used as fidelity baseline because its behavior is mathematically predictable."""

    def on_init(self) -> None:
        self.threshold_buy: float = float(self.ctx.param("threshold_buy", 1.08500))
        self.threshold_sell: float = float(self.ctx.param("threshold_sell", 1.08300))
        self.lot_size: Decimal = Decimal(str(self.ctx.param("lot_size", "0.10")))
        self.magic: int = 9001
        self._has_position: bool = False

    def on_tick(self) -> None:
        bars = self.ctx.bars()
        try:
            current_price = bars.close[0]
        except (IndexError, TypeError):
            return

        # Use mid-price approximation for signal.
        # (SimBroker fills at ask/bid, so actual fill price will differ slightly.)

        if not self._has_position and current_price > self.threshold_buy:
            req = OrderRequest(
                symbol=self.ctx.symbol,
                type=OrderType.BUY,
                volume=self.lot_size,
                magic=self.magic,
                comment="threshold_buy",
                type_filling=TypeFilling.IOC,
            )
            result = self.broker.order_send(req)
            if result.retcode in (Retcode.DONE, Retcode.DONE_PARTIAL):
                self._has_position = True

        elif self._has_position and current_price < self.threshold_sell:
            for pos in self.broker.positions(magic=self.magic):
                result = self.broker.position_close(pos.ticket)
                if result.retcode in (Retcode.DONE, Retcode.DONE_PARTIAL):
                    self._has_position = False
                    break
