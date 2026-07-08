-- 192: Add UNIQUE index on agent_experience(user_id, fingerprint) for dedup.
-- Existing partial btree index is non-unique; StoreExperience does plain INSERT
-- with no ON CONFLICT, causing duplicate rows on strategy re-generation.
-- Replace the non-unique partial index with a unique one (still partial to allow
-- legacy rows with empty fingerprint='').
DROP INDEX IF EXISTS idx_agent_experience_fingerprint;
CREATE UNIQUE INDEX IF NOT EXISTS uq_agent_experience_fingerprint
    ON agent_experience (user_id, fingerprint) WHERE fingerprint <> ''::text;
