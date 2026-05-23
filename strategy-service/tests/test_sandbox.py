"""Tests for app/engine/sandbox.py."""

from __future__ import annotations

import numpy as np
import pytest

from app.engine.sandbox import (
    StrategyRunner,
    code_sha256,
    validate_strategy_code,
)
from app.engine.types import StrategyCompileError, StrategyRuntimeError


# --- validate_strategy_code ---------------------------------------------


def test_validate_accepts_run_function():
    code = "def run(context):\n    return {'signal': 'hold'}\n"
    r = validate_strategy_code(code)
    assert r.valid is True
    assert r.errors == []


def test_validate_accepts_signal_variable():
    code = "signal = {'signal': 'hold'}\n"
    r = validate_strategy_code(code)
    assert r.valid is True


def test_validate_rejects_import():
    code = "import os\nsignal = 'hold'\n"
    r = validate_strategy_code(code)
    assert r.valid is False
    assert any("import" in e for e in r.errors)


def test_validate_rejects_from_import():
    code = "from os import path\nsignal = 'hold'\n"
    r = validate_strategy_code(code)
    assert r.valid is False


def test_validate_rejects_global_keyword():
    code = "def run(context):\n    global x\n    return None\n"
    r = validate_strategy_code(code)
    assert r.valid is False
    assert any("global" in e for e in r.errors)


def test_validate_rejects_dunder_attribute():
    code = "def run(context):\n    return context.__class__\n"
    r = validate_strategy_code(code)
    assert r.valid is False


def test_validate_rejects_dunder_name():
    code = "def run(context):\n    x = __name__\n    return None\n"
    r = validate_strategy_code(code)
    assert r.valid is False


@pytest.mark.parametrize("name", ["open", "eval", "exec", "compile", "input", "globals", "locals", "vars", "dir"])
def test_validate_rejects_forbidden_builtins(name):
    code = f"def run(context):\n    {name}(1)\n    return None\n"
    r = validate_strategy_code(code)
    assert r.valid is False


def test_validate_rejects_multiple_run_functions():
    code = "def run(a):\n    pass\ndef run(b):\n    pass\n"
    r = validate_strategy_code(code)
    assert r.valid is False
    assert any("只允许定义一个" in e for e in r.errors)


def test_validate_rejects_bad_run_signature():
    code = "def run(a, b):\n    return None\n"
    r = validate_strategy_code(code)
    assert r.valid is False


def test_validate_rejects_varargs():
    code = "def run(*args):\n    return None\n"
    r = validate_strategy_code(code)
    assert r.valid is False


def test_validate_requires_signal_or_run():
    r = validate_strategy_code("x = 1\n")
    assert r.valid is False


def test_validate_reports_syntax_error():
    r = validate_strategy_code("def run(context:\n    pass\n")
    assert r.valid is False


def test_validate_deduplicates_errors():
    code = "import os\nimport sys\n"
    r = validate_strategy_code(code)
    # Both imports trigger the same message; dedup keeps one.
    import_errs = [e for e in r.errors if "import" in e]
    assert len(import_errs) == 1


# --- StrategyRunner.call -------------------------------------------------


def test_runner_rejects_invalid_code():
    with pytest.raises(StrategyCompileError):
        StrategyRunner("import os\n")


def test_runner_returns_signal_from_run():
    code = "def run(context):\n    return {'signal': 'buy', 'volume': 1.0}\n"
    sr = StrategyRunner(code)
    result = sr.call({"close": np.array([1.0, 1.1, 1.2])})
    assert result == {"signal": "buy", "volume": 1.0}


def test_runner_returns_signal_variable_fallback():
    code = "signal = {'signal': 'sell'}\n"
    sr = StrategyRunner(code)
    assert sr.call({}) == {"signal": "sell"}


def test_runner_coerces_string_signal_to_dict():
    code = "def run(context):\n    return 'hold'\n"
    sr = StrategyRunner(code)
    assert sr.call({}) == {"signal": "hold"}


def test_runner_none_signal_is_ok():
    code = "def run(context):\n    return None\n"
    sr = StrategyRunner(code)
    assert sr.call({}) is None


def test_runner_rejects_non_dict_non_string_return():
    code = "def run(context):\n    return 42\n"
    sr = StrategyRunner(code)
    with pytest.raises(StrategyRuntimeError):
        sr.call({})


def test_runner_propagates_runtime_error():
    code = "def run(context):\n    return 1 / 0\n"
    sr = StrategyRunner(code)
    with pytest.raises(StrategyRuntimeError):
        sr.call({})


def test_runner_requires_signal_or_run():
    # Need to pass validation but have no signal/run at runtime.
    # Validation forces presence of one, so we can't actually reach the branch
    # without bypass. This test just documents the guard exists.
    with pytest.raises(StrategyCompileError):
        StrategyRunner("x = 1\n")


# --- helpers: indicators visible inside sandbox --------------------------


def test_sandbox_exposes_iRSI_and_np():
    code = (
        "def run(context):\n"
        "    closes = context['close']\n"
        "    rsi = iRSI(closes, 14)\n"
        "    return {'signal': 'hold', 'rsi': rsi}\n"
    )
    sr = StrategyRunner(code)
    r = sr.call({"close": np.linspace(1.0, 2.0, 30)})
    assert r is not None
    assert r["rsi"] == 100.0  # monotone up → 100


def test_sandbox_exposes_iMA():
    code = (
        "def run(context):\n"
        "    ma = iMA(context['close'], 5)\n"
        "    return {'signal': 'buy' if ma > 1.0 else 'hold'}\n"
    )
    sr = StrategyRunner(code)
    r = sr.call({"close": np.linspace(1.0, 2.0, 30)})
    assert r == {"signal": "buy"}


def test_sandbox_blocks_imports_at_compile():
    code = "def run(context):\n    import os\n    return None\n"
    with pytest.raises(StrategyCompileError):
        StrategyRunner(code)


# --- bytecode cache / sha -----------------------------------------------


def test_source_sha256_is_stable():
    code = "def run(context):\n    return None\n"
    sr1 = StrategyRunner(code)
    sr2 = StrategyRunner(code)
    assert sr1.source_sha256 == sr2.source_sha256 == code_sha256(code)
