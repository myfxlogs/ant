-- Add gate_results column to store individual GateResult proto bytes.
-- Allows restoreGateEvaluation to replay individual gate tags on reconnect.
ALTER TABLE gate_evaluations ADD COLUMN IF NOT EXISTS gate_results BYTEA;
