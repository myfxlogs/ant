-- 279_data_truth_3_cleanup.up.sql
-- DATA-TRUTH-3: drop dead column + annotate credentials-only compat view.
--
-- last_checked_at: created in 001, zero writers codebase-wide, 16/16 rows
-- NULL on live DB (2026-09-19 measured). Read plumbing removed in the same
-- commit (sqlc regen + hand-edited scan sites).
ALTER TABLE mt_accounts DROP COLUMN IF EXISTS last_checked_at;

-- mt_accounts_v2 is a credentials-only compat view for the mdgateway runner
-- config path (wiring.go) — NOT a migration target. mt_accounts (v1) is the
-- runtime source of truth. Do not extend this view or add new consumers.
COMMENT ON VIEW mt_accounts_v2 IS 'credentials-only compat view for mdgateway runner config (wiring.go); mt_accounts is the runtime truth — do not extend, do not add consumers';
