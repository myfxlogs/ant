-- 016_account_management_indexes.down.sql
-- 回滚账户管理页面索引优化

-- 删除覆盖索引
DROP INDEX IF EXISTS idx_mt_accounts_covering;

-- 删除 trigram 索引
DROP INDEX IF EXISTS idx_users_nickname_trgm;
DROP INDEX IF EXISTS idx_users_email_trgm;
DROP INDEX IF EXISTS idx_mt_accounts_login_trgm;

-- 删除复合索引
DROP INDEX IF EXISTS idx_mt_accounts_status_type;
DROP INDEX IF EXISTS idx_mt_accounts_created_at;
DROP INDEX IF EXISTS idx_mt_accounts_user_type;
DROP INDEX IF EXISTS idx_mt_accounts_user_status;

-- 注意：pg_trgm 扩展不删除，因为可能被其他索引使用
