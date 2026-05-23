-- Add per-profile inference parameters: temperature, timeout (seconds),
-- max_tokens. NULL semantics:
--   * temperature      NULL => use provider default sampling
--   * timeout_seconds  NULL => use built-in default (60s)
--   * max_tokens       NULL => use provider default (no client-side cap)
-- All three are optional and the existing rows keep working unchanged.

ALTER TABLE ai_config_profiles
    ADD COLUMN IF NOT EXISTS temperature      DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS timeout_seconds  INTEGER,
    ADD COLUMN IF NOT EXISTS max_tokens       INTEGER;
