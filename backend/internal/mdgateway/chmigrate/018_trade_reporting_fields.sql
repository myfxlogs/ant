-- 018_trade_reporting_fields.sql
-- M11-17: Reserve EMIR / MiFID II trade reporting fields on trade_event_log.
-- These columns are schema reservations only — no writer populates them yet.
-- Full regulatory reporting implementation is deferred to a future milestone.

-- EMIR (European Market Infrastructure Regulation) fields
ALTER TABLE ant.trade_event_log ADD COLUMN IF NOT EXISTS reporting_uti String DEFAULT '';
ALTER TABLE ant.trade_event_log ADD COLUMN IF NOT EXISTS reporting_entity_lei LowCardinality(String) DEFAULT '';
ALTER TABLE ant.trade_event_log ADD COLUMN IF NOT EXISTS reporting_counterparty_lei LowCardinality(String) DEFAULT '';
ALTER TABLE ant.trade_event_log ADD COLUMN IF NOT EXISTS reporting_action_type LowCardinality(String) DEFAULT '';
ALTER TABLE ant.trade_event_log ADD COLUMN IF NOT EXISTS reporting_exec_ts DateTime64(3, 'UTC') DEFAULT 0;
ALTER TABLE ant.trade_event_log ADD COLUMN IF NOT EXISTS reporting_venue LowCardinality(String) DEFAULT '';
ALTER TABLE ant.trade_event_log ADD COLUMN IF NOT EXISTS reporting_clearing_obligation String DEFAULT '';

-- MiFID II / MiFIR fields
ALTER TABLE ant.trade_event_log ADD COLUMN IF NOT EXISTS reporting_mifid_trading_capacity LowCardinality(String) DEFAULT '';
ALTER TABLE ant.trade_event_log ADD COLUMN IF NOT EXISTS reporting_mifid_waiver String DEFAULT '';
ALTER TABLE ant.trade_event_log ADD COLUMN IF NOT EXISTS reporting_mifid_investment_decision LowCardinality(String) DEFAULT '';
ALTER TABLE ant.trade_event_log ADD COLUMN IF NOT EXISTS reporting_mifid_execution_entity LowCardinality(String) DEFAULT '';
