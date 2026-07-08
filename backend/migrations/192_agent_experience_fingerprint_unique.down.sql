-- 192 down: Restore non-unique index.
DROP INDEX IF EXISTS uq_agent_experience_fingerprint;
CREATE INDEX IF NOT EXISTS idx_agent_experience_fingerprint
    ON agent_experience (fingerprint) WHERE fingerprint <> ''::text;
