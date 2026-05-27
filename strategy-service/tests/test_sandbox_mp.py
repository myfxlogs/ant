"""Tests for app/engine/backtest_sandbox.py and live_sandbox.py (B-3.6).

Covers:
  (a) Infinite-loop timeout: ``while True: pass``
  (b) Sleep-style timeout: long-running pure-Python computation
  (c) C-extension timeout: ``numpy.dot()`` large matrix
  (d) Child OOM: parent process survives child memory exhaustion
"""

from __future__ import annotations

import multiprocessing as mp
import os

import numpy as np
import pytest

from app.engine.backtest_sandbox import BacktestSandbox, SandboxTimeoutError
from app.engine.live_sandbox import LiveWorker
from app.engine.types import StrategyCompileError, StrategyRuntimeError

# Use short timeouts so tests finish fast.
_FAST_TIMEOUT = 2_000  # ms

# Strategy that passes AST validation but spins forever.
_INFINITE_LOOP = "def run(context):\n    while True:\n        pass\n"

# Strategy that burns CPU for a very long time (no imports needed).
_LONG_COMPUTE = (
    "def run(context):\n"
    "    x = 0\n"
    "    for i in range(10**10):\n"
    "        x += 1\n"
    "    return 'hold'\n"
)

# Strategy that calls into a C extension (numpy) with a big computation.
# np is already in sandbox globals — no import needed.
_NUMPY_HEAVY = (
    "def run(context):\n"
    "    a = np.random.rand(4000, 4000)\n"
    "    b = np.random.rand(4000, 4000)\n"
    "    c = np.dot(a, b)\n"
    "    return 'hold'\n"
)

# Strategy that uses signal variable (valid but minimal).
_SIGNAL_HOLD = "signal = 'hold'\n"


# --- (a) infinite-loop timeout -------------------------------------------

def test_backtest_sandbox_timeout_infinite_loop():
    """BacktestSandbox should raise SandboxTimeoutError on while True: pass."""
    sb = BacktestSandbox(_INFINITE_LOOP, timeout_ms=_FAST_TIMEOUT)
    try:
        with pytest.raises(SandboxTimeoutError):
            sb.call({"close": np.array([1.0, 1.1, 1.2])})
    finally:
        sb.shutdown()


# --- (b) sleep-style timeout ---------------------------------------------

def test_backtest_sandbox_timeout_long_compute():
    """BacktestSandbox should timeout on a long-running pure-Python loop."""
    sb = BacktestSandbox(_LONG_COMPUTE, timeout_ms=_FAST_TIMEOUT)
    try:
        with pytest.raises(SandboxTimeoutError):
            sb.call({"close": np.array([1.0, 1.1, 1.2])})
    finally:
        sb.shutdown()


# --- (c) C-extension timeout ---------------------------------------------

@pytest.mark.slow
def test_backtest_sandbox_timeout_numpy_dot():
    """BacktestSandbox should timeout when C extension (numpy.dot) runs long.

    This is the key fix from V3-R-8: threading.Timer could not interrupt
    C extension loops, but Process.terminate() can.
    """
    sb = BacktestSandbox(_NUMPY_HEAVY, timeout_ms=_FAST_TIMEOUT + 1000)
    try:
        with pytest.raises(SandboxTimeoutError):
            sb.call({"close": np.array([1.0, 1.1, 1.2])})
    finally:
        sb.shutdown()


# --- (d) child OOM: parent survives --------------------------------------

def test_backtest_sandbox_parent_survives_child_oom():
    """Parent process remains alive after child exhausts memory and dies."""
    # Use a small-ish memory bound to trigger OOM in the child.
    import resource
    soft, hard = resource.getrlimit(resource.RLIMIT_AS)
    # Give the child 64MB of address space — numpy will quickly exceed that.
    old_soft, old_hard = soft, hard

    try:
        resource.setrlimit(resource.RLIMIT_AS, (64 * 1024 * 1024, hard))
    except (ValueError, OSError):
        pytest.skip("cannot set RLIMIT_AS in this environment")

    sb = BacktestSandbox(_NUMPY_HEAVY, timeout_ms=_FAST_TIMEOUT)
    try:
        # Child should die (OOM or timeout); parent must not crash.
        try:
            sb.call({"close": np.array([1.0, 1.1, 1.2])})
        except (SandboxTimeoutError, StrategyRuntimeError):
            pass  # either outcome is fine — child is gone
        # Parent is still alive — we can create and use another sandbox.
        sb2 = BacktestSandbox(_SIGNAL_HOLD, timeout_ms=_FAST_TIMEOUT)
        try:
            result = sb2.call({})
            assert result == {"signal": "hold"}
        finally:
            sb2.shutdown()
    finally:
        sb.shutdown()
        # Restore original rlimit.
        try:
            resource.setrlimit(resource.RLIMIT_AS, (old_soft, old_hard))
        except (ValueError, OSError):
            pass


# --- normal execution (sanity) -------------------------------------------

def test_backtest_sandbox_returns_signal_normally():
    """BacktestSandbox should return correct result for a normal strategy."""
    code = "def run(context):\n    return {'signal': 'buy', 'volume': 0.1}\n"
    sb = BacktestSandbox(code, timeout_ms=10_000)
    try:
        result = sb.call({"close": np.array([1.0, 1.1, 1.2])})
        assert result == {"signal": "buy", "volume": 0.1}
    finally:
        sb.shutdown()


def test_backtest_sandbox_returns_signal_variable():
    """BacktestSandbox should handle signal variable style."""
    code = "signal = {'signal': 'sell', 'volume': 1.5}\n"
    sb = BacktestSandbox(code, timeout_ms=10_000)
    try:
        result = sb.call({})
        assert result == {"signal": "sell", "volume": 1.5}
    finally:
        sb.shutdown()


def test_backtest_sandbox_rejects_invalid_code_on_init():
    """BacktestSandbox should reject invalid code at __init__ time."""
    with pytest.raises(StrategyCompileError):
        BacktestSandbox("import os\n", timeout_ms=10_000)


def test_backtest_sandbox_source_sha256():
    """source_sha256 property matches the code hash."""
    code = "signal = 'hold'\n"
    sb = BacktestSandbox(code, timeout_ms=10_000)
    from app.engine.sandbox import code_sha256
    assert sb.source_sha256 == code_sha256(code)


# --- LiveWorker tests -----------------------------------------------------

def test_live_worker_basic_call():
    """LiveWorker should return correct signal for a valid strategy."""
    code = "def run(context):\n    ctx_sym = context.get('symbol', '')\n    return {'signal': 'hold', 'sym': ctx_sym}\n"
    worker = LiveWorker(code, timeout_ms=5_000)
    try:
        result = worker.call({"symbol": "EURUSD"})
        assert result == {"signal": "hold", "sym": "EURUSD"}
    finally:
        worker.shutdown()


def test_live_worker_rejects_invalid_code():
    """LiveWorker should reject invalid code at __init__."""
    with pytest.raises(StrategyCompileError):
        LiveWorker("import os\n", timeout_ms=5_000)


def test_live_worker_multiple_calls():
    """LiveWorker should handle multiple calls without restarting."""
    code = "def run(context):\n    return {'signal': 'hold', 'count': context.get('n', 0)}\n"
    worker = LiveWorker(code, timeout_ms=5_000)
    try:
        for i in range(10):
            result = worker.call({"n": i})
            assert result == {"signal": "hold", "count": i}
    finally:
        worker.shutdown()


def test_live_worker_uses_precompiled_bytecode():
    """LiveWorker stores pre-compiled bytecode and uses it across calls."""
    code = "def run(context):\n    return {'signal': 'hold'}\n"
    worker = LiveWorker(code, timeout_ms=5_000)
    try:
        # self._serialized_code should exist and be bytes.
        assert isinstance(worker._serialized_code, bytes)
        assert len(worker._serialized_code) > 0
    finally:
        worker.shutdown()
