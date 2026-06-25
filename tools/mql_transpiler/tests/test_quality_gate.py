"""T1 — Quality gate tests (parser-independent).

Validates that the gates:
  1. Pass valid SDK code (like the 5 hand-written references).
  2. Fail code with MQL // comments.
  3. Fail code with bare MQL function names.
  4. Fail code with Python syntax errors.
  5. Catch the current regression: "high confidence illegal code" is judged LOW.

These tests are PARSER-INDEPENDENT — they exercise the quality gates against
known-good and known-bad Python code strings, not against any specific transpiler.
"""

import ast as _ast
import os
import re
import unittest

# Tree-sitter may not be available in CI.
try:
    from tools.mql_transpiler.tree_sitter_parser import available as _ts_available
    _TREE_SITTER_OK = _ts_available()
except Exception:
    _TREE_SITTER_OK = False

from tools.mql_transpiler.quality_gate import (
    GateFailure,
    QualityGate,
    QualityReport,
    QualityVerdict,
    confidence_from_output,
)


# ── Helpers ──────────────────────────────────────────────────────────────

def _read_sample(name: str) -> str:
    """Read a hand-written SDK reference strategy."""
    sdk_dir = os.path.join(
        os.path.dirname(__file__), "..", "..", "..",
        "strategy-service", "tests", "sdk_samples",
    )
    with open(os.path.join(sdk_dir, name)) as f:
        return f.read()


def _read_fixture(name: str) -> str:
    """Read an MQL fixture."""
    fixtures_dir = os.path.join(os.path.dirname(__file__), "fixtures")
    with open(os.path.join(fixtures_dir, name)) as f:
        return f.read()


# Fixture → SDK sample mapping.
FIXTURE_TO_SAMPLE = {
    "simple_ma_cross.mq4": "single_ma_cross.py",
    "grid_trader.mq4": "grid_trader.py",
    "martingale.mq4": "martingale.py",
    "hedge_twins.mq4": "hedge_twins.py",
    "custom_signal.mq4": "custom_signal.py",
}


# ── Gate 1: Compile ──────────────────────────────────────────────────────

class TestCompileGate(unittest.TestCase):
    """Gate 1: ast.parse() must succeed."""

    def test_valid_python_passes(self):
        """Hand-written SDK reference code must pass the compile gate."""
        for mql_fixture, sample_file in FIXTURE_TO_SAMPLE.items():
            code = _read_sample(sample_file)
            with self.subTest(sample=sample_file):
                report = QualityGate.assess(code)
                self.assertTrue(
                    report.compile_ok,
                    f"{sample_file}: compile gate failed but should pass\n"
                    f"Failures: {[f.message for f in report.failures]}",
                )

    def test_empty_output_fails(self):
        report = QualityGate.assess("")
        self.assertFalse(report.compile_ok)
        self.assertEqual(report.verdict, QualityVerdict.LOW)

    def test_syntax_error_fails(self):
        report = QualityGate.assess("def broken(:\n    pass\n")
        self.assertFalse(report.compile_ok)
        self.assertEqual(report.verdict, QualityVerdict.LOW)
        self.assertTrue(any("SyntaxError" in f.message for f in report.failures))

    def test_mql_comment_in_output_fails_compile(self):
        """// comments are valid MQL but invalid Python."""
        # A line like: // This is an MQL comment
        code = 'class Test(StrategyBase):\n    def on_init(self):\n        // MQL comment\n        pass\n'
        report = QualityGate.assess(code)
        # This should fail compile (// is not a Python comment).
        self.assertFalse(report.compile_ok)

    def test_incomplete_code_fails(self):
        report = QualityGate.assess("class Foo(StrategyBase):\n    def on_init(self):")
        self.assertFalse(report.compile_ok)


# ── Gate 2: SDK imports ──────────────────────────────────────────────────

class TestSDKImportGate(unittest.TestCase):
    """Gate 2: required SDK imports must be present."""

    def test_valid_samples_pass(self):
        for mql_fixture, sample_file in FIXTURE_TO_SAMPLE.items():
            code = _read_sample(sample_file)
            with self.subTest(sample=sample_file):
                report = QualityGate.assess(code)
                self.assertTrue(
                    report.sdk_import_ok,
                    f"{sample_file}: SDK import gate failed\n"
                    f"Failures: {[f.message for f in report.failures]}",
                )

    def test_missing_sdk_import_fails(self):
        code = 'class Test:\n    def on_init(self):\n        pass\n'
        report = QualityGate.assess(code)
        self.assertFalse(report.sdk_import_ok)

    def test_missing_decimal_import_fails(self):
        code = (
            'from app.sdk import StrategyBase\n'
            'class Test(StrategyBase):\n'
            '    def on_init(self):\n'
            '        pass\n'
        )
        report = QualityGate.assess(code)
        self.assertFalse(report.sdk_import_ok)

    def test_missing_strategybase_fails(self):
        code = (
            'from app.sdk import OrderType\n'
            'from decimal import Decimal\n'
            'class Test:\n'
            '    def on_init(self):\n'
            '        pass\n'
        )
        report = QualityGate.assess(code)
        self.assertFalse(report.sdk_import_ok)


# ── Gate 3: Lint ─────────────────────────────────────────────────────────

class TestLintGate(unittest.TestCase):
    """Gate 3: no MQL artifacts in Python output."""

    def test_valid_samples_pass(self):
        for mql_fixture, sample_file in FIXTURE_TO_SAMPLE.items():
            code = _read_sample(sample_file)
            with self.subTest(sample=sample_file):
                report = QualityGate.assess(code)
                self.assertTrue(
                    report.lint_ok,
                    f"{sample_file}: lint gate failed\n"
                    f"Failures: {[f.message for f in report.failures]}",
                )

    def test_bare_ordersend_fails(self):
        code = (
            'from app.sdk import StrategyBase\n'
            'from decimal import Decimal\n'
            'class Test(StrategyBase):\n'
            '    def on_init(self):\n'
            '        OrderSend("EURUSD", OP_BUY, 0.1, Ask, 3, 0, 0)\n'
        )
        report = QualityGate.assess(code)
        self.assertFalse(report.lint_ok)
        self.assertTrue(any("OrderSend" in f.message for f in report.failures))

    def test_bare_ima_fails(self):
        code = (
            'from app.sdk import StrategyBase\n'
            'from decimal import Decimal\n'
            'class Test(StrategyBase):\n'
            '    def on_init(self):\n'
            '        ma = iMA(Symbol(), 0, 14, 0, MODE_EMA, PRICE_CLOSE)\n'
        )
        report = QualityGate.assess(code)
        self.assertFalse(report.lint_ok)

    def test_normalizedouble_fails(self):
        code = (
            'from app.sdk import StrategyBase\n'
            'from decimal import Decimal\n'
            'class Test(StrategyBase):\n'
            '    def on_init(self):\n'
            '        price = NormalizeDouble(Ask, 4)\n'
        )
        report = QualityGate.assess(code)
        self.assertFalse(report.lint_ok)
        self.assertTrue(any("NormalizeDouble" in f.message for f in report.failures))


# ── Integrated: all 5 reference samples pass ─────────────────────────────

class TestAllReferenceSamplesPass(unittest.TestCase):
    """The 5 hand-written SDK reference samples MUST pass all gates.
    These are the golden oracle strategies — if they don't pass, the gates
    are too strict."""

    def test_all_five_pass_all_gates(self):
        for mql_fixture, sample_file in FIXTURE_TO_SAMPLE.items():
            with self.subTest(sample=sample_file):
                code = _read_sample(sample_file)
                report = QualityGate.assess(code)
                self.assertEqual(
                    report.verdict, QualityVerdict.HIGH,
                    f"{sample_file}: expected HIGH verdict\n"
                    f"  Compile: {report.compile_ok}\n"
                    f"  SDK import: {report.sdk_import_ok}\n"
                    f"  Lint: {report.lint_ok}\n"
                    f"  Failures: {[f.message for f in report.failures]}\n"
                    f"  Warnings: {report.warnings}",
                )


# ── Regression: current transpiler output is caught ──────────────────────

@unittest.skipUnless(_TREE_SITTER_OK, "tree-sitter MQL grammar not available")
class TestCurrentTranspilerOutputValidated(unittest.TestCase):
    """The current (fixed) transpiler output should pass quality gates.

    This replaces the old regression test that asserted the broken transpiler
    would be caught.  Now that the transpiler is fixed (T2+T3), all fixture
    outputs should pass.  The regression protection is now in
    TestConfidenceGateCatchesIllegalCode (test_transpiler.py) — it verifies
    that intentionally broken code is still caught.
    """

    def _get_current_transpiler_output(self, fixture_name: str) -> str:
        source = _read_fixture(fixture_name)
        try:
            from tools.mql_transpiler.ast_bridge import parse_mql
            from tools.mql_transpiler.ast_transpiler import ASTTranspiler
            ast_root = parse_mql(source)
            tp = ASTTranspiler("TestEA")
            result = tp._transpile_ast(ast_root)
            return result.output
        except Exception:
            return ""

    def test_current_output_passes_gates(self):
        """All 5 fixtures must produce gate-passing output (T2+T3 success)."""
        all_pass = True
        failures = []
        for fixture_name in FIXTURE_TO_SAMPLE:
            output = self._get_current_transpiler_output(fixture_name)
            if not output:
                failures.append(f"{fixture_name}: empty output")
                all_pass = False
                continue
            report = QualityGate.assess(output)
            if report.verdict != QualityVerdict.HIGH:
                failures.append(
                    f"{fixture_name}: {[f.message for f in report.failures]}"
                )
                all_pass = False

        self.assertTrue(
            all_pass,
            f"Expected all fixtures to pass gates, but some failed:\n"
            + "\n".join(failures),
        )

    def test_current_confidence_is_redefined(self):
        """Verify the new confidence function returns verdict, not float."""
        verdict = confidence_from_output("")
        self.assertIsInstance(verdict, QualityVerdict)
        self.assertEqual(verdict, QualityVerdict.LOW)


# ── Confidence redefinition ──────────────────────────────────────────────

class TestConfidenceRedefined(unittest.TestCase):
    """confidence is now HIGH/LOW, not a float."""

    def test_valid_code_is_high(self):
        code = _read_sample("single_ma_cross.py")
        self.assertEqual(confidence_from_output(code), QualityVerdict.HIGH)

    def test_empty_is_low(self):
        self.assertEqual(confidence_from_output(""), QualityVerdict.LOW)

    def test_syntax_error_is_low(self):
        self.assertEqual(
            confidence_from_output("this is not python {{{"),
            QualityVerdict.LOW,
        )

    def test_no_matched_by_total_algorithm(self):
        """The old matched/(matched+gaps) algorithm must not exist.
        confidence is a QualityVerdict, not a float."""
        verdict = confidence_from_output("valid_code = 1  # not really but ok")
        self.assertIsInstance(verdict, QualityVerdict)
        with self.assertRaises(TypeError):
            # It's an enum, not a float — can't do float comparison.
            _ = verdict > 0.5  # type: ignore


if __name__ == "__main__":
    unittest.main()
