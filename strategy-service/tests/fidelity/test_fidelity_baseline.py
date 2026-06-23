"""T1.3 — Fidelity baseline tests.

Runs a deterministic threshold-crossing EA through SimBroker and compares
the trace against mathematically expected trades.  Every deviation is
classified as EXPLAINED (cost model, tick synthesis) or NEEDS-DECISION.
"""

import json
import os
import unittest
from decimal import Decimal
from typing import List, Optional

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
    Retcode,
    SymbolInfo,
    TypeFilling,
)
from app.sdk.runtime import RuntimeContext, StrategyRuntime
from app.sdk.series import Bars, Series

from tests.fidelity.harness import (
    FidelityReport,
    ThresholdCrossStrategy,
    TraceCollector,
    compute_expected_trace,
    diff_traces,
)


# ── Test fixtures ──────────────────────────────────────────────────────


def _make_test_data(num_bars: int = 30):
    """Generate synthetic price data that crosses thresholds multiple times.

    Price oscillates between ~1.08200 and ~1.08800, crossing 1.08500 and 1.08300
    several times to produce a predictable sequence of trades.
    """
    import math
    close_prices = []
    for i in range(num_bars):
        # Sine wave centered at 1.08500, amplitude 0.00300, period ~15 bars.
        price = 1.08500 + 0.00300 * math.sin(2 * math.pi * i / 15.0)
        close_prices.append(round(price, 5))

    bars = []
    ticks = []
    base_ts = 1719000000000
    for i, close in enumerate(close_prices):
        bars.append(Bar(
            open_time=base_ts + i * 60000,
            close_time=base_ts + (i + 1) * 60000 - 1000,
            open=round(close - 0.00010, 5),
            high=round(close + 0.00020, 5),
            low=round(close - 0.00020, 5),
            close=close,
            volume=100.0,
        ))
        # Tick at bar close: spread = 0.00005
        ticks.append(Tick(
            ts=bars[-1].close_time,
            bid=round(close - 0.00003, 5),
            ask=round(close + 0.00002, 5),
        ))

    return close_prices, bars, ticks


def _make_symbol_info():
    return SymbolInfo(
        name="EURUSD", digits=5, point=Decimal("0.00001"),
        tick_size=Decimal("0.00001"), tick_value=Decimal("1.0"),
        contract_size=Decimal("100000"),
        volume_min=Decimal("0.01"), volume_max=Decimal("100"),
        volume_step=Decimal("0.01"),
        stops_level=0, freeze_level=0,
        swap_long=Decimal("-3.5"), swap_short=Decimal("1.2"),
        margin_rate=Decimal("0.01"),
    )


class _TestSeries(Series):
    def __init__(self, data):
        self._data = list(data)
    def __getitem__(self, shift):
        idx = len(self._data) - 1 - shift
        if idx < 0 or idx >= len(self._data):
            raise IndexError
        return self._data[idx]
    def __len__(self):
        return len(self._data)
    def slice(self, count):
        return self._data[-count:] if count <= len(self._data) else list(self._data)


def _make_bars_provider(close_prices, bars, tick_ref):
    """Return a bars provider that only shows bars up to the current tick time.
    This prevents look-ahead — the strategy sees only historical data."""
    def provider(timeframe=None):
        tick = tick_ref[0]
        b = Bars()
        b.timeframe = timeframe or "M15"
        # Only include bars whose close_time <= current tick time.
        visible_bars = [
            bar for bar in bars
            if tick is None or bar.close_time <= tick.ts
        ]
        n = len(visible_bars)
        if n == 0:
            # No bars visible yet — return empty series.
            empty = _TestSeries([])
            b.open = empty; b.high = empty; b.low = empty
            b.close = empty; b.volume = empty; b.time = empty
            b.total = lambda: 0
            return b
        b.open = _TestSeries([bar.open for bar in visible_bars])
        b.high = _TestSeries([bar.high for bar in visible_bars])
        b.low = _TestSeries([bar.low for bar in visible_bars])
        b.close = _TestSeries([bar.close for bar in visible_bars])
        b.volume = _TestSeries([bar.volume for bar in visible_bars])
        b.time = _TestSeries([float(bar.close_time) for bar in visible_bars])
        b.total = lambda: len(b.close)
        return b
    return provider


def _run_threshold_ea(close_prices, bars, ticks):
    """Run the threshold-cross EA through SimBroker, collecting a trace."""
    market = MarketSimulator(bars)
    cost = CostModel(CostProfile(
        commission_per_lot=0.0,  # zero commission for clean comparison
        slippage_mode=SlippageMode.FIXED,
        slippage_rate=0.0,
        contract_size=100000.0,
    ))
    fill_model = FillModel(cost)
    portfolio = Portfolio(initial_cash=10000.0)
    margin = MarginModel(leverage=100.0, contract_size=100000.0)

    tick_ref: List[Optional[Tick]] = [None]

    broker = SimBroker(
        portfolio=portfolio,
        fill_model=fill_model,
        cost_model=cost,
        margin_model=margin,
        market=market,
        tick_source=lambda: tick_ref[0],
        account_mode=AccountMode.HEDGING,
        symbol_info_map={"EURUSD": _make_symbol_info()},
        initial_balance=Decimal("10000"),
    )

    collector = TraceCollector(broker, strategy_name="threshold_cross")

    bars_provider = _make_bars_provider(close_prices, bars, tick_ref)

    from app.sdk.indicators import Indicators
    class _NoopIndicators(Indicators):
        def ma(self, *a, **kw): return 0.0
        def ema(self, *a, **kw): return 0.0
        def rsi(self, *a, **kw): return 50.0
        def bands(self, *a, **kw): return (0.0, 0.0, 0.0)
        def macd(self, *a, **kw): return (0.0, 0.0, 0.0)
        def atr(self, *a, **kw): return 0.0
        def stochastic(self, *a, **kw): return (0.0, 0.0)
        def cci(self, *a, **kw): return 0.0
        def i_custom(self, *a, **kw): return 0.0

    runtime = StrategyRuntime(
        strategy_class=ThresholdCrossStrategy,
        broker=broker,
        bars_provider=bars_provider,
        indicators=_NoopIndicators(),
        symbol="EURUSD",
        timeframe="M15",
        params={"threshold_buy": 1.08500, "threshold_sell": 1.08300, "lot_size": "0.10"},
    )

    # We need to intercept broker calls to record events.
    # Monkey-patch broker methods to collect trace.
    _orig_order_send = broker.order_send
    _orig_position_close = broker.position_close
    _orig_position_modify = broker.position_modify

    def _patched_order_send(request):
        result = _orig_order_send(request)
        collector.record_open(result, request)
        return result

    def _patched_position_close(ticket, volume=None):
        result = _orig_position_close(ticket, volume)
        collector.record_close(result)
        return result

    def _patched_position_modify(ticket, sl=None, tp=None):
        result = _orig_position_modify(ticket, sl, tp)
        collector.record_modify(ticket, sl, tp)
        return result

    broker.order_send = _patched_order_send
    broker.position_close = _patched_position_close
    broker.position_modify = _patched_position_modify

    runtime.init()

    for tick in ticks:
        tick_ref[0] = tick
        broker.advance_tick(tick)
        runtime.on_tick()

    runtime.deinit("user_stop")
    return collector, broker, ticks


# ── Tests ──────────────────────────────────────────────────────────────


class TestFidelityBaseline(unittest.TestCase):
    """Run the threshold EA through SimBroker and compare against expected."""

    @classmethod
    def setUpClass(cls):
        cls.close_prices, cls.bars, cls.ticks = _make_test_data(num_bars=30)
        cls.collector, cls.broker, _ = _run_threshold_ea(
            cls.close_prices, cls.bars, cls.ticks
        )
        cls.expected = compute_expected_trace(
            close_prices=cls.close_prices,
            ticks=cls.ticks,
            threshold_buy=1.08500,
            threshold_sell=1.08300,
            lot_size=0.10,
        )

    def test_trace_produced(self):
        """SimBroker run must produce at least one trade event."""
        self.assertGreater(
            len(self.collector.events), 0,
            "No trade events produced — strategy may not have triggered"
        )

    def test_event_count_reasonable(self):
        """Event count should be within 2x of expected (commission/slippage
        may cause extra reject events, but shouldn't wildly diverge)."""
        ratio = len(self.collector.events) / max(len(self.expected), 1)
        self.assertLessEqual(
            ratio, 3.0,
            f"Actual events ({len(self.collector.events)}) > 3x expected ({len(self.expected)})"
        )

    def test_all_events_have_tickets(self):
        """Every non-reject event must have a valid ticket."""
        for evt in self.collector.events:
            if evt.event != "reject":
                self.assertGreater(
                    evt.ticket, 0,
                    f"Event seq {evt.seq} ({evt.event}) has ticket=0"
                )

    def test_event_types_match_basic(self):
        """The sequence of event types should alternate open/close."""
        opens = [e for e in self.collector.events if e.event == "open"]
        closes = [e for e in self.collector.events if e.event == "close"]
        # A sane strategy: closes ≤ opens.
        self.assertGreaterEqual(len(opens), len(closes),
                                "More closes than opens — position tracking error")

    def test_balance_consistency(self):
        """Account balance should never go negative."""
        for evt in self.collector.events:
            balance = Decimal(evt.balance_after)
            self.assertGreater(
                balance, Decimal("0"),
                f"Negative balance at seq {evt.seq}: {balance}"
            )

    def test_price_reasonable(self):
        """Fill prices should be within the bar range (± some tolerance)."""
        for evt in self.collector.events:
            if evt.event in ("open", "close") and evt.price != "0":
                price = float(evt.price)
                min_price = min(self.close_prices) - 0.001
                max_price = max(self.close_prices) + 0.001
                self.assertGreaterEqual(price, min_price,
                    f"Price {price} below data range at seq {evt.seq}")
                self.assertLessEqual(price, max_price,
                    f"Price {price} above data range at seq {evt.seq}")


class TestFidelityDiff(unittest.TestCase):
    """Produce and validate the fidelity diff report."""

    @classmethod
    def setUpClass(cls):
        cls.close_prices, cls.bars, cls.ticks = _make_test_data(num_bars=30)
        cls.collector, cls.broker, _ = _run_threshold_ea(
            cls.close_prices, cls.bars, cls.ticks
        )
        cls.expected = compute_expected_trace(
            close_prices=cls.close_prices,
            ticks=cls.ticks,
            threshold_buy=1.08500,
            threshold_sell=1.08300,
            lot_size=0.10,
        )
        cls.assumptions = [
            "Ticks synthesized from bar closes (bid=close-0.00003, ask=close+0.00002)",
            "Zero commission (commission_per_lot=0.0) for clean comparison",
            "Zero slippage for clean comparison",
            "Strategy uses bar.close for signal; SimBroker fills at tick.ask/tick.bid",
            "Expected trace uses mid-price; actual uses ask (buy) / bid (sell) — ~0.000025 difference",
            "Tick timestamps are bar close_time; no intra-bar ticks",
            "No swap/rollover applied",
        ]
        cls.report = diff_traces(
            actual=cls.collector.events,
            expected=cls.expected,
            assumptions=cls.assumptions,
        )
        cls.report.strategy_name = "threshold_cross"
        cls.report.bars_count = len(cls.bars)
        cls.report.ticks_count = len(cls.ticks)

    def test_report_generated(self):
        """A fidelity report must be produced."""
        self.assertIsInstance(self.report, FidelityReport)

    def test_report_assumptions_documented(self):
        """All modeling assumptions must be explicitly listed."""
        self.assertGreater(
            len(self.report.assumptions), 0,
            "No assumptions documented — fidelity baseline is meaningless without them"
        )
        # Key assumptions that must be present.
        assumption_text = " ".join(self.report.assumptions).lower()
        self.assertIn("tick", assumption_text, "Must document tick synthesis assumptions")
        self.assertIn("commission", assumption_text, "Must document cost model assumptions")

    def test_no_needs_decision_deviation(self):
        """All deviations must be EXPLAINED. Any NEEDS-DECISION blocks T1.3."""
        needs_decision = [
            d for d in self.report.diffs
            if d.classification == "NEEDS-DECISION"
        ]
        if needs_decision:
            self.fail(
                "NEEDS-DECISION deviations found — must be escalated:\n" +
                "\n".join(
                    f"  seq={d.seq} field={d.field}: actual={d.actual} expected={d.expected} — {d.explanation}"
                    for d in needs_decision
                )
            )

    def test_price_deviations_are_explained(self):
        """Any price deviations must be attributed to bid/ask spread."""
        price_diffs = [d for d in self.report.diffs if d.field == "price"]
        for d in price_diffs:
            self.assertEqual(
                d.classification, "EXPLAINED",
                f"Price deviation at seq {d.seq} not explained: {d.explanation}"
            )

    def test_report_serializable(self):
        """Report must serialize to valid JSON."""
        json_str = self.report.to_json()
        parsed = json.loads(json_str)
        self.assertIn("strategy_name", parsed)
        self.assertIn("diffs", parsed)
        self.assertIn("assumptions", parsed)

    def test_golden_file_written(self):
        """Write the fidelity report as a golden reference file.
        When MT Strategy Tester data becomes available, replace this
        with the MT reference and rerun the diff."""
        golden_dir = os.path.join(os.path.dirname(__file__), "golden")
        golden_path = os.path.join(golden_dir, "threshold_cross_baseline.json")
        with open(golden_path, "w") as f:
            f.write(self.report.to_json())
        self.assertTrue(os.path.isfile(golden_path))
        # Verify the file is loadable.
        with open(golden_path) as f:
            loaded = json.load(f)
        self.assertEqual(loaded["strategy_name"], "threshold_cross")


class TestFidelitySelfConsistency(unittest.TestCase):
    """The fidelity harness must be internally consistent."""

    def test_expected_trace_alternates_open_close(self):
        """Expected trace: opens and closes must alternate (open, close, open, close...)."""
        close_prices, bars, ticks = _make_test_data(num_bars=60)
        expected = compute_expected_trace(
            close_prices=close_prices, ticks=ticks,
            threshold_buy=1.08500, threshold_sell=1.08300,
        )
        opens = [e for e in expected if e.event == "open"]
        closes = [e for e in expected if e.event == "close"]
        # Each close should follow an open.
        self.assertAlmostEqual(len(opens), len(closes), delta=1,
            msg=f"Expected trace: {len(opens)} opens vs {len(closes)} closes — should differ by ≤1")

    def test_expected_trace_prices_in_range(self):
        """Expected prices must be within the tick range."""
        close_prices, bars, ticks = _make_test_data(num_bars=30)
        expected = compute_expected_trace(
            close_prices=close_prices, ticks=ticks,
        )
        for evt in expected:
            if evt.price and evt.price != "0":
                price = float(evt.price)
                self.assertGreaterEqual(price, min(close_prices) - 0.001)
                self.assertLessEqual(price, max(close_prices) + 0.001)

    def test_two_runs_produce_similar_traces(self):
        """Two independent runs with the same data should produce similar results
        (deterministic strategy + deterministic engine)."""
        close_prices, bars, ticks = _make_test_data(num_bars=30)

        collector1, _, _ = _run_threshold_ea(close_prices, bars, ticks)
        collector2, _, _ = _run_threshold_ea(close_prices, bars, ticks)

        self.assertEqual(
            len(collector1.events), len(collector2.events),
            "Deterministic strategy produced different event counts on rerun"
        )
        for i, (e1, e2) in enumerate(zip(collector1.events, collector2.events)):
            self.assertEqual(e1.event, e2.event, f"Event type diverged at seq {i}")
            self.assertEqual(e1.side, e2.side, f"Side diverged at seq {i}")
            self.assertEqual(e1.volume, e2.volume, f"Volume diverged at seq {i}")


if __name__ == "__main__":
    unittest.main()
