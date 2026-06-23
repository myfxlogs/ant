-- 173: Add pnl column to paper_orders for PnL tracking (T3.1 live paper trading).
ALTER TABLE paper_orders ADD COLUMN IF NOT EXISTS pnl NUMERIC(18,8) NOT NULL DEFAULT 0;
