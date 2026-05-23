-- 063: 移除 Google Gemini 厂商（system_ai 与默认主模型引用）。
BEGIN;

DELETE FROM system_ai_configs WHERE provider_id = 'gemini';

UPDATE users
   SET ai_primary_provider_id = '',
       ai_primary_model = ''
 WHERE ai_primary_provider_id = 'gemini';

COMMIT;
