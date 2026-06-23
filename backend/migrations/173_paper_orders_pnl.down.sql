-- 173: Remove pnl column from paper_orders.
ALTER TABLE paper_orders DROP COLUMN IF EXISTS pnl;
