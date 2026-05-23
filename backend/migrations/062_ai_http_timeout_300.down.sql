ALTER TABLE system_ai_configs
    ALTER COLUMN timeout_seconds SET DEFAULT 60;

UPDATE system_ai_configs
   SET timeout_seconds = 60,
       updated_at = NOW()
 WHERE timeout_seconds = 300;
