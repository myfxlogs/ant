-- 182_backfill_template_strategy_id.up.sql
-- ADR-0023: Backfill strategy_templates.strategy_id and replace Go code with MQL source.
-- Existing templates that store transpiled Go code (from mql2go) are matched to
-- imported_strategies by (user_id, name) and their code is replaced with raw MQL.

-- 1. Link templates to imported_strategies by (user_id, name) where strategy_id is NULL.
UPDATE strategy_templates t
  SET strategy_id = s.id
  FROM imported_strategies s
  WHERE t.strategy_id IS NULL
    AND t.user_id IS NOT NULL
    AND t.user_id = s.user_id
    AND t.name = s.name;

-- 2. Replace Go code with MQL source for templates now linked to imported_strategies.
--    The Go code was a transpiled preview; MQL is the single source of truth.
UPDATE strategy_templates t
  SET code = s.source_code
  FROM imported_strategies s
  WHERE t.strategy_id = s.id
    AND t.code != s.source_code;

-- 3. System templates: these may store Go code directly (not from imported_strategies).
--    Leave them as-is — they will be migrated individually if needed.
