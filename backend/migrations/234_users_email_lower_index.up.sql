-- 234: Functional index for case-insensitive email lookup in VerifyMTIdentity.
-- LOWER(email) = $1 cannot use the existing idx_users_email (case-sensitive B-tree).
-- The GIN trigram index (idx_users_email_trgm) helps fuzzy search but is suboptimal
-- for exact LOWER() equality. This functional index ensures O(log n) for case-insensitive match.
CREATE INDEX IF NOT EXISTS idx_users_email_lower ON users (LOWER(email)) WHERE deleted_at IS NULL;
