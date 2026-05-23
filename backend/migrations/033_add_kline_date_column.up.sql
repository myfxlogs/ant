-- 033_add_kline_date_column.up.sql

DO $$
BEGIN
    IF to_regclass('public.kline_data') IS NOT NULL THEN
        IF NOT EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public' AND table_name = 'kline_data' AND column_name = 'kline_date'
        ) THEN
            ALTER TABLE kline_data ADD COLUMN kline_date DATE;
            UPDATE kline_data SET kline_date = open_time::date WHERE kline_date IS NULL;
            CREATE INDEX IF NOT EXISTS idx_kline_kline_date ON kline_data(kline_date);
        END IF;
    END IF;
END $$;

INSERT INTO schema_migrations (version) VALUES ('033_add_kline_date_column') ON CONFLICT DO NOTHING;
