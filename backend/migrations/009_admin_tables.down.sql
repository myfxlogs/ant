-- 009_admin_tables.down.sql
-- 回滚管理员相关表

DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS admin_logs;
