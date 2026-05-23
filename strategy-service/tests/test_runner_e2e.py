"""End-to-end tests for app/engine/runner.py.

Runs the full tick loop across small fixture datasets to verify:

* Market order buy/sell flow produces paired open/close events.
* Pending limit / stop orders fill and close correctly.
* SL / TP triggers.
* Single-position lock.
* End-of-test force liquidation.
* Runner error handling surfaces ``success=False`` with an error message.
* BacktestResult shape matches the contract.
"""

from __future__ import annotations

from datetime import datetime, timezone

import pytest

from app.engine.runner import BacktestRunner, run_backtest
from app.engine.types import (
    Bar,
    BacktestRequest,
    CloseReason,
    CostProfile,
    Side,
    Tick,
)


# --- fixtures ------------------------------------------------------------


def _bars(prices: list[float], step_ms: int = 60_000) -> list[Bar]:
    bars = []
    for i, c in enumerate(prices):
        ot = i * step_ms
        ct = ot + step_ms
        bars.append(Bar(open_time=ot, close_time=ct,
                        open=c, high=c + 0.001, low=c - 0.001,
                        close=c, volume=100.0))
    return bars


def _req(
    strategy_code: str,
    prices: list[float],
    ticks: list[Tick] | None = None,
    **over,
) -> BacktestRequest:
    bars = _bars(prices)
    base = dict(
        run_id="test-run",
        user_id=1,
        account_id=1,
        symbol="EURUSD",
        timeframe="M1",
        start=datetime(2024, 1, 1, tzinfo=timezone.utc),
        end=datetime(2024, 1, 2, tzinfo=timezone.utc),
        initial_cash=10_000.0,
        bars=bars,
        ticks=ticks,
        strategy_code=strategy_code,
    )
    base.update(over)
    return BacktestRequest(**base)


# --- simple paths --------------------------------------------------------


def test_hold_strategy_produces_flat_curve():
    code = "def run(context):\n    return {'signal': 'hold'}\n"
    req = _req(code, [1.0, 1.01, 1.02, 1.03])
    res = run_backtest(req)
    assert res.success is True
    assert res.events == []
    assert res.trades == []
    assert res.equity_curve[0] == 10_000.0
    assert res.equity_curve[-1] == 10_000.0


def test_buy_then_end_of_test_liquidates_when_not_legacy():
    # Buy on first bar, hold forever; when legacy_pnl=False the runner must
    # close at end of test so that all PnL is realised.
    code = (
        "def run(context):\n"
        "    if len(context['close']) == 1:\n"
        "        return {'signal': 'buy', 'volume': 1.0}\n"
        "    return {'signal': 'hold'}\n"
    )
    prices = [1.0, 1.1, 1.2, 1.3]
    res = run_backtest(_req(code, prices, legacy_pnl=False))
    assert res.success is True
    opens = [e for e in res.events if e["type"] == "position_open"]
    closes = [e for e in res.events if e["type"] == "position_close"]
    assert len(opens) == 1
    assert len(closes) == 1
    assert closes[0]["reason"] == CloseReason.END_OF_TEST.value
    # legacy_pnl=True default: pnl = (close_bid - open_ask) * vol
    # ticks are synthesized from bars (bid=ask=close). Open at bar 0 close=1.0,
    # close at bar 3 close=1.3 → pnl ≈ 0.3.
    assert res.trades[0].pnl == pytest.approx(0.3)


def test_single_position_only_blocks_second_buy():
    code = (
        "def run(context):\n"
        "    return {'signal': 'buy', 'volume': 1.0}\n"
    )
    # Strategy tries to buy every bar; single_position_only must reject all but first.
    res = run_backtest(_req(code, [1.0, 1.1, 1.2]))
    opens = [e for e in res.events if e["type"] == "position_open"]
    assert len(opens) == 1


def test_single_position_off_allows_multiple_buys():
    code = (
        "def run(context):\n"
        "    return {'signal': 'buy', 'volume': 1.0}\n"
    )
    res = run_backtest(_req(code, [1.0, 1.1, 1.2], single_position_only=False))
    opens = [e for e in res.events if e["type"] == "position_open"]
    assert len(opens) >= 2


# --- pending orders ------------------------------------------------------


def test_pending_buy_limit_fills_when_price_drops():
    # Bar 0 enqueues a BUY_LIMIT at 0.98; price drops to 0.97 → fills.
    code = (
        "def run(context):\n"
        "    if len(context['close']) == 1:\n"
        "        return {'signal': 'buy_limit', 'price': 0.98, 'volume': 1.0}\n"
        "    return {'signal': 'hold'}\n"
    )
    prices = [1.0, 0.97, 0.97, 0.97]
    res = run_backtest(_req(code, prices))
    opens = [e for e in res.events if e["type"] == "position_open"]
    assert len(opens) == 1
    # Legacy: filled at tick.ask == 0.97.
    assert opens[0]["price"] == pytest.approx(0.97)


def test_cancel_pending_clears_queue():
    # Enqueue a limit then next bar cancel.
    code = (
        "def run(context):\n"
        "    n = len(context['close'])\n"
        "    if n == 1:\n"
        "        return {'signal': 'buy_limit', 'price': 0.5, 'volume': 1.0}\n"
        "    if n == 2:\n"
        "        return {'signal': 'cancel_pending'}\n"
        "    return {'signal': 'hold'}\n"
    )
    res = run_backtest(_req(code, [1.0, 1.0, 0.4, 0.3]))
    # Even though price later drops below 0.5, queue was cancelled.
    opens = [e for e in res.events if e["type"] == "position_open"]
    assert opens == []


# --- SL / TP -------------------------------------------------------------


def test_stop_loss_triggers_sl_event():
    # Buy with SL=0.98; price drops → SL hit.
    code = (
        "def run(context):\n"
        "    if len(context['close']) == 1:\n"
        "        return {'signal': 'buy', 'volume': 1.0, 'stop_loss': 0.98}\n"
        "    return {'signal': 'hold'}\n"
    )
    res = run_backtest(_req(code, [1.0, 0.97, 0.96]))
    closes = [e for e in res.events if e["type"] == "position_close"]
    assert len(closes) == 1
    assert closes[0]["reason"] == CloseReason.SL.value


def test_take_profit_triggers_tp_event():
    code = (
        "def run(context):\n"
        "    if len(context['close']) == 1:\n"
        "        return {'signal': 'buy', 'volume': 1.0, 'take_profit': 1.05}\n"
        "    return {'signal': 'hold'}\n"
    )
    res = run_backtest(_req(code, [1.0, 1.06, 1.07]))
    closes = [e for e in res.events if e["type"] == "position_close"]
    assert len(closes) == 1
    assert closes[0]["reason"] == CloseReason.TP.value


# --- close action --------------------------------------------------------


def test_close_signal_liquidates_open_positions():
    code = (
        "def run(context):\n"
        "    n = len(context['close'])\n"
        "    if n == 1:\n"
        "        return {'signal': 'buy', 'volume': 1.0}\n"
        "    if n == 3:\n"
        "        return {'signal': 'close'}\n"
        "    return {'signal': 'hold'}\n"
    )
    res = run_backtest(_req(code, [1.0, 1.05, 1.1, 1.15]))
    closes = [e for e in res.events if e["type"] == "position_close"]
    assert len(closes) == 1
    assert closes[0]["reason"] == CloseReason.SIGNAL.value


# --- error surfacing -----------------------------------------------------


def test_runtime_error_surfaces_as_success_false():
    code = (
        "def run(context):\n"
        "    return 1 / 0\n"  # ZeroDivision at first bar
    )
    res = run_backtest(_req(code, [1.0, 1.1]))
    assert res.success is False
    assert res.error is not None
    # Metrics still computed on the partial equity curve.
    assert res.metrics is not None


def test_compile_error_raises_at_construction():
    from app.engine.types import StrategyCompileError
    req = _req("import os\n", [1.0, 1.1])
    with pytest.raises(StrategyCompileError):
        BacktestRunner(req)


# --- snapshot & shape ----------------------------------------------------


def test_result_shape_matches_contract():
    code = "def run(context):\n    return {'signal': 'hold'}\n"
    res = run_backtest(_req(code, [1.0, 1.0, 1.0]))
    assert isinstance(res.equity_curve, list)
    assert all(isinstance(v, float) for v in res.equity_curve)
    assert isinstance(res.events, list)
    assert res.snapshot is not None
    assert res.snapshot.code_sha256 != ""
    assert res.snapshot.bars_count == 3
    assert res.snapshot.ticks_count == 3  # synthesized from bars


# --- swap applied to cash ------------------------------------------------


def test_swap_reduces_equity_over_many_days():
    code = "def run(context):\n    return {'signal': 'hold'}\n"
    # Span 3 days; expect swap to kick in daily and reduce cash.
    day = 86_400_000
    bars = [
        Bar(open_time=i * day, close_time=(i + 1) * day,
            open=1.0, high=1.0, low=1.0, close=1.0, volume=0.0)
        for i in range(4)
    ]
    req = BacktestRequest(
        run_id="swap", user_id=1, account_id=1,
        symbol="EURUSD", timeframe="D1",
        start=datetime(2024, 1, 1, tzinfo=timezone.utc),
        end=datetime(2024, 1, 5, tzinfo=timezone.utc),
        initial_cash=10_000.0,
        bars=bars,
        strategy_code=code,
        cost_profile=CostProfile(swap_rate_per_rollover=0.001),
    )
    res = run_backtest(req)
    assert res.success is True
    # Equity strictly decreased below initial.
    assert res.equity_curve[-1] < 10_000.0
