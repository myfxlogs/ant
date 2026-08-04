ALTER TABLE mt_accounts DROP CONSTRAINT IF EXISTS chk_account_status;
ALTER TABLE mt_accounts ADD CONSTRAINT chk_account_status
  CHECK (account_status IN ('connecting', 'connected', 'disconnected', 'reconnecting', 'frozen'));
