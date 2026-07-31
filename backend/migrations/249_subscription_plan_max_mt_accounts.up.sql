-- GAP-3: Add max_mt_accounts to subscription_plans for tier-based account slot limits.
-- Free=1, Pro=5, Enterprise=0 (unlimited). 0 = unlimited (same convention as other max_* columns).

ALTER TABLE subscription_plans ADD COLUMN IF NOT EXISTS max_mt_accounts INTEGER NOT NULL DEFAULT 1;

-- Update existing plans with correct account slot limits.
UPDATE subscription_plans SET max_mt_accounts = 1 WHERE name = 'free';
UPDATE subscription_plans SET max_mt_accounts = 5 WHERE name = 'pro';
UPDATE subscription_plans SET max_mt_accounts = 0 WHERE name = 'enterprise';

COMMENT ON COLUMN subscription_plans.max_mt_accounts IS 'Max MT account slots per user on this plan; 0 = unlimited.';
