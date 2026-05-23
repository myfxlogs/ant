-- 094_risk_control_migration.up.sql
-- M3-3: Migrate Strategy.RiskControl JSONB → user_risk_profiles.
-- Extracts first non-null RiskControl per user and seeds user_risk_profiles.

INSERT INTO user_risk_profiles (user_id, max_positions, daily_loss_limit, max_drawdown_pct)
SELECT
    s.user_id,
    COALESCE(
        (s.risk_control->>'max_positions')::int,
        5
    ) AS max_positions,
    (s.risk_control->>'daily_loss_limit')::numeric AS daily_loss_limit,
    (s.risk_control->>'max_drawdown_pct')::numeric AS max_drawdown_pct
FROM strategies s
WHERE s.risk_control IS NOT NULL
  AND NOT EXISTS (
      SELECT 1 FROM user_risk_profiles urp WHERE urp.user_id = s.user_id
  )
GROUP BY s.user_id, s.risk_control
LIMIT 1;
