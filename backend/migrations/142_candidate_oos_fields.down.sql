ALTER TABLE strategy_experiment_candidates
    DROP COLUMN IF EXISTS oos_score,
    DROP COLUMN IF EXISTS oos_total_return,
    DROP COLUMN IF EXISTS oos_sharpe_ratio,
    DROP COLUMN IF EXISTS degradation_pct,
    DROP COLUMN IF EXISTS is_overfit;
