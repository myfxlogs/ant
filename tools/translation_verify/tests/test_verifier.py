"""T2.3 verification harness tests."""

import json
import os
import unittest

from tools.translation_verify.coverage import (
    CoverageReport,
    generate_coverage_report,
    scan_mql_features,
    scan_transpiler_gaps,
)
from tools.translation_verify.verifier import TranslationVerifier, VerificationResult

# Tree-sitter may not be available in CI.
try:
    from tools.mql_transpiler.tree_sitter_parser import available as _ts_available
    _TREE_SITTER_OK = _ts_available()
except Exception:
    _TREE_SITTER_OK = False

_NEEDS_TREE_SITTER = unittest.skipUnless(
    _TREE_SITTER_OK,
    "tree-sitter MQL grammar not available",
)

FIXTURES_DIR = os.path.join(
    os.path.dirname(__file__), "..", "..", "mql_transpiler", "tests", "fixtures"
)


def _read_fixture(name: str) -> str:
    with open(os.path.join(FIXTURES_DIR, name)) as f:
        return f.read()


# ── Coverage scanner tests ─────────────────────────────────────────────


class TestFeatureScanner(unittest.TestCase):
    """MQL feature detection."""

    def test_detects_lifecycle(self):
        source = "int OnInit() { return INIT_SUCCEEDED; }\nvoid OnTick() { }\nvoid OnDeinit(const int r) { }"
        found = scan_mql_features(source)
        self.assertIn("Lifecycle", found)
        self.assertGreaterEqual(found["Lifecycle"], 3)

    def test_detects_indicators(self):
        source = "double ma = iMA(NULL, 0, 14, 0, MODE_EMA, PRICE_CLOSE);\ndouble rsi = iRSI(NULL, 0, 14, PRICE_CLOSE, 0);"
        found = scan_mql_features(source)
        self.assertIn("Technical Indicators", found)
        self.assertGreaterEqual(found["Technical Indicators"], 2)

    def test_detects_unsupported_features(self):
        source = "int h = FileOpen('test.txt', FILE_WRITE);\nObjectCreate(0, 'l', OBJ_LABEL, 0, 0, 0);"
        found = scan_mql_features(source)
        self.assertIn("File I/O", found)
        self.assertIn("GUI Objects", found)

    def test_empty_source_no_features(self):
        found = scan_mql_features("")
        self.assertEqual(len(found), 0)


class TestGapScanner(unittest.TestCase):
    """TRANSPILER-GAP extraction."""

    def test_extracts_gaps(self):
        output = """
        # TRANSPILER-GAP: FileIO
        # TRANSPILER-GAP: GUI
        # TRANSPILER-GAP: Variable: double x
        print("ok")
        """
        gaps = scan_transpiler_gaps(output)
        self.assertIn("FileIO", gaps)
        self.assertIn("GUI", gaps)

    def test_no_gaps(self):
        gaps = scan_transpiler_gaps("print('hello')\n# comment")
        self.assertEqual(len(gaps), 0)


class TestCoverageReport(unittest.TestCase):
    """Full coverage report generation."""

    def test_generates_report(self):
        source = """
        int OnInit() { return INIT_SUCCEEDED; }
        void OnTick() {
            double ma = iMA(NULL, 0, 14, 0, MODE_EMA, PRICE_CLOSE);
            OrderSend(Symbol(), OP_BUY, 0.1, Ask, 3, 0, 0, "e", 1, 0, 0);
            FileWrite(1, "data");
        }
        void OnDeinit(const int reason) { }
        """
        output = """
        class TranslatedStrategy(StrategyBase):
            def on_init(self) -> None:
                pass
            def on_tick(self) -> None:
                self.indicators.ma(period=14, shift=0, method='ema')
                self.broker.order_send(OrderRequest(...))
                # TRANSPILER-GAP: FileIO
            def on_deinit(self, reason='user_stop') -> None:
                pass
        """
        report = generate_coverage_report(source, output, "test.mq4")
        self.assertIsInstance(report, CoverageReport)
        self.assertGreater(report.total_categories, 0)
        self.assertGreater(report.overall_pct, 0)

        # File I/O should be NONE.
        file_cov = next((c for c in report.categories if c.name == "File I/O"), None)
        self.assertIsNotNone(file_cov)
        self.assertEqual(file_cov.status, "NONE")

        # Lifecycle should be FULL.
        life_cov = next((c for c in report.categories if c.name == "Lifecycle"), None)
        self.assertIsNotNone(life_cov)
        self.assertEqual(life_cov.status, "FULL", f"Lifecycle status: {life_cov.status}")

    def test_empty_source(self):
        report = generate_coverage_report("", "class X(StrategyBase): pass", "empty.mq4")
        self.assertEqual(report.total_categories, 0)
        self.assertEqual(report.overall_pct, 0.0)


# ── Verifier tests ─────────────────────────────────────────────────────


@_NEEDS_TREE_SITTER
class TestTranslationVerifier(unittest.TestCase):
    """End-to-end verification pipeline."""

    @classmethod
    def setUpClass(cls):
        cls.verifier = TranslationVerifier()

    def test_verify_simple_ma_cross(self):
        source = _read_fixture("simple_ma_cross.mq4")
        result = self.verifier.verify(source, "simple_ma_cross.mq4")
        self.assertTrue(result.transpile_ok)
        self.assertIsNotNone(result.coverage_report)
        self.assertGreater(result.transpile_stats.get("lines_out", 0), 0)

    def test_verify_grid_trader(self):
        source = _read_fixture("grid_trader.mq4")
        result = self.verifier.verify(source, "grid_trader.mq4")
        self.assertTrue(result.transpile_ok)
        self.assertIn("Lifecycle", str(result.coverage_report))

    def test_verify_martingale(self):
        source = _read_fixture("martingale.mq4")
        result = self.verifier.verify(source, "martingale.mq4")
        self.assertTrue(result.transpile_ok)

    def test_verify_hedge_twins(self):
        source = _read_fixture("hedge_twins.mq4")
        result = self.verifier.verify(source, "hedge_twins.mq4")
        self.assertTrue(result.transpile_ok)

    def test_verify_custom_signal(self):
        source = _read_fixture("custom_signal.mq4")
        result = self.verifier.verify(source, "custom_signal.mq4")
        self.assertTrue(result.transpile_ok)
        # Custom signal uses iCustom.
        cov_cats = [c["name"] for c in result.coverage_report.get("categories", [])]
        self.assertIn("Custom Indicators", cov_cats)

    def test_result_serialization(self):
        source = _read_fixture("simple_ma_cross.mq4")
        result = self.verifier.verify(source, "test.mq4")
        json_str = result.to_json()
        parsed = json.loads(json_str)
        self.assertEqual(parsed["mql_file"], "test.mq4")
        self.assertIn("coverage", parsed)
        self.assertIn("transpile_stats", parsed)

    def test_all_five_fixtures_produce_valid_reports(self):
        fixtures = [
            "simple_ma_cross.mq4",
            "grid_trader.mq4",
            "martingale.mq4",
            "hedge_twins.mq4",
            "custom_signal.mq4",
        ]
        for name in fixtures:
            source = _read_fixture(name)
            result = self.verifier.verify(source, name)
            with self.subTest(fixture=name):
                self.assertTrue(result.transpile_ok, f"{name}: transpile failed")
                self.assertIsNotNone(result.coverage_report, f"{name}: no coverage report")
                self.assertGreater(
                    result.coverage_report.get("overall_pct", 0), 0,
                    f"{name}: zero coverage"
                )

    def test_coverage_report_includes_all_categories(self):
        """A comprehensive MQL source should cover all 20 categories."""
        # Build a source that exercises every feature category.
        source = """
        extern double LotSize = 0.1;
        int OnInit() { EventSetTimer(60); return INIT_SUCCEEDED; }
        void OnTick() {
            double ma = iMA(Symbol(), 0, 14, 0, MODE_EMA, PRICE_CLOSE);
            double rsi = iRSI(Symbol(), 0, 14, PRICE_CLOSE, 0);
            OrderSend(Symbol(), OP_BUY, 0.1, Ask, 3, 0, 0, "e", 1, 0, 0);
            OrderSend(Symbol(), OP_BUYLIMIT, 0.1, 1.08000, 3, 0, 0, "e", 1, 0, 0);
            double bal = AccountBalance();
            double eq = AccountEquity();
            double pt = Point;
            int dg = Digits;
            double v = MathAbs(-1.0);
            string s = StringConcatenate("a", "b");
            double c = Close[0];
            int h = FileOpen("f", FILE_WRITE);
            ObjectCreate(0, "l", OBJ_LABEL, 0, 0, 0);
        }
        void OnDeinit(const int r) { EventKillTimer(); }
        """
        result = self.verifier.verify(source, "comprehensive.mq4")
        cov = result.coverage_report
        cats = {c["name"]: c["status"] for c in cov.get("categories", [])}

        # Must detect these categories.
        for expected_cat in ["Lifecycle", "Market Orders", "Pending Orders",
                             "Technical Indicators", "Account Functions",
                             "Symbol Metadata", "Parameters", "Timer",
                             "Math Functions", "Price Data Access",
                             "File I/O", "GUI Objects"]:
            self.assertIn(expected_cat, cats, f"Missing coverage category: {expected_cat}")

        # File I/O and GUI must be NONE.
        self.assertEqual(cats.get("File I/O"), "NONE")
        self.assertEqual(cats.get("GUI Objects"), "NONE")


class TestGoldenCorpus(unittest.TestCase):
    """Golden reference corpus validation."""

    def test_golden_directory_exists(self):
        golden_dir = os.path.join(
            os.path.dirname(__file__), "..", "..", "..",
            "strategy-service", "tests", "golden"
        )
        self.assertTrue(os.path.isdir(golden_dir), f"Golden dir missing: {golden_dir}")

    def test_threshold_cross_baseline_exists(self):
        golden_dir = os.path.join(
            os.path.dirname(__file__), "..", "..", "..",
            "strategy-service", "tests", "golden"
        )
        baseline = os.path.join(golden_dir, "threshold_cross_baseline.json")
        self.assertTrue(os.path.isfile(baseline), f"Baseline missing: {baseline}")

        with open(baseline) as f:
            data = json.load(f)
        self.assertIn("strategy_name", data)
        self.assertIn("actual_events", data)
        self.assertIn("assumptions", data)


if __name__ == "__main__":
    unittest.main()
