-- Migration 150 rollback: Revert ON DELETE behavior for user FKs
BEGIN;

ALTER TABLE account_connection_logs DROP CONSTRAINT account_connection_logs_user_id_fkey;
ALTER TABLE account_connection_logs ADD CONSTRAINT account_connection_logs_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE admins DROP CONSTRAINT admins_user_id_fkey;
ALTER TABLE admins ADD CONSTRAINT admins_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE api_keys DROP CONSTRAINT api_keys_user_id_fkey;
ALTER TABLE api_keys ADD CONSTRAINT api_keys_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE order_history DROP CONSTRAINT order_history_user_id_fkey;
ALTER TABLE order_history ADD CONSTRAINT order_history_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE strategy_execution_logs DROP CONSTRAINT strategy_execution_logs_user_id_fkey;
ALTER TABLE strategy_execution_logs ADD CONSTRAINT strategy_execution_logs_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE system_operation_logs DROP CONSTRAINT system_operation_logs_user_id_fkey;
ALTER TABLE system_operation_logs ADD CONSTRAINT system_operation_logs_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE user_ai_agents DROP CONSTRAINT user_ai_agents_user_id_fkey;
ALTER TABLE user_ai_agents ADD CONSTRAINT user_ai_agents_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE user_strategy_publishes DROP CONSTRAINT user_strategy_publishes_user_id_fkey;
ALTER TABLE user_strategy_publishes ADD CONSTRAINT user_strategy_publishes_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE user_subscriptions DROP CONSTRAINT user_subscriptions_subscriber_user_id_fkey;
ALTER TABLE user_subscriptions ADD CONSTRAINT user_subscriptions_subscriber_user_id_fkey FOREIGN KEY (subscriber_user_id) REFERENCES users(id);

ALTER TABLE user_subscriptions DROP CONSTRAINT user_subscriptions_target_user_id_fkey;
ALTER TABLE user_subscriptions ADD CONSTRAINT user_subscriptions_target_user_id_fkey FOREIGN KEY (target_user_id) REFERENCES users(id);

ALTER TABLE marketplace_strategies DROP CONSTRAINT marketplace_strategies_publisher_id_fkey;
ALTER TABLE marketplace_strategies ADD CONSTRAINT marketplace_strategies_publisher_id_fkey FOREIGN KEY (publisher_id) REFERENCES users(id);

ALTER TABLE platform_strategies DROP CONSTRAINT platform_strategies_published_by_fkey;
ALTER TABLE platform_strategies ADD CONSTRAINT platform_strategies_published_by_fkey FOREIGN KEY (published_by) REFERENCES users(id);

ALTER TABLE sanctioned_countries DROP CONSTRAINT sanctioned_countries_added_by_fkey;
ALTER TABLE sanctioned_countries ADD CONSTRAINT sanctioned_countries_added_by_fkey FOREIGN KEY (added_by) REFERENCES users(id);

ALTER TABLE user_jurisdiction DROP CONSTRAINT user_jurisdiction_kyc_verified_by_fkey;
ALTER TABLE user_jurisdiction ADD CONSTRAINT user_jurisdiction_kyc_verified_by_fkey FOREIGN KEY (kyc_verified_by) REFERENCES users(id);

COMMIT;
