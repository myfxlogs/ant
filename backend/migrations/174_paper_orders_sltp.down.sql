-- 174: Remove stop_loss and take_profit columns from paper_orders.
ALTER TABLE paper_orders DROP COLUMN IF EXISTS stop_loss;
ALTER TABLE paper_orders DROP COLUMN IF EXISTS take_profit;
