ALTER TABLE system_ai_configs
    ALTER COLUMN timeout_seconds SET DEFAULT 300;

UPDATE system_ai_configs
   SET timeout_seconds = 300,
       updated_at = NOW()
 WHERE timeout_seconds IS NULL
    OR timeout_seconds = 60;
