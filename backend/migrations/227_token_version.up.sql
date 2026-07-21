-- Add token_version to users for JWT revocation.
-- Incremented on password change and explicit logout.
-- Tokens issued before the increment become invalid on next validation.
ALTER TABLE users ADD COLUMN token_version INT NOT NULL DEFAULT 0;
