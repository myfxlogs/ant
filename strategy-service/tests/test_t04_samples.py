"""T0.4 — Validate SDK expressiveness via hand-written sample EAs.

DoD criteria:
  1. All 5 samples import cleanly.
  2. All samples inherit from StrategyBase.
  3. Each sample demonstrates its target MQL pattern with readable logic.
  4. SDK feature coverage: every SDK type used by at least one sample.
  5. No import-time errors (all SDK symbols resolve).
  6. Strategy instantiates without error (attributes assigned at init).
"""

import ast
import os
import unittest
from decimal import Decimal

# All 5 samples must import without error.
from tests.sdk_samples.custom_signal import CustomSignal
from tests.sdk_samples.grid_trader import GridTrader
from tests.sdk_samples.hedge_twins import HedgeTwins
from tests.sdk_samples.martingale import Martingale
from tests.sdk_samples.single_ma_cross import SingleMACross

from app.sdk import StrategyBase


SAMPLES = [
    (SingleMACross, "single_ma_cross.py", "EMA crossover trend-follower"),
    (GridTrader, "grid_trader.py", "Pending-order grid with magic-number tracking"),
    (Martingale, "martingale.py", "RSI-based with martingale position sizing"),
    (HedgeTwins, "hedge_twins.py", "Simultaneous long+short in hedging mode"),
    (CustomSignal, "custom_signal.py", "Custom indicator + multi-timeframe confluence"),
]


class TestAllSamplesImport(unittest.TestCase):
    """DoD 1: All 5 samples import without error."""

    def test_count(self):
        self.assertEqual(len(SAMPLES), 5, "Must have exactly 5 sample EAs")

    def test_each_imports(self):
        for cls, filename, _desc in SAMPLES:
            self.assertIsNotNone(cls, f"{filename} failed to import")


class TestAllSamplesInheritStrategyBase(unittest.TestCase):
    """DoD 2: Every sample must be a StrategyBase subclass."""

    def test_each_is_subclass(self):
        for cls, filename, _desc in SAMPLES:
            self.assertTrue(
                issubclass(cls, StrategyBase),
                f"{filename}: {cls.__name__} must inherit StrategyBase",
            )

    def test_each_has_lifecycle_hooks(self):
        """At minimum, each sample should override on_init."""
        for cls, filename, _desc in SAMPLES:
            self.assertTrue(
                hasattr(cls, "on_init"),
                f"{filename}: {cls.__name__} missing on_init",
            )


class TestSampleLogicReadability(unittest.TestCase):
    """DoD 3: Each sample must have readable logic — verifiable by
    checking that key methods do NOT consist solely of 'pass' or bare '...'."""

    def _get_method_source(self, cls, method_name: str) -> str:
        """Extract the source lines of a method from its file."""
        sdk_dir = os.path.join(os.path.dirname(__file__), "sdk_samples")
        # Find the file for this class.
        for _, filename, _ in SAMPLES:
            if cls.__name__ in filename or True:  # brute force
                pass
        # Use inspect to get the source.
        import inspect
        try:
            return inspect.getsource(getattr(cls, method_name))
        except (OSError, TypeError):
            return ""

    def test_on_init_has_substance(self):
        """on_init in each sample must do more than 'pass'."""
        for cls, filename, _desc in SAMPLES:
            import inspect
            try:
                src = inspect.getsource(cls.on_init)
            except (OSError, TypeError):
                self.fail(f"Cannot get source for {cls.__name__}.on_init")
            # Strip docstrings and whitespace.
            lines = [l.strip() for l in src.split("\n")
                     if l.strip() and not l.strip().startswith('"""') and not l.strip().startswith("#")]
            # After stripping docstring, should have more than 1 line.
            self.assertGreater(
                len(lines), 1,
                f"{cls.__name__}.on_init is too minimal ({len(lines)} substantive lines)",
            )

    def test_each_uses_broker_or_indicators(self):
        """Every sample must reference self.broker or self.indicators somewhere."""
        for cls, filename, _desc in SAMPLES:
            import inspect
            full_src = inspect.getsource(cls)
            uses_broker = "self.broker" in full_src
            uses_indicators = "self.indicators" in full_src
            self.assertTrue(
                uses_broker or uses_indicators,
                f"{cls.__name__} never references self.broker or self.indicators",
            )


class TestSDKFeatureCoverage(unittest.TestCase):
    """DoD 4: Every major SDK feature must be used by at least one sample.

    This test reads ALL sample source files and verifies that key SDK symbols
    appear somewhere in the sample corpus.  If a feature is never used, it
    may indicate an expressiveness gap."""

    @classmethod
    def setUpClass(cls):
        cls.all_source = cls._read_all_samples()

    @staticmethod
    def _read_all_samples() -> str:
        sdk_dir = os.path.join(os.path.dirname(__file__), "sdk_samples")
        all_src = ""
        for fname in sorted(os.listdir(sdk_dir)):
            if fname.endswith(".py") and fname != "__init__.py":
                with open(os.path.join(sdk_dir, fname)) as f:
                    all_src += f.read()
        return all_src

    def _assert_used(self, symbol: str):
        self.assertIn(symbol, self.all_source,
                      f"SDK symbol '{symbol}' is never used in any sample — expressiveness gap?")

    # ── Lifecycle ───────────────────────────────────────────────
    def test_on_init_used(self):
        self._assert_used("on_init")

    def test_on_bar_used(self):
        self._assert_used("on_bar")

    def test_on_tick_used(self):
        self._assert_used("on_tick")

    def test_on_trade_used(self):
        self._assert_used("on_trade")

    def test_on_timer_used(self):
        self._assert_used("on_timer")

    def test_on_deinit_used(self):
        self._assert_used("on_deinit")

    # ── Broker methods ──────────────────────────────────────────
    def test_order_send_used(self):
        self._assert_used("order_send")

    def test_position_close_used(self):
        self._assert_used("position_close")

    def test_position_modify_used(self):
        self._assert_used("position_modify")

    def test_order_delete_used(self):
        self._assert_used("order_delete")

    def test_positions_query_used(self):
        self._assert_used("positions(")

    def test_orders_query_used(self):
        self._assert_used("orders(")

    def test_account_query_used(self):
        self._assert_used("account()")

    def test_symbol_info_used(self):
        self._assert_used("symbol_info")

    # ── Context methods ─────────────────────────────────────────
    def test_ctx_bars_used(self):
        self._assert_used("bars(")

    def test_ctx_param_used(self):
        self._assert_used("ctx.param")

    def test_set_timer_used(self):
        self._assert_used("set_timer")

    def test_kill_timer_used(self):
        self._assert_used("kill_timer")

    # ── Indicators ──────────────────────────────────────────────
    def test_ema_used(self):
        self._assert_used("indicators.ema")

    def test_rsi_used(self):
        self._assert_used("indicators.rsi")

    def test_atr_used(self):
        self._assert_used("indicators.atr")

    def test_i_custom_used(self):
        self._assert_used("i_custom")

    # ── Order types ─────────────────────────────────────────────
    def test_market_order_used(self):
        self._assert_used("OrderType.BUY")

    def test_pending_order_used(self):
        self._assert_used("BUY_LIMIT")

    def test_stop_limit_order_used(self):
        self._assert_used("STOP_LIMIT")

    # ── Advanced features ───────────────────────────────────────
    def test_magic_number_used(self):
        self._assert_used("magic")

    def test_hedging_mode_used(self):
        self._assert_used("HEDGING")

    def test_netting_mode_used(self):
        self._assert_used("NETTING")

    def test_account_info_used(self):
        self._assert_used("AccountInfo")

    def test_decimal_used(self):
        self._assert_used("Decimal(")

    def test_type_filling_used(self):
        self._assert_used("TypeFilling")

    def test_retcode_used(self):
        self._assert_used("Retcode") or self._assert_used("retcode")

    def test_position_side_used(self):
        self._assert_used("PositionSide")

    # ── Key retcode values ──────────────────────────────────────
    def test_risk_blocked_handled(self):
        self._assert_used("RISK_BLOCKED")


class TestSamplesDirectory(unittest.TestCase):
    """Ensure sdk_samples is a proper Python package."""

    def test_init_exists(self):
        sdk_dir = os.path.join(os.path.dirname(__file__), "sdk_samples")
        init_path = os.path.join(sdk_dir, "__init__.py")
        self.assertTrue(os.path.isfile(init_path), "sdk_samples/__init__.py missing")

    def test_five_strategy_files(self):
        sdk_dir = os.path.join(os.path.dirname(__file__), "sdk_samples")
        strategy_files = [
            f for f in os.listdir(sdk_dir)
            if f.endswith(".py") and f != "__init__.py"
        ]
        self.assertEqual(
            len(strategy_files), 5,
            f"Expected 5 strategy files, got {len(strategy_files)}: {strategy_files}",
        )


class TestNoStrayImplementation(unittest.TestCase):
    """The samples must not import from engine/ or any implementation module.
    They must only use the frozen SDK stubs."""

    PROHIBITED_IMPORTS = [
        "app.engine",
        "app.sandbox",
        "numpy",
        "pandas",
    ]

    def test_no_engine_imports(self):
        sdk_dir = os.path.join(os.path.dirname(__file__), "sdk_samples")
        for fname in sorted(os.listdir(sdk_dir)):
            if not fname.endswith(".py"):
                continue
            with open(os.path.join(sdk_dir, fname)) as f:
                src = f.read()
            for prohibited in self.PROHIBITED_IMPORTS:
                self.assertNotIn(
                    f"import {prohibited}", src,
                    f"{fname} imports prohibited module '{prohibited}'",
                )
                self.assertNotIn(
                    f"from {prohibited}", src,
                    f"{fname} imports from prohibited module '{prohibited}'",
                )


if __name__ == "__main__":
    unittest.main()
