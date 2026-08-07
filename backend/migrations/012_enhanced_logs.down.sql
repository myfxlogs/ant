-- 012_enhanced_logs.down.sql
-- Auto-generated rollback for 012_enhanced_logs

-- Drop indexes
DROP INDEX IF EXISTS idx_account_conn_logs_account;
DROP INDEX IF EXISTS idx_account_conn_logs_created_at;
DROP INDEX IF EXISTS idx_account_conn_logs_event_type;
DROP INDEX IF EXISTS idx_account_conn_logs_user;
DROP INDEX IF EXISTS idx_order_history_account;
DROP INDEX IF EXISTS idx_order_history_close_time;
DROP INDEX IF EXISTS idx_order_history_open_time;
DROP INDEX IF EXISTS idx_order_history_symbol;
DROP INDEX IF EXISTS idx_order_history_ticket;
DROP INDEX IF EXISTS idx_order_history_user;
DROP INDEX IF EXISTS idx_strategy_exec_logs_account;
DROP INDEX IF EXISTS idx_strategy_exec_logs_created_at;
DROP INDEX IF EXISTS idx_strategy_exec_logs_schedule;
DROP INDEX IF EXISTS idx_strategy_exec_logs_status;
DROP INDEX IF EXISTS idx_strategy_exec_logs_user;
DROP INDEX IF EXISTS idx_system_op_logs_created_at;
DROP INDEX IF EXISTS idx_system_op_logs_module;
DROP INDEX IF EXISTS idx_system_op_logs_operation;
DROP INDEX IF EXISTS idx_system_op_logs_resource;
DROP INDEX IF EXISTS idx_system_op_logs_user;

-- Drop tables
DROP TABLE IF EXISTS system_operation_logs CASCADE;
DROP TABLE IF EXISTS strategy_execution_logs CASCADE;
DROP TABLE IF EXISTS order_history CASCADE;
DROP TABLE IF EXISTS account_connection_logs CASCADE;

