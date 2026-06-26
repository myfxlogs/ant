"""行为等价端到端测试 — MQL → Python → 信号对比。

验证生成的策略与参考 SDK 策略在相同数据下产生相同的交易信号。

完整的端到端流程:
  1. MQL 源码 → 意图 IR → Python 代码
  2. Python 代码 ast.parse 编译
  3. 生成的策略 vs 参考策略各跑一套标准行情
  4. 对比信号序列

当前阶段: 编译验证 + 结构检查（完整行为验证需要 SimBroker 运行时）。
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


def _read_reference(name: str) -> str:
    """Returns the reference SDK strategy .py code."""
    import os
    ref_map = {
        "simple_ma_cross": "single_ma_cross.py",
        "grid_trader": "grid_trader.py",
        "martingale": "martingale.py",
        "hedge_twins": "hedge_twins.py",
        "custom_signal": "custom_signal.py",
    }
    path = os.path.join(
        os.path.dirname(__file__),
        "..", "..", "..",
        "strategy-service", "tests", "sdk_samples",
        ref_map[name],
    )
    with open(path) as f:
        return f.read()


class TestE2ECompile(unittest.TestCase):
    """完整流水线产出必须编译。"""

    def setUp(self):
        self._pipe = MigrationPipeline()

    def test_all_fixtures_e2e(self):
        for name in FIXTURES:
            with self.subTest(fixture=name):
                result = self._pipe.run(_read_fixture(name),
                                       source_name=f"{name}.mq4")
                code = result.python_code

                # Gate 1: Compile
                try:
                    py_ast.parse(code)
                except SyntaxError as e:
                    lines = code.split("\n")
                    ctx = lines[e.lineno - 1][:120] if e.lineno and e.lineno <= len(lines) else "?"
                    self.fail(f"{name}: SyntaxError L{e.lineno}: {e.msg}\n  {ctx}")

                # Gate 2: Quality gates
                report = QualityGate.assess(code)
                self.assertEqual(
                    report.verdict, QualityVerdict.HIGH,
                    f"{name}: gate failed: {[f.message for f in report.failures]}"
                )

                # Gate 3: Coverage threshold
                self.assertGreaterEqual(
                    result.intent.coverage_score, 0.5,
                    f"{name}: coverage {result.intent.coverage_score:.0%} < 50%"
                )


class TestE2EStructuralEquivalence(unittest.TestCase):
    """生成代码与参考策略的结构对比。

    验证: 相同的方法、相同的 broker 调用模式、相同的指标调用。
    完整信号对比需要在 SimBroker 就绪后进行。
    """

    def setUp(self):
        self._pipe = MigrationPipeline()

    def _extract_patterns(self, code: str) -> dict:
        import re
        return {
            "methods": set(re.findall(r'def (\w+)', code)),
            "broker_calls": set(re.findall(r'self\.broker\.(\w+)\(', code)),
            "indicator_calls": set(re.findall(r'self\.indicators\.(\w+)\(', code)),
            "has_type_filling": "type_filling" in code,
            "has_account_mode": "AccountMode" in code,
        }

    def test_generated_has_type_filling(self):
        for name in FIXTURES:
            with self.subTest(fixture=name):
                result = self._pipe.run(_read_fixture(name),
                                       source_name=f"{name}.mq4")
                pat = self._extract_patterns(result.python_code)
                self.assertTrue(pat["has_type_filling"],
                                f"{name}: missing type_filling in generated code")

    def test_generated_has_account_mode(self):
        for name in FIXTURES:
            with self.subTest(fixture=name):
                result = self._pipe.run(_read_fixture(name),
                                       source_name=f"{name}.mq4")
                pat = self._extract_patterns(result.python_code)
                self.assertTrue(pat["has_account_mode"],
                                f"{name}: missing AccountMode in generated code")

    def test_reference_has_type_filling(self):
        """Verify our assumption: all reference strategies use type_filling."""
        for name in FIXTURES:
            with self.subTest(fixture=name):
                ref_code = _read_reference(name)
                pat = self._extract_patterns(ref_code)
                self.assertTrue(pat["has_type_filling"],
                                f"{name}: reference strategy missing type_filling")


class TestE2EDeterministic(unittest.TestCase):
    """相同输入两次 → 完全相同输出。"""

    def setUp(self):
        self._pipe = MigrationPipeline()

    def test_idempotent(self):
        for name in FIXTURES:
            with self.subTest(fixture=name):
                src = _read_fixture(name)
                o1 = self._pipe.run(src, source_name=f"{name}.mq4").python_code
                o2 = self._pipe.run(src, source_name=f"{name}.mq4").python_code
                self.assertEqual(o1, o2, f"{name}: output differs between runs")


class TestE2EBehavioralEquivalence(unittest.TestCase):
    """行为等价验证骨架。

    完整实现需要:
      1. RecordingBroker + StubContext (已有 behavioral_harness.py)
      2. 动态加载生成策略类
      3. 运行标准行情序列
      4. 对比信号录制

    当前: 验证生成代码可被 import（语法层等价）。
    """

    def setUp(self):
        self._pipe = MigrationPipeline()

    def test_generated_code_is_importable(self):
        """验证生成的代码可以作为 Python 模块 import。"""
        for name in FIXTURES:
            with self.subTest(fixture=name):
                result = self._pipe.run(_read_fixture(name),
                                       source_name=f"{name}.mq4")
                code = result.python_code
                # Compile to code object (stronger than ast.parse)
                try:
                    compile(code, f"<{name}>", "exec")
                except SyntaxError as e:
                    self.fail(f"{name}: compile() failed: {e}")


if __name__ == "__main__":
    unittest.main()
