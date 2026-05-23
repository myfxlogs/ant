-- 008_fix_trade_records_precision.down.sql
-- 回滚 trade_records 表的数值精度修改

-- 先删除约束
ALTER TABLE trade_records DROP CONSTRAINT IF EXISTS uk_trade_record_ticket;

-- 恢复原来的字段精度
ALTER TABLE trade_records 
    ALTER COLUMN open_price TYPE DECIMAL(18, 8),
    ALTER COLUMN close_price TYPE DECIMAL(18, 8),
    ALTER COLUMN stop_loss TYPE DECIMAL(18, 8),
    ALTER COLUMN take_profit TYPE DECIMAL(18, 8),
    ALTER COLUMN profit TYPE DECIMAL(18, 4),
    ALTER COLUMN swap TYPE DECIMAL(18, 4),
    ALTER COLUMN commission TYPE DECIMAL(18, 4),
    ALTER COLUMN volume TYPE DECIMAL(10, 4);

-- 重新添加约束
ALTER TABLE trade_records 
    ADD CONSTRAINT uk_trade_record_ticket UNIQUE (account_id, ticket, close_time);
