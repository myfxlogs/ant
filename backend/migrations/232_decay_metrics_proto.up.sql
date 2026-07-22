-- Phase 5 audit: change decay_metrics from JSONB to BYTEA for proto serialization.
-- JSONB is prohibited for application-layer serialization; use proto.Marshal instead.
ALTER TABLE marketplace_strategy_optimization_tasks
    ALTER COLUMN decay_metrics TYPE BYTEA USING NULL;
