-- Migration 150: Add ON DELETE CASCADE / SET NULL to all user FKs
-- 14 FKs reference users(id) without proper delete behavior.
-- Fix applied in 149 (wallet_transactions), this handles the remaining 13.
--
-- CASCADE: user-owned data that should be cleaned up on user deletion.
-- SET NULL: reference columns (publisher/auditor) that should survive.

BEGIN;

-- === ON DELETE CASCADE (user-owned data) ===

ALTER TABLE account_connection_logs
  DROP CONSTRAINT account_connection_logs_user_id_fkey;
ALTER TABLE account_connection_logs
  ADD CONSTRAINT account_connection_logs_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE admins
  DROP CONSTRAINT admins_user_id_fkey;
ALTER TABLE admins
  ADD CONSTRAINT admins_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE api_keys
  DROP CONSTRAINT api_keys_user_id_fkey;
ALTER TABLE api_keys
  ADD CONSTRAINT api_keys_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE order_history
  DROP CONSTRAINT order_history_user_id_fkey;
ALTER TABLE order_history
  ADD CONSTRAINT order_history_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE strategy_execution_logs
  DROP CONSTRAINT strategy_execution_logs_user_id_fkey;
ALTER TABLE strategy_execution_logs
  ADD CONSTRAINT strategy_execution_logs_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE system_operation_logs
  DROP CONSTRAINT system_operation_logs_user_id_fkey;
ALTER TABLE system_operation_logs
  ADD CONSTRAINT system_operation_logs_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- user_ai_agents FK dropped (166_cleanup_dead_tables)
-- user_ai_agents FK dropped (166_cleanup_dead_tables)

ALTER TABLE user_strategy_publishes
  DROP CONSTRAINT user_strategy_publishes_user_id_fkey;
ALTER TABLE user_strategy_publishes
  ADD CONSTRAINT user_strategy_publishes_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE user_subscriptions
  DROP CONSTRAINT user_subscriptions_subscriber_user_id_fkey;
ALTER TABLE user_subscriptions
  ADD CONSTRAINT user_subscriptions_subscriber_user_id_fkey
    FOREIGN KEY (subscriber_user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE user_subscriptions
  DROP CONSTRAINT user_subscriptions_target_user_id_fkey;
ALTER TABLE user_subscriptions
  ADD CONSTRAINT user_subscriptions_target_user_id_fkey
    FOREIGN KEY (target_user_id) REFERENCES users(id) ON DELETE CASCADE;

-- === ON DELETE SET NULL (reference/audit columns) ===

ALTER TABLE marketplace_strategies
  DROP CONSTRAINT marketplace_strategies_publisher_id_fkey;
ALTER TABLE marketplace_strategies
  ADD CONSTRAINT marketplace_strategies_publisher_id_fkey
    FOREIGN KEY (publisher_id) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE platform_strategies
  DROP CONSTRAINT platform_strategies_published_by_fkey;
ALTER TABLE platform_strategies
  ADD CONSTRAINT platform_strategies_published_by_fkey
    FOREIGN KEY (published_by) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE sanctioned_countries
  DROP CONSTRAINT sanctioned_countries_added_by_fkey;
ALTER TABLE sanctioned_countries
  ADD CONSTRAINT sanctioned_countries_added_by_fkey
    FOREIGN KEY (added_by) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE user_jurisdiction
  DROP CONSTRAINT user_jurisdiction_kyc_verified_by_fkey;
ALTER TABLE user_jurisdiction
  ADD CONSTRAINT user_jurisdiction_kyc_verified_by_fkey
    FOREIGN KEY (kyc_verified_by) REFERENCES users(id) ON DELETE SET NULL;

COMMIT;
