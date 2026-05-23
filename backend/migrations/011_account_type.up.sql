-- 011_account_type.up.sql
-- 添加账户类型字段

ALTER TABLE mt_accounts ADD COLUMN IF NOT EXISTS account_type VARCHAR(20) DEFAULT 'unknown';

COMMENT ON COLUMN mt_accounts.account_type IS '账户类型: demo, real, contest, unknown';

-- 更新现有数据，根据is_investor和其他信息推断类型
-- 由于无法从现有数据推断，保持为unknown，将在下次连接时更新
