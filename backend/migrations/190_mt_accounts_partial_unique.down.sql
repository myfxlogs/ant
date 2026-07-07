-- 190 down: restore full unique constraint, drop partial index.

DROP INDEX IF EXISTS uk_mt_account_login_active;
ALTER TABLE mt_accounts ADD CONSTRAINT uk_mt_account_login UNIQUE (login, mt_type, broker_server);
