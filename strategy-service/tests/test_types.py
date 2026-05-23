"""Tests for app/engine/types.py."""

from __future__ import annotations

from dataclasses import FrozenInstanceError
from datetime import datetime, timezone

import pytest

from app.engine import (
    Bar,
    BacktestRequest,
    BacktestResult,
    CloseReason,
    CostProfile,
    DataUnavailableError,
    DeadlineExceededError,
    EngineError,
    Fill,
    MarginCallError,
    Metrics,
    Order,
    OrderStatus,
    OrderType,
    Position,
    RiskAssessment,
    RunMode,
    RunSnapshot,
    Side,
    SlippageMode,
    StrategyCompileError,
    StrategyRuntimeError,
    Tick,
    Trade,
)


# --- enums ---------------------------------------------------------------


def test_side_values():
    assert Side.BUY.value == "buy"
    assert Side.SELL.value == "sell"


def test_order_type_covers_legacy_set():
    # Legacy engine supports 6 pending types + 2 market + cancel_pending.
    expected = {
        "buy", "sell",
        "buy_limit", "sell_limit",
        "buy_stop", "sell_stop",
        "buy_stop_limit", "sell_stop_limit",
        "close", "cancel_pending",
    }
    actual = {o.value for o in OrderType}
    assert expected.issubset(actual)


def test_order_status_values():
    names = {s.value for s in OrderStatus}
    assert {"pending", "activated", "filled", "cancelled", "expired", "rejected"}.issubset(names)


def test_run_mode_values():
    assert RunMode.ADVICE.value == "advice"
    assert RunMode.BACKTEST.value == "backtest"


def test_slippage_mode_values():
    assert {SlippageMode.FIXED.value, SlippageMode.RANDOM.value} == {"fixed", "random"}


def test_close_reason_values():
    names = {r.value for r in CloseReason}
    assert {"signal", "sl", "tp", "margin_call", "expired", "end_of_test"}.issubset(names)


# --- frozen dataclasses --------------------------------------------------


def test_bar_is_frozen():
    b = Bar(open_time=0, close_time=60_000, open=1.0, high=1.1, low=0.9, close=1.05)
    with pytest.raises(FrozenInstanceError):
        b.open = 2.0  # type: ignore[misc]


def test_tick_is_frozen():
    t = Tick(ts=1, bid=1.0, ask=1.0001)
    with pytest.raises(FrozenInstanceError):
        t.bid = 2.0  # type: ignore[misc]


def test_cost_profile_defaults_match_contract():
    c = CostProfile()
    assert c.commission_per_lot == 0.0
    assert c.slippage_mode == SlippageMode.FIXED
    assert c.slippage_rate == 0.0
    assert c.triple_swap_weekday == 3
    assert c.rollover_hour == 0
    assert c.server_timezone == "UTC"
    assert c.contract_size == 100_000.0
    assert c.pip_size == 0.0001


# --- mutable dataclasses defaults ---------------------------------------


def test_order_defaults():
    o = Order(id=1, type=OrderType.BUY_LIMIT, volume=1.0)
    assert o.status == OrderStatus.PENDING
    assert o.activated is False
    assert o.sl == 0.0 and o.tp == 0.0
    assert o.expiration is None


def test_fill_roundtrip_fields():
    f = Fill(order_id=1, ts=100, price=1.2345, volume=0.5, commission=0.1, slippage=0.0001)
    assert f.price == 1.2345
    assert f.volume == 0.5


def test_position_fields():
    p = Position(ticket=42, side=Side.BUY, volume=1.0, open_price=1.0, open_ts=100)
    assert p.side == Side.BUY


def test_trade_requires_reason():
    t = Trade(
        ticket=1,
        side=Side.SELL,
        volume=1.0,
        open_ts=0,
        open_price=1.0,
        close_ts=60,
        close_price=0.99,
        pnl=0.01,
        commission=0.0,
        reason=CloseReason.TP,
    )
    assert t.reason is CloseReason.TP


def test_metrics_default_zero():
    m = Metrics()
    assert m.total_return == 0.0
    assert m.total_trades == 0


def test_risk_assessment_default_lists_are_independent():
    r1 = RiskAssessment()
    r2 = RiskAssessment()
    r1.warnings.append("x")
    assert r2.warnings == []  # default_factory produced distinct lists


def test_run_snapshot_defaults():
    s = RunSnapshot()
    assert s.code_sha256 == ""
    assert s.bars_count == 0
    assert s.params == {}


# --- request / result ----------------------------------------------------


def _req() -> BacktestRequest:
    return BacktestRequest(
        run_id="r1",
        user_id=1,
        account_id=2,
        symbol="EURUSD",
        timeframe="H1",
        start=datetime(2024, 1, 1, tzinfo=timezone.utc),
        end=datetime(2024, 2, 1, tzinfo=timezone.utc),
        initial_cash=10_000.0,
    )


def test_backtest_request_defaults():
    r = _req()
    assert r.leverage == 0.0
    assert r.source == "MT_LIVE"
    assert r.single_position_only is True
    assert r.legacy_pnl is True
    assert r.max_fill_volume == 0.0
    assert r.deadline_ms == 120_000
    assert r.ticks is None
    assert r.bars == []
    assert isinstance(r.cost_profile, CostProfile)


def test_backtest_request_is_frozen():
    r = _req()
    with pytest.raises(FrozenInstanceError):
        r.initial_cash = 1.0  # type: ignore[misc]


def test_backtest_result_defaults_align_with_legacy_shape():
    res = BacktestResult(run_id="r1")
    # Must match legacy response shape: List[float] + List[dict] + metrics + risk.
    assert res.equity_curve == []
    assert res.events == []
    assert res.trades == []
    assert isinstance(res.metrics, Metrics)
    assert isinstance(res.risk_assessment, RiskAssessment)
    assert res.success is False


# --- exceptions ----------------------------------------------------------


@pytest.mark.parametrize(
    "cls",
    [
        StrategyCompileError,
        StrategyRuntimeError,
        DataUnavailableError,
        DeadlineExceededError,
        MarginCallError,
    ],
)
def test_engine_exceptions_inherit_base(cls):
    assert issubclass(cls, EngineError)
    with pytest.raises(EngineError):
        raise cls("boom")
