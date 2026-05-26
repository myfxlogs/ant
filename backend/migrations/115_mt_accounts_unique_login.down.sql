ALTER TABLE mt_accounts DROP CONSTRAINT IF EXISTS uk_mt_account_login;
ALTER TABLE mt_accounts ADD CONSTRAINT uk_user_mttype_login UNIQUE (user_id, mt_type, login);
