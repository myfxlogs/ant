-- Migration 146: Add account_number to users table
-- Each user gets a unique 5-digit account number for login + wallet identification.
-- Rules: 5 digits, no leading 0, no 4 or 7.
-- NULL = not yet assigned (legacy users, or admin hasn't assigned one).

ALTER TABLE users ADD COLUMN IF NOT EXISTS account_number VARCHAR(5);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_account_number ON users (account_number);
