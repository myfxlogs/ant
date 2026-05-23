-- 008_fix_order_type_length.up.sql
-- 修复 trade_records 表的 order_type 字段长度

ALTER TABLE trade_records 
    ALTER COLUMN order_type TYPE VARCHAR(30);
