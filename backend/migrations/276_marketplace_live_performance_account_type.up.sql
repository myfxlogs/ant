-- 276_marketplace_live_performance_account_type.up.sql
-- TRUST-1: distinguish demo/real/contest accounts in marketplace live performance.
-- Q1=A (real-only): leaderboard filters account_type = 'real'.

-- daily 表加列
ALTER TABLE marketplace_live_performance ADD COLUMN IF NOT EXISTS account_type VARCHAR(20) NOT NULL DEFAULT 'unknown';
CREATE INDEX IF NOT EXISTS idx_marketplace_live_performance_account_type ON marketplace_live_performance(account_type);

-- summary 表加列（leaderboard 查的是 summary 表）
ALTER TABLE marketplace_live_performance_summary ADD COLUMN IF NOT EXISTS account_type VARCHAR(20) NOT NULL DEFAULT 'unknown';
