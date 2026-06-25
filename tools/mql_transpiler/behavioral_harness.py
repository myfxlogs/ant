"""Behavioral alignment harness — compares translated strategy signals to
hand-written SDK reference strategies (oracles).

ADR-0020 D8, C2: the ONLY trustworthiness metric for a translated strategy is
"does it produce the same trade signals as the hand-written reference when
fed the same data?"

Architecture
  - RecordingBroker: wraps Broker interface, records every API call.
  - StubContext: provides bar data from a golden bar sequence.
  - SignalComparator: diffs two recordings → pass/fail with details.

The 5 hand-written SDK reference strategies (tests/sdk_samples/*.py) serve as
oracles.  For each MQL fixture, the corresponding reference strategy is the
ground truth.

Usage after T3 (when codegen produces compilable Python)::

    from tools.mql_transpiler.behavioral_harness import (
        RecordingBroker, StubContext, run_strategy_with_recording,
        SignalComparator, BehavioralResult,
    )
    # Load reference strategy.
    ref_broker = RecordingBroker()
    ref_ctx = StubContext(golden_bars)
    ref_signals = run_strategy_with_recording(ReferenceStrategy, ref_broker, ref_ctx)

    # Load translated strategy (from transpiler output).
    trans_broker = RecordingBroker()
    trans_ctx = StubContext(golden_bars)
    trans_signals = run_strategy_with_recording(TranslatedStrategy, trans_broker, trans_ctx)

    # Compare.
    result = SignalComparator.compare(ref_signals, trans_signals)
    assert result.match, f"Behavioral mismatch: {result.diff}"
"""

from __future__ import annotations

from dataclasses import dataclass, field
from decimal import Decimal
from typing import Any, Callable, Dict, List, Optional, Tuple

from app.sdk.account import AccountInfo, AccountMode
from app.sdk.broker import Broker
from app.sdk.context import Context
from app.sdk.indicators import Indicators
from app.sdk.series import Bars, Series
from app.sdk.strategy_base import StrategyBase
from app.sdk.symbol import SymbolInfo
from app.sdk.types import (
    Deal,
    OrderRequest,
    OrderResult,
    OrderType,
    PendingOrder,
    Position,
    PositionSide,
    Retcode,
    TypeFilling,
)


# ── Golden bar sequence ──────────────────────────────────────────────────

@dataclass
class GoldenBar:
    """A single bar in a golden test sequence."""
    open: float
    high: float
    low: float
    close: float
    volume: float = 0.0
    timestamp: int = 0


# ── Recording types ──────────────────────────────────────────────────────

@dataclass
class RecordedCall:
    """A single broker API call."""
    method: str                # "order_send", "position_close", "position_modify", etc.
    args: Dict[str, Any] = field(default_factory=dict)
    bar_index: int = 0         # Which bar were we on when this call happened?


@dataclass
class StrategyRecording:
    """Complete recording of a strategy's broker calls during execution."""
    strategy_name: str
    calls: List[RecordedCall] = field(default_factory=list)
    # Per-bar signal summary: for each bar, what did the strategy do?
    bar_signals: Dict[int, List[str]] = field(default_factory=dict)


# ── RecordingBroker ──────────────────────────────────────────────────────

class RecordingBroker(Broker):
    """Wraps the Broker interface, recording every API call.

    Does NOT execute real trades — just records what the strategy asked for.
    Returns plausible stub results so the strategy logic can continue.
    """

    def __init__(self, symbol: str = "EURUSD"):
        self._symbol = symbol
        self.recording = StrategyRecording(strategy_name="")
        self._current_bar: int = 0
        self._ticket_counter: int = 1000
        self._callbacks: Dict[str, List[RecordedCall]] = {
            "order_send": [],
            "position_close": [],
            "position_modify": [],
            "order_delete": [],
            "orders": [],
            "positions": [],
            "account": [],
            "symbol_info": [],
            "history_orders": [],
            "deals": [],
        }

    def set_bar(self, index: int) -> None:
        self._current_bar = index

    def _record(self, method: str, args: Dict[str, Any]) -> None:
        call = RecordedCall(method=method, args=args, bar_index=self._current_bar)
        self._callbacks[method].append(call)
        self.recording.calls.append(call)
        # Per-bar summary.
        if self._current_bar not in self.recording.bar_signals:
            self.recording.bar_signals[self._current_bar] = []
        self.recording.bar_signals[self._current_bar].append(
            f"{method}({', '.join(f'{k}={v}' for k, v in args.items())})"
        )

    def _next_ticket(self) -> int:
        self._ticket_counter += 1
        return self._ticket_counter

    # ── Broker interface ─────────────────────────────────────────────

    def order_send(self, req: OrderRequest) -> OrderResult:
        self._record("order_send", {
            "symbol": req.symbol,
            "type": req.type.name if hasattr(req.type, "name") else str(req.type),
            "volume": str(req.volume) if req.volume else "0",
            "price": str(req.price) if req.price else "market",
            "sl": str(req.sl) if req.sl else "None",
            "tp": str(req.tp) if req.tp else "None",
            "magic": req.magic,
            "comment": req.comment or "",
        })
        ticket = self._next_ticket()
        return OrderResult(
            retcode=Retcode.DONE,
            ticket=ticket,
            volume=req.volume or Decimal("0"),
            price=req.price or Decimal("0"),
        )

    def position_close(self, ticket: int, volume: Optional[Decimal] = None) -> OrderResult:
        self._record("position_close", {
            "ticket": ticket,
            "volume": str(volume) if volume else "all",
        })
        return OrderResult(retcode=Retcode.DONE, ticket=ticket)

    def position_modify(self, ticket: int, sl: Optional[Decimal] = None,
                         tp: Optional[Decimal] = None) -> OrderResult:
        self._record("position_modify", {
            "ticket": ticket,
            "sl": str(sl) if sl else "None",
            "tp": str(tp) if tp else "None",
        })
        return OrderResult(retcode=Retcode.DONE, ticket=ticket)

    def order_delete(self, ticket: int) -> OrderResult:
        self._record("order_delete", {"ticket": ticket})
        return OrderResult(retcode=Retcode.DONE, ticket=ticket)

    def orders(self, magic: Optional[int] = None) -> List[PendingOrder]:
        self._record("orders", {"magic": magic})
        return []

    def positions(self, magic: Optional[int] = None) -> List[Position]:
        self._record("positions", {"magic": magic})
        return []

    def history_orders(self, from_time: Optional[int] = None,
                        to_time: Optional[int] = None) -> List[Position]:
        self._record("history_orders", {})
        return []

    def deals(self, from_time: Optional[int] = None,
              to_time: Optional[int] = None,
              magic: Optional[int] = None) -> List[Deal]:
        self._record("deals", {})
        return []

    def account(self) -> AccountInfo:
        self._record("account", {})
        return AccountInfo(
            balance=Decimal("10000"),
            equity=Decimal("10000"),
            margin=Decimal("0"),
            free_margin=Decimal("10000"),
            leverage=100,
            currency="USD",
            mode=AccountMode.HEDGING,
        )

    def symbol_info(self, symbol: str) -> Optional[SymbolInfo]:
        self._record("symbol_info", {"symbol": symbol})
        return SymbolInfo(
            name=symbol,
            digits=5,
            point=Decimal("0.00001"),
            volume_min=Decimal("0.01"),
            volume_max=Decimal("100"),
            volume_step=Decimal("0.01"),
        )

    def server_time(self) -> int:
        return 0


# ── ListSeries — minimal real Series backed by a list ───────────────────

class ListSeries(Series):
    """A Series backed by a plain Python list with MQL inverse-indexing.

    series[0] = current (last element), series[1] = previous, etc.
    """

    def __init__(self, data: List[float]):
        self._data = list(data)  # stored in chronological order

    def __getitem__(self, shift: int) -> float:
        # shift=0 → last element, shift=1 → second-to-last, etc.
        idx = len(self._data) - 1 - shift
        if idx < 0 or idx >= len(self._data):
            raise IndexError(f"shift={shift} out of range (len={len(self._data)})")
        return self._data[idx]

    def __len__(self) -> int:
        return len(self._data)

    def slice(self, count: int) -> List[float]:
        return self._data[-count:]


class ListBars(Bars):
    """Bars backed by plain Python lists."""

    def __init__(self, open_vals, high_vals, low_vals, close_vals,
                 volume_vals, time_vals, timeframe: str):
        self.open = ListSeries(open_vals)
        self.high = ListSeries(high_vals)
        self.low = ListSeries(low_vals)
        self.close = ListSeries(close_vals)
        self.volume = ListSeries(volume_vals)
        self.time = ListSeries(time_vals)
        self.timeframe = timeframe

    def total(self) -> int:
        return len(self.open)


# ── StubContext ──────────────────────────────────────────────────────────

class StubContext(Context):
    """Provides golden bar data to a strategy during behavioral testing."""

    def __init__(self, bars: List[GoldenBar], symbol: str = "EURUSD",
                 timeframe: str = "H1"):
        self._bars_data = bars
        self.symbol = symbol
        self.timeframe = timeframe
        self._params: Dict[str, Any] = {}
        self._timer_set: bool = False

    def bars(self, timeframe: Optional[str] = None) -> Bars:
        """Return bars as a ListBars with MQL inverse-index semantics."""
        open_vals = [b.open for b in self._bars_data]
        high_vals = [b.high for b in self._bars_data]
        low_vals = [b.low for b in self._bars_data]
        close_vals = [b.close for b in self._bars_data]
        volume_vals = [b.volume for b in self._bars_data]
        time_vals = [float(b.timestamp) for b in self._bars_data]
        return ListBars(
            open_vals=open_vals,
            high_vals=high_vals,
            low_vals=low_vals,
            close_vals=close_vals,
            volume_vals=volume_vals,
            time_vals=time_vals,
            timeframe=timeframe or self.timeframe,
        )

    def param(self, name: str, default: object = None) -> object:
        return self._params.get(name, default)

    def set_timer(self, seconds: int) -> None:
        self._timer_set = True

    def kill_timer(self) -> None:
        self._timer_set = False

    def set_params(self, **kwargs) -> None:
        self._params.update(kwargs)


# ── Signal comparator ────────────────────────────────────────────────────

@dataclass
class BehavioralResult:
    """Result of comparing translated strategy behavior to reference."""
    match: bool
    reference_name: str
    translated_name: str
    total_ref_calls: int
    total_trans_calls: int
    matching_calls: int
    missing_calls: List[str] = field(default_factory=list)
    extra_calls: List[str] = field(default_factory=list)
    mismatched_calls: List[Tuple[str, str]] = field(default_factory=list)
    # Per-bar breakdown.
    bar_diffs: Dict[int, str] = field(default_factory=dict)

    @property
    def call_match_rate(self) -> float:
        total = max(self.total_ref_calls, self.total_trans_calls)
        if total == 0:
            return 1.0
        return self.matching_calls / total


class SignalComparator:
    """Compare two strategy recordings for behavioral alignment."""

    @staticmethod
    def compare(ref: StrategyRecording, trans: StrategyRecording) -> BehavioralResult:
        """Compare a translated strategy's signals to the reference oracle.

        Matching is strict on method name + key args (symbol, type, magic).
        """
        # Normalize calls to comparable strings.
        ref_sigs = SignalComparator._normalize_calls(ref.calls)
        trans_sigs = SignalComparator._normalize_calls(trans.calls)

        # Match by method + key fields.
        ref_set = set(ref_sigs)
        trans_set = set(trans_sigs)

        matching = ref_set & trans_set
        missing = ref_set - trans_set
        extra = trans_set - ref_set

        # Per-bar comparison.
        all_bars = sorted(set(ref.bar_signals.keys()) | set(trans.bar_signals.keys()))
        bar_diffs = {}
        for bar in all_bars:
            ref_bar = set(ref.bar_signals.get(bar, []))
            trans_bar = set(trans.bar_signals.get(bar, []))
            if ref_bar != trans_bar:
                bar_diffs[bar] = f"bar {bar}: ref={len(ref_bar)} signals, trans={len(trans_bar)} signals"

        return BehavioralResult(
            match=len(missing) == 0 and len(extra) == 0,
            reference_name=ref.strategy_name,
            translated_name=trans.strategy_name,
            total_ref_calls=len(ref_sigs),
            total_trans_calls=len(trans_sigs),
            matching_calls=len(matching),
            missing_calls=sorted(missing),
            extra_calls=sorted(extra),
            bar_diffs=bar_diffs,
        )

    @staticmethod
    def _normalize_calls(calls: List[RecordedCall]) -> List[str]:
        """Convert calls to comparable normalized strings."""
        norms = []
        for c in calls:
            # Keep method + key identity fields.
            key_fields = {"symbol", "type", "magic", "ticket"}
            key_parts = [f"{k}={c.args[k]}" for k in sorted(key_fields) if k in c.args]
            norms.append(f"{c.method}({','.join(key_parts)})")
        return norms


# ── Strategy runner ──────────────────────────────────────────────────────

def run_strategy_with_recording(
    strategy_cls: type,
    broker: RecordingBroker,
    ctx: StubContext,
    bars: Optional[List[GoldenBar]] = None,
) -> StrategyRecording:
    """Instantiate and run a strategy through a bar sequence, recording all broker calls.

    Args:
        strategy_cls: A ``StrategyBase`` subclass (reference or translated).
        broker: A ``RecordingBroker`` instance.
        ctx: A ``StubContext`` with golden bar data.
        bars: Optional bar sequence (uses ctx.bars() if not provided).

    Returns:
        ``StrategyRecording`` with all broker calls made during execution.
    """
    broker.recording.strategy_name = strategy_cls.__name__

    # Instantiate the strategy and inject broker/ctx/indicators.
    strategy = strategy_cls()
    strategy.broker = broker
    strategy.ctx = ctx
    strategy.indicators = Indicators()

    # on_init.
    strategy.on_init()

    # For each bar, call on_bar.
    if bars is None:
        bars = []  # Use ctx's internal bars if needed.
    else:
        ctx._bars_data = bars

    num_bars = len(ctx._bars_data) if ctx._bars_data else 0

    for i in range(num_bars):
        broker.set_bar(i)
        try:
            strategy.on_bar(ctx.timeframe)
        except Exception:
            # Continue — partial failure should still record what happened.
            pass

    # on_deinit.
    strategy.on_deinit("user_stop")

    return broker.recording


# ── Golden bar generators ────────────────────────────────────────────────

def generate_trending_up_bars(n: int = 50) -> List[GoldenBar]:
    """Generate a simple uptrend bar sequence (for testing trend-following EAs)."""
    bars = []
    price = 1.10000
    for i in range(n):
        o = price
        c = price + 0.00020
        h = c + 0.00010
        l = o - 0.00005
        bars.append(GoldenBar(open=o, high=h, low=l, close=c, volume=100, timestamp=i))
        price = c
    return bars


def generate_ranging_bars(n: int = 50) -> List[GoldenBar]:
    """Generate a ranging/sideways bar sequence."""
    import math
    bars = []
    for i in range(n):
        mid = 1.10000 + math.sin(i * 0.1) * 0.00050
        o = mid
        c = mid + 0.00010
        h = max(o, c) + 0.00005
        l = min(o, c) - 0.00005
        bars.append(GoldenBar(open=o, high=h, low=l, close=c, volume=100, timestamp=i))
    return bars


def generate_crossover_bars(n: int = 50) -> List[GoldenBar]:
    """Generate bars that cause EMA crossover (for MA cross EAs).

    First half: downtrend → slow > fast.  Second half: uptrend → fast > slow.
    """
    bars = []
    price = 1.12000
    for i in range(n):
        if i < n // 2:
            # Downtrend.
            o = price
            c = price - 0.00050
        else:
            # Uptrend — crossover happens here.
            o = price
            c = price + 0.00050
        h = max(o, c) + 0.00010
        l = min(o, c) - 0.00010
        bars.append(GoldenBar(open=o, high=h, low=l, close=c, volume=100, timestamp=i))
        price = c
    return bars


# ── Fixture → reference mapping ──────────────────────────────────────────

# Maps MQL fixture filenames to their hand-written SDK oracle classes.
FIXTURE_ORACLE_MAP: Dict[str, type] = {}

def _lazy_load_oracles():
    """Lazy-load the 5 reference strategy classes to avoid import-time deps."""
    global FIXTURE_ORACLE_MAP
    if FIXTURE_ORACLE_MAP:
        return
    from tests.sdk_samples.single_ma_cross import SingleMACross
    from tests.sdk_samples.grid_trader import GridTrader
    from tests.sdk_samples.martingale import Martingale
    from tests.sdk_samples.hedge_twins import HedgeTwins
    from tests.sdk_samples.custom_signal import CustomSignal

    FIXTURE_ORACLE_MAP = {
        "simple_ma_cross.mq4": SingleMACross,
        "grid_trader.mq4": GridTrader,
        "martingale.mq4": Martingale,
        "hedge_twins.mq4": HedgeTwins,
        "custom_signal.mq4": CustomSignal,
    }
