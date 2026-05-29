-- Add per-broker margin call / stop out thresholds.
-- Default 100/50 are industry medians; individual brokers diverge widely
-- (Exness 60/0, IC Markets 100/50, FBS 40/20).
-- mdgateway will UPDATE these columns when mtapi provides real values.
ALTER TABLE mt_accounts
  ADD COLUMN broker_margin_call_pct REAL NOT NULL DEFAULT 100.0,
  ADD COLUMN broker_stop_out_pct REAL NOT NULL DEFAULT 50.0;
