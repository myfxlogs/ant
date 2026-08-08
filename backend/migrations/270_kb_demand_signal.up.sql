-- K3: Demand signal capture for unsupported MQL builtins.
-- When checkUnreliableCoverage hits a fatal blind spot, record a demand signal
-- so admins can prioritize which builtins to implement next (by frequency).
-- Push-first: pg_notify('kb_demand_update') on insert/update for cache invalidation.

CREATE TABLE IF NOT EXISTS kb_demand_signal (
    builtin_name  TEXT        NOT NULL,
    user_id       UUID        NOT NULL,
    hit_count     INTEGER     NOT NULL DEFAULT 1,
    last_hit_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (builtin_name, user_id)
);

COMMENT ON TABLE  kb_demand_signal IS 'K3: demand signals for unsupported MQL builtins, hit_count per user';
COMMENT ON COLUMN kb_demand_signal.builtin_name IS 'the unsupported builtin name (e.g. iCustom, ObjectCreate)';
COMMENT ON COLUMN kb_demand_signal.user_id      IS 'the user who hit this blind spot';
COMMENT ON COLUMN kb_demand_signal.hit_count    IS 'how many times this user hit this blind spot';

-- Aggregate view: total hits + unique users per builtin, ordered by demand.
CREATE OR REPLACE VIEW kb_demand_summary AS
    SELECT
        builtin_name,
        SUM(hit_count)     AS total_hits,
        COUNT(*)           AS unique_users,
        MAX(last_hit_at)   AS last_hit_at
    FROM kb_demand_signal
    GROUP BY builtin_name
    ORDER BY SUM(hit_count) DESC;
