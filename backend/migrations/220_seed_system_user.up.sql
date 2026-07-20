-- 220_seed_system_user.up.sql
-- Seed the system user (uuid.Nil) so that FK constraints referencing
-- users(id) are satisfied for system-generated strategies, platform
-- fee wallets, and other system-level operations.
--
-- Without this row:
--   marketplace_strategies.publisher_id  → FK violation
--   user_strategy_publishes.user_id      → FK violation
--   user_wallets.user_id (system wallet) → FK violation

INSERT INTO users (id, email, password_hash, nickname, role, status)
VALUES ('00000000-0000-0000-0000-000000000000',
        'system@alphaforge.internal',
        '',  -- no password — cannot log in
        'System',
        'system',
        'active')
ON CONFLICT (id) DO NOTHING;
