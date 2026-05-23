-- 016_account_management_indexes.up.sql
-- 优化账户管理页面查询性能的索引

-- 为 mt_accounts 表添加复合索引，加速管理员页面的 JOIN 查询
-- 1. user_id + status 复合索引：加速按用户和状态筛选
CREATE INDEX IF NOT EXISTS idx_mt_accounts_user_status ON mt_accounts(user_id, account_status);

-- 2. user_id + mt_type 复合索引：加速按用户和类型筛选
CREATE INDEX IF NOT EXISTS idx_mt_accounts_user_type ON mt_accounts(user_id, mt_type);

-- 3. login + email 索引：加速搜索功能（使用 trigram 索引）
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE INDEX IF NOT EXISTS idx_mt_accounts_login_trgm ON mt_accounts USING gin(login gin_trgm_ops);

-- 4. created_at 降序索引：加速按创建时间排序
CREATE INDEX IF NOT EXISTS idx_mt_accounts_created_at ON mt_accounts(created_at DESC);

-- 5. account_status + mt_type 复合索引：加速多条件筛选
CREATE INDEX IF NOT EXISTS idx_mt_accounts_status_type ON mt_accounts(account_status, mt_type);

-- 为 users 表添加 trigram 索引，加速邮箱和昵称搜索
CREATE INDEX IF NOT EXISTS idx_users_email_trgm ON users USING gin(email gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_users_nickname_trgm ON users USING gin(nickname gin_trgm_ops);

-- 添加覆盖索引，减少回表查询
-- 这个覆盖索引包含了查询所需的所有字段，可以避免回表
CREATE INDEX IF NOT EXISTS idx_mt_accounts_covering 
ON mt_accounts(user_id, account_status, mt_type, created_at) 
INCLUDE (login, balance, equity, margin, broker_company, broker_server);

-- 分析表以更新统计信息
ANALYZE mt_accounts;
ANALYZE users;
