ALTER TABLE marketplace_strategy_optimization_tasks
    ALTER COLUMN decay_metrics TYPE JSONB USING NULL;
