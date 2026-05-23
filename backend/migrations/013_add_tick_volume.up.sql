-- 013_add_tick_volume.up.sql
-- 添加 tick_volume 列到 kline_data 表

ALTER TABLE kline_data ADD COLUMN IF NOT EXISTS tick_volume BIGINT DEFAULT 0;
ALTER TABLE kline_data ADD COLUMN IF NOT EXISTS real_volume DECIMAL(18, 2) DEFAULT 0;
ALTER TABLE kline_data ADD COLUMN IF NOT EXISTS spread INTEGER DEFAULT 0;
