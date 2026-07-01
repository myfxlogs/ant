-- 182_backfill_template_strategy_id.down.sql
-- Cannot reliably reverse the code replacement (original Go code is lost).
-- This migration only nulls the strategy_id FK; code stays as MQL.

UPDATE strategy_templates SET strategy_id = NULL WHERE strategy_id IS NOT NULL;
