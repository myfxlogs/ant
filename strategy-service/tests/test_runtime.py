"""T1.1 — StrategyRuntime lifecycle driver tests.

Covers:
  - Lifecycle ordering (init before events, deinit last)
  - Exception isolation (strategy crash does not crash runtime)
  - Context injection (broker / ctx / indicators available to strategy)
  - Timer registration / unregistration
  - All 5 T0.4 samples loadable through runtime
  - Idempotent deinit
  - State machine transitions
"""

import unittest
from decimal import Decimal
from typing import List, Optional

from app.sdk import (
    AccountInfo,
    AccountMode,
    Broker,
    Context,
    Indicators,
    OrderRequest,
    OrderResult,
    OrderType,
    PendingOrder,
    Position,
    Retcode,
    StrategyBase,
    SymbolInfo,
)
from app.sdk.runtime import RuntimeContext, StrategyRuntime
from app.sdk.series import Bars, Series


# ── Test doubles ───────────────────────────────────────────────────────


class _FakeSeries(Series):
    """Series backed by a simple list (forward order, mapped to MQL reverse)."""

    def __init__(self, data: List[float]):
        self._data = data

    def __getitem__(self, shift: int) -> float:
        # shift=0 → last element (current bar)
        idx = len(self._data) - 1 - shift
        if idx < 0 or idx >= len(self._data):
            raise IndexError(f"shift {shift} out of range (len={len(self._data)})")
        return self._data[idx]

    def __len__(self) -> int:
        return len(self._data)

    def slice(self, count: int) -> List[float]:
        return self._data[-count:] if count <= len(self._data) else self._data[:]


def _make_bars(close_values: List[float]) -> Bars:
    """Build a Bars object with fake OHLCV data for testing."""
    bars = Bars()
    bars.timeframe = "M15"
    n = len(close_values)
    zeros = [0.0] * n
    bars.open = _FakeSeries([v - 0.0001 for v in close_values])
    bars.high = _FakeSeries([v + 0.0002 for v in close_values])
    bars.low = _FakeSeries([v - 0.0002 for v in close_values])
    bars.close = _FakeSeries(close_values)
    bars.volume = _FakeSeries([100.0] * n)
    bars.time = _FakeSeries([1719000000000.0 + i * 60000 for i in range(n)])
    bars.total = lambda: len(bars.close)
    return bars


class _FakeBroker(Broker):
    """Minimal Broker stub that records calls and returns safe defaults."""

    def __init__(self):
        self.order_log: List[OrderRequest] = []
        self.close_log: List[int] = []
        self.modify_log: List[tuple] = []
        self.delete_log: List[int] = []
        self._account = AccountInfo(
            balance=Decimal("10000"), equity=Decimal("10050"),
            margin=Decimal("500"), free_margin=Decimal("9550"),
            margin_level=Decimal("20.1"), leverage=100,
            currency="USD", mode=AccountMode.HEDGING,
        )
        self._positions: List[Position] = []
        self._orders: List[PendingOrder] = []
        self._next_ticket = 1000

    def order_send(self, request: OrderRequest) -> OrderResult:
        self.order_log.append(request)
        ticket = self._next_ticket
        self._next_ticket += 1
        return OrderResult(
            retcode=Retcode.DONE, ticket=ticket,
            price=request.price or Decimal("1.08500"),
            volume=request.volume,
        )

    def position_modify(self, ticket: int, sl=None, tp=None) -> OrderResult:
        self.modify_log.append((ticket, sl, tp))
        return OrderResult(retcode=Retcode.DONE)

    def position_close(self, ticket: int, volume=None) -> OrderResult:
        self.close_log.append(ticket)
        return OrderResult(retcode=Retcode.DONE, ticket=ticket)

    def order_delete(self, ticket: int) -> OrderResult:
        self.delete_log.append(ticket)
        return OrderResult(retcode=Retcode.DONE)

    def positions(self, symbol=None, magic=None) -> List[Position]:
        result = self._positions
        if symbol is not None:
            result = [p for p in result if p.symbol == symbol]
        if magic is not None:
            result = [p for p in result if p.magic == magic]
        return result

    def orders(self, symbol=None, magic=None) -> List[PendingOrder]:
        result = self._orders
        if symbol is not None:
            result = [o for o in result if o.symbol == symbol]
        if magic is not None:
            result = [o for o in result if o.magic == magic]
        return result

    def deals(self, symbol=None, magic=None, from_ms=None, to_ms=None):
        return []  # not tested in runtime suite

    def account(self) -> AccountInfo:
        return self._account

    def symbol_info(self, symbol: str) -> SymbolInfo:
        return SymbolInfo(
            name=symbol, digits=5, point=Decimal("0.00001"),
            tick_size=Decimal("0.00001"), tick_value=Decimal("1.0"),
            contract_size=Decimal("100000"),
            volume_min=Decimal("0.01"), volume_max=Decimal("100"),
            volume_step=Decimal("0.01"),
            stops_level=0, freeze_level=0,
            swap_long=Decimal("-3.5"), swap_short=Decimal("1.2"),
            margin_rate=Decimal("0.01"),
        )

    def server_time(self) -> int:
        return 1719000000000


class _FakeIndicators(Indicators):
    """Indicators stub — returns synthetic values for testing."""

    def ma(self, period=14, shift=0, method="sma") -> float:
        return 1.08500

    def ema(self, period=14, shift=0) -> float:
        return 1.08500

    def rsi(self, period=14, shift=0) -> float:
        return 50.0

    def bands(self, period=20, deviation=2.0, shift=0):
        return (1.09000, 1.08500, 1.08000)

    def macd(self, fast=12, slow=26, signal=9, shift=0):
        return (0.00050, 0.00030, 0.00020)

    def atr(self, period=14, shift=0) -> float:
        return 0.00100

    def stochastic(self, k_period=5, d_period=3, shift=0):
        return (50.0, 50.0)

    def cci(self, period=14, shift=0) -> float:
        return 0.0

    def i_custom(self, name, params=(), buffer=0, shift=0) -> float:
        return 1.0 if buffer == 0 else 1.08100


def _make_runtime(strategy_class, **kwargs):
    """Factory for a StrategyRuntime with test doubles."""
    return StrategyRuntime(
        strategy_class=strategy_class,
        broker=kwargs.get("broker", _FakeBroker()),
        bars_provider=kwargs.get(
            "bars_provider",
            lambda tf=None: _make_bars([1.08000, 1.08100, 1.08200, 1.08300, 1.08400, 1.08500]),
        ),
        indicators=kwargs.get("indicators", _FakeIndicators()),
        symbol=kwargs.get("symbol", "EURUSD"),
        timeframe=kwargs.get("timeframe", "M15"),
        params=kwargs.get("params", {"fast_period": 12, "slow_period": 26}),
    )


# ── Minimal test strategy ──────────────────────────────────────────────


class _NoTimerStrategy(StrategyBase):
    """Strategy that does NOT register a timer."""

    def on_init(self):
        pass

    def on_timer(self):
        pass


class _SpyStrategy(StrategyBase):
    """Strategy that records every lifecycle callback."""

    def __init__(self):
        self.calls: List[str] = []
        self.args_log: List[tuple] = []

    def on_init(self):
        self.ctx.set_timer(10)  # register timer so on_timer events fire
        self.calls.append("on_init")

    def on_tick(self):
        self.calls.append("on_tick")

    def on_bar(self, timeframe: str):
        self.calls.append(f"on_bar({timeframe})")

    def on_timer(self):
        self.calls.append("on_timer")

    def on_trade(self):
        self.calls.append("on_trade")

    def on_deinit(self, reason: str):
        self.calls.append(f"on_deinit({reason})")


class _CrashingStrategy(StrategyBase):
    """Strategy that throws in every hook — runtime must survive."""

    def on_init(self):
        raise ValueError("init boom")

    def on_tick(self):
        raise RuntimeError("tick boom")

    def on_bar(self, timeframe: str):
        raise RuntimeError("bar boom")

    def on_timer(self):
        raise RuntimeError("timer boom")

    def on_trade(self):
        raise RuntimeError("trade boom")

    def on_deinit(self, reason: str):
        raise RuntimeError("deinit boom")


class _TimerStrategy(StrategyBase):
    """Strategy that registers a timer and tracks timer events."""

    def on_init(self):
        self.ctx.set_timer(5)
        self.timer_fired: bool = False

    def on_timer(self):
        self.timer_fired = True


class _InjectCheckStrategy(StrategyBase):
    """Strategy that verifies broker/ctx/indicators are injected."""

    def on_init(self):
        self.broker_ok = self.broker is not None
        self.ctx_ok = self.ctx is not None
        self.indicators_ok = self.indicators is not None
        self.symbol_ok = self.ctx.symbol == "EURUSD"
        self.timeframe_ok = self.ctx.timeframe == "M15"
        self.param_ok = self.ctx.param("fast_period") == 12


class _DeinitOnlyStrategy(StrategyBase):
    """Strategy that only overrides on_deinit — checks minimal override."""

    def on_deinit(self, reason: str):
        self.reason = reason


# ── Tests ──────────────────────────────────────────────────────────────


class TestLifecycleOrdering(unittest.TestCase):
    """Lifecycle must follow: init → events → deinit."""

    def setUp(self):
        self.broker = _FakeBroker()
        self.runtime = _make_runtime(_SpyStrategy, broker=self.broker)

    def test_full_lifecycle_order(self):
        self.runtime.init()
        strategy = self.runtime.strategy  # capture before deinit clears it
        self.runtime.on_tick()
        self.runtime.on_bar("M15")
        self.runtime.on_timer()
        self.runtime.on_trade()
        self.runtime.deinit("user_stop")

        expected = [
            "on_init",
            "on_tick",
            "on_bar(M15)",
            "on_timer",
            "on_trade",
            "on_deinit(user_stop)",
        ]
        self.assertEqual(strategy.calls, expected)

    def test_deinit_idempotent(self):
        self.runtime.init()
        self.runtime.deinit("user_stop")
        # Second deinit should be a no-op.
        self.runtime.deinit("error")
        self.assertEqual(self.runtime.state, "deinit")

    def test_deinit_without_init(self):
        self.runtime.deinit("user_stop")
        self.assertEqual(self.runtime.state, "deinit")

    def test_cannot_init_twice(self):
        self.runtime.init()
        with self.assertRaises(RuntimeError):
            self.runtime.init()

    def test_cannot_dispatch_after_deinit(self):
        self.runtime.init()
        self.runtime.deinit("user_stop")
        with self.assertRaises(RuntimeError):
            self.runtime.on_tick()
        with self.assertRaises(RuntimeError):
            self.runtime.on_bar("M15")
        with self.assertRaises(RuntimeError):
            self.runtime.on_timer()
        with self.assertRaises(RuntimeError):
            self.runtime.on_trade()

    def test_cannot_dispatch_before_init(self):
        with self.assertRaises(RuntimeError):
            self.runtime.on_tick()
        with self.assertRaises(RuntimeError):
            self.runtime.on_bar("M15")


class TestExceptionIsolation(unittest.TestCase):
    """Strategy exceptions must not crash the runtime."""

    def setUp(self):
        self.runtime = _make_runtime(_CrashingStrategy)

    def test_init_crash_does_not_raise(self):
        # init crash is caught, state becomes 'ready'
        self.runtime.init()
        self.assertEqual(self.runtime.state, "ready")

    def test_tick_crash_does_not_raise(self):
        self.runtime.init()
        self.runtime.on_tick()  # should not raise
        self.assertEqual(self.runtime.state, "ready")

    def test_bar_crash_does_not_raise(self):
        self.runtime.init()
        self.runtime.on_bar("M15")  # should not raise
        self.assertEqual(self.runtime.state, "ready")

    def test_timer_crash_does_not_raise(self):
        self.runtime.init()
        self.runtime.on_timer()  # should not raise
        self.assertEqual(self.runtime.state, "ready")

    def test_trade_crash_does_not_raise(self):
        self.runtime.init()
        self.runtime.on_trade()  # should not raise
        self.assertEqual(self.runtime.state, "ready")

    def test_deinit_crash_does_not_raise(self):
        self.runtime.init()
        self.runtime.deinit("user_stop")  # should not raise
        self.assertEqual(self.runtime.state, "deinit")

    def test_all_crashes_isolated_full_sequence(self):
        """Full lifecycle with crashing strategy — runtime survives every step."""
        self.runtime.init()
        self.runtime.on_tick()
        self.runtime.on_bar("M15")
        self.runtime.on_timer()
        self.runtime.on_trade()
        self.runtime.deinit("user_stop")
        self.assertEqual(self.runtime.state, "deinit")


class TestDependencyInjection(unittest.TestCase):
    """Strategy must see broker, ctx, indicators after init."""

    def test_injection(self):
        broker = _FakeBroker()
        runtime = _make_runtime(_InjectCheckStrategy, broker=broker)
        runtime.init()

        strategy = runtime.strategy
        self.assertTrue(strategy.broker_ok, "broker not injected")
        self.assertTrue(strategy.ctx_ok, "ctx not injected")
        self.assertTrue(strategy.indicators_ok, "indicators not injected")
        self.assertTrue(strategy.symbol_ok, "ctx.symbol incorrect")
        self.assertTrue(strategy.timeframe_ok, "ctx.timeframe incorrect")
        self.assertTrue(strategy.param_ok, "ctx.param incorrect")

    def test_broker_is_same_instance(self):
        broker = _FakeBroker()
        runtime = _make_runtime(_SpyStrategy, broker=broker)
        runtime.init()
        self.assertIs(runtime.strategy.broker, broker)

    def test_indicators_is_same_instance(self):
        ind = _FakeIndicators()
        runtime = _make_runtime(_SpyStrategy, indicators=ind)
        runtime.init()
        self.assertIs(runtime.strategy.indicators, ind)


class TestTimerMechanism(unittest.TestCase):
    """Timer registration / unregistration via ctx."""

    def test_timer_registered(self):
        runtime = _make_runtime(_TimerStrategy)
        self.assertFalse(runtime.is_timer_active)
        runtime.init()
        self.assertTrue(runtime.is_timer_active)

    def test_timer_event_dispatched_when_active(self):
        runtime = _make_runtime(_TimerStrategy)
        runtime.init()
        runtime.on_timer()
        self.assertTrue(runtime.strategy.timer_fired)

    def test_timer_skipped_when_inactive(self):
        runtime = _make_runtime(_NoTimerStrategy)
        runtime.init()
        # _NoTimerStrategy does not register a timer, so on_timer should be skipped.
        runtime.on_timer()
        self.assertFalse(runtime.is_timer_active)

    def test_kill_timer(self):
        runtime = _make_runtime(_TimerStrategy)
        runtime.init()
        self.assertTrue(runtime.is_timer_active)
        runtime.strategy.ctx.kill_timer()
        self.assertFalse(runtime.is_timer_active)


class TestTimerInvalidInterval(unittest.TestCase):
    """Timer registration with invalid interval."""

    def test_zero_seconds_raises(self):
        runtime = _make_runtime(_SpyStrategy)
        runtime.init()
        with self.assertRaises(ValueError):
            runtime.strategy.ctx.set_timer(0)

    def test_negative_seconds_raises(self):
        runtime = _make_runtime(_SpyStrategy)
        runtime.init()
        with self.assertRaises(ValueError):
            runtime.strategy.ctx.set_timer(-5)


class TestRuntimeContext(unittest.TestCase):
    """RuntimeContext standalone tests."""

    def setUp(self):
        self.bars_data = _make_bars([1.08000, 1.08100, 1.08200])
        self.timer_calls: List[int] = []
        self.kill_calls: List[bool] = []
        self.ctx = RuntimeContext(
            symbol="EURUSD",
            timeframe="M15",
            bars_provider=lambda tf=None: self.bars_data,
            params={"key1": "value1", "key2": 42},
            timer_setter=lambda s: self.timer_calls.append(s),
            timer_killer=lambda: self.kill_calls.append(True),
        )

    def test_symbol_and_timeframe(self):
        self.assertEqual(self.ctx.symbol, "EURUSD")
        self.assertEqual(self.ctx.timeframe, "M15")

    def test_param_known(self):
        self.assertEqual(self.ctx.param("key1"), "value1")
        self.assertEqual(self.ctx.param("key2"), 42)

    def test_param_default(self):
        self.assertIsNone(self.ctx.param("nonexistent"))
        self.assertEqual(self.ctx.param("nonexistent", "fallback"), "fallback")

    def test_bars_returns_primary(self):
        bars = self.ctx.bars()
        self.assertEqual(bars.timeframe, "M15")
        self.assertAlmostEqual(bars.close[0], 1.08200, places=5)
        self.assertAlmostEqual(bars.close[1], 1.08100, places=5)
        self.assertAlmostEqual(bars.close[2], 1.08000, places=5)

    def test_bars_multi_timeframe(self):
        h1_bars = _make_bars([1.07000, 1.07500])
        h1_bars.timeframe = "H1"

        def provider(tf=None):
            if tf == "H1":
                return h1_bars
            return self.bars_data

        ctx = RuntimeContext(
            symbol="EURUSD", timeframe="M15",
            bars_provider=provider, params={},
            timer_setter=lambda s: None, timer_killer=lambda: None,
        )
        bars = ctx.bars("H1")
        self.assertEqual(bars.timeframe, "H1")
        self.assertAlmostEqual(bars.close[0], 1.07500, places=5)

    def test_set_timer_delegates(self):
        self.ctx.set_timer(30)
        self.assertEqual(self.timer_calls, [30])

    def test_kill_timer_delegates(self):
        self.ctx.kill_timer()
        self.assertEqual(len(self.kill_calls), 1)


class TestFakeSeries(unittest.TestCase):
    """FakeSeries must match MQL reverse-indexing contract."""

    def setUp(self):
        self.s = _FakeSeries([1.0, 2.0, 3.0, 4.0, 5.0])

    def test_current_bar(self):
        self.assertAlmostEqual(self.s[0], 5.0)

    def test_previous_bar(self):
        self.assertAlmostEqual(self.s[1], 4.0)

    def test_oldest_bar(self):
        self.assertAlmostEqual(self.s[4], 1.0)

    def test_out_of_range(self):
        with self.assertRaises(IndexError):
            _ = self.s[5]

    def test_len(self):
        self.assertEqual(len(self.s), 5)

    def test_slice(self):
        result = self.s.slice(3)
        self.assertEqual(result, [3.0, 4.0, 5.0])

    def test_slice_more_than_available(self):
        result = self.s.slice(10)
        self.assertEqual(result, [1.0, 2.0, 3.0, 4.0, 5.0])


class TestStrategyClassValidation(unittest.TestCase):
    """Runtime must reject non-StrategyBase classes."""

    def test_rejects_plain_class(self):
        class NotAStrategy:
            pass
        with self.assertRaises(TypeError):
            _make_runtime(NotAStrategy)

    def test_rejects_object(self):
        with self.assertRaises(TypeError):
            _make_runtime(object)

    def test_accepts_minimal_strategy(self):
        runtime = _make_runtime(_DeinitOnlyStrategy)
        runtime.init()
        strategy = runtime.strategy  # capture before deinit
        runtime.deinit("user_stop")
        self.assertEqual(strategy.reason, "user_stop")


class TestAllT04SamplesLoadable(unittest.TestCase):
    """Every T0.4 sample EA must be loadable through the runtime."""

    @classmethod
    def setUpClass(cls):
        from tests.sdk_samples.custom_signal import CustomSignal
        from tests.sdk_samples.grid_trader import GridTrader
        from tests.sdk_samples.hedge_twins import HedgeTwins
        from tests.sdk_samples.martingale import Martingale
        from tests.sdk_samples.single_ma_cross import SingleMACross
        cls.samples = [
            SingleMACross,
            GridTrader,
            Martingale,
            HedgeTwins,
            CustomSignal,
        ]

    def _load_and_drive(self, strategy_class):
        broker = _FakeBroker()
        runtime = _make_runtime(strategy_class, broker=broker)
        runtime.init()
        runtime.on_tick()
        runtime.on_bar("M15")
        runtime.on_timer()
        runtime.on_trade()
        runtime.deinit("user_stop")
        return runtime

    def test_single_ma_cross(self):
        rt = self._load_and_drive(self.samples[0])
        self.assertEqual(rt.state, "deinit")

    def test_grid_trader(self):
        rt = self._load_and_drive(self.samples[1])
        self.assertEqual(rt.state, "deinit")

    def test_martingale(self):
        rt = self._load_and_drive(self.samples[2])
        self.assertEqual(rt.state, "deinit")

    def test_hedge_twins(self):
        rt = self._load_and_drive(self.samples[3])
        self.assertEqual(rt.state, "deinit")

    def test_custom_signal(self):
        rt = self._load_and_drive(self.samples[4])
        self.assertEqual(rt.state, "deinit")


class TestStateTransitions(unittest.TestCase):
    """Explicit state machine tests."""

    def test_created_to_ready(self):
        rt = _make_runtime(_SpyStrategy)
        self.assertEqual(rt.state, "created")
        rt.init()
        self.assertEqual(rt.state, "ready")

    def test_ready_to_deinit(self):
        rt = _make_runtime(_SpyStrategy)
        rt.init()
        rt.deinit("user_stop")
        self.assertEqual(rt.state, "deinit")

    def test_created_to_deinit(self):
        rt = _make_runtime(_SpyStrategy)
        rt.deinit("user_stop")
        self.assertEqual(rt.state, "deinit")


class TestStrategyAccess(unittest.TestCase):
    """Strategy property access."""

    def test_none_before_init(self):
        rt = _make_runtime(_SpyStrategy)
        self.assertIsNone(rt.strategy)

    def test_available_after_init(self):
        rt = _make_runtime(_SpyStrategy)
        rt.init()
        self.assertIsNotNone(rt.strategy)
        self.assertIsInstance(rt.strategy, _SpyStrategy)

    def test_none_after_deinit(self):
        rt = _make_runtime(_SpyStrategy)
        rt.init()
        rt.deinit("user_stop")
        self.assertIsNone(rt.strategy)


class TestRuntimeContextBarsError(unittest.TestCase):
    """RuntimeContext.bars() handles missing data gracefully."""

    def test_none_provider_returns_none_bars(self):
        def bad_provider(tf=None):
            raise LookupError("no data")

        ctx = RuntimeContext(
            symbol="X", timeframe="M1",
            bars_provider=bad_provider, params={},
            timer_setter=lambda s: None, timer_killer=lambda: None,
        )
        with self.assertRaises(LookupError):
            ctx.bars()


if __name__ == "__main__":
    unittest.main()
