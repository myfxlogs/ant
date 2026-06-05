ALTER TABLE trade_records ADD COLUMN IF NOT EXISTS user_id uuid;
UPDATE trade_records tr SET user_id = ma.user_id FROM mt_accounts ma WHERE ma.id = tr.account_id AND tr.user_id IS NULL;
ALTER TABLE trade_records ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE trade_records ADD CONSTRAINT trade_records_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_trade_records_user_id ON trade_records(user_id);
