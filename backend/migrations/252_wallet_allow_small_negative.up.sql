-- Allow wallet balance to go slightly negative for streaming AI billing.
-- Industry standard (telecom, cloud, OpenAI): micro-transactions can briefly overdraft;
-- the pre-check walletChecker blocks future calls until balance > $1.00.
ALTER TABLE user_wallets DROP CONSTRAINT IF EXISTS user_wallets_balance_check;
ALTER TABLE user_wallets ADD CONSTRAINT user_wallets_balance_check CHECK (balance >= '-0.10'::numeric);
