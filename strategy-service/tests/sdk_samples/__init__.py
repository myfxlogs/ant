"""T0.4 — Sample EAs hand-written against the frozen Strategy SDK.

These 5 strategies validate that the SDK has sufficient expressiveness
to represent real EA forms.  They import the SDK stubs only; no engine
or runtime dependency.

Strategies:
  1. single_ma_cross  — EMA crossover, one position at a time
  2. grid_trader      — pending-order grid with magic-number identification
  3. martingale       — double-after-loss, reset-after-win
  4. hedge_twins      — simultaneous long+short in hedging mode
  5. custom_signal    — i_custom indicator + multi-timeframe
"""
