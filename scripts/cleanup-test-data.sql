-- Clean up test/E2E user data from the database.
-- Run: docker exec -i alphaforge-postgres psql -U ant -d ant < scripts/cleanup-test-data.sql

BEGIN;

-- 1. Identify test users
CREATE TEMP TABLE test_users AS
SELECT id FROM users
WHERE email LIKE 'e2e-reg-%'
   OR email = 'devtest@ant.local'
   OR email LIKE '%_test_%alphaforge.io';

-- 2. Delete data from tables that reference users or mt_accounts
-- strategy_signals.account_id is varchar
DELETE FROM strategy_signals WHERE account_id::uuid IN (SELECT id FROM test_users);

-- backtest_runs
DELETE FROM backtest_run_trades WHERE run_id IN (
    SELECT id FROM backtest_runs WHERE user_id IN (SELECT id FROM test_users)
);
DELETE FROM backtest_runs WHERE user_id IN (SELECT id FROM test_users);

-- deposits
DELETE FROM deposits WHERE user_id IN (SELECT id FROM test_users);

-- wallet transactions via user_wallets
DELETE FROM wallet_transactions WHERE wallet_id IN (
    SELECT id FROM user_wallets WHERE user_id IN (SELECT id FROM test_users)
);
DELETE FROM user_wallets WHERE user_id IN (SELECT id FROM test_users);

-- strategy assets
DELETE FROM strategy_asset_clones WHERE user_id IN (SELECT id FROM test_users);
DELETE FROM strategy_asset_versions WHERE asset_id IN (
    SELECT id FROM strategy_assets WHERE owner_user_id IN (SELECT id FROM test_users)
);
DELETE FROM strategy_assets WHERE owner_user_id IN (SELECT id FROM test_users);

-- marketplace data
DELETE FROM marketplace_purchases WHERE buyer_user_id IN (SELECT id FROM test_users);
DELETE FROM marketplace_ratings WHERE user_id IN (SELECT id FROM test_users);
DELETE FROM marketplace_comments WHERE user_id IN (SELECT id FROM test_users);
DELETE FROM marketplace_settlements WHERE provider_id IN (SELECT id FROM test_users);
DELETE FROM marketplace_subscriptions WHERE subscriber_user_id IN (SELECT id FROM test_users);
DELETE FROM provider_earnings WHERE provider_id IN (SELECT id FROM test_users);

-- Notifications
DELETE FROM user_notifications WHERE user_id IN (SELECT id FROM test_users);

-- AI conversations
DELETE FROM ai_conversation_messages WHERE conversation_id IN (
    SELECT id FROM ai_conversations WHERE user_id IN (SELECT id FROM test_users)
);
DELETE FROM ai_conversations WHERE user_id IN (SELECT id FROM test_users);

-- 3. Soft-delete test users
UPDATE users SET deleted_at = NOW() WHERE id IN (SELECT id FROM test_users);

-- Report
SELECT 'test users cleaned' as result, count(*) FROM test_users;

COMMIT;
