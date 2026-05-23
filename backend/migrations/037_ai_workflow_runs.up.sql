CREATE TABLE IF NOT EXISTS ai_workflow_runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL DEFAULT 'AI 工作流',
    status VARCHAR(20) NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'succeeded', 'failed', 'canceled')),
    context_json TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ai_workflow_runs_user_id ON ai_workflow_runs(user_id);
CREATE INDEX IF NOT EXISTS idx_ai_workflow_runs_updated_at ON ai_workflow_runs(updated_at DESC);

CREATE TABLE IF NOT EXISTS ai_workflow_steps (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    run_id UUID NOT NULL REFERENCES ai_workflow_runs(id) ON DELETE CASCADE,
    step_key VARCHAR(50) NOT NULL,
    title VARCHAR(200) NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'done', 'error')),
    input TEXT NOT NULL DEFAULT '',
    output TEXT NOT NULL DEFAULT '',
    error TEXT NOT NULL DEFAULT '',
    duration_ms BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ai_workflow_steps_run_id ON ai_workflow_steps(run_id);
CREATE INDEX IF NOT EXISTS idx_ai_workflow_steps_created_at ON ai_workflow_steps(created_at);
