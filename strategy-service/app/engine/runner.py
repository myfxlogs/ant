"""Backtest runner: the tick-driven main loop (D10 — SDK-native execution).

Two execution paths share the same tick-driven loop:

1. **SDK-native**: Strategy calls ``self.broker.order_send()`` → SimBroker
2. **Vectorized**: ``run_dataframe`` pre-computed signal DataFrame

Legacy ``def run(context)`` / ``signal = {...}`` is no longer supported.
"""

from __future__ import annotations

import ast
from datetime import datetime, timezone
from decimal import Decimal
from typing import Any, Callable, Dict, List, Optional

import numpy as np

from app.engine.cost import CostModel
from app.engine.fill import FillModel
from app.engine.margin import MarginModel
from app.engine.market import MarketSimulator, MultiSymbolMarket, TickSimulator
from app.engine.metrics import build_metrics
from app.engine.portfolio import Portfolio
from app.engine.sandbox import code_sha256
from app.engine.sim_broker import SimBroker, _derive_symbol_info_from_bars
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
from app.engine.sdk_indicators import SDKIndicators
from app.engine.sdk_loader import load_sdk_strategy
from app.sdk.account import AccountMode
from app.sdk.runtime import StrategyRuntime
from app.sdk.series import Bars, Series
from app.sdk.symbol import SymbolInfo

# ── Signal-dispatch constants (vectorized path only) ─────────────────────

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
    def __init__(
        self,
        market: MarketSimulator,
        timeframe: str = "",
        slice_end: int | None = None,
    ):
        """Wrap market data as SDK Bars.

        If *slice_end* is given, only bars ``[0..slice_end]`` (inclusive) are
        visible — this prevents look-ahead bias during tick-by-tick backtesting.
        When *slice_end* is ``None`` (the default), all bars in the market are
        visible (suitable for live trading where data is naturally bounded).
        """
        self.timeframe = timeframe
        if slice_end is not None:
            end = slice_end + 1
            self.open = _EngineSeries(market._open[:end])
            self.high = _EngineSeries(market._high[:end])
            self.low = _EngineSeries(market._low[:end])
            self.close = _EngineSeries(market._close[:end])
            self.volume = _EngineSeries(market._volume[:end])
            self.time = _EngineSeries(market._close_times[:end])
        else:
            self.open = _EngineSeries(market._open)
            self.high = _EngineSeries(market._high)
            self.low = _EngineSeries(market._low)
            self.close = _EngineSeries(market._close)
            self.volume = _EngineSeries(market._volume)
            self.time = _EngineSeries(market._close_times)

    def total(self) -> int:
        return len(self.close)


class _BarsProvider:
    """Time-sliced Bars provider for SDK strategy backtesting.

    Created once per backtest run and called by the strategy via
    ``ctx.bars(timeframe)`` on each tick.  Slices the underlying market
    data to the current bar index so that the strategy sees exactly the
    bars that have completed — no look-ahead bias.

    Design::

        provider = _BarsProvider(market, get_bar_idx, primary_tf)
        bars = provider(timeframe="H1")  # sliced view

    The *get_bar_idx* callback returns the current bar index (updated
    by the runner loop).  We inject a callback rather than capturing
    ``BacktestRunner`` so the provider is independently testable.
    """

    def __init__(
        self,
        market: MarketSimulator | MultiSymbolMarket,
        get_bar_idx: Callable[[], int],
        primary_timeframe: str,
    ) -> None:
        self._get_bar_idx = get_bar_idx
        self._primary_timeframe = primary_timeframe
        # Resolve the primary MarketSimulator once; secondary-symbol
        # bars would need a separate provider keyed by symbol (Phase B2).
        if isinstance(market, MultiSymbolMarket):
            self._primary = market.primary_market()
        else:
            self._primary = market

    def __call__(self, timeframe: str | None = None) -> _EngineBars:
        end = self._get_bar_idx() + 1
        return _EngineBars(
            self._primary,
            timeframe or self._primary_timeframe,
            slice_end=end,
        )


def _symbol_info_from_dict(symbol: str, d: dict) -> SymbolInfo:
    """Convert a proto SymbolInfo dict (via MessageToDict) to SDK SymbolInfo.

    Zero values from the broker are treated as "not provided" and replaced
    with sensible defaults — MT4 connections may not supply volume limits.
    """
    def _decimal(key: str, fallback: str) -> Decimal:
        v = d.get(key)
        if v is None:
            return Decimal(fallback)
        dv = Decimal(str(v))
        return dv if dv > 0 else Decimal(fallback)

    return SymbolInfo(
        name=symbol,
        digits=int(d.get("digits", 5)),
        point=_decimal("point", "0.00001"),
        tick_size=_decimal("tick_size", "0.00001"),
        tick_value=_decimal("tick_value", "1.0"),
        contract_size=_decimal("contract_size", "1"),
        volume_min=_decimal("volume_min", "0.01"),
        volume_max=_decimal("volume_max", "100"),
        volume_step=_decimal("volume_step", "0.01"),
        stops_level=int(d.get("stops_level", 0)),
        freeze_level=int(d.get("freeze_level", 0)),
        swap_long=_decimal("swap_long", "0"),
        swap_short=_decimal("swap_short", "0"),
        margin_rate=_decimal("margin_rate", "0.01"),
    )


def _is_sdk_strategy(code: str) -> bool:
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
    """SDK-native backtest execution (D10).

    Two paths sharing one tick-driven loop:

    1. **SDK-native**: Strategy calls ``self.broker.order_send()`` → SimBroker
    2. **Vectorized**: Pre-computed ``run_dataframe`` signal DataFrame
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
            initial_cash=req.initial_cash, legacy_pnl=req.legacy_pnl,
            contract_size=contract_size,
        )
        self._margin = MarginModel(req.leverage, contract_size)

        strategy_is_sdk = _is_sdk_strategy(req.strategy_code)
        self._signal_df = None
        self._use_sdk_path = False

        if sandbox is not None:
            # Caller-provided sandbox (test hooks).
            self._sandbox = sandbox
        elif strategy_is_sdk:
            self._init_sdk_path(req)
            self._use_sdk_path = True
        elif detect_strategy_type(req.strategy_code) == "run_dataframe":
            import pandas as pd
            df_runner = DataFrameStrategyRunner(req.strategy_code, timeout_ms=req.deadline_ms)
            self._signal_df = df_runner.call_dataframe(pd.DataFrame({
                "open": [b.open for b in self._primary_bars],
                "high": [b.high for b in self._primary_bars],
                "low": [b.low for b in self._primary_bars],
                "close": [b.close for b in self._primary_bars],
                "volume": [b.tick_volume for b in self._primary_bars],
                "time": [b.time for b in self._primary_bars],
            }), req.strategy_params or {})
        else:
            raise StrategyCompileError(
                "只支持 SDK 策略或 run_dataframe 策略。"
                "请使用继承 StrategyBase 的类定义来编写策略。"
            )

        self._equity_curve: List[float] = [req.initial_cash]
        self._events: List[dict] = []
        self._last_bar_idx = -1
        self._rollover_cursor = None
        self._margin_called = False

    # ── SDK path init ─────────────────────────────────────────────────────

    def _init_sdk_path(self, req: BacktestRequest) -> None:
        strategy_cls = load_sdk_strategy(req.strategy_code)

        # Broker-provided SymbolInfo takes priority; fall back to data-driven
        # derivation from K-line prices when no MT gateway data is available.
        if req.symbol_info:
            symbol_info = _symbol_info_from_dict(req.symbol, req.symbol_info)
        else:
            symbol_info = _derive_symbol_info_from_bars(req.symbol, self._primary_bars)
        symbol_info_map = {req.symbol: symbol_info}

        self._broker = SimBroker(
            portfolio=self._portfolio, fill_model=self._fill,
            cost_model=self._cost, margin_model=self._margin,
            market=self._market,
            tick_source=lambda: getattr(self, "_current_tick", None),
            account_mode=AccountMode.HEDGING,
            symbol_info_map=symbol_info_map,
            initial_balance=Decimal(str(req.initial_cash)),
        )

        bars_provider = _BarsProvider(
            market=self._market,
            get_bar_idx=lambda: self._last_bar_idx,
            primary_timeframe=req.timeframe,
        )

        backtest_config = {
            "initial_capital": req.initial_cash,
            "leverage": req.leverage,
            "commission": req.cost_profile.commission_per_lot,
            "slippage": req.cost_profile.slippage_rate,
            "trade_direction": req.trade_direction,
            "strict_mode": req.strict_mode,
        }
        self._runtime = StrategyRuntime(
            strategy_class=strategy_cls, broker=self._broker,
            bars_provider=bars_provider,
            indicators=SDKIndicators(bars_provider),
            symbol=req.symbol, timeframe=req.timeframe,
            params=dict(req.strategy_params or {}),
            backtest_config=backtest_config,
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
            self._equity_curve, self._portfolio.closed_trades, self._primary_bars,
        )
        return BacktestResult(
            run_id=self._req.run_id, success=success,
            equity_curve=list(self._equity_curve), events=list(self._events),
            metrics=metrics, risk_assessment=risk,
            trades=list(self._portfolio.closed_trades),
            snapshot=self._build_snapshot(),
            execution_assumptions=self._build_execution_assumptions(),
            error=error,
        )

    # ── main loop ─────────────────────────────────────────────────────────

    def _run_loop(self) -> None:
        req = self._req
        last_tick: Optional[Tick] = None
        use_sdk = self._use_sdk_path

        for tick in self._ticks:
            last_tick = tick
            if use_sdk:
                self._current_tick = tick

            # 0. SDK tick callback (MQL OnTick equivalent).
            if use_sdk:
                self._runtime.on_tick()

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

                self._equity_curve.append(self._portfolio.cash)

            # 5. Rollover swaps.
            if req.cost_profile.swap_rate_per_rollover > 0:
                new_equity, self._rollover_cursor = self._cost.apply_rollover_swaps(
                    self._portfolio.cash, tick.ts, self._rollover_cursor
                )
                self._portfolio.set_cash(new_equity)

        if use_sdk:
            self._runtime.deinit("end_of_test")

        if last_tick is not None and self._portfolio.has_open() and not self._margin_called:
            if not req.legacy_pnl:
                for trade in self._portfolio.force_liquidate_all(last_tick, CloseReason.END_OF_TEST):
                    self._events.append(self._close_event(trade))
                self._equity_curve.append(self._portfolio.cash)

    # ── signal dispatch (vectorized path) ─────────────────────────────────

    def _dispatch_signal(self, signal, tick: Tick) -> None:
        if not signal or not isinstance(signal, dict):
            return
        action = str(signal.get("signal") or "hold").lower()
        if action in ("hold", ""):
            return
        td = self._req.trade_direction
        if td == "long" and action in _SELL_ACTIONS: return
        if td == "short" and action in _BUY_ACTIONS: return
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
            if result is None: return
            fill, filled_order = result
            pos = self._portfolio.apply_fill(fill, filled_order, tick)
            self._events.append(self._open_event(pos, fill))
            return

        if action in _PENDING_TYPES:
            price = float(signal.get("price") or 0.0)
            if price <= 0: return
            stop_limit_price = float(
                signal.get("stop_limit_price") or signal.get("limit_price") or 0.0)
            order = Order(id=0, type=_PENDING_TYPES[action], volume=volume,
                          price=price, sl=sl, tp=tp, stop_limit_price=stop_limit_price,
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

    def _build_snapshot(self):
        req = self._req
        return RunSnapshot(
            code_sha256=code_sha256(req.strategy_code),
            params=dict(req.strategy_params or {}),
            cost_profile=req.cost_profile,
            dataset_id=req.dataset_id,
            bars_count=len(self._primary_bars),
            ticks_count=len(self._ticks),
        )

    def _build_execution_assumptions(self):
        req = self._req
        return ExecutionAssumptions(
            simulation_mode=req.source,
            signal_timing="next_bar_open" if req.strict_mode else "same_bar_close",
            fill_rule="next_bar_open" if req.strict_mode else "same_bar_close",
            mtf_fallback_reason="",
            actual_commission=req.cost_profile.commission_per_lot,
            actual_slippage=req.cost_profile.slippage_rate,
            actual_leverage=req.leverage if req.leverage > 0 else 1.0,
            trade_direction=req.trade_direction,
        )


def run_backtest(req: BacktestRequest) -> BacktestResult:
    """Worker-process entry point."""
    try:
        return BacktestRunner(req).run()
    except Exception as e:
        import traceback
        return BacktestResult(
            run_id=req.run_id, success=False,
            error=f"策略执行错误: {e}\n{traceback.format_exc()}",
        )
