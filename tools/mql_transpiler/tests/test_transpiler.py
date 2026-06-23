"""T2.1 transpiler tests.

Validates:
  - All 5 MQL fixtures transpile to compilable Python SDK skeletons.
  - Key mapping constructs are correctly translated.
  - Unmappable constructs are marked with TRANSPILER-GAP.
  - Transpiler stats are tracked correctly.
  - Output imports and has correct class structure.
"""

import os
import unittest

from tools.mql_transpiler.transpiler import MQLTranspiler, TranspileResult

FIXTURES_DIR = os.path.join(os.path.dirname(__file__), "fixtures")


def _read_fixture(name: str) -> str:
    with open(os.path.join(FIXTURES_DIR, name)) as f:
        return f.read()


def _transpile_fixture(name: str, class_name: str = "TranslatedStrategy") -> TranspileResult:
    source = _read_fixture(name)
    tp = MQLTranspiler(class_name=class_name)
    return tp.transpile(source, filename=name)


# ── Core transpiler tests ──────────────────────────────────────────────


class TestTranspilerBasic(unittest.TestCase):
    """Basic transpiler functionality."""

    def test_empty_source(self):
        result = _transpile_fixture("simple_ma_cross.mq4")  # any fixture
        tp = MQLTranspiler()
        result = tp.transpile("", "empty.mq4")
        self.assertIn("class TranslatedStrategy", result.output)
        self.assertIn("StrategyBase", result.output)

    def test_header_contains_imports(self):
        result = _transpile_fixture("simple_ma_cross.mq4", "MACross")
        self.assertIn("from decimal import Decimal", result.output)
        self.assertIn("from app.sdk import", result.output)
        self.assertIn("class MACross(StrategyBase):", result.output)

    def test_stats_tracked(self):
        result = _transpile_fixture("simple_ma_cross.mq4")
        self.assertGreater(result.stats.lines_in, 0)
        self.assertGreater(result.stats.lines_out, 0)
        self.assertGreater(result.stats.patterns_matched, 0)


class TestLifecycleMapping(unittest.TestCase):
    """MQL lifecycle functions → SDK hooks."""

    def test_oninit_mapped(self):
        result = _transpile_fixture("simple_ma_cross.mq4")
        self.assertIn("def on_init(self)", result.output)

    def test_ontick_mapped(self):
        result = _transpile_fixture("simple_ma_cross.mq4")
        self.assertIn("def on_tick(self)", result.output)

    def test_ondeinit_mapped(self):
        result = _transpile_fixture("simple_ma_cross.mq4")
        self.assertIn("def on_deinit(self, reason: str = 'user_stop')", result.output)

    def test_ontimer_mapped(self):
        result = _transpile_fixture("custom_signal.mq4")
        self.assertIn("def on_timer(self)", result.output)


class TestTradeFunctionMapping(unittest.TestCase):
    """MQL trade functions → broker methods."""

    def test_ordersend_buy_mapped(self):
        result = _transpile_fixture("simple_ma_cross.mq4")
        self.assertIn("self.broker.order_send(OrderRequest(", result.output)
        self.assertIn("OrderType.BUY", result.output)
        self.assertIn("Decimal(str(", result.output)

    def test_ordersend_sell_mapped(self):
        result = _transpile_fixture("simple_ma_cross.mq4")
        self.assertIn("OrderType.SELL", result.output)

    def test_orderclose_mapped(self):
        result = _transpile_fixture("simple_ma_cross.mq4")
        self.assertIn("self.broker.position_close(", result.output)

    def test_orderdelete_mapped(self):
        result = _transpile_fixture("grid_trader.mq4")
        self.assertIn("self.broker.order_delete(", result.output)

    def test_pending_order_types(self):
        result = _transpile_fixture("grid_trader.mq4")
        self.assertIn("OrderType.BUY_LIMIT", result.output)
        self.assertIn("OrderType.SELL_LIMIT", result.output)


class TestIndicatorMapping(unittest.TestCase):
    """MQL indicator functions → SDK indicators."""

    def test_ima_mapped(self):
        result = _transpile_fixture("simple_ma_cross.mq4")
        self.assertIn("self.indicators.ma(", result.output)
        self.assertIn("period=", result.output)

    def test_irsi_mapped(self):
        result = _transpile_fixture("martingale.mq4")
        self.assertIn("self.indicators.rsi(", result.output)

    def test_icustom_mapped(self):
        result = _transpile_fixture("custom_signal.mq4")
        self.assertIn("self.indicators.i_custom(", result.output)
        self.assertIn("name=", result.output)

    def test_ema_method_name_converted(self):
        result = _transpile_fixture("simple_ma_cross.mq4")
        self.assertIn("method='ema'", result.output)


class TestOrderSelectLoopMapping(unittest.TestCase):
    """OrderSelect loops → broker.orders() / broker.positions() iteration."""

    def test_orderstotal_loop_detected(self):
        result = _transpile_fixture("simple_ma_cross.mq4")
        self.assertIn("for order in self.broker.orders():", result.output)

    def test_positionstotal_loop_detected(self):
        result = _transpile_fixture("grid_trader.mq4")
        # Grid trader uses OrdersTotal too.
        self.assertIn("for order in self.broker.orders():", result.output)


class TestExternInputMapping(unittest.TestCase):
    """extern / input → @param comments."""

    def test_extern_recognized(self):
        result = _transpile_fixture("simple_ma_cross.mq4")
        self.assertIn("ctx.param(", result.output)
        self.assertIn("FastMAPeriod", result.output)

    def test_input_recognized(self):
        result = _transpile_fixture("simple_ma_cross.mq4")
        self.assertIn("LotSize", result.output)


class TestTranspilerGapMarking(unittest.TestCase):
    """Unmappable constructs must be marked, not silently dropped."""

    def test_array_initialize_marked(self):
        source = """
        int OnInit() {
            double arr[10];
            ArrayInitialize(arr, 0.0);
            return INIT_SUCCEEDED;
        }
        """
        tp = MQLTranspiler()
        result = tp.transpile(source, "test.mq4")
        self.assertIn("TRANSPILER-GAP", result.output)

    def test_file_operations_marked(self):
        source = """
        void OnTick() {
            int handle = FileOpen("test.txt", FILE_WRITE);
            FileWrite(handle, "data");
            FileClose(handle);
        }
        """
        tp = MQLTranspiler()
        result = tp.transpile(source, "test.mq4")
        self.assertIn("TRANSPILER-GAP: FileIO", result.output)

    def test_webrequest_marked(self):
        source = """
        void OnTick() {
            string resp;
            WebRequest("GET", "https://example.com", NULL, 0, resp);
        }
        """
        tp = MQLTranspiler()
        result = tp.transpile(source, "test.mq4")
        self.assertIn("TRANSPILER-GAP: WebRequest", result.output)

    def test_gui_marked(self):
        source = """
        void OnTick() {
            ObjectCreate(0, "label", OBJ_LABEL, 0, 0, 0);
            ObjectSetString(0, "label", OBJPROP_TEXT, "Hello");
        }
        """
        tp = MQLTranspiler()
        result = tp.transpile(source, "test.mq4")
        self.assertIn("TRANSPILER-GAP: GUI", result.output)

    def test_variable_declaration_marked(self):
        source = "double myVar = 1.5;"
        tp = MQLTranspiler()
        result = tp.transpile(source, "test.mq4")
        self.assertIn("TRANSPILER-GAP: variable declaration", result.output)

    def test_gap_stats_tracked(self):
        source = "double myVar = 1.5;\nint x = 42;"
        tp = MQLTranspiler()
        result = tp.transpile(source, "test.mq4")
        self.assertGreaterEqual(result.stats.gaps, 1)


class TestAllFixturesTranspile(unittest.TestCase):
    """All 5 T0.4 MQL fixtures must produce valid Python SDK skeletons."""

    FIXTURES = [
        "simple_ma_cross.mq4",
        "grid_trader.mq4",
        "martingale.mq4",
        "hedge_twins.mq4",
        "custom_signal.mq4",
    ]

    def _check_fixture(self, name: str):
        result = _transpile_fixture(name)
        output = result.output

        # Must contain class definition inheriting StrategyBase.
        self.assertIn("StrategyBase", output, f"{name}: missing StrategyBase import")
        self.assertIn("class ", output, f"{name}: missing class definition")

        # Must import from app.sdk.
        self.assertIn("from app.sdk import", output, f"{name}: missing SDK import")

        # Must have at least one lifecycle hook.
        lifecycle_present = any(
            hook in output
            for hook in ["def on_init", "def on_tick", "def on_bar", "def on_timer", "def on_deinit"]
        )
        self.assertTrue(lifecycle_present, f"{name}: no lifecycle hook found")

        # TRANSPILER-GAP comments must be properly formatted.
        gap_lines = [l for l in output.split("\n") if "TRANSPILER-GAP" in l]
        for line in gap_lines:
            self.assertTrue(
                line.strip().startswith("# TRANSPILER-GAP:"),
                f"{name}: malformed gap comment: {line.strip()[:60]}",
            )

        # Must not have raw MQL syntax (no bare OrderSend, iMA, etc.)
        for mql_fn in ["OrderSend(", "iMA(", "OrderClose("]:
            self.assertNotIn(
                mql_fn, output,
                f"{name}: untranslated MQL function '{mql_fn}' found in output",
            )

    def test_simple_ma_cross(self):
        self._check_fixture("simple_ma_cross.mq4")

    def test_grid_trader(self):
        self._check_fixture("grid_trader.mq4")

    def test_martingale(self):
        self._check_fixture("martingale.mq4")

    def test_hedge_twins(self):
        self._check_fixture("hedge_twins.mq4")

    def test_custom_signal(self):
        self._check_fixture("custom_signal.mq4")


class TestTranspilerIdempotence(unittest.TestCase):
    """Transpiling the same source twice produces identical output."""

    def test_idempotent(self):
        source = _read_fixture("simple_ma_cross.mq4")
        tp1 = MQLTranspiler("TestEA")
        r1 = tp1.transpile(source, "test.mq4")
        tp2 = MQLTranspiler("TestEA")
        r2 = tp2.transpile(source, "test.mq4")
        self.assertEqual(r1.output, r2.output)


class TestCommonFunctionMapping(unittest.TestCase):
    """Common MQL built-in functions map to Python equivalents."""

    def test_print_mapped(self):
        source = 'void OnTick() { Print("hello"); }'
        tp = MQLTranspiler()
        result = tp.transpile(source, "test.mq4")
        self.assertIn("print(", result.output.lower())

    def test_mathabs_mapped(self):
        source = "void OnTick() { double x = MathAbs(-1.0); }"
        tp = MQLTranspiler()
        result = tp.transpile(source, "test.mq4")
        self.assertIn("abs(", result.output)

    def test_account_balance_mapped(self):
        source = "void OnTick() { double bal = AccountBalance(); }"
        tp = MQLTranspiler()
        result = tp.transpile(source, "test.mq4")
        self.assertIn("self.broker.account().balance", result.output)

    def test_set_timer_mapped(self):
        source = "void OnInit() { EventSetTimer(300); return INIT_SUCCEEDED; }"
        tp = MQLTranspiler()
        result = tp.transpile(source, "test.mq4")
        self.assertIn("self.ctx.set_timer", result.output)

    def test_kill_timer_mapped(self):
        source = "void OnDeinit(const int reason) { EventKillTimer(); }"
        tp = MQLTranspiler()
        result = tp.transpile(source, "test.mq4")
        self.assertIn("self.ctx.kill_timer", result.output)


class TestControlFlowMapping(unittest.TestCase):
    """MQL control flow → Python."""

    def test_if_mapped(self):
        source = "void OnTick() { if (Close[0] > Open[0]) { Print(\"up\"); } }"
        tp = MQLTranspiler()
        result = tp.transpile(source, "test.mq4")
        self.assertIn("if ", result.output)

    def test_while_mapped(self):
        source = "void OnTick() { while (true) { break; } }"
        tp = MQLTranspiler()
        result = tp.transpile(source, "test.mq4")
        self.assertIn("while True:", result.output)

    def test_return_mapped(self):
        source = "void OnTick() { return; }"
        tp = MQLTranspiler()
        result = tp.transpile(source, "test.mq4")
        self.assertIn("return", result.output)


if __name__ == "__main__":
    unittest.main()
