"""T1.3 — Backtest fidelity baseline.

Framework for comparing SimBroker strategy execution against expected
behavior (golden reference).  When MT Strategy Tester reports are
available, they serve as the golden source.

Currently establishes a self-consistency baseline: runs a deterministic
EA through SimBroker, computes expected trades from the price data, and
diffs the two.  Deviations are classified as:
  - EXPLAINED: cost model, tick synthesis, fill-timing assumptions
  - NEEDS-DECISION: unexplained — must be escalated per T1.3 DoD
"""
