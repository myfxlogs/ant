-- 280_mt_accounts_margin_mode.up.sql
-- MT5-ACCMETHOD-ADAPTER-1: broker account margin mode (hedging/netting)
-- from mtapi AccountSummary.Method, written at account bind/verify.
-- NULL = unknown (MT5 Default / pre-migration rows); distinct from
-- account_method which stores the investor flag ("master"/"investor").
ALTER TABLE mt_accounts ADD COLUMN IF NOT EXISTS margin_mode VARCHAR(16);
COMMENT ON COLUMN mt_accounts.margin_mode IS 'broker margin mode: hedging|netting|NULL(unknown); from mtapi AccMethod — NOT the investor-flag account_method';
