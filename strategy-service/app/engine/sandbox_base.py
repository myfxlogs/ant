"""Base sandbox abstract interface (B-3.0).

All sandbox implementations — BacktestSandbox, LiveWorker, and the legacy
StrategyRunner — conform to this interface so calling code never depends on
a concrete implementation.
"""

from __future__ import annotations

from abc import ABC, abstractmethod
from typing import Optional


class BaseSandbox(ABC):
    """Abstract sandbox that compiles and executes user strategy code."""

    @abstractmethod
    def call(self, ctx: dict) -> Optional[dict]:
        """Execute the strategy with *ctx* and return a signal dict (or None)."""
        ...

    @abstractmethod
    def shutdown(self) -> None:
        """Release sandbox resources (process, pipe, etc.)."""
        ...

    @property
    @abstractmethod
    def source_sha256(self) -> str:
        """Hex digest of the strategy source code."""
        ...
