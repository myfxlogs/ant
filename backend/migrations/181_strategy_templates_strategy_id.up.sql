-- 181_strategy_templates_strategy_id.up.sql
-- ADR-0023: MQL source is the single source of truth.
-- Link strategy_templates to imported_strategies so templates can
-- resolve the original MQL source for AST→Bytecode execution.

ALTER TABLE strategy_templates
  ADD COLUMN IF NOT EXISTS strategy_id UUID REFERENCES imported_strategies(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_strategy_templates_strategy_id
  ON strategy_templates(strategy_id) WHERE strategy_id IS NOT NULL;
