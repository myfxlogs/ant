"""Tests for app/engine/cost.py."""

from __future__ import annotations

from datetime import datetime, timezone

import pytest

from app.engine.cost import CostModel
from app.engine.types import CostProfile, SlippageMode


def _profile(**over) -> CostProfile:
    base = dict(
        commission_per_lot=0.0,
        slippage_mode=SlippageMode.FIXED,
        slippage_rate=0.0,
        slippage_seed=0,
        swap_rate_per_rollover=0.0,
        triple_swap_weekday=3,
        rollover_hour=0,
        server_timezone="UTC",
    )
    base.update(over)
    return CostProfile(**base)


# --- commission ----------------------------------------------------------


def test_commission_zero_when_rate_zero():
    assert CostModel(_profile()).commission(1.0) == 0.0


def test_commission_scales_with_volume():
    m = CostModel(_profile(commission_per_lot=7.0))
    assert m.commission(2.5) == 17.5


def test_commission_negative_rate_clamped_to_zero():
    # Legacy behaviour: max(0, rate) * vol.
    m = CostModel(_profile(commission_per_lot=-1.0))
    assert m.commission(10.0) == 0.0


# --- slippage ------------------------------------------------------------


def test_slippage_zero_rate_returns_input_unchanged():
    m = CostModel(_profile())
    assert m.apply_slippage(1.2345, is_buy_side=True) == 1.2345
    assert m.apply_slippage(1.2345, is_buy_side=False) == 1.2345


def test_slippage_fixed_buy_raises_price():
    m = CostModel(_profile(slippage_rate=0.01))
    assert m.apply_slippage(1.0, is_buy_side=True) == pytest.approx(1.01)


def test_slippage_fixed_sell_lowers_price():
    m = CostModel(_profile(slippage_rate=0.01))
    assert m.apply_slippage(1.0, is_buy_side=False) == pytest.approx(0.99)


def test_slippage_random_uses_seed_for_reproducibility():
    p = _profile(slippage_mode=SlippageMode.RANDOM, slippage_rate=0.01, slippage_seed=42)
    a = CostModel(p)
    b = CostModel(p)
    # First two calls on independent instances with the same seed → same output.
    assert a.apply_slippage(1.0, True) == b.apply_slippage(1.0, True)
    assert a.apply_slippage(1.0, True) == b.apply_slippage(1.0, True)


def test_slippage_random_bounded_by_rate():
    m = CostModel(_profile(slippage_mode=SlippageMode.RANDOM, slippage_rate=0.01, slippage_seed=1))
    # 50 samples: buy side should always be in [1.0, 1.01].
    for _ in range(50):
        v = m.apply_slippage(1.0, True)
        assert 1.0 <= v <= 1.01


# --- rollover swaps ------------------------------------------------------


def _dt_ms(y, m, d, h=0, mi=0) -> int:
    return int(datetime(y, m, d, h, mi, tzinfo=timezone.utc).timestamp() * 1000)


def test_swaps_disabled_when_rate_zero_returns_equity_unchanged():
    m = CostModel(_profile())
    eq, cur = m.apply_rollover_swaps(10_000.0, _dt_ms(2024, 1, 1, 12), None)
    assert eq == 10_000.0
    # Cursor still populated so runner can keep state.
    assert cur is not None


def test_single_rollover_applies_1x_on_non_triple_weekday():
    # 2024-01-02 is a Tuesday. Triple day default = 3 (Wednesday).
    m = CostModel(_profile(swap_rate_per_rollover=0.0001, rollover_hour=0))
    ts_before = _dt_ms(2024, 1, 1, 23, 59)   # before 2024-01-02 00:00 UTC
    ts_after = _dt_ms(2024, 1, 2, 0, 1)      # after rollover once
    eq1, cur1 = m.apply_rollover_swaps(10_000.0, ts_before, None)
    # No rollover hit yet → equity unchanged (cursor is future).
    assert eq1 == 10_000.0
    eq2, _ = m.apply_rollover_swaps(eq1, ts_after, cur1)
    assert eq2 == pytest.approx(10_000.0 * (1 - 0.0001))


def test_triple_swap_weekday_hits_3x_multiplier():
    # Pick a Wednesday: 2024-01-03.
    m = CostModel(_profile(swap_rate_per_rollover=0.001, triple_swap_weekday=2, rollover_hour=0))
    # Cursor initialized at 2024-01-03 00:00 UTC. Advance one step past it.
    eq, _ = m.apply_rollover_swaps(10_000.0, _dt_ms(2024, 1, 3, 0, 1), None)
    # Wait: on first call cursor = next_rollover AFTER ts. If ts is 00:01 on Jan 3,
    # the next rollover is 2024-01-04 00:00 (Thursday), so no swap yet.
    assert eq == 10_000.0


def test_multiple_rollovers_accumulate():
    m = CostModel(_profile(swap_rate_per_rollover=0.0001, rollover_hour=0))
    # Start cursor below 2024-01-02 00:00, tick far in the future to force N rollovers.
    start = _dt_ms(2024, 1, 1, 0, 1)
    _, cur = m.apply_rollover_swaps(10_000.0, start, None)
    # Now jump four days.
    end = _dt_ms(2024, 1, 5, 12, 0)
    eq, _ = m.apply_rollover_swaps(10_000.0, end, cur)
    # 3 rollovers hit (01-02 Tue, 01-03 Wed triple, 01-04 Thu, 01-05 Fri)
    # Actually 4 rollovers: cursor starts at 01-02 00:00, we cross 01-02, 01-03, 01-04, 01-05.
    # Mults: Tue=1, Wed=3, Thu=1, Fri=1 → equity *= (1-r)(1-3r)(1-r)(1-r)
    r = 0.0001
    expected = 10_000.0 * (1 - r) * (1 - 3 * r) * (1 - r) * (1 - r)
    assert eq == pytest.approx(expected)


def test_server_timezone_shifts_rollover_instant():
    # With server_timezone=Asia/Shanghai (UTC+8) and rollover_hour=0,
    # rollover at local 00:00 on 2024-01-02 is UTC 2024-01-01 16:00.
    m = CostModel(
        _profile(
            swap_rate_per_rollover=0.01,
            server_timezone="Asia/Shanghai",
            rollover_hour=0,
        )
    )
    # Tick just after 2024-01-01 16:00 UTC should already have triggered one swap.
    ts = _dt_ms(2024, 1, 1, 16, 1)
    # First call seeds the cursor; equity must not change yet.
    eq, cur = m.apply_rollover_swaps(10_000.0, _dt_ms(2024, 1, 1, 15, 0), None)
    assert eq == 10_000.0
    eq2, _ = m.apply_rollover_swaps(eq, ts, cur)
    # One rollover fired → equity *= (1 - 0.01 * mult).
    assert eq2 < 10_000.0
