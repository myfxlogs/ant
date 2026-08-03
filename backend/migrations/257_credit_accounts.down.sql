DROP TABLE IF EXISTS credit_transactions;
DROP TABLE IF EXISTS credit_accounts;
ALTER TABLE ai_models DROP COLUMN IF EXISTS markup_rate;
ALTER TABLE ai_models DROP COLUMN IF EXISTS model_tier;
