-- 174: Add stop_loss and take_profit columns to paper_orders (M12-PAPER).
ALTER TABLE paper_orders ADD COLUMN IF NOT EXISTS stop_loss NUMERIC(18,8) NOT NULL DEFAULT 0;
ALTER TABLE paper_orders ADD COLUMN IF NOT EXISTS take_profit NUMERIC(18,8) NOT NULL DEFAULT 0;
