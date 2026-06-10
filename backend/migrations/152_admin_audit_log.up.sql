-- Migration 152: Admin audit log for user management operations.
-- Records who performed what action on which user, including a snapshot
-- of affected child-table row counts before soft-delete.
BEGIN;

CREATE TABLE admin_audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    actor_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,         -- 'delete_user', 'restore_user', 'batch_delete'
    target_id UUID NOT NULL,
    target_email VARCHAR(255) NOT NULL,
    affected_data JSONB NOT NULL DEFAULT '{}',  -- {"table": row_count, ...}
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_admin_audit_log_actor ON admin_audit_log (actor_id);
CREATE INDEX idx_admin_audit_log_target ON admin_audit_log (target_id);
CREATE INDEX idx_admin_audit_log_created ON admin_audit_log (created_at DESC);

COMMIT;
