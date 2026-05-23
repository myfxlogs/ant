-- 094_risk_control_migration.down.sql
DELETE FROM user_risk_profiles WHERE updated_at >= now() - interval '1 hour';
