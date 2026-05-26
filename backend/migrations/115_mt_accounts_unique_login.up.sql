-- 115: mt_accounts login must be globally unique (one account → one user).
-- Before: uk_user_mttype_login only prevented the SAME user from re-adding the same login.
-- After:  uk_mt_account_login ensures no two users can bind the same (login, mt_type, broker_server).

ALTER TABLE mt_accounts DROP CONSTRAINT IF EXISTS uk_user_mttype_login;
ALTER TABLE mt_accounts ADD CONSTRAINT uk_mt_account_login UNIQUE (login, mt_type, broker_server);
