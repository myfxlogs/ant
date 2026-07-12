-- 197_backfill_email_verified.up.sql
-- Mark all existing users as email-verified so the REQUIRE_EMAIL_VERIFICATION
-- flag doesn't lock out users who registered before the verification flow.
UPDATE users SET email_verified_at = COALESCE(email_verified_at, created_at)
WHERE email_verified_at IS NULL;
