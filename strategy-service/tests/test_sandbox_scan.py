# M5-4: strategy-service sandbox unit tests.
import pytest
from app.sandbox_scan import scan_code, ScanResult


class TestSandboxScan:
    def test_safe_code_passes(self):
        code = """
def moving_average(prices, window):
    return sum(prices[-window:]) / window

prices = [1, 2, 3, 4, 5]
result = moving_average(prices, 3)
"""
        r = scan_code(code)
        assert r.passed, f"safe code should pass, got violations: {r.violations}"

    def test_banned_import_detected(self):
        code = "import os\nos.system('rm -rf /')"
        r = scan_code(code)
        assert not r.passed
        assert any("os" in v for v in r.violations), f"expected os import violation, got {r.violations}"

    def test_banned_builtins_detected(self):
        code = "eval('1+1')"
        r = scan_code(code)
        assert not r.passed
        assert any("eval" in v for v in r.violations), f"expected eval violation, got {r.violations}"

    def test_exec_blocked(self):
        code = "exec('x=1')"
        r = scan_code(code)
        assert not r.passed

    def test_subprocess_blocked(self):
        code = "import subprocess\nsubprocess.run(['ls'])"
        r = scan_code(code)
        assert not r.passed

    def test_syntax_error_caught(self):
        code = "def broken("
        r = scan_code(code)
        assert not r.passed
        assert any("syntax" in v for v in r.violations)

    def test_http_import_blocked(self):
        code = "from urllib.request import urlopen"
        r = scan_code(code)
        assert not r.passed

    def test_code_too_long(self):
        code = "x = 1\n" * 6000
        r = scan_code(code)
        assert not r.passed
