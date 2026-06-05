ALTER TABLE trade_records DROP CONSTRAINT IF EXISTS trade_records_user_id_fkey;
DROP INDEX IF EXISTS idx_trade_records_user_id;
ALTER TABLE trade_records DROP COLUMN IF EXISTS user_id;
