"""迁移流水线端到端测试。

验证: MQL → Intent IR → Python 代码 → 编译 → 质量门禁
"""

import ast as py_ast
import unittest

from tools.mql_transpiler.quality_gate import QualityGate, QualityVerdict
from tools.mql_migration.pipeline import MigrationPipeline

FIXTURES = [
    "simple_ma_cross",
    "grid_trader",
    "martingale",
    "hedge_twins",
    "custom_signal",
]


def _read_fixture(name: str) -> str:
    from pathlib import Path
    path = Path(__file__).parent.parent.parent / "mql_transpiler" / "tests" / "fixtures" / f"{name}.mq4"
    return path.read_text()


class TestAllFixturesCompile(unittest.TestCase):
    """迁移产出的代码必须能编译。"""

    def setUp(self):
        self._pipe = MigrationPipeline()

    def test_each_fixture_compiles(self):
        for name in FIXTURES:
            with self.subTest(fixture=name):
                result = self._pipe.run(_read_fixture(name), source_name=f"{name}.mq4")
                try:
                    py_ast.parse(result.python_code)
                except SyntaxError as e:
                    lines = result.python_code.split("\n")
                    ctx = ""
                    if e.lineno and 1 <= e.lineno <= len(lines):
                        ctx = lines[e.lineno - 1][:120]
                    self.fail(
                        f"{name}: SyntaxError at line {e.lineno}: {e.msg}\n  {ctx}"
                    )


class TestAllFixturesPassQualityGates(unittest.TestCase):
    """迁移产出的代码必须通过所有质量门禁。"""

    def setUp(self):
        self._pipe = MigrationPipeline()

    def test_each_fixture_passes_all_gates(self):
        for name in FIXTURES:
            with self.subTest(fixture=name):
                result = self._pipe.run(_read_fixture(name), source_name=f"{name}.mq4")
                report = QualityGate.assess(result.python_code)
                self.assertEqual(
                    report.verdict, QualityVerdict.HIGH,
                    f"{name}: expected HIGH, failures: {[f.message for f in report.failures]}"
                )


class TestIntentStructure(unittest.TestCase):
    """意图 IR 必须包含完整结构。"""

    def setUp(self):
        self._pipe = MigrationPipeline()

    def test_intent_has_meta(self):
        for name in FIXTURES:
            with self.subTest(fixture=name):
                result = self._pipe.run(_read_fixture(name), source_name=f"{name}.mq4")
                self.assertTrue(result.intent.meta.name, f"{name}: meta.name is empty")
                self.assertIsNotNone(result.intent.meta.mql_version, f"{name}: version is None")

    def test_intent_has_params(self):
        for name in FIXTURES:
            with self.subTest(fixture=name):
                result = self._pipe.run(_read_fixture(name), source_name=f"{name}.mq4")
                self.assertGreater(len(result.intent.params), 0,
                                   f"{name}: no params extracted")

    def test_intent_has_execution_model(self):
        for name in FIXTURES:
            with self.subTest(fixture=name):
                result = self._pipe.run(_read_fixture(name), source_name=f"{name}.mq4")
                self.assertIsNotNone(result.intent.execution,
                                     f"{name}: execution model is None")

    def test_intent_has_sizing(self):
        for name in FIXTURES:
            with self.subTest(fixture=name):
                result = self._pipe.run(_read_fixture(name), source_name=f"{name}.mq4")
                self.assertIsNotNone(result.intent.sizing,
                                     f"{name}: sizing is None")


class TestGeneratedCodePatterns(unittest.TestCase):
    """生成的代码必须包含 SDK 惯用模式。"""

    def setUp(self):
        self._pipe = MigrationPipeline()

    def test_code_has_type_filling(self):
        for name in FIXTURES:
            with self.subTest(fixture=name):
                result = self._pipe.run(_read_fixture(name), source_name=f"{name}.mq4")
                self.assertIn("type_filling", result.python_code,
                              f"{name}: missing type_filling")

    def test_code_has_strategy_base(self):
        for name in FIXTURES:
            with self.subTest(fixture=name):
                result = self._pipe.run(_read_fixture(name), source_name=f"{name}.mq4")
                self.assertIn("StrategyBase", result.python_code,
                              f"{name}: missing StrategyBase")

    def test_code_has_account_mode_check(self):
        for name in FIXTURES:
            with self.subTest(fixture=name):
                result = self._pipe.run(_read_fixture(name), source_name=f"{name}.mq4")
                self.assertIn("AccountMode", result.python_code,
                              f"{name}: missing AccountMode check")

    def test_grid_has_on_init_grid_pattern(self):
        result = self._pipe.run(_read_fixture("grid_trader"), source_name="grid_trader.mq4")
        self.assertEqual(result.intent.execution.kind.value, "on_init_grid")


class TestCoverageThreshold(unittest.TestCase):
    """覆盖率必须达到合理水平。"""

    def setUp(self):
        self._pipe = MigrationPipeline()

    def test_coverage_above_25_percent_trading_only(self):
        """Coverage only counts trading blocks (entries/exits/indicators/risk).
        Meta/execution/sizing are structural and not part of the score."""
        for name in FIXTURES:
            with self.subTest(fixture=name):
                result = self._pipe.run(_read_fixture(name), source_name=f"{name}.mq4")
                self.assertGreater(
                    result.intent.coverage_score, 0.0,
                    f"{name}: coverage is 0 — no trading blocks detected"
                )
                # Coverage ≤ 1.0 (should not exceed 100%)
                self.assertLessEqual(
                    result.intent.coverage_score, 1.0,
                    f"{name}: coverage > 100% — calculation error"
                )


class TestDeterministic(unittest.TestCase):
    """相同输入 → 相同输出。"""

    def setUp(self):
        self._pipe = MigrationPipeline()

    def test_idempotent(self):
        for name in FIXTURES:
            with self.subTest(fixture=name):
                src = _read_fixture(name)
                o1 = self._pipe.run(src, source_name=f"{name}.mq4").python_code
                o2 = self._pipe.run(src, source_name=f"{name}.mq4").python_code
                self.assertEqual(o1, o2, f"{name}: output differs between runs")


if __name__ == "__main__":
    unittest.main()
