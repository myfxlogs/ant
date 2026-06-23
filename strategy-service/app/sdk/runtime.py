"""SDK Strategy Runtime — lifecycle driver (T1.1).

Drives a StrategyBase subclass through the MQL event model:
  on_init → (on_tick / on_bar / on_timer / on_trade)* → on_deinit

Responsibilities:
  - Instantiate strategy, inject broker / ctx / indicators.
  - Enforce lifecycle ordering (init before events, no events after deinit).
  - Isolate strategy exceptions — a crashing hook does not bring down the runtime.
  - Provide a concrete RuntimeContext backed by real or test data.

Reuses:
  - engine/context.py dict shape for market data (adapting to SDK Context).
  - engine/indicators.py functions (wrapped by SDK Indicators interface).
  - engine/live_sandbox.py worker pattern (long-lived, restartable).
"""

from __future__ import annotations

import logging
from typing import Any, Callable, Dict, List, Optional

from app.sdk.account import AccountInfo
from app.sdk.broker import Broker
from app.sdk.context import Context
from app.sdk.indicators import Indicators
from app.sdk.series import Bars, Series
from app.sdk.strategy_base import StrategyBase
from app.sdk.symbol import SymbolInfo

logger = logging.getLogger(__name__)


# ── Strategy Runtime ───────────────────────────────────────────────────


class StrategyRuntime:
    """Lifecycle orchestrator for a single strategy instance.

    Usage::

        runtime = StrategyRuntime(
            strategy_class=SingleMACross,
            broker=sim_broker,
            bars_provider=my_bars_factory,
            indicators=indicator_impl,
            symbol="EURUSD",
            timeframe="M15",
            params={"fast_period": 12, "slow_period": 26},
        )
        runtime.init()
        runtime.on_tick()
        runtime.on_bar("M15")
        runtime.on_timer()
        runtime.on_trade()
        runtime.deinit("user_stop")
    """

    def __init__(
        self,
        strategy_class: type[StrategyBase],
        broker: Broker,
        bars_provider: Callable[[Optional[str]], Bars],
        indicators: Indicators,
        symbol: str,
        timeframe: str,
        params: Optional[Dict[str, Any]] = None,
    ) -> None:
        if not issubclass(strategy_class, StrategyBase):
            raise TypeError(
                f"strategy_class must be a StrategyBase subclass, got {strategy_class}"
            )

        self._strategy_class = strategy_class
        self._broker = broker
        self._bars_provider = bars_provider
        self._indicators = indicators
        self._symbol = symbol
        self._timeframe = timeframe
        self._params = dict(params or {})

        self._strategy: Optional[StrategyBase] = None
        self._ctx: Optional[RuntimeContext] = None
        self._state: str = "created"  # created → ready → deinit

        # Timer state.
        self._timer_seconds: int = 0
        self._timer_active: bool = False

    # ── Lifecycle ──────────────────────────────────────────────────────

    def init(self) -> None:
        """Instantiate strategy, inject dependencies, call on_init()."""
        if self._state != "created":
            raise RuntimeError(f"Cannot init: runtime state is '{self._state}'")

        strategy = self._strategy_class()

        # Build context.
        self._ctx = RuntimeContext(
            symbol=self._symbol,
            timeframe=self._timeframe,
            bars_provider=self._bars_provider,
            params=self._params,
            timer_setter=self._register_timer,
            timer_killer=self._unregister_timer,
        )

        # Inject.
        strategy.broker = self._broker
        strategy.ctx = self._ctx
        strategy.indicators = self._indicators

        self._strategy = strategy
        self._state = "ready"

        self._safe_call("on_init")

    def on_tick(self) -> None:
        """Feed a tick event to the strategy."""
        self._require_ready()
        self._safe_call("on_tick")

    def on_bar(self, timeframe: str) -> None:
        """Feed a bar-close event to the strategy."""
        self._require_ready()
        self._safe_call("on_bar", timeframe)

    def on_timer(self) -> None:
        """Feed a timer event to the strategy (triggered by external scheduler)."""
        self._require_ready()
        if not self._timer_active:
            return
        self._safe_call("on_timer")

    def on_trade(self) -> None:
        """Feed a trade/position-change event to the strategy."""
        self._require_ready()
        self._safe_call("on_trade")

    def deinit(self, reason: str = "user_stop") -> None:
        """Shutdown the strategy with a reason code.

        Valid reasons: user_stop, kill_switch, error, schedule_end.
        """
        if self._state == "deinit":
            return  # idempotent
        if self._state == "created":
            self._state = "deinit"
            return  # never initialized, nothing to clean up

        self._safe_call("on_deinit", reason)
        self._state = "deinit"
        self._strategy = None
        self._ctx = None
        self._timer_active = False

    # ── State ──────────────────────────────────────────────────────────

    @property
    def state(self) -> str:
        return self._state

    @property
    def strategy(self) -> Optional[StrategyBase]:
        return self._strategy

    @property
    def is_timer_active(self) -> bool:
        return self._timer_active

    # ── Internals ──────────────────────────────────────────────────────

    def _require_ready(self) -> None:
        if self._state != "ready":
            raise RuntimeError(
                f"Cannot dispatch event: runtime state is '{self._state}' (expected 'ready')"
            )

    def _safe_call(self, method_name: str, *args: Any) -> Optional[Any]:
        """Call a strategy hook, catching and logging any exception.

        The runtime MUST survive a crashing strategy — the exception is logged
        but does not propagate.  The strategy instance remains in whatever
        state the crash left it; the caller can decide to deinit.
        """
        strategy = self._strategy
        if strategy is None:
            return None

        method = getattr(strategy, method_name, None)
        if method is None:
            return None

        try:
            return method(*args)
        except Exception:
            logger.exception(
                "Strategy %s.%s raised an exception",
                self._strategy_class.__name__,
                method_name,
            )
            return None

    def _register_timer(self, seconds: int) -> None:
        if seconds < 1:
            raise ValueError(f"Timer interval must be >= 1s, got {seconds}")
        self._timer_seconds = seconds
        self._timer_active = True

    def _unregister_timer(self) -> None:
        self._timer_active = False
        self._timer_seconds = 0


# ── Runtime Context (concrete) ─────────────────────────────────────────


class RuntimeContext(Context):
    """Concrete Context backed by a bars provider and params dict.

    This bridges the SDK Context interface to whatever data source is
    available — real market data (via engine/market.py), test fixtures,
    or mock implementations.
    """

    def __init__(
        self,
        symbol: str,
        timeframe: str,
        bars_provider: Callable[[Optional[str]], Bars],
        params: Dict[str, Any],
        timer_setter: Callable[[int], None],
        timer_killer: Callable[[], None],
    ) -> None:
        self.symbol = symbol
        self.timeframe = timeframe
        self._bars_provider = bars_provider
        self._params = params
        self._timer_setter = timer_setter
        self._timer_killer = timer_killer

    def bars(self, timeframe: Optional[str] = None) -> Bars:
        """Return Bars for the requested (or primary) timeframe."""
        return self._bars_provider(timeframe)

    def param(self, name: str, default: object = None) -> object:
        """Read a strategy parameter (MQL extern/input equivalent)."""
        return self._params.get(name, default)

    def set_timer(self, seconds: int) -> None:
        """Register a periodic on_timer() callback."""
        self._timer_setter(seconds)

    def kill_timer(self) -> None:
        """Unregister the timer."""
        self._timer_killer()
