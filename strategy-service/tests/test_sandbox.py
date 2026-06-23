"""Tests for app/engine/sandbox.py."""

from __future__ import annotations

import pytest

from app.engine.sandbox import (
    validate_strategy_code,
)


# --- helpers --------------------------------------------------------------


def _sdk_class(body: str, class_name: str = "TestStrategy") -> str:
    """Wrap strategy body lines in a minimal valid SDK class definition."""
    lines = [l for l in body.split("\n") if l.strip()]
    indented = "\n".join(f"    {l}" for l in lines)
    return (
        "from app.sdk import StrategyBase\n"
        f"class {class_name}(StrategyBase):\n"
        f"{indented}\n"
    )


# --- validate_strategy_code (SDK-only) -----------------------------------


def test_validate_accepts_sdk_with_lifecycle_hook():
    code = _sdk_class(
        "def on_bar(self):\n"
        "    self.broker.send_order('EURUSD', 0.1, 'buy')\n"
    )
    r = validate_strategy_code(code)
    assert r.valid is True
    assert r.errors == []


def test_validate_rejects_non_sdk_run_function():
    code = "def run(context):\n    return {'signal': 'hold'}\n"
    r = validate_strategy_code(code)
    assert r.valid is False
    assert any("SDK" in e for e in r.errors)


def test_validate_rejects_non_sdk_signal_variable():
    code = "signal = {'signal': 'hold'}\n"
    r = validate_strategy_code(code)
    assert r.valid is False
    assert any("SDK" in e for e in r.errors)


def test_validate_rejects_class_without_strategy_base():
    code = "class Foo:\n    def on_bar(self):\n        pass\n"
    r = validate_strategy_code(code)
    assert r.valid is False


def test_validate_rejects_sdk_without_lifecycle_hook():
    code = (
        "from app.sdk import StrategyBase\n"
        "class TestStrategy(StrategyBase):\n"
        "    def some_helper(self):\n"
        "        pass\n"
    )
    r = validate_strategy_code(code)
    assert r.valid is False
    assert any("生命周期" in e for e in r.errors)


def test_validate_reports_syntax_error():
    r = validate_strategy_code("class Foo(StrategyBase:\n    pass\n")
    assert r.valid is False


def test_validate_accepts_underscore_helpers():
    """Full Python allows _-prefixed helper names (no RestrictedPython)."""
    code = _sdk_class(
        "def on_bar(self):\n"
        "    self._count_orders('EURUSD')\n"
        "def _count_orders(self, symbol):\n"
        "    return 0\n"
    )
    r = validate_strategy_code(code)
    assert r.valid is True
    assert r.errors == []


def test_validate_accepts_private_attributes():
    code = _sdk_class(
        "def on_init(self):\n"
        "    self._orders = []\n"
        "    self._ticket = None\n"
        "def on_bar(self):\n"
        "    self.broker.send_order('EURUSD', 0.1, 'buy')\n"
    )
    r = validate_strategy_code(code)
    assert r.valid is True
