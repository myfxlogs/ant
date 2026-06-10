-- Migration 152 rollback: Remove admin audit log table.
BEGIN;

DROP TABLE IF EXISTS admin_audit_log;

COMMIT;
