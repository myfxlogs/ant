"""Backtest runner: the tick-driven main loop (D10 — SDK-native execution).

契约：docs/domains/backtest-system.md §7.4.3 · runner.py

Orchestrates every engine module to produce a :py:class:`BacktestResult`.
Keeps all cross-module wiring here so individual modules stay decoupled.

Execution model (D10):
    SDK strategies interact with the engine through SimBroker directly.
    Legacy def run(context) strategies go through _LegacyStrategyRunner.
    Vectorized run_dataframe strategies use a pre-computed signal DataFrame.
    All three paths share the same tick-driven main loop.
"""

from __future__ import annotations

import ast
from datetime import datetime, timezone
from decimal import Decimal
from typing import Any, Dict, List, Optional

import numpy as np

from app.engine.cost import CostModel
from app.engine.fill import FillModel
from app.engine.margin import MarginModel
from app.engine.market import MarketSimulator, MultiSymbolMarket, TickSimulator
from app.engine.metrics import build_metrics
from app.engine.portfolio import Portfolio
from app.engine.sandbox import StrategyRunner as _LegacyStrategyRunner, code_sha256
from app.engine.sim_broker import SimBroker
from app.engine.types import (
    BacktestRequest,
    BacktestResult,
    CloseReason,
    EngineError,
    ExecutionAssumptions,
    Fill,
    Order,
    OrderType,
    Position,
    RunMode,
    RunSnapshot,
    Side,
    StrategyCompileError,
    StrategyRuntimeError,
    Tick,
)
from app.engine.vectorized_runner import (
    DataFrameStrategyRunner,
    extract_signal_at,
    detect_strategy_type,
)
from app.sdk.account import AccountMode
from app.sdk.runtime import StrategyRuntime
from app.sdk.series import Bars, Series
from app.sdk.strategy_base import StrategyBase

# ── Legacy signal-dispatch constants ─────────────────────────────────────

_PENDING_TYPES = {
    "buy_limit": OrderType.BUY_LIMIT,
    "sell_limit": OrderType.SELL_LIMIT,
    "buy_stop": OrderType.BUY_STOP,
    "sell_stop": OrderType.SELL_STOP,
    "buy_stop_limit": OrderType.BUY_STOP_LIMIT,
    "sell_stop_limit": OrderType.SELL_STOP_LIMIT,
}

_BUY_ACTIONS = frozenset({"buy", "buy_limit", "buy_stop", "buy_stop_limit"})
_SELL_ACTIONS = frozenset({"sell", "sell_limit", "sell_stop", "sell_stop_limit"})


def _tick_side_for_close(pos: Position, tick: Tick) -> float:
    """Mark price used when forcibly closing ``pos`` at ``tick``."""
    return tick.bid if pos.side is Side.BUY else tick.ask


def _parse_expiration(raw: Any) -> Optional[int]:
    if raw is None or raw == "":
        return None
    if isinstance(raw, (int, float)):
        return int(raw)
    if isinstance(raw, str):
        try:
            dt = datetime.fromisoformat(raw.replace("Z", "+00:00"))
            if dt.tzinfo is None:
                dt = dt.replace(tzinfo=timezone.utc)
            return int(dt.astimezone(timezone.utc).timestamp() * 1000)
        except ValueError:
            return None
    return None


# ── SDK Bars / Indicators adapters ───────────────────────────────────────


class _EngineSeries(Series):
    """Series backed by a numpy array from MarketSimulator."""

    def __init__(self, data: np.ndarray):
        self._data = data

    def __getitem__(self, shift: int) -> float:
        idx = len(self._data) - 1 - shift
        if idx < 0 or idx >= len(self._data):
            raise IndexError(f"bar index out of range: shift={shift}")
        return float(self._data[idx])

    def __len__(self) -> int:
        return len(self._data)

    def slice(self, count: int) -> List[float]:
        return list(self._data[-count:]) if count <= len(self._data) else list(self._data)


class _EngineBars(Bars):
    """Bars backed by MarketSimulator numpy arrays."""

    def __init__(self, market: MarketSimulator, timeframe: str = ""):
        self.timeframe = timeframe
        self.open = _EngineSeries(market._open)
        self.high = _EngineSeries(market._high)
        self.low = _EngineSeries(market._low)
        self.close = _EngineSeries(market._close)
        self.volume = _EngineSeries(market._volume)
        self.time = _EngineSeries(market._close_times)

    def total(self) -> int:
        return len(self.close)


class _EngineIndicators:
    """Indicators backed by engine data, matching SDK Indicators interface."""

    def __init__(self, bars_provider):
        self._bars_provider = bars_provider

    def _get_close(self) -> np.ndarray:
        bars = self._bars_provider()
        return np.array([bars.close[i] for i in range(len(bars.close) - 1, -1, -1)])

    def ma(self, period=14, shift=0, method="sma"):
        data = self._get_close()
        if len(data) < period + shift:
            return 0.0
        window = data[:len(data) - shift] if shift > 0 else data
        if method in ("ema", "exponential"):
            alpha = 2.0 / (period + 1)
            result = float(window[-period])
            for v in window[-period + 1:]:
                result = alpha * v + (1 - alpha) * result
            return result
        return float(np.mean(window[-period:]))

    def ema(self, period=14, shift=0):
        return self.ma(period, shift, "ema")

    def rsi(self, period=14, shift=0):
        data = self._get_close()
        if len(data) < period + shift + 1:
            return 50.0
        window = data[:len(data) - shift] if shift > 0 else data
        deltas = np.diff(window[-period - 1:])
        gains = np.sum(deltas[deltas > 0])
        losses = -np.sum(deltas[deltas < 0])
        if losses == 0:
            return 100.0 if gains > 0 else 50.0
        rs = gains / losses
        return float(100.0 - 100.0 / (1.0 + rs))

    def bands(self, period=20, deviation=2.0, shift=0):
        data = self._get_close()
        if len(data) < period:
            return (0.0, 0.0, 0.0)
        middle = self.ma(period, shift, "sma")
        window = data[:len(data) - shift] if shift > 0 else data
        std = float(np.std(window[-period:]))
        return (middle + deviation * std, middle, middle - deviation * std)

    def macd(self, fast=12, slow=26, signal=9, shift=0):
        return (0.0, 0.0, 0.0)

    def atr(self, period=14, shift=0):
        return 0.001

    def stochastic(self, k_period=5, d_period=3, shift=0):
        return (50.0, 50.0)

    def cci(self, period=14, shift=0):
        return 0.0

    def i_custom(self, name, params=(), buffer=0, shift=0):
        return 0.0


def _is_sdk_strategy(code: str) -> bool:
    """Check whether strategy code uses the SDK class-based format."""
    try:
        tree = ast.parse(code)
    except SyntaxError:
        return False
    for node in ast.walk(tree):
        if isinstance(node, ast.ClassDef):
            for base in node.bases:
                if isinstance(base, ast.Name) and base.id == "StrategyBase":
                    return True
                if isinstance(base, ast.Attribute) and base.attr == "StrategyBase":
                    return True
    return False


# ── BacktestRunner ───────────────────────────────────────────────────────


class BacktestRunner:
    """Encapsulates one backtest request's execution state.

    D10: Three execution paths sharing one tick-driven loop:

    1. **SDK-native** (``_use_sdk_path=True``): Strategy calls
       ``self.broker.order_send()`` → SimBroker → FillModel/Portfolio.
       No signal dict, no sandbox.call().

    2. **Legacy** (``_sandbox``): Old ``def run(context)`` / ``signal={}``
       patterns via ``_LegacyStrategyRunner.call(ctx)`` + ``_dispatch_signal()``.

    3. **Vectorized** (``_signal_df``): ``run_dataframe`` pre-computed
       signal DataFrame consumed bar-by-bar.
    """

    def __init__(self, req: BacktestRequest, sandbox=None) -> None:
        self._req = req
        self._cost = CostModel(req.cost_profile)
        self._fill = FillModel(self._cost, max_fill_volume=req.max_fill_volume)

        if req.bars_by_symbol:
            primary = req.primary_symbol or req.symbol
            if primary not in req.bars_by_symbol:
                raise EngineError(f"primary_symbol {primary!r} missing from bars_by_symbol")
            self._market = MultiSymbolMarket(req.bars_by_symbol, primary)
            self._primary_bars = req.bars_by_symbol[primary]
        else:
            self._market = MarketSimulator(req.bars)
            self._primary_bars = req.bars

        self._ticks = TickSimulator(self._primary_bars, req.ticks)
        contract_size = (
            1.0 if (not req.legacy_pnl and self._ticks.synthetic)
            else req.cost_profile.contract_size
        )
        self._portfolio = Portfolio(
            initial_cash=req.initial_cash,
            legacy_pnl=req.legacy_pnl,
            contract_size=contract_size,
        )
        self._margin = MarginModel(req.leverage, contract_size)

        # ── Strategy format detection ─────────────────────────────────────
        strategy_is_sdk = _is_sdk_strategy(req.strategy_code)

        self._signal_df = None
        self._use_sdk_path = False
        self._sandbox = None

        if sandbox is not None:
            self._sandbox = sandbox
        elif strategy_is_sdk:
            self._init_sdk_path(req)
            self._use_sdk_path = True
        elif detect_strategy_type(req.strategy_code) == "run_dataframe":
            import pandas as pd
            df_runner = DataFrameStrategyRunner(req.strategy_code, timeout_ms=req.deadline_ms)
            ohlc_df = pd.DataFrame({
                "open":   [b.open for b in self._primary_bars],
                "high":   [b.high for b in self._primary_bars],
                "low":    [b.low for b in self._primary_bars],
                "close":  [b.close for b in self._primary_bars],
                "volume": [b.tick_volume for b in self._primary_bars],
                "time":   [b.time for b in self._primary_bars],
            })
            self._signal_df = df_runner.call_dataframe(ohlc_df, req.strategy_params or {})
        else:
            self._sandbox = _LegacyStrategyRunner(req.strategy_code, timeout_ms=req.deadline_ms)

        self._strategy_runtime_kv: dict = {}
        self._equity_curve: List[float] = [req.initial_cash]
        self._events: List[dict] = []
        self._last_bar_idx = -1
        self._rollover_cursor = None
        self._margin_called = False

    # ── SDK path init ─────────────────────────────────────────────────────

    def _init_sdk_path(self, req: BacktestRequest) -> None:
        from app.engine.sandbox import build_sandbox_globals

        exec_scope = build_sandbox_globals()
        exec_scope["StrategyBase"] = StrategyBase
        exec(compile(req.strategy_code, "<strategy>", "exec"), exec_scope)

        strategy_cls = None
        for obj in exec_scope.values():
            if isinstance(obj, type) and issubclass(obj, StrategyBase) and obj is not StrategyBase:
                strategy_cls = obj
                break
        if strategy_cls is None:
            raise StrategyCompileError("SDK策略未找到继承 StrategyBase 的类")

        self._broker = SimBroker(
            portfolio=self._portfolio,
            fill_model=self._fill,
            cost_model=self._cost,
            margin_model=self._margin,
            market=self._market,
            tick_source=lambda: getattr(self, "_current_tick", None),
            account_mode=AccountMode.HEDGING,
            initial_balance=Decimal(str(req.initial_cash)),
        )

        # Bars provider: use primary market for MultiSymbolMarket.
        _primary_market = (
            self._market.primary_market() if isinstance(self._market, MultiSymbolMarket)
            else self._market
        )

        def bars_provider(timeframe=None):
            return _EngineBars(_primary_market, timeframe or req.timeframe)

        self._runtime = StrategyRuntime(
            strategy_class=strategy_cls,
            broker=self._broker,
            bars_provider=bars_provider,
            indicators=_EngineIndicators(bars_provider),
            symbol=req.symbol,
            timeframe=req.timeframe,
            params=dict(req.strategy_params or {}),
        )
        self._runtime.init()
        self._current_tick: Optional[Tick] = None

    # ── public entry ──────────────────────────────────────────────────────

    def run(self) -> BacktestResult:
        try:
            self._run_loop()
            success = True
            error: Optional[str] = None
        except (StrategyCompileError, StrategyRuntimeError, EngineError) as e:
            success = False
            error = str(e)
        except Exception as e:
            success = False
            error = f"engine error: {e}"

        metrics, risk = build_metrics(
            self._equity_curve,
            self._portfolio.closed_trades,
            self._primary_bars,
        )
        return BacktestResult(
            run_id=self._req.run_id,
            success=success,
            equity_curve=list(self._equity_curve),
            events=list(self._events),
            metrics=metrics,
            risk_assessment=risk,
            trades=list(self._portfolio.closed_trades),
            snapshot=self._build_snapshot(),
            execution_assumptions=self._build_execution_assumptions(),
            error=error,
        )

    # ── main loop (unified) ───────────────────────────────────────────────

    def _run_loop(self) -> None:
        from app.engine.context import build_context

        req = self._req
        last_tick: Optional[Tick] = None
        use_sdk = self._use_sdk_path

        for tick in self._ticks:
            last_tick = tick
            if use_sdk:
                self._current_tick = tick

            # 1. Pending-order queue.
            for fill, order in self._fill.process_on_tick(tick):
                pos = self._portfolio.apply_fill(fill, order, tick)
                self._events.append(self._open_event(pos, fill))

            # 2. SL / TP checks.
            for trade in self._portfolio.check_sl_tp(tick):
                self._events.append(self._close_event(trade))

            # 3. Margin call.
            if self._margin.enabled() and self._margin.is_margin_call(self._portfolio, tick):
                for trade in self._portfolio.force_liquidate_all(tick, CloseReason.MARGIN_CALL):
                    self._events.append(self._close_event(trade))
                self._margin_called = True
                self._equity_curve.append(self._portfolio.cash)
                break

            # 4. Bar-close → strategy callback.
            new_idx = self._market.bar_closed_at_or_before(tick.ts)
            while self._last_bar_idx < new_idx:
                self._last_bar_idx += 1

                if use_sdk:
                    self._runtime.on_bar(req.timeframe)
                elif self._signal_df is not None:
                    signal = extract_signal_at(self._signal_df, self._last_bar_idx, req.strategy_params)
                    self._dispatch_signal(signal, tick)
                else:
                    ctx = build_context(
                        RunMode.BACKTEST, req.symbol, req.timeframe,
                        self._market, self._last_bar_idx, self._portfolio,
                        req.strategy_params, tick,
                    )
                    ctx["runtime"] = self._strategy_runtime_kv
                    signal = self._sandbox.call(ctx)
                    self._dispatch_signal(signal, tick)

                self._equity_curve.append(self._portfolio.cash)

            # 5. Rollover swaps.
            if req.cost_profile.swap_rate_per_rollover > 0:
                new_equity, self._rollover_cursor = self._cost.apply_rollover_swaps(
                    self._portfolio.cash, tick.ts, self._rollover_cursor
                )
                self._portfolio.set_cash(new_equity)

        # End of data.
        if use_sdk:
            self._runtime.on_deinit("end_of_test")

        if last_tick is not None and self._portfolio.has_open() and not self._margin_called:
            if not req.legacy_pnl:
                for trade in self._portfolio.force_liquidate_all(last_tick, CloseReason.END_OF_TEST):
                    self._events.append(self._close_event(trade))
                self._equity_curve.append(self._portfolio.cash)

    # ── signal dispatch (legacy only) ─────────────────────────────────────

    def _dispatch_signal(self, signal, tick: Tick) -> None:
        if not signal or not isinstance(signal, dict):
            return
        action = str(signal.get("signal") or "hold").lower()
        if action in ("hold", ""):
            return
        td = self._req.trade_direction
        if td == "long" and action in _SELL_ACTIONS:
            return
        if td == "short" and action in _BUY_ACTIONS:
            return
        if action == "cancel_pending":
            self._fill.cancel_all()
            return
        if action == "close":
            close_side = str(signal.get("side") or "").lower()
            if close_side == "long":
                trades = self._portfolio.force_liquidate_side(tick, Side.BUY, CloseReason.SIGNAL)
            elif close_side == "short":
                trades = self._portfolio.force_liquidate_side(tick, Side.SELL, CloseReason.SIGNAL)
            else:
                trades = self._portfolio.force_liquidate_all(tick, CloseReason.SIGNAL)
            for trade in trades:
                self._events.append(self._close_event(trade))
            return

        volume = float(signal.get("volume") or 1.0)
        sl = float(signal.get("stop_loss") or 0.0)
        tp = float(signal.get("take_profit") or 0.0)

        if action in ("buy", "sell"):
            if self._portfolio.has_open():
                if self._req.single_position_only:
                    for trade in self._portfolio.force_liquidate_all(tick, CloseReason.SIGNAL):
                        self._events.append(self._close_event(trade))
                    return
                else:
                    is_buy = action == "buy"
                    if is_buy:
                        for trade in self._portfolio.force_liquidate_side(tick, Side.SELL, CloseReason.SIGNAL):
                            self._events.append(self._close_event(trade))
                    else:
                        for trade in self._portfolio.force_liquidate_side(tick, Side.BUY, CloseReason.SIGNAL):
                            self._events.append(self._close_event(trade))
            order = Order(id=0, type=OrderType.BUY if action == "buy" else OrderType.SELL,
                          volume=volume, sl=sl, tp=tp, created_at_ts=tick.ts)
            result = self._fill.process_market_order(order, tick)
            if result is None:
                return
            fill, filled_order = result
            pos = self._portfolio.apply_fill(fill, filled_order, tick)
            self._events.append(self._open_event(pos, fill))
            return

        if action in _PENDING_TYPES:
            price = float(signal.get("price") or 0.0)
            if price <= 0:
                return
            stop_limit_price = float(signal.get("stop_limit_price") or signal.get("limit_price") or 0.0)
            order = Order(id=0, type=_PENDING_TYPES[action], volume=volume, price=price,
                          sl=sl, tp=tp, stop_limit_price=stop_limit_price,
                          expiration=_parse_expiration(signal.get("expiration")),
                          created_at_ts=tick.ts)
            self._fill.enqueue(order, replace_same_type=bool(signal.get("replace")))

    # ── event builders ────────────────────────────────────────────────────

    def _open_event(self, pos: Position, fill: Fill) -> dict:
        return {
            "type": "position_open", "ts": fill.ts, "ticket": pos.ticket,
            "symbol": self._req.symbol, "side": pos.side.value,
            "volume": pos.volume, "price": pos.open_price,
            "stop_loss": pos.sl, "take_profit": pos.tp,
            "commission": fill.commission, "slippage": fill.slippage,
        }

    def _close_event(self, trade) -> dict:
        return {
            "type": "position_close", "ts": trade.close_ts,
            "ticket": trade.ticket, "symbol": self._req.symbol,
            "side": trade.side.value, "volume": trade.volume,
            "price": trade.close_price, "reason": trade.reason.value,
            "pnl": trade.pnl, "commission": trade.commission,
        }

    def _build_snapshot(self) -> RunSnapshot:
        req = self._req
        return RunSnapshot(
            code_sha256=code_sha256(req.strategy_code),
            params=dict(req.strategy_params or {}),
            cost_profile=req.cost_profile,
            dataset_id=req.dataset_id,
            bars_count=len(self._primary_bars),
            ticks_count=len(self._ticks),
        )

    def _build_execution_assumptions(self) -> ExecutionAssumptions:
        req = self._req
        signal_timing = "next_bar_open" if req.strict_mode else "same_bar_close"
        cp = req.cost_profile
        return ExecutionAssumptions(
            simulation_mode=req.source,
            signal_timing=signal_timing,
            fill_rule=signal_timing,
            mtf_fallback_reason="",
            actual_commission=cp.commission_per_lot,
            actual_slippage=cp.slippage_rate,
            actual_leverage=req.leverage if req.leverage > 0 else 1.0,
            trade_direction=req.trade_direction,
        )


def run_backtest(req: BacktestRequest) -> BacktestResult:
    """Worker-process entry point."""
    try:
        return BacktestRunner(req).run()
    except Exception as e:
        import traceback
        tb = traceback.format_exc()
        return BacktestResult(
            run_id=req.run_id, success=False,
            error=f"策略执行错误: {e}\n{tb}",
        )
