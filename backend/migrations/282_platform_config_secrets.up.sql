-- 282_platform_config_secrets.up.sql
-- ENV-TO-PG-1: D-档秘密件密文 KV 轨（ANT_MASTER_KEY AES-256-GCM）。
-- C-档业务旋钮不复表——落既有 system_config（key/value/value_type 语义已备）。
-- value_enc 为 secrets.Client(PurposePlatformSecret) 自包含密文（版本号+nonce+密文+tag），
-- 明文永不落库、永不进日志；管理面只透出 has_value/note/updated_at。
CREATE TABLE IF NOT EXISTS platform_secrets (
    key VARCHAR(100) PRIMARY KEY,
    value_enc BYTEA NOT NULL,
    note TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
COMMENT ON TABLE platform_secrets IS 'platform-level secrets (JWT/SMTP/API keys), AES-256-GCM sealed with ANT_MASTER_KEY; plaintext never stored';
