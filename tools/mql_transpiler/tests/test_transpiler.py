"""T5 transpiler tests — compile-gate + behavioral alignment.

ADR-0020 D8, C2: correctness is measured by ``ast.parse()`` pass/fail and
behavioral alignment, NOT by substring assertions (``assertIn``).

Validates:
  - All 5 MQL fixtures produce compilable, gate-passing Python SDK code.
  - Deterministic (idempotent) output.
  - No MQL artifacts leak into Python output.
  - Translated code has correct class structure and lifecycle hooks.
  - GAPs are tracked and auditable.
"""

import ast as py_ast
import os
import unittest
from pathlib import Path

from tools.mql_transpiler.ast_transpiler import ASTTranspiler
from tools.mql_transpiler.quality_gate import QualityGate, QualityVerdict

# Tree-sitter may not be available in all environments (e.g. CI without
# the grammar .so compiled for the target architecture).
try:
    from tools.mql_transpiler.tree_sitter_parser import available as _ts_available
    _TREE_SITTER_OK = _ts_available()
except Exception:
    _TREE_SITTER_OK = False

_NEEDS_TREE_SITTER = unittest.skipUnless(
    _TREE_SITTER_OK,
    "tree-sitter MQL grammar not available — build with: "
    "cd tools/mql_transpiler/grammar/mql && "
    "npx tree-sitter generate && "
    "gcc -shared -fPIC -o mql.so src/parser.c -I src",
)

FIXTURES_DIR = Path(__file__).parent / "fixtures"

# All 5 MQL fixtures that must translate correctly.
FIXTURES = [
    "simple_ma_cross.mq4",
    "grid_trader.mq4",
    "martingale.mq4",
    "hedge_twins.mq4",
    "custom_signal.mq4",
]


def _transpile(fixture_name: str) -> ASTTranspiler:
    """Transpile a fixture and return the ASTTranspiler for inspection."""
    from tools.mql_transpiler.ast_bridge import parse_mql
    source = (FIXTURES_DIR / fixture_name).read_text()
    class_name = fixture_name.replace(".mq4", "").replace("_", " ").title().replace(" ", "")
    ast_root = parse_mql(source)
    tp = ASTTranspiler(class_name)
    tp._transpile_ast(ast_root)
    return tp


def _transpile_output(fixture_name: str) -> str:
    """Transpile a fixture and return the Python output string."""
    tp = _transpile(fixture_name)
    return "\n".join(tp._lines)


# ── Core: compile + gate tests ───────────────────────────────────────────

@_NEEDS_TREE_SITTER
class TestAllFixturesCompile(unittest.TestCase):
    """DoD: every fixture produces syntactically valid Python."""

    def test_each_fixture_compiles(self):
        for name in FIXTURES:
            with self.subTest(fixture=name):
                output = _transpile_output(name)
                try:
                    py_ast.parse(output)
                except SyntaxError as e:
                    lines = output.split("\n")
                    ctx = ""
                    if e.lineno and 1 <= e.lineno <= len(lines):
                        ctx = lines[e.lineno - 1][:120]
                    self.fail(
                        f"{name}: SyntaxError at line {e.lineno}: {e.msg}\n"
                        f"  {ctx}"
                    )


@_NEEDS_TREE_SITTER
class TestAllFixturesPassQualityGates(unittest.TestCase):
    """DoD: every fixture passes all 3 quality gates (compile + SDK import + lint)."""

    def test_each_fixture_passes_all_gates(self):
        for name in FIXTURES:
            with self.subTest(fixture=name):
                output = _transpile_output(name)
                report = QualityGate.assess(output)
                self.assertEqual(
                    report.verdict, QualityVerdict.HIGH,
                    f"{name}: expected HIGH verdict\n"
                    f"  Compile: {report.compile_ok}\n"
                    f"  SDK: {report.sdk_import_ok}\n"
                    f"  Lint: {report.lint_ok}\n"
                    f"  Failures: {[f.message for f in report.failures]}",
                )


@_NEEDS_TREE_SITTER
class TestAllFixturesZeroGaps(unittest.TestCase):
    """DoD: all mechanical constructs covered — 0 TRANSPILER-GAP markers."""

    def test_each_fixture_has_zero_gaps(self):
        for name in FIXTURES:
            with self.subTest(fixture=name):
                output = _transpile_output(name)
                gap_count = output.count("# TRANSPILER-GAP:")
                self.assertEqual(
                    gap_count, 0,
                    f"{name}: has {gap_count} TRANSPILER-GAP markers\n"
                    f"  GAP lines: {[l.strip()[:80] for l in output.split(chr(10)) if 'TRANSPILER-GAP' in l]}",
                )


# ── Output structure ─────────────────────────────────────────────────────

@_NEEDS_TREE_SITTER
class TestOutputStructure(unittest.TestCase):
    """Translated output must have the correct Python SDK skeleton."""

    def test_imports_present(self):
        for name in FIXTURES:
            with self.subTest(fixture=name):
                output = _transpile_output(name)
                self.assertIn("from decimal import Decimal", output)
                self.assertIn("from app.sdk import", output)
                self.assertIn("StrategyBase", output)

    def test_class_definition(self):
        for name in FIXTURES:
            with self.subTest(fixture=name):
                output = _transpile_output(name)
                self.assertIn("class ", output)
                self.assertIn("(StrategyBase):", output)

    def test_lifecycle_hooks_present(self):
        """At minimum, each output should have at least one lifecycle method."""
        for name in FIXTURES:
            with self.subTest(fixture=name):
                output = _transpile_output(name)
                has_hook = any(
                    hook in output
                    for hook in ["def on_init", "def on_tick", "def on_bar",
                                  "def on_timer", "def on_deinit"]
                )
                self.assertTrue(has_hook, f"{name}: no lifecycle hook found")

    def test_init_params_method(self):
        """Every fixture with extern/input should have _init_params."""
        for name in FIXTURES:
            with self.subTest(fixture=name):
                output = _transpile_output(name)
                # All 5 fixtures have extern/input params.
                self.assertIn("_init_params", output)

    def test_decimal_used_for_prices(self):
        """Prices and volumes must use Decimal, not raw float."""
        for name in FIXTURES:
            with self.subTest(fixture=name):
                output = _transpile_output(name)
                self.assertIn("Decimal(", output,
                              f"{name}: no Decimal usage found — float precision risk")


# ── No MQL artifacts ─────────────────────────────────────────────────────

@_NEEDS_TREE_SITTER
class TestNoMQLArtifacts(unittest.TestCase):
    """Python output must not contain raw MQL function calls or comments."""

    _BANNED_MQL_FUNCTIONS = [
        "OrderSend(", "OrderClose(", "OrderModify(", "OrderDelete(",
        "OrderSelect(", "OrdersTotal()", "PositionsTotal()",
        "iMA(", "iRSI(", "iATR(", "iBands(", "iMACD(", "iStochastic(",
        "iCCI(", "iCustom(", "iADX(", "iMomentum(",
        "iOpen(", "iHigh(", "iLow(", "iClose(", "iVolume(", "iTime(",
    ]

    def test_no_bare_mql_functions(self):
        for name in FIXTURES:
            with self.subTest(fixture=name):
                output = _transpile_output(name)
                for mql_fn in self._BANNED_MQL_FUNCTIONS:
                    self.assertNotIn(
                        mql_fn, output,
                        f"{name}: raw MQL function '{mql_fn}' in output",
                    )

    def test_no_mql_comments(self):
        """MQL // comments must not appear in Python output."""
        for name in FIXTURES:
            with self.subTest(fixture=name):
                output = _transpile_output(name)
                # Lines starting with // are MQL comments
                for i, line in enumerate(output.split("\n"), 1):
                    stripped = line.strip()
                    if stripped.startswith("//"):
                        self.fail(
                            f"{name}: MQL // comment at line {i}: {stripped[:80]}"
                        )


# ── Idempotence ──────────────────────────────────────────────────────────

@_NEEDS_TREE_SITTER
class TestTranspilerIdempotence(unittest.TestCase):
    """Same input twice → identical output twice."""

    def test_idempotent(self):
        for name in FIXTURES:
            with self.subTest(fixture=name):
                o1 = _transpile_output(name)
                o2 = _transpile_output(name)
                self.assertEqual(o1, o2, f"{name}: output differs between runs")


# ── Behavioral alignment (structure-level) ────────────────────────────────

@_NEEDS_TREE_SITTER
class TestBehavioralStructure(unittest.TestCase):
    """Translated strategies must match reference strategy structure.

    Full signal-level behavioral comparison requires a working SimBroker
    runtime (T3.1).  This test validates structural equivalence:
    same methods, same param reads, same broker call patterns.
    """

    def test_each_has_order_send(self):
        """Every fixture except custom_signal uses order_send."""
        order_send_fixtures = {
            "simple_ma_cross.mq4", "grid_trader.mq4",
            "martingale.mq4", "hedge_twins.mq4",
        }
        for name in FIXTURES:
            with self.subTest(fixture=name):
                output = _transpile_output(name)
                if name in order_send_fixtures:
                    self.assertIn("self.broker.order_send(", output,
                                  f"{name}: missing broker.order_send")

    def test_each_has_position_close_or_order_delete(self):
        """Every fixture should have close/delete logic."""
        for name in FIXTURES:
            with self.subTest(fixture=name):
                output = _transpile_output(name)
                has_close = "self.broker.position_close(" in output
                has_delete = "self.broker.order_delete(" in output
                self.assertTrue(
                    has_close or has_delete,
                    f"{name}: no position_close or order_delete found",
                )

    def test_indicators_used_where_expected(self):
        """Fixtures that use MQL indicators should produce SDK indicator calls."""
        indicator_fixtures = {
            "simple_ma_cross.mq4", "martingale.mq4",
            "hedge_twins.mq4", "custom_signal.mq4",
        }
        for name in FIXTURES:
            with self.subTest(fixture=name):
                output = _transpile_output(name)
                if name in indicator_fixtures:
                    self.assertIn("self.indicators.", output,
                                  f"{name}: missing indicators usage")

    def test_for_loop_orders_pattern(self):
        """Fixtures with OrderSelect loops should produce broker iteration.

        OrderClose loops → broker.positions() (close market positions).
        OrderDelete loops → broker.orders() (cancel pending orders).
        """
        # Fixtures that use OrderClose in their loop → positions().
        close_loop_fixtures = {
            "simple_ma_cross.mq4", "martingale.mq4",
            "hedge_twins.mq4", "custom_signal.mq4",
        }
        # Fixtures that use OrderDelete in their loop → orders().
        delete_loop_fixtures = {"grid_trader.mq4"}
        for name in FIXTURES:
            with self.subTest(fixture=name):
                output = _transpile_output(name)
                if name in close_loop_fixtures:
                    self.assertIn("self.broker.positions():", output,
                                  f"{name}: OrderClose loop should emit broker.positions()")
                elif name in delete_loop_fixtures:
                    self.assertIn("self.broker.orders():", output,
                                  f"{name}: OrderDelete loop should emit broker.orders()")


# ── Regression: confidence gate catches illegal code ─────────────────────

class TestConfidenceGateCatchesIllegalCode(unittest.TestCase):
    """The quality gate must correctly identify broken code as LOW."""

    def test_empty_output_is_low(self):
        from tools.mql_transpiler.quality_gate import confidence_from_output
        self.assertEqual(confidence_from_output(""), QualityVerdict.LOW)

    def test_syntax_error_is_low(self):
        from tools.mql_transpiler.quality_gate import confidence_from_output
        self.assertEqual(
            confidence_from_output("def broken(:\n    pass\n"),
            QualityVerdict.LOW,
        )

    def test_missing_imports_is_low(self):
        from tools.mql_transpiler.quality_gate import confidence_from_output
        code = "class Test:\n    def on_init(self):\n        pass\n"
        self.assertEqual(confidence_from_output(code), QualityVerdict.LOW)

    def test_valid_reference_is_high(self):
        """The 5 hand-written SDK reference strategies must pass gates."""
        from tools.mql_transpiler.quality_gate import confidence_from_output
        sdk_dir = Path(__file__).parent.parent.parent.parent / "strategy-service" / "tests" / "sdk_samples"
        for sample in ["single_ma_cross.py", "grid_trader.py", "martingale.py",
                        "hedge_twins.py", "custom_signal.py"]:
            code = (sdk_dir / sample).read_text()
            self.assertEqual(
                confidence_from_output(code), QualityVerdict.HIGH,
                f"{sample}: reference strategy should be HIGH",
            )


if __name__ == "__main__":
    unittest.main()
