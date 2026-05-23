-- 010_auto_trading.down.sql
-- 回滚自动化交易功能相关表

DROP TABLE IF EXISTS trading_logs;
DROP TABLE IF EXISTS global_settings;
DROP TABLE IF EXISTS risk_configs;
DROP TABLE IF EXISTS strategy_executions;
DROP TABLE IF EXISTS strategy_schedules;
