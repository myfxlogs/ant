-- 046_system_config_econ_translation.down.sql

DELETE FROM system_config
WHERE key IN ('econ.translation.zhipu_api_key', 'econ.translation.zhipu_model');
