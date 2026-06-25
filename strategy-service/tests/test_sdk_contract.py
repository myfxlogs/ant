"""T0.1 + T0.2 contract freeze tests.

Validates that the Strategy SDK type stubs and Broker ABC conform to the
frozen contracts defined in docs/spec/30-strategy-sdk.md and docs/spec/31-risk-gate.md.

These tests assert the interface shape — NOT implementation behavior.
They must pass before any implementation work begins (Phase 1+).
"""

import ast
import os
import unittest
from abc import abstractmethod
from decimal import Decimal

import app.sdk
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
    PositionSide,
    Retcode,
    Series,
    StrategyBase,
    SymbolInfo,
    TypeFilling,
)
from app.sdk.series import Bars


def _sample_symbol_info():
    return SymbolInfo(
        name="EURUSD",
        digits=5,
        point=Decimal("0.00001"),
        tick_size=Decimal("0.00001"),
        tick_value=Decimal("1.0"),
        contract_size=Decimal("100000"),
        volume_min=Decimal("0.01"),
        volume_max=Decimal("100"),
        volume_step=Decimal("0.01"),
        stops_level=0,
        freeze_level=0,
        swap_long=Decimal("-3.5"),
        swap_short=Decimal("1.2"),
        margin_rate=Decimal("0.01"),
    )


def _sample_account_info():
    return AccountInfo(
        balance=Decimal("10000"),
        equity=Decimal("10050"),
        margin=Decimal("500"),
        free_margin=Decimal("9550"),
        margin_level=Decimal("20.1"),
        leverage=100,
        currency="USD",
        mode=AccountMode.HEDGING,
    )


# ── T0.1: Strategy SDK specification ───────────────────────────────────


class TestStrategyLifecycle(unittest.TestCase):
    """Section 3: StrategyBase lifecycle hooks must mirror MQL event model."""

    REQUIRED_HOOKS = [
        "on_init", "on_tick", "on_bar", "on_timer", "on_trade", "on_deinit",
    ]

    def test_all_hooks_present(self):
        for hook in self.REQUIRED_HOOKS:
            self.assertTrue(hasattr(StrategyBase, hook), f"missing hook: {hook}")

    def test_hooks_are_callable(self):
        for hook in self.REQUIRED_HOOKS:
            self.assertTrue(callable(getattr(StrategyBase, hook)), f"{hook} is not callable")


class TestSymbolInfo(unittest.TestCase):
    """Section 6: SymbolInfo must expose all MQL SYMBOL_* attributes."""

    REQUIRED_FIELDS = [
        "name", "digits", "point", "tick_size", "tick_value",
        "contract_size", "volume_min", "volume_max", "volume_step",
        "stops_level", "freeze_level", "swap_long", "swap_short", "margin_rate",
    ]

    def setUp(self):
        self.si = _sample_symbol_info()

    def test_all_fields_present(self):
        for field in self.REQUIRED_FIELDS:
            self.assertTrue(hasattr(self.si, field), f"missing field: {field}")

    def test_digits_is_int(self):
        self.assertIsInstance(self.si.digits, int)

    def test_monetary_fields_are_decimal(self):
        monetary = [
            "point", "tick_size", "tick_value", "contract_size",
            "volume_min", "volume_max", "volume_step",
            "swap_long", "swap_short", "margin_rate",
        ]
        for field in monetary:
            val = getattr(self.si, field)
            self.assertIsInstance(val, Decimal, f"{field} is {type(val)}, not Decimal")

    def test_normalize_price_is_stub(self):
        with self.assertRaises(NotImplementedError):
            self.si.normalize_price(Decimal("1.234567"))

    def test_normalize_volume_is_stub(self):
        with self.assertRaises(NotImplementedError):
            self.si.normalize_volume(Decimal("0.123"))


class TestAccountInfo(unittest.TestCase):
    """Section 7: AccountInfo must mirror MQL ACCOUNT_* attributes."""

    REQUIRED_FIELDS = [
        "balance", "equity", "margin", "free_margin",
        "margin_level", "leverage", "currency", "mode",
    ]

    def setUp(self):
        self.ai = _sample_account_info()

    def test_all_fields_present(self):
        for field in self.REQUIRED_FIELDS:
            self.assertTrue(hasattr(self.ai, field), f"missing field: {field}")

    def test_monetary_fields_are_decimal(self):
        for field in ["balance", "equity", "margin", "free_margin", "margin_level"]:
            val = getattr(self.ai, field)
            self.assertIsInstance(val, Decimal, f"{field} is {type(val)}, not Decimal")

    def test_mode_is_enum(self):
        self.assertEqual(self.ai.mode, AccountMode.HEDGING)

    def test_netting_mode(self):
        ai_net = AccountInfo(
            balance=Decimal("10000"), equity=Decimal("10000"),
            margin=Decimal("0"), free_margin=Decimal("10000"),
            margin_level=Decimal("0"), leverage=100,
            currency="USD", mode=AccountMode.NETTING,
        )
        self.assertEqual(ai_net.mode, AccountMode.NETTING)


class TestSeries(unittest.TestCase):
    """Section 5: Series must support MQL reverse indexing."""

    def test_getitem_exists(self):
        self.assertTrue(hasattr(Series, "__getitem__"))

    def test_len_exists(self):
        self.assertTrue(hasattr(Series, "__len__"))

    def test_slice_exists(self):
        self.assertTrue(hasattr(Series, "slice"))

    def test_bars_has_all_series_fields(self):
        bars_fields = ["open", "high", "low", "close", "volume", "time", "timeframe"]
        for field in bars_fields:
            self.assertIn(field, Bars.__annotations__, f"missing Bars field: {field}")


class TestIndicators(unittest.TestCase):
    """Section 8: All required indicators must be present."""

    REQUIRED = ["ma", "ema", "rsi", "bands", "macd", "atr", "stochastic", "cci", "i_custom"]

    def test_all_indicators_present(self):
        for name in self.REQUIRED:
            self.assertTrue(hasattr(Indicators, name), f"missing indicator: {name}")
            self.assertTrue(callable(getattr(Indicators, name)), f"{name} is not callable")

    def test_i_custom_signature(self):
        import inspect
        sig = inspect.signature(Indicators.i_custom)
        params = list(sig.parameters.keys())
        for p in ["name", "params", "buffer", "shift"]:
            self.assertIn(p, params, f"i_custom missing param: {p}")


class TestContext(unittest.TestCase):
    """Section 4: Context must provide bars/params/timer."""

    def test_bars_method(self):
        self.assertTrue(hasattr(Context, "bars"))
        self.assertTrue(callable(Context.bars))

    def test_param_method(self):
        self.assertTrue(hasattr(Context, "param"))
        self.assertTrue(callable(Context.param))

    def test_timer_methods(self):
        for m in ["set_timer", "kill_timer"]:
            self.assertTrue(hasattr(Context, m), f"missing: {m}")
            self.assertTrue(callable(getattr(Context, m)), f"{m} not callable")


# ── T0.2: Broker interface ─────────────────────────────────────────────


class TestOrderTypes(unittest.TestCase):
    """Section 9.1: All 8 MT order types must exist."""

    def test_count(self):
        self.assertEqual(len(set(OrderType)), 8)

    def test_market_types(self):
        self.assertIsNotNone(OrderType.BUY)
        self.assertIsNotNone(OrderType.SELL)

    def test_pending_types(self):
        for ot in [OrderType.BUY_LIMIT, OrderType.SELL_LIMIT,
                    OrderType.BUY_STOP, OrderType.SELL_STOP,
                    OrderType.BUY_STOP_LIMIT, OrderType.SELL_STOP_LIMIT]:
            self.assertIsNotNone(ot)


class TestRetcodes(unittest.TestCase):
    """Section 9.4: All 9 retcodes including RISK_BLOCKED."""

    def test_count(self):
        self.assertEqual(len(set(Retcode)), 9)

    def test_risk_blocked_exists(self):
        self.assertIsNotNone(Retcode.RISK_BLOCKED)


class TestTypeFilling(unittest.TestCase):
    """Section 9.3: All 3 filling modes."""

    def test_count(self):
        self.assertEqual(len(set(TypeFilling)), 3)

    def test_variants(self):
        for v in [TypeFilling.FOK, TypeFilling.IOC, TypeFilling.RETURN]:
            self.assertIsNotNone(v)


class TestOrderRequest(unittest.TestCase):
    """Section 9.2: OrderRequest must accept all required fields."""

    def test_full_construction(self):
        req = OrderRequest(
            symbol="EURUSD", type=OrderType.BUY_LIMIT, volume=Decimal("0.10"),
            price=Decimal("1.08000"), sl=Decimal("1.07500"), tp=Decimal("1.09500"),
            deviation=10, magic=42, comment="test",
            type_filling=TypeFilling.FOK, stop_limit_price=Decimal("1.08100"),
        )
        self.assertEqual(req.symbol, "EURUSD")
        self.assertEqual(req.type, OrderType.BUY_LIMIT)
        self.assertEqual(req.volume, Decimal("0.10"))
        self.assertEqual(req.magic, 42)

    def test_market_order_no_price(self):
        req = OrderRequest(symbol="EURUSD", type=OrderType.BUY, volume=Decimal("0.10"))
        self.assertIsNone(req.price)

    def test_volume_is_decimal(self):
        req = OrderRequest(symbol="EURUSD", type=OrderType.BUY, volume=Decimal("0.10"))
        self.assertIsInstance(req.volume, Decimal)

    def test_price_when_set_is_decimal(self):
        req = OrderRequest(
            symbol="EURUSD", type=OrderType.BUY_LIMIT,
            volume=Decimal("0.10"), price=Decimal("1.08500"),
        )
        self.assertIsInstance(req.price, Decimal)


class TestOrderResult(unittest.TestCase):
    """Section 9.4: OrderResult must support partial fills."""

    def test_full_fill(self):
        res = OrderResult(retcode=Retcode.DONE, ticket=12345,
                          price=Decimal("1.08500"), volume=Decimal("0.10"))
        self.assertEqual(res.retcode, Retcode.DONE)
        self.assertEqual(res.ticket, 12345)

    def test_partial_fill(self):
        res = OrderResult(retcode=Retcode.DONE_PARTIAL, ticket=12346,
                          price=Decimal("1.08500"), volume=Decimal("0.05"),
                          comment="partial")
        self.assertEqual(res.retcode, Retcode.DONE_PARTIAL)
        self.assertLess(res.volume, Decimal("0.10"))

    def test_risk_blocked(self):
        res = OrderResult(retcode=Retcode.RISK_BLOCKED, comment="Go risk gate: R4 drawdown")
        self.assertEqual(res.retcode, Retcode.RISK_BLOCKED)
        self.assertIsNone(res.ticket)

    def test_price_and_volume_are_decimal(self):
        res = OrderResult(retcode=Retcode.DONE, ticket=1,
                          price=Decimal("1.08500"), volume=Decimal("0.10"))
        self.assertIsInstance(res.price, Decimal)
        self.assertIsInstance(res.volume, Decimal)


class TestPosition(unittest.TestCase):
    """Section 9.6: Position dataclass."""

    def test_full_construction(self):
        pos = Position(
            ticket=1, symbol="EURUSD", side=PositionSide.BUY,
            volume=Decimal("0.10"), open_price=Decimal("1.08500"),
            sl=Decimal("1.08000"), tp=Decimal("1.09500"),
            profit=Decimal("5.00"), swap=Decimal("-0.30"),
            magic=42, comment="grid", open_time_ms=1719000000000,
        )
        self.assertEqual(pos.ticket, 1)
        self.assertEqual(pos.side, PositionSide.BUY)
        self.assertIsInstance(pos.volume, Decimal)
        self.assertIsInstance(pos.open_price, Decimal)
        self.assertIsInstance(pos.profit, Decimal)
        self.assertIsInstance(pos.swap, Decimal)

    def test_defaults(self):
        pos = Position(ticket=2, symbol="GBPUSD", side=PositionSide.SELL,
                       volume=Decimal("0.50"), open_price=Decimal("1.30000"))
        self.assertEqual(pos.profit, Decimal("0"))
        self.assertEqual(pos.swap, Decimal("0"))
        self.assertEqual(pos.magic, 0)
        self.assertIsNone(pos.sl)


class TestPendingOrder(unittest.TestCase):
    """Section 9.6: PendingOrder dataclass."""

    def test_construction(self):
        po = PendingOrder(
            ticket=3, symbol="EURUSD", type=OrderType.BUY_LIMIT,
            volume=Decimal("0.10"), price=Decimal("1.08000"),
            magic=42, comment="entry",
        )
        self.assertEqual(po.ticket, 3)
        self.assertEqual(po.type, OrderType.BUY_LIMIT)
        self.assertIsInstance(po.price, Decimal)


class TestBrokerABC(unittest.TestCase):
    """Section 9.5: Broker ABC must enforce all 9 methods."""

    REQUIRED_METHODS = [
        "order_send", "position_modify", "position_close",
        "order_delete", "positions", "orders",
        "account", "symbol_info", "server_time",
    ]

    def test_abstract_method_count(self):
        abstract = [
            n for n, v in vars(Broker).items()
            if hasattr(v, "__isabstractmethod__") and v.__isabstractmethod__
        ]
        self.assertEqual(len(abstract), 10,
                         f"expected 10 abstract methods, got {len(abstract)}: {abstract}")

    def test_empty_subclass_blocked(self):
        class EmptyBroker(Broker):
            pass
        with self.assertRaises(TypeError):
            EmptyBroker()

    def test_full_subclass_allowed(self):
        class FullBroker(Broker):
            def order_send(self, req):
                return OrderResult(retcode=Retcode.DONE)

            def position_modify(self, t, sl=None, tp=None):
                return OrderResult(retcode=Retcode.DONE)

            def position_close(self, t, v=None):
                return OrderResult(retcode=Retcode.DONE)

            def order_delete(self, t):
                return OrderResult(retcode=Retcode.DONE)

            def positions(self, s=None, m=None):
                return []

            def orders(self, s=None, m=None):
                return []

            def account(self):
                return _sample_account_info()

            def symbol_info(self, s):
                return _sample_symbol_info()

            def server_time(self):
                return 0

        fb = FullBroker()
        self.assertIsInstance(fb, Broker)

    def test_each_method_is_callable(self):
        for name in self.REQUIRED_METHODS:
            self.assertTrue(hasattr(Broker, name), f"missing method: {name}")
            self.assertTrue(callable(getattr(Broker, name)), f"{name} is not callable")


class TestDealType(unittest.TestCase):
    """DealType enum for on_trade callback."""

    def test_variants(self):
        from app.sdk import DealType
        for v in [DealType.ENTRY, DealType.EXIT, DealType.MODIFY, DealType.PARTIAL_CLOSE]:
            self.assertIsNotNone(v)


class TestAccountMode(unittest.TestCase):
    """Section 10: Account modes."""

    def test_netting_and_hedging(self):
        self.assertIsNotNone(AccountMode.NETTING)
        self.assertIsNotNone(AccountMode.HEDGING)


# ── Public re-exports ──────────────────────────────────────────────────


class TestPublicAPI(unittest.TestCase):
    """Section 2: All types must be re-exported from app.sdk."""

    EXPECTED = [
        "AccountInfo", "AccountMode", "Broker", "Context", "DealType",
        "Indicators", "OrderRequest", "OrderResult", "OrderType",
        "PendingOrder", "Position", "PositionSide", "Retcode",
        "RuntimeContext", "Series", "StrategyBase", "StrategyRuntime",
        "SymbolInfo", "TypeFilling",
    ]

    def test_all_re_exported(self):
        for name in self.EXPECTED:
            self.assertTrue(hasattr(app.sdk, name), f"missing re-export: {name}")

    def test_all_in___all__(self):
        for name in self.EXPECTED:
            self.assertIn(name, app.sdk.__all__, f"missing from __all__: {name}")

    def test_strategy_base_accessible(self):
        self.assertTrue(issubclass(StrategyBase, object))


# ── Anti-float rule (CLAUDE.md) ────────────────────────────────────────


class TestAntiFloat(unittest.TestCase):
    """Section 12: No float annotations in financial data modules."""

    FINANCIAL_MODULES = ["types.py", "account.py", "symbol.py", "broker.py"]

    def test_no_float_annotation_in_financial_modules(self):
        sdk_dir = os.path.join(os.path.dirname(__file__), "..", "app", "sdk")
        for mod_name in self.FINANCIAL_MODULES:
            path = os.path.join(sdk_dir, mod_name)
            with open(path) as f:
                tree = ast.parse(f.read())
            for node in ast.walk(tree):
                if isinstance(node, ast.AnnAssign) and node.annotation:
                    if isinstance(node.annotation, ast.Name) and node.annotation.id == "float":
                        self.fail(f"float annotation in {mod_name}: line {node.lineno}")


if __name__ == "__main__":
    unittest.main()
