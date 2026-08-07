-- 252_wallet_allow_small_negative.down.sql
-- Restore original balance constraint (balance >= 0)
ALTER TABLE user_wallets DROP CONSTRAINT IF EXISTS user_wallets_balance_check;
ALTER TABLE user_wallets ADD CONSTRAINT user_wallets_balance_check CHECK (balance >= 0);
