-- 052_backtest_runs_extra_symbols.up.sql
-- Phase B2: record the set of secondary symbols whose K-lines were pulled
-- into a backtest run as features. Trading execution is still anchored on
-- ``symbol`` (the primary). The list is stored as a TEXT[] so Postgres can
-- index / query it natively if needed.

ALTER TABLE backtest_runs
  ADD COLUMN IF NOT EXISTS extra_symbols TEXT[] NOT NULL DEFAULT '{}'::TEXT[];
