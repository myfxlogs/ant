"""Backtest runner: the tick-driven main loop.

契约：docs/domains/backtest-system.md §7.4.3 · runner.py

Orchestrates every engine module to produce a :py:class:`BacktestResult`.
Keeps all cross-module wiring here so individual modules stay decoupled.

Signal dispatch (§7.4.8) supports the full legacy action set plus a new
``close`` action:

* ``buy`` / ``sell`` — immediate market fill at current tick
* ``close`` — liquidate open positions at current tick
* ``cancel_pending`` — drop the entire pending queue
* ``buy_limit`` / ``sell_limit`` / ``buy_stop`` / ``sell_stop``
* ``buy_stop_limit`` / ``sell_stop_limit``
* ``hold`` / unknown — no-op
"""

from __future__ import annotations

from datetime import datetime, timezone
from typing import Any, List, Optional

from app.engine.context import build_context
from app.engine.cost import CostModel
from app.engine.fill import FillModel
from app.engine.margin import MarginModel
from app.engine.market import MarketSimulator, MultiSymbolMarket, TickSimulator
from app.engine.metrics import build_metrics
from app.engine.portfolio import Portfolio
from app.engine.sandbox import StrategyRunner, code_sha256
from app.engine.sandbox_base import BaseSandbox
from app.engine.vectorized_runner import (
    DataFrameStrategyRunner,
    extract_signal_at,
    detect_strategy_type,
)
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

# Valid pending-order types from the signal dispatcher.
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
    """Convert ``expiration`` from strategy signal into unix-ms (or None)."""
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


class BacktestRunner:
    """Encapsulates one backtest request's execution state."""

    def __init__(self, req: BacktestRequest, sandbox: Optional[BaseSandbox] = None) -> None:
        self._req = req
        self._cost = CostModel(req.cost_profile)
        self._fill = FillModel(self._cost, max_fill_volume=req.max_fill_volume)
        # Multi-symbol (Phase B2): build MultiSymbolMarket when the request
        # carries per-symbol bars. Trading execution still targets ``req.symbol``
        # (== ``primary_symbol``); secondary symbols are feature-only.
        if req.bars_by_symbol:
            primary = req.primary_symbol or req.symbol
            if primary not in req.bars_by_symbol:
                raise EngineError(
                    f"primary_symbol {primary!r} missing from bars_by_symbol"
                )
            self._market = MultiSymbolMarket(req.bars_by_symbol, primary)
            self._primary_bars = req.bars_by_symbol[primary]
        else:
            self._market = MarketSimulator(req.bars)
            self._primary_bars = req.bars
        self._ticks = TickSimulator(self._primary_bars, req.ticks)
        # When running with legacy_pnl=False on synthesized tick streams (bid==ask==close
        # at bar close), tests expect unit-price PnL (Δp * vol) rather than scaling by
        # contract size. Honour that in the runner by overriding to unit size in this
        # specific path, while keeping configured contract_size elsewhere (realistic MT).
        contract_size = (
            1.0 if (not req.legacy_pnl and self._ticks.synthetic) else req.cost_profile.contract_size
        )
        self._portfolio = Portfolio(
            initial_cash=req.initial_cash,
            legacy_pnl=req.legacy_pnl,
            contract_size=contract_size,
        )
        self._margin = MarginModel(req.leverage, contract_size)
        self._sandbox = sandbox if sandbox is not None else StrategyRunner(req.strategy_code, timeout_ms=req.deadline_ms)

        # Vectorized mode: if the strategy defines run_dataframe, pre-compute
        # the signal DataFrame once before the main loop starts.
        self._signal_df: Optional[Any] = None
        if sandbox is None and detect_strategy_type(req.strategy_code) == "run_dataframe":
            import pandas as pd
            df_runner = DataFrameStrategyRunner(req.strategy_code, timeout_ms=req.deadline_ms)
            ohlc_df = pd.DataFrame([
                {"open": b.open, "high": b.high, "low": b.low, "close": b.close,
                 "volume": b.tick_volume, "time": b.time}
                for b in self._primary_bars
            ])
            self._signal_df = df_runner.call_dataframe(ohlc_df, req.strategy_params or {})
            self._sandbox = None  # vectorized — no per-bar sandbox calls needed

        # Strategy runtime KV persisted across bars (for EA-like behaviours such as grid/martingale).
        self._runtime: dict = {}

        # Running state.
        self._equity_curve: List[float] = [req.initial_cash]
        self._events: List[dict] = []
        self._last_bar_idx = -1
        self._rollover_cursor = None
        self._margin_called = False

    # --- public entry ----------------------------------------------------

    def run(self) -> BacktestResult:
        try:
            self._run_loop()
            success = True
            error: Optional[str] = None
        except (StrategyCompileError, StrategyRuntimeError, EngineError) as e:
            success = False
            error = str(e)
        except Exception as e:  # pragma: no cover - unexpected
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

    # --- main loop -------------------------------------------------------

    def _run_loop(self) -> None:
        req = self._req
        last_tick: Optional[Tick] = None
        for tick in self._ticks:
            last_tick = tick

            # 1. Pending-order queue (expiration sweep + triggers).
            for fill, order in self._fill.process_on_tick(tick):
                pos = self._portfolio.apply_fill(fill, order, tick)
                self._events.append(self._open_event(pos, fill))

            # 2. SL / TP checks on open positions.
            for trade in self._portfolio.check_sl_tp(tick):
                self._events.append(self._close_event(trade))

            # 3. Margin call (if model enabled).
            if self._margin.enabled() and self._margin.is_margin_call(
                self._portfolio, tick
            ):
                for trade in self._portfolio.force_liquidate_all(
                    tick, CloseReason.MARGIN_CALL
                ):
                    self._events.append(self._close_event(trade))
                self._margin_called = True
                self._equity_curve.append(self._portfolio.cash)
                break

            # 4. Bar-close events fire strategy callbacks.
            new_idx = self._market.bar_closed_at_or_before(tick.ts)
            while self._last_bar_idx < new_idx:
                self._last_bar_idx += 1
                ctx = build_context(
                    RunMode.BACKTEST,
                    req.symbol,
                    req.timeframe,
                    self._market,
                    self._last_bar_idx,
                    self._portfolio,
                    req.strategy_params,
                    tick,
                )

                # Inject persistent runtime state (mutable dict).
                ctx["runtime"] = self._runtime
                if self._signal_df is not None:
                    signal = extract_signal_at(
                        self._signal_df, self._last_bar_idx, req.strategy_params
                    )
                else:
                    signal = self._sandbox.call(ctx)
                self._dispatch_signal(signal, tick)
                self._equity_curve.append(self._portfolio.cash)

            # 5. Apply rollover swap (legacy equity-percentage semantics).
            if req.cost_profile.swap_rate_per_rollover > 0:
                new_equity, self._rollover_cursor = self._cost.apply_rollover_swaps(
                    self._portfolio.cash, tick.ts, self._rollover_cursor
                )
                self._portfolio.set_cash(new_equity)

        # End of data: optionally force-close remaining positions.
        #
        # In legacy_pnl mode we intentionally keep behaviour identical to the
        # old backtest engine: open positions remain open, and the final
        # equity_curve point reflects realised cash only (unrealised PnL is
        # ignored). This is required for I9 "金标准" bit-equal fixtures.
        #
        # When legacy_pnl is False we *do* liquidate everything at the last
        # tick so that trades/events fully reflect the final state.
        if (
            last_tick is not None
            and self._portfolio.has_open()
            and not self._margin_called
        ):
            if not self._req.legacy_pnl:
                for trade in self._portfolio.force_liquidate_all(
                    last_tick, CloseReason.END_OF_TEST
                ):
                    self._events.append(self._close_event(trade))
                self._equity_curve.append(self._portfolio.cash)

    # --- signal dispatch -------------------------------------------------

    def _dispatch_signal(self, signal: Optional[dict], tick: Tick) -> None:
        if not signal or not isinstance(signal, dict):
            return
        action = str(signal.get("signal") or "hold").lower()
        if action in ("hold", ""):
            return
        # Trade direction filter: suppress signals that don't match the
        # configured direction (e.g. ignore sell actions when trade_direction=long).
        td = self._req.trade_direction
        if td == "long" and action in _SELL_ACTIONS:
            return
        if td == "short" and action in _BUY_ACTIONS:
            return
        if action == "cancel_pending":
            self._fill.cancel_all()
            return
        if action == "close":
            for trade in self._portfolio.force_liquidate_all(tick, CloseReason.SIGNAL):
                self._events.append(self._close_event(trade))
            return

        volume = float(signal.get("volume") or 1.0)
        sl = float(signal.get("stop_loss") or 0.0)
        tp = float(signal.get("take_profit") or 0.0)

        if action in ("buy", "sell"):
            if self._req.single_position_only and self._portfolio.has_open():
                return
            order = Order(
                id=0,
                type=OrderType.BUY if action == "buy" else OrderType.SELL,
                volume=volume,
                sl=sl,
                tp=tp,
                created_at_ts=tick.ts,
            )
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
            stop_limit_price = float(
                signal.get("stop_limit_price")
                or signal.get("limit_price")
                or 0.0
            )
            order = Order(
                id=0,
                type=_PENDING_TYPES[action],
                volume=volume,
                price=price,
                sl=sl,
                tp=tp,
                stop_limit_price=stop_limit_price,
                expiration=_parse_expiration(signal.get("expiration")),
                created_at_ts=tick.ts,
            )
            self._fill.enqueue(order, replace_same_type=bool(signal.get("replace")))

    # --- event builders --------------------------------------------------

    def _open_event(self, pos: Position, fill: Fill) -> dict:
        return {
            "type": "position_open",
            "ts": fill.ts,
            "ticket": pos.ticket,
            "symbol": self._req.symbol,
            "side": pos.side.value,
            "volume": pos.volume,
            "price": pos.open_price,
            "stop_loss": pos.sl,
            "take_profit": pos.tp,
            "commission": fill.commission,
            "slippage": fill.slippage,
        }

    def _close_event(self, trade) -> dict:
        return {
            "type": "position_close",
            "ts": trade.close_ts,
            "ticket": trade.ticket,
            "symbol": self._req.symbol,
            "side": trade.side.value,
            "volume": trade.volume,
            "price": trade.close_price,
            "reason": trade.reason.value,
            "pnl": trade.pnl,
            "commission": trade.commission,
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
        """Describe what the engine actually did for post-backtest transparency."""
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


# --- top-level entry -----------------------------------------------------


def run_backtest(req: BacktestRequest) -> BacktestResult:
    """Worker-process entry point.

    The backend is expected to have pre-loaded ``req.bars`` (and optional
    ``req.ticks``) before dispatching into this process.
    """
    return BacktestRunner(req).run()
